package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/player"
	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

const (
	fixtureAddress      = "127.0.0.1:34119"
	fixtureEdgeUsername = "players"
	fixtureEdgePassword = "password-long-enough"
	fixtureRandomSeed   = uint64(0x435254)
)

type ids struct{ next atomic.Uint64 }

type fixtureRandom struct {
	mu     sync.Mutex
	state  uint64
	forced []int
}

func (random *fixtureRandom) Intn(limit int) int {
	if limit <= 1 {
		return 0
	}
	random.mu.Lock()
	defer random.mu.Unlock()
	if len(random.forced) != 0 {
		value := random.forced[0]
		random.forced = random.forced[1:]
		if value < 0 {
			return limit - 1
		}
		return value % limit
	}
	random.state = random.state*6364136223846793005 + 1442695040888963407
	return int(random.state % uint64(limit))
}

func (random *fixtureRandom) forceDudRemoval(position string) bool {
	random.mu.Lock()
	defer random.mu.Unlock()
	switch position {
	case "revealed":
		random.forced = []int{0, 0}
	case "pending":
		random.forced = []int{0, -1}
	default:
		return false
	}
	return true
}

func (random *fixtureRandom) reset() {
	random.mu.Lock()
	defer random.mu.Unlock()
	random.state = fixtureRandomSeed
	random.forced = nil
}

type fixtureEdge struct {
	active           atomic.Bool
	publicGeneration atomic.Uint64
	service          *control.Service
	connect          *player.ConnectService
	ingress          tunnel.PublicIngress
	publicURL        string
}

type fixtureEdgeStatus struct {
	AuthBoundary           string `json:"authBoundary"`
	Upstream               string `json:"upstream"`
	Active                 bool   `json:"active"`
	AuthorizationForwarded bool   `json:"authorizationForwarded"`
	PublicURL              string `json:"publicUrl"`
}

func (source *ids) Next() string {
	return fmt.Sprintf("browser-fixture-%d", source.next.Add(1))
}

func (edge *fixtureEdge) reset() error {
	if edge.ingress == nil || edge.ingress.URL() == nil {
		return errors.New("fixture ingress is unavailable")
	}
	if err := edge.ingress.Activate(edge.ingress.URL().Host, fixtureEdgeUsername, []byte(fixtureEdgePassword)); err != nil {
		return err
	}
	edge.active.Store(true)
	edge.publicGeneration.Store(0)
	return nil
}

type publicAccessFixtureSnapshot struct {
	Preferences            publicAccessFixturePreferences `json:"preferences"`
	ProviderTokenPresence  string                         `json:"providerTokenPresence"`
	PlayerPasswordPresence string                         `json:"playerPasswordPresence"`
	Status                 publicAccessFixtureStatus      `json:"status"`
}

type publicAccessFixturePreferences struct {
	Version  int    `json:"version"`
	Username string `json:"username"`
	Revision uint64 `json:"revision"`
}

type publicAccessFixtureStatus struct {
	State            string `json:"state"`
	Generation       uint64 `json:"generation"`
	SettingsRevision uint64 `json:"settingsRevision"`
	PublicURL        string `json:"publicUrl,omitempty"`
	ErrorCategory    string `json:"errorCategory,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

func (edge *fixtureEdge) publicFailure(kind string) (publicAccessFixtureSnapshot, bool) {
	failures := map[string][2]string{
		"invalid-token":        {"provider_authentication", "The provider rejected the account credential."},
		"revoked-token":        {"provider_authentication", "The provider rejected the account credential."},
		"no-network":           {"network_unavailable", "The network is unavailable; local access remains available."},
		"dns-timeout":          {"timeout", "Public access did not become ready in time."},
		"domain-conflict":      {"domain_unavailable", "The reserved domain is unavailable for this account."},
		"keychain-locked":      {"secret_store_locked", "Unlock Keychain and try again."},
		"keychain-denied":      {"secret_store_denied", "Allow Keychain access and try again."},
		"keychain-unavailable": {"secret_store_unavailable", "Keychain is unavailable; local access remains available."},
		"policy-failure":       {"provider_failure", "The public-access provider stopped unexpectedly."},
		"provider-failure":     {"provider_failure", "The public-access provider stopped unexpectedly."},
		"unexpected-done":      {"provider_failure", "The public-access provider stopped unexpectedly."},
		"close-failure":        {"provider_failure", "The public-access provider stopped unexpectedly."},
	}
	if kind == "stale-completion" {
		generation := edge.publicGeneration.Load()
		if generation > 0 {
			generation--
		}
		return edge.publicSnapshot(publicAccessFixtureStatus{
			State: "ready", Generation: generation, PublicURL: "https://stale.invalid",
		}), true
	}
	failure, ok := failures[kind]
	if !ok {
		return publicAccessFixtureSnapshot{}, false
	}
	return edge.publicSnapshot(publicAccessFixtureStatus{
		State: "error", Generation: edge.publicGeneration.Add(1),
		ErrorCategory: failure[0], ErrorMessage: failure[1],
	}), true
}

func (edge *fixtureEdge) publicRecovery() publicAccessFixtureSnapshot {
	return edge.publicSnapshot(publicAccessFixtureStatus{
		State: "ready", Generation: edge.publicGeneration.Add(1), PublicURL: "https://recovered.example",
	})
}

func (edge *fixtureEdge) publicSnapshot(status publicAccessFixtureStatus) publicAccessFixtureSnapshot {
	status.SettingsRevision = 0
	return publicAccessFixtureSnapshot{
		Preferences:           publicAccessFixturePreferences{Version: 1, Username: "players", Revision: 0},
		ProviderTokenPresence: "present", PlayerPasswordPresence: "present", Status: status,
	}
}

func (edge *fixtureEdge) update(response http.ResponseWriter) {
	updated := fixtureTerminal()
	updated.Tree.Children[0].Children = append(updated.Tree.Children[0].Children, domain.ContentNode{
		ID: "public-update", Type: domain.NodeEntry, Name: "PUBLIC UPDATE", Description: "STREAM UPDATE RECEIVED",
	})
	if _, err := edge.service.UpdateLiveTerminal(updated.Tree, &updated.IntroText); err != nil {
		http.Error(response, err.Error(), http.StatusConflict)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func (edge *fixtureEdge) activateHacking(response http.ResponseWriter) {
	target := fixtureTerminal()
	target.TerminalID = "terminal-hacking"
	target.TerminalName = "Security"
	target.HackLevel = 1
	if _, err := edge.service.RequestTerminalActivation(target); err != nil {
		http.Error(response, err.Error(), http.StatusConflict)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func activateCRTTerminal(service *control.Service, response http.ResponseWriter, target domain.TerminalTarget) bool {
	state, err := service.RequestTerminalActivation(target)
	if err != nil {
		http.Error(response, err.Error(), http.StatusConflict)
		return false
	}
	if state.PendingSwitch == nil {
		return true
	}
	if _, err = service.ResolveTerminalSwitch(state.PendingSwitch.SwitchID, domain.TerminalSwitchDiscard); err != nil {
		http.Error(response, err.Error(), http.StatusConflict)
		return false
	}
	return true
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	playerAssets, err := fs.Sub(os.DirFS("../../client"), "dist")
	if err != nil {
		return fmt.Errorf("open built player assets: %w", err)
	}
	fixtureHackRandom := &fixtureRandom{state: fixtureRandomSeed}
	liveService := live.New(fixtureHackRandom, nil)
	var connectPlayer *player.ConnectService
	service := control.New(control.Config{
		IDs:         &ids{},
		Runtime:     liveService,
		Terminals:   liveService,
		TrustedHack: liveService,
		Enqueue: func(effect control.Effect) {
			if connectPlayer != nil {
				connectPlayer.PublishEffect(effect)
			}
		},
	})
	connectPlayer, err = player.NewConnectService(player.ConnectServiceConfig{Coordinator: service, Assets: playerAssets})
	if err != nil {
		return fmt.Errorf("construct fixture Connect service: %w", err)
	}
	rpcPath, rpcHandler := player.NewConnectHandler(connectPlayer)
	applicationHandler := player.NewApplicationHandler(playerAssets, rpcPath, rpcHandler)
	edge := &fixtureEdge{service: service, connect: connectPlayer}

	for _, name := range []string{"Mara", "Boone", "Arcade", "Cass", "Veronica", "Raul", "Lily"} {
		if _, err := service.AddCharacter(name); err != nil {
			return err
		}
	}
	if _, err := service.StartBroadcast(); err != nil {
		return err
	}
	if _, err := service.RequestTerminalActivation(fixtureTerminal()); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /__fixture/desktop-api", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(response, `<!doctype html><meta charset="utf-8">
<script type="importmap">{"imports":{"@wailsio/runtime":"/__fixture/desktop-bindings.js","/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js":"/__fixture/desktop-bindings.js"}}</script>
<script type="module" src="/__fixture/desktop-api.js"></script>`)
	})
	mux.HandleFunc("GET /__fixture/public-access-settings", func(response http.ResponseWriter, _ *http.Request) {
		raw, readErr := os.ReadFile(filepath.Clean("../../frontend/src/index.html"))
		if readErr != nil {
			http.Error(response, "fixture master page is unavailable", http.StatusInternalServerError)
			return
		}
		page := string(raw)
		page = strings.Replace(page, `<head>`, `<head>
<script type="importmap">{"imports":{"@wailsio/runtime":"/__fixture/desktop-bindings.js","/bindings/github.com/obalunenko/Fallout-Terminal/desktopservice.js":"/__fixture/desktop-bindings.js"}}</script>`, 1)
		page = strings.Replace(page, `class="start-screen" id="startScreen"`, `class="start-screen" id="startScreen" style="display:none"`, 1)
		page = strings.Replace(page, `id="mainLayout" style="display:none"`, `id="mainLayout" style="display:flex"`, 1)
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write([]byte(page))
	})
	mux.HandleFunc("GET /__fixture/master.css", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "../../frontend/src/master.css")
	})
	mux.HandleFunc("GET /__fixture/master.js", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "../../frontend/src/master.js")
	})
	mux.HandleFunc("GET /__fixture/desktop-api.js", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "../../frontend/src/desktop-api.js")
	})
	mux.HandleFunc("GET /__fixture/desktop-bindings.js", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "fixtures/desktop-bindings.js")
	})
	mux.HandleFunc("POST /__fixture/reset", func(response http.ResponseWriter, _ *http.Request) {
		fixtureHackRandom.reset()
		if err := edge.reset(); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := service.EndBroadcast(); err != nil {
			http.Error(response, err.Error(), http.StatusConflict)
			return
		}
		if _, err := service.StartBroadcast(); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := service.RequestTerminalActivation(fixtureTerminal()); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/update", func(response http.ResponseWriter, _ *http.Request) {
		updated := fixtureTerminal()
		updated.Tree.Children = append(updated.Tree.Children, domain.ContentNode{
			ID: "public-update", Type: domain.NodeEntry, Name: "PUBLIC UPDATE", Description: "STREAM UPDATE RECEIVED",
		})
		if _, err := service.UpdateLiveTerminal(updated.Tree, &updated.IntroText); err != nil {
			http.Error(response, err.Error(), http.StatusConflict)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/local/hacking", func(response http.ResponseWriter, _ *http.Request) {
		edge.activateHacking(response)
	})
	mux.HandleFunc("POST /__fixture/local/crt/{state}", func(response http.ResponseWriter, request *http.Request) {
		state := request.PathValue("state")
		switch state {
		case "content":
			if _, err := service.RequestTerminalActivation(crtFixtureTerminal()); err != nil {
				http.Error(response, err.Error(), http.StatusConflict)
				return
			}
		case "unchanged":
			target := crtFixtureTerminal()
			if _, err := service.UpdateLiveTerminal(target.Tree, &target.IntroText); err != nil {
				http.Error(response, err.Error(), http.StatusConflict)
				return
			}
		case "replacement":
			target := crtReplacementTerminal()
			if _, err := service.UpdateLiveTerminal(target.Tree, &target.IntroText); err != nil {
				http.Error(response, err.Error(), http.StatusConflict)
				return
			}
		case "waiting":
			if _, err := service.RequestTerminalClear(); err != nil {
				http.Error(response, err.Error(), http.StatusConflict)
				return
			}
		case "hacking", "blocked":
			if !activateCRTTerminal(service, response, crtHackingTerminal("a")) {
				return
			}
		case "hacking-unchanged":
			target := crtHackingTerminal("a")
			if _, err := service.UpdateLiveTerminal(target.Tree, &target.IntroText); err != nil {
				http.Error(response, err.Error(), http.StatusConflict)
				return
			}
		case "hacking-replacement":
			if !activateCRTTerminal(service, response, crtHackingTerminal("b")) {
				return
			}
		default:
			http.NotFound(response, request)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/local/crt/hacking-dud/{position}", func(response http.ResponseWriter, request *http.Request) {
		if !fixtureHackRandom.forceDudRemoval(request.PathValue("position")) {
			http.NotFound(response, request)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/local/disconnect", func(response http.ResponseWriter, _ *http.Request) {
		connectPlayer.CloseSubscriptions()
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/public-access/failure/", func(response http.ResponseWriter, request *http.Request) {
		kind := strings.TrimPrefix(request.URL.Path, "/__fixture/public-access/failure/")
		snapshot, ok := edge.publicFailure(kind)
		if !ok || strings.Contains(kind, "/") {
			http.NotFound(response, request)
			return
		}
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(snapshot)
	})
	mux.HandleFunc("POST /__fixture/public-access/recover", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(edge.publicRecovery())
	})
	mux.HandleFunc("GET /__fixture/edge/status", func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(fixtureEdgeStatus{
			AuthBoundary: "application-ingress", Upstream: "http://" + fixtureAddress,
			Active: edge.active.Load(), AuthorizationForwarded: request.Header.Get("Authorization") != "",
			PublicURL: edge.publicURL,
		})
	})
	mux.HandleFunc("POST /__fixture/edge/update", func(response http.ResponseWriter, _ *http.Request) {
		edge.update(response)
	})
	mux.HandleFunc("POST /__fixture/edge/hacking", func(response http.ResponseWriter, _ *http.Request) {
		edge.activateHacking(response)
	})
	mux.HandleFunc("POST /__fixture/edge/disconnect", func(response http.ResponseWriter, _ *http.Request) {
		edge.connect.CloseSubscriptions()
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__fixture/edge/disable", func(response http.ResponseWriter, _ *http.Request) {
		edge.ingress.Deny()
		edge.active.Store(false)
		edge.connect.CloseSubscriptions()
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/__fixture/protected/", func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != fixtureEdgeUsername || password != fixtureEdgePassword {
			response.Header().Set("WWW-Authenticate", `Basic realm="Fallout Terminal Players"`)
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		forwarded := request.Clone(request.Context())
		forwarded.URL.Path = "/" + strings.TrimPrefix(request.URL.Path, "/__fixture/protected/")
		applicationHandler.ServeHTTP(response, forwarded)
	})
	mux.Handle("/", applicationHandler)

	listener, err := net.Listen("tcp4", fixtureAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", fixtureAddress, err)
	}
	ingress, err := tunnel.NewPublicIngressFactory().Start(context.Background(), "http://"+fixtureAddress)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("start fixture public ingress: %w", err)
	}
	edge.ingress = ingress
	edge.publicURL = ingress.URL().String()
	if err := edge.reset(); err != nil {
		_ = ingress.Close(context.Background())
		_ = listener.Close()
		return fmt.Errorf("activate fixture public ingress: %w", err)
	}
	httpServer := &http.Server{Handler: mux}
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
	case err := <-serveErrors:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ingress.Deny()
	return errors.Join(ingress.Close(shutdownContext), httpServer.Shutdown(shutdownContext))
}

func fixtureTerminal() domain.TerminalTarget {
	return domain.TerminalTarget{
		TerminalID: "terminal-1", TerminalName: "Overseer", HackLevel: 0, IntroText: "WELCOME",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT",
			Children: []domain.ContentNode{
				{ID: "docs", Type: domain.NodeFolder, Name: "DOCS", Children: []domain.ContentNode{
					{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "SYSTEM NOMINAL"},
				}},
				{ID: "status", Type: domain.NodeEntry, Name: "STATUS", Description: "ALL SYSTEMS OPERATIONAL"},
			},
		},
	}
}

func crtFixtureTerminal() domain.TerminalTarget {
	children := make([]domain.ContentNode, 0, 25)
	for index := 1; index <= 21; index++ {
		children = append(children, domain.ContentNode{
			ID:          fmt.Sprintf("crt-row-%02d", index),
			Type:        domain.NodeEntry,
			Name:        fmt.Sprintf("ARCHIVE %02d", index),
			Description: fmt.Sprintf("CRT ARCHIVE LINE %02d", index),
		})
	}
	children = append(children,
		domain.ContentNode{ID: "crt-empty", Type: domain.NodeFolder, Name: "EMPTY", Children: []domain.ContentNode{}},
		domain.ContentNode{
			ID: "crt-record", Type: domain.NodeEntry, Name: "LONG RECORD",
			Description: strings.Repeat("ROBCO RECORD LINE\n", 48) + "RECORD COMPLETE",
		},
		domain.ContentNode{
			ID: "crt-command", Type: domain.NodeCommand, Name: "RUN DIAGNOSTIC",
			Text: strings.Repeat("DIAGNOSTIC OUTPUT\n", 36) + "DIAGNOSTIC COMPLETE",
		},
		domain.ContentNode{
			ID: "crt-literal", Type: domain.NodeEntry, Name: `<img data-crt-injected src=x onerror="window.__crtInjected=true">`,
			Description: `<script>window.__crtInjected=true</script> & literal terminal text`,
		},
	)

	return domain.TerminalTarget{
		TerminalID: "terminal-crt", TerminalName: "CRT Acceptance", HackLevel: 0,
		IntroText: "CRT PRESENTATION ACCEPTANCE",
		Tree:      domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: children},
	}
}

func crtReplacementTerminal() domain.TerminalTarget {
	target := crtFixtureTerminal()
	target.IntroText = "CRT REPLACEMENT ACCEPTANCE"
	target.Tree.Children = []domain.ContentNode{
		{ID: "crt-replacement-a", Type: domain.NodeEntry, Name: "REPLACEMENT ALPHA", Description: "ALPHA"},
		{ID: "crt-replacement-b", Type: domain.NodeEntry, Name: "REPLACEMENT BETA", Description: "BETA"},
		{ID: "crt-replacement-c", Type: domain.NodeEntry, Name: "REPLACEMENT GAMMA", Description: "GAMMA"},
	}
	return target
}

func crtHackingTerminal(identity string) domain.TerminalTarget {
	target := crtFixtureTerminal()
	target.TerminalID = "terminal-crt-hacking-" + identity
	target.TerminalName = "CRT Security " + strings.ToUpper(identity)
	target.HackLevel = 1
	return target
}

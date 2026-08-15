package main

import (
	"context"
	"crypto/subtle"
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/player"
)

const (
	fixtureAddress          = "127.0.0.1:34119"
	protectedFixtureAddress = "127.0.0.1:34120"
	fixtureEdgeUsername     = "players"
	fixtureEdgePassword     = "password-long-enough"
)

type ids struct{ next atomic.Uint64 }

type fixtureEdge struct {
	active           atomic.Bool
	publicGeneration atomic.Uint64
	service          *control.Service
	connect          *player.ConnectService
	application      http.Handler
}

type fixtureEdgeStatus struct {
	AuthBoundary           string `json:"authBoundary"`
	Upstream               string `json:"upstream"`
	Active                 bool   `json:"active"`
	AuthorizationForwarded bool   `json:"authorizationForwarded"`
}

func (source *ids) Next() string {
	return fmt.Sprintf("browser-fixture-%d", source.next.Add(1))
}

func (edge *fixtureEdge) reset() {
	edge.active.Store(true)
	edge.publicGeneration.Store(0)
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

func (edge *fixtureEdge) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	username, password, ok := request.BasicAuth()
	if !ok || subtle.ConstantTimeCompare([]byte(username), []byte(fixtureEdgeUsername)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(fixtureEdgePassword)) != 1 {
		response.Header().Set("WWW-Authenticate", `Basic realm="Fallout Terminal Players"`)
		http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/__fixture/edge/status":
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(fixtureEdgeStatus{
			AuthBoundary:           "fixture-edge",
			Upstream:               "http://" + fixtureAddress,
			Active:                 edge.active.Load(),
			AuthorizationForwarded: false,
		})
		return
	case request.Method == http.MethodPost && request.URL.Path == "/__fixture/edge/update":
		edge.update(response)
		return
	case request.Method == http.MethodPost && request.URL.Path == "/__fixture/edge/hacking":
		edge.activateHacking(response)
		return
	case request.Method == http.MethodPost && request.URL.Path == "/__fixture/edge/disconnect":
		edge.connect.CloseSubscriptions()
		response.WriteHeader(http.StatusNoContent)
		return
	case request.Method == http.MethodPost && request.URL.Path == "/__fixture/edge/disable":
		edge.active.Store(false)
		edge.connect.CloseSubscriptions()
		response.WriteHeader(http.StatusNoContent)
		return
	}

	if !edge.active.Load() {
		http.Error(response, http.StatusText(http.StatusGone), http.StatusGone)
		return
	}
	forwarded := request.Clone(request.Context())
	forwarded.Header.Del("Authorization")
	edge.application.ServeHTTP(response, forwarded)
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
	liveService := live.New(nil, nil)
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
	protectedApplicationHandler := player.NewApplicationHandler(playerAssets, rpcPath, rpcHandler)
	edge := &fixtureEdge{service: service, connect: connectPlayer, application: protectedApplicationHandler}
	edge.reset()

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
		edge.reset()
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
	protectedListener, err := net.Listen("tcp4", protectedFixtureAddress)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("listen on %s: %w", protectedFixtureAddress, err)
	}
	httpServer := &http.Server{Handler: mux}
	protectedHTTPServer := &http.Server{Handler: edge}
	serveErrors := make(chan error, 2)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()
	go func() {
		serveErrors <- protectedHTTPServer.Serve(protectedListener)
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
	return errors.Join(httpServer.Shutdown(shutdownContext), protectedHTTPServer.Shutdown(shutdownContext))
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

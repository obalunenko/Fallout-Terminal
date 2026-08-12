package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	controlservice "github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	liveservice "github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/platform"
	playerserver "github.com/obalunenko/Fallout-Terminal/internal/player"
	playerconfigservice "github.com/obalunenko/Fallout-Terminal/internal/playerconfig"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
	tunnelservice "github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

// Wails runs the configured frontend build before production compilation. The
// checked-in .keep keeps ordinary Go tooling compile-safe on a clean checkout.
//
//go:embed all:frontend/dist
var frontendSource embed.FS

// The player remains a distinct browser application but is owned and served
// by the same Go process.
//
//go:embed all:client
var playerSource embed.FS

func main() {
	frontendAssets, err := fs.Sub(frontendSource, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	playerAssets, err := fs.Sub(playerSource, "client")
	if err != nil {
		log.Fatal(err)
	}

	app, err := composeApplication(playerAssets)
	if err != nil {
		log.Fatal(err)
	}
	if err := wails.Run(&options.App{
		Title:            "Fallout Terminal — Master Control",
		Width:            1200,
		Height:           780,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: options.NewRGB(11, 13, 10),
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		OnStartup:  app.startup,
		OnDomReady: app.domReady,
		OnShutdown: app.shutdown,
		Bind:       []interface{}{app},
	}); err != nil {
		log.Fatal(err)
	}
}

func composeApplication(playerAssets fs.FS) (*App, error) {
	locations, err := platform.DefaultSessionLocations(applicationResourceRoot())
	if err != nil {
		return nil, err
	}
	if err := validateProductionResources(playerAssets, locations.BundledDemo); err != nil {
		return nil, err
	}
	desktop := platform.NewDesktop(nil)
	events := &wailsEventSink{}
	live := liveservice.New(nil, nil)
	playerConfigs := playerconfigservice.NewService(
		playerconfigservice.NewStorage(nil), desktop, locations.DocumentsDefault,
	)
	effectRouter := &coordinationEffectRouter{}
	coordination := controlservice.New(controlservice.Config{
		Enqueue:     effectRouter.Enqueue,
		Runtime:     live,
		Terminals:   live,
		TrustedHack: live,
		RosterStore: playerConfigs,
	})
	playerCoordination := newPlayerCoordinatorAdapter(coordination)
	playerConfig := playerserver.Config{Assets: playerAssets, Live: live}
	playerConfig.Coordinator = playerCoordination
	playerConfig.OnClientCount = func(count int) {
		if app := effectRouter.App(); app != nil {
			app.updateClientCount(count)
		}
	}
	playerConfig.OnHackState = func(state *domain.PublicHackState) {
		if app := effectRouter.App(); app != nil {
			app.updateHackState(state)
		}
	}
	player, err := playerserver.NewServer(playerConfig)
	if err != nil {
		return nil, fmt.Errorf("construct player server: %w", err)
	}
	sessions := sessionservice.NewService(
		sessionservice.NewStorage(nil),
		desktop,
		sessionservice.Locations{
			DocumentsDefault:   locations.DocumentsDefault,
			BundledDemo:        locations.BundledDemo,
			ApplicationSupport: locations.ApplicationSupport,
		},
	)
	tunnel, tunnelEnabled := configureTunnel(os.Args[1:])
	app := NewAppWithDependencies(AppDependencies{
		Sessions:      sessions,
		PlayerConfigs: playerConfigs,
		Live:          live,
		Coordination:  coordination,
		Player:        player,
		Tunnel:        tunnel,
		Desktop:       desktop,
		Browser:       desktop,
		Events:        events,
		TunnelEnabled: tunnelEnabled,
	})
	effectRouter.Bind(player, app)
	return app, nil
}

// coordinationEffectRouter closes the construction cycle without letting the
// coordinator know about WebSockets or Wails. Enqueue snapshots its targets
// under a short lock and releases that lock before dispatch, so coordinator
// publication cannot re-enter the router or invert lock ownership.
type coordinationEffectRouter struct {
	mu     sync.RWMutex
	player *playerserver.Server
	app    *App
}

func (router *coordinationEffectRouter) Bind(player *playerserver.Server, app *App) {
	router.mu.Lock()
	router.player = player
	router.app = app
	router.mu.Unlock()
}

func (router *coordinationEffectRouter) App() *App {
	router.mu.RLock()
	defer router.mu.RUnlock()
	return router.app
}

func (router *coordinationEffectRouter) Enqueue(effect controlservice.Effect) {
	router.mu.RLock()
	player := router.player
	app := router.app
	router.mu.RUnlock()

	if player != nil {
		player.PublishCoordinationEffect(effect)
	}
	if app != nil && effect.Master != nil {
		app.publishCoordinationState(effect.Master)
	}
	if app != nil {
		switch {
		case effect.Hack != nil:
			app.updateHackState(effect.Hack)
		case effect.Live != nil:
			app.updateHackState(effect.Live.Hack)
		}
	}
}

// playerCoordinatorAdapter presents the player server seam over the same
// canonical control.Service injected into App. It retains only the
// connection-to-session lookup needed to translate character selections; the
// service owns token recognition and aggregate connection presence.
type playerCoordinatorAdapter struct {
	mu                  sync.RWMutex
	service             *controlservice.Service
	sessionByConnection map[domain.ConnectionID]domain.LogicalSessionID
}

func newPlayerCoordinatorAdapter(service *controlservice.Service) *playerCoordinatorAdapter {
	return &playerCoordinatorAdapter{
		service:             service,
		sessionByConnection: make(map[domain.ConnectionID]domain.LogicalSessionID),
	}
}

func (adapter *playerCoordinatorAdapter) AttachConnection(connectionID domain.ConnectionID, browserToken domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	if adapter == nil || adapter.service == nil {
		return "", nil
	}
	returnedToken, state := adapter.service.AttachConnection(connectionID, browserToken)
	if returnedToken == "" || state == nil || state.SessionID == "" {
		return "", nil
	}
	adapter.mu.Lock()
	adapter.sessionByConnection[connectionID] = state.SessionID
	adapter.mu.Unlock()
	return returnedToken, state
}

func (adapter *playerCoordinatorAdapter) DetachConnection(connectionID domain.ConnectionID) {
	if adapter == nil || adapter.service == nil {
		return
	}
	adapter.mu.Lock()
	delete(adapter.sessionByConnection, connectionID)
	adapter.mu.Unlock()
	adapter.service.DetachConnection(connectionID)
}

func (adapter *playerCoordinatorAdapter) SelectCharacter(connectionID domain.ConnectionID, requestID string, broadcastID domain.BroadcastID, characterID domain.CharacterID) {
	if adapter == nil || adapter.service == nil {
		return
	}
	adapter.mu.RLock()
	sessionID := adapter.sessionByConnection[connectionID]
	adapter.mu.RUnlock()
	adapter.service.SelectCharacter(controlservice.CharacterSelection{
		ConnectionID: connectionID,
		SessionID:    sessionID,
		RequestID:    requestID,
		BroadcastID:  broadcastID,
		CharacterID:  characterID,
	})
}

func (adapter *playerCoordinatorAdapter) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	if adapter == nil || adapter.service == nil {
		return
	}
	adapter.service.DispatchPlayerAction(connectionID, command)
}

func (adapter *playerCoordinatorAdapter) CurrentLiveForSession(sessionID domain.LogicalSessionID) (*domain.PublicLiveState, uint64, bool) {
	if adapter == nil || adapter.service == nil {
		return nil, 0, false
	}
	return adapter.service.CurrentLiveForSession(sessionID)
}

func validateProductionResources(playerAssets fs.FS, bundledDemo string) error {
	if playerAssets == nil {
		return errors.New("player assets are unavailable")
	}
	if info, err := fs.Stat(playerAssets, "index.html"); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("player assets are incomplete: index.html is unavailable")
	}
	if info, err := os.Stat(bundledDemo); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("bundled demo is unavailable at %s", bundledDemo)
	}
	return nil
}

func configureTunnel(args []string) (TunnelService, bool) {
	config, err := tunnelservice.ParseConfig(args, os.LookupEnv)
	enabled := config.Enabled || publicModeRequested(args)
	if !enabled {
		return nil, false
	}
	if err != nil {
		return configurationErrorTunnel{err: err}, true
	}
	return tunnelservice.NewService(config, tunnelservice.NewProcessRunner(), tunnelservice.ServiceOptions{}), true
}

func publicModeRequested(args []string) bool {
	if enabled, exists := os.LookupEnv("NGROK_ENABLED"); exists && enabled == "1" {
		return true
	}
	for _, argument := range args {
		if argument == "--ngrok" {
			return true
		}
	}
	return false
}

type configurationErrorTunnel struct {
	err error
}

func (tunnel configurationErrorTunnel) Start(context.Context) (domain.ServerInfo, error) {
	return domain.ServerInfo{}, tunnel.err
}

func (configurationErrorTunnel) Stop(context.Context) error {
	return nil
}

func applicationResourceRoot() string {
	executable, err := os.Executable()
	if err == nil {
		macOSDirectory := filepath.Dir(executable)
		if filepath.Base(macOSDirectory) == "MacOS" && filepath.Base(filepath.Dir(macOSDirectory)) == "Contents" {
			return filepath.Join(filepath.Dir(macOSDirectory), "Resources")
		}
	}
	workingDirectory, err := os.Getwd()
	if err == nil {
		return workingDirectory
	}
	return filepath.Dir(executable)
}

type wailsEventSink struct {
	mu  sync.RWMutex
	ctx context.Context
}

func (sink *wailsEventSink) SetContext(ctx context.Context) {
	sink.mu.Lock()
	sink.ctx = ctx
	sink.mu.Unlock()
}

func (sink *wailsEventSink) Emit(name string, payload any) error {
	sink.mu.RLock()
	ctx := sink.ctx
	sink.mu.RUnlock()
	if ctx == nil {
		return errors.New("Wails event runtime is not ready")
	}
	wailsruntime.EventsEmit(ctx, name, payload)
	return nil
}

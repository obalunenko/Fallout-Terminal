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
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	controlservice "github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	configv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/config/v1"
	liveservice "github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/platform"
	playerserver "github.com/obalunenko/Fallout-Terminal/internal/player"
	playerconfigservice "github.com/obalunenko/Fallout-Terminal/internal/playerconfig"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
	tunnelservice "github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

// The repository-owned Go build command prepares the frontend before production
// compilation. The checked-in .keep keeps ordinary Go tooling compile-safe on a
// clean checkout.
//
//go:embed all:frontend/overseer/dist
var overseerSource embed.FS

// The player remains a distinct browser application but is owned and served
// by the same Go process.
//
//go:embed all:frontend/client/dist
var clientSource embed.FS

func main() {
	overseerAssets, err := fs.Sub(overseerSource, "frontend/overseer/dist")
	if err != nil {
		log.Fatal(err)
	}
	clientAssets, err := fs.Sub(clientSource, "frontend/client/dist")
	if err != nil {
		log.Fatal(err)
	}

	host := newWailsApplication(overseerAssets)
	core, err := composeApplication(host, clientAssets)
	if err != nil {
		log.Fatal(err)
	}
	registerWailsServices(host, core)
	newOverseerWindow(host)
	if err := host.Run(); err != nil {
		log.Fatal(err)
	}
}

func composeApplication(host *application.App, clientAssets fs.FS) (*App, error) {
	locations, err := platform.DefaultSessionLocations(applicationResourceRoot())
	if err != nil {
		return nil, err
	}
	if err := validateProductionResources(clientAssets, locations.BundledDemo); err != nil {
		return nil, err
	}
	publicAccessSettingsPath, err := platform.PublicAccessSettingsPath(locations.ApplicationSupport)
	if err != nil {
		return nil, fmt.Errorf("resolve public-access settings path: %w", err)
	}
	runtimeConfig := defaultApplicationConfig(locations)
	desktop := platform.NewDesktop(nil, host.Dialog, host.Browser)
	events := newWailsEventSink(host.Event)
	live := liveservice.New(nil, nil)
	playerConfigs := playerconfigservice.NewService(
		playerconfigservice.NewStorage(nil), desktop, locations.DocumentsDefault,
	)
	sessions := sessionservice.NewService(
		sessionservice.NewStorage(nil),
		desktop,
		sessionservice.Locations{
			DocumentsDefault:   locations.DocumentsDefault,
			BundledDemo:        locations.BundledDemo,
			ApplicationSupport: locations.ApplicationSupport,
		},
	)
	effectRouter := &coordinationEffectRouter{}
	coordination := controlservice.New(controlservice.Config{
		Enqueue:            effectRouter.Enqueue,
		Runtime:            live,
		Terminals:          live,
		TrustedHack:        live,
		RosterStore:        playerConfigs,
		CommandStateStore:  &sessionCommandStateStore{service: sessions},
		TerminalCatalog:    sessions,
		RequestResultLimit: int(runtimeConfig.Coordination.RequestResultLimit),
	})
	playerConfig := playerserver.Config{
		Address: runtimeConfig.PlayerServer.Address,
		Assets:  clientAssets,
	}
	connectPlayer, err := playerserver.NewConnectService(playerserver.ConnectServiceConfig{
		Coordinator: coordination,
		Assets:      clientAssets,
		QueueSize:   int(runtimeConfig.PlayerServer.DeliveryQueueSize),
		OnClientCount: func(count int) {
			if app := effectRouter.App(); app != nil {
				app.updateClientCount(count)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("construct Connect player service: %w", err)
	}
	playerConfig.Connect = connectPlayer
	player, err := playerserver.NewServer(playerConfig)
	if err != nil {
		return nil, fmt.Errorf("construct player server: %w", err)
	}
	packaged := isPackagedApplication()
	publicSettings := tunnelservice.NewPublicAccessSettingsStore(publicAccessSettingsPath, nil, nil)
	publicSecrets := platform.NewPlatformKeychainSecretStore(packaged)
	effectivePublicSettings, effectivePublicSecrets := publicAccessStoresForProfile(publicSettings, publicSecrets, packaged, os.LookupEnv)
	var app *App
	publicAccess, err := tunnelservice.NewPublicAccessManager(tunnelservice.ManagerConfig{
		Settings:    effectivePublicSettings,
		Secrets:     effectivePublicSecrets,
		Tunnel:      tunnelservice.NewEmbeddedNgrokService(),
		Ingress:     tunnelservice.NewPublicIngressFactory(),
		UpstreamURL: publicAccessCompositionRoute().UpstreamURL,
		Publish: func(snapshot tunnelservice.PublicAccessSnapshot) {
			if app != nil {
				app.acceptPublicAccessSnapshot(snapshot, true)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("construct embedded public access: %w", err)
	}
	app = NewAppWithDependencies(AppDependencies{
		Sessions:        sessions,
		PlayerConfigs:   playerConfigs,
		Live:            live,
		Coordination:    coordination,
		Player:          player,
		Desktop:         desktop,
		Browser:         desktop,
		Events:          events,
		PublicSettings:  effectivePublicSettings,
		PublicSecrets:   effectivePublicSecrets,
		PublicAccess:    publicAccess,
		StartupTimeout:  time.Duration(runtimeConfig.Startup.TimeoutMilliseconds) * time.Millisecond,
		ShutdownTimeout: time.Duration(runtimeConfig.Shutdown.TimeoutMilliseconds) * time.Millisecond,
	})
	effectRouter.Bind(player, app)
	return app, nil
}

// sessionCommandStateStore adapts the session owner's context-aware mutation
// API to the coordinator's callback-free synchronous durability seam.
type sessionCommandStateStore struct {
	service *sessionservice.Service
}

var _ controlservice.CommandStateStore = (*sessionCommandStateStore)(nil)

func (store *sessionCommandStateStore) ExecuteCommandState(terminalID, commandID string) (controlservice.CommandStateMutation, error) {
	if store == nil || store.service == nil {
		return controlservice.CommandStateMutation{}, errors.New("session command-state store is unavailable")
	}
	return commandStateMutation(store.service.ExecuteCommandState(context.Background(), terminalID, commandID))
}

func (store *sessionCommandStateStore) ResetCommandState(terminalID, commandID string) (controlservice.CommandStateMutation, error) {
	if store == nil || store.service == nil {
		return controlservice.CommandStateMutation{}, errors.New("session command-state store is unavailable")
	}
	return commandStateMutation(store.service.ResetCommandState(context.Background(), terminalID, commandID))
}

func (store *sessionCommandStateStore) ResetTerminalCommandStates(terminalID string) (controlservice.CommandStateMutation, error) {
	if store == nil || store.service == nil {
		return controlservice.CommandStateMutation{}, errors.New("session command-state store is unavailable")
	}
	return commandStateMutation(store.service.ResetTerminalCommandStates(context.Background(), terminalID))
}

func commandStateMutation(result sessionservice.CommandStateResult) (controlservice.CommandStateMutation, error) {
	if !result.OK {
		message := result.Error
		if message == "" {
			message = "session command-state mutation failed"
		}
		return controlservice.CommandStateMutation{}, errors.New(message)
	}
	if result.Session == nil {
		return controlservice.CommandStateMutation{}, errors.New("session command-state mutation returned no document")
	}
	return controlservice.CommandStateMutation{
		Changed: result.Changed, Revision: result.Revision, Session: *result.Session,
	}, nil
}

func publicAccessStoresForProfile(
	settings tunnelservice.PublicAccessSettings,
	secrets tunnelservice.SecretStore,
	packaged bool,
	lookup tunnelservice.EnvironmentLookup,
) (tunnelservice.PublicAccessSettings, tunnelservice.SecretStore) {
	if packaged || lookup == nil {
		return settings, secrets
	}
	override := tunnelservice.NewDevelopmentTestPublicAccessOverride(settings, secrets, lookup)
	return override, override
}

type publicAccessRoute struct {
	PlayerTarget string
	UpstreamURL  string
}

func publicAccessCompositionRoute() publicAccessRoute {
	return publicAccessRoute{
		PlayerTarget: tunnelservice.PlayerUpstreamAddress,
		UpstreamURL:  "http://" + tunnelservice.PlayerUpstreamAddress,
	}
}

func isPackagedApplication() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	macOSDirectory := filepath.Dir(executable)
	return filepath.Base(macOSDirectory) == "MacOS" && filepath.Base(filepath.Dir(macOSDirectory)) == "Contents"
}

func defaultApplicationConfig(locations platform.SessionLocations) *configv1.ApplicationConfig {
	return &configv1.ApplicationConfig{
		PlayerServer: &configv1.PlayerServerConfig{
			Address: "0.0.0.0:3690", DeliveryQueueSize: 32, StartupTimeoutMilliseconds: 20_000,
			RequestLimits: &configv1.PublicRequestLimits{
				UncompressedMessageBytes: playerserver.MaxUncompressedMessageBytes,
				EncodedBodyBytes:         playerserver.MaxEncodedBodyBytes,
				DecompressedMessageBytes: playerserver.MaxDecompressedMessageBytes,
				RecognitionHandleBytes:   domain.MaxRecognitionHandleBytes,
				RequestIdBytes:           domain.MaxRequestIDBytes,
				BroadcastIdBytes:         domain.MaxBroadcastIDBytes,
				GenerationIdBytes:        domain.MaxGenerationIDBytes,
				TerminalIdBytes:          domain.MaxTerminalIDBytes,
				CharacterIdBytes:         domain.MaxCharacterIDBytes,
				ActionTargetBytes:        domain.MaxActionTargetBytes,
				SoundCategoryBytes:       domain.MaxSoundCategoryBytes,
			},
		},
		Coordination: &configv1.CoordinationConfig{RequestResultLimit: 256},
		Browser: &configv1.BrowserClientConfig{
			RecognitionStorageKey:      "fallout-terminal.player-token",
			ReconnectDelayMilliseconds: 3_000,
			ElectionLeaseMilliseconds:  5_000,
		},
		Paths: &configv1.PathConfig{
			DocumentsDirectory:          locations.DocumentsDefault,
			BundledDemoPath:             locations.BundledDemo,
			ApplicationSupportDirectory: locations.ApplicationSupport,
		},
		Startup:  &configv1.StartupConfig{TimeoutMilliseconds: 30_000},
		Shutdown: &configv1.ShutdownConfig{GracePeriodMilliseconds: 2_000, TimeoutMilliseconds: 5_000},
	}
}

// coordinationEffectRouter closes the construction cycle without letting the
// coordinator know about Connect streams or Wails. Enqueue snapshots its targets
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

func applicationResourceRoot() string {
	executable, err := os.Executable()
	workingDirectory, err := os.Getwd()
	if err != nil {
		workingDirectory = ""
	}
	return applicationResourceRootFor(executable, workingDirectory)
}

func applicationResourceRootFor(executable, workingDirectory string) string {
	macOSDirectory := filepath.Dir(executable)
	if filepath.Base(macOSDirectory) == "MacOS" && filepath.Base(filepath.Dir(macOSDirectory)) == "Contents" {
		return filepath.Join(filepath.Dir(macOSDirectory), "Resources")
	}
	if workingDirectory != "" {
		return workingDirectory
	}
	return filepath.Dir(executable)
}

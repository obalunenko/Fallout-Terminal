package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerconfigservice "github.com/obalunenko/Fallout-Terminal/internal/playerconfig"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
	tunnelservice "github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

const (
	serverInfoEvent         = "server-info"
	clientCountEvent        = "client-count"
	hackStateEvent          = "hack-state"
	coordinationStateEvent  = "coordination-state"
	publicAccessStatusEvent = "public-access-status"
)

// SessionService is the lifecycle boundary for the ordered persistence worker.
// Session commands extend this interface when they are introduced.
type SessionService interface {
	Shutdown(context.Context) error
}

type sessionCommands interface {
	Create(context.Context) sessionservice.SessionResult
	Open(context.Context) sessionservice.SessionResult
	CopyDemo(context.Context) sessionservice.SessionResult
	Save(context.Context, domain.Session, uint64) sessionservice.SaveResult
	Snapshot() sessionservice.ActiveSession
}

type sessionPlayerConfigCommands interface {
	Snapshot() sessionservice.ActiveSession
	AssociatePlayerConfig(context.Context, string) sessionservice.SessionResult
}

// PlayerConfigService owns trusted native selection and strict durable files.
type PlayerConfigService interface {
	Create(context.Context) playerconfigservice.Result
	Open(context.Context) playerconfigservice.Result
	LoadReferenced(string, string) playerconfigservice.Result
}

// LiveService owns canonical shared player state.
type LiveService interface {
	Update(domain.ContentNode, *string) (*domain.PublicLiveState, bool)
	Clear()
	Snapshot() *domain.PublicLiveState
	ForceHackSuccess() (*domain.PublicHackState, bool)
}

// CoordinationService owns process-local logical sessions, roster, claims,
// broadcast lifetime, and controller state behind one transaction boundary.
type CoordinationService interface {
	Snapshot() *domain.MasterCoordinationState
	AddCharacter(string) (*domain.MasterCoordinationState, error)
	StartBroadcast() (*domain.MasterCoordinationState, error)
}

// coordinationCorrectionService is the additive trusted roster/session seam.
// Keeping it additive lets focused lifecycle fakes provide only the commands
// they exercise while production control.Service implements the complete set.
type coordinationCorrectionService interface {
	RenameCharacter(domain.CharacterID, string) (*domain.MasterCoordinationState, error)
	DeleteCharacter(domain.CharacterID) (*domain.MasterCoordinationState, error)
	RenameLogicalSession(domain.LogicalSessionID, string) (*domain.MasterCoordinationState, error)
	AssignCharacter(domain.LogicalSessionID, domain.CharacterID) (*domain.MasterCoordinationState, error)
	ReleaseCharacter(domain.LogicalSessionID) (*domain.MasterCoordinationState, error)
	MoveCharacter(domain.CharacterID, domain.LogicalSessionID) (*domain.MasterCoordinationState, error)
}

type coordinationControllerService interface {
	SetActiveController(domain.LogicalSessionID) (*domain.MasterCoordinationState, error)
}

// coordinationTerminalService is the trusted terminal-selection boundary. It
// keeps terminal choice, runtime checkpoints, and publication ordered by the
// same coordinator that owns assignments and controller authority.
type coordinationTerminalService interface {
	RequestTerminalActivation(domain.TerminalTarget) (*domain.MasterCoordinationState, error)
	RequestTerminalClear() (*domain.MasterCoordinationState, error)
	UpdateLiveTerminal(domain.ContentNode, *string) (*domain.MasterCoordinationState, error)
	ResetFailedHack(domain.TerminalTarget) (*domain.MasterCoordinationState, error)
}

type coordinationTerminalDecisionService interface {
	ResolveTerminalSwitch(domain.SwitchID, domain.TerminalSwitchChoice) (*domain.MasterCoordinationState, error)
}

type coordinationBroadcastLifecycleService interface {
	EndBroadcast() (*domain.MasterCoordinationState, error)
}

type coordinationPlayerConfigService interface {
	InstallPlayerConfig(domain.PlayerConfigHandle, []domain.CharacterRosterEntry) (*domain.MasterCoordinationState, error)
	ClearPlayerConfig() (*domain.MasterCoordinationState, error)
}

type coordinationShutdownService interface {
	Shutdown()
}

type coordinatedHackSuccess interface {
	ForceHackSuccess() (*domain.PublicHackState, bool)
}

// Browser is the allowlisted external-browser boundary.
type Browser interface {
	OpenURL(string) error
}

// PlayerServer owns the in-process HTTP/Connect listener.
type PlayerServer interface {
	Start(context.Context) (domain.ServerInfo, error)
	Stop(context.Context) error
}

// playerPublisher is implemented by the complete player server. Keeping
// publication separate from the lifecycle interface preserves construction
// compatibility with partial-start and test servers while making every live
// bridge mutation observable by connected browsers.
type playerPublisher interface {
	PublishUpdate()
	PublishHack()
}

// DesktopRuntime represents the readiness and release boundary of the desktop
// host independently of Wails globals.
type DesktopRuntime interface {
	Ready(context.Context) error
	Close(context.Context) error
}

// EventSink publishes narrow, public values to the game-master frontend.
type EventSink interface {
	Emit(name string, payload any) error
}

// PublicAccessSettingsStore persists only the versioned, non-secret public-
// access preferences. Production credentials remain behind SecretStore.
type PublicAccessSettingsStore interface {
	Load() (tunnelservice.PublicAccessPreferences, error)
	Save(tunnelservice.PublicAccessPreferences) error
}

// PublicAccessCore owns the embedded endpoint lifecycle and its secret-free
// state. App exposes only the five trusted desktop methods around this core.
type PublicAccessCore interface {
	Initialize(context.Context) tunnelservice.PublicAccessSnapshot
	Snapshot() tunnelservice.PublicAccessSnapshot
	Start(context.Context, uint64) tunnelservice.PublicAccessResult
	Stop(context.Context, uint64) tunnelservice.PublicAccessResult
	Reconfigure(context.Context, tunnelservice.PublicAccessMutation) tunnelservice.PublicAccessResult
	Shutdown(context.Context) error
}

// AppDependencies contains constructed services. Construction acquires no
// external resources; Start owns acquisition in contract order.
type AppDependencies struct {
	Sessions        SessionService
	PlayerConfigs   PlayerConfigService
	Live            LiveService
	Coordination    CoordinationService
	Player          PlayerServer
	Desktop         DesktopRuntime
	Browser         Browser
	Events          EventSink
	PublicSettings  PublicAccessSettingsStore
	PublicSecrets   tunnelservice.SecretStore
	PublicAccess    PublicAccessCore
	PasswordEntropy io.Reader
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// RuntimeStatus is the synchronous startup/status snapshot used to avoid
// losing events emitted before the frontend subscribes.
type RuntimeStatus struct {
	ServerInfo        *domain.ServerInfo              `json:"serverInfo"`
	ClientCount       int                             `json:"clientCount"`
	HackState         *domain.PublicHackState         `json:"hackState"`
	StartupError      string                          `json:"startupError,omitempty"`
	SaveState         string                          `json:"saveState"`
	RequestedRevision uint64                          `json:"requestedRevision"`
	SavedRevision     uint64                          `json:"savedRevision"`
	CoordinationState *domain.MasterCoordinationState `json:"coordinationState"`
}

// CommandResult is used for privileged commands that do not return a model.
type CommandResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// PublicAccessPreferences is the secret-free native desktop projection.
type PublicAccessPreferences struct {
	Version                   uint32 `json:"version"`
	EnabledPreference         bool   `json:"enabledPreference"`
	ReservedDomain            string `json:"reservedDomain,omitempty"`
	Username                  string `json:"username"`
	ProviderTokenPresentHint  bool   `json:"providerTokenPresentHint"`
	PlayerPasswordPresentHint bool   `json:"playerPasswordPresentHint"`
	Revision                  uint64 `json:"revision"`
}

type PublicAccessStatus struct {
	State            string `json:"state"`
	Generation       uint64 `json:"generation"`
	SettingsRevision uint64 `json:"settingsRevision"`
	PublicURL        string `json:"publicUrl,omitempty"`
	ErrorCategory    string `json:"errorCategory,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

type PublicAccessSnapshot struct {
	Preferences            PublicAccessPreferences `json:"preferences"`
	ProviderTokenPresence  string                  `json:"providerTokenPresence"`
	PlayerPasswordPresence string                  `json:"playerPasswordPresence"`
	Status                 PublicAccessStatus      `json:"status"`
}

// SavePublicAccessSettingsPayload is ephemeral trusted input. Its secret
// fields are consumed into byte buffers and are never copied into a result,
// event, status, or persistence model.
type SavePublicAccessSettingsPayload struct {
	ExpectedRevision          uint64 `json:"expectedRevision"`
	EnabledPreference         bool   `json:"enabledPreference"`
	ReservedDomain            string `json:"reservedDomain,omitempty"`
	Username                  string `json:"username"`
	ReplacementProviderToken  string `json:"replacementProviderToken,omitempty"`
	DeleteProviderToken       bool   `json:"deleteProviderToken,omitempty"`
	ReplacementPlayerPassword string `json:"replacementPlayerPassword,omitempty"`
	DeletePlayerPassword      bool   `json:"deletePlayerPassword,omitempty"`
}

type PublicAccessCommandPayload struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
}

type PublicAccessCommandResult struct {
	OK       bool                 `json:"ok"`
	Error    string               `json:"error,omitempty"`
	Snapshot PublicAccessSnapshot `json:"snapshot"`
}

// GeneratedPlayerPasswordResult is deliberately not reusable and contains no
// snapshot. GeneratedPassword is populated only after secure-store replacement
// and settings persistence both succeed.
type GeneratedPlayerPasswordResult struct {
	OK                bool   `json:"ok"`
	Error             string `json:"error,omitempty"`
	GeneratedPassword string `json:"generatedPassword,omitempty"`
	SettingsRevision  uint64 `json:"settingsRevision"`
}

// CoordinationCommandResult returns the authoritative detached state for
// both accepted and rejected roster/broadcast commands.
type CoordinationCommandResult struct {
	OK    bool                            `json:"ok"`
	Error string                          `json:"error,omitempty"`
	State *domain.MasterCoordinationState `json:"state"`
}

// PlayerConfigCommandResult combines durable metadata with authoritative roster state.
type PlayerConfigCommandResult struct {
	OK       bool                            `json:"ok"`
	Canceled bool                            `json:"canceled"`
	Error    string                          `json:"error,omitempty"`
	Config   *domain.PlayerConfigMetadata    `json:"playerConfig,omitempty"`
	Session  *domain.Session                 `json:"session,omitempty"`
	State    *domain.MasterCoordinationState `json:"state"`
}

// TerminalSwitchCommandResult describes either a completed terminal selection
// or a later decision gate. SwitchID remains empty for direct activation and
// clear operations; unfinished-puzzle choices populate it in the US8 flow.
type TerminalSwitchCommandResult struct {
	OK       bool                            `json:"ok"`
	Error    string                          `json:"error,omitempty"`
	Status   string                          `json:"status,omitempty"`
	SwitchID domain.SwitchID                 `json:"switchId,omitempty"`
	State    *domain.MasterCoordinationState `json:"state"`
}

// CharacterRenamePayload identifies one stable roster entry and its new
// player-facing display name.
type CharacterRenamePayload struct {
	CharacterID domain.CharacterID `json:"characterId"`
	Name        string             `json:"name"`
}

// LogicalSessionRenamePayload changes only a process-local fallback label.
type LogicalSessionRenamePayload struct {
	SessionID    domain.LogicalSessionID `json:"sessionId"`
	FallbackName string                  `json:"fallbackName"`
}

// AssignmentPayload assigns one available character to one unassigned
// logical session.
type AssignmentPayload struct {
	SessionID   domain.LogicalSessionID `json:"sessionId"`
	CharacterID domain.CharacterID      `json:"characterId"`
}

// MoveCharacterPayload atomically transfers one stable character identity.
type MoveCharacterPayload struct {
	CharacterID domain.CharacterID      `json:"characterId"`
	ToSessionID domain.LogicalSessionID `json:"toSessionId"`
}

// LiveTerminalPayload is the validated set-live bridge input.
type LiveTerminalPayload struct {
	TerminalID   string             `json:"terminalId"`
	TerminalName string             `json:"terminalName"`
	Tree         domain.ContentNode `json:"tree"`
	HackLevel    int                `json:"hackLevel"`
	IntroText    string             `json:"introText"`
}

// LiveUpdatePayload replaces published content without resetting a puzzle.
type LiveUpdatePayload struct {
	Tree      domain.ContentNode `json:"tree"`
	IntroText *string            `json:"introText,omitempty"`
}

// TerminalSwitchDecisionPayload resolves one opaque unfinished-puzzle switch
// request with an exact allowlisted decision.
type TerminalSwitchDecisionPayload struct {
	SwitchID domain.SwitchID             `json:"switchId"`
	Decision domain.TerminalSwitchChoice `json:"decision"`
}

// App is the Wails composition root. Domain behavior remains in internal
// packages; App owns lifecycle and the narrow desktop facade.
type App struct {
	lifecycleMu           sync.Mutex
	coordinationCommandMu sync.Mutex
	publicAccessCommandMu sync.Mutex
	mu                    sync.RWMutex

	deps AppDependencies
	ctx  context.Context

	phase               string
	serverInfo          *domain.ServerInfo
	clientCount         int
	hackState           *domain.PublicHackState
	coordinationState   *domain.MasterCoordinationState
	startupError        string
	saveState           string
	requestedRevision   uint64
	savedRevision       uint64
	playerStarted       bool
	desktopReady        bool
	sessionsClosed      bool
	processStateCleared bool
	publicAccessClosed  bool
	stopped             bool

	publicAccessLoaded      bool
	publicAccessPreferences tunnelservice.PublicAccessPreferences
	providerTokenPresence   tunnelservice.SecretPresence
	playerPasswordPresence  tunnelservice.SecretPresence
	publicAccessStatus      tunnelservice.PublicAccessStatus
}

// lifecyclePhase returns the host-owned lifecycle observation used by Go
// tests and adapters. It is deliberately absent from RuntimeStatus and every
// protobuf/native DTO; the master derives presentation from serverInfo and
// startupError instead.
func (app *App) lifecyclePhase() string {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.phase
}

// NewApp constructs the application without acquiring external resources.
func NewApp() *App {
	return NewAppWithDependencies(AppDependencies{})
}

// NewAppWithDependencies constructs a testable composition root.
func NewAppWithDependencies(deps AppDependencies) *App {
	preferences := tunnelservice.DefaultPublicAccessPreferences()
	app := &App{
		deps: deps, phase: "constructed", saveState: string(sessionservice.SaveStateIdle),
		publicAccessPreferences: preferences,
		providerTokenPresence:   tunnelservice.SecretUnknown,
		playerPasswordPresence:  tunnelservice.SecretUnknown,
		publicAccessStatus: tunnelservice.PublicAccessStatus{
			State: tunnelservice.LifecycleDisabled, SettingsRevision: preferences.Revision,
		},
	}
	if deps.Coordination != nil {
		app.coordinationState = domain.CloneMasterCoordinationState(deps.Coordination.Snapshot())
	}
	return app
}

// GetPublicAccess returns only reconciled presence and non-secret settings.
func (app *App) GetPublicAccess() PublicAccessSnapshot {
	app.publicAccessCommandMu.Lock()
	defer app.publicAccessCommandMu.Unlock()
	if app.deps.PublicAccess != nil {
		snapshot := app.deps.PublicAccess.Snapshot()
		app.acceptPublicAccessSnapshot(snapshot, false)
		return routePublicAccessSnapshot(snapshot)
	}
	app.loadPublicAccessLocked(app.contextSnapshot())
	return app.publicAccessSnapshotLocked()
}

// SavePublicAccessSettings validates the complete proposed revision before
// applying scoped Keychain changes and then the atomic non-secret file write.
func (app *App) SavePublicAccessSettings(payload SavePublicAccessSettingsPayload) PublicAccessCommandResult {
	app.publicAccessCommandMu.Lock()
	defer app.publicAccessCommandMu.Unlock()
	ctx := app.contextSnapshot()
	if app.deps.PublicAccess != nil {
		routed, err := routeSavePublicAccessSettingsRequest(payload)
		if err != nil {
			return app.publicAccessCoreFailure(app.deps.PublicAccess.Snapshot(), tunnelservice.ErrorValidation)
		}
		providerValue := []byte(routed.ReplacementProviderToken)
		passwordValue := []byte(routed.ReplacementPlayerPassword)
		routed.ReplacementProviderToken = ""
		routed.ReplacementPlayerPassword = ""
		defer clear(providerValue)
		defer clear(passwordValue)
		current := app.deps.PublicAccess.Snapshot()
		mutation := tunnelservice.PublicAccessMutation{
			ExpectedRevision:        routed.ExpectedRevision,
			PersistVisibleOverrides: true,
			Preferences: tunnelservice.PublicAccessPreferences{
				Version: tunnelservice.PublicAccessSettingsVersion, EnabledPreference: routed.EnabledPreference,
				ReservedDomain: routed.ReservedDomain, Username: routed.Username,
			},
			ProviderToken:  tunnelservice.SecretMutation{Replacement: providerValue, Delete: routed.DeleteProviderToken},
			PlayerPassword: tunnelservice.SecretMutation{Replacement: passwordValue, Delete: routed.DeletePlayerPassword},
		}
		if routed.ExpectedRevision != current.Preferences.Revision {
			return app.publicAccessCoreFailure(current, tunnelservice.ErrorConflict)
		}
		result := app.deps.PublicAccess.Reconfigure(ctx, mutation)
		app.acceptPublicAccessSnapshot(result.Snapshot, true)
		return routePublicAccessCommandResult(PublicAccessCommandResult{
			OK: result.OK, Error: result.Error, Snapshot: routePublicAccessSnapshot(result.Snapshot),
		})
	}
	app.loadPublicAccessLocked(ctx)

	routed, err := routeSavePublicAccessSettingsRequest(payload)
	if err != nil {
		return app.publicAccessFailureLocked(tunnelservice.ErrorValidation, err.Error())
	}
	app.mu.RLock()
	current := app.publicAccessPreferences
	app.mu.RUnlock()
	if routed.ExpectedRevision != current.Revision {
		return app.publicAccessFailureLocked(tunnelservice.ErrorConflict, tunnelservice.ErrorConflict.SafeMessage())
	}
	proposed := tunnelservice.PublicAccessPreferences{
		Version: tunnelservice.PublicAccessSettingsVersion, EnabledPreference: routed.EnabledPreference,
		ReservedDomain: routed.ReservedDomain, Username: routed.Username, Revision: current.Revision + 1,
		ProviderTokenPresentHint:  current.ProviderTokenPresentHint,
		PlayerPasswordPresentHint: current.PlayerPasswordPresentHint,
	}
	proposed, err = proposed.Normalized()
	if err != nil {
		return app.publicAccessFailureLocked(tunnelservice.ErrorValidation, tunnelservice.ErrorValidation.SafeMessage())
	}
	providerValue := []byte(routed.ReplacementProviderToken)
	passwordValue := []byte(routed.ReplacementPlayerPassword)
	defer clear(providerValue)
	defer clear(passwordValue)
	if len(providerValue) > 0 {
		if err := tunnelservice.ValidateProviderToken(providerValue); err != nil {
			return app.publicAccessFailureLocked(tunnelservice.ErrorValidation, tunnelservice.ErrorValidation.SafeMessage())
		}
	}
	if len(passwordValue) > 0 {
		if err := tunnelservice.ValidatePlayerPassword(passwordValue); err != nil {
			return app.publicAccessFailureLocked(tunnelservice.ErrorValidation, tunnelservice.ErrorValidation.SafeMessage())
		}
	}
	if err := app.applySecretMutationLocked(ctx, tunnelservice.ProviderAccountToken, providerValue, routed.DeleteProviderToken); err != nil {
		return app.publicAccessSecretFailureLocked(err)
	}
	if err := app.applySecretMutationLocked(ctx, tunnelservice.PlayerBasicAuthPassword, passwordValue, routed.DeletePlayerPassword); err != nil {
		app.reconcilePublicAccessPresenceLocked(ctx)
		return app.publicAccessSecretFailureLocked(err)
	}
	app.reconcilePublicAccessPresenceLocked(ctx)
	app.mu.RLock()
	proposed.ProviderTokenPresentHint = app.providerTokenPresence == tunnelservice.SecretPresent
	proposed.PlayerPasswordPresentHint = app.playerPasswordPresence == tunnelservice.SecretPresent
	app.mu.RUnlock()
	if app.deps.PublicSettings == nil {
		return app.publicAccessFailureLocked(tunnelservice.ErrorSettingsCorrupt, tunnelservice.ErrorSettingsCorrupt.SafeMessage())
	}
	if err := app.deps.PublicSettings.Save(proposed); err != nil {
		return app.publicAccessFailureLocked(tunnelservice.ErrorSettingsCorrupt, tunnelservice.ErrorSettingsCorrupt.SafeMessage())
	}
	app.mu.Lock()
	app.publicAccessPreferences = proposed
	app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleDisabled, Generation: app.publicAccessStatus.Generation + 1, SettingsRevision: proposed.Revision}
	app.mu.Unlock()
	return app.publicAccessSuccessLocked()
}

func (app *App) GeneratePlayerPassword(payload PublicAccessCommandPayload) GeneratedPlayerPasswordResult {
	app.publicAccessCommandMu.Lock()
	defer app.publicAccessCommandMu.Unlock()
	ctx := app.contextSnapshot()
	if app.deps.PublicAccess != nil {
		current := app.deps.PublicAccess.Snapshot()
		routed := routePublicAccessCommandRequest(payload)
		if routed.ExpectedRevision != current.Preferences.Revision {
			return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: tunnelservice.ErrorConflict.SafeMessage(), SettingsRevision: current.Preferences.Revision})
		}
		source := app.deps.PasswordEntropy
		if source == nil {
			source = rand.Reader
		}
		generated, err := tunnelservice.GeneratePlayerPassword(source)
		if err != nil {
			return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: tunnelservice.ErrorProviderFailure.SafeMessage(), SettingsRevision: current.Preferences.Revision})
		}
		defer clear(generated)
		oneTimeValue := string(generated)
		result := app.deps.PublicAccess.Reconfigure(ctx, tunnelservice.PublicAccessMutation{
			ExpectedRevision: routed.ExpectedRevision,
			Preferences:      current.Preferences,
			PlayerPassword:   tunnelservice.SecretMutation{Replacement: generated},
		})
		app.acceptPublicAccessSnapshot(result.Snapshot, true)
		if !result.OK {
			oneTimeValue = ""
			return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: result.Error, SettingsRevision: result.Snapshot.Preferences.Revision})
		}
		return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{OK: true, GeneratedPassword: oneTimeValue, SettingsRevision: result.Snapshot.Preferences.Revision})
	}
	app.loadPublicAccessLocked(ctx)
	routed := routePublicAccessCommandRequest(payload)
	app.mu.RLock()
	current := app.publicAccessPreferences
	app.mu.RUnlock()
	if routed.ExpectedRevision != current.Revision {
		return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: tunnelservice.ErrorConflict.SafeMessage(), SettingsRevision: current.Revision})
	}
	source := app.deps.PasswordEntropy
	if source == nil {
		source = rand.Reader
	}
	generated, err := tunnelservice.GeneratePlayerPassword(source)
	if err != nil {
		return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: tunnelservice.ErrorProviderFailure.SafeMessage(), SettingsRevision: current.Revision})
	}
	defer clear(generated)
	if err := tunnelservice.ReplaceSecret(ctx, app.deps.PublicSecrets, tunnelservice.PlayerBasicAuthPassword, generated); err != nil {
		app.publicAccessSecretFailureLocked(err)
		return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: publicAccessSecretCategory(err).SafeMessage(), SettingsRevision: current.Revision})
	}
	current.Revision++
	current.PlayerPasswordPresentHint = true
	if app.deps.PublicSettings == nil || savePublicAccessMutationSettings(app.deps.PublicSettings, current, false) != nil {
		app.reconcilePublicAccessPresenceLocked(ctx)
		app.publicAccessFailureLocked(tunnelservice.ErrorSettingsCorrupt, tunnelservice.ErrorSettingsCorrupt.SafeMessage())
		return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{Error: tunnelservice.ErrorSettingsCorrupt.SafeMessage(), SettingsRevision: current.Revision - 1})
	}
	app.mu.Lock()
	app.publicAccessPreferences = current
	app.playerPasswordPresence = tunnelservice.SecretPresent
	app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleDisabled, Generation: app.publicAccessStatus.Generation + 1, SettingsRevision: current.Revision}
	app.mu.Unlock()
	app.emitPublicAccessStatusLocked()
	return routeGeneratedPlayerPasswordResult(GeneratedPlayerPasswordResult{OK: true, GeneratedPassword: string(generated), SettingsRevision: current.Revision})
}

type mutationAwarePublicAccessSettingsStore interface {
	SaveForMutation(tunnelservice.PublicAccessPreferences, bool) error
}

func savePublicAccessMutationSettings(settings PublicAccessSettingsStore, preferences tunnelservice.PublicAccessPreferences, persistVisibleOverrides bool) error {
	if aware, ok := settings.(mutationAwarePublicAccessSettingsStore); ok {
		return aware.SaveForMutation(preferences, persistVisibleOverrides)
	}
	return settings.Save(preferences)
}

func (app *App) StartPublicAccess(payload PublicAccessCommandPayload) PublicAccessCommandResult {
	app.publicAccessCommandMu.Lock()
	defer app.publicAccessCommandMu.Unlock()
	if app.deps.PublicAccess != nil {
		result := app.deps.PublicAccess.Start(app.contextSnapshot(), routePublicAccessCommandRequest(payload).ExpectedRevision)
		app.acceptPublicAccessSnapshot(result.Snapshot, true)
		return routePublicAccessCommandResult(PublicAccessCommandResult{
			OK: result.OK, Error: result.Error, Snapshot: routePublicAccessSnapshot(result.Snapshot),
		})
	}
	app.loadPublicAccessLocked(app.contextSnapshot())
	if routePublicAccessCommandRequest(payload).ExpectedRevision != app.publicAccessPreferences.Revision {
		return app.publicAccessFailureLocked(tunnelservice.ErrorConflict, tunnelservice.ErrorConflict.SafeMessage())
	}
	return app.publicAccessFailureLocked(tunnelservice.ErrorProviderFailure, "Public access is not available yet; local access remains available.")
}

func (app *App) StopPublicAccess(payload PublicAccessCommandPayload) PublicAccessCommandResult {
	app.publicAccessCommandMu.Lock()
	defer app.publicAccessCommandMu.Unlock()
	if app.deps.PublicAccess != nil {
		result := app.deps.PublicAccess.Stop(app.contextSnapshot(), routePublicAccessCommandRequest(payload).ExpectedRevision)
		app.acceptPublicAccessSnapshot(result.Snapshot, true)
		return routePublicAccessCommandResult(PublicAccessCommandResult{
			OK: result.OK, Error: result.Error, Snapshot: routePublicAccessSnapshot(result.Snapshot),
		})
	}
	app.loadPublicAccessLocked(app.contextSnapshot())
	if routePublicAccessCommandRequest(payload).ExpectedRevision != app.publicAccessPreferences.Revision {
		return app.publicAccessFailureLocked(tunnelservice.ErrorConflict, tunnelservice.ErrorConflict.SafeMessage())
	}
	app.mu.Lock()
	app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleDisabled, Generation: app.publicAccessStatus.Generation + 1, SettingsRevision: app.publicAccessPreferences.Revision}
	app.mu.Unlock()
	return app.publicAccessSuccessLocked()
}

func (app *App) loadPublicAccessLocked(ctx context.Context) {
	app.mu.RLock()
	loaded := app.publicAccessLoaded
	app.mu.RUnlock()
	if loaded {
		return
	}
	preferences := tunnelservice.DefaultPublicAccessPreferences()
	category := tunnelservice.ErrorCategory(0)
	message := ""
	if app.deps.PublicSettings != nil {
		loadedPreferences, err := app.deps.PublicSettings.Load()
		if err == nil {
			preferences = loadedPreferences
		} else if errors.Is(err, tunnelservice.ErrSettingsRecovered) {
			category, message = tunnelservice.ErrorSettingsCorrupt, tunnelservice.ErrorSettingsCorrupt.SafeMessage()
		} else {
			category, message = tunnelservice.ErrorSettingsCorrupt, tunnelservice.ErrorSettingsCorrupt.SafeMessage()
		}
	}
	app.mu.Lock()
	app.publicAccessLoaded = true
	app.publicAccessPreferences = preferences
	app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleDisabled, SettingsRevision: preferences.Revision}
	if category != 0 {
		app.publicAccessStatus.State = tunnelservice.LifecycleFailed
		app.publicAccessStatus.ErrorCategory = category
		app.publicAccessStatus.ErrorMessage = message
	}
	app.mu.Unlock()
	app.reconcilePublicAccessPresenceLocked(ctx)
}

func (app *App) reconcilePublicAccessPresenceLocked(ctx context.Context) {
	provider, providerErr := tunnelservice.SecretUnknown, error(nil)
	password, passwordErr := tunnelservice.SecretUnknown, error(nil)
	if app.deps.PublicSecrets != nil {
		provider, providerErr = app.deps.PublicSecrets.Presence(ctx, tunnelservice.ProviderAccountToken)
		password, passwordErr = app.deps.PublicSecrets.Presence(ctx, tunnelservice.PlayerBasicAuthPassword)
	}
	app.mu.Lock()
	app.providerTokenPresence = provider
	app.playerPasswordPresence = password
	if providerErr != nil || passwordErr != nil {
		failure := providerErr
		if failure == nil {
			failure = passwordErr
		}
		category := publicAccessSecretCategory(failure)
		app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleFailed, Generation: app.publicAccessStatus.Generation, SettingsRevision: app.publicAccessPreferences.Revision, ErrorCategory: category, ErrorMessage: category.SafeMessage()}
	}
	app.mu.Unlock()
}

func (app *App) applySecretMutationLocked(ctx context.Context, ref tunnelservice.SecretRef, replacement []byte, deleteValue bool) error {
	if len(replacement) > 0 {
		return tunnelservice.ReplaceSecret(ctx, app.deps.PublicSecrets, ref, replacement)
	}
	if deleteValue {
		return tunnelservice.DeleteSecret(ctx, app.deps.PublicSecrets, ref)
	}
	return nil
}

func (app *App) publicAccessSnapshotLocked() PublicAccessSnapshot {
	app.mu.RLock()
	snapshot := tunnelservice.PublicAccessSnapshot{Preferences: app.publicAccessPreferences, ProviderTokenPresence: app.providerTokenPresence, PlayerPasswordPresence: app.playerPasswordPresence, Status: app.publicAccessStatus}
	app.mu.RUnlock()
	return routePublicAccessSnapshot(snapshot)
}

func (app *App) publicAccessSuccessLocked() PublicAccessCommandResult {
	snapshot := app.publicAccessSnapshotLocked()
	app.emitPublicAccessStatusLocked()
	return routePublicAccessCommandResult(PublicAccessCommandResult{OK: true, Snapshot: snapshot})
}

func (app *App) publicAccessFailureLocked(category tunnelservice.ErrorCategory, message string) PublicAccessCommandResult {
	app.mu.Lock()
	app.publicAccessStatus = tunnelservice.PublicAccessStatus{State: tunnelservice.LifecycleFailed, Generation: app.publicAccessStatus.Generation, SettingsRevision: app.publicAccessPreferences.Revision, ErrorCategory: category, ErrorMessage: message}
	app.mu.Unlock()
	snapshot := app.publicAccessSnapshotLocked()
	app.emitPublicAccessStatusLocked()
	return routePublicAccessCommandResult(PublicAccessCommandResult{Error: message, Snapshot: snapshot})
}

func (app *App) publicAccessSecretFailureLocked(err error) PublicAccessCommandResult {
	category := publicAccessSecretCategory(err)
	return app.publicAccessFailureLocked(category, category.SafeMessage())
}

func publicAccessSecretCategory(err error) tunnelservice.ErrorCategory {
	switch {
	case errors.Is(err, tunnelservice.ErrSecretStoreLocked):
		return tunnelservice.ErrorSecretStoreLocked
	case errors.Is(err, tunnelservice.ErrSecretStoreDenied), errors.Is(err, tunnelservice.ErrSecretStoreUserCancelled):
		return tunnelservice.ErrorSecretStoreDenied
	default:
		return tunnelservice.ErrorSecretStoreUnavailable
	}
}

func (app *App) emitPublicAccessStatusLocked() {
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(publicAccessStatusEvent, app.publicAccessSnapshotLocked())
	}
}

func (app *App) acceptPublicAccessSnapshot(snapshot tunnelservice.PublicAccessSnapshot, emit bool) {
	app.mu.Lock()
	if app.publicAccessLoaded && publicAccessSnapshotOlder(snapshot, app.publicAccessPreferences, app.publicAccessStatus) {
		app.mu.Unlock()
		return
	}
	app.publicAccessLoaded = true
	app.publicAccessPreferences = snapshot.Preferences
	app.providerTokenPresence = snapshot.ProviderTokenPresence
	app.playerPasswordPresence = snapshot.PlayerPasswordPresence
	app.publicAccessStatus = snapshot.Status
	serverInfoChanged := false
	if app.serverInfo != nil {
		if snapshot.Status.State == tunnelservice.LifecycleReady && snapshot.Status.PublicURL != "" {
			localURL := app.serverInfo.LocalURL
			if localURL == "" || app.serverInfo.Tunnel == false {
				localURL = app.serverInfo.URL
			}
			if app.serverInfo.URL != snapshot.Status.PublicURL || !app.serverInfo.Tunnel {
				app.serverInfo.URL = snapshot.Status.PublicURL
				app.serverInfo.LocalURL = localURL
				app.serverInfo.Tunnel = true
				app.serverInfo.TunnelError = ""
				serverInfoChanged = true
			}
		} else if app.serverInfo.Tunnel {
			app.serverInfo.URL = app.serverInfo.LocalURL
			app.serverInfo.Tunnel = false
			app.serverInfo.TunnelError = snapshot.Status.ErrorMessage
			serverInfoChanged = true
		}
	}
	serverInfo := cloneServerInfoPointer(app.serverInfo)
	app.mu.Unlock()
	if emit && app.deps.Events != nil {
		if serverInfoChanged && serverInfo != nil {
			_ = app.deps.Events.Emit(serverInfoEvent, routeServerInfoEvent(*serverInfo))
		}
		_ = app.deps.Events.Emit(publicAccessStatusEvent, routePublicAccessSnapshot(snapshot))
	}
}

func (app *App) publicAccessCoreFailure(snapshot tunnelservice.PublicAccessSnapshot, category tunnelservice.ErrorCategory) PublicAccessCommandResult {
	app.acceptPublicAccessSnapshot(snapshot, false)
	return routePublicAccessCommandResult(PublicAccessCommandResult{
		Error: category.SafeMessage(), Snapshot: routePublicAccessSnapshot(snapshot),
	})
}

func publicAccessSnapshotOlder(candidate tunnelservice.PublicAccessSnapshot, preferences tunnelservice.PublicAccessPreferences, status tunnelservice.PublicAccessStatus) bool {
	return candidate.Status.Generation < status.Generation ||
		candidate.Status.Generation == status.Generation && candidate.Preferences.Revision < preferences.Revision
}

// Start acquires the player listener, publishes local status, allows the
// desktop to become ready, then starts optional public access. Fatal partial
// startup is unwound in reverse acquisition order.
func (app *App) Start(ctx context.Context) error {
	app.lifecycleMu.Lock()
	defer app.lifecycleMu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}

	app.mu.RLock()
	alreadyStarted := app.playerStarted
	app.mu.RUnlock()
	if alreadyStarted {
		return nil
	}
	if app.deps.Player == nil {
		return app.failLocked(errors.New("player server is not configured"))
	}
	if setter, ok := app.deps.Events.(interface{ SetContext(context.Context) }); ok {
		setter.SetContext(ctx)
	}
	app.mu.Lock()
	app.ctx = ctx
	app.stopped = false
	app.mu.Unlock()
	if app.deps.PublicAccess == nil {
		app.publicAccessCommandMu.Lock()
		app.loadPublicAccessLocked(ctx)
		app.publicAccessCommandMu.Unlock()
	}

	acquisitionContext := ctx
	if app.deps.StartupTimeout > 0 {
		bounded, cancel := context.WithTimeout(ctx, app.deps.StartupTimeout)
		defer cancel()
		acquisitionContext = bounded
	}

	app.setPhase("starting-player-server")
	info, err := app.deps.Player.Start(acquisitionContext)
	if err != nil {
		return app.failLocked(fmt.Errorf("start player server: %w", err))
	}
	app.mu.Lock()
	app.playerStarted = true
	app.serverInfo = cloneServerInfo(info)
	app.mu.Unlock()
	if app.deps.PublicAccess != nil {
		app.acceptPublicAccessSnapshot(app.deps.PublicAccess.Initialize(ctx), false)
	}

	if app.deps.Events != nil {
		if err := app.deps.Events.Emit(serverInfoEvent, routeServerInfoEvent(info)); err != nil {
			return app.failLocked(fmt.Errorf("publish player server status to desktop bridge: %w", err))
		}
	}

	app.setPhase("desktop-loading")
	if app.deps.Desktop != nil {
		if err := app.deps.Desktop.Ready(ctx); err != nil {
			return app.failLocked(fmt.Errorf("make desktop ready: %w", err))
		}
		app.mu.Lock()
		app.desktopReady = true
		app.mu.Unlock()
	}
	app.setPhase("ready-local")
	if app.deps.PublicAccess != nil || app.deps.PublicSettings != nil || app.deps.PublicSecrets != nil {
		app.publicAccessCommandMu.Lock()
		app.emitPublicAccessStatusLocked()
		app.publicAccessCommandMu.Unlock()
	}

	return nil
}

// Shutdown releases public access, the player listener, persistence worker, then desktop.
// It is safe in every lifecycle phase and on repeated calls.
func (app *App) Shutdown(ctx context.Context) error {
	app.lifecycleMu.Lock()
	defer app.lifecycleMu.Unlock()
	return app.shutdownLocked(ctx, false)
}

// GetRuntimeStatus returns a detached status snapshot.
func (app *App) GetRuntimeStatus() RuntimeStatus {
	app.mu.RLock()
	defer app.mu.RUnlock()
	status := RuntimeStatus{
		ServerInfo:        cloneServerInfoPointer(app.serverInfo),
		ClientCount:       app.clientCount,
		HackState:         clonePublicHackState(app.hackState),
		StartupError:      app.startupError,
		SaveState:         app.saveState,
		RequestedRevision: app.requestedRevision,
		SavedRevision:     app.savedRevision,
		CoordinationState: domain.CloneMasterCoordinationState(app.coordinationState),
	}
	return routeRuntimeStatus(status)
}

// NewSession opens the native destination dialog and creates a validated
// starter session.
func (app *App) NewSession() sessionservice.SessionResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SessionResult{Error: "session service is unavailable"}
	}
	result := commands.Create(app.contextSnapshot())
	app.captureSessionStatus(commands)
	app.resetPlayerConfigForSession(result)
	return routeSessionOperationResult(result)
}

// OpenSession opens and validates an existing version-1 session.
func (app *App) OpenSession() sessionservice.SessionResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SessionResult{Error: "session service is unavailable"}
	}
	result := commands.Open(app.contextSnapshot())
	app.captureSessionStatus(commands)
	app.resetPlayerConfigForSession(result)
	return routeSessionOperationResult(result)
}

// CopyDemo creates an explicit writable copy of the bundled demo.
func (app *App) CopyDemo() sessionservice.SessionResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SessionResult{Error: "session service is unavailable"}
	}
	result := commands.CopyDemo(app.contextSnapshot())
	app.captureSessionStatus(commands)
	app.resetPlayerConfigForSession(result)
	return routeSessionOperationResult(result)
}

// SaveSession assigns a monotonic revision and waits until it or a newer
// accepted revision is durably replaced.
func (app *App) SaveSession(session domain.Session) sessionservice.SaveResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SaveResult{Error: "session service is unavailable"}
	}
	app.mu.Lock()
	app.requestedRevision++
	revision := app.requestedRevision
	app.saveState = string(sessionservice.SaveStateSaving)
	app.mu.Unlock()

	result := commands.Save(app.contextSnapshot(), session, revision)
	app.mu.Lock()
	if result.SavedRevision > app.savedRevision {
		app.savedRevision = result.SavedRevision
	}
	if revision == app.requestedRevision {
		if result.OK {
			app.saveState = string(sessionservice.SaveStateSaved)
		} else {
			app.saveState = string(sessionservice.SaveStateFailed)
		}
	}
	app.mu.Unlock()
	return routeSaveSessionResult(result)
}

// LoadReferencedPlayerConfig reloads the active session's durable roster.
func (app *App) LoadReferencedPlayerConfig() PlayerConfigCommandResult {
	app.coordinationCommandMu.Lock()
	defer app.coordinationCommandMu.Unlock()
	sessions, ok := app.deps.Sessions.(sessionPlayerConfigCommands)
	if !ok || app.deps.PlayerConfigs == nil {
		return app.playerConfigFailure("player config service is unavailable", false)
	}
	if failure := app.playerConfigChangePrecondition(); failure != nil {
		return *failure
	}
	active := sessions.Snapshot()
	if strings.TrimSpace(active.Session.PlayerConfig) == "" {
		return app.playerConfigFailure("no player config is associated with this session", false)
	}
	loaded := app.deps.PlayerConfigs.LoadReferenced(active.Path, active.Session.PlayerConfig)
	return app.installPlayerConfig(loaded, false)
}

// NewPlayerConfig creates, associates, and installs one empty durable roster.
func (app *App) NewPlayerConfig() PlayerConfigCommandResult {
	app.coordinationCommandMu.Lock()
	defer app.coordinationCommandMu.Unlock()
	if app.deps.PlayerConfigs == nil {
		return app.playerConfigFailure("player config service is unavailable", false)
	}
	if failure := app.playerConfigChangePrecondition(); failure != nil {
		return *failure
	}
	return app.installPlayerConfig(app.deps.PlayerConfigs.Create(app.contextSnapshot()), true)
}

// OpenPlayerConfig selects, associates, and installs an existing durable roster.
func (app *App) OpenPlayerConfig() PlayerConfigCommandResult {
	app.coordinationCommandMu.Lock()
	defer app.coordinationCommandMu.Unlock()
	if app.deps.PlayerConfigs == nil {
		return app.playerConfigFailure("player config service is unavailable", false)
	}
	if failure := app.playerConfigChangePrecondition(); failure != nil {
		return *failure
	}
	return app.installPlayerConfig(app.deps.PlayerConfigs.Open(app.contextSnapshot()), true)
}

func (app *App) playerConfigChangePrecondition() *PlayerConfigCommandResult {
	sessions, ok := app.deps.Sessions.(sessionPlayerConfigCommands)
	if !ok {
		result := app.playerConfigFailure("session service cannot associate player configs", false)
		return &result
	}
	active := sessions.Snapshot()
	if active.Path == "" || active.Session == nil {
		result := app.playerConfigFailure("there is no active session", false)
		return &result
	}
	app.mu.RLock()
	broadcastActive := app.coordinationState != nil && app.coordinationState.Broadcast != nil
	app.mu.RUnlock()
	if broadcastActive {
		result := app.playerConfigFailure("player config cannot change during a broadcast", false)
		return &result
	}
	return nil
}

func (app *App) installPlayerConfig(result playerconfigservice.Result, associate bool) PlayerConfigCommandResult {
	if result.Canceled {
		return app.playerConfigFailure("", true)
	}
	if !result.OK || result.Config == nil || result.FilePath == "" {
		message := result.Error
		if message == "" {
			message = "player config could not be loaded"
		}
		return app.playerConfigFailure(message, false)
	}
	coordination, ok := app.deps.Coordination.(coordinationPlayerConfigService)
	if !ok {
		return app.playerConfigFailure("coordination service cannot install player configs", false)
	}
	var associatedSession *domain.Session
	if associate {
		sessions, ok := app.deps.Sessions.(sessionPlayerConfigCommands)
		if !ok {
			return app.playerConfigFailure("session service cannot associate player configs", false)
		}
		associated := sessions.AssociatePlayerConfig(app.contextSnapshot(), result.FilePath)
		if !associated.OK {
			message := associated.Error
			if message == "" {
				message = "player config association could not be saved"
			}
			return app.playerConfigFailure(message, associated.Canceled)
		}
		associatedSession = associated.Session
	} else if sessions, ok := app.deps.Sessions.(sessionPlayerConfigCommands); ok {
		associatedSession = sessions.Snapshot().Session
	}
	handle := domain.PlayerConfigHandle{Path: result.FilePath, Version: result.Config.Version, Name: result.Config.Name}
	state, err := coordination.InstallPlayerConfig(handle, result.Config.Roster)
	if err != nil {
		return app.playerConfigFailure(err.Error(), false)
	}
	app.publishCoordinationState(state)
	return routePlayerConfigResult(PlayerConfigCommandResult{
		OK:      true,
		Config:  &domain.PlayerConfigMetadata{Status: "loaded", FilePath: handle.Path, Version: handle.Version, Name: handle.Name},
		Session: associatedSession,
		State:   domain.CloneMasterCoordinationState(state),
	})
}

func (app *App) playerConfigFailure(message string, canceled bool) PlayerConfigCommandResult {
	app.mu.RLock()
	state := domain.CloneMasterCoordinationState(app.coordinationState)
	app.mu.RUnlock()
	return routePlayerConfigResult(PlayerConfigCommandResult{Canceled: canceled, Error: message, State: state})
}

func (app *App) resetPlayerConfigForSession(result sessionservice.SessionResult) {
	if !result.OK {
		return
	}
	coordination, ok := app.deps.Coordination.(coordinationPlayerConfigService)
	if !ok {
		return
	}
	state, err := coordination.ClearPlayerConfig()
	if err == nil && state != nil {
		app.publishCoordinationState(state)
	}
}

// AddCharacter validates the game-master display name before entering the
// coordinator and publishes only its detached authoritative projection.
func (app *App) AddCharacter(name string) CoordinationCommandResult {
	name = routeAddCharacterRequest(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return app.coordinationFailure("character name must not be blank")
	}
	if len([]rune(name)) > 80 {
		return app.coordinationFailure("character name must be at most 80 characters")
	}
	if app.deps.Coordination == nil {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := app.deps.Coordination.AddCharacter(name)
	if err != nil {
		return app.coordinationFailure(err.Error())
	}
	if state == nil {
		return app.coordinationFailure("character could not be added")
	}
	app.publishCoordinationState(state)
	return routeCoordinationResult(CoordinationCommandResult{OK: true, State: domain.CloneMasterCoordinationState(state)})
}

// RenameCharacter validates the stable roster ID and display name before
// entering the coordinator transaction.
func (app *App) RenameCharacter(payload CharacterRenamePayload) CoordinationCommandResult {
	payload = routeRenameCharacterRequest(payload)
	if strings.TrimSpace(string(payload.CharacterID)) == "" {
		return app.coordinationFailure("character ID must not be blank")
	}
	name, err := validatedCoordinationDisplayName(payload.Name, "character name")
	if err != nil {
		return app.coordinationFailure(err.Error())
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, commandErr := coordination.RenameCharacter(payload.CharacterID, name)
	return app.completeCoordinationCommand(state, commandErr, "character could not be renamed")
}

// DeleteCharacter removes only an existing unclaimed roster identity.
func (app *App) DeleteCharacter(characterID string) CoordinationCommandResult {
	characterID = routeDeleteCharacterRequest(characterID)
	if strings.TrimSpace(characterID) == "" {
		return app.coordinationFailure("character ID must not be blank")
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.DeleteCharacter(domain.CharacterID(characterID))
	return app.completeCoordinationCommand(state, err, "character could not be deleted")
}

// RenameLogicalSession validates and trims the private fallback label without
// exposing it to durable session persistence.
func (app *App) RenameLogicalSession(payload LogicalSessionRenamePayload) CoordinationCommandResult {
	payload = routeRenameLogicalSessionRequest(payload)
	if strings.TrimSpace(string(payload.SessionID)) == "" {
		return app.coordinationFailure("session ID must not be blank")
	}
	fallbackName, err := validatedCoordinationDisplayName(payload.FallbackName, "session fallback name")
	if err != nil {
		return app.coordinationFailure(err.Error())
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, commandErr := coordination.RenameLogicalSession(payload.SessionID, fallbackName)
	return app.completeCoordinationCommand(state, commandErr, "logical session could not be renamed")
}

// AssignCharacter installs one authoritative current-broadcast claim.
func (app *App) AssignCharacter(payload AssignmentPayload) CoordinationCommandResult {
	payload = routeAssignCharacterRequest(payload)
	if strings.TrimSpace(string(payload.SessionID)) == "" {
		return app.coordinationFailure("session ID must not be blank")
	}
	if strings.TrimSpace(string(payload.CharacterID)) == "" {
		return app.coordinationFailure("character ID must not be blank")
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.AssignCharacter(payload.SessionID, payload.CharacterID)
	return app.completeCoordinationCommand(state, err, "character could not be assigned")
}

// ReleaseCharacter removes one claim and leaves controller selection to an
// explicit trusted command.
func (app *App) ReleaseCharacter(sessionID string) CoordinationCommandResult {
	sessionID = routeReleaseCharacterRequest(sessionID)
	if strings.TrimSpace(sessionID) == "" {
		return app.coordinationFailure("session ID must not be blank")
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.ReleaseCharacter(domain.LogicalSessionID(sessionID))
	return app.completeCoordinationCommand(state, err, "character could not be released")
}

// MoveCharacter transfers a stable roster identity to one unassigned logical
// session in the coordinator's single authoritative order.
func (app *App) MoveCharacter(payload MoveCharacterPayload) CoordinationCommandResult {
	payload = routeMoveCharacterRequest(payload)
	if strings.TrimSpace(string(payload.CharacterID)) == "" {
		return app.coordinationFailure("character ID must not be blank")
	}
	if strings.TrimSpace(string(payload.ToSessionID)) == "" {
		return app.coordinationFailure("destination session ID must not be blank")
	}
	coordination, ok := app.deps.Coordination.(coordinationCorrectionService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.MoveCharacter(payload.CharacterID, payload.ToSessionID)
	return app.completeCoordinationCommand(state, err, "character could not be moved")
}

// SetActiveController atomically designates one connected assigned logical
// session as the sole controller for the current broadcast.
func (app *App) SetActiveController(sessionID string) CoordinationCommandResult {
	sessionID = routeSetActiveControllerRequest(sessionID)
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return app.coordinationFailure("session ID must not be blank")
	}
	coordination, ok := app.deps.Coordination.(coordinationControllerService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.SetActiveController(domain.LogicalSessionID(sessionID))
	return app.completeCoordinationCommand(state, err, "active controller could not be changed")
}

// StartBroadcast creates a fresh broadcast epoch through the coordinator.
func (app *App) StartBroadcast() CoordinationCommandResult {
	app.coordinationCommandMu.Lock()
	defer app.coordinationCommandMu.Unlock()
	if app.deps.Coordination == nil {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := app.deps.Coordination.StartBroadcast()
	if err != nil {
		return app.coordinationFailure(err.Error())
	}
	if state == nil {
		return app.coordinationFailure("broadcast could not be started")
	}
	app.publishCoordinationState(state)
	return routeCoordinationResult(CoordinationCommandResult{OK: true, State: domain.CloneMasterCoordinationState(state)})
}

// EndBroadcast ends only the process-local broadcast epoch. Authored durable
// terminals and the active session document remain outside this boundary.
func (app *App) EndBroadcast() CoordinationCommandResult {
	coordination, ok := app.deps.Coordination.(coordinationBroadcastLifecycleService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.EndBroadcast()
	return app.completeCoordinationCommand(state, err, "broadcast could not be ended")
}

// RequestTerminalActivation validates and normalizes authored content before
// asking the coordinator to atomically select the broadcast-wide terminal.
func (app *App) RequestTerminalActivation(payload LiveTerminalPayload) TerminalSwitchCommandResult {
	payload.TerminalID = strings.TrimSpace(payload.TerminalID)
	payload.TerminalName = strings.TrimSpace(payload.TerminalName)
	if err := validateLiveTerminal(payload.TerminalID, payload.TerminalName, payload.Tree, payload.HackLevel, payload.IntroText); err != nil {
		return app.terminalSwitchFailure(err.Error(), nil)
	}
	payload, err := routeTerminalActivationRequest(payload, false)
	if err != nil {
		return app.terminalSwitchFailure("terminal request could not be represented by the private contract", nil)
	}
	coordination, ok := app.deps.Coordination.(coordinationTerminalService)
	if !ok {
		return app.terminalSwitchFailure("coordination service is unavailable", nil)
	}
	state, err := coordination.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: payload.TerminalID, TerminalName: payload.TerminalName,
		Tree: payload.Tree, HackLevel: payload.HackLevel, IntroText: payload.IntroText,
	})
	return app.completeTerminalSwitchRequest(state, err, "activated", "terminal could not be activated")
}

// RequestTerminalClear clears only the active terminal selection. The
// broadcast, assignments, identities, and controller remain authoritative.
func (app *App) RequestTerminalClear() TerminalSwitchCommandResult {
	coordination, ok := app.deps.Coordination.(coordinationTerminalService)
	if !ok {
		return app.terminalSwitchFailure("coordination service is unavailable", nil)
	}
	state, err := coordination.RequestTerminalClear()
	return app.completeTerminalSwitchRequest(state, err, "cleared", "active terminal could not be cleared")
}

// ResolveTerminalSwitch validates the opaque decision payload and lets the
// coordinator re-evaluate the latest private source state before committing.
func (app *App) ResolveTerminalSwitch(payload TerminalSwitchDecisionPayload) TerminalSwitchCommandResult {
	if payload.SwitchID == "" {
		return app.terminalSwitchFailure("switch ID must not be blank", nil)
	}
	if payload.Decision != domain.TerminalSwitchPreserve && payload.Decision != domain.TerminalSwitchDiscard && payload.Decision != domain.TerminalSwitchCancel {
		return app.terminalSwitchFailure("terminal switch decision must be preserve, discard, or cancel", nil)
	}
	payload, err := routeTerminalSwitchDecisionRequest(payload)
	if err != nil {
		return app.terminalSwitchFailure("terminal switch decision could not be represented by the private contract", nil)
	}
	coordination, ok := app.deps.Coordination.(coordinationTerminalDecisionService)
	if !ok {
		return app.terminalSwitchFailure("coordination service is unavailable", nil)
	}
	app.mu.RLock()
	pending := domain.CloneMasterCoordinationState(app.coordinationState)
	app.mu.RUnlock()
	status := "activated"
	if payload.Decision == domain.TerminalSwitchCancel {
		status = "cancelled"
	} else if pending != nil && pending.PendingSwitch != nil && pending.PendingSwitch.SwitchID == payload.SwitchID && pending.PendingSwitch.TargetTerminalID == nil {
		status = "cleared"
	}
	state, err := coordination.ResolveTerminalSwitch(payload.SwitchID, payload.Decision)
	return app.completeTerminalSwitchCommand(state, err, status, "terminal switch could not be resolved")
}

// UpdateLiveTerminal validates replacement content and preserves the current
// puzzle through the ordered coordinator boundary. The legacy fallback keeps
// older focused hosts compatible while production always provides coordination.
func (app *App) UpdateLiveTerminal(payload LiveUpdatePayload) CoordinationCommandResult {
	intro := ""
	if payload.IntroText != nil {
		intro = *payload.IntroText
	}
	if err := validateLiveTerminal("live-terminal", "Live Terminal", payload.Tree, 0, intro); err != nil {
		return app.coordinationFailure(err.Error())
	}
	payload, err := routeLiveTerminalUpdateRequest(payload)
	if err != nil {
		return app.coordinationFailure("live terminal update could not be represented by the private contract")
	}
	if coordination, ok := app.deps.Coordination.(coordinationTerminalService); ok {
		state, err := coordination.UpdateLiveTerminal(payload.Tree, payload.IntroText)
		return app.completeCoordinationCommand(state, err, "no terminal is active")
	}
	if app.deps.Live == nil {
		return app.coordinationFailure("live service is unavailable")
	}
	state, ok := app.deps.Live.Update(payload.Tree, payload.IntroText)
	if !ok || state == nil {
		return app.coordinationFailure("no terminal is live")
	}
	if publisher, ok := app.deps.Player.(playerPublisher); ok {
		publisher.PublishUpdate()
	}
	app.updateHackState(state.Hack)
	app.mu.RLock()
	coordinationState := domain.CloneMasterCoordinationState(app.coordinationState)
	app.mu.RUnlock()
	return routeCoordinationResult(CoordinationCommandResult{OK: true, State: coordinationState})
}

// ResetFailedHack validates the latest authored terminal payload at the
// trusted desktop edge, then asks the coordinator to replace only the failed
// active puzzle. Player transports never receive this command surface.
func (app *App) ResetFailedHack(payload LiveTerminalPayload) CoordinationCommandResult {
	payload.TerminalID = strings.TrimSpace(payload.TerminalID)
	payload.TerminalName = strings.TrimSpace(payload.TerminalName)
	if err := validateLiveTerminal(payload.TerminalID, payload.TerminalName, payload.Tree, payload.HackLevel, payload.IntroText); err != nil {
		return app.coordinationFailure(err.Error())
	}
	payload, err := routeTerminalActivationRequest(payload, true)
	if err != nil {
		return app.coordinationFailure("failed hacking reset could not be represented by the private contract")
	}
	coordination, ok := app.deps.Coordination.(coordinationTerminalService)
	if !ok {
		return app.coordinationFailure("coordination service is unavailable")
	}
	state, err := coordination.ResetFailedHack(domain.TerminalTarget{
		TerminalID: payload.TerminalID, TerminalName: payload.TerminalName,
		Tree: payload.Tree, HackLevel: payload.HackLevel, IntroText: payload.IntroText,
	})
	return app.completeCoordinationCommand(state, err, "failed hacking puzzle could not be reset")
}

// ForceHackSuccess completes an eligible puzzle and publishes the sanitized
// result.
func (app *App) ForceHackSuccess() CommandResult {
	if coordination, ok := app.deps.Coordination.(coordinatedHackSuccess); ok {
		state, accepted := coordination.ForceHackSuccess()
		if !accepted {
			return commandFailure("no active hacking puzzle")
		}
		app.updateHackState(state)
		return routeCommandResult(CommandResult{OK: true})
	}

	// Compatibility for partial compositions that predate the coordinator's
	// trusted operation. Production injects the ordered coordinator path.
	if app.deps.Live == nil {
		return commandFailure("live service is unavailable")
	}
	state, ok := app.deps.Live.ForceHackSuccess()
	if !ok {
		return commandFailure("no active hacking puzzle")
	}
	if publisher, ok := app.deps.Player.(playerPublisher); ok {
		publisher.PublishHack()
	}
	app.updateHackState(state)
	return routeCommandResult(CommandResult{OK: true})
}

// OpenURL validates immediately before crossing the system-browser boundary.
func (app *App) OpenURL(rawURL string) CommandResult {
	rawURL = routeOpenURLRequest(rawURL)
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return commandFailure("external URL must be an absolute HTTP or HTTPS URL")
	}
	if app.deps.Browser == nil {
		return commandFailure("external browser is unavailable")
	}
	if err := app.deps.Browser.OpenURL(parsed.String()); err != nil {
		return commandFailure("could not open the external URL")
	}
	return routeCommandResult(CommandResult{OK: true})
}

func (app *App) failLocked(cause error) error {
	app.mu.Lock()
	app.startupError = cause.Error()
	app.phase = "failed"
	app.mu.Unlock()
	cleanupContext, cancel := app.freshShutdownContext()
	defer cancel()
	cleanupErr := app.shutdownLocked(cleanupContext, true)
	if cleanupErr != nil {
		return errors.Join(cause, cleanupErr)
	}
	return cause
}

func (app *App) shutdownLocked(ctx context.Context, preserveFailure bool) error {
	app.mu.Lock()
	if app.stopped {
		app.mu.Unlock()
		return nil
	}
	app.phase = "stopping"
	playerStarted := app.playerStarted
	desktopReady := app.desktopReady
	sessionsOpen := app.deps.Sessions != nil && !app.sessionsClosed
	publicAccessOpen := app.deps.PublicAccess != nil && !app.publicAccessClosed
	clearProcessState := !app.processStateCleared
	app.processStateCleared = true
	app.ctx = nil
	app.mu.Unlock()

	var cleanupErrors []error
	if publicAccessOpen {
		if err := app.deps.PublicAccess.Shutdown(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop embedded public access: %w", err))
		} else {
			app.mu.Lock()
			app.publicAccessClosed = true
			app.mu.Unlock()
		}
	}
	if clearProcessState && app.deps.Live != nil {
		app.deps.Live.Clear()
		app.mu.Lock()
		app.hackState = nil
		app.mu.Unlock()
	}
	if clearProcessState {
		if coordination, ok := app.deps.Coordination.(coordinationShutdownService); ok {
			coordination.Shutdown()
			app.mu.Lock()
			app.coordinationState = nil
			app.mu.Unlock()
		}
	}
	if playerStarted && app.deps.Player != nil {
		if err := app.deps.Player.Stop(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop player server: %w", err))
		} else {
			app.mu.Lock()
			app.playerStarted = false
			app.mu.Unlock()
		}
	}
	if sessionsOpen {
		if err := app.deps.Sessions.Shutdown(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop session service: %w", err))
		} else {
			app.mu.Lock()
			app.sessionsClosed = true
			app.mu.Unlock()
		}
	}
	if desktopReady && app.deps.Desktop != nil {
		if err := app.deps.Desktop.Close(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("close desktop runtime: %w", err))
		} else {
			app.mu.Lock()
			app.desktopReady = false
			app.mu.Unlock()
		}
	}

	app.mu.Lock()
	remaining := app.playerStarted || app.desktopReady ||
		(app.deps.Sessions != nil && !app.sessionsClosed) ||
		(app.deps.PublicAccess != nil && !app.publicAccessClosed)
	app.stopped = !remaining
	if !remaining {
		app.serverInfo = nil
	}
	if preserveFailure {
		app.phase = "failed"
	} else if remaining {
		app.phase = "stop-failed"
	} else {
		app.phase = "stopped"
	}
	app.mu.Unlock()
	return errors.Join(cleanupErrors...)
}

func (app *App) setPhase(phase string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.phase = phase
	app.stopped = false
}

func (app *App) freshShutdownContext() (context.Context, context.CancelFunc) {
	timeout := app.deps.ShutdownTimeout
	if timeout <= 0 {
		timeout = wailsShutdownTimeout
	}
	return context.WithTimeout(context.Background(), timeout)
}

func cloneServerInfo(info domain.ServerInfo) *domain.ServerInfo {
	clone := info
	return &clone
}

func cloneServerInfoPointer(info *domain.ServerInfo) *domain.ServerInfo {
	if info == nil {
		return nil
	}
	return cloneServerInfo(*info)
}

func (app *App) contextSnapshot() context.Context {
	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.ctx == nil {
		return context.Background()
	}
	return app.ctx
}

func (app *App) captureSessionStatus(commands sessionCommands) {
	snapshot := commands.Snapshot()
	app.mu.Lock()
	app.saveState = string(snapshot.SaveState)
	app.requestedRevision = snapshot.RequestedRevision
	app.savedRevision = snapshot.SavedRevision
	app.mu.Unlock()
}

// updateHackState is the trusted desktop projection boundary for accepted
// typed guess and pattern actions. Only the detached public projection enters
// RuntimeStatus or crosses the desktop event bridge.
func (app *App) updateHackState(state *domain.PublicHackState) {
	clone := clonePublicHackState(state)
	app.mu.Lock()
	app.hackState = clone
	app.mu.Unlock()
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(hackStateEvent, routeHackStateEvent(clonePublicHackState(clone)))
	}
}

func (app *App) updateClientCount(count int) {
	if count < 0 {
		count = 0
	}
	app.mu.Lock()
	app.clientCount = count
	app.mu.Unlock()
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(clientCountEvent, routeClientCountEvent(count))
	}
}

func (app *App) publishCoordinationState(state *domain.MasterCoordinationState) {
	clone := domain.CloneMasterCoordinationState(state)
	app.mu.Lock()
	app.coordinationState = clone
	app.mu.Unlock()
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(coordinationStateEvent, routeCoordinationEvent(domain.CloneMasterCoordinationState(clone)))
	}
}

func (app *App) coordinationFailure(message string) CoordinationCommandResult {
	app.mu.RLock()
	state := domain.CloneMasterCoordinationState(app.coordinationState)
	app.mu.RUnlock()
	return routeCoordinationResult(CoordinationCommandResult{Error: message, State: state})
}

func (app *App) completeCoordinationCommand(state *domain.MasterCoordinationState, err error, nilStateMessage string) CoordinationCommandResult {
	if err != nil {
		if state == nil {
			return app.coordinationFailure(err.Error())
		}
		return routeCoordinationResult(CoordinationCommandResult{Error: err.Error(), State: domain.CloneMasterCoordinationState(state)})
	}
	if state == nil {
		return app.coordinationFailure(nilStateMessage)
	}
	app.publishCoordinationState(state)
	return routeCoordinationResult(CoordinationCommandResult{OK: true, State: domain.CloneMasterCoordinationState(state)})
}

func (app *App) terminalSwitchFailure(message string, state *domain.MasterCoordinationState) TerminalSwitchCommandResult {
	if state == nil {
		app.mu.RLock()
		state = domain.CloneMasterCoordinationState(app.coordinationState)
		app.mu.RUnlock()
	}
	return routeTerminalSwitchResult(TerminalSwitchCommandResult{Error: message, State: domain.CloneMasterCoordinationState(state)})
}

func (app *App) completeTerminalSwitchCommand(state *domain.MasterCoordinationState, err error, status, nilStateMessage string) TerminalSwitchCommandResult {
	if err != nil {
		return app.terminalSwitchFailure(err.Error(), state)
	}
	if state == nil {
		return app.terminalSwitchFailure(nilStateMessage, nil)
	}
	app.publishCoordinationState(state)
	return routeTerminalSwitchResult(TerminalSwitchCommandResult{
		OK: true, Status: status, State: domain.CloneMasterCoordinationState(state),
	})
}

func (app *App) completeTerminalSwitchRequest(state *domain.MasterCoordinationState, err error, completedStatus, nilStateMessage string) TerminalSwitchCommandResult {
	if err != nil {
		return app.terminalSwitchFailure(err.Error(), state)
	}
	if state == nil {
		return app.terminalSwitchFailure(nilStateMessage, nil)
	}
	status := completedStatus
	var switchID domain.SwitchID
	if state.PendingSwitch != nil {
		status = "decision-required"
		switchID = state.PendingSwitch.SwitchID
	}
	app.publishCoordinationState(state)
	return routeTerminalSwitchResult(TerminalSwitchCommandResult{
		OK: true, Status: status, SwitchID: switchID,
		State: domain.CloneMasterCoordinationState(state),
	})
}

func validatedCoordinationDisplayName(value string, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s must not be blank", label)
	}
	if len([]rune(value)) > 80 {
		return "", fmt.Errorf("%s must be at most 80 characters", label)
	}
	return value, nil
}

func validateLiveTerminal(id, name string, tree domain.ContentNode, hackLevel int, intro string) error {
	validationTree := cloneTreeForBridgeValidation(tree)
	terminal := domain.Terminal{
		ID: id, Name: name, HackLevel: hackLevel, IntroText: intro, Root: validationTree,
	}
	return domain.ValidateSession(domain.Session{
		Version:   1,
		Name:      "Live Terminal",
		Terminals: []domain.Terminal{terminal},
	})
}

// Wails decodes an empty authored folder as either nil or an empty slice
// depending on the JavaScript payload shape. Both represent the same empty
// folder at this bridge; durable JSON decoding remains strict about arrays.
func cloneTreeForBridgeValidation(node domain.ContentNode) domain.ContentNode {
	clone := node
	if node.Type == domain.NodeFolder && node.Children == nil {
		clone.Children = []domain.ContentNode{}
	} else {
		clone.Children = make([]domain.ContentNode, len(node.Children))
	}
	for index := range node.Children {
		clone.Children[index] = cloneTreeForBridgeValidation(node.Children[index])
	}
	return clone
}

func commandFailure(message string) CommandResult {
	return routeCommandResult(CommandResult{Error: message})
}

func clonePublicHackState(state *domain.PublicHackState) *domain.PublicHackState {
	if state == nil {
		return nil
	}
	clone := *state
	if state.Log != nil {
		clone.Log = append([]string{}, state.Log...)
	}
	clone.Patterns = append([]domain.PublicHackPattern(nil), state.Patterns...)
	clone.Columns = make([]domain.HackColumn, len(state.Columns))
	for index, column := range state.Columns {
		clone.Columns[index] = column
		clone.Columns[index].Addresses = append([]string(nil), column.Addresses...)
		clone.Columns[index].Words = append([]domain.HackWord(nil), column.Words...)
	}
	return &clone
}

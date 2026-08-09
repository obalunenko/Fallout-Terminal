package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
)

const (
	serverInfoEvent             = "server-info"
	clientCountEvent            = "client-count"
	hackStateEvent              = "hack-state"
	tunnelStartupFailureMessage = "public tunnel could not start; verify credentials, tunnel executable, endpoint, and network access"
	tunnelAddressFailureMessage = "public tunnel returned an invalid protected HTTPS address; local access remains available"
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

// LiveService owns canonical shared player state.
type LiveService interface {
	Set(string, string, domain.ContentNode, int, string) *domain.PublicLiveState
	Update(domain.ContentNode, *string) (*domain.PublicLiveState, bool)
	Clear()
	Snapshot() *domain.PublicLiveState
	ForceHackSuccess() (*domain.PublicHackState, bool)
}

// Browser is the allowlisted external-browser boundary.
type Browser interface {
	OpenURL(string) error
}

// PlayerServer owns the in-process HTTP/WebSocket listener.
type PlayerServer interface {
	Start(context.Context) (domain.ServerInfo, error)
	Stop(context.Context) error
}

// playerPublisher is implemented by the complete player server. Keeping
// publication separate from the lifecycle interface preserves construction
// compatibility with partial-start and test servers while making every live
// bridge mutation observable by connected browsers.
type playerPublisher interface {
	PublishLive()
	PublishUpdate()
	PublishClear()
	PublishHack()
}

// TunnelService owns the optional public-access process and its temporary
// credential material.
type TunnelService interface {
	Start(context.Context) (domain.ServerInfo, error)
	Stop(context.Context) error
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

// AppDependencies contains constructed services. Construction acquires no
// external resources; Start owns acquisition in contract order.
type AppDependencies struct {
	Sessions      SessionService
	Live          LiveService
	Player        PlayerServer
	Tunnel        TunnelService
	Desktop       DesktopRuntime
	Browser       Browser
	Events        EventSink
	TunnelEnabled bool
}

// RuntimeStatus is the synchronous startup/status snapshot used to avoid
// losing events emitted before the frontend subscribes.
type RuntimeStatus struct {
	ServerInfo        *domain.ServerInfo      `json:"serverInfo"`
	ClientCount       int                     `json:"clientCount"`
	HackState         *domain.PublicHackState `json:"hackState"`
	StartupError      string                  `json:"startupError,omitempty"`
	SaveState         string                  `json:"saveState"`
	RequestedRevision uint64                  `json:"requestedRevision"`
	SavedRevision     uint64                  `json:"savedRevision"`
}

// CommandResult is used for privileged commands that do not return a model.
type CommandResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
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

// App is the Wails composition root. Domain behavior remains in internal
// packages; App owns lifecycle and the narrow desktop facade.
type App struct {
	lifecycleMu sync.Mutex
	mu          sync.RWMutex

	deps AppDependencies
	ctx  context.Context

	phase             string
	serverInfo        *domain.ServerInfo
	clientCount       int
	hackState         *domain.PublicHackState
	startupError      string
	saveState         string
	requestedRevision uint64
	savedRevision     uint64
	playerStarted     bool
	tunnelStarted     bool
	desktopReady      bool
	sessionsClosed    bool
	stopped           bool
}

// NewApp constructs the application without acquiring external resources.
func NewApp() *App {
	return NewAppWithDependencies(AppDependencies{})
}

// NewAppWithDependencies constructs a testable composition root.
func NewAppWithDependencies(deps AppDependencies) *App {
	return &App{deps: deps, phase: "constructed", saveState: string(sessionservice.SaveStateIdle)}
}

// Start acquires the player listener, publishes local status, allows the
// desktop to become ready, then starts optional public access. Fatal partial
// startup is unwound in reverse acquisition order.
func (app *App) Start(ctx context.Context) error {
	app.lifecycleMu.Lock()
	defer app.lifecycleMu.Unlock()

	app.mu.RLock()
	alreadyStarted := app.playerStarted
	app.mu.RUnlock()
	if alreadyStarted {
		return nil
	}
	if app.deps.Player == nil {
		return app.failLocked(ctx, errors.New("player server is not configured"))
	}

	app.setPhase(ctx, "starting-player-server")
	info, err := app.deps.Player.Start(ctx)
	if err != nil {
		return app.failLocked(ctx, fmt.Errorf("start player server: %w", err))
	}
	app.mu.Lock()
	app.playerStarted = true
	app.serverInfo = cloneServerInfo(info)
	app.mu.Unlock()

	if app.deps.Events != nil {
		if err := app.deps.Events.Emit(serverInfoEvent, info); err != nil {
			return app.failLocked(ctx, fmt.Errorf("publish player server status to desktop bridge: %w", err))
		}
	}

	app.setPhase(ctx, "desktop-loading")
	if app.deps.Desktop != nil {
		if err := app.deps.Desktop.Ready(ctx); err != nil {
			return app.failLocked(ctx, fmt.Errorf("make desktop ready: %w", err))
		}
		app.mu.Lock()
		app.desktopReady = true
		app.mu.Unlock()
	}
	app.setPhase(ctx, "ready-local")

	if app.deps.TunnelEnabled {
		app.startTunnelLocked(ctx, info)
	}
	return nil
}

// Shutdown releases tunnel, player listener, persistence worker, then desktop.
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
	return RuntimeStatus{
		ServerInfo:        cloneServerInfoPointer(app.serverInfo),
		ClientCount:       app.clientCount,
		HackState:         clonePublicHackState(app.hackState),
		StartupError:      app.startupError,
		SaveState:         app.saveState,
		RequestedRevision: app.requestedRevision,
		SavedRevision:     app.savedRevision,
	}
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
	return result
}

// OpenSession opens and validates an existing version-1 session.
func (app *App) OpenSession() sessionservice.SessionResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SessionResult{Error: "session service is unavailable"}
	}
	result := commands.Open(app.contextSnapshot())
	app.captureSessionStatus(commands)
	return result
}

// CopyDemo creates an explicit writable copy of the bundled demo.
func (app *App) CopyDemo() sessionservice.SessionResult {
	commands, ok := app.deps.Sessions.(sessionCommands)
	if !ok {
		return sessionservice.SessionResult{Error: "session service is unavailable"}
	}
	result := commands.CopyDemo(app.contextSnapshot())
	app.captureSessionStatus(commands)
	return result
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
	return result
}

// SetLiveTerminal validates privileged input before installing canonical live
// state and emits only the public hacking projection.
func (app *App) SetLiveTerminal(payload LiveTerminalPayload) CommandResult {
	if app.deps.Live == nil {
		return commandFailure("live service is unavailable")
	}
	if err := validateLiveTerminal(payload.TerminalID, payload.TerminalName, payload.Tree, payload.HackLevel, payload.IntroText); err != nil {
		return commandFailure(err.Error())
	}
	state := app.deps.Live.Set(payload.TerminalID, payload.TerminalName, payload.Tree, payload.HackLevel, payload.IntroText)
	if state == nil {
		return commandFailure("live terminal could not be created")
	}
	if publisher, ok := app.deps.Player.(playerPublisher); ok {
		publisher.PublishLive()
	}
	app.updateHackState(state.Hack)
	return CommandResult{OK: true}
}

// UpdateLiveTerminal validates replacement content and preserves the current
// puzzle through the live service.
func (app *App) UpdateLiveTerminal(payload LiveUpdatePayload) CommandResult {
	if app.deps.Live == nil {
		return commandFailure("live service is unavailable")
	}
	intro := ""
	if payload.IntroText != nil {
		intro = *payload.IntroText
	}
	if err := validateLiveTerminal("live-terminal", "Live Terminal", payload.Tree, 0, intro); err != nil {
		return commandFailure(err.Error())
	}
	state, ok := app.deps.Live.Update(payload.Tree, payload.IntroText)
	if !ok || state == nil {
		return commandFailure("no terminal is live")
	}
	if publisher, ok := app.deps.Player.(playerPublisher); ok {
		publisher.PublishUpdate()
	}
	app.updateHackState(state.Hack)
	return CommandResult{OK: true}
}

// ClearLiveTerminal clears process-only live state and its public status.
func (app *App) ClearLiveTerminal() CommandResult {
	if app.deps.Live == nil {
		return commandFailure("live service is unavailable")
	}
	app.deps.Live.Clear()
	if publisher, ok := app.deps.Player.(playerPublisher); ok {
		publisher.PublishClear()
	}
	app.updateHackState(nil)
	return CommandResult{OK: true}
}

// ForceHackSuccess completes an eligible puzzle and publishes the sanitized
// result.
func (app *App) ForceHackSuccess() CommandResult {
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
	return CommandResult{OK: true}
}

// OpenURL validates immediately before crossing the system-browser boundary.
func (app *App) OpenURL(rawURL string) CommandResult {
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
	return CommandResult{OK: true}
}

func (app *App) startup(ctx context.Context) {
	if setter, ok := app.deps.Events.(interface{ SetContext(context.Context) }); ok {
		setter.SetContext(ctx)
	}
	_ = app.Start(ctx)
}

func (app *App) domReady(ctx context.Context) {
	app.mu.Lock()
	app.ctx = ctx
	info := cloneServerInfoPointer(app.serverInfo)
	clientCount := app.clientCount
	hackState := clonePublicHackState(app.hackState)
	app.mu.Unlock()
	if app.deps.Events != nil {
		if info != nil {
			_ = app.deps.Events.Emit(serverInfoEvent, *info)
		}
		_ = app.deps.Events.Emit(clientCountEvent, clientCount)
		_ = app.deps.Events.Emit(hackStateEvent, hackState)
	}
}

func (app *App) shutdown(ctx context.Context) {
	_ = app.Shutdown(ctx)
}

func (app *App) startTunnelLocked(ctx context.Context, local domain.ServerInfo) {
	if app.deps.Tunnel == nil {
		app.recordTunnelFailure("public tunnel is enabled but not configured")
		return
	}
	app.setPhase(ctx, "starting-tunnel")
	public, err := app.deps.Tunnel.Start(ctx)
	if err != nil {
		// Tunnel implementations own detailed diagnostics and credential
		// redaction. The desktop boundary deliberately emits a fixed actionable
		// message so an unexpected dependency error can never disclose secrets.
		app.recordTunnelFailure(tunnelStartupFailureMessage)
		return
	}
	publicURL, valid := protectedPublicURL(public.URL)
	if !valid {
		// Start returning success means a process may have been acquired. Stop it
		// before reporting the invalid address and falling back to local mode.
		_ = app.deps.Tunnel.Stop(ctx)
		app.recordTunnelFailure(tunnelAddressFailureMessage)
		return
	}
	public.URL = publicURL
	public.LocalURL = local.URL
	public.Tunnel = true
	public.TunnelError = ""
	app.mu.Lock()
	app.tunnelStarted = true
	app.serverInfo = cloneServerInfo(public)
	app.startupError = ""
	app.phase = "ready-public"
	app.mu.Unlock()
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(serverInfoEvent, public)
	}
}

func protectedPublicURL(rawURL string) (string, bool) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	return parsed.String(), true
}

func (app *App) recordTunnelFailure(message string) {
	app.mu.Lock()
	app.startupError = message
	if app.serverInfo != nil {
		app.serverInfo.TunnelError = message
	}
	app.phase = "ready-local"
	info := cloneServerInfoPointer(app.serverInfo)
	app.mu.Unlock()
	if info != nil && app.deps.Events != nil {
		_ = app.deps.Events.Emit(serverInfoEvent, *info)
	}
}

func (app *App) failLocked(ctx context.Context, cause error) error {
	app.mu.Lock()
	app.startupError = cause.Error()
	app.phase = "failed"
	app.mu.Unlock()
	cleanupErr := app.shutdownLocked(ctx, true)
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
	tunnelStarted := app.tunnelStarted
	playerStarted := app.playerStarted
	desktopReady := app.desktopReady
	sessionsOpen := app.deps.Sessions != nil && !app.sessionsClosed
	app.tunnelStarted = false
	app.playerStarted = false
	app.desktopReady = false
	app.sessionsClosed = true
	app.mu.Unlock()

	var cleanupErrors []error
	if app.deps.Live != nil {
		app.deps.Live.Clear()
		app.mu.Lock()
		app.hackState = nil
		app.mu.Unlock()
	}
	if tunnelStarted && app.deps.Tunnel != nil {
		if err := app.deps.Tunnel.Stop(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop public tunnel: %w", err))
		}
	}
	if playerStarted && app.deps.Player != nil {
		if err := app.deps.Player.Stop(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop player server: %w", err))
		}
	}
	if sessionsOpen {
		if err := app.deps.Sessions.Shutdown(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("stop session service: %w", err))
		}
	}
	if desktopReady && app.deps.Desktop != nil {
		if err := app.deps.Desktop.Close(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("close desktop runtime: %w", err))
		}
	}

	app.mu.Lock()
	app.ctx = nil
	app.serverInfo = nil
	app.stopped = true
	if preserveFailure {
		app.phase = "failed"
	} else {
		app.phase = "stopped"
	}
	app.mu.Unlock()
	return errors.Join(cleanupErrors...)
}

func (app *App) setPhase(ctx context.Context, phase string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.ctx = ctx
	app.phase = phase
	app.stopped = false
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

// updateHackState is also the player-server callback boundary for accepted
// HACK_GUESS and HACK_ADMIN actions. Only the detached public projection enters
// RuntimeStatus or crosses the desktop event bridge.
func (app *App) updateHackState(state *domain.PublicHackState) {
	clone := clonePublicHackState(state)
	app.mu.Lock()
	app.hackState = clone
	app.mu.Unlock()
	if app.deps.Events != nil {
		_ = app.deps.Events.Emit(hackStateEvent, clonePublicHackState(clone))
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
		_ = app.deps.Events.Emit(clientCountEvent, count)
	}
}

func validateLiveTerminal(id, name string, tree domain.ContentNode, hackLevel int, intro string) error {
	terminal := domain.Terminal{
		ID: id, Name: name, HackLevel: hackLevel, IntroText: intro, Root: tree,
	}
	return domain.ValidateSession(domain.Session{
		Version:   1,
		Name:      "Live Terminal",
		Terminals: []domain.Terminal{terminal},
	})
}

func commandFailure(message string) CommandResult {
	return CommandResult{Error: message}
}

func clonePublicHackState(state *domain.PublicHackState) *domain.PublicHackState {
	if state == nil {
		return nil
	}
	clone := *state
	clone.Log = append([]string(nil), state.Log...)
	clone.Columns = make([]domain.HackColumn, len(state.Columns))
	for index, column := range state.Columns {
		clone.Columns[index] = column
		clone.Columns[index].Addresses = append([]string(nil), column.Addresses...)
		clone.Columns[index].Words = append([]domain.HackWord(nil), column.Words...)
	}
	return &clone
}

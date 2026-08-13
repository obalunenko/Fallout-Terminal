// Package session owns validated, user-selected session documents and their
// ordered durable saves.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

// SaveState is the user-visible persistence state of the active session.
type SaveState string

const (
	SaveStateIdle   SaveState = "idle"
	SaveStateSaving SaveState = "saving"
	SaveStateSaved  SaveState = "saved"
	SaveStateFailed SaveState = "failed"
)

// Locations separates user session suggestions, the immutable bundled demo,
// and application-owned metadata. Session content is never written to
// ApplicationSupport.
type Locations struct {
	DocumentsDefault   string
	BundledDemo        string
	ApplicationSupport string
}

// Dialog is the native file-dialog boundary used by the session service. An
// empty path with no error means the user canceled the dialog.
type Dialog interface {
	OpenFile(defaultPath string) (string, error)
	SaveFile(defaultPath string) (string, error)
}

// Store is the durable storage boundary used by the session service.
type Store interface {
	Read(path string) ([]byte, error)
	WriteAtomic(path string, data []byte) error
	CopyAtomic(source, destination string) error
}

// SessionResult is returned by create, open, and demo-copy commands.
type SessionResult struct {
	OK       bool            `json:"ok"`
	Canceled bool            `json:"canceled"`
	Error    string          `json:"error,omitempty"`
	FilePath string          `json:"filePath,omitempty"`
	Session  *domain.Session `json:"session,omitempty"`
}

// SaveResult reports both the caller's revision and the newest revision known
// to be durable when that caller completes.
type SaveResult struct {
	OK                bool   `json:"ok"`
	Error             string `json:"error,omitempty"`
	RequestedRevision uint64 `json:"requestedRevision"`
	SavedRevision     uint64 `json:"savedRevision,omitempty"`
}

// ActiveSession is an immutable snapshot of the current user-owned document.
type ActiveSession struct {
	Path              string          `json:"path,omitempty"`
	Session           *domain.Session `json:"session,omitempty"`
	RequestedRevision uint64          `json:"requestedRevision"`
	SavedRevision     uint64          `json:"savedRevision"`
	SaveState         SaveState       `json:"saveState"`
}

type savePayload struct {
	epoch    uint64
	path     string
	revision uint64
	session  domain.Session
	data     []byte
}

type saveWaiter struct {
	epoch    uint64
	revision uint64
	reply    chan SaveResult
}

// Service coordinates dialogs, validation, active-path ownership, and one
// serialized writer. The writer may coalesce queued revisions for the same
// document, but every accepted caller waits until its revision (or a newer
// one) is durable.
type Service struct {
	store     Store
	dialog    Dialog
	locations Locations

	commandMu sync.Mutex
	mu        sync.Mutex
	active    ActiveSession
	epoch     uint64
	closed    bool

	pending    []savePayload
	waiters    []saveWaiter
	durable    map[uint64]uint64
	wake       chan struct{}
	workerDone chan struct{}
}

// NewService starts an idle session service and its serialized save worker.
func NewService(store Store, dialog Dialog, locations Locations) *Service {
	service := &Service{
		store:      store,
		dialog:     dialog,
		locations:  locations,
		active:     ActiveSession{SaveState: SaveStateIdle},
		durable:    make(map[uint64]uint64),
		wake:       make(chan struct{}, 1),
		workerDone: make(chan struct{}),
	}
	go service.runSaveWorker()
	return service
}

// Create asks for an explicit destination, writes a valid starter document,
// and activates it only after the write succeeds.
func (service *Service) Create(ctx context.Context) SessionResult {
	service.commandMu.Lock()
	defer service.commandMu.Unlock()

	if err := service.available(ctx); err != nil {
		return sessionFailure("new session is unavailable")
	}
	if service.dialog == nil {
		return sessionFailure("new session dialog is unavailable")
	}

	defaultPath := filepath.Join(service.locations.DocumentsDefault, "session.json")
	target, err := service.dialog.SaveFile(defaultPath)
	if err != nil {
		return sessionFailure("could not choose a new session destination")
	}
	if target == "" {
		return SessionResult{Canceled: true}
	}
	if err := contextError(ctx); err != nil {
		return sessionFailure("new session was canceled")
	}
	target, err = service.writableTarget(target)
	if err != nil {
		return sessionFailure(err.Error())
	}

	created := starterSession(sessionNameFromPath(target))
	if err := verifySessionContract(created); err != nil {
		return sessionFailure("could not verify the new session contract")
	}
	data, err := domain.EncodeSession(created)
	if err != nil {
		return sessionFailure("could not prepare the new session")
	}
	if service.store == nil {
		return sessionFailure("session storage is unavailable")
	}
	if err := service.store.WriteAtomic(target, data); err != nil {
		return sessionFailure("could not write the new session")
	}
	return service.activate(target, created)
}

// Open asks for an explicit document and activates it only after the complete
// version-1 document has decoded and validated successfully.
func (service *Service) Open(ctx context.Context) SessionResult {
	service.commandMu.Lock()
	defer service.commandMu.Unlock()

	if err := service.available(ctx); err != nil {
		return sessionFailure("open session is unavailable")
	}
	if service.dialog == nil {
		return sessionFailure("open session dialog is unavailable")
	}

	target, err := service.dialog.OpenFile(service.locations.DocumentsDefault)
	if err != nil {
		return sessionFailure("could not choose a session to open")
	}
	if target == "" {
		return SessionResult{Canceled: true}
	}
	if err := contextError(ctx); err != nil {
		return sessionFailure("open session was canceled")
	}
	target, err = service.userDocumentPath(target)
	if err != nil {
		return sessionFailure(err.Error())
	}
	if samePath(target, service.locations.BundledDemo) {
		return sessionFailure("the bundled demo is read-only; copy it before editing")
	}
	if service.store == nil {
		return sessionFailure("session storage is unavailable")
	}

	data, err := service.store.Read(target)
	if err != nil {
		return sessionFailure("could not read the selected session")
	}
	opened, err := domain.DecodeSession(data)
	if err != nil {
		return sessionFailure("the selected file is not a valid version-1 session")
	}
	if err := verifySessionContract(opened); err != nil {
		return sessionFailure("the selected file does not satisfy the session contract")
	}
	return service.activate(target, opened)
}

// CopyDemo validates the immutable bundled sample, asks for a distinct
// destination, and activates only the writable copy.
func (service *Service) CopyDemo(ctx context.Context) SessionResult {
	service.commandMu.Lock()
	defer service.commandMu.Unlock()

	if err := service.available(ctx); err != nil {
		return sessionFailure("demo copy is unavailable")
	}
	if service.dialog == nil {
		return sessionFailure("demo copy dialog is unavailable")
	}

	defaultPath := filepath.Join(service.locations.DocumentsDefault, "demo.json")
	destination, err := service.dialog.SaveFile(defaultPath)
	if err != nil {
		return sessionFailure("could not choose a demo destination")
	}
	if destination == "" {
		return SessionResult{Canceled: true}
	}
	if err := contextError(ctx); err != nil {
		return sessionFailure("demo copy was canceled")
	}
	destination, err = service.writableTarget(destination)
	if err != nil {
		return sessionFailure(err.Error())
	}
	if service.store == nil {
		return sessionFailure("session storage is unavailable")
	}

	data, err := service.store.Read(service.locations.BundledDemo)
	if err != nil {
		return sessionFailure("could not read the bundled demo")
	}
	demo, err := domain.DecodeSession(data)
	if err != nil {
		return sessionFailure("the bundled demo is not a valid version-1 session")
	}
	if err := verifySessionContract(demo); err != nil {
		return sessionFailure("the bundled demo does not satisfy the session contract")
	}
	if err := service.store.CopyAtomic(service.locations.BundledDemo, destination); err != nil {
		return sessionFailure("could not create a writable demo copy")
	}
	return service.activate(destination, demo)
}

// Save validates and accepts a monotonically increasing revision, then waits
// for the serialized writer to make that revision or a newer one durable.
func (service *Service) Save(ctx context.Context, session domain.Session, revision uint64) SaveResult {
	if err := contextError(ctx); err != nil {
		return saveFailure(revision, 0, "save was canceled")
	}
	if revision == 0 {
		return saveFailure(revision, 0, "save revision must be greater than zero")
	}
	if err := verifySessionContract(session); err != nil {
		return saveFailure(revision, 0, "session is invalid and was not saved")
	}

	data, err := domain.EncodeSession(session)
	if err != nil {
		return saveFailure(revision, 0, "session is invalid and was not saved")
	}
	accepted, err := domain.DecodeSession(data)
	if err != nil {
		return saveFailure(revision, 0, "session is invalid and was not saved")
	}

	reply := make(chan SaveResult, 1)
	service.mu.Lock()
	if service.closed {
		service.mu.Unlock()
		return saveFailure(revision, 0, "session service is shut down")
	}
	if service.active.Path == "" || service.active.Session == nil {
		saved := service.active.SavedRevision
		service.mu.Unlock()
		return saveFailure(revision, saved, "there is no active session path")
	}
	if samePath(service.active.Path, service.locations.BundledDemo) {
		saved := service.active.SavedRevision
		service.mu.Unlock()
		return saveFailure(revision, saved, "the bundled demo cannot be saved in place")
	}

	epoch := service.epoch
	if saved := service.durable[epoch]; saved >= revision {
		service.mu.Unlock()
		return SaveResult{OK: true, RequestedRevision: revision, SavedRevision: saved}
	}

	if revision > service.active.RequestedRevision {
		service.active.RequestedRevision = revision
		service.active.Session = sessionPointer(accepted)
		service.active.SaveState = SaveStateSaving
		service.pending = append(service.pending, savePayload{
			epoch:    epoch,
			path:     service.active.Path,
			revision: revision,
			session:  accepted,
			data:     append([]byte(nil), data...),
		})
	}
	service.waiters = append(service.waiters, saveWaiter{epoch: epoch, revision: revision, reply: reply})
	service.signalWorkerLocked()
	service.mu.Unlock()

	return <-reply
}

// AssociatePlayerConfig atomically saves a normalized relative reference and
// updates the active session only after the durable replacement succeeds.
func (service *Service) AssociatePlayerConfig(ctx context.Context, playerConfigPath string) SessionResult {
	service.commandMu.Lock()
	defer service.commandMu.Unlock()

	if err := contextError(ctx); err != nil {
		return sessionFailure("player config association was canceled")
	}
	service.mu.Lock()
	if service.closed || service.active.Path == "" || service.active.Session == nil {
		service.mu.Unlock()
		return sessionFailure("there is no active session")
	}
	active := cloneActive(service.active)
	epoch := service.epoch
	service.mu.Unlock()

	playerConfigPath = filepath.Clean(strings.TrimSpace(playerConfigPath))
	if !filepath.IsAbs(playerConfigPath) || playerConfigPath == string(filepath.Separator) {
		return sessionFailure("player config path must be absolute")
	}
	reference, err := filepath.Rel(filepath.Dir(active.Path), playerConfigPath)
	if err != nil || filepath.IsAbs(reference) || filepath.Clean(reference) == "." {
		return sessionFailure("could not create a relative player config reference")
	}
	candidate := cloneSession(*active.Session)
	candidate.PlayerConfig = filepath.Clean(reference)
	if err := verifySessionContract(candidate); err != nil {
		return sessionFailure("could not verify the player config association")
	}
	data, err := domain.EncodeSession(candidate)
	if err != nil {
		return sessionFailure("could not prepare the player config association")
	}
	if service.store == nil || service.store.WriteAtomic(active.Path, data) != nil {
		return sessionFailure("could not save the player config association")
	}

	service.mu.Lock()
	defer service.mu.Unlock()
	if service.closed || service.epoch != epoch || service.active.Path != active.Path {
		return sessionFailure("active session changed while saving the player config association")
	}
	revision := service.active.RequestedRevision + 1
	if revision <= service.active.SavedRevision {
		revision = service.active.SavedRevision + 1
	}
	service.active.Session = sessionPointer(candidate)
	service.active.RequestedRevision = revision
	service.active.SavedRevision = revision
	service.active.SaveState = SaveStateSaved
	service.durable[epoch] = revision
	return SessionResult{OK: true, FilePath: active.Path, Session: sessionPointer(candidate)}
}

// Snapshot returns detached state safe for concurrent callers to inspect.
func (service *Service) Snapshot() ActiveSession {
	if service == nil {
		return ActiveSession{SaveState: SaveStateIdle}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	return cloneActive(service.active)
}

// Shutdown rejects new work, drains every already accepted save, and is safe
// to call repeatedly or concurrently.
func (service *Service) Shutdown(ctx context.Context) error {
	if service == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	service.mu.Lock()
	service.closed = true
	service.signalWorkerLocked()
	done := service.workerDone
	service.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (service *Service) runSaveWorker() {
	defer close(service.workerDone)
	for {
		<-service.wake
		for {
			payload, ok, stop := service.nextPayload()
			if stop {
				return
			}
			if !ok {
				break
			}

			var writeErr error
			if service.store == nil {
				writeErr = errors.New("session storage is unavailable")
			} else {
				writeErr = service.store.WriteAtomic(payload.path, payload.data)
			}
			service.finishPayload(payload, writeErr)
		}
	}
}

// nextPayload takes the newest queued revision for the oldest document epoch.
// This preserves path ordering while collapsing bursts of edits for one file.
func (service *Service) nextPayload() (savePayload, bool, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if len(service.pending) == 0 {
		return savePayload{}, false, service.closed
	}

	selected := service.pending[0]
	consumed := 1
	for consumed < len(service.pending) && service.pending[consumed].epoch == selected.epoch {
		selected = service.pending[consumed]
		consumed++
	}
	service.pending = append([]savePayload(nil), service.pending[consumed:]...)
	if service.epoch == selected.epoch {
		service.active.SaveState = SaveStateSaving
	}
	return selected, true, false
}

func (service *Service) finishPayload(payload savePayload, writeErr error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if writeErr == nil && payload.revision > service.durable[payload.epoch] {
		service.durable[payload.epoch] = payload.revision
	}
	durableRevision := service.durable[payload.epoch]
	if service.epoch == payload.epoch {
		service.active.SavedRevision = durableRevision
		if writeErr == nil {
			service.active.Session = sessionPointer(payload.session)
			if service.active.RequestedRevision <= durableRevision {
				service.active.SaveState = SaveStateSaved
			} else {
				service.active.SaveState = SaveStateSaving
			}
		} else {
			service.active.SaveState = SaveStateFailed
		}
	}

	remaining := service.waiters[:0]
	for _, waiter := range service.waiters {
		if waiter.epoch != payload.epoch || waiter.revision > payload.revision {
			remaining = append(remaining, waiter)
			continue
		}
		if durableRevision >= waiter.revision {
			waiter.reply <- SaveResult{
				OK:                true,
				RequestedRevision: waiter.revision,
				SavedRevision:     durableRevision,
			}
		} else {
			waiter.reply <- saveFailure(waiter.revision, durableRevision, "could not save the session")
		}
	}
	service.waiters = remaining
}

func (service *Service) activate(path string, session domain.Session) SessionResult {
	copy := cloneSession(session)
	service.mu.Lock()
	if service.closed {
		service.mu.Unlock()
		return sessionFailure("session service is shut down")
	}
	service.epoch++
	service.active = ActiveSession{
		Path:      path,
		Session:   sessionPointer(copy),
		SaveState: SaveStateSaved,
	}
	service.durable[service.epoch] = 0
	service.mu.Unlock()
	return SessionResult{OK: true, FilePath: path, Session: sessionPointer(copy)}
}

func (service *Service) available(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.closed {
		return errors.New("session service is shut down")
	}
	return nil
}

func (service *Service) writableTarget(path string) (string, error) {
	path, err := service.userDocumentPath(path)
	if err != nil {
		return "", err
	}
	if samePath(path, service.locations.BundledDemo) {
		return "", errors.New("the bundled demo is read-only; choose another destination")
	}
	return path, nil
}

func (service *Service) userDocumentPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("session path is empty")
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", errors.New("session path must be absolute")
	}
	if cleaned == string(filepath.Separator) {
		return "", errors.New("session path does not name a file")
	}
	return cleaned, nil
}

func (service *Service) signalWorkerLocked() {
	select {
	case service.wake <- struct{}{}:
	default:
	}
}

func starterSession(name string) domain.Session {
	return domain.Session{
		Version: 1,
		Name:    name,
		Terminals: []domain.Terminal{{
			ID:        "terminal-1",
			Name:      "Terminal 1",
			HackLevel: 0,
			IntroText: "",
			Root: domain.ContentNode{
				ID:       "root",
				Type:     domain.NodeFolder,
				Name:     "ROOT",
				Children: []domain.ContentNode{},
			},
		}},
	}
}

func sessionNameFromPath(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSpace(strings.TrimSuffix(base, filepath.Ext(base)))
	if name == "" {
		return "New Session"
	}
	return name
}

func samePath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func sessionFailure(message string) SessionResult {
	return SessionResult{Error: message}
}

func saveFailure(requested, saved uint64, message string) SaveResult {
	return SaveResult{Error: message, RequestedRevision: requested, SavedRevision: saved}
}

func sessionPointer(session domain.Session) *domain.Session {
	copy := cloneSession(session)
	return &copy
}

func cloneActive(active ActiveSession) ActiveSession {
	copy := active
	if active.Session != nil {
		copy.Session = sessionPointer(*active.Session)
	}
	return copy
}

func cloneSession(session domain.Session) domain.Session {
	copy := session
	copy.Extra = cloneExtra(session.Extra)
	copy.Terminals = make([]domain.Terminal, len(session.Terminals))
	for index, terminal := range session.Terminals {
		copy.Terminals[index] = terminal
		copy.Terminals[index].Extra = cloneExtra(terminal.Extra)
		copy.Terminals[index].Root = cloneNode(terminal.Root)
	}
	return copy
}

func cloneNode(node domain.ContentNode) domain.ContentNode {
	copy := node
	copy.Extra = cloneExtra(node.Extra)
	if node.Children != nil {
		copy.Children = make([]domain.ContentNode, len(node.Children))
		for index := range node.Children {
			copy.Children[index] = cloneNode(node.Children[index])
		}
	}
	return copy
}

func cloneExtra(extra map[string]json.RawMessage) map[string]json.RawMessage {
	if extra == nil {
		return nil
	}
	copy := make(map[string]json.RawMessage, len(extra))
	for key, value := range extra {
		copy[key] = append([]byte(nil), value...)
	}
	return copy
}

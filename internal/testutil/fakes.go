// Package testutil contains deterministic fakes shared by Go tests.
package testutil

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FakeClock is a manually advanced clock. Sleep advances time immediately and
// records the requested duration, so tests never wait on wall-clock time.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	sleeps []time.Duration
	timers []fakeClockTimer
}

// NewFakeClock returns a clock initialized to now.
func NewFakeClock(now time.Time) *FakeClock {
	return &FakeClock{now: now}
}

// Now returns the clock's current time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.fireTimersLocked()
}

// Sleep records d and advances the clock by that duration without blocking.
func (c *FakeClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sleeps = append(c.sleeps, d)
	c.now = c.now.Add(d)
	c.fireTimersLocked()
}

// SleepCalls returns a copy of the durations passed to Sleep.
func (c *FakeClock) SleepCalls() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Duration(nil), c.sleeps...)
}

// FileWrite records one WriteFile call.
type FileWrite struct {
	Path string
	Data []byte
	Perm fs.FileMode
}

// FileRename records one Rename call.
type FileRename struct {
	OldPath string
	NewPath string
}

// DirectoryCreate records one MkdirAll call.
type DirectoryCreate struct {
	Path string
	Perm fs.FileMode
}

// FakeFileSystem is a small in-memory filesystem fake for persistence tests.
// Per-operation errors may be injected by exact path.
type FakeFileSystem struct {
	mu sync.Mutex

	files map[string][]byte
	dirs  map[string]fs.FileMode
	modes map[string]fs.FileMode
	temps int

	ReadErrors   map[string]error
	WriteErrors  map[string]error
	RenameErrors map[string]error
	RemoveErrors map[string]error
	MkdirErrors  map[string]error

	reads   []string
	writes  []FileWrite
	renames []FileRename
	removes []string
	mkdirs  []DirectoryCreate
}

// NewFakeFileSystem returns an empty in-memory filesystem.
func NewFakeFileSystem() *FakeFileSystem {
	return &FakeFileSystem{
		files:        make(map[string][]byte),
		dirs:         make(map[string]fs.FileMode),
		modes:        make(map[string]fs.FileMode),
		ReadErrors:   make(map[string]error),
		WriteErrors:  make(map[string]error),
		RenameErrors: make(map[string]error),
		RemoveErrors: make(map[string]error),
		MkdirErrors:  make(map[string]error),
	}
}

// SeedFile places data at path without recording a filesystem operation.
func (f *FakeFileSystem) SeedFile(path string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.files[path] = append([]byte(nil), data...)
}

// ReadFile returns a defensive copy of the file content at path.
func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.reads = append(f.reads, path)
	if err := f.ReadErrors[path]; err != nil {
		return nil, err
	}
	data, ok := f.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
	}
	return append([]byte(nil), data...), nil
}

// WriteFile stores a defensive copy of data at path.
func (f *FakeFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.writes = append(f.writes, FileWrite{Path: path, Data: append([]byte(nil), data...), Perm: perm})
	if err := f.WriteErrors[path]; err != nil {
		return err
	}
	f.files[path] = append([]byte(nil), data...)
	return nil
}

// Rename moves a file from oldPath to newPath.
func (f *FakeFileSystem) Rename(oldPath, newPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.renames = append(f.renames, FileRename{OldPath: oldPath, NewPath: newPath})
	if err := f.RenameErrors[oldPath]; err != nil {
		return err
	}
	data, ok := f.files[oldPath]
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldPath, Err: fs.ErrNotExist}
	}
	f.files[newPath] = data
	delete(f.files, oldPath)
	if mode, ok := f.modes[oldPath]; ok {
		f.modes[newPath] = mode
		delete(f.modes, oldPath)
	}
	return nil
}

// Remove deletes a file or an empty directory.
func (f *FakeFileSystem) Remove(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.removes = append(f.removes, path)
	if err := f.RemoveErrors[path]; err != nil {
		return err
	}
	if _, ok := f.files[path]; ok {
		delete(f.files, path)
		return nil
	}
	if _, ok := f.dirs[path]; ok {
		delete(f.dirs, path)
		return nil
	}
	return &fs.PathError{Op: "remove", Path: path, Err: fs.ErrNotExist}
}

// MkdirAll records and creates path and its parents.
func (f *FakeFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	f.mkdirs = append(f.mkdirs, DirectoryCreate{Path: path, Perm: perm})
	if err := f.MkdirErrors[path]; err != nil {
		return err
	}
	for current := filepath.Clean(path); current != "."; current = filepath.Dir(current) {
		f.dirs[current] = perm
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return nil
}

// File returns a defensive copy of path's content and whether it exists.
func (f *FakeFileSystem) File(path string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	data, ok := f.files[path]
	return append([]byte(nil), data...), ok
}

// Mode returns the most recently applied mode for a fake file or directory.
func (f *FakeFileSystem) Mode(path string) (fs.FileMode, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensureMaps()
	mode, ok := f.modes[path]
	if !ok {
		mode, ok = f.dirs[path]
	}
	return mode, ok
}

// Paths returns all fake file paths in stable order.
func (f *FakeFileSystem) Paths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	paths := make([]string, 0, len(f.files))
	for path := range f.files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// ReadCalls returns paths passed to ReadFile in call order.
func (f *FakeFileSystem) ReadCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.reads...)
}

// WriteCalls returns WriteFile calls in call order.
func (f *FakeFileSystem) WriteCalls() []FileWrite {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]FileWrite, len(f.writes))
	for i, call := range f.writes {
		result[i] = call
		result[i].Data = append([]byte(nil), call.Data...)
	}
	return result
}

// RenameCalls returns Rename calls in call order.
func (f *FakeFileSystem) RenameCalls() []FileRename {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]FileRename(nil), f.renames...)
}

// RemoveCalls returns paths passed to Remove in call order.
func (f *FakeFileSystem) RemoveCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.removes...)
}

// MkdirCalls returns MkdirAll calls in call order.
func (f *FakeFileSystem) MkdirCalls() []DirectoryCreate {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]DirectoryCreate(nil), f.mkdirs...)
}

func (f *FakeFileSystem) ensureMaps() {
	if f.files == nil {
		f.files = make(map[string][]byte)
	}
	if f.dirs == nil {
		f.dirs = make(map[string]fs.FileMode)
	}
	if f.modes == nil {
		f.modes = make(map[string]fs.FileMode)
	}
	if f.ReadErrors == nil {
		f.ReadErrors = make(map[string]error)
	}
	if f.WriteErrors == nil {
		f.WriteErrors = make(map[string]error)
	}
	if f.RenameErrors == nil {
		f.RenameErrors = make(map[string]error)
	}
	if f.RemoveErrors == nil {
		f.RemoveErrors = make(map[string]error)
	}
	if f.MkdirErrors == nil {
		f.MkdirErrors = make(map[string]error)
	}
}

// DialogCall records a native file-dialog request.
type DialogCall struct {
	Kind        string
	DefaultPath string
}

// FakeDialog returns configured open and save selections.
type FakeDialog struct {
	mu sync.Mutex

	OpenResult string
	OpenErr    error
	SaveResult string
	SaveErr    error

	calls []DialogCall
}

// OpenFile records an open-dialog request and returns its configured result.
func (d *FakeDialog) OpenFile(defaultPath string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, DialogCall{Kind: "open", DefaultPath: defaultPath})
	return d.OpenResult, d.OpenErr
}

// SaveFile records a save-dialog request and returns its configured result.
func (d *FakeDialog) SaveFile(defaultPath string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls = append(d.calls, DialogCall{Kind: "save", DefaultPath: defaultPath})
	return d.SaveResult, d.SaveErr
}

// Calls returns dialog calls in call order.
func (d *FakeDialog) Calls() []DialogCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]DialogCall(nil), d.calls...)
}

// FakeBrowser records external URLs and returns an optional configured error.
type FakeBrowser struct {
	mu sync.Mutex

	Err  error
	urls []string
}

// OpenURL records rawURL.
func (b *FakeBrowser) OpenURL(rawURL string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.urls = append(b.urls, rawURL)
	return b.Err
}

// URLs returns opened URLs in call order.
func (b *FakeBrowser) URLs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.urls...)
}

// ProcessStart records one process start request.
type ProcessStart struct {
	Name string
	Args []string
}

// FakeProcessRunner returns a configured FakeProcess and records start calls.
type FakeProcessRunner struct {
	mu sync.Mutex

	Process  *FakeProcess
	StartErr error
	starts   []ProcessStart
}

// Start records a process request. It returns a fresh FakeProcess when Process
// has not been configured.
func (r *FakeProcessRunner) Start(_ context.Context, name string, args ...string) (*FakeProcess, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts = append(r.starts, ProcessStart{Name: name, Args: append([]string(nil), args...)})
	if r.StartErr != nil {
		return nil, r.StartErr
	}
	if r.Process == nil {
		r.Process = NewFakeProcess()
	}
	return r.Process, nil
}

// StartCalls returns process starts in call order.
func (r *FakeProcessRunner) StartCalls() []ProcessStart {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]ProcessStart, len(r.starts))
	for i, call := range r.starts {
		result[i] = ProcessStart{Name: call.Name, Args: append([]string(nil), call.Args...)}
	}
	return result
}

// FakeProcess is a controllable process handle with deterministic Wait, Signal,
// and Kill behavior.
type FakeProcess struct {
	mu sync.Mutex

	WaitErr   error
	SignalErr error
	KillErr   error
	waited    int
	signals   []os.Signal
	killed    int
}

// NewFakeProcess returns an idle fake process.
func NewFakeProcess() *FakeProcess {
	return &FakeProcess{}
}

// Wait records the call and returns WaitErr immediately.
func (p *FakeProcess) Wait() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.waited++
	return p.WaitErr
}

// Signal records signal and returns SignalErr.
func (p *FakeProcess) Signal(signal os.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.signals = append(p.signals, signal)
	return p.SignalErr
}

// Kill records the call and returns KillErr.
func (p *FakeProcess) Kill() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.killed++
	return p.KillErr
}

// WaitCalls returns the number of Wait calls.
func (p *FakeProcess) WaitCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waited
}

// Signals returns signals in call order.
func (p *FakeProcess) Signals() []os.Signal {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]os.Signal(nil), p.signals...)
}

// KillCalls returns the number of Kill calls.
func (p *FakeProcess) KillCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.killed
}

// FakePlayerServer is a typed lifecycle fake. Info may be instantiated with the
// production server-info type once that type exists.
type FakePlayerServer[Info any] struct {
	mu sync.Mutex

	Info     Info
	StartErr error
	StopErr  error
	starts   int
	stops    int
}

// Start records startup and returns the configured server information.
func (s *FakePlayerServer[Info]) Start(_ context.Context) (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
	return s.Info, s.StartErr
}

// Stop records shutdown and returns the configured error.
func (s *FakePlayerServer[Info]) Stop(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stops++
	return s.StopErr
}

// StartCalls returns the number of startup attempts.
func (s *FakePlayerServer[Info]) StartCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// StopCalls returns the number of shutdown attempts.
func (s *FakePlayerServer[Info]) StopCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stops
}

// FakeTunnel is a typed lifecycle fake. Request and Info may be instantiated
// with production configuration and status types once those types exist.
type FakeTunnel[Request, Info any] struct {
	mu sync.Mutex

	Info      Info
	StartErr  error
	StopErr   error
	requests  []Request
	stopCalls int
}

// Start records request and returns the configured tunnel information.
func (t *FakeTunnel[Request, Info]) Start(_ context.Context, request Request) (Info, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requests = append(t.requests, request)
	return t.Info, t.StartErr
}

// Stop records shutdown and returns the configured error.
func (t *FakeTunnel[Request, Info]) Stop(_ context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopCalls++
	return t.StopErr
}

// StartRequests returns tunnel startup requests in call order.
func (t *FakeTunnel[Request, Info]) StartRequests() []Request {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Request(nil), t.requests...)
}

// StopCalls returns the number of shutdown attempts.
func (t *FakeTunnel[Request, Info]) StopCalls() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stopCalls
}

// FakeOpaqueIDSource returns a caller-supplied sequence of opaque identifiers.
// It is safe for concurrent use and panics on exhaustion so an unexpected ID
// allocation cannot silently make a concurrency test nondeterministic.
type FakeOpaqueIDSource struct {
	mu sync.Mutex

	values []string
	next   int
}

// FakeRandom returns a caller-supplied deterministic integer sequence and
// records every limit so rejected paths can prove they consumed no randomness.
type FakeRandom struct {
	mu sync.Mutex

	values []int
	next   int
	limits []int
}

// NewFakeRandom returns a deterministic random source.
func NewFakeRandom(values ...int) *FakeRandom {
	return &FakeRandom{values: append([]int(nil), values...)}
}

// Intn returns the next configured value reduced to limit.
func (r *FakeRandom) Intn(limit int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limits = append(r.limits, limit)
	if limit <= 1 {
		return 0
	}
	if r.next >= len(r.values) {
		panic("testutil: fake random source exhausted")
	}
	value := r.values[r.next]
	r.next++
	if value < 0 {
		value = -value
	}
	return value % limit
}

// Calls returns the number of random draws.
func (r *FakeRandom) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.limits)
}

// Limits returns each requested random bound in order.
func (r *FakeRandom) Limits() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.limits...)
}

// FakeCanonicalInvoker records structurally valid requests that reached an
// application adapter and returns a configured result.
type FakeCanonicalInvoker[Request, Response any] struct {
	mu sync.Mutex

	Response Response
	Err      error
	requests []Request
}

// Invoke records request and returns the configured response.
func (s *FakeCanonicalInvoker[Request, Response]) Invoke(_ context.Context, request Request) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
	return s.Response, s.Err
}

// Requests returns all canonical invocations in order.
func (s *FakeCanonicalInvoker[Request, Response]) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Request(nil), s.requests...)
}

// FakeStreamSink records detached stream values and can deterministically
// reject an offer after a configured count.
type FakeStreamSink[Message any] struct {
	mu sync.Mutex

	RejectAfter int
	values      []Message
}

// Offer records value unless the configured bound has been reached.
func (s *FakeStreamSink[Message]) Offer(value Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.RejectAfter > 0 && len(s.values) >= s.RejectAfter {
		return false
	}
	s.values = append(s.values, value)
	return true
}

// Values returns all accepted stream values in order.
func (s *FakeStreamSink[Message]) Values() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.values...)
}

// NewFakeOpaqueIDSource returns an ID source that yields values in order.
func NewFakeOpaqueIDSource(values ...string) *FakeOpaqueIDSource {
	return &FakeOpaqueIDSource{values: append([]string(nil), values...)}
}

// Next returns the next configured opaque identifier.
func (s *FakeOpaqueIDSource) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next >= len(s.values) {
		panic("testutil: fake opaque ID source exhausted")
	}
	value := s.values[s.next]
	s.next++
	return value
}

// Calls returns the number of identifiers issued so far.
func (s *FakeOpaqueIDSource) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.next
}

// Remaining returns the number of configured identifiers not yet issued.
func (s *FakeOpaqueIDSource) Remaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.values) - s.next
}

// OrderedEffect records one effect and its one-based enqueue sequence.
type OrderedEffect[Effect any] struct {
	Sequence int
	Value    Effect
}

// FakeOrderedEffectSink serializes and records transport-neutral effects in
// enqueue order. The fake deliberately retains each value exactly as received:
// tests can mutate a producer or recorded projection afterward to expose an
// aliasing bug in the production boundary.
type FakeOrderedEffectSink[Effect any] struct {
	mu sync.Mutex

	effects []OrderedEffect[Effect]
	notify  chan struct{}
}

// NewFakeOrderedEffectSink returns an empty ordered effect recorder.
func NewFakeOrderedEffectSink[Effect any]() *FakeOrderedEffectSink[Effect] {
	return &FakeOrderedEffectSink[Effect]{notify: make(chan struct{}, 1)}
}

// Enqueue records effect synchronously in total call order.
func (s *FakeOrderedEffectSink[Effect]) Enqueue(effect Effect) {
	s.mu.Lock()
	s.ensureNotify()
	s.effects = append(s.effects, OrderedEffect[Effect]{
		Sequence: len(s.effects) + 1,
		Value:    effect,
	})
	notify := s.notify
	s.mu.Unlock()

	select {
	case notify <- struct{}{}:
	default:
	}
}

// Effects returns a copy of all recorded effects in enqueue order.
func (s *FakeOrderedEffectSink[Effect]) Effects() []OrderedEffect[Effect] {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]OrderedEffect[Effect](nil), s.effects...)
}

// Values returns recorded effect values in enqueue order.
func (s *FakeOrderedEffectSink[Effect]) Values() []Effect {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := make([]Effect, len(s.effects))
	for index, effect := range s.effects {
		values[index] = effect.Value
	}
	return values
}

// Calls returns the number of enqueued effects.
func (s *FakeOrderedEffectSink[Effect]) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.effects)
}

// WaitForCount waits until at least count effects have been enqueued or ctx is
// cancelled. It avoids polling and wall-clock sleeps in concurrent tests.
func (s *FakeOrderedEffectSink[Effect]) WaitForCount(ctx context.Context, count int) bool {
	if count <= 0 {
		return true
	}
	for {
		s.mu.Lock()
		s.ensureNotify()
		if len(s.effects) >= count {
			s.mu.Unlock()
			return true
		}
		notify := s.notify
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return false
		case <-notify:
		}
	}
}

func (s *FakeOrderedEffectSink[Effect]) ensureNotify() {
	if s.notify == nil {
		s.notify = make(chan struct{}, 1)
	}
}

// Event records one emitted event.
type Event struct {
	Name    string
	Payload any
}

// FakeEventSink records emitted application events.
type FakeEventSink struct {
	mu sync.Mutex

	Err    error
	events []Event
}

// Emit records an event and returns the configured error.
func (s *FakeEventSink) Emit(name string, payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, Event{Name: name, Payload: payload})
	return s.Err
}

// Events returns emitted events in call order.
func (s *FakeEventSink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}

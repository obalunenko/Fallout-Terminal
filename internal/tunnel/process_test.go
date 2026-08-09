package tunnel

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestOwnedProcessStartPassesExactSpecToRunner(t *testing.T) {
	t.Parallel()

	handle := newFakeProcessHandle()
	runner := &fakeProcessRunner{handle: handle}
	process := NewOwnedProcess(runner, ProcessOptions{})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	spec := ProcessSpec{
		Path:   "ngrok",
		Args:   []string{"http", "3690", "--log", "stdout"},
		Env:    []string{"LANG=C"},
		Dir:    "/private/tmp/tunnel-policy",
		Stdout: stdout,
		Stderr: stderr,
	}
	ctx := context.WithValue(context.Background(), processContextKey{}, "start-context")

	if err := process.Start(ctx, spec); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	call := runner.singleCall(t)
	if call.contextValue != "start-context" {
		t.Fatalf("runner context value = %v", call.contextValue)
	}
	if call.spec.Path != spec.Path || call.spec.Dir != spec.Dir ||
		!reflect.DeepEqual(call.spec.Args, spec.Args) || !reflect.DeepEqual(call.spec.Env, spec.Env) ||
		call.spec.Stdout != stdout || call.spec.Stderr != stderr {
		t.Fatalf("runner spec = %#v, want %#v", call.spec, spec)
	}
	if err := process.Start(ctx, spec); err == nil {
		t.Fatal("second Start() succeeded while the child was running")
	}

	handle.finish(nil)
	awaitDone(t, process.Done())
}

func TestOwnedProcessStartFailureLeavesShutdownSafe(t *testing.T) {
	t.Parallel()

	startErr := errors.New("binary not found")
	runner := &fakeProcessRunner{startErr: startErr}
	process := NewOwnedProcess(runner, ProcessOptions{})

	if err := process.Start(context.Background(), ProcessSpec{Path: "missing-ngrok"}); !errors.Is(err, startErr) {
		t.Fatalf("Start() error = %v, want %v", err, startErr)
	}
	if err := process.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() after failed start = %v", err)
	}
	if err := process.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop() after failed start = %v", err)
	}
}

func TestOwnedProcessGracefulStopTerminatesAndWaits(t *testing.T) {
	t.Parallel()

	handle := newFakeProcessHandle()
	handle.exitOnTerminate = true
	runner := &fakeProcessRunner{handle: handle}
	process := NewOwnedProcess(runner, ProcessOptions{GracePeriod: time.Second})
	if err := process.Start(context.Background(), ProcessSpec{Path: "ngrok"}); err != nil {
		t.Fatal(err)
	}

	if err := process.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if terminate, kill, waits := handle.counts(); terminate != 1 || kill != 0 || waits != 1 {
		t.Fatalf("calls terminate=%d kill=%d wait=%d, want 1, 0, 1", terminate, kill, waits)
	}
	awaitDone(t, process.Done())
}

func TestOwnedProcessForcedStopEscalatesAfterGracePeriod(t *testing.T) {
	t.Parallel()

	handle := newFakeProcessHandle()
	handle.exitOnKill = true
	runner := &fakeProcessRunner{handle: handle}
	timer := make(chan time.Time, 1)
	process := NewOwnedProcess(runner, ProcessOptions{
		GracePeriod: 250 * time.Millisecond,
		After: func(duration time.Duration) <-chan time.Time {
			if duration != 250*time.Millisecond {
				t.Errorf("grace duration = %s", duration)
			}
			return timer
		},
	})
	if err := process.Start(context.Background(), ProcessSpec{Path: "ngrok"}); err != nil {
		t.Fatal(err)
	}

	stopped := make(chan error, 1)
	go func() { stopped <- process.Stop(context.Background()) }()
	awaitSignal(t, handle.terminateCalled, "Terminate")
	timer <- time.Now()
	awaitSignal(t, handle.killCalled, "Kill")
	if err := <-stopped; err != nil {
		t.Fatalf("forced Stop() error = %v", err)
	}
	if terminate, kill, waits := handle.counts(); terminate != 1 || kill != 1 || waits != 1 {
		t.Fatalf("calls terminate=%d kill=%d wait=%d, want 1, 1, 1", terminate, kill, waits)
	}
}

func TestOwnedProcessReportsEarlyExitWithoutSignallingAgain(t *testing.T) {
	t.Parallel()

	exitErr := errors.New("exit status 7")
	handle := newFakeProcessHandle()
	handle.finish(exitErr)
	process := NewOwnedProcess(&fakeProcessRunner{handle: handle}, ProcessOptions{})
	if err := process.Start(context.Background(), ProcessSpec{Path: "ngrok"}); err != nil {
		t.Fatal(err)
	}

	awaitDone(t, process.Done())
	if err := process.Wait(); !errors.Is(err, exitErr) {
		t.Fatalf("Wait() error = %v, want %v", err, exitErr)
	}
	if err := process.Stop(context.Background()); !errors.Is(err, exitErr) {
		t.Fatalf("Stop() after early exit = %v, want cached exit error", err)
	}
	if terminate, kill, waits := handle.counts(); terminate != 0 || kill != 0 || waits != 1 {
		t.Fatalf("calls terminate=%d kill=%d wait=%d, want 0, 0, 1", terminate, kill, waits)
	}
}

func TestOwnedProcessShutdownIsIdempotentAcrossConcurrentCallers(t *testing.T) {
	t.Parallel()

	handle := newFakeProcessHandle()
	handle.exitOnTerminate = true
	process := NewOwnedProcess(&fakeProcessRunner{handle: handle}, ProcessOptions{})
	if err := process.Start(context.Background(), ProcessSpec{Path: "ngrok"}); err != nil {
		t.Fatal(err)
	}

	const callers = 8
	errorsSeen := make(chan error, callers)
	var callersDone sync.WaitGroup
	for range callers {
		callersDone.Add(1)
		go func() {
			defer callersDone.Done()
			errorsSeen <- process.Stop(context.Background())
		}()
	}
	callersDone.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		if err != nil {
			t.Errorf("concurrent Stop() error = %v", err)
		}
	}
	if err := process.Stop(context.Background()); err != nil {
		t.Fatalf("later Stop() error = %v", err)
	}
	if terminate, kill, waits := handle.counts(); terminate != 1 || kill != 0 || waits != 1 {
		t.Fatalf("calls terminate=%d kill=%d wait=%d, want 1, 0, 1", terminate, kill, waits)
	}
}

type processContextKey struct{}

type processStartCall struct {
	contextValue any
	spec         ProcessSpec
}

type fakeProcessRunner struct {
	mu       sync.Mutex
	handle   ProcessHandle
	startErr error
	calls    []processStartCall
}

func (runner *fakeProcessRunner) Start(ctx context.Context, spec ProcessSpec) (ProcessHandle, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	runner.calls = append(runner.calls, processStartCall{contextValue: ctx.Value(processContextKey{}), spec: spec})
	return runner.handle, runner.startErr
}

func (runner *fakeProcessRunner) singleCall(t *testing.T) processStartCall {
	t.Helper()
	runner.mu.Lock()
	defer runner.mu.Unlock()
	if len(runner.calls) != 1 {
		t.Fatalf("runner calls = %d, want 1", len(runner.calls))
	}
	return runner.calls[0]
}

type fakeProcessHandle struct {
	mu sync.Mutex

	waitResult chan error
	finishOnce sync.Once

	exitOnTerminate bool
	exitOnKill      bool
	terminateErr    error
	killErr         error

	terminateCalls int
	killCalls      int
	waitCalls      int

	terminateCalled chan struct{}
	killCalled      chan struct{}
}

func newFakeProcessHandle() *fakeProcessHandle {
	return &fakeProcessHandle{
		waitResult:      make(chan error, 1),
		terminateCalled: make(chan struct{}, 1),
		killCalled:      make(chan struct{}, 1),
	}
}

func (handle *fakeProcessHandle) Wait() error {
	handle.mu.Lock()
	handle.waitCalls++
	handle.mu.Unlock()
	return <-handle.waitResult
}

func (handle *fakeProcessHandle) Terminate() error {
	handle.mu.Lock()
	handle.terminateCalls++
	err := handle.terminateErr
	exit := handle.exitOnTerminate
	handle.mu.Unlock()
	select {
	case handle.terminateCalled <- struct{}{}:
	default:
	}
	if exit {
		handle.finish(nil)
	}
	return err
}

func (handle *fakeProcessHandle) Kill() error {
	handle.mu.Lock()
	handle.killCalls++
	err := handle.killErr
	exit := handle.exitOnKill
	handle.mu.Unlock()
	select {
	case handle.killCalled <- struct{}{}:
	default:
	}
	if exit {
		handle.finish(nil)
	}
	return err
}

func (handle *fakeProcessHandle) finish(err error) {
	handle.finishOnce.Do(func() {
		handle.waitResult <- err
	})
}

func (handle *fakeProcessHandle) counts() (terminate, kill, waits int) {
	handle.mu.Lock()
	defer handle.mu.Unlock()
	return handle.terminateCalls, handle.killCalls, handle.waitCalls
}

func awaitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for process completion")
	}
}

func awaitSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

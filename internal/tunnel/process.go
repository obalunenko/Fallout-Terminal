package tunnel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const defaultProcessGracePeriod = 2 * time.Second

// ProcessOptions controls only deterministic shutdown timing. Process launch
// and signaling remain behind ProcessRunner so lifecycle tests need no child
// processes or wall-clock sleeps.
type ProcessOptions struct {
	GracePeriod time.Duration
	After       func(time.Duration) <-chan time.Time
}

// OwnedProcess serializes one child lifecycle, caches its Wait result, and
// escalates from graceful termination to forced termination exactly once.
type OwnedProcess struct {
	mu      sync.Mutex
	runner  ProcessRunner
	options ProcessOptions
	handle  ProcessHandle
	done    chan struct{}
	waitErr error
	started bool

	stopOnce sync.Once
	stopErr  error
}

func NewOwnedProcess(runner ProcessRunner, options ProcessOptions) *OwnedProcess {
	if runner == nil {
		runner = NewProcessRunner()
	}
	if options.GracePeriod <= 0 {
		options.GracePeriod = defaultProcessGracePeriod
	}
	if options.After == nil {
		options.After = time.After
	}
	done := make(chan struct{})
	close(done)
	return &OwnedProcess{runner: runner, options: options, done: done}
}

func (process *OwnedProcess) Start(ctx context.Context, spec ProcessSpec) error {
	if process == nil {
		return errors.New("owned tunnel process is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	process.mu.Lock()
	defer process.mu.Unlock()
	if process.started {
		return errors.New("tunnel process is already started")
	}
	handle, err := process.runner.Start(ctx, spec)
	if err != nil {
		return err
	}
	if handle == nil {
		return errors.New("tunnel process runner returned no handle")
	}
	process.handle = handle
	process.done = make(chan struct{})
	process.started = true
	go process.wait(handle, process.done)
	return nil
}

func (process *OwnedProcess) wait(handle ProcessHandle, done chan struct{}) {
	err := handle.Wait()
	process.mu.Lock()
	process.waitErr = err
	process.mu.Unlock()
	close(done)
}

func (process *OwnedProcess) Done() <-chan struct{} {
	if process == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.done
}

func (process *OwnedProcess) Wait() error {
	if process == nil {
		return nil
	}
	done := process.Done()
	<-done
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.waitErr
}

func (process *OwnedProcess) Stop(ctx context.Context) error {
	if process == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	process.mu.Lock()
	started := process.started
	done := process.done
	handle := process.handle
	process.mu.Unlock()
	if !started || handle == nil {
		return nil
	}
	select {
	case <-done:
		return process.Wait()
	default:
	}

	process.stopOnce.Do(func() {
		process.stopErr = process.stop(ctx, handle, done)
	})
	return process.stopErr
}

func (process *OwnedProcess) stop(ctx context.Context, handle ProcessHandle, done <-chan struct{}) error {
	terminateErr := handle.Terminate()
	select {
	case <-done:
		return errors.Join(terminateErr, process.Wait())
	case <-process.options.After(process.options.GracePeriod):
		killErr := handle.Kill()
		select {
		case <-done:
			return errors.Join(terminateErr, killErr, process.Wait())
		case <-ctx.Done():
			return errors.Join(terminateErr, killErr, fmt.Errorf("wait for forced tunnel shutdown: %w", ctx.Err()))
		}
	case <-ctx.Done():
		killErr := handle.Kill()
		return errors.Join(terminateErr, killErr, fmt.Errorf("wait for tunnel shutdown: %w", ctx.Err()))
	}
}

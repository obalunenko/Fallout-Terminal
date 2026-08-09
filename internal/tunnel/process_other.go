//go:build !darwin

package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ProcessSpec is the complete, non-shell process launch request. Output
// writers are supplied by the tunnel service so it can parse bounded logs.
type ProcessSpec struct {
	Path   string
	Args   []string
	Env    []string
	Dir    string
	Stdout io.Writer
	Stderr io.Writer
}

// ProcessRunner starts one owned external process.
type ProcessRunner interface {
	Start(context.Context, ProcessSpec) (ProcessHandle, error)
}

// ProcessHandle is the portable lifecycle surface used by OwnedProcess.
type ProcessHandle interface {
	Wait() error
	Terminate() error
	Kill() error
}

// NewProcessRunner returns the portable child-process runner. Darwin has a
// process-group implementation; other platforms signal the direct child and
// retain Kill as the deterministic escalation fallback.
func NewProcessRunner() ProcessRunner {
	return osProcessRunner{}
}

type osProcessRunner struct{}

func (osProcessRunner) Start(ctx context.Context, spec ProcessSpec) (ProcessHandle, error) {
	if spec.Path == "" {
		return nil, errors.New("tunnel process path is empty")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("start tunnel process: %w", err)
	}

	// Context gates acquisition only. OwnedProcess remains responsible for
	// graceful termination and forced escalation.
	command := exec.Command(spec.Path, spec.Args...)
	command.Dir = spec.Dir
	if spec.Env != nil {
		command.Env = append([]string(nil), spec.Env...)
	}
	command.Stdout = spec.Stdout
	command.Stderr = spec.Stderr
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start tunnel process %q: %w", spec.Path, err)
	}
	return &osProcessHandle{command: command}, nil
}

type osProcessHandle struct {
	command *exec.Cmd
}

func (handle *osProcessHandle) Wait() error {
	if handle == nil || handle.command == nil {
		return errors.New("tunnel process is unavailable")
	}
	return handle.command.Wait()
}

func (handle *osProcessHandle) Terminate() error {
	if handle == nil || handle.command == nil || handle.command.Process == nil {
		return errors.New("tunnel process is unavailable")
	}
	return handle.command.Process.Signal(os.Interrupt)
}

func (handle *osProcessHandle) Kill() error {
	if handle == nil || handle.command == nil || handle.command.Process == nil {
		return errors.New("tunnel process is unavailable")
	}
	return handle.command.Process.Kill()
}

var _ ProcessRunner = osProcessRunner{}
var _ ProcessHandle = (*osProcessHandle)(nil)

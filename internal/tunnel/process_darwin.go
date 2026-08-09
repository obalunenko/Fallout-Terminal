//go:build darwin

package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"syscall"
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
// Darwin's implementation applies both signals to the dedicated child group.
type ProcessHandle interface {
	Wait() error
	Terminate() error
	Kill() error
}

// NewProcessRunner returns the supported Darwin runner. It starts every child
// in a new process group so shutdown also reaches descendants owned by ngrok.
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
	// shutdown so cancellation cannot bypass process-group cleanup.
	command := exec.Command(spec.Path, spec.Args...)
	command.Dir = spec.Dir
	if spec.Env != nil {
		command.Env = append([]string(nil), spec.Env...)
	}
	command.Stdout = spec.Stdout
	command.Stderr = spec.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start tunnel process %q: %w", spec.Path, err)
	}
	return &osProcessHandle{command: command, processGroupID: command.Process.Pid}, nil
}

type osProcessHandle struct {
	command        *exec.Cmd
	processGroupID int
}

func (handle *osProcessHandle) Wait() error {
	if handle == nil || handle.command == nil {
		return errors.New("tunnel process is unavailable")
	}
	return handle.command.Wait()
}

func (handle *osProcessHandle) Terminate() error {
	return handle.signalGroup(syscall.SIGTERM)
}

func (handle *osProcessHandle) Kill() error {
	return handle.signalGroup(syscall.SIGKILL)
}

func (handle *osProcessHandle) signalGroup(signal syscall.Signal) error {
	if handle == nil || handle.processGroupID <= 0 {
		return errors.New("tunnel process group is unavailable")
	}
	return syscall.Kill(-handle.processGroupID, signal)
}

var _ ProcessRunner = osProcessRunner{}
var _ ProcessHandle = (*osProcessHandle)(nil)

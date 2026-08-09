//go:build darwin

package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

const darwinOwnerGuardScript = `
cleanup_path=$1
shift
"$@" &
child_pid=$!
cat <&3 >/dev/null &
owner_guard_pid=$!

cleanup() {
  trap - HUP INT TERM EXIT
  kill -TERM "$child_pid" 2>/dev/null || true
  attempt=0
  while kill -0 "$child_pid" 2>/dev/null && [ "$attempt" -lt 20 ]; do
    sleep 0.1
    attempt=$((attempt + 1))
  done
  kill -KILL "$child_pid" 2>/dev/null || true
  wait "$child_pid" 2>/dev/null || true
  kill "$owner_guard_pid" 2>/dev/null || true
  wait "$owner_guard_pid" 2>/dev/null || true
  case "$cleanup_path" in
    */fallout-terminal-ngrok-*) rm -rf -- "$cleanup_path" ;;
  esac
}

trap 'cleanup; exit 0' HUP INT TERM
trap cleanup EXIT

while kill -0 "$child_pid" 2>/dev/null && kill -0 "$owner_guard_pid" 2>/dev/null; do
  sleep 0.1
done

if kill -0 "$owner_guard_pid" 2>/dev/null; then
  kill "$owner_guard_pid" 2>/dev/null || true
  wait "$owner_guard_pid" 2>/dev/null || true
  wait "$child_pid"
  child_status=$?
  trap - EXIT
  exit "$child_status"
fi

cleanup
trap - EXIT
exit 0
`

// ProcessSpec is the complete, non-shell process launch request. Output
// writers are supplied by the tunnel service so it can parse bounded logs.
type ProcessSpec struct {
	Path        string
	Args        []string
	Env         []string
	Dir         string
	Stdout      io.Writer
	Stderr      io.Writer
	CleanupPath string
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
	ownerReader, ownerWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create tunnel owner guard: %w", err)
	}
	commandArgs := []string{"-c", darwinOwnerGuardScript, "fallout-terminal-tunnel-guardian", spec.CleanupPath, spec.Path}
	commandArgs = append(commandArgs, spec.Args...)
	command := exec.Command("/bin/sh", commandArgs...)
	command.Dir = spec.Dir
	if spec.Env != nil {
		command.Env = append([]string(nil), spec.Env...)
	}
	command.Stdout = spec.Stdout
	command.Stderr = spec.Stderr
	command.ExtraFiles = []*os.File{ownerReader}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		_ = ownerReader.Close()
		_ = ownerWriter.Close()
		return nil, fmt.Errorf("start tunnel process %q: %w", spec.Path, err)
	}
	_ = ownerReader.Close()
	return &osProcessHandle{
		command:        command,
		processGroupID: command.Process.Pid,
		ownerPipe:      ownerWriter,
	}, nil
}

type osProcessHandle struct {
	command        *exec.Cmd
	processGroupID int
	ownerPipe      *os.File
	ownerClose     sync.Once
	ownerCloseErr  error
}

func (handle *osProcessHandle) Wait() error {
	if handle == nil || handle.command == nil {
		return errors.New("tunnel process is unavailable")
	}
	err := handle.command.Wait()
	_ = handle.CloseOwner()
	return err
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

// CloseOwner simulates or records loss of the owning application process. The
// write end also closes automatically in the kernel if that process is killed,
// allowing the Darwin guardian to reap ngrok even when Wails' native shutdown
// callback cannot run.
func (handle *osProcessHandle) CloseOwner() error {
	if handle == nil {
		return nil
	}
	handle.ownerClose.Do(func() {
		if handle.ownerPipe != nil {
			handle.ownerCloseErr = handle.ownerPipe.Close()
		}
	})
	return handle.ownerCloseErr
}

var _ ProcessRunner = osProcessRunner{}
var _ ProcessHandle = (*osProcessHandle)(nil)

//go:build darwin

package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const developmentSupervisorHelper = "FALLOUT_DEVELOPMENT_SUPERVISOR_HELPER"

func TestDevelopmentSupervisorInterruptCleansRealOwnedResources(t *testing.T) {
	if os.Getenv(developmentSupervisorHelper) == "1" {
		runDevelopmentSupervisorResourceHelper()
		return
	}

	temporary := t.TempDir()
	readyPath := filepath.Join(temporary, "ready")
	policyDirectory := filepath.Join(temporary, "fallout-terminal-ngrok-policy")
	if err := os.Mkdir(policyDirectory, 0o700); err != nil {
		t.Fatal(err)
	}

	runner := NewProcessRunner()
	handle, err := runner.Start(context.Background(), ProcessSpec{
		Path: os.Args[0],
		Args: []string{"-test.run=^TestDevelopmentSupervisorInterruptCleansRealOwnedResources$"},
		Env: append(os.Environ(),
			developmentSupervisorHelper+"=1",
			"FALLOUT_HELPER_READY="+readyPath,
			"FALLOUT_HELPER_POLICY="+policyDirectory,
		),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = handle.Kill() }()

	waitForFile(t, readyPath, 5*time.Second)
	if listener, listenErr := net.Listen("tcp4", "127.0.0.1:3690"); listenErr == nil {
		_ = listener.Close()
		t.Fatal("helper did not acquire the documented player port")
	}

	owner, ok := handle.(interface{ CloseOwner() error })
	if !ok {
		t.Fatal("Darwin tunnel process has no parent-lifetime guard; a supervisor kill can orphan ngrok")
	}
	if err := owner.CloseOwner(); err != nil {
		t.Fatalf("simulate development-supervisor process loss: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() { waitDone <- handle.Wait() }()
	select {
	case waitErr := <-waitDone:
		if waitErr != nil && !isExpectedSignalExit(waitErr) {
			t.Fatalf("guarded helper exit: %v", waitErr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("owned public process remained after the development supervisor disappeared")
	}

	waitForPortRelease(t, "127.0.0.1:3690", 5*time.Second)
	if _, err := os.Stat(policyDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("credential-policy directory still exists after handled supervisor cleanup: %v", err)
	}
}

func runDevelopmentSupervisorResourceHelper() {
	listener, err := net.Listen("tcp4", "127.0.0.1:3690")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	readyPath := os.Getenv("FALLOUT_HELPER_READY")
	policyDirectory := os.Getenv("FALLOUT_HELPER_POLICY")
	if err := os.WriteFile(readyPath, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		_ = listener.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown
	signal.Stop(shutdown)
	_ = listener.Close()
	_ = os.RemoveAll(policyDirectory)
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func waitForPortRelease(t *testing.T, address string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		listener, err := net.Listen("tcp4", address)
		if err == nil {
			_ = listener.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to be released", address)
}

func isExpectedSignalExit(err error) bool {
	var exitError *os.SyscallError
	return errors.As(err, &exitError)
}

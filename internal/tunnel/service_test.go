package tunnel

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigCredentialPrecedence(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"NGROK_ENABLED":    "1",
		"NGROK_USERNAME":   "environment-user",
		"NGROK_PASSWORD":   "environment-password",
		"NGROK_BASIC_AUTH": "combined-environment:combined-password",
	}
	tests := []struct {
		name string
		args []string
		want Credentials
	}{
		{
			name: "argument combined credential wins",
			args: []string{"--ngrok-basic-auth=argument-user:argument-password"},
			want: Credentials{Username: "argument-user", Password: "argument-password"},
		},
		{
			name: "argument pair wins over environment",
			args: []string{"--ngrok-username=argument-user", "--ngrok-password=argument-password"},
			want: Credentials{Username: "argument-user", Password: "argument-password"},
		},
		{
			name: "environment pair wins over combined environment",
			want: Credentials{Username: "environment-user", Password: "environment-password"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			config, err := ParseConfig(test.args, environmentLookup(environment))
			require.Falsef(t, err != nil,
				"ParseConfig() error = %v", err)
			require.False(t, !config.Enabled,
				"ParseConfig() did not enable public mode from NGROK_ENABLED")
			require.Falsef(t, !cmp.Equal(config.Credentials, test.want),
				"credentials = %#v, want %#v", config.Credentials, test.want)

		})
	}
}

func TestParseConfigRejectsInvalidCredentialsWithoutDisclosingPassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		password string
	}{
		{name: "missing username", password: "password-long-enough"},
		{name: "short password", username: "players", password: "short"},
		{name: "username newline", username: "play\ners", password: "password-long-enough"},
		{name: "password newline", username: "players", password: "password\nlong-enough"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			args := []string{"--ngrok", "--ngrok-username=" + test.username, "--ngrok-password=" + test.password}
			_, err := ParseConfig(args, environmentLookup(nil))
			require.False(t, err == nil,
				"ParseConfig() accepted invalid credentials")
			require.Falsef(t, test.password != "" && strings.Contains(err.Error(), test.password),
				"configuration error disclosed password: %q", err)

		})
	}
}

func TestFindPublicURLAcceptsOnlyHTTPSStartedTunnelMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want string
	}{
		{name: "JSON", line: `{"msg":"started tunnel","url":"https://fallout.example"}`, want: "https://fallout.example"},
		{name: "mixed log fallback", line: `INFO started tunnel url=https://fallback.example request=ready`, want: "https://fallback.example"},
		{name: "reject HTTP", line: `{"msg":"started tunnel","url":"http://unprotected.example"}`},
		{name: "reject unrelated HTTPS", line: `health check https://unrelated.example`},
		{name: "reject malformed URL", line: `started tunnel https://%zz`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			{
				got := FindPublicURL(test.line)
				require.Falsef(t, got != test.want,
					"FindPublicURL(%q) = %q, want %q", test.line, got, test.want)
			}

		})
	}
}

func TestServiceTimeoutStopsProcessAndRemovesPolicy(t *testing.T) {
	t.Parallel()

	timeout := make(chan time.Time, 1)
	runner := newServiceProcessRunner()
	policyParent := t.TempDir()
	service := NewService(serviceConfig(policyParent), runner, ServiceOptions{
		After: func(time.Duration) <-chan time.Time { return timeout },
	})

	result := make(chan error, 1)
	go func() {
		_, err := service.Start(t.Context())
		result <- err
	}()
	runner.waitStarted(t)
	timeout <- time.Now()

	select {
	case err := <-result:
		require.Falsef(t, err == nil || !strings.Contains(strings.ToLower(err.Error()), "timed out"),
			"Start() timeout error = %v", err)

	case <-time.After(time.Second):
		assert.FailNow(t, "Start() did not return after its injected timeout")
	}
	require.Falsef(t, runner.handle.terminateCalls() != 1,
		"timeout terminate calls = %d, want 1", runner.handle.terminateCalls())

	assertNoPolicyDirectories(t, policyParent)
}

func TestServiceEarlyExitReturnsBoundedRedactedDiagnosticAndCleansUp(t *testing.T) {
	t.Parallel()

	config := serviceConfig(t.TempDir())
	secret := config.Credentials.Password
	runner := newServiceProcessRunner()
	runner.stderr = "ngrok rejected credential " + secret + "\n" + strings.Repeat("x", 6000) + " diagnostic-tail"
	runner.handle.exit(errors.New("exit status 1"))
	service := NewService(config, runner, ServiceOptions{})

	_, err := service.Start(t.Context())
	require.False(t, err == nil,
		"Start() succeeded after an early process exit")
	require.Falsef(t, strings.Contains(err.Error(), secret),
		"Start() diagnostic disclosed password: %q", err)
	require.Falsef(t, !strings.Contains(err.Error(), "diagnostic-tail"),
		"Start() lost actionable diagnostic tail: %q", err)
	require.Falsef(t, len(err.Error()) > 4200,
		"Start() diagnostic is unbounded: %d bytes", len(err.Error()))

	assertNoPolicyDirectories(t, config.PolicyParent)
}

func TestServiceSuccessReturnsPublicInfoWithoutCredentialsOrTrafficPolicy(t *testing.T) {
	t.Parallel()

	config := serviceConfig(t.TempDir())
	runner := newServiceProcessRunner()
	runner.stdout = `{"msg":"started tunnel","url":"https://fallout.example"}` + "\n"
	service := NewService(config, runner, ServiceOptions{})

	info, err := service.Start(t.Context())
	require.Falsef(t, err != nil,
		"Start() error = %v", err)
	require.Falsef(t, info.URL != "https://fallout.example" || info.LocalURL != config.LocalURL || !info.Tunnel || info.TunnelError != "",
		"Start() info = %#v", info)

	serialized := fmt.Sprintf("%+v", info)
	require.Falsef(t, strings.Contains(serialized, config.Credentials.Username) || strings.Contains(serialized, config.Credentials.Password),
		"public status disclosed credentials: %s", serialized)
	require.Falsef(t, strings.Contains(strings.Join(runner.spec.Args, " "), config.Credentials.Password),
		"process arguments disclosed password: %#v", runner.spec.Args)

	wantArgs := []string{
		"http", "3690",
		"--url", "https://fallout-terminal.ngrok.app",
		"--log", "stdout",
		"--log-format", "json",
	}
	require.Falsef(t, !cmp.Equal(runner.spec.Args, wantArgs),
		"protected forwarding args = %#v, want %#v", runner.spec.Args, wantArgs)

	assertNoPolicyDirectories(t, config.PolicyParent)
	{

		err := service.Stop(t.Context())
		require.Falsef(t, err != nil,
			"Stop() error = %v", err)
	}

}

func TestServiceFailedFirstStopCanBeRetriedAfterProcessExit(t *testing.T) {
	t.Parallel()

	config := serviceConfig(t.TempDir())
	runner := newServiceProcessRunner()
	runner.stdout = `{"msg":"started tunnel","url":"https://fallout.example"}` + "\n"
	runner.handle.terminateErr = errors.New("temporary terminate failure")
	service := NewService(config, runner, ServiceOptions{})

	_, err := service.Start(t.Context())
	require.NoError(t, err)
	require.ErrorContains(t, service.Stop(t.Context()), "temporary terminate failure")
	require.NoError(t, service.Stop(t.Context()))
	require.Equal(t, 1, runner.handle.terminateCalls())
}

func serviceConfig(policyParent string) Config {
	return Config{
		Enabled:        true,
		Binary:         "ngrok",
		Domain:         "fallout-terminal.ngrok.app",
		Port:           3690,
		LocalURL:       "http://192.0.2.10:3690",
		StartupTimeout: 20 * time.Second,
		PolicyParent:   policyParent,
		Credentials: Credentials{
			Username: "players",
			Password: "super-secret-password",
		},
	}
}

func environmentLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func assertNoPolicyDirectories(t *testing.T, parent string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(parent, "fallout-terminal-ngrok-*"))
	if err != nil {
		require.NoError(t, err)
	}
	require.Falsef(t, len(matches) != 0,
		"credential policy directories remain: %#v", matches)

}

type serviceProcessRunner struct {
	mu      sync.Mutex
	spec    ProcessSpec
	stdout  string
	stderr  string
	started chan struct{}
	handle  *serviceProcessHandle
}

func newServiceProcessRunner() *serviceProcessRunner {
	return &serviceProcessRunner{
		started: make(chan struct{}),
		handle:  newServiceProcessHandle(),
	}
}

func (runner *serviceProcessRunner) Start(_ context.Context, spec ProcessSpec) (ProcessHandle, error) {
	runner.mu.Lock()
	runner.spec = spec
	stdout := runner.stdout
	stderr := runner.stderr
	runner.mu.Unlock()

	close(runner.started)
	go func() {
		if stdout != "" && spec.Stdout != nil {
			_, _ = fmt.Fprint(spec.Stdout, stdout)
		}
		if stderr != "" && spec.Stderr != nil {
			_, _ = fmt.Fprint(spec.Stderr, stderr)
		}
	}()
	return runner.handle, nil
}

func (runner *serviceProcessRunner) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-runner.started:
	case <-time.After(time.Second):
		assert.FailNow(t, "process runner was not started")
	}
}

type serviceProcessHandle struct {
	done         chan struct{}
	once         sync.Once
	mu           sync.Mutex
	err          error
	terminate    int
	kill         int
	terminateErr error
}

func newServiceProcessHandle() *serviceProcessHandle {
	return &serviceProcessHandle{done: make(chan struct{})}
}

func (process *serviceProcessHandle) Wait() error {
	<-process.done
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.err
}

func (process *serviceProcessHandle) Terminate() error {
	process.mu.Lock()
	process.terminate++
	process.mu.Unlock()
	process.exit(nil)
	return process.terminateErr
}

func (process *serviceProcessHandle) Kill() error {
	process.mu.Lock()
	process.kill++
	process.mu.Unlock()
	process.exit(nil)
	return nil
}

func (process *serviceProcessHandle) exit(err error) {
	process.once.Do(func() {
		process.mu.Lock()
		process.err = err
		process.mu.Unlock()
		close(process.done)
	})
}

func (process *serviceProcessHandle) terminateCalls() int {
	process.mu.Lock()
	defer process.mu.Unlock()
	return process.terminate
}

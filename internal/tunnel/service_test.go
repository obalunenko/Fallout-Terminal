package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
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
			if err != nil {
				t.Fatalf("ParseConfig() error = %v", err)
			}
			if !config.Enabled {
				t.Fatal("ParseConfig() did not enable public mode from NGROK_ENABLED")
			}
			if !reflect.DeepEqual(config.Credentials, test.want) {
				t.Fatalf("credentials = %#v, want %#v", config.Credentials, test.want)
			}
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
			if err == nil {
				t.Fatal("ParseConfig() accepted invalid credentials")
			}
			if test.password != "" && strings.Contains(err.Error(), test.password) {
				t.Fatalf("configuration error disclosed password: %q", err)
			}
		})
	}
}

func TestCreatePolicyEscapesCredentialAndUsesPrivatePermissions(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	credential := Credentials{
		Username: `player"\\name`,
		Password: `long-"password\\value`,
	}
	policy, err := CreatePolicy(parent, credential)
	if err != nil {
		t.Fatalf("CreatePolicy() error = %v", err)
	}
	t.Cleanup(func() { _ = policy.Cleanup() })

	raw, err := os.ReadFile(policy.Path)
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	var document struct {
		OnHTTPRequest []struct {
			Actions []struct {
				Type   string `json:"type"`
				Config struct {
					Realm       string   `json:"realm"`
					Credentials []string `json:"credentials"`
					Enforce     bool     `json:"enforce"`
				} `json:"config"`
			} `json:"actions"`
		} `json:"on_http_request"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("policy is not valid escaped JSON: %v\n%s", err, raw)
	}
	if len(document.OnHTTPRequest) != 1 || len(document.OnHTTPRequest[0].Actions) != 1 {
		t.Fatalf("policy actions = %#v", document.OnHTTPRequest)
	}
	action := document.OnHTTPRequest[0].Actions[0]
	wantCredential := credential.Username + ":" + credential.Password
	if action.Type != "basic-auth" || !action.Config.Enforce || !reflect.DeepEqual(action.Config.Credentials, []string{wantCredential}) {
		t.Fatalf("basic-auth action = %#v", action)
	}

	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(policy.Path)
		if err != nil {
			t.Fatal(err)
		}
		if got := fileInfo.Mode().Perm(); got != 0o600 {
			t.Errorf("policy permissions = %04o, want 0600", got)
		}
		directoryInfo, err := os.Stat(filepath.Dir(policy.Path))
		if err != nil {
			t.Fatal(err)
		}
		if got := directoryInfo.Mode().Perm(); got != 0o700 {
			t.Errorf("policy directory permissions = %04o, want 0700", got)
		}
	}
}

func TestPolicyCleanupIsIdempotent(t *testing.T) {
	t.Parallel()

	policy, err := CreatePolicy(t.TempDir(), Credentials{Username: "players", Password: "password-long-enough"})
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Dir(policy.Path)
	if err := policy.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if err := policy.Cleanup(); err != nil {
		t.Fatalf("second Cleanup() error = %v", err)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("policy directory remains after cleanup: %v", err)
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
			if got := FindPublicURL(test.line); got != test.want {
				t.Fatalf("FindPublicURL(%q) = %q, want %q", test.line, got, test.want)
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
		_, err := service.Start(context.Background())
		result <- err
	}()
	runner.waitStarted(t)
	timeout <- time.Now()

	select {
	case err := <-result:
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timed out") {
			t.Fatalf("Start() timeout error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start() did not return after its injected timeout")
	}
	if runner.handle.terminateCalls() != 1 {
		t.Fatalf("timeout terminate calls = %d, want 1", runner.handle.terminateCalls())
	}
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

	_, err := service.Start(context.Background())
	if err == nil {
		t.Fatal("Start() succeeded after an early process exit")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("Start() diagnostic disclosed password: %q", err)
	}
	if !strings.Contains(err.Error(), "diagnostic-tail") {
		t.Fatalf("Start() lost actionable diagnostic tail: %q", err)
	}
	if len(err.Error()) > 4200 {
		t.Fatalf("Start() diagnostic is unbounded: %d bytes", len(err.Error()))
	}
	assertNoPolicyDirectories(t, config.PolicyParent)
}

func TestServiceSuccessReturnsPublicInfoWithoutCredentialsAndRemovesPolicy(t *testing.T) {
	t.Parallel()

	config := serviceConfig(t.TempDir())
	runner := newServiceProcessRunner()
	runner.stdout = `{"msg":"started tunnel","url":"https://fallout.example"}` + "\n"
	service := NewService(config, runner, ServiceOptions{})

	info, err := service.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if info.URL != "https://fallout.example" || info.LocalURL != config.LocalURL || !info.Tunnel || info.TunnelError != "" {
		t.Fatalf("Start() info = %#v", info)
	}
	serialized := fmt.Sprintf("%+v", info)
	if strings.Contains(serialized, config.Credentials.Username) || strings.Contains(serialized, config.Credentials.Password) {
		t.Fatalf("public status disclosed credentials: %s", serialized)
	}
	if strings.Contains(strings.Join(runner.spec.Args, " "), config.Credentials.Password) {
		t.Fatalf("process arguments disclosed password: %#v", runner.spec.Args)
	}
	assertNoPolicyDirectories(t, config.PolicyParent)

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
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
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("credential policy directories remain: %#v", matches)
	}
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
		t.Fatal("process runner was not started")
	}
}

type serviceProcessHandle struct {
	done      chan struct{}
	once      sync.Once
	mu        sync.Mutex
	err       error
	terminate int
	kill      int
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
	return nil
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

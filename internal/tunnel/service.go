package tunnel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

const diagnosticLimit = 4000

var startedTunnelURL = regexp.MustCompile(`https://[^\s"']+`)

// ServiceOptions supplies deterministic clocks for startup tests and the
// owned-process grace period.
type ServiceOptions struct {
	After       func(time.Duration) <-chan time.Time
	GracePeriod time.Duration
}

// Service owns one optional ngrok process and its short-lived credential
// policy. The policy is removed as soon as startup succeeds or fails.
type Service struct {
	mu      sync.Mutex
	config  Config
	process *OwnedProcess
	after   func(time.Duration) <-chan time.Time
	started bool
}

func NewService(config Config, runner ProcessRunner, options ServiceOptions) *Service {
	if options.After == nil {
		options.After = time.After
	}
	return &Service{
		config: config,
		process: NewOwnedProcess(runner, ProcessOptions{
			GracePeriod: options.GracePeriod,
		}),
		after: options.After,
	}
}

func (service *Service) Start(ctx context.Context) (domain.ServerInfo, error) {
	if service == nil {
		return domain.ServerInfo{}, errors.New("public tunnel service is unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	service.mu.Lock()
	if service.started {
		service.mu.Unlock()
		return domain.ServerInfo{}, errors.New("public tunnel is already started")
	}
	service.mu.Unlock()
	if !service.config.Enabled {
		return domain.ServerInfo{}, errors.New("public tunnel is not enabled")
	}
	if err := validateCredentials(service.config.Credentials); err != nil {
		return domain.ServerInfo{}, err
	}
	endpoint, err := publicEndpoint(service.config.Domain)
	if err != nil {
		return domain.ServerInfo{}, err
	}

	policy, err := CreatePolicy(service.config.PolicyParent, service.config.Credentials)
	if err != nil {
		return domain.ServerInfo{}, fmt.Errorf("prepare protected public access: %w", err)
	}
	defer policy.Cleanup()

	urls := make(chan string, 1)
	stdout := newURLWriter(urls)
	stderr := newTailWriter(diagnosticLimit)
	spec := ProcessSpec{
		Path: service.config.Binary,
		Args: []string{
			"http", strconv.Itoa(service.config.Port),
			"--url", endpoint,
			"--traffic-policy-file", policy.Path,
			"--log", "stdout",
			"--log-format", "json",
		},
		Env:         os.Environ(),
		Stdout:      stdout,
		Stderr:      stderr,
		CleanupPath: policy.directory,
	}
	if err := service.process.Start(ctx, spec); err != nil {
		return domain.ServerInfo{}, service.redactedError("start ngrok", err.Error())
	}

	timeout := service.config.StartupTimeout
	if timeout <= 0 {
		timeout = DefaultStartupTimeout
	}
	select {
	case publicURL := <-urls:
		service.mu.Lock()
		service.started = true
		service.mu.Unlock()
		return domain.ServerInfo{
			URL: publicURL, LocalURL: service.config.LocalURL, Tunnel: true,
		}, nil
	case <-service.process.Done():
		waitErr := service.process.Wait()
		diagnostic := stderr.String()
		if diagnostic == "" {
			select {
			case <-stderr.Written():
				diagnostic = stderr.String()
			case <-time.After(10 * time.Millisecond):
			}
		}
		if diagnostic == "" && waitErr != nil {
			diagnostic = waitErr.Error()
		}
		return domain.ServerInfo{}, service.redactedError("ngrok exited before publishing an HTTPS address", diagnostic)
	case <-service.after(timeout):
		_ = service.process.Stop(ctx)
		return domain.ServerInfo{}, errors.New("ngrok startup timed out before publishing an HTTPS address")
	case <-ctx.Done():
		_ = service.process.Stop(context.Background())
		return domain.ServerInfo{}, fmt.Errorf("start public tunnel: %w", ctx.Err())
	}
}

func (service *Service) Stop(ctx context.Context) error {
	if service == nil || service.process == nil {
		return nil
	}
	err := service.process.Stop(ctx)
	service.mu.Lock()
	service.started = false
	service.mu.Unlock()
	return err
}

// FindPublicURL accepts only an HTTPS URL on an ngrok "started tunnel" log
// entry. Unrelated URLs and unprotected HTTP endpoints are ignored.
func FindPublicURL(line string) string {
	var entry struct {
		Message string `json:"msg"`
		URL     string `json:"url"`
	}
	if json.Unmarshal([]byte(line), &entry) == nil && entry.Message == "started tunnel" {
		if validPublicURL(entry.URL) {
			return entry.URL
		}
		return ""
	}
	if !strings.Contains(strings.ToLower(line), "started tunnel") {
		return ""
	}
	candidate := startedTunnelURL.FindString(line)
	if validPublicURL(candidate) {
		return candidate
	}
	return ""
}

func publicEndpoint(domainName string) (string, error) {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		return "", errors.New("ngrok domain must not be empty")
	}
	if !strings.Contains(domainName, "://") {
		domainName = "https://" + domainName
	}
	if !validPublicURL(domainName) {
		return "", errors.New("ngrok endpoint must be a valid HTTPS URL")
	}
	return domainName, nil
}

func validPublicURL(raw string) bool {
	parsed, err := url.ParseRequestURI(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func (service *Service) redactedError(prefix, diagnostic string) error {
	diagnostic = strings.TrimSpace(diagnostic)
	for _, secret := range []string{service.config.Credentials.Password, service.config.Credentials.Username} {
		if secret != "" {
			diagnostic = strings.ReplaceAll(diagnostic, secret, "[REDACTED]")
		}
	}
	if len(diagnostic) > diagnosticLimit {
		diagnostic = diagnostic[len(diagnostic)-diagnosticLimit:]
	}
	if diagnostic == "" {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %s", prefix, diagnostic)
}

type urlWriter struct {
	mu     sync.Mutex
	buffer string
	urls   chan<- string
}

func newURLWriter(urls chan<- string) *urlWriter {
	return &urlWriter{urls: urls}
}

func (writer *urlWriter) Write(payload []byte) (int, error) {
	writer.mu.Lock()
	writer.buffer += string(payload)
	lines := strings.Split(writer.buffer, "\n")
	writer.buffer = lines[len(lines)-1]
	if len(writer.buffer) > diagnosticLimit {
		writer.buffer = writer.buffer[len(writer.buffer)-diagnosticLimit:]
	}
	writer.mu.Unlock()
	for _, line := range lines[:len(lines)-1] {
		if publicURL := FindPublicURL(strings.TrimSuffix(line, "\r")); publicURL != "" {
			select {
			case writer.urls <- publicURL:
			default:
			}
		}
	}
	return len(payload), nil
}

type tailWriter struct {
	mu      sync.Mutex
	buffer  []byte
	limit   int
	written chan struct{}
	once    sync.Once
}

func newTailWriter(limit int) *tailWriter {
	return &tailWriter{limit: limit, written: make(chan struct{})}
}

func (writer *tailWriter) Write(payload []byte) (int, error) {
	writer.mu.Lock()
	writer.buffer = append(writer.buffer, payload...)
	if len(writer.buffer) > writer.limit {
		writer.buffer = append([]byte(nil), writer.buffer[len(writer.buffer)-writer.limit:]...)
	}
	writer.mu.Unlock()
	writer.once.Do(func() { close(writer.written) })
	return len(payload), nil
}

func (writer *tailWriter) String() string {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return string(bytes.Clone(writer.buffer))
}

func (writer *tailWriter) Written() <-chan struct{} {
	return writer.written
}

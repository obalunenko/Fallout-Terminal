package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeNgrokCodedError struct{ code string }

func (failure fakeNgrokCodedError) Error() string { return "synthetic provider diagnostic" }
func (failure fakeNgrokCodedError) Code() string  { return failure.code }

type fakeSDKFactory struct {
	agent *fakeSDKAgent
	err   error
	seen  []byte
}

func (factory *fakeSDKFactory) New(accountToken []byte) (ngrokAgent, error) {
	factory.seen = append([]byte(nil), accountToken...)
	if factory.err != nil {
		return nil, factory.err
	}
	return factory.agent, nil
}

type fakeSDKAgent struct {
	mu             sync.Mutex
	forwarder      *fakeSDKForwarder
	request        ngrokForwardRequest
	forwardErr     error
	forwardGate    chan struct{}
	disconnectErrs []error
	disconnectGate chan struct{}
	disconnects    int
}

func (agent *fakeSDKAgent) Forward(_ context.Context, request ngrokForwardRequest) (ngrokForwarder, error) {
	agent.mu.Lock()
	agent.request = request
	gate := agent.forwardGate
	agent.mu.Unlock()
	if gate != nil {
		<-gate
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	if agent.forwardErr != nil {
		return nil, agent.forwardErr
	}
	return agent.forwarder, nil
}

func (agent *fakeSDKAgent) Disconnect() error {
	agent.mu.Lock()
	gate := agent.disconnectGate
	agent.mu.Unlock()
	if gate != nil {
		<-gate
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.disconnects++
	if agent.disconnects <= len(agent.disconnectErrs) {
		return agent.disconnectErrs[agent.disconnects-1]
	}
	return nil
}

type fakeSDKForwarder struct {
	mu        sync.Mutex
	endpoint  *url.URL
	done      chan struct{}
	doneOnce  sync.Once
	closes    int
	closeErrs []error
	closeGate chan struct{}
}

func newFakeSDKForwarder(raw string) *fakeSDKForwarder {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return &fakeSDKForwarder{endpoint: parsed, done: make(chan struct{})}
}

func (forwarder *fakeSDKForwarder) URL() *url.URL         { return forwarder.endpoint }
func (forwarder *fakeSDKForwarder) Done() <-chan struct{} { return forwarder.done }
func (forwarder *fakeSDKForwarder) Close(context.Context) error {
	forwarder.mu.Lock()
	gate := forwarder.closeGate
	forwarder.mu.Unlock()
	if gate != nil {
		<-gate
	}
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	forwarder.closes++
	if forwarder.closes <= len(forwarder.closeErrs) {
		return forwarder.closeErrs[forwarder.closes-1]
	}
	forwarder.doneOnce.Do(func() { close(forwarder.done) })
	return nil
}

func (forwarder *fakeSDKForwarder) closeCalls() int {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	return forwarder.closes
}

func protectedStartRequest(reserved string) TunnelStartRequest {
	return TunnelStartRequest{
		UpstreamURL: "http://127.0.0.1:3690", ReservedDomain: reserved,
		AccountToken: []byte("synthetic-scoped-input"), PlayerUsername: []byte("players"),
		PlayerPassword: []byte("synthetic-player-input"),
	}
}

func TestEmbeddedNgrokAdapterAttachesBasicAuthPolicyToDirectUpstreamBeforeReturningEndpoint(t *testing.T) {
	for _, test := range []struct{ name, reserved string }{
		{name: "random address omits URL"},
		{name: "reserved address is exact", reserved: "vault.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			forwarder := newFakeSDKForwarder("https://" + map[bool]string{true: "vault.example", false: "random.example"}[test.reserved != ""])
			agent := &fakeSDKAgent{forwarder: forwarder}
			factory := &fakeSDKFactory{agent: agent}
			endpoint, err := newNgrokServiceWithFactory(factory).Start(t.Context(), protectedStartRequest(test.reserved))
			require.NoError(t, err)
			require.NotNil(t, endpoint)
			assert.Equal(t, []byte("synthetic-scoped-input"), factory.seen)
			assert.Equal(t, "http://127.0.0.1:3690", agent.request.UpstreamURL)
			assert.Equal(t, test.reserved, agent.request.ReservedDomain)

			var policy map[string]any
			require.NoError(t, json.Unmarshal([]byte(agent.request.TrafficPolicy), &policy))
			phases, ok := policy["on_http_request"].([]any)
			require.True(t, ok)
			require.Len(t, phases, 1)
			rules := phases[0].(map[string]any)
			assert.NotContains(t, rules, "expressions", "endpoint Basic Auth must cover every HTTP path")
			actions := rules["actions"].([]any)
			require.Len(t, actions, 1)
			action := actions[0].(map[string]any)
			assert.Equal(t, "basic-auth", action["type"])
			config := action["config"].(map[string]any)
			assert.Equal(t, true, config["enforce"])
			assert.Equal(t, "Fallout Terminal Players", config["realm"])
			assert.Equal(t, []any{"players:synthetic-player-input"}, config["credentials"])
		})
	}
}

func TestEmbeddedNgrokAdapterRejectsMissingOrUnsafeEndpointCredentialsBeforeForward(t *testing.T) {
	for _, mutate := range []func(*TunnelStartRequest){
		func(request *TunnelStartRequest) { request.AccountToken = nil },
		func(request *TunnelStartRequest) { request.PlayerUsername = nil },
		func(request *TunnelStartRequest) { request.PlayerPassword = nil },
		func(request *TunnelStartRequest) { request.PlayerPassword = []byte("short") },
		func(request *TunnelStartRequest) { request.PlayerUsername = []byte("bad:name") },
		func(request *TunnelStartRequest) { request.UpstreamURL = "http://127.0.0.1:3691" },
	} {
		agent := &fakeSDKAgent{forwarder: newFakeSDKForwarder("https://random.example")}
		request := protectedStartRequest("")
		mutate(&request)
		endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), request)
		require.Error(t, err)
		assert.Nil(t, endpoint)
		assert.Empty(t, agent.request.UpstreamURL)
	}
}

func TestEmbeddedNgrokAdapterRejectsUnsafeOrMismatchedHTTPSPrivately(t *testing.T) {
	for _, test := range []struct{ endpoint, reserved string }{
		{endpoint: "http://random.example"},
		{endpoint: "https://user@random.example"},
		{endpoint: "https://other.example", reserved: "vault.example"},
	} {
		forwarder := newFakeSDKForwarder(test.endpoint)
		agent := &fakeSDKAgent{forwarder: forwarder}
		endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), protectedStartRequest(test.reserved))
		require.Error(t, err)
		assert.Nil(t, endpoint)
		assert.Equal(t, 1, forwarder.closes)
		assert.Equal(t, 1, agent.disconnects)
	}
}

func TestEmbeddedNgrokEndpointDoneAndIdempotentBoundedCloseDisconnect(t *testing.T) {
	forwarder := newFakeSDKForwarder("https://random.example")
	agent := &fakeSDKAgent{forwarder: forwarder}
	endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), protectedStartRequest(""))
	require.NoError(t, err)
	assert.True(t, endpoint.Done() == forwarder.done)
	require.NoError(t, endpoint.Close(t.Context()))
	require.NoError(t, endpoint.Close(t.Context()))
	assert.Equal(t, 1, forwarder.closes)
	assert.Equal(t, 1, agent.disconnects)
}

func TestEmbeddedNgrokEndpointCloseFailureIsRedactedAndRetryable(t *testing.T) {
	forwarder := newFakeSDKForwarder("https://random.example")
	forwarder.closeErrs = []error{errors.New("synthetic forwarder close diagnostic")}
	agent := &fakeSDKAgent{
		forwarder:      forwarder,
		disconnectErrs: []error{errors.New("synthetic agent disconnect diagnostic")},
	}
	endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), protectedStartRequest(""))
	require.NoError(t, err)

	closeErr := endpoint.Close(t.Context())
	require.Error(t, closeErr)
	assert.Equal(t, ErrorProviderFailure.SafeMessage(), closeErr.Error())
	assert.NotContains(t, closeErr.Error(), "synthetic")
	assert.Nil(t, endpoint.URL())
	require.NoError(t, endpoint.Close(t.Context()))
	require.NoError(t, endpoint.Close(t.Context()))
	assert.Equal(t, 2, forwarder.closes)
	assert.Equal(t, 2, agent.disconnects)
}

func TestEmbeddedNgrokAdapterMapsAgentAndForwardFailuresWithoutSDKDiagnostics(t *testing.T) {
	for _, factory := range []*fakeSDKFactory{
		{err: errors.New("sensitive agent diagnostic")},
		{agent: &fakeSDKAgent{forwardErr: errors.New("sensitive forward diagnostic")}},
	} {
		_, err := newNgrokServiceWithFactory(factory).Start(t.Context(), protectedStartRequest(""))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "sensitive")
	}
}

func TestEmbeddedNgrokAdapterFailureMatrixIsStableAndRedacted(t *testing.T) {
	tests := []struct {
		name     string
		failure  error
		category ErrorCategory
	}{
		{name: "invalid token", failure: fakeNgrokCodedError{code: "ERR_NGROK_105"}, category: ErrorProviderAuthentication},
		{name: "revoked token", failure: fakeNgrokCodedError{code: "ERR_NGROK_107"}, category: ErrorProviderAuthentication},
		{name: "domain conflict", failure: fakeNgrokCodedError{code: "ERR_NGROK_320"}, category: ErrorDomainUnavailable},
		{name: "dns unavailable", failure: &net.DNSError{Err: "synthetic lookup diagnostic", Name: "private.invalid"}, category: ErrorNetworkUnavailable},
		{name: "deadline", failure: context.DeadlineExceeded, category: ErrorTimeout},
		{name: "policy construction", failure: errors.New("synthetic policy diagnostic"), category: ErrorProviderFailure},
		{name: "provider failure", failure: fakeNgrokCodedError{code: "ERR_NGROK_999"}, category: ErrorProviderFailure},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factory := &fakeSDKFactory{agent: &fakeSDKAgent{forwardErr: test.failure}}
			_, err := newNgrokServiceWithFactory(factory).Start(t.Context(), protectedStartRequest(""))
			require.Error(t, err)
			category, message := redactedPublicAccessFailure(err)
			assert.Equal(t, test.category, category)
			assert.Equal(t, test.category.SafeMessage(), message)
			assert.NotContains(t, message, "synthetic")
			assert.NotContains(t, message, "private.invalid")
		})
	}
}

func TestEmbeddedNgrokStartCancellationAfterLateForwardClosesAcquiredResources(t *testing.T) {
	forwardGate := make(chan struct{})
	forwarder := newFakeSDKForwarder("https://late.example")
	agent := &fakeSDKAgent{forwarder: forwarder, forwardGate: forwardGate}
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan struct {
		endpoint TunnelEndpoint
		err      error
	}, 1)
	go func() {
		endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(ctx, protectedStartRequest(""))
		result <- struct {
			endpoint TunnelEndpoint
			err      error
		}{endpoint: endpoint, err: err}
	}()
	require.Eventually(t, func() bool {
		agent.mu.Lock()
		defer agent.mu.Unlock()
		return agent.request.UpstreamURL != ""
	}, time.Second, time.Millisecond)
	cancel()
	close(forwardGate)
	started := <-result
	require.Error(t, started.err)
	assert.Nil(t, started.endpoint)
	assert.Equal(t, 1, forwarder.closeCalls())
	assert.Equal(t, 1, agent.disconnects)
}

func TestEmbeddedNgrokEndpointConcurrentCloseIsBoundedAndReleasesReferences(t *testing.T) {
	forwarder := newFakeSDKForwarder("https://random.example")
	agent := &fakeSDKAgent{forwarder: forwarder}
	owned, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), protectedStartRequest(""))
	require.NoError(t, err)
	endpoint := owned.(*embeddedNgrokEndpoint)

	results := make(chan error, 32)
	for range 32 {
		go func() { results <- endpoint.Close(t.Context()) }()
	}
	for range 32 {
		assert.NoError(t, <-results)
	}
	assert.Equal(t, 1, forwarder.closeCalls())
	assert.Equal(t, 1, agent.disconnects)
	endpoint.stateMu.Lock()
	assert.Nil(t, endpoint.forwarder)
	assert.Nil(t, endpoint.agent)
	endpoint.stateMu.Unlock()
	require.Eventually(t, func() bool {
		select {
		case <-endpoint.Done():
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond, "Done must be closed after successful cleanup")
}

func TestEmbeddedNgrokEndpointCloseDoesNotTrustBlockingSDKComponentsToHonorContext(t *testing.T) {
	closeGate := make(chan struct{})
	var closeGateOnce sync.Once
	t.Cleanup(func() { closeGateOnce.Do(func() { close(closeGate) }) })
	forwarder := newFakeSDKForwarder("https://random.example")
	forwarder.closeGate = closeGate
	agent := &fakeSDKAgent{forwarder: forwarder}
	endpoint, err := newNgrokServiceWithFactory(&fakeSDKFactory{agent: agent}).Start(t.Context(), protectedStartRequest(""))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- endpoint.Close(ctx) }()
	var closeErr error
	require.Eventually(t, func() bool {
		select {
		case closeErr = <-finished:
			return true
		default:
			return false
		}
	}, 250*time.Millisecond, time.Millisecond, "endpoint Close exceeded its bound when the SDK forwarder ignored context")
	require.Error(t, closeErr)
	assert.Equal(t, ErrorShutdownTimeout.SafeMessage(), closeErr.Error())

	closeGateOnce.Do(func() { close(closeGate) })
	require.NoError(t, endpoint.Close(t.Context()))
	assert.Equal(t, 1, agent.disconnects)
}

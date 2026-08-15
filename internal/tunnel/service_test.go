package tunnel

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contractAgentFactory struct {
	agent *contractAgent
	err   error
}

func (factory contractAgentFactory) New([]byte) (ngrokAgent, error) {
	if factory.err != nil {
		return nil, factory.err
	}
	return factory.agent, nil
}

type contractAgent struct {
	forwarder  *contractForwarder
	forward    func(context.Context) error
	mu         sync.Mutex
	disconnect int
}

func (agent *contractAgent) Forward(ctx context.Context, _ ngrokForwardRequest) (ngrokForwarder, error) {
	if agent.forward != nil {
		if err := agent.forward(ctx); err != nil {
			return nil, err
		}
	}
	return agent.forwarder, nil
}

func (agent *contractAgent) Disconnect() error {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.disconnect++
	return nil
}

func (agent *contractAgent) disconnectCalls() int {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.disconnect
}

type contractForwarder struct {
	endpoint *url.URL
	done     chan struct{}
	once     sync.Once
	mu       sync.Mutex
	closes   int
}

func newContractForwarder(t *testing.T) *contractForwarder {
	t.Helper()
	endpoint, err := url.Parse("https://random.example")
	require.NoError(t, err)
	return &contractForwarder{endpoint: endpoint, done: make(chan struct{})}
}

func (forwarder *contractForwarder) URL() *url.URL         { return forwarder.endpoint }
func (forwarder *contractForwarder) Done() <-chan struct{} { return forwarder.done }
func (forwarder *contractForwarder) Close(context.Context) error {
	forwarder.mu.Lock()
	forwarder.closes++
	forwarder.mu.Unlock()
	forwarder.once.Do(func() { close(forwarder.done) })
	return nil
}

func (forwarder *contractForwarder) closeCalls() int {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()
	return forwarder.closes
}

func contractStartRequest() TunnelStartRequest {
	return TunnelStartRequest{
		UpstreamURL:    "http://" + PlayerUpstreamAddress,
		AccountToken:   []byte("synthetic-account-input"),
		PlayerUsername: []byte("players"),
		PlayerPassword: []byte("synthetic-player-input"),
	}
}

func TestEmbeddedTunnelServiceLifecycleContract(t *testing.T) {
	t.Run("timeout and cancellation disconnect without an endpoint", func(t *testing.T) {
		for _, test := range []struct {
			name    string
			prepare func(*TunnelStartRequest) (context.Context, context.CancelFunc)
		}{
			{
				name: "timeout",
				prepare: func(request *TunnelStartRequest) (context.Context, context.CancelFunc) {
					request.Timeout = time.Millisecond
					return t.Context(), func() {}
				},
			},
			{
				name: "cancellation",
				prepare: func(*TunnelStartRequest) (context.Context, context.CancelFunc) {
					return context.WithCancel(t.Context())
				},
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				request := contractStartRequest()
				ctx, cancel := test.prepare(&request)
				if test.name == "cancellation" {
					cancel()
				} else {
					defer cancel()
				}
				agent := &contractAgent{forward: func(ctx context.Context) error {
					<-ctx.Done()
					return ctx.Err()
				}}
				service := newNgrokServiceWithFactory(contractAgentFactory{agent: agent})

				endpoint, err := service.Start(ctx, request)
				require.Error(t, err)
				assert.Nil(t, endpoint)
				category, message := redactedPublicAccessFailure(err)
				assert.Equal(t, ErrorTimeout, category)
				assert.Equal(t, ErrorTimeout.SafeMessage(), message)
				assert.Equal(t, 1, agent.disconnectCalls())
			})
		}
	})

	t.Run("Done and concurrent Close have one owned lifecycle", func(t *testing.T) {
		forwarder := newContractForwarder(t)
		agent := &contractAgent{forwarder: forwarder}
		service := newNgrokServiceWithFactory(contractAgentFactory{agent: agent})
		request := contractStartRequest()
		accountInput := request.AccountToken
		passwordInput := request.PlayerPassword

		endpoint, err := service.Start(t.Context(), request)
		require.NoError(t, err)
		require.NotNil(t, endpoint)
		assert.Equal(t, "https://random.example", endpoint.URL().String())
		assert.Equal(t, make([]byte, len(accountInput)), accountInput)
		assert.Equal(t, make([]byte, len(passwordInput)), passwordInput)
		assert.Equal(t, (<-chan struct{})(forwarder.done), endpoint.Done())

		results := make(chan error, 16)
		for range 16 {
			go func() { results <- endpoint.Close(t.Context()) }()
		}
		for range 16 {
			require.NoError(t, <-results)
		}
		assert.Equal(t, 1, forwarder.closeCalls())
		assert.Equal(t, 1, agent.disconnectCalls())
		assert.Nil(t, endpoint.URL())
		select {
		case <-endpoint.Done():
		default:
			assert.Fail(t, "Done remained open after successful concurrent Close")
		}
	})

	t.Run("provider diagnostics are redacted", func(t *testing.T) {
		const diagnostic = "synthetic private provider diagnostic"
		service := newNgrokServiceWithFactory(contractAgentFactory{err: errors.New(diagnostic)})
		endpoint, err := service.Start(t.Context(), contractStartRequest())
		require.Error(t, err)
		assert.Nil(t, endpoint)
		assert.NotContains(t, err.Error(), diagnostic)
		assert.Equal(t, ErrorProviderFailure.SafeMessage(), err.Error())
	})
}

func TestTunnelStartRequestClearRemovesEphemeralSecrets(t *testing.T) {
	t.Parallel()

	request := contractStartRequest()
	request.Clear()

	require.Nil(t, request.AccountToken)
	require.Nil(t, request.PlayerUsername)
	require.Nil(t, request.PlayerPassword)
}

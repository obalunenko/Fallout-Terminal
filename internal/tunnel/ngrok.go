package tunnel

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"

	ngrok "golang.ngrok.com/ngrok/v2"
)

type ngrokForwardRequest struct {
	UpstreamURL    string
	ReservedDomain string
	TrafficPolicy  string
}

type ngrokForwarder interface {
	URL() *url.URL
	Done() <-chan struct{}
	Close(context.Context) error
}

type ngrokAgent interface {
	Forward(context.Context, ngrokForwardRequest) (ngrokForwarder, error)
	Disconnect() error
}

type ngrokAgentFactory interface {
	New([]byte) (ngrokAgent, error)
}

type sdkAgentFactory struct{}

func (sdkAgentFactory) New(accountToken []byte) (ngrokAgent, error) {
	if len(accountToken) == 0 {
		return nil, errors.New(ErrorCredentialMissing.SafeMessage())
	}
	token := string(accountToken)
	agent, err := ngrok.NewAgent(ngrok.WithAuthtoken(token), ngrok.WithAutoConnect(false))
	token = ""
	if err != nil {
		return nil, newRedactedPublicAccessError(err)
	}
	return &sdkAgent{agent: agent}, nil
}

type sdkAgent struct {
	agent ngrok.Agent
}

func (agent *sdkAgent) Forward(ctx context.Context, request ngrokForwardRequest) (ngrokForwarder, error) {
	if err := agent.agent.Connect(ctx); err != nil {
		return nil, newRedactedPublicAccessError(err)
	}
	options := make([]ngrok.EndpointOption, 0, 2)
	if request.ReservedDomain != "" {
		options = append(options, ngrok.WithURL("https://"+request.ReservedDomain))
	}
	options = append(options, ngrok.WithTrafficPolicy(request.TrafficPolicy))
	forwarder, err := agent.agent.Forward(ctx, ngrok.WithUpstream(request.UpstreamURL), options...)
	if err != nil {
		return nil, newRedactedPublicAccessError(err)
	}
	return sdkForwarder{EndpointForwarder: forwarder}, nil
}

func (agent *sdkAgent) Disconnect() error {
	return agent.agent.Disconnect()
}

type sdkForwarder struct {
	ngrok.EndpointForwarder
}

func (forwarder sdkForwarder) Close(ctx context.Context) error {
	return forwarder.CloseWithContext(ctx)
}

type embeddedNgrokService struct {
	factory ngrokAgentFactory
}

func NewEmbeddedNgrokService() TunnelService {
	return newNgrokServiceWithFactory(sdkAgentFactory{})
}

func newNgrokServiceWithFactory(factory ngrokAgentFactory) *embeddedNgrokService {
	return &embeddedNgrokService{factory: factory}
}

func (service *embeddedNgrokService) Start(ctx context.Context, request TunnelStartRequest) (TunnelEndpoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ownedToken := append([]byte(nil), request.AccountToken...)
	ownedUsername := append([]byte(nil), request.PlayerUsername...)
	ownedPassword := append([]byte(nil), request.PlayerPassword...)
	request.Clear()
	defer clear(ownedToken)
	defer clear(ownedUsername)
	defer clear(ownedPassword)

	if service == nil || service.factory == nil {
		return nil, errors.New(ErrorProviderFailure.SafeMessage())
	}
	if request.UpstreamURL != "http://"+PlayerUpstreamAddress {
		return nil, errors.New(ErrorValidation.SafeMessage())
	}
	reservedDomain, err := NormalizeReservedDomain(request.ReservedDomain)
	if err != nil {
		return nil, errors.New(ErrorValidation.SafeMessage())
	}
	if len(ownedToken) == 0 {
		return nil, errors.New(ErrorCredentialMissing.SafeMessage())
	}
	policy, err := basicAuthTrafficPolicy(ownedUsername, ownedPassword)
	if err != nil {
		return nil, err
	}
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	agent, err := service.factory.New(ownedToken)
	if err != nil {
		return nil, newRedactedPublicAccessError(err)
	}
	forwarder, err := agent.Forward(ctx, ngrokForwardRequest{
		UpstreamURL:    request.UpstreamURL,
		ReservedDomain: reservedDomain,
		TrafficPolicy:  policy,
	})
	if err != nil {
		_ = agent.Disconnect()
		return nil, newRedactedPublicAccessError(err)
	}
	ownedEndpoint := newEmbeddedNgrokEndpoint(nil, forwarder, agent)
	if ctx.Err() != nil {
		_ = ownedEndpoint.Close(context.Background())
		return nil, publicAccessCategorizedError{category: ErrorTimeout}
	}

	publicURL := forwarder.URL()
	if publicURL == nil {
		_ = ownedEndpoint.Close(context.Background())
		return nil, errors.New(ErrorProviderFailure.SafeMessage())
	}
	canonicalURL, _, err := NormalizeEndpointURL(publicURL.String(), reservedDomain)
	if err != nil {
		_ = ownedEndpoint.Close(context.Background())
		return nil, errors.New(ErrorProviderFailure.SafeMessage())
	}
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		_ = ownedEndpoint.Close(context.Background())
		return nil, errors.New(ErrorProviderFailure.SafeMessage())
	}
	ownedEndpoint.stateMu.Lock()
	ownedEndpoint.url = parsed
	ownedEndpoint.stateMu.Unlock()
	return ownedEndpoint, nil
}

func basicAuthTrafficPolicy(username, password []byte) (string, error) {
	if len(username) == 0 || strings.ContainsAny(string(username), ":\x00\r\n") ||
		ValidatePlayerPassword(password) != nil || len(password) > MaximumPlayerPasswordBytes {
		return "", errors.New(ErrorValidation.SafeMessage())
	}
	credential := make([]byte, 0, len(username)+1+len(password))
	credential = append(credential, username...)
	credential = append(credential, ':')
	credential = append(credential, password...)
	defer clear(credential)
	policy := struct {
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
	}{OnHTTPRequest: make([]struct {
		Actions []struct {
			Type   string `json:"type"`
			Config struct {
				Realm       string   `json:"realm"`
				Credentials []string `json:"credentials"`
				Enforce     bool     `json:"enforce"`
			} `json:"config"`
		} `json:"actions"`
	}, 1)}
	policy.OnHTTPRequest[0].Actions = make([]struct {
		Type   string `json:"type"`
		Config struct {
			Realm       string   `json:"realm"`
			Credentials []string `json:"credentials"`
			Enforce     bool     `json:"enforce"`
		} `json:"config"`
	}, 1)
	action := &policy.OnHTTPRequest[0].Actions[0]
	action.Type = "basic-auth"
	action.Config.Realm = "Fallout Terminal Players"
	action.Config.Credentials = []string{string(credential)}
	action.Config.Enforce = true
	encoded, err := json.Marshal(policy)
	clear(action.Config.Credentials)
	if err != nil {
		return "", errors.New(ErrorProviderFailure.SafeMessage())
	}
	return string(encoded), nil
}

type embeddedNgrokEndpoint struct {
	url               *url.URL
	forwarder         ngrokForwarder
	agent             ngrokAgent
	stateMu           sync.Mutex
	closeMu           sync.Mutex
	forwarderClosed   bool
	agentDisconnected bool
	closeAttempt      *embeddedNgrokCloseAttempt
	done              <-chan struct{}
}

type embeddedNgrokCloseAttempt struct {
	done                 chan struct{}
	forwarderClosed      bool
	agentDisconnected    bool
	forwarderError       error
	agentDisconnectError error
}

func newEmbeddedNgrokEndpoint(publicURL *url.URL, forwarder ngrokForwarder, agent ngrokAgent) *embeddedNgrokEndpoint {
	endpoint := &embeddedNgrokEndpoint{
		url: publicURL, forwarder: forwarder, agent: agent,
		done: forwarder.Done(),
	}
	return endpoint
}

func (endpoint *embeddedNgrokEndpoint) URL() *url.URL {
	if endpoint == nil {
		return nil
	}
	endpoint.stateMu.Lock()
	defer endpoint.stateMu.Unlock()
	if endpoint.url == nil {
		return nil
	}
	copy := *endpoint.url
	return &copy
}

func (endpoint *embeddedNgrokEndpoint) Done() <-chan struct{} {
	if endpoint == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return endpoint.done
}

func (endpoint *embeddedNgrokEndpoint) Close(ctx context.Context) error {
	if endpoint == nil {
		return nil
	}
	ctx, cancel := boundedPublicAccessCleanupContext(ctx)
	defer cancel()

	endpoint.closeMu.Lock()
	endpoint.stateMu.Lock()
	endpoint.url = nil
	complete := endpoint.forwarderClosed && endpoint.agentDisconnected
	endpoint.stateMu.Unlock()
	if complete {
		endpoint.closeMu.Unlock()
		return nil
	}
	attempt := endpoint.closeAttempt
	if attempt == nil {
		endpoint.stateMu.Lock()
		forwarder := endpoint.forwarder
		agent := endpoint.agent
		forwarderClosed := endpoint.forwarderClosed || forwarder == nil
		agentDisconnected := endpoint.agentDisconnected || agent == nil
		endpoint.stateMu.Unlock()
		attempt = &embeddedNgrokCloseAttempt{done: make(chan struct{})}
		endpoint.closeAttempt = attempt
		go runEmbeddedNgrokCloseAttempt(ctx, attempt, forwarder, agent, forwarderClosed, agentDisconnected)
	}
	endpoint.closeMu.Unlock()

	select {
	case <-attempt.done:
	case <-ctx.Done():
		return publicAccessCategorizedError{category: ErrorShutdownTimeout}
	}

	endpoint.closeMu.Lock()
	if endpoint.closeAttempt == attempt {
		endpoint.closeAttempt = nil
	}
	endpoint.stateMu.Lock()
	endpoint.forwarderClosed = endpoint.forwarderClosed || attempt.forwarderClosed
	endpoint.agentDisconnected = endpoint.agentDisconnected || attempt.agentDisconnected
	if endpoint.forwarderClosed {
		endpoint.forwarder = nil
	}
	if endpoint.agentDisconnected {
		endpoint.agent = nil
	}
	complete = endpoint.forwarderClosed && endpoint.agentDisconnected
	endpoint.stateMu.Unlock()
	endpoint.closeMu.Unlock()
	if !complete {
		category, _ := redactedPublicAccessFailure(errors.Join(attempt.forwarderError, attempt.agentDisconnectError))
		if errors.Is(attempt.forwarderError, context.DeadlineExceeded) || errors.Is(attempt.forwarderError, context.Canceled) {
			category = ErrorShutdownTimeout
		}
		return publicAccessCategorizedError{category: category}
	}
	return nil
}

func runEmbeddedNgrokCloseAttempt(
	ctx context.Context,
	attempt *embeddedNgrokCloseAttempt,
	forwarder ngrokForwarder,
	agent ngrokAgent,
	forwarderClosed bool,
	agentDisconnected bool,
) {
	forwarderResult := make(chan error, 1)
	agentResult := make(chan error, 1)
	if forwarderClosed {
		forwarderResult <- nil
	} else {
		go func() { forwarderResult <- forwarder.Close(ctx) }()
	}
	if agentDisconnected {
		agentResult <- nil
	} else {
		go func() { agentResult <- agent.Disconnect() }()
	}
	attempt.forwarderError = <-forwarderResult
	attempt.agentDisconnectError = <-agentResult
	attempt.forwarderClosed = forwarderClosed || attempt.forwarderError == nil
	attempt.agentDisconnected = agentDisconnected || attempt.agentDisconnectError == nil
	close(attempt.done)
}

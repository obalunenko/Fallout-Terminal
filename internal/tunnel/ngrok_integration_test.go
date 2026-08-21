package tunnel_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1/playerv1connect"
	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type publicStreamProbeService struct {
	playerv1connect.UnimplementedPlayerServiceHandler
	upstreamArrival chan time.Time
}

func (service *publicStreamProbeService) Subscribe(
	ctx context.Context,
	_ *connect.Request[playerv1.SubscribeRequest],
	stream *connect.ServerStream[playerv1.SubscriptionMessage],
) error {
	select {
	case service.upstreamArrival <- time.Now():
	default:
	}
	if err := stream.Send(&playerv1.SubscriptionMessage{Payload: &playerv1.SubscriptionMessage_Snapshot{
		Snapshot: &playerv1.PersonalizedSnapshot{RecognitionHandle: "integration-handle", Revision: 1},
	}}); err != nil {
		return err
	}
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	if err := stream.Send(&playerv1.SubscriptionMessage{Payload: &playerv1.SubscriptionMessage_Update{
		Update: &playerv1.CompoundUpdate{Revision: 2},
	}}); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (service *publicStreamProbeService) SoundManifest(
	context.Context,
	*connect.Request[playerv1.SoundManifestRequest],
) (*connect.Response[playerv1.SoundManifestResponse], error) {
	return connect.NewResponse(&playerv1.SoundManifestResponse{}), nil
}

type subscribeHeaderEvidence struct {
	status      int
	contentType string
	at          time.Time
}

type publicAuthTransport struct {
	base     http.RoundTripper
	username string
	password []byte
	headers  chan subscribeHeaderEvidence
}

func (transport *publicAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	forwarded := request.Clone(request.Context())
	forwarded.SetBasicAuth(transport.username, string(transport.password))
	response, err := transport.base.RoundTrip(forwarded)
	if response != nil && strings.HasSuffix(request.URL.Path, "/Subscribe") {
		select {
		case transport.headers <- subscribeHeaderEvidence{
			status: response.StatusCode, contentType: response.Header.Get("Content-Type"), at: time.Now(),
		}:
		default:
		}
	}
	return response, err
}

func TestEmbeddedNgrokSDKOptInAuthenticatedGeneratedSubscribe(t *testing.T) {
	if os.Getenv("FALLOUT_NGROK_INTEGRATION") != "1" {
		t.Skip("NOT RUN: explicit real-ngrok integration opt-in was not provided")
	}
	token := []byte(os.Getenv("FALLOUT_NGROK_AUTHTOKEN"))
	password := []byte(os.Getenv("FALLOUT_PUBLIC_TEST_PASSWORD"))
	defer clear(token)
	defer clear(password)
	if len(token) == 0 || len(password) < tunnel.MinimumPlayerPasswordBytes {
		t.Skip("NOT RUN: external real-ngrok test credentials are unavailable")
	}
	username := strings.TrimSpace(os.Getenv("FALLOUT_PUBLIC_TEST_USERNAME"))
	if username == "" {
		username = tunnel.DefaultPlayerUsername
	}
	reservedDomain := strings.TrimSpace(os.Getenv("FALLOUT_NGROK_RESERVED_DOMAIN"))

	listener, err := net.Listen("tcp4", tunnel.PlayerUpstreamAddress)
	require.NoError(t, err)
	probe := &publicStreamProbeService{upstreamArrival: make(chan time.Time, 1)}
	rpcPath, rpcHandler := playerv1connect.NewPlayerServiceHandler(probe)
	mux := http.NewServeMux()
	mux.Handle(rpcPath, rpcHandler)
	mux.HandleFunc("/", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Shutdown(t.Context()) }()

	ingress, err := tunnel.NewPublicIngressFactory().Start(t.Context(), "http://"+tunnel.PlayerUpstreamAddress)
	require.NoError(t, err)
	defer func() { require.NoError(t, ingress.Close(t.Context())) }()

	service := tunnel.NewEmbeddedNgrokService()
	endpoint, err := service.Start(t.Context(), tunnel.TunnelStartRequest{
		UpstreamURL: ingress.URL().String(), ReservedDomain: reservedDomain,
		AccountToken: token, Timeout: 30 * time.Second,
	})
	require.NoError(t, err)
	defer func() {
		ingress.Deny()
		require.NoError(t, endpoint.Close(t.Context()))
	}()

	publicURL := endpoint.URL()
	require.NotNil(t, publicURL)
	require.Equal(t, "https", publicURL.Scheme)
	if reservedDomain != "" {
		assert.Equal(t, strings.TrimSuffix(strings.ToLower(reservedDomain), "."), strings.ToLower(publicURL.Hostname()))
	}
	require.NoError(t, ingress.Activate(publicURL.Hostname(), username, password))

	client := &http.Client{Timeout: 15 * time.Second}
	for _, test := range []struct {
		name, username string
		password       []byte
		want           int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong", username: username, password: []byte("synthetic-wrong-input"), want: http.StatusUnauthorized},
		{name: "correct", username: username, password: password, want: http.StatusNoContent},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, requestErr := http.NewRequestWithContext(t.Context(), http.MethodGet, publicURL.String(), nil)
			require.NoError(t, requestErr)
			if test.username != "" {
				request.SetBasicAuth(test.username, string(test.password))
			}
			response, requestErr := client.Do(request)
			require.NoError(t, requestErr)
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			assert.Equal(t, test.want, response.StatusCode)
		})
	}

	headers := make(chan subscribeHeaderEvidence, 1)
	authenticatedClient := &http.Client{Transport: &publicAuthTransport{
		base: http.DefaultTransport, username: username, password: password, headers: headers,
	}}
	generated := playerv1connect.NewPlayerServiceClient(authenticatedClient, publicURL.String())
	streamDeadline, stopStreamDeadline := context.WithTimeoutCause(t.Context(), 5*time.Second, errors.New("test ngrok stream timed out"))
	streamContext, cancelStream := context.WithCancelCause(streamDeadline)
	defer func() {
		cancelStream(errors.New("test ngrok stream completed"))
		stopStreamDeadline()
	}()
	startedAt := time.Now()
	stream, err := generated.Subscribe(streamContext, connect.NewRequest(&playerv1.SubscribeRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), "initial public Subscribe frame: %v", stream.Err())
	firstAt := time.Now()
	require.NotNil(t, stream.Msg().GetSnapshot())
	require.True(t, stream.Receive(), "later public Subscribe frame: %v", stream.Err())
	updateAt := time.Now()
	require.Equal(t, uint64(2), stream.Msg().GetUpdate().GetRevision())

	headerEvidence := <-headers
	upstreamAt := <-probe.upstreamArrival
	assert.Equal(t, http.StatusOK, headerEvidence.status)
	assert.Equal(t, "application/connect+proto", headerEvidence.contentType)
	t.Logf(
		"public stream evidence status=%d content_type=%q upstream_arrival_ms=%d response_headers_ms=%d first_snapshot_ms=%d later_update_ms=%d",
		headerEvidence.status,
		headerEvidence.contentType,
		upstreamAt.Sub(startedAt).Milliseconds(),
		headerEvidence.at.Sub(startedAt).Milliseconds(),
		firstAt.Sub(startedAt).Milliseconds(),
		updateAt.Sub(startedAt).Milliseconds(),
	)
}

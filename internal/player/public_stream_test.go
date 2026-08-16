package player

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/obalunenko/Fallout-Terminal/internal/control"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1/playerv1connect"
	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

const (
	edgeTestUsername = "players"
	edgeTestPassword = "synthetic-player-input"
)

type endpointAuthTransport struct {
	base     http.RoundTripper
	username string
	password string
}

type publicIngressTransport struct {
	base     http.RoundTripper
	target   *url.URL
	host     string
	username string
	password string
}

func (transport publicIngressTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	forwarded := request.Clone(request.Context())
	targetURL := *request.URL
	targetURL.Scheme = transport.target.Scheme
	targetURL.Host = transport.target.Host
	forwarded.URL = &targetURL
	forwarded.Host = transport.host
	forwarded.SetBasicAuth(transport.username, transport.password)
	return transport.base.RoundTrip(forwarded)
}

func (transport endpointAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	forwarded := request.Clone(request.Context())
	forwarded.SetBasicAuth(transport.username, transport.password)
	return transport.base.RoundTrip(forwarded)
}

func endpointAuthSeam(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != edgeTestUsername || password != edgeTestPassword {
			response.Header().Set("WWW-Authenticate", `Basic realm="Fallout Terminal Players"`)
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		forwarded := request.Clone(request.Context())
		forwarded.Header.Del("Authorization")
		next.ServeHTTP(response, forwarded)
	})
}

func TestPublicIngressProtectsStaticUnaryAndStreamingBeforeUnchangedPlayerBoundary(t *testing.T) {
	var service *ConnectService
	coordinator := newConnectTestCoordinator(t, func(effect control.Effect) {
		if service != nil {
			service.PublishEffect(effect)
		}
	})
	var err error
	service, err = NewConnectService(ConnectServiceConfig{Coordinator: coordinator, Assets: playerAssets()})
	require.NoError(t, err)
	rpcPath, rpcHandler := NewConnectHandler(service)
	application := NewApplicationHandler(playerAssets(), rpcPath, rpcHandler)
	server := httptest.NewServer(application)
	t.Cleanup(server.Close)
	ingress, err := tunnel.NewPublicIngressFactory().Start(t.Context(), server.URL)
	require.NoError(t, err)
	defer func() { require.NoError(t, ingress.Close(t.Context())) }()
	require.NoError(t, ingress.Activate("public.example", edgeTestUsername, []byte(edgeTestPassword)))

	for _, path := range []string{
		"/", "/client.js",
		"/fallout.terminal.player.v1.PlayerService/SoundManifest",
		"/fallout.terminal.player.v1.PlayerService/Subscribe",
	} {
		for _, credentials := range []struct{ username, password string }{{}, {username: edgeTestUsername, password: "synthetic-wrong-input"}} {
			request, requestErr := http.NewRequestWithContext(t.Context(), http.MethodPost, ingress.URL().String()+path, bytes.NewReader([]byte{0, 0, 0, 0, 0}))
			require.NoError(t, requestErr)
			request.Host = "public.example"
			request.Header.Set("Content-Type", "application/connect+proto")
			if credentials.username != "" {
				request.SetBasicAuth(credentials.username, credentials.password)
			}
			response, requestErr := server.Client().Do(request)
			require.NoError(t, requestErr)
			_ = response.Body.Close()
			assert.Equal(t, http.StatusUnauthorized, response.StatusCode, path)
			assert.NotEmpty(t, response.Header.Get("WWW-Authenticate"), path)
		}
	}

	authenticatedClient := &http.Client{Transport: publicIngressTransport{
		base: server.Client().Transport, target: ingress.URL(), host: "public.example",
		username: edgeTestUsername, password: edgeTestPassword,
	}}
	staticRequest, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://public.example/", nil)
	require.NoError(t, err)
	staticResponse, err := authenticatedClient.Do(staticRequest)
	require.NoError(t, err)
	_ = staticResponse.Body.Close()
	assert.Equal(t, http.StatusOK, staticResponse.StatusCode)
	assert.Empty(t, staticResponse.Header.Get("WWW-Authenticate"))

	client := playerv1connect.NewPlayerServiceClient(authenticatedClient, "http://public.example")
	manifest, err := client.SoundManifest(t.Context(), connect.NewRequest(&playerv1.SoundManifestRequest{
		Category: playerv1.SoundCategory_SOUND_CATEGORY_AMBIENT,
	}))
	require.NoError(t, err)
	require.NotNil(t, manifest.Msg)

	streamContext, cancelStream := context.WithCancel(t.Context())
	defer cancelStream()
	stream, err := client.Subscribe(streamContext, connect.NewRequest(&playerv1.SubscribeRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), "stream error: %v", stream.Err())
	snapshot := stream.Msg().GetSnapshot()
	require.NotNil(t, snapshot)
	clonedSnapshot := proto.Clone(snapshot).(*playerv1.PersonalizedSnapshot)
	require.Empty(t, cmp.Diff(snapshot, clonedSnapshot, protocmp.Transform()))

	selected, err := client.SelectCharacter(t.Context(), connect.NewRequest(&playerv1.SelectCharacterRequest{
		RecognitionHandle: snapshot.GetRecognitionHandle(), RequestId: "edge-request-1",
		BroadcastId: "broadcast-1", CharacterId: "character-1",
	}))
	require.NoError(t, err)
	require.True(t, selected.Msg.GetAccepted())
	require.True(t, stream.Receive(), "stream error: %v", stream.Err())
	update := stream.Msg().GetUpdate()
	require.NotNil(t, update)
	assert.Equal(t, selected.Msg.GetRevision(), update.GetRevision())
}

func TestAuthenticatedForwardingStillAppliesOriginAndBodyLimitsInsidePlayer(t *testing.T) {
	var calls int
	application := NewApplicationHandler(playerAssets(), "/fallout.terminal.player.v1.PlayerService/", http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls++
		response.WriteHeader(http.StatusNoContent)
	}))
	handler := endpointAuthSeam(application)

	for _, test := range []struct {
		name   string
		origin string
		body   []byte
		want   int
	}{
		{name: "same origin reaches RPC", origin: "https://public.example", want: http.StatusNoContent},
		{name: "foreign origin rejected after edge", origin: "https://other.example", want: http.StatusForbidden},
		{name: "encoded limit rejected after edge", origin: "https://public.example", body: bytes.Repeat([]byte{'x'}, MaxEncodedBodyBytes+1), want: http.StatusTooManyRequests},
	} {
		request := httptest.NewRequest(http.MethodPost, "https://public.example/fallout.terminal.player.v1.PlayerService/Navigate", bytes.NewReader(test.body))
		request.Host = "public.example"
		request.Header.Set("Origin", test.origin)
		request.Header.Set("Content-Type", "application/proto")
		request.SetBasicAuth(edgeTestUsername, edgeTestPassword)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assert.Equal(t, test.want, recorder.Code, test.name)
	}
	assert.Equal(t, 1, calls)
}

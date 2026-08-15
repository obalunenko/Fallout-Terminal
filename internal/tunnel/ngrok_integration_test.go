package tunnel_test

import (
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedNgrokSDKOptInProtectedDirectUpstream(t *testing.T) {
	if os.Getenv("FALLOUT_NGROK_INTEGRATION") != "1" {
		t.Skip("NOT RUN: explicit real-ngrok integration opt-in was not provided")
	}
	token := []byte(os.Getenv("FALLOUT_NGROK_AUTHTOKEN"))
	password := []byte(os.Getenv("FALLOUT_NGROK_TEST_PASSWORD"))
	defer clear(token)
	defer clear(password)
	if len(token) == 0 || len(password) < tunnel.MinimumPlayerPasswordBytes {
		t.Skip("NOT RUN: external real-ngrok test credentials are unavailable")
	}
	username := strings.TrimSpace(os.Getenv("FALLOUT_NGROK_TEST_USERNAME"))
	if username == "" {
		username = tunnel.DefaultPlayerUsername
	}
	reservedDomain := strings.TrimSpace(os.Getenv("FALLOUT_NGROK_RESERVED_DOMAIN"))

	listener, err := net.Listen("tcp4", tunnel.PlayerUpstreamAddress)
	require.NoError(t, err)
	defer listener.Close()
	upstreamRequests := make(chan struct{}, 4)
	server := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		upstreamRequests <- struct{}{}
		response.WriteHeader(http.StatusNoContent)
	})}
	go func() { _ = server.Serve(listener) }()
	defer func() { _ = server.Shutdown(t.Context()) }()

	service := tunnel.NewEmbeddedNgrokService()
	endpoint, err := service.Start(t.Context(), tunnel.TunnelStartRequest{
		UpstreamURL: "http://" + tunnel.PlayerUpstreamAddress, ReservedDomain: reservedDomain,
		AccountToken: token, PlayerUsername: []byte(username), PlayerPassword: password,
		Timeout: 30 * time.Second,
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, endpoint.Close(t.Context())) }()

	publicURL := endpoint.URL()
	require.NotNil(t, publicURL)
	require.Equal(t, "https", publicURL.Scheme)
	if reservedDomain != "" {
		assert.Equal(t, strings.TrimSuffix(strings.ToLower(reservedDomain), "."), strings.ToLower(publicURL.Hostname()))
	}

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

	select {
	case <-upstreamRequests:
	case <-time.After(15 * time.Second):
		assert.Fail(t, "authenticated request did not reach the direct upstream")
	}
	select {
	case <-upstreamRequests:
		assert.Fail(t, "an unauthenticated request reached the direct upstream")
	default:
	}
}

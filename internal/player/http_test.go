package player

import (
	"bytes"
	"compress/gzip"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"connectrpc.com/connect"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type countingConnectCoordinator struct {
	ConnectCoordinator
	mutations atomic.Int64
}

func (coordinator *countingConnectCoordinator) DispatchPlayerActionForRecognition(handle domain.RecognitionHandle, command domain.RuntimeCommand) domain.ActionResult {
	coordinator.mutations.Add(1)
	return coordinator.ConnectCoordinator.DispatchPlayerActionForRecognition(handle, command)
}

func TestConnectHTTPRejectsDecodedCompressedUnknownAndMalformedBodiesBeforeCanonicalMutation(t *testing.T) {
	t.Parallel()

	base := newConnectTestCoordinator(t)
	coordinator := &countingConnectCoordinator{ConnectCoordinator: base}
	service, err := NewConnectService(ConnectServiceConfig{Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	rpcPath, rpcHandler := NewConnectHandler(service)
	handler := NewApplicationHandler(playerAssets(), rpcPath, rpcHandler)
	request := &playerv1.NavigateRequest{
		RecognitionHandle: "recognition-1", RequestId: "request-1", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
		Action: &playerv1.NavigateRequest_Back{Back: &playerv1.NavigateBack{}},
	}
	unknown := protowire.AppendTag(nil, 100, protowire.BytesType)
	unknown = protowire.AppendBytes(unknown, bytes.Repeat([]byte{'x'}, MaxUncompressedMessageBytes))
	request.ProtoReflect().SetUnknown(unknown)
	oversized, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(oversized) <= MaxUncompressedMessageBytes || len(oversized) >= MaxEncodedBodyBytes {
		t.Fatalf("unknown-field fixture size = %d, want between decoded and encoded limits", len(oversized))
	}

	var compressed bytes.Buffer
	zipper, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := zipper.Write(oversized); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	if compressed.Len() >= MaxUncompressedMessageBytes {
		t.Fatalf("compressed fixture size = %d, want below decoded limit", compressed.Len())
	}

	tests := []struct {
		name            string
		body            []byte
		contentEncoding string
		wantStatus      int
		wantCode        string
	}{
		{name: "unknown field growth", body: oversized, wantStatus: http.StatusTooManyRequests, wantCode: "resource_exhausted"},
		{name: "compressed expansion", body: compressed.Bytes(), contentEncoding: "gzip", wantStatus: http.StatusTooManyRequests, wantCode: "resource_exhausted"},
		{name: "malformed bounded protobuf", body: []byte{0x0a}, wantStatus: http.StatusBadRequest, wantCode: "invalid_argument"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpRequest := httptest.NewRequest(http.MethodPost, "http://player.test/fallout.terminal.player.v1.PlayerService/Navigate", bytes.NewReader(test.body))
			httpRequest.Header.Set("Content-Type", "application/proto")
			if test.contentEncoding != "" {
				httpRequest.Header.Set("Content-Encoding", test.contentEncoding)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httpRequest)
			if recorder.Code != test.wantStatus || !strings.Contains(recorder.Body.String(), test.wantCode) {
				t.Fatalf("status/body = %d %q, want %d containing %q", recorder.Code, recorder.Body.String(), test.wantStatus, test.wantCode)
			}
		})
	}
	if got := coordinator.mutations.Load(); got != 0 {
		t.Fatalf("canonical mutation calls = %d, want zero", got)
	}
	if base.Revision() != 2 {
		t.Fatalf("canonical revision changed after boundary rejection: %d", base.Revision())
	}
}

func TestApplicationHandlerRejectsCrossOriginMalformedHostAndOversizedBodiesBeforeRPC(t *testing.T) {
	t.Parallel()

	var calls int
	rpc := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		calls++
		response.WriteHeader(http.StatusNoContent)
	})
	handler := NewApplicationHandler(playerAssets(), "/fallout.terminal.player.v1.PlayerService/", rpc)

	tests := []struct {
		name   string
		host   string
		origin string
		body   []byte
		path   string
		status int
		calls  int
	}{
		{name: "same origin", host: "player.test", origin: "https://player.test", path: "/fallout.terminal.player.v1.PlayerService/Navigate", status: http.StatusNoContent, calls: 1},
		{name: "foreign origin", host: "player.test", origin: "https://evil.example", path: "/fallout.terminal.player.v1.PlayerService/Navigate", status: http.StatusForbidden, calls: 1},
		{name: "malformed host", host: "player.test bad", path: "/fallout.terminal.player.v1.PlayerService/Navigate", status: http.StatusForbidden, calls: 1},
		{name: "encoded body over eight KiB", host: "player.test", body: bytes.Repeat([]byte{'x'}, MaxEncodedBodyBytes+1), path: "/fallout.terminal.player.v1.PlayerService/Navigate", status: http.StatusRequestEntityTooLarge, calls: 1},
		{name: "lookalike service path", host: "player.test", path: "/fallout.terminal.player.v1.PlayerServiceEvil/Navigate", status: http.StatusMethodNotAllowed, calls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://player.test"+test.path, bytes.NewReader(test.body))
			request.Host = test.host
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, test.status, recorder.Body.String())
			}
			if calls != test.calls {
				t.Fatalf("RPC calls = %d, want %d", calls, test.calls)
			}
			if recorder.Header().Get("Access-Control-Allow-Origin") == "*" {
				t.Fatal("public handler emitted wildcard CORS")
			}
		})
	}
}

func TestTypedSoundManifestAllowsOnlyEightCategoriesAndSafeSortedAssets(t *testing.T) {
	t.Parallel()

	service, err := NewConnectService(ConnectServiceConfig{Coordinator: newConnectTestCoordinator(t), Assets: playerAssets()})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		category playerv1.SoundCategory
		want     []string
	}{
		{playerv1.SoundCategory_SOUND_CATEGORY_AMBIENT, []string{"sounds/ambient/HISS.OGG", "sounds/ambient/hum.wav", "sounds/ambient/theme.m4a"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_HACK_GOOD, []string{"sounds/hack-good/good.mp3"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_HACK_BAD, []string{"sounds/hack-bad/bad.wav"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_MENU_FOCUS, []string{"sounds/menu-focus/focus.wav"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_SINGLE, []string{"sounds/single/single.wav"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_MULTIPLE, []string{"sounds/multiple/multiple.wav"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_ENTER, []string{"sounds/enter/enter.wav"}},
		{playerv1.SoundCategory_SOUND_CATEGORY_CHARSCROLL, []string{"sounds/charscroll/scroll.wav"}},
	}
	for _, test := range tests {
		response, err := service.SoundManifest(context.Background(), connect.NewRequest(&playerv1.SoundManifestRequest{Category: test.category}))
		if err != nil {
			t.Fatalf("SoundManifest(%s): %v", test.category, err)
		}
		if !reflect.DeepEqual(response.Msg.Assets, test.want) {
			t.Errorf("SoundManifest(%s) assets = %#v, want %#v", test.category, response.Msg.Assets, test.want)
		}
	}
	for _, invalid := range []playerv1.SoundCategory{playerv1.SoundCategory_SOUND_CATEGORY_UNSPECIFIED, 999} {
		_, err := service.SoundManifest(context.Background(), connect.NewRequest(&playerv1.SoundManifestRequest{Category: invalid}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("SoundManifest(%d) code = %s, want invalid_argument", invalid, connect.CodeOf(err))
		}
	}

	empty, err := NewConnectService(ConnectServiceConfig{Coordinator: newConnectTestCoordinator(t), Assets: fstest.MapFS{}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := empty.SoundManifest(context.Background(), connect.NewRequest(&playerv1.SoundManifestRequest{Category: playerv1.SoundCategory_SOUND_CATEGORY_AMBIENT}))
	if err != nil || len(response.Msg.Assets) != 0 {
		t.Fatalf("missing sound category = %#v, %v; want empty success", response, err)
	}
}

func TestHTTPHandlerServesStaticAssetsAndIndexFallback(t *testing.T) {
	t.Parallel()

	assets := playerAssets()
	handler := NewHTTPHandler(assets)

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		body        string
	}{
		{
			name:        "root index",
			path:        "/",
			status:      http.StatusOK,
			contentType: "text/html",
			body:        "player-shell",
		},
		{
			name:        "JavaScript asset",
			path:        "/client.js",
			status:      http.StatusOK,
			contentType: "javascript",
			body:        "player-client",
		},
		{
			name:        "nested font asset",
			path:        "/fonts/Fixedsys.ttf",
			status:      http.StatusOK,
			contentType: "font/ttf",
			body:        "fake-font",
		},
		{
			name:        "extensionless browser route falls back to index",
			path:        "/terminal/root/status",
			status:      http.StatusOK,
			contentType: "text/html",
			body:        "player-shell",
		},
		{
			name:   "missing asset does not fall back",
			path:   "/missing.js",
			status: http.StatusNotFound,
		},
		{
			name:   "directories are not listed",
			path:   "/sounds/",
			status: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := serveRequest(t, handler, test.path)
			if recorder.Code != test.status {
				t.Fatalf("GET %s status = %d, want %d; body = %q", test.path, recorder.Code, test.status, recorder.Body.String())
			}
			if test.contentType != "" && !strings.Contains(recorder.Header().Get("Content-Type"), test.contentType) {
				t.Errorf("GET %s Content-Type = %q, want it to contain %q", test.path, recorder.Header().Get("Content-Type"), test.contentType)
			}
			if test.body != "" && !strings.Contains(recorder.Body.String(), test.body) {
				t.Errorf("GET %s body = %q, want it to contain %q", test.path, recorder.Body.String(), test.body)
			}
		})
	}
}

func TestHTTPHandlerRejectsTraversalWithoutNormalizingItIntoAnAsset(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(playerAssets())
	for _, requestPath := range []string{
		"/../outside.txt",
		"/%2e%2e/outside.txt",
		"/sounds/ambient/../../../outside.txt",
		"/sounds/%2e%2e",
	} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := serveRequest(t, handler, requestPath)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("GET %s status = %d, want 404; body = %q", requestPath, recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "outside-client-root") {
				t.Fatalf("GET %s exposed a normalized asset: %q", requestPath, recorder.Body.String())
			}
		})
	}
}

func TestHTTPHandlerSetsPlayerSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(playerAssets())
	for _, requestPath := range []string{"/", "/terminal/root", "/sounds/ambient/hum.wav"} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := serveRequest(t, handler, requestPath)
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", requestPath, recorder.Code)
			}
			if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			policy := recorder.Header().Get("Content-Security-Policy")
			for _, directive := range []string{"default-src 'self'", "connect-src 'self'", "media-src 'self'", "object-src 'none'"} {
				if !strings.Contains(policy, directive) {
					t.Errorf("Content-Security-Policy = %q, want directive %q", policy, directive)
				}
			}
		})
	}
}

func TestBrowserRecognitionNeverUsesHTTPURLsOrWeakensOriginAndHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(playerAssets())
	const secretToken = "opaque-browser-token-that-must-not-be-reflected"

	for _, requestPath := range []string{
		"/api/session",
		"/api/token",
		"/api/browser-token",
		"/api/identity",
	} {
		recorder := serveRequest(t, handler, requestPath)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want no recognition endpoint", requestPath, recorder.Code)
		}
	}

	for _, requestPath := range []string{
		"/?browserToken=" + secretToken,
		"/client.js?token=" + secretToken,
		"/terminal/root?session=" + secretToken,
	} {
		recorder := serveRequest(t, handler, requestPath)
		serialized := recorder.Body.String() + recorder.Header().Get("Location") + recorder.Header().Get("Set-Cookie")
		if strings.Contains(serialized, secretToken) {
			t.Errorf("GET %s reflected recognition material in an HTTP response", requestPath)
		}
		if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("GET %s lost nosniff", requestPath)
		}
		if policy := recorder.Header().Get("Content-Security-Policy"); policy != playerContentSecurityPolicy {
			t.Errorf("GET %s CSP changed: %q", requestPath, policy)
		}
	}

	for _, test := range []struct {
		origin string
		want   bool
	}{
		{origin: "", want: true},
		{origin: "https://player.test", want: true},
		{origin: "http://player.test", want: true},
		{origin: "https://evil.example", want: false},
	} {
		request := httptest.NewRequest(http.MethodGet, "http://player.test/", nil)
		request.Host = "player.test"
		if test.origin != "" {
			request.Header.Set("Origin", test.origin)
		}
		if got := sameHostOrigin(request); got != test.want {
			t.Errorf("sameHostOrigin(%q) = %t, want %t", test.origin, got, test.want)
		}
	}

	clientScript, err := os.ReadFile(filepath.Join("..", "..", "client", "client.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(clientScript)
	start := strings.Index(js, "const playerTransport = createConnectTransport(")
	end := strings.Index(js, "const playerRPC = createClient(")
	if start < 0 || end <= start {
		t.Fatal("player script is missing the same-origin Connect transport boundary")
	}
	urlBoundary := js[start:end]
	for _, forbidden := range []string{"browserToken", "PLAYER_TOKEN_KEY", "searchParams", "?token", "?session"} {
		if strings.Contains(urlBoundary, forbidden) {
			t.Errorf("Connect base URL construction exposes recognition material through %q", forbidden)
		}
	}
}

func TestHTTPHandlerReturnsNotFoundWhenRequiredAssetsAreMissing(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(fstest.MapFS{})
	for _, requestPath := range []string{"/", "/terminal/root", "/client.js"} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := serveRequest(t, handler, requestPath)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("GET %s status = %d, want 404; body = %q", requestPath, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func playerAssets() fs.FS {
	return fstest.MapFS{
		"index.html":                   {Data: []byte("<!doctype html><title>player-shell</title>")},
		"client.js":                    {Data: []byte("const client = 'player-client';")},
		"fonts/Fixedsys.ttf":           {Data: []byte("fake-font")},
		"outside.txt":                  {Data: []byte("outside-client-root")},
		"sounds/ambient/HISS.OGG":      {Data: []byte("ogg")},
		"sounds/ambient/hum.wav":       {Data: []byte("wav")},
		"sounds/ambient/theme.m4a":     {Data: []byte("m4a")},
		"sounds/ambient/README.txt":    {Data: []byte("not audio")},
		"sounds/charscroll/scroll.wav": {Data: []byte("wav")},
		"sounds/enter/enter.wav":       {Data: []byte("wav")},
		"sounds/hack-bad/bad.wav":      {Data: []byte("wav")},
		"sounds/hack-good/good.mp3":    {Data: []byte("mp3")},
		"sounds/menu-focus/focus.wav":  {Data: []byte("wav")},
		"sounds/menu-focus/empty.txt":  {Data: []byte("not audio")},
		"sounds/multiple/multiple.wav": {Data: []byte("wav")},
		"sounds/single/single.wav":     {Data: []byte("wav")},
	}
}

func serveRequest(t *testing.T, handler http.Handler, requestPath string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "http://player.test"+requestPath, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

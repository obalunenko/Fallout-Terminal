package player

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

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
		"/api/sounds/%2e%2e",
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
	for _, requestPath := range []string{"/", "/terminal/root", "/api/sounds/ambient"} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := serveRequest(t, handler, requestPath)
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", requestPath, recorder.Code)
			}
			if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			policy := recorder.Header().Get("Content-Security-Policy")
			for _, directive := range []string{"default-src 'self'", "connect-src", "ws:", "wss:", "media-src 'self'", "object-src 'none'"} {
				if !strings.Contains(policy, directive) {
					t.Errorf("Content-Security-Policy = %q, want directive %q", policy, directive)
				}
			}
		})
	}
}

func TestHTTPHandlerListsOnlyAllowlistedSoundFiles(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(playerAssets())

	recorder := serveRequest(t, handler, "/api/sounds/ambient")
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET ambient status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var got []string
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode sound response: %v; body = %q", err, recorder.Body.String())
	}
	want := []string{"HISS.OGG", "hum.wav", "theme.m4a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ambient sounds = %#v, want sorted supported filenames %#v", got, want)
	}
	for _, name := range got {
		if strings.ContainsAny(name, `/\\`) {
			t.Errorf("sound response exposed a path instead of a filename: %q", name)
		}
	}
}

func TestHTTPHandlerSoundDiscoveryDegradesToAnEmptyJSONArray(t *testing.T) {
	t.Parallel()

	handler := NewHTTPHandler(playerAssets())
	for _, requestPath := range []string{
		"/api/sounds/not-allowed",
		"/api/sounds/hack-bad",
		"/api/sounds/menu-focus",
	} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := serveRequest(t, handler, requestPath)
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200", requestPath, recorder.Code)
			}
			if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
				t.Fatalf("GET %s body = %q, want exact empty JSON array", requestPath, got)
			}
		})
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
		"index.html":                  {Data: []byte("<!doctype html><title>player-shell</title>")},
		"client.js":                   {Data: []byte("const client = 'player-client';")},
		"fonts/Fixedsys.ttf":          {Data: []byte("fake-font")},
		"outside.txt":                 {Data: []byte("outside-client-root")},
		"sounds/ambient/HISS.OGG":     {Data: []byte("ogg")},
		"sounds/ambient/hum.wav":      {Data: []byte("wav")},
		"sounds/ambient/theme.m4a":    {Data: []byte("m4a")},
		"sounds/ambient/README.txt":   {Data: []byte("not audio")},
		"sounds/hack-good/good.mp3":   {Data: []byte("mp3")},
		"sounds/menu-focus/empty.txt": {Data: []byte("not audio")},
	}
}

func serveRequest(t *testing.T, handler http.Handler, requestPath string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "http://player.test"+requestPath, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

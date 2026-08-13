package player

import (
	"bytes"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const playerContentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; media-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'"

var (
	allowedSoundExtensions = map[string]struct{}{
		".mp3":  {},
		".wav":  {},
		".ogg":  {},
		".m4a":  {},
		".webm": {},
	}
)

// NewHTTPHandler serves a player filesystem rooted at client/. The supplied
// filesystem is the handler's complete namespace; no host filesystem paths are
// opened or derived from requests.
func NewHTTPHandler(assets fs.FS) http.Handler {
	return &playerHTTPHandler{assets: assets}
}

// NewApplicationHandler mounts generated Connect procedures before the static
// player application. RPC paths never fall through to the SPA index, and all
// page, generated client, sound, and RPC traffic remains same-origin.
func NewApplicationHandler(assets fs.FS, rpcPath string, rpcHandler http.Handler) http.Handler {
	staticHandler := NewHTTPHandler(assets)
	boundedRPC := http.MaxBytesHandler(rpcHandler, MaxEncodedBodyBytes)
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if rpcHandler != nil && rpcPath != "" && rpcRequestPath(request.URL.Path, rpcPath) {
			response.Header().Set("X-Content-Type-Options", "nosniff")
			response.Header().Set("Content-Security-Policy", playerContentSecurityPolicy)
			if !validRequestHost(request.Host) || !sameHostOrigin(request) {
				http.Error(response, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			if request.ContentLength > MaxEncodedBodyBytes {
				http.Error(response, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
				return
			}
			boundedRPC.ServeHTTP(response, request)
			return
		}
		staticHandler.ServeHTTP(response, request)
	})
}

func rpcRequestPath(requestPath, servicePath string) bool {
	servicePath = strings.TrimSuffix(servicePath, "/")
	return requestPath == servicePath || strings.HasPrefix(requestPath, servicePath+"/")
}

func validRequestHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "/\\\r\n\t @") {
		return false
	}
	if strings.HasPrefix(host, "[") {
		_, _, err := net.SplitHostPort(host)
		return err == nil || strings.HasSuffix(host, "]")
	}
	if strings.Count(host, ":") > 1 {
		return false
	}
	if strings.Contains(host, ":") {
		name, port, err := net.SplitHostPort(host)
		return err == nil && name != "" && port != ""
	}
	return true
}

// sameHostOrigin accepts non-browser clients without Origin and browser
// clients whose HTTP(S) origin host exactly matches the request Host.
func sameHostOrigin(request *http.Request) bool {
	if request == nil {
		return false
	}
	rawOrigin := strings.TrimSpace(request.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}
	origin, err := url.Parse(rawOrigin)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" {
		return false
	}
	return strings.EqualFold(origin.Host, request.Host)
}

type playerHTTPHandler struct {
	assets fs.FS
}

func (handler *playerHTTPHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	response.Header().Set("Content-Security-Policy", playerContentSecurityPolicy)

	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		http.Error(response, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if unsafePlayerPath(request.URL) {
		http.NotFound(response, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/api/") {
		http.NotFound(response, request)
		return
	}

	handler.serveAsset(response, request)
}

func (handler *playerHTTPHandler) serveAsset(response http.ResponseWriter, request *http.Request) {
	assetPath := strings.TrimPrefix(request.URL.Path, "/")
	if assetPath == "" {
		assetPath = "index.html"
	} else if strings.HasSuffix(assetPath, "/") {
		http.NotFound(response, request)
		return
	}

	if handler.serveExistingFile(response, request, assetPath) {
		return
	}
	if path.Ext(assetPath) == "" && handler.serveExistingFile(response, request, "index.html") {
		return
	}
	http.NotFound(response, request)
}

func (handler *playerHTTPHandler) serveExistingFile(response http.ResponseWriter, request *http.Request, name string) bool {
	if handler.assets == nil {
		return false
	}
	info, err := fs.Stat(handler.assets, name)
	if err != nil || info.IsDir() {
		return false
	}
	contents, err := fs.ReadFile(handler.assets, name)
	if err != nil {
		return false
	}
	http.ServeContent(response, request, name, info.ModTime(), bytes.NewReader(contents))
	return true
}

func unsafePlayerPath(requestURL *url.URL) bool {
	for _, requestPath := range []string{requestURL.Path, requestURL.RawPath} {
		if requestPath == "" {
			continue
		}
		decoded, err := url.PathUnescape(requestPath)
		if err != nil || strings.Contains(decoded, `\`) {
			return true
		}
		for _, segment := range strings.Split(decoded, "/") {
			if segment == "." || segment == ".." {
				return true
			}
		}
	}
	return false
}

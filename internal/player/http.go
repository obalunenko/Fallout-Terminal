package player

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
)

const playerContentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' data:; media-src 'self'; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'"

var (
	allowedSoundFolders = map[string]struct{}{
		"ambient":    {},
		"hack-good":  {},
		"hack-bad":   {},
		"menu-focus": {},
		"single":     {},
		"multiple":   {},
		"enter":      {},
		"charscroll": {},
	}
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
	if strings.HasPrefix(request.URL.Path, "/api/sounds/") {
		handler.serveSoundList(response, request)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/api/") {
		http.NotFound(response, request)
		return
	}

	handler.serveAsset(response, request)
}

func (handler *playerHTTPHandler) serveSoundList(response http.ResponseWriter, request *http.Request) {
	folder := strings.TrimPrefix(request.URL.Path, "/api/sounds/")
	if _, allowed := allowedSoundFolders[folder]; !allowed || strings.Contains(folder, "/") {
		writeSoundList(response, nil)
		return
	}

	files := make([]string, 0)
	if handler.assets != nil {
		entries, err := fs.ReadDir(handler.assets, "sounds/"+folder)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if _, allowed := allowedSoundExtensions[strings.ToLower(path.Ext(entry.Name()))]; allowed {
					files = append(files, entry.Name())
				}
			}
		}
	}
	sort.Strings(files)
	writeSoundList(response, files)
}

func writeSoundList(response http.ResponseWriter, files []string) {
	if files == nil {
		files = make([]string, 0)
	}
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(response).Encode(files)
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

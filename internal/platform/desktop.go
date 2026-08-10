package platform

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var errDesktopNotReady = errors.New("desktop runtime is not ready")

// Desktop is the narrow Wails-backed implementation used for native session
// dialogs and external HTTP(S) links. It retains only the Wails application
// context and exposes no general filesystem or process surface.
type Desktop struct {
	mu  sync.RWMutex
	ctx context.Context
}

// NewDesktop constructs a wrapper without acquiring resources. A non-nil
// context may be supplied by tests or composition code that already started.
func NewDesktop(ctx context.Context) *Desktop {
	return &Desktop{ctx: ctx}
}

// Ready installs the Wails application context.
func (desktop *Desktop) Ready(ctx context.Context) error {
	if ctx == nil {
		return errDesktopNotReady
	}
	desktop.mu.Lock()
	desktop.ctx = ctx
	desktop.mu.Unlock()
	return nil
}

// Close releases the retained desktop context. It is idempotent.
func (desktop *Desktop) Close(context.Context) error {
	desktop.mu.Lock()
	desktop.ctx = nil
	desktop.mu.Unlock()
	return nil
}

// OpenFile shows a native JSON session picker. An absent suggested directory
// is never created merely to display the dialog; the nearest existing ancestor
// is used as the native starting point.
func (desktop *Desktop) OpenFile(defaultPath string) (string, error) {
	ctx, err := desktop.context()
	if err != nil {
		return "", err
	}
	directory, filename := dialogLocation(defaultPath, false)
	return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:            "Open Fallout Terminal Session",
		DefaultDirectory: directory,
		DefaultFilename:  filename,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Fallout Terminal sessions (*.json)", Pattern: "*.json"},
		},
		ResolvesAliases: true,
	})
}

// SaveFile shows a native JSON session destination picker.
func (desktop *Desktop) SaveFile(defaultPath string) (string, error) {
	ctx, err := desktop.context()
	if err != nil {
		return "", err
	}
	directory, filename := dialogLocation(defaultPath, true)
	return wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:                "Save Fallout Terminal Session",
		DefaultDirectory:     directory,
		DefaultFilename:      filename,
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Fallout Terminal sessions (*.json)", Pattern: "*.json"},
		},
	})
}

// OpenURL opens only absolute HTTP(S) URLs in the system browser.
func (desktop *Desktop) OpenURL(rawURL string) error {
	ctx, err := desktop.context()
	if err != nil {
		return err
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("external URL must be an absolute HTTP or HTTPS URL")
	}
	wailsruntime.BrowserOpenURL(ctx, parsed.String())
	return nil
}

func (desktop *Desktop) context() (context.Context, error) {
	desktop.mu.RLock()
	defer desktop.mu.RUnlock()
	if desktop.ctx == nil {
		return nil, errDesktopNotReady
	}
	return desktop.ctx, nil
}

func dialogLocation(path string, pathIncludesFilename bool) (directory, filename string) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || path == "" {
		return "", ""
	}
	if pathIncludesFilename {
		directory = filepath.Dir(path)
		filename = filepath.Base(path)
	} else {
		directory = path
	}
	for directory != "" && directory != "." {
		info, err := os.Stat(directory)
		if err == nil && info.IsDir() {
			return directory, filename
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", filename
}

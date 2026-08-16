// Package platform contains operating-system integration that is kept outside
// the domain and persistence packages.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	documentsDirectoryName   = "Documents"
	sessionsDirectoryName    = "Sessions"
	applicationSupportName   = "Application Support"
	applicationIdentifier    = "com.vaulttec.fallout-terminal"
	productDirectoryName     = "Fallout Terminal"
	bundledSessionsDirectory = "sessions"
	bundledDemoFilename      = "demo.json"
	publicAccessFilename     = "public-access.json"
)

// SessionLocations separates user-owned session documents, the immutable
// bundled sample, and private application metadata.
//
// Resolving these locations has no filesystem side effects. In particular,
// DocumentsDefault is created only after a native save dialog is confirmed.
type SessionLocations struct {
	DocumentsDefault   string
	BundledDemo        string
	ApplicationSupport string
}

// PublicAccessSettingsPath resolves the separate version-1 non-secret settings file.
// It has no filesystem side effects and never points into session or player-config storage.
func PublicAccessSettingsPath(applicationSupportDirectory string) (string, error) {
	directory, err := cleanAbsolutePath("application support directory", applicationSupportDirectory)
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, publicAccessFilename), nil
}

// DefaultSessionLocations resolves locations for the current user beneath
// resourceRoot. In a packaged build resourceRoot is the app's Contents/Resources
// directory; during development it is the repository root.
func DefaultSessionLocations(resourceRoot string) (SessionLocations, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return SessionLocations{}, fmt.Errorf("resolve user home directory: %w", err)
	}
	return NewSessionLocations(homeDirectory, resourceRoot)
}

// NewSessionLocations resolves deterministic session paths without touching
// the filesystem. Both inputs must be absolute so a later working-directory
// change cannot redirect user data or the bundled read-only sample.
func NewSessionLocations(homeDirectory, resourceRoot string) (SessionLocations, error) {
	homeDirectory, err := cleanAbsolutePath("home directory", homeDirectory)
	if err != nil {
		return SessionLocations{}, err
	}
	resourceRoot, err = cleanAbsolutePath("resource root", resourceRoot)
	if err != nil {
		return SessionLocations{}, err
	}

	return SessionLocations{
		DocumentsDefault:   filepath.Join(homeDirectory, documentsDirectoryName, productDirectoryName, sessionsDirectoryName),
		BundledDemo:        filepath.Join(resourceRoot, bundledSessionsDirectory, bundledDemoFilename),
		ApplicationSupport: filepath.Join(homeDirectory, "Library", applicationSupportName, applicationIdentifier),
	}, nil
}

func cleanAbsolutePath(name, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s is empty", name)
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("%s must be absolute", name)
	}
	return cleaned, nil
}

// dialogLocation resolves a suggested native-dialog location without creating
// directories. Missing paths fall back to the nearest existing ancestor.
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

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestValidateProductionResources(t *testing.T) {
	demo := filepath.Join(t.TempDir(), "demo.json")
	require.NoError(t, os.WriteFile(demo, []byte("{}"), 0o600))

	completePlayer := fs.FS(fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html>")},
	})

	t.Run("complete package", func(t *testing.T) {
		require.NoError(t, validateProductionResources(completePlayer, demo))
	})

	t.Run("missing player index", func(t *testing.T) {
		err := validateProductionResources(fstest.MapFS{".keep": {}}, demo)
		require.EqualError(t, err, "player assets are incomplete: index.html is unavailable")
	})

	t.Run("missing bundled demo", func(t *testing.T) {
		err := validateProductionResources(completePlayer, filepath.Join(t.TempDir(), "missing.json"))
		require.ErrorContains(t, err, "bundled demo is unavailable")
	})

	t.Run("empty bundled demo", func(t *testing.T) {
		emptyDemo := filepath.Join(t.TempDir(), "demo.json")
		require.NoError(t, os.WriteFile(emptyDemo, nil, 0o600))
		err := validateProductionResources(completePlayer, emptyDemo)
		require.ErrorContains(t, err, "bundled demo is unavailable")
	})

	t.Run("player index must be a regular file", func(t *testing.T) {
		err := validateProductionResources(fstest.MapFS{"index.html": {Mode: fs.ModeDir}}, demo)
		require.ErrorContains(t, err, "player assets are incomplete")
	})
}

func TestWailsV3HostKeepsOverseerAndClientResourceBoundariesSeparate(t *testing.T) {
	root, err := os.Getwd()
	require.NoError(t, err)

	mainSource, err := os.ReadFile(filepath.Join(root, "main.go"))
	require.NoError(t, err)
	hostSource, err := os.ReadFile(filepath.Join(root, "wails_host.go"))
	require.NoError(t, err)

	mainText := string(mainSource)
	hostText := string(hostSource)
	require.Contains(t, mainText, `fs.Sub(overseerSource, "frontend/overseer/dist")`)
	require.Contains(t, mainText, `fs.Sub(clientSource, "frontend/client/dist")`)
	require.Contains(t, mainText, "composeApplication(host, clientAssets)")
	require.Contains(t, hostText, "application.AssetFileServerFS(overseerAssets)")
	require.Contains(t, hostText, "newDesktopService(core)")
	require.NotContains(t, hostText, "clientAssets")
	require.Equal(t, 1, strings.Count(hostText, "host.Window.NewWithOptions("))
}

func TestApplicationResourceRootUsesPackagedContentsResources(t *testing.T) {
	packagedExecutable := filepath.Join(string(filepath.Separator), "Applications", "Fallout Terminal.app", "Contents", "MacOS", "Fallout Terminal")
	packagedRoot := applicationResourceRootFor(packagedExecutable, filepath.Join(string(filepath.Separator), "unrelated", "working-directory"))
	require.Equal(t,
		filepath.Join(string(filepath.Separator), "Applications", "Fallout Terminal.app", "Contents", "Resources"),
		packagedRoot,
	)

	developmentRoot := filepath.Join(string(filepath.Separator), "checkout", "Fallout-Terminal")
	require.Equal(t, developmentRoot, applicationResourceRootFor(filepath.Join(developmentRoot, "build", "bin", "Fallout Terminal"), developmentRoot))
}

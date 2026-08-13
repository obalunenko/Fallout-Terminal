//go:build !bindings

package main

import (
	"io/fs"
	"os"
	"path/filepath"
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
}

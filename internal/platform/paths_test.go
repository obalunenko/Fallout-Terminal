package platform

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicAccessSettingsPathUsesApplicationSupportWithoutSideEffects(t *testing.T) {
	t.Parallel()

	locations, err := NewSessionLocations("/Users/player", "/Applications/Fallout Terminal.app/Contents/Resources")
	require.NoError(t, err)
	path, err := PublicAccessSettingsPath(locations.ApplicationSupport)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(
		"/Users/player", "Library", "Application Support", "com.vaulttec.fallout-terminal", "public-access.json",
	), path)

	_, err = PublicAccessSettingsPath("relative/application-support")
	require.Error(t, err)
}

package tunnel_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const settingsPath = "/private/application-support/public-access.json"

func TestPublicAccessSettingsMissingFileReturnsSafeDisabledDefaults(t *testing.T) {
	t.Parallel()

	store := tunnel.NewPublicAccessSettingsStore(settingsPath, testutil.NewFakeFileSystem(), testutil.NewFakeClock(time.Unix(100, 0)))
	preferences, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, tunnel.DefaultPublicAccessPreferences(), preferences)
	assert.False(t, preferences.EnabledPreference, "saved preference never auto-starts an endpoint")
}

func TestPublicAccessSettingsRoundTripExactVersionOneSecretFreeJSON(t *testing.T) {
	t.Parallel()

	filesystem := testutil.NewFakeFileSystem()
	store := tunnel.NewPublicAccessSettingsStore(settingsPath, filesystem, testutil.NewFakeClock(time.Unix(100, 0)))
	preferences := tunnel.PublicAccessPreferences{
		Version: 1, EnabledPreference: true, ReservedDomain: "Vault.Example.", Username: " players ",
		ProviderTokenPresentHint: true, PlayerPasswordPresentHint: true, Revision: 7,
	}
	require.NoError(t, store.Save(preferences))

	raw, ok := filesystem.File(settingsPath)
	require.True(t, ok)
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &document))
	assert.ElementsMatch(t, []string{
		"version", "enabledPreference", "reservedDomain", "username",
		"providerTokenPresentHint", "playerPasswordPresentHint", "revision",
	}, mapKeys(document))
	for _, forbidden := range []string{"token", "password", "authtoken", "credential", "secret"} {
		assert.NotContains(t, strings.ToLower(string(raw)), `"`+forbidden+`"`)
	}

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "vault.example", loaded.ReservedDomain)
	assert.Equal(t, "players", loaded.Username)
	assert.True(t, loaded.EnabledPreference, "enabled preference is presentation only")
	assert.Equal(t, uint64(7), loaded.Revision)

	directoryMode, ok := filesystem.Mode(filepath.Dir(settingsPath))
	require.True(t, ok)
	assert.Equal(t, 0o700, int(directoryMode.Perm()))
	fileMode, ok := filesystem.Mode(settingsPath)
	require.True(t, ok)
	assert.Equal(t, 0o600, int(fileMode.Perm()))
	assert.Len(t, filesystem.RenameCalls(), 1, "save uses one same-directory atomic rename")
	for _, path := range filesystem.Paths() {
		assert.True(t, strings.HasPrefix(path, filepath.Dir(settingsPath)+string(filepath.Separator)))
		assert.NotContains(t, path, "session")
		assert.NotContains(t, path, "player-config")
	}
}

func TestPublicAccessSettingsQuarantinesMalformedUnknownAndFutureDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "malformed", raw: `{"version":`},
		{name: "unknown field", raw: `{"version":1,"username":"players","surprise":true}`},
		{name: "future version", raw: `{"version":2,"username":"players"}`},
		{name: "invalid domain", raw: `{"version":1,"username":"players","reservedDomain":"https://vault.example"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			filesystem := testutil.NewFakeFileSystem()
			filesystem.SeedFile(settingsPath, []byte(test.raw))
			store := tunnel.NewPublicAccessSettingsStore(settingsPath, filesystem, testutil.NewFakeClock(time.Unix(100, 0)))
			preferences, err := store.Load()
			require.ErrorIs(t, err, tunnel.ErrSettingsRecovered)
			assert.NotContains(t, err.Error(), test.raw)
			assert.Equal(t, tunnel.DefaultPublicAccessPreferences(), preferences)
			_, exists := filesystem.File(settingsPath)
			assert.False(t, exists)
			paths := filesystem.Paths()
			require.Len(t, paths, 1)
			assert.Contains(t, filepath.Base(paths[0]), "public-access.corrupt-")
			mode, ok := filesystem.Mode(paths[0])
			require.True(t, ok)
			assert.Equal(t, 0o600, int(mode.Perm()))
		})
	}
}

func TestPublicAccessSettingsAtomicFailureRemovesTemporaryFile(t *testing.T) {
	t.Parallel()

	filesystem := testutil.NewFakeFileSystem()
	temporary := filepath.Join(filepath.Dir(settingsPath), ".public-access-000001")
	filesystem.RenameErrors[temporary] = errors.New("injected rename failure")
	store := tunnel.NewPublicAccessSettingsStore(settingsPath, filesystem, testutil.NewFakeClock(time.Unix(100, 0)))
	err := store.Save(tunnel.DefaultPublicAccessPreferences())
	require.Error(t, err)
	assert.NotContains(t, err.Error(), temporary)
	_, exists := filesystem.File(temporary)
	assert.False(t, exists)
	_, exists = filesystem.File(settingsPath)
	assert.False(t, exists)
}

func mapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

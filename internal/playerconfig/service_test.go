package playerconfig

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCreateLoadSaveAndReusePlayerConfig(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	target := "/Campaigns/shared/vault-13-players.json"
	dialog := &testutil.FakeDialog{SaveResult: target}
	service := NewService(NewStorage(fs), dialog, "/Campaigns")

	created := service.Create(t.Context())
	require.Falsef(t, !created.OK || created.Canceled || created.FilePath != target || created.Config == nil,
		"Create() = %#v", created)
	require.Falsef(t, created.Config.Version != 1 || created.Config.Name != "vault-13-players" || len(created.Config.Roster) != 0,
		"created config = %#v", created.Config)
	require.False(t, created.Config.Roster == nil,
		"created config roster is nil, want a non-nil empty array")

	stored, ok := fs.File(target)
	require.Falsef(t, !ok,
		"created config was not written to %q", target)
	require.Falsef(t, !strings.Contains(string(stored), `"roster": []`),
		"created config does not encode an empty roster array: %s", stored)

	handle := domain.PlayerConfigHandle{Path: target, Version: 1, Name: created.Config.Name}
	wantRoster := []domain.CharacterRosterEntry{
		{ID: "mara", Name: "Mara"},
		{ID: "boone", Name: "Boone"},
		{ID: "arcade", Name: "Arcade"},
	}
	{
		err := service.Save(handle, wantRoster)
		require.Falsef(t, err != nil,
			"Save() error = %v", err)
	}

	reference, err := RelativeReference("/Campaigns/sessions/game.json", target)
	if err != nil {
		require.NoError(t, err)
	}
	{
		want := filepath.Join("..", "shared", "vault-13-players.json")
		require.Falsef(t, reference != want,
			"reference = %q, want %q", reference, want)
	}

	loaded := service.LoadReferenced("/Campaigns/sessions/game.json", reference)
	require.Falsef(t, !loaded.OK || loaded.Config == nil || !cmp.Equal(loaded.Config.Roster, wantRoster),
		"LoadReferenced() = %#v", loaded)

	dialog.OpenResult = target
	opened := service.Open(t.Context())
	require.Falsef(t, !opened.OK || opened.Config == nil || !cmp.Equal(opened.Config.Roster, wantRoster),
		"Open() shared config = %#v", opened)

}

func TestPlayerConfigCancellationAndFailuresAreNonMutating(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	service := NewService(NewStorage(fs), &testutil.FakeDialog{}, "/Campaigns")
	{
		result := service.Create(t.Context())
		require.Falsef(t, !result.Canceled || result.OK,
			"canceled Create() = %#v", result)
	}
	{

		result := service.Open(t.Context())
		require.Falsef(t, !result.Canceled || result.OK,
			"canceled Open() = %#v", result)
	}
	{

		writes := fs.WriteCalls()
		require.Falsef(t, len(writes) != 0,
			"cancellation wrote files: %#v", writes)
	}

	fs.SeedFile("/Campaigns/invalid.json", []byte(`{"version":1,"name":"Players","roster":[{"id":"","name":"Mara"}]}`))
	result := service.LoadReferenced("/Campaigns/game.json", "invalid.json")
	require.Falsef(t, result.OK || result.Error == "" || result.Config != nil,
		"invalid LoadReferenced() = %#v", result)

}

func TestCompleteCandidateSaveFailurePublishesNoSuccessfulConfig(t *testing.T) {
	t.Parallel()
	store := &failingPlayerConfigStore{err: errors.New("disk full")}
	service := NewService(store, &testutil.FakeDialog{SaveResult: "/Campaigns/players.json"}, "/Campaigns")
	created := service.Create(t.Context())
	require.Falsef(t, created.OK || created.Config != nil || created.FilePath != "" || store.writes != 1,
		"failed atomic create published state: result=%#v writes=%d", created, store.writes)

	err := service.Save(domain.PlayerConfigHandle{Path: "/Campaigns/players.json", Version: 1, Name: "Players"}, []domain.CharacterRosterEntry{{ID: "mara", Name: "Mara"}})
	require.Falsef(t, err == nil || store.writes != 2,
		"failed atomic save err=%v writes=%d", err, store.writes)

}

type failingPlayerConfigStore struct {
	err    error
	writes int
}

func (*failingPlayerConfigStore) Read(string) ([]byte, error) { return nil, errors.New("not found") }
func (store *failingPlayerConfigStore) WriteAtomic(string, []byte) error {
	store.writes++
	return store.err
}

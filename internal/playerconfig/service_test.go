package playerconfig

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
)

func TestCreateLoadSaveAndReusePlayerConfig(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	target := "/Campaigns/shared/vault-13-players.json"
	dialog := &testutil.FakeDialog{SaveResult: target}
	service := NewService(NewStorage(fs), dialog, "/Campaigns")

	created := service.Create(context.Background())
	if !created.OK || created.Canceled || created.FilePath != target || created.Config == nil {
		t.Fatalf("Create() = %#v", created)
	}
	if created.Config.Version != 1 || created.Config.Name != "vault-13-players" || len(created.Config.Roster) != 0 {
		t.Fatalf("created config = %#v", created.Config)
	}
	if created.Config.Roster == nil {
		t.Fatal("created config roster is nil, want a non-nil empty array")
	}
	stored, ok := fs.File(target)
	if !ok {
		t.Fatalf("created config was not written to %q", target)
	}
	if !strings.Contains(string(stored), `"roster": []`) {
		t.Fatalf("created config does not encode an empty roster array: %s", stored)
	}

	handle := domain.PlayerConfigHandle{Path: target, Version: 1, Name: created.Config.Name}
	wantRoster := []domain.CharacterRosterEntry{
		{ID: "mara", Name: "Mara"},
		{ID: "boone", Name: "Boone"},
		{ID: "arcade", Name: "Arcade"},
	}
	if err := service.Save(handle, wantRoster); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reference, err := RelativeReference("/Campaigns/sessions/game.json", target)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join("..", "shared", "vault-13-players.json"); reference != want {
		t.Fatalf("reference = %q, want %q", reference, want)
	}
	loaded := service.LoadReferenced("/Campaigns/sessions/game.json", reference)
	if !loaded.OK || loaded.Config == nil || !reflect.DeepEqual(loaded.Config.Roster, wantRoster) {
		t.Fatalf("LoadReferenced() = %#v", loaded)
	}

	dialog.OpenResult = target
	opened := service.Open(context.Background())
	if !opened.OK || opened.Config == nil || !reflect.DeepEqual(opened.Config.Roster, wantRoster) {
		t.Fatalf("Open() shared config = %#v", opened)
	}
}

func TestPlayerConfigCancellationAndFailuresAreNonMutating(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	service := NewService(NewStorage(fs), &testutil.FakeDialog{}, "/Campaigns")
	if result := service.Create(context.Background()); !result.Canceled || result.OK {
		t.Fatalf("canceled Create() = %#v", result)
	}
	if result := service.Open(context.Background()); !result.Canceled || result.OK {
		t.Fatalf("canceled Open() = %#v", result)
	}
	if writes := fs.WriteCalls(); len(writes) != 0 {
		t.Fatalf("cancellation wrote files: %#v", writes)
	}

	fs.SeedFile("/Campaigns/invalid.json", []byte(`{"version":1,"name":"Players","roster":[{"id":"","name":"Mara"}]}`))
	result := service.LoadReferenced("/Campaigns/game.json", "invalid.json")
	if result.OK || result.Error == "" || result.Config != nil {
		t.Fatalf("invalid LoadReferenced() = %#v", result)
	}
}

func TestCompleteCandidateSaveFailurePublishesNoSuccessfulConfig(t *testing.T) {
	t.Parallel()
	store := &failingPlayerConfigStore{err: errors.New("disk full")}
	service := NewService(store, &testutil.FakeDialog{SaveResult: "/Campaigns/players.json"}, "/Campaigns")
	created := service.Create(context.Background())
	if created.OK || created.Config != nil || created.FilePath != "" || store.writes != 1 {
		t.Fatalf("failed atomic create published state: result=%#v writes=%d", created, store.writes)
	}
	err := service.Save(domain.PlayerConfigHandle{Path: "/Campaigns/players.json", Version: 1, Name: "Players"}, []domain.CharacterRosterEntry{{ID: "mara", Name: "Mara"}})
	if err == nil || store.writes != 2 {
		t.Fatalf("failed atomic save err=%v writes=%d", err, store.writes)
	}
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

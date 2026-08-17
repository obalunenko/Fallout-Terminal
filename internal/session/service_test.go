package session

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
)

var testLocations = Locations{
	DocumentsDefault:   "/Users/test/Documents/Fallout Terminal/Sessions",
	BundledDemo:        "/Applications/Fallout Terminal.app/Contents/Resources/sessions/demo.json",
	ApplicationSupport: "/Users/test/Library/Application Support/com.vaulttec.fallout-terminal",
}

func TestCanceledSessionDialogsDoNotChangeStateOrFilesystem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		invoke      func(context.Context, *Service) SessionResult
		wantKind    string
		wantDefault string
	}{
		{
			name:        "new session",
			invoke:      func(ctx context.Context, service *Service) SessionResult { return service.Create(ctx) },
			wantKind:    "save",
			wantDefault: filepath.Join(testLocations.DocumentsDefault, "session.json"),
		},
		{
			name:        "open session",
			invoke:      func(ctx context.Context, service *Service) SessionResult { return service.Open(ctx) },
			wantKind:    "open",
			wantDefault: testLocations.DocumentsDefault,
		},
		{
			name:        "copy bundled demo",
			invoke:      func(ctx context.Context, service *Service) SessionResult { return service.CopyDemo(ctx) },
			wantKind:    "save",
			wantDefault: filepath.Join(testLocations.DocumentsDefault, "demo.json"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fileSystem := testutil.NewFakeFileSystem()
			dialog := &testutil.FakeDialog{}
			service := NewService(NewStorage(fileSystem), dialog, testLocations)
			t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

			result := test.invoke(context.Background(), service)
			if result.OK || !result.Canceled || result.Error != "" || result.FilePath != "" || result.Session != nil {
				t.Fatalf("canceled result = %#v", result)
			}
			if got, want := dialog.Calls(), []testutil.DialogCall{{Kind: test.wantKind, DefaultPath: test.wantDefault}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("dialog calls = %#v, want %#v", got, want)
			}
			assertInactive(t, service.Snapshot())
			if calls := fileSystem.MkdirCalls(); len(calls) != 0 {
				t.Fatalf("cancellation created directories: %#v", calls)
			}
			if calls := fileSystem.WriteCalls(); len(calls) != 0 {
				t.Fatalf("cancellation wrote files: %#v", calls)
			}
		})
	}
}

func TestCreateUsesDocumentsSuggestionAndActivatesChosenPathAfterWrite(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/My Wasteland.json"
	dialog := &testutil.FakeDialog{SaveResult: target}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	result := service.Create(context.Background())
	if !result.OK || result.Canceled || result.Error != "" || result.FilePath != target || result.Session == nil {
		t.Fatalf("Create() = %#v", result)
	}
	if result.Session.Name != "My Wasteland" || result.Session.Version != 1 || len(result.Session.Terminals) != 1 {
		t.Fatalf("created session = %#v", result.Session)
	}
	if err := domain.ValidateSession(*result.Session); err != nil {
		t.Fatalf("created session is invalid: %v", err)
	}
	written, ok := fileSystem.File(target)
	if !ok {
		t.Fatalf("chosen target %q was not written", target)
	}
	if !bytes.HasSuffix(written, []byte("\n")) || bytes.Contains(written, []byte("\t")) {
		t.Fatalf("created JSON is not human-readable with a final newline:\n%s", written)
	}
	snapshot := service.Snapshot()
	if snapshot.Path != target || snapshot.Session == nil || snapshot.Session.Name != "My Wasteland" {
		t.Fatalf("active session = %#v", snapshot)
	}
	assertNoApplicationSupportWrites(t, fileSystem)
}

func TestInvalidOpenRetainsPreviousActiveSessionAndPath(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	validPath := "/Volumes/Campaigns/valid.json"
	invalidPath := "/Volumes/Campaigns/invalid.json"
	validData, err := domain.EncodeSession(validSession("safe"))
	if err != nil {
		t.Fatal(err)
	}
	fileSystem.SeedFile(validPath, validData)
	fileSystem.SeedFile(invalidPath, []byte(`{"version":2,"name":"bad","terminals":[]}`))
	dialog := &testutil.FakeDialog{OpenResult: validPath}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil || opened.FilePath != validPath {
		t.Fatalf("first Open() = %#v", opened)
	}
	dialog.OpenResult = invalidPath
	failed := service.Open(context.Background())
	if failed.OK || failed.Canceled || failed.Error == "" || failed.Session != nil {
		t.Fatalf("invalid Open() = %#v", failed)
	}
	if strings.Contains(failed.Error, string(fileSystemFileData(t, fileSystem, invalidPath))) {
		t.Fatalf("Open() error disclosed file contents: %q", failed.Error)
	}
	snapshot := service.Snapshot()
	if snapshot.Path != validPath || snapshot.Session == nil || snapshot.Session.Name != "safe" {
		t.Fatalf("failed open replaced active state: %#v", snapshot)
	}
}

func TestOpenAndSavePreserveUnknownFieldsAtExplicitPath(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/forward-compatible.json"
	raw := []byte(`{
  "version": 1,
  "name": "before",
  "campaignNote": {"keep": true},
  "terminals": [{
    "id": "t1",
    "name": "Terminal",
    "hackLevel": 0,
    "introText": "",
    "terminalNote": 42,
    "root": {"id":"root","type":"folder","name":"ROOT","children":[],"nodeNote":[1,2]}
  }]
}`)
	fileSystem.SeedFile(target, raw)
	dialog := &testutil.FakeDialog{OpenResult: target}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil {
		t.Fatalf("Open() = %#v", opened)
	}
	edited := *opened.Session
	edited.Name = "after"
	saved := service.Save(context.Background(), edited, 1)
	if !saved.OK || saved.Error != "" || saved.RequestedRevision != 1 || saved.SavedRevision != 1 {
		t.Fatalf("Save() = %#v", saved)
	}
	written := fileSystemFileData(t, fileSystem, target)
	for _, field := range []string{"campaignNote", "terminalNote", "nodeNote"} {
		if !bytes.Contains(written, []byte(`"`+field+`"`)) {
			t.Errorf("save dropped unknown field %q:\n%s", field, written)
		}
	}
	decoded, err := domain.DecodeSession(written)
	if err != nil {
		t.Fatalf("saved document does not reopen: %v", err)
	}
	if decoded.Name != "after" {
		t.Fatalf("saved name = %q, want after", decoded.Name)
	}
	for _, rename := range fileSystem.RenameCalls() {
		if rename.NewPath != target {
			t.Fatalf("autosave moved explicit target: %#v", rename)
		}
	}
	assertNoApplicationSupportWrites(t, fileSystem)
}

func TestOpenAndSaveLegacyVersionOnePreservesOrdinaryContentWithoutAddingStateFields(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/legacy-ordinary.json"
	raw := []byte(`{
  "version": 1,
  "name": "Legacy ordinary",
  "futureSession": {"keep": true},
  "terminals": [{
    "id": "t1",
    "name": "Terminal",
    "hackLevel": 0,
    "introText": "",
    "futureTerminal": 17,
    "root": {
      "id": "root",
      "type": "folder",
      "name": "ROOT",
      "children": [{
        "id": "status",
        "type": "command",
        "name": "Read status",
        "text": "All systems nominal.",
        "futureCommand": "keep"
      }]
    }
  }]
}`)
	fileSystem.SeedFile(target, raw)
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil {
		t.Fatalf("Open() = %#v", opened)
	}
	ordinary := opened.Session.Terminals[0].Root.Children[0]
	if ordinary.StateChange != nil || len(opened.Session.Terminals[0].CommandStates) != 0 {
		t.Fatalf("legacy open synthesized state fields: command=%#v states=%#v", ordinary, opened.Session.Terminals[0].CommandStates)
	}

	saved := service.Save(context.Background(), *opened.Session, 1)
	if !saved.OK || saved.RequestedRevision != 1 || saved.SavedRevision != 1 {
		t.Fatalf("Save() = %#v", saved)
	}
	written := fileSystemFileData(t, fileSystem, target)
	for _, absent := range []string{`"stateChange"`, `"commandStates"`} {
		if bytes.Contains(written, []byte(absent)) {
			t.Errorf("legacy save added optional field %s:\n%s", absent, written)
		}
	}
	for _, extra := range []string{`"futureSession"`, `"futureTerminal"`, `"futureCommand"`} {
		if !bytes.Contains(written, []byte(extra)) {
			t.Errorf("legacy save dropped unknown field %s:\n%s", extra, written)
		}
	}

	roundTrip, err := domain.DecodeSession(written)
	if err != nil {
		t.Fatalf("saved legacy document does not reopen: %v", err)
	}
	if roundTrip.Version != 1 {
		t.Fatalf("saved legacy version = %d, want 1", roundTrip.Version)
	}
	if got := roundTrip.Terminals[0].Root.Children[0]; !reflect.DeepEqual(got, ordinary) {
		t.Fatalf("ordinary content changed during service round trip\ngot:  %#v\nwant: %#v", got, ordinary)
	}
}

func TestAssociatePlayerConfigPersistsRelativeReferenceAndKeepsActiveSession(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	sessionPath := "/Campaigns/Chapter One/session.json"
	data, err := domain.EncodeSession(validSession("chapter one"))
	if err != nil {
		t.Fatal(err)
	}
	fs.SeedFile(sessionPath, data)
	service := NewService(NewStorage(fs), &testutil.FakeDialog{OpenResult: sessionPath}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	if opened := service.Open(context.Background()); !opened.OK {
		t.Fatalf("Open() = %#v", opened)
	}

	configPath := "/Campaigns/Players/shared.json"
	result := service.AssociatePlayerConfig(context.Background(), configPath)
	if !result.OK || result.Session == nil {
		t.Fatalf("AssociatePlayerConfig() = %#v", result)
	}
	want := filepath.Join("..", "Players", "shared.json")
	if result.Session.PlayerConfig != want {
		t.Fatalf("playerConfig = %q, want %q", result.Session.PlayerConfig, want)
	}

	written, err := domain.DecodeSession(fileSystemFileData(t, fs, sessionPath))
	if err != nil {
		t.Fatal(err)
	}
	if written.PlayerConfig != want || service.Snapshot().Session.PlayerConfig != want {
		t.Fatalf("association was not durable and active: written=%q active=%q", written.PlayerConfig, service.Snapshot().Session.PlayerConfig)
	}
}

func TestSaveWithoutActivePathFailsWithoutWriting(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	result := service.Save(context.Background(), validSession("orphan"), 1)
	if result.OK || result.Error == "" || result.RequestedRevision != 1 || result.SavedRevision != 0 {
		t.Fatalf("Save() without active path = %#v", result)
	}
	if len(fileSystem.WriteCalls()) != 0 || len(fileSystem.RenameCalls()) != 0 {
		t.Fatalf("Save() without active path mutated filesystem")
	}
	assertInactive(t, service.Snapshot())
}

func TestCopyDemoRequiresExplicitDestinationAndActivatesWritableCopy(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	demo := validSession("demo")
	demo.PlayerConfig = "demo-players.json"
	demoData, err := domain.EncodeSession(demo)
	if err != nil {
		t.Fatal(err)
	}
	playerConfigData, err := domain.EncodePlayerConfig(domain.PlayerConfig{
		Version: 1,
		Name:    "demo-players",
		Roster: []domain.CharacterRosterEntry{
			{ID: "scout", Name: "Разведчик"},
			{ID: "medic", Name: "Медик"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fileSystem.SeedFile(testLocations.BundledDemo, demoData)
	bundledPlayerConfig := filepath.Join(filepath.Dir(testLocations.BundledDemo), demo.PlayerConfig)
	fileSystem.SeedFile(bundledPlayerConfig, playerConfigData)
	destination := "/Volumes/Campaigns/demo-copy.json"
	dialog := &testutil.FakeDialog{SaveResult: destination}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })

	result := service.CopyDemo(context.Background())
	if !result.OK || result.Canceled || result.Error != "" || result.FilePath != destination || result.Session == nil {
		t.Fatalf("CopyDemo() = %#v", result)
	}
	if got := fileSystemFileData(t, fileSystem, testLocations.BundledDemo); !bytes.Equal(got, demoData) {
		t.Fatalf("bundled demo changed:\n%s", got)
	}
	if got := fileSystemFileData(t, fileSystem, bundledPlayerConfig); !bytes.Equal(got, playerConfigData) {
		t.Fatalf("bundled demo player config changed:\n%s", got)
	}
	if got := fileSystemFileData(t, fileSystem, destination); !bytes.Equal(got, demoData) {
		t.Fatalf("writable copy = %s, want %s", got, demoData)
	}
	destinationPlayerConfig := filepath.Join(filepath.Dir(destination), demo.PlayerConfig)
	if got := fileSystemFileData(t, fileSystem, destinationPlayerConfig); !bytes.Equal(got, playerConfigData) {
		t.Fatalf("writable player config copy = %s, want %s", got, playerConfigData)
	}
	if snapshot := service.Snapshot(); snapshot.Path != destination || snapshot.Session == nil {
		t.Fatalf("active demo copy = %#v", snapshot)
	}
}

func TestTwentyQueuedRevisionsFinishAtNewestAcceptedSession(t *testing.T) {
	store := newBlockingStore()
	target := "/Volumes/Campaigns/ordered.json"
	store.seed(target, mustEncodeSession(t, validSession("initial")))
	dialog := &testutil.FakeDialog{OpenResult: target}
	service := NewService(store, dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	if result := service.Open(context.Background()); !result.OK {
		t.Fatalf("Open() = %#v", result)
	}

	type completion struct {
		revision uint64
		result   SaveResult
	}
	completions := make(chan completion, 20)
	startSave := func(revision uint64) {
		session := validSession(fmt.Sprintf("revision-%02d", revision))
		go func() {
			completions <- completion{revision: revision, result: service.Save(context.Background(), session, revision)}
		}()
	}

	startSave(1)
	select {
	case <-store.firstWriteStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first revision did not begin writing")
	}
	t.Cleanup(store.release)
	for revision := uint64(2); revision <= 20; revision++ {
		startSave(revision)
		waitForRequestedRevision(t, service, revision)
	}
	store.release()

	results := make(map[uint64]SaveResult, 20)
	for range 20 {
		select {
		case completion := <-completions:
			results[completion.revision] = completion.result
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d of 20 saves completed", len(results))
		}
	}
	for revision := uint64(1); revision <= 20; revision++ {
		result := results[revision]
		if !result.OK || result.Error != "" || result.RequestedRevision != revision {
			t.Errorf("revision %d result = %#v", revision, result)
		}
		if result.SavedRevision < revision || result.SavedRevision > 20 {
			t.Errorf("revision %d durable result = %d, want [%d,20]", revision, result.SavedRevision, revision)
		}
	}
	if results[20].SavedRevision != 20 {
		t.Fatalf("newest result = %#v, want durable revision 20", results[20])
	}
	final, err := domain.DecodeSession(store.file(target))
	if err != nil {
		t.Fatalf("final saved file is invalid: %v", err)
	}
	if final.Name != "revision-20" {
		t.Fatalf("final session name = %q, want revision-20", final.Name)
	}
	snapshot := service.Snapshot()
	if snapshot.Path != target || snapshot.RequestedRevision != 20 || snapshot.SavedRevision != 20 || snapshot.SaveState != SaveStateSaved {
		t.Fatalf("final active state = %#v", snapshot)
	}
}

func TestCommandStateMutationsAllocateMonotonicDocumentRevisions(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/state-mutations.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("chapter")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	if opened := service.Open(context.Background()); !opened.OK {
		t.Fatalf("Open() = %#v", opened)
	}

	doors := service.ExecuteCommandState(context.Background(), "t1", "doors")
	if !doors.OK || !doors.Changed || doors.Error != "" || doors.Revision != 1 || doors.Session == nil {
		t.Fatalf("ExecuteCommandState(doors) = %#v", doors)
	}
	if got := doors.Session.Terminals[0].CommandStates["doors"]; got.CompletedName != "Двери открыты" || got.ResultText != "Доступ в сектор разрешён." {
		t.Fatalf("durable doors snapshot = %#v", got)
	}

	alarm := service.ExecuteCommandState(context.Background(), "t1", "alarm")
	if !alarm.OK || !alarm.Changed || alarm.Revision != 2 {
		t.Fatalf("ExecuteCommandState(alarm) = %#v", alarm)
	}
	one := service.ResetCommandState(context.Background(), "t1", "doors")
	if !one.OK || !one.Changed || one.Revision != 3 || one.Session == nil {
		t.Fatalf("ResetCommandState() = %#v", one)
	}
	if _, exists := one.Session.Terminals[0].CommandStates["doors"]; exists {
		t.Fatalf("reset-one retained doors snapshot: %#v", one.Session.Terminals[0].CommandStates)
	}
	if _, exists := one.Session.Terminals[0].CommandStates["alarm"]; !exists {
		t.Fatalf("reset-one removed sibling snapshot: %#v", one.Session.Terminals[0].CommandStates)
	}

	all := service.ResetTerminalCommandStates(context.Background(), "t1")
	if !all.OK || !all.Changed || all.Revision != 4 || all.Session == nil || len(all.Session.Terminals[0].CommandStates) != 0 {
		t.Fatalf("ResetTerminalCommandStates() = %#v", all)
	}
	writesBeforeNoOp := len(fileSystem.WriteCalls())
	noOp := service.ResetTerminalCommandStates(context.Background(), "t1")
	if !noOp.OK || noOp.Changed || noOp.Revision != 4 || noOp.Session == nil {
		t.Fatalf("idempotent ResetTerminalCommandStates() = %#v", noOp)
	}
	if writes := len(fileSystem.WriteCalls()); writes != writesBeforeNoOp {
		t.Fatalf("idempotent reset wrote file: writes %d -> %d", writesBeforeNoOp, writes)
	}
	if snapshot := service.Snapshot(); snapshot.RequestedRevision != 4 || snapshot.SavedRevision != 4 || snapshot.SaveState != SaveStateSaved {
		t.Fatalf("active document revisions = %#v", snapshot)
	}
}

func TestStaleFullSavePreservesCanonicalFrozenStateAndAppliesAuthoredEdits(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/stale-save.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("before")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil {
		t.Fatalf("Open() = %#v", opened)
	}
	stale := *opened.Session

	executed := service.ExecuteCommandState(context.Background(), "t1", "doors")
	if !executed.OK || !executed.Changed || executed.Revision != 1 {
		t.Fatalf("ExecuteCommandState() = %#v", executed)
	}
	stale.Name = "after"
	stale.Terminals[0].Root.Children[0].Name = "Открыть гермодвери"
	stale.Terminals[0].Root.Children[0].Text = "Новый результат для следующего выполнения."
	stale.Terminals[0].Root.Children[0].StateChange.CompletedName = "Гермодвери открыты"
	stale.Terminals[0].CommandStates = map[string]domain.CommandExecutionState{
		"doors": {CompletedName: "ПОДДЕЛКА", ResultText: "ПОДДЕЛКА"},
	}

	saved := service.Save(context.Background(), stale, 2)
	if !saved.OK || saved.RequestedRevision != 2 || saved.SavedRevision != 2 {
		t.Fatalf("Save(stale) = %#v", saved)
	}
	active := service.Snapshot()
	if active.Session == nil || active.Session.Name != "after" {
		t.Fatalf("active authored document = %#v", active)
	}
	command := active.Session.Terminals[0].Root.Children[0]
	if command.Name != "Открыть гермодвери" || command.StateChange.CompletedName != "Гермодвери открыты" {
		t.Fatalf("authored edits were not applied: %#v", command)
	}
	wantFrozen := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
	if got := active.Session.Terminals[0].CommandStates["doors"]; !reflect.DeepEqual(got, wantFrozen) {
		t.Fatalf("stale save replaced frozen state: got %#v want %#v", got, wantFrozen)
	}
	reopened, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Terminals[0].CommandStates["doors"]; !reflect.DeepEqual(got, wantFrozen) {
		t.Fatalf("durable stale-save merge = %#v, want %#v", got, wantFrozen)
	}
}

func TestFullSavePrunesFrozenStateWhenCommandIsDeleted(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/delete-command.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("before")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	if opened := service.Open(context.Background()); !opened.OK {
		t.Fatalf("Open() = %#v", opened)
	}
	if result := service.ExecuteCommandState(context.Background(), "t1", "doors"); !result.OK || result.Revision != 1 {
		t.Fatalf("ExecuteCommandState() = %#v", result)
	}

	candidate := *service.Snapshot().Session
	candidate.Terminals[0].Root.Children = append([]domain.ContentNode(nil), candidate.Terminals[0].Root.Children[1:]...)
	if result := service.Save(context.Background(), candidate, 2); !result.OK || result.SavedRevision != 2 {
		t.Fatalf("Save(with deletion) = %#v", result)
	}
	active := service.Snapshot()
	if active.Session == nil || len(active.Session.Terminals[0].CommandStates) != 0 {
		t.Fatalf("deleted command retained canonical state: %#v", active.Session)
	}
	written, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Terminals[0].CommandStates) != 0 {
		t.Fatalf("deleted command retained durable state: %#v", written.Terminals[0].CommandStates)
	}
}

func TestStableIDAndFrozenStateRulesAcross100CompletedCommands(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/one-hundred-completed-commands.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSessionWith100CompletedCommands()))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.Background()) })
	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil {
		t.Fatalf("Open() = %#v", opened)
	}

	candidate := *opened.Session
	moved := domain.ContentNode{ID: "moved-folder", Type: domain.NodeFolder, Name: "MOVED COMMANDS"}
	for index := 99; index >= 0; index-- {
		command := candidate.Terminals[0].Root.Children[index]
		command.Name = fmt.Sprintf("Renamed command %03d", index)
		command.Text = fmt.Sprintf("Next result %03d", index)
		command.StateChange.CompletedName = fmt.Sprintf("Next completed %03d", index)
		command.StateChange.ConfirmationText = fmt.Sprintf("Run renamed command %03d?", index)
		moved.Children = append(moved.Children, command)
	}
	candidate.Terminals[0].Root.Children = []domain.ContentNode{moved}

	saved := service.Save(context.Background(), candidate, 1)
	if !saved.OK || saved.SavedRevision != 1 {
		t.Fatalf("Save(rename/move/delete 100 commands) = %#v", saved)
	}
	active := service.Snapshot()
	if active.Session == nil || len(active.Session.Terminals[0].CommandStates) != 100 {
		t.Fatalf("stable-ID merge retained %d states, want 100", len(active.Session.Terminals[0].CommandStates))
	}
	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		wantFrozen := domain.CommandExecutionState{
			CompletedName: fmt.Sprintf("Frozen completed %03d", index),
			ResultText:    fmt.Sprintf("Frozen result %03d", index),
		}
		if got := active.Session.Terminals[0].CommandStates[commandID]; !reflect.DeepEqual(got, wantFrozen) {
			t.Fatalf("renamed/moved command %q snapshot = %#v, want %#v", commandID, got, wantFrozen)
		}
	}

	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		reset := service.ResetCommandState(context.Background(), "t100", commandID)
		if !reset.OK || !reset.Changed {
			t.Fatalf("ResetCommandState(%q) = %#v", commandID, reset)
		}
		executed := service.ExecuteCommandState(context.Background(), "t100", commandID)
		if !executed.OK || !executed.Changed || executed.Session == nil {
			t.Fatalf("ExecuteCommandState(%q) after reset = %#v", commandID, executed)
		}
		wantNext := domain.CommandExecutionState{
			CompletedName: fmt.Sprintf("Next completed %03d", index),
			ResultText:    fmt.Sprintf("Next result %03d", index),
		}
		if got := executed.Session.Terminals[0].CommandStates[commandID]; !reflect.DeepEqual(got, wantNext) {
			t.Fatalf("re-executed command %q snapshot = %#v, want %#v", commandID, got, wantNext)
		}
	}
	active = service.Snapshot()
	if active.Session == nil || len(active.Session.Terminals[0].CommandStates) != 100 {
		t.Fatalf("reset/re-execute retained %d states, want 100", len(active.Session.Terminals[0].CommandStates))
	}

	replacements := domain.ContentNode{ID: "replacement-folder", Type: domain.NodeFolder, Name: "REPLACEMENTS"}
	for index := range 100 {
		replacements.Children = append(replacements.Children, domain.ContentNode{
			ID:   fmt.Sprintf("replacement-%03d", index),
			Type: domain.NodeCommand,
			Name: fmt.Sprintf("Replacement %03d", index),
			Text: fmt.Sprintf("Replacement result %03d", index),
			StateChange: &domain.StateChangeConfig{
				CompletedName:    fmt.Sprintf("Replacement completed %03d", index),
				ConfirmationText: fmt.Sprintf("Replace command %03d?", index),
			},
		})
	}
	deletedCandidate := *active.Session
	deletedCandidate.Terminals[0].Root.Children = []domain.ContentNode{replacements}
	deleteRevision := active.RequestedRevision + 1
	deleted := service.Save(context.Background(), deletedCandidate, deleteRevision)
	if !deleted.OK || deleted.SavedRevision != deleteRevision {
		t.Fatalf("Save(delete all 100 commands and add replacements) = %#v", deleted)
	}
	active = service.Snapshot()
	if active.Session == nil || len(active.Session.Terminals[0].CommandStates) != 0 {
		t.Fatalf("deleting all 100 commands retained snapshots: %#v", active.Session)
	}
	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		replacementID := fmt.Sprintf("replacement-%03d", index)
		if _, exists := active.Session.Terminals[0].CommandStates[commandID]; exists {
			t.Fatalf("deleted command %q retained a snapshot", commandID)
		}
		if _, exists := active.Session.Terminals[0].CommandStates[replacementID]; exists {
			t.Fatalf("replacement command %q inherited a deleted snapshot", replacementID)
		}
	}

	reopened, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	if err != nil {
		t.Fatalf("DecodeSession(after 100-command lifecycle) error = %v", err)
	}
	if len(reopened.Terminals[0].CommandStates) != 0 {
		t.Fatalf("durable deletion retained %d command states, want 0", len(reopened.Terminals[0].CommandStates))
	}
}

func TestFailedCommandStateMutationKeepsPriorDocumentAndRevision(t *testing.T) {
	t.Parallel()

	target := "/Volumes/Campaigns/failed-command-state.json"
	initial := mustEncodeSession(t, stateChangingSession("safe"))
	store := &failingMutationStore{path: target, data: initial, err: fmt.Errorf("injected atomic replacement failure")}
	service := NewService(store, &testutil.FakeDialog{OpenResult: target}, testLocations)
	if opened := service.Open(context.Background()); !opened.OK {
		t.Fatalf("Open() = %#v", opened)
	}

	for attempt := range 100 {
		result := service.ExecuteCommandState(context.Background(), "t1", "doors")
		if result.OK || result.Changed || result.Error == "" || result.Revision != 0 || result.Session != nil {
			t.Fatalf("failed ExecuteCommandState() attempt %d = %#v", attempt, result)
		}
		if store.writes != attempt+1 || !bytes.Equal(store.data, initial) {
			t.Fatalf("failed mutation attempt %d changed durable document: writes=%d data=%s", attempt, store.writes, store.data)
		}
		active := service.Snapshot()
		if active.RequestedRevision != 0 || active.SavedRevision != 0 || active.Session == nil || len(active.Session.Terminals[0].CommandStates) != 0 {
			t.Fatalf("failed mutation attempt %d changed active state: %#v", attempt, active)
		}
	}
	if err := service.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	restarted := NewService(store, &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = restarted.Shutdown(context.Background()) })
	reopened := restarted.Open(context.Background())
	if !reopened.OK || reopened.Session == nil || len(reopened.Session.Terminals[0].CommandStates) != 0 {
		t.Fatalf("reopen after 100 failed mutations = %#v", reopened)
	}
	if !bytes.Equal(store.data, initial) {
		t.Fatalf("reopen after failures observed a changed document: %s", store.data)
	}
}

func TestCompletedCommandStateSurvivesFreshProcessReopen(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "process-restart-session.json")
	initial := mustEncodeSession(t, stateChangingSession("process restart"))
	if err := os.WriteFile(target, initial, 0o600); err != nil {
		t.Fatalf("seed process-restart session: %v", err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	for _, mode := range []string{"execute", "reopen"} {
		command := exec.Command(executable, "-test.run=^TestCommandStateFreshProcessHelper$", "-test.v")
		command.Env = append(os.Environ(),
			"FALLOUT_COMMAND_STATE_PROCESS_MODE="+mode,
			"FALLOUT_COMMAND_STATE_PROCESS_PATH="+target,
		)
		output, runErr := command.CombinedOutput()
		if runErr != nil {
			t.Fatalf("fresh process %q failed: %v\n%s", mode, runErr, output)
		}
	}

	durable, err := domain.DecodeSession(mustReadFile(t, target))
	if err != nil {
		t.Fatalf("DecodeSession(after process restart) error = %v", err)
	}
	want := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
	if got := durable.Terminals[0].CommandStates["doors"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("durable process-restart snapshot = %#v, want %#v", got, want)
	}
}

func TestCommandStateFreshProcessHelper(t *testing.T) {
	mode := os.Getenv("FALLOUT_COMMAND_STATE_PROCESS_MODE")
	if mode == "" {
		t.Skip("helper runs only in a fresh subprocess")
	}
	target := os.Getenv("FALLOUT_COMMAND_STATE_PROCESS_PATH")
	if target == "" {
		t.Fatal("FALLOUT_COMMAND_STATE_PROCESS_PATH is empty")
	}

	service := NewService(NewStorage(nil), &testutil.FakeDialog{OpenResult: target}, testLocations)
	opened := service.Open(context.Background())
	if !opened.OK || opened.Session == nil {
		t.Fatalf("Open() in %s process = %#v", mode, opened)
	}
	switch mode {
	case "execute":
		result := service.ExecuteCommandState(context.Background(), "t1", "doors")
		if !result.OK || !result.Changed || result.Session == nil {
			t.Fatalf("ExecuteCommandState() in fresh process = %#v", result)
		}
	case "reopen":
		want := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
		if got := opened.Session.Terminals[0].CommandStates["doors"]; !reflect.DeepEqual(got, want) {
			t.Fatalf("fresh process reopened snapshot = %#v, want %#v", got, want)
		}
	default:
		t.Fatalf("unknown helper mode %q", mode)
	}
	if err := service.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() in %s process = %v", mode, err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return data
}

func stateChangingSession(name string) domain.Session {
	return domain.Session{
		Version: 1,
		Name:    name,
		Terminals: []domain.Terminal{
			{
				ID: "t1", Name: "Terminal 1", Root: domain.ContentNode{
					ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{
						{
							ID: "doors", Type: domain.NodeCommand, Name: "Открыть двери", Text: "Доступ в сектор разрешён.",
							StateChange: &domain.StateChangeConfig{CompletedName: "Двери открыты", ConfirmationText: "Открыть двери?"},
						},
						{
							ID: "alarm", Type: domain.NodeCommand, Name: "Отключить тревогу", Text: "Тревога отключена.",
							StateChange: &domain.StateChangeConfig{CompletedName: "Тревога отключена", ConfirmationText: "Отключить тревогу?"},
						},
					},
				},
			},
			{
				ID: "t2", Name: "Terminal 2", Root: domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}},
			},
		},
	}
}

func stateChangingSessionWith100CompletedCommands() domain.Session {
	terminal := domain.Terminal{
		ID: "t100", Name: "One hundred commands", HackLevel: 1,
		Root:          domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT"},
		CommandStates: make(map[string]domain.CommandExecutionState, 100),
	}
	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		terminal.Root.Children = append(terminal.Root.Children, domain.ContentNode{
			ID:   commandID,
			Type: domain.NodeCommand,
			Name: fmt.Sprintf("Initial command %03d", index),
			Text: fmt.Sprintf("Authored result %03d", index),
			StateChange: &domain.StateChangeConfig{
				CompletedName:    fmt.Sprintf("Authored completed %03d", index),
				ConfirmationText: fmt.Sprintf("Run command %03d?", index),
			},
		})
		terminal.CommandStates[commandID] = domain.CommandExecutionState{
			CompletedName: fmt.Sprintf("Frozen completed %03d", index),
			ResultText:    fmt.Sprintf("Frozen result %03d", index),
		}
	}
	return domain.Session{
		Version:   1,
		Name:      "Stable ID sample of 100 completed commands",
		Terminals: []domain.Terminal{terminal},
	}
}

func validSession(name string) domain.Session {
	return domain.Session{
		Version: 1,
		Name:    name,
		Terminals: []domain.Terminal{{
			ID:        "t1",
			Name:      "Terminal 1",
			HackLevel: 0,
			IntroText: "",
			Root: domain.ContentNode{
				ID:       "root",
				Type:     domain.NodeFolder,
				Name:     "ROOT",
				Children: []domain.ContentNode{},
			},
		}},
	}
}

func assertInactive(t *testing.T, snapshot ActiveSession) {
	t.Helper()
	if snapshot.Path != "" || snapshot.Session != nil || snapshot.RequestedRevision != 0 || snapshot.SavedRevision != 0 || snapshot.SaveState != SaveStateIdle {
		t.Fatalf("active state = %#v, want idle and empty", snapshot)
	}
}

func assertNoApplicationSupportWrites(t *testing.T, fileSystem *testutil.FakeFileSystem) {
	t.Helper()
	for _, write := range fileSystem.WriteCalls() {
		if strings.HasPrefix(write.Path, testLocations.ApplicationSupport+string(filepath.Separator)) {
			t.Fatalf("session content written to Application Support: %q", write.Path)
		}
	}
}

func fileSystemFileData(t *testing.T, fileSystem *testutil.FakeFileSystem, path string) []byte {
	t.Helper()
	data, ok := fileSystem.File(path)
	if !ok {
		t.Fatalf("file %q does not exist", path)
	}
	return data
}

func mustEncodeSession(t *testing.T, session domain.Session) []byte {
	t.Helper()
	data, err := domain.EncodeSession(session)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func waitForRequestedRevision(t *testing.T, service *Service, want uint64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if service.Snapshot().RequestedRevision >= want {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("requested revision never reached %d; state = %#v", want, service.Snapshot())
}

type blockingStore struct {
	mu sync.Mutex

	files map[string][]byte

	firstWriteStarted chan struct{}
	releaseFirstWrite chan struct{}
	firstOnce         sync.Once
	releaseOnce       sync.Once
}

func newBlockingStore() *blockingStore {
	return &blockingStore{
		files:             make(map[string][]byte),
		firstWriteStarted: make(chan struct{}),
		releaseFirstWrite: make(chan struct{}),
	}
}

func (store *blockingStore) seed(path string, data []byte) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.files[path] = append([]byte(nil), data...)
}

func (store *blockingStore) Read(path string) ([]byte, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	data, ok := store.files[path]
	if !ok {
		return nil, fmt.Errorf("read %q: not found", path)
	}
	return append([]byte(nil), data...), nil
}

func (store *blockingStore) WriteAtomic(path string, data []byte) error {
	blocked := false
	store.firstOnce.Do(func() {
		blocked = true
		close(store.firstWriteStarted)
	})
	if blocked {
		<-store.releaseFirstWrite
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.files[path] = append([]byte(nil), data...)
	return nil
}

func (store *blockingStore) CopyAtomic(source, destination string) error {
	data, err := store.Read(source)
	if err != nil {
		return err
	}
	return store.WriteAtomic(destination, data)
}

func (store *blockingStore) file(path string) []byte {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]byte(nil), store.files[path]...)
}

func (store *blockingStore) release() {
	store.releaseOnce.Do(func() { close(store.releaseFirstWrite) })
}

type failingMutationStore struct {
	path   string
	data   []byte
	err    error
	writes int
}

func (store *failingMutationStore) Read(path string) ([]byte, error) {
	if path != store.path {
		return nil, fmt.Errorf("read unexpected path %q", path)
	}
	return append([]byte(nil), store.data...), nil
}

func (store *failingMutationStore) WriteAtomic(path string, data []byte) error {
	store.writes++
	if path != store.path {
		return fmt.Errorf("write unexpected path %q", path)
	}
	if store.err != nil {
		return store.err
	}
	store.data = append([]byte(nil), data...)
	return nil
}

func (store *failingMutationStore) CopyAtomic(source, destination string) error {
	return fmt.Errorf("unexpected copy from %q to %q", source, destination)
}

package session

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

			result := test.invoke(t.Context(), service)
			assert.False(t, result.OK)
			assert.True(t, result.Canceled)
			assert.Empty(t, result.Error)
			assert.Empty(t, result.FilePath)
			assert.Nil(t, result.Session)
			assert.Equal(t, []testutil.DialogCall{{Kind: test.wantKind, DefaultPath: test.wantDefault}}, dialog.Calls())
			assertInactive(t, service.Snapshot())
			assert.Empty(t, fileSystem.MkdirCalls(), "cancellation created directories")
			assert.Empty(t, fileSystem.WriteCalls(), "cancellation wrote files")
		})
	}
}

func TestCreateUsesDocumentsSuggestionAndActivatesChosenPathAfterWrite(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/My Wasteland.json"
	dialog := &testutil.FakeDialog{SaveResult: target}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	result := service.Create(t.Context())
	require.True(t, result.OK, "Create() = %#v", result)
	assert.False(t, result.Canceled)
	assert.Empty(t, result.Error)
	assert.Equal(t, target, result.FilePath)
	require.NotNil(t, result.Session)
	assert.Equal(t, "My Wasteland", result.Session.Name)
	assert.Equal(t, 1, result.Session.Version)
	assert.Len(t, result.Session.Terminals, 1)
	require.NoError(t, domain.ValidateSession(*result.Session))
	written, ok := fileSystem.File(target)
	require.True(t, ok, "chosen target %q was not written", target)
	assert.True(t, bytes.HasSuffix(written, []byte("\n")), "created JSON must have a final newline")
	assert.NotContains(t, string(written), "\t", "created JSON must not contain tabs")
	snapshot := service.Snapshot()
	assert.Equal(t, target, snapshot.Path)
	require.NotNil(t, snapshot.Session)
	assert.Equal(t, "My Wasteland", snapshot.Session.Name)
	assertNoApplicationSupportWrites(t, fileSystem)
}

func TestInvalidOpenRetainsPreviousActiveSessionAndPath(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	validPath := "/Volumes/Campaigns/valid.json"
	invalidPath := "/Volumes/Campaigns/invalid.json"
	validData, err := domain.EncodeSession(validSession("safe"))
	require.NoError(t, err)
	fileSystem.SeedFile(validPath, validData)
	fileSystem.SeedFile(invalidPath, []byte(`{"version":2,"name":"bad","terminals":[]}`))
	dialog := &testutil.FakeDialog{OpenResult: validPath}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	opened := service.Open(t.Context())
	require.True(t, opened.OK, "first Open() = %#v", opened)
	require.NotNil(t, opened.Session)
	assert.Equal(t, validPath, opened.FilePath)
	dialog.OpenResult = invalidPath
	failed := service.Open(t.Context())
	assert.False(t, failed.OK)
	assert.False(t, failed.Canceled)
	assert.NotEmpty(t, failed.Error)
	assert.Nil(t, failed.Session)
	assert.NotContains(t, failed.Error, string(fileSystemFileData(t, fileSystem, invalidPath)))
	snapshot := service.Snapshot()
	assert.Equal(t, validPath, snapshot.Path)
	require.NotNil(t, snapshot.Session)
	assert.Equal(t, "safe", snapshot.Session.Name)
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
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)
	require.NotNil(t, opened.Session)
	edited := *opened.Session
	edited.Name = "after"
	saved := service.Save(t.Context(), edited, 1)
	require.True(t, saved.OK, "Save() = %#v", saved)
	assert.Empty(t, saved.Error)
	assert.Equal(t, uint64(1), saved.RequestedRevision)
	assert.Equal(t, uint64(1), saved.SavedRevision)
	written := fileSystemFileData(t, fileSystem, target)
	for _, field := range []string{"campaignNote", "terminalNote", "nodeNote"} {
		assert.Contains(t, string(written), `"`+field+`"`)
	}
	decoded, err := domain.DecodeSession(written)
	require.NoError(t, err)
	assert.Equal(t, "after", decoded.Name)
	for _, rename := range fileSystem.RenameCalls() {
		assert.Equal(t, target, rename.NewPath)
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
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)
	require.NotNil(t, opened.Session)
	ordinary := opened.Session.Terminals[0].Root.Children[0]
	assert.Nil(t, ordinary.StateChange)
	assert.Empty(t, opened.Session.Terminals[0].CommandStates)

	saved := service.Save(t.Context(), *opened.Session, 1)
	require.True(t, saved.OK, "Save() = %#v", saved)
	assert.Equal(t, uint64(1), saved.RequestedRevision)
	assert.Equal(t, uint64(1), saved.SavedRevision)
	written := fileSystemFileData(t, fileSystem, target)
	for _, absent := range []string{`"stateChange"`, `"commandStates"`} {
		assert.NotContains(t, string(written), absent)
	}
	for _, extra := range []string{`"futureSession"`, `"futureTerminal"`, `"futureCommand"`} {
		assert.Contains(t, string(written), extra)
	}

	roundTrip, err := domain.DecodeSession(written)
	require.NoError(t, err)
	assert.Equal(t, 1, roundTrip.Version)
	assert.Equal(t, ordinary, roundTrip.Terminals[0].Root.Children[0])
}

func TestAssociatePlayerConfigPersistsRelativeReferenceAndKeepsActiveSession(t *testing.T) {
	t.Parallel()

	fs := testutil.NewFakeFileSystem()
	sessionPath := "/Campaigns/Chapter One/session.json"
	data, err := domain.EncodeSession(validSession("chapter one"))
	require.NoError(t, err)
	fs.SeedFile(sessionPath, data)
	service := NewService(NewStorage(fs), &testutil.FakeDialog{OpenResult: sessionPath}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)

	configPath := "/Campaigns/Players/shared.json"
	result := service.AssociatePlayerConfig(t.Context(), configPath)
	require.True(t, result.OK, "AssociatePlayerConfig() = %#v", result)
	require.NotNil(t, result.Session)
	want := filepath.Join("..", "Players", "shared.json")
	assert.Equal(t, want, result.Session.PlayerConfig)

	written, err := domain.DecodeSession(fileSystemFileData(t, fs, sessionPath))
	require.NoError(t, err)
	assert.Equal(t, want, written.PlayerConfig)
	require.NotNil(t, service.Snapshot().Session)
	assert.Equal(t, want, service.Snapshot().Session.PlayerConfig)
}

func TestSaveWithoutActivePathFailsWithoutWriting(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	result := service.Save(t.Context(), validSession("orphan"), 1)
	assert.False(t, result.OK)
	assert.NotEmpty(t, result.Error)
	assert.Equal(t, uint64(1), result.RequestedRevision)
	assert.Zero(t, result.SavedRevision)
	assert.Empty(t, fileSystem.WriteCalls())
	assert.Empty(t, fileSystem.RenameCalls())
	assertInactive(t, service.Snapshot())
}

func TestCopyDemoRequiresExplicitDestinationAndActivatesWritableCopy(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	demo := validSession("demo")
	demo.PlayerConfig = "demo-players.json"
	demoData, err := domain.EncodeSession(demo)
	require.NoError(t, err)
	playerConfigData, err := domain.EncodePlayerConfig(domain.PlayerConfig{
		Version: 1,
		Name:    "demo-players",
		Roster: []domain.CharacterRosterEntry{
			{ID: "scout", Name: "Разведчик"},
			{ID: "medic", Name: "Медик"},
		},
	})
	require.NoError(t, err)
	fileSystem.SeedFile(testLocations.BundledDemo, demoData)
	bundledPlayerConfig := filepath.Join(filepath.Dir(testLocations.BundledDemo), demo.PlayerConfig)
	fileSystem.SeedFile(bundledPlayerConfig, playerConfigData)
	destination := "/Volumes/Campaigns/demo-copy.json"
	dialog := &testutil.FakeDialog{SaveResult: destination}
	service := NewService(NewStorage(fileSystem), dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })

	result := service.CopyDemo(t.Context())
	require.True(t, result.OK, "CopyDemo() = %#v", result)
	assert.False(t, result.Canceled)
	assert.Empty(t, result.Error)
	assert.Equal(t, destination, result.FilePath)
	require.NotNil(t, result.Session)
	assert.Equal(t, demoData, fileSystemFileData(t, fileSystem, testLocations.BundledDemo))
	assert.Equal(t, playerConfigData, fileSystemFileData(t, fileSystem, bundledPlayerConfig))
	assert.Equal(t, demoData, fileSystemFileData(t, fileSystem, destination))
	destinationPlayerConfig := filepath.Join(filepath.Dir(destination), demo.PlayerConfig)
	assert.Equal(t, playerConfigData, fileSystemFileData(t, fileSystem, destinationPlayerConfig))
	snapshot := service.Snapshot()
	assert.Equal(t, destination, snapshot.Path)
	assert.NotNil(t, snapshot.Session)
}

func TestTwentyQueuedRevisionsFinishAtNewestAcceptedSession(t *testing.T) {
	store := newBlockingStore()
	target := "/Volumes/Campaigns/ordered.json"
	store.seed(target, mustEncodeSession(t, validSession("initial")))
	dialog := &testutil.FakeDialog{OpenResult: target}
	service := NewService(store, dialog, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	result := service.Open(t.Context())
	require.True(t, result.OK, "Open() = %#v", result)

	type completion struct {
		revision uint64
		result   SaveResult
	}
	completions := make(chan completion, 20)
	startSave := func(revision uint64) {
		session := validSession(fmt.Sprintf("revision-%02d", revision))
		go func() {
			completions <- completion{revision: revision, result: service.Save(t.Context(), session, revision)}
		}()
	}

	startSave(1)
	select {
	case <-store.firstWriteStarted:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "first revision did not begin writing")
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
			require.FailNowf(t, "saves did not complete", "only %d of 20 saves completed", len(results))
		}
	}
	for revision := uint64(1); revision <= 20; revision++ {
		result := results[revision]
		assert.True(t, result.OK, "revision %d result = %#v", revision, result)
		assert.Empty(t, result.Error, "revision %d", revision)
		assert.Equal(t, revision, result.RequestedRevision)
		assert.GreaterOrEqual(t, result.SavedRevision, revision)
		assert.LessOrEqual(t, result.SavedRevision, uint64(20))
	}
	assert.Equal(t, uint64(20), results[20].SavedRevision)
	final, err := domain.DecodeSession(store.file(target))
	require.NoError(t, err)
	assert.Equal(t, "revision-20", final.Name)
	snapshot := service.Snapshot()
	assert.Equal(t, target, snapshot.Path)
	assert.Equal(t, uint64(20), snapshot.RequestedRevision)
	assert.Equal(t, uint64(20), snapshot.SavedRevision)
	assert.Equal(t, SaveStateSaved, snapshot.SaveState)
}

func TestCommandStateMutationsAllocateMonotonicDocumentRevisions(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/state-mutations.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("chapter")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)

	doors := service.ExecuteCommandState(t.Context(), "t1", "doors")
	require.True(t, doors.OK, "ExecuteCommandState(doors) = %#v", doors)
	assert.True(t, doors.Changed)
	assert.Empty(t, doors.Error)
	assert.Equal(t, uint64(1), doors.Revision)
	require.NotNil(t, doors.Session)
	assert.Equal(t, domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}, doors.Session.Terminals[0].CommandStates["doors"])

	alarm := service.ExecuteCommandState(t.Context(), "t1", "alarm")
	require.True(t, alarm.OK, "ExecuteCommandState(alarm) = %#v", alarm)
	assert.True(t, alarm.Changed)
	assert.Equal(t, uint64(2), alarm.Revision)
	one := service.ResetCommandState(t.Context(), "t1", "doors")
	require.True(t, one.OK, "ResetCommandState() = %#v", one)
	assert.True(t, one.Changed)
	assert.Equal(t, uint64(3), one.Revision)
	require.NotNil(t, one.Session)
	assert.NotContains(t, one.Session.Terminals[0].CommandStates, "doors")
	assert.Contains(t, one.Session.Terminals[0].CommandStates, "alarm")

	all := service.ResetTerminalCommandStates(t.Context(), "t1")
	require.True(t, all.OK, "ResetTerminalCommandStates() = %#v", all)
	assert.True(t, all.Changed)
	assert.Equal(t, uint64(4), all.Revision)
	require.NotNil(t, all.Session)
	assert.Empty(t, all.Session.Terminals[0].CommandStates)
	writesBeforeNoOp := len(fileSystem.WriteCalls())
	noOp := service.ResetTerminalCommandStates(t.Context(), "t1")
	require.True(t, noOp.OK, "idempotent ResetTerminalCommandStates() = %#v", noOp)
	assert.False(t, noOp.Changed)
	assert.Equal(t, uint64(4), noOp.Revision)
	require.NotNil(t, noOp.Session)
	assert.Equal(t, writesBeforeNoOp, len(fileSystem.WriteCalls()))
	snapshot := service.Snapshot()
	assert.Equal(t, uint64(4), snapshot.RequestedRevision)
	assert.Equal(t, uint64(4), snapshot.SavedRevision)
	assert.Equal(t, SaveStateSaved, snapshot.SaveState)
}

func TestStaleFullSavePreservesCanonicalFrozenStateAndAppliesAuthoredEdits(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/stale-save.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("before")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)
	require.NotNil(t, opened.Session)
	stale := *opened.Session

	executed := service.ExecuteCommandState(t.Context(), "t1", "doors")
	require.True(t, executed.OK, "ExecuteCommandState() = %#v", executed)
	assert.True(t, executed.Changed)
	assert.Equal(t, uint64(1), executed.Revision)
	stale.Name = "after"
	stale.Terminals[0].Root.Children[0].Name = "Открыть гермодвери"
	stale.Terminals[0].Root.Children[0].Text = "Новый результат для следующего выполнения."
	stale.Terminals[0].Root.Children[0].StateChange.CompletedName = "Гермодвери открыты"
	stale.Terminals[0].CommandStates = map[string]domain.CommandExecutionState{
		"doors": {CompletedName: "ПОДДЕЛКА", ResultText: "ПОДДЕЛКА"},
	}

	saved := service.Save(t.Context(), stale, 2)
	require.True(t, saved.OK, "Save(stale) = %#v", saved)
	assert.Equal(t, uint64(2), saved.RequestedRevision)
	assert.Equal(t, uint64(2), saved.SavedRevision)
	active := service.Snapshot()
	require.NotNil(t, active.Session)
	assert.Equal(t, "after", active.Session.Name)
	command := active.Session.Terminals[0].Root.Children[0]
	assert.Equal(t, "Открыть гермодвери", command.Name)
	require.NotNil(t, command.StateChange)
	assert.Equal(t, "Гермодвери открыты", command.StateChange.CompletedName)
	wantFrozen := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
	assert.Equal(t, wantFrozen, active.Session.Terminals[0].CommandStates["doors"])
	reopened, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	require.NoError(t, err)
	assert.Equal(t, wantFrozen, reopened.Terminals[0].CommandStates["doors"])
}

func TestFullSavePrunesFrozenStateWhenCommandIsDeleted(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/delete-command.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSession("before")))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)
	result := service.ExecuteCommandState(t.Context(), "t1", "doors")
	require.True(t, result.OK, "ExecuteCommandState() = %#v", result)
	assert.Equal(t, uint64(1), result.Revision)

	candidate := *service.Snapshot().Session
	candidate.Terminals[0].Root.Children = append([]domain.ContentNode(nil), candidate.Terminals[0].Root.Children[1:]...)
	saveResult := service.Save(t.Context(), candidate, 2)
	require.True(t, saveResult.OK, "Save(with deletion) = %#v", saveResult)
	assert.Equal(t, uint64(2), saveResult.SavedRevision)
	active := service.Snapshot()
	require.NotNil(t, active.Session)
	assert.Empty(t, active.Session.Terminals[0].CommandStates)
	written, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	require.NoError(t, err)
	assert.Empty(t, written.Terminals[0].CommandStates)
}

func TestStableIDAndFrozenStateRulesAcross100CompletedCommands(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Volumes/Campaigns/one-hundred-completed-commands.json"
	fileSystem.SeedFile(target, mustEncodeSession(t, stateChangingSessionWith100CompletedCommands()))
	service := NewService(NewStorage(fileSystem), &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = service.Shutdown(context.WithoutCancel(t.Context())) })
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)
	require.NotNil(t, opened.Session)

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

	saved := service.Save(t.Context(), candidate, 1)
	require.True(t, saved.OK, "Save(rename/move/delete 100 commands) = %#v", saved)
	assert.Equal(t, uint64(1), saved.SavedRevision)
	active := service.Snapshot()
	require.NotNil(t, active.Session)
	require.Len(t, active.Session.Terminals[0].CommandStates, 100)
	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		wantFrozen := domain.CommandExecutionState{
			CompletedName: fmt.Sprintf("Frozen completed %03d", index),
			ResultText:    fmt.Sprintf("Frozen result %03d", index),
		}
		assert.Equal(t, wantFrozen, active.Session.Terminals[0].CommandStates[commandID], "command %q", commandID)
	}

	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		reset := service.ResetCommandState(t.Context(), "t100", commandID)
		require.True(t, reset.OK, "ResetCommandState(%q) = %#v", commandID, reset)
		assert.True(t, reset.Changed, "command %q", commandID)
		executed := service.ExecuteCommandState(t.Context(), "t100", commandID)
		require.True(t, executed.OK, "ExecuteCommandState(%q) after reset = %#v", commandID, executed)
		assert.True(t, executed.Changed, "command %q", commandID)
		require.NotNil(t, executed.Session)
		wantNext := domain.CommandExecutionState{
			CompletedName: fmt.Sprintf("Next completed %03d", index),
			ResultText:    fmt.Sprintf("Next result %03d", index),
		}
		assert.Equal(t, wantNext, executed.Session.Terminals[0].CommandStates[commandID], "command %q", commandID)
	}
	active = service.Snapshot()
	require.NotNil(t, active.Session)
	require.Len(t, active.Session.Terminals[0].CommandStates, 100)

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
	deleted := service.Save(t.Context(), deletedCandidate, deleteRevision)
	require.True(t, deleted.OK, "Save(delete all 100 commands and add replacements) = %#v", deleted)
	assert.Equal(t, deleteRevision, deleted.SavedRevision)
	active = service.Snapshot()
	require.NotNil(t, active.Session)
	assert.Empty(t, active.Session.Terminals[0].CommandStates)
	for index := range 100 {
		commandID := fmt.Sprintf("command-%03d", index)
		replacementID := fmt.Sprintf("replacement-%03d", index)
		assert.NotContains(t, active.Session.Terminals[0].CommandStates, commandID)
		assert.NotContains(t, active.Session.Terminals[0].CommandStates, replacementID)
	}

	reopened, err := domain.DecodeSession(fileSystemFileData(t, fileSystem, target))
	require.NoError(t, err)
	assert.Empty(t, reopened.Terminals[0].CommandStates)
}

func TestFailedCommandStateMutationKeepsPriorDocumentAndRevision(t *testing.T) {
	t.Parallel()

	target := "/Volumes/Campaigns/failed-command-state.json"
	initial := mustEncodeSession(t, stateChangingSession("safe"))
	store := &failingMutationStore{path: target, data: initial, err: fmt.Errorf("injected atomic replacement failure")}
	service := NewService(store, &testutil.FakeDialog{OpenResult: target}, testLocations)
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() = %#v", opened)

	for attempt := range 100 {
		result := service.ExecuteCommandState(t.Context(), "t1", "doors")
		assert.False(t, result.OK, "attempt %d", attempt)
		assert.False(t, result.Changed, "attempt %d", attempt)
		assert.NotEmpty(t, result.Error, "attempt %d", attempt)
		assert.Zero(t, result.Revision, "attempt %d", attempt)
		assert.Nil(t, result.Session, "attempt %d", attempt)
		assert.Equal(t, attempt+1, store.writes, "attempt %d", attempt)
		assert.Equal(t, initial, store.data, "attempt %d", attempt)
		active := service.Snapshot()
		assert.Zero(t, active.RequestedRevision, "attempt %d", attempt)
		assert.Zero(t, active.SavedRevision, "attempt %d", attempt)
		require.NotNil(t, active.Session)
		assert.Empty(t, active.Session.Terminals[0].CommandStates, "attempt %d", attempt)
	}
	require.NoError(t, service.Shutdown(t.Context()))

	restarted := NewService(store, &testutil.FakeDialog{OpenResult: target}, testLocations)
	t.Cleanup(func() { _ = restarted.Shutdown(context.WithoutCancel(t.Context())) })
	reopened := restarted.Open(t.Context())
	require.True(t, reopened.OK, "reopen after 100 failed mutations = %#v", reopened)
	require.NotNil(t, reopened.Session)
	assert.Empty(t, reopened.Session.Terminals[0].CommandStates)
	assert.Equal(t, initial, store.data)
}

func TestCompletedCommandStateSurvivesFreshProcessReopen(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "process-restart-session.json")
	initial := mustEncodeSession(t, stateChangingSession("process restart"))
	require.NoError(t, os.WriteFile(target, initial, 0o600))
	executable, err := os.Executable()
	require.NoError(t, err)

	for _, mode := range []string{"execute", "reopen"} {
		command := exec.CommandContext(t.Context(), executable, "-test.run=^TestCommandStateFreshProcessHelper$", "-test.v")
		command.Env = append(os.Environ(),
			"FALLOUT_COMMAND_STATE_PROCESS_MODE="+mode,
			"FALLOUT_COMMAND_STATE_PROCESS_PATH="+target,
		)
		output, runErr := command.CombinedOutput()
		require.NoError(t, runErr, "fresh process %q failed:\n%s", mode, output)
	}

	durable, err := domain.DecodeSession(mustReadFile(t, target))
	require.NoError(t, err)
	want := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
	assert.Equal(t, want, durable.Terminals[0].CommandStates["doors"])
}

func TestCommandStateFreshProcessHelper(t *testing.T) {
	mode := os.Getenv("FALLOUT_COMMAND_STATE_PROCESS_MODE")
	if mode == "" {
		t.Skip("helper runs only in a fresh subprocess")
	}
	target := os.Getenv("FALLOUT_COMMAND_STATE_PROCESS_PATH")
	require.NotEmpty(t, target, "FALLOUT_COMMAND_STATE_PROCESS_PATH is empty")

	service := NewService(NewStorage(nil), &testutil.FakeDialog{OpenResult: target}, testLocations)
	opened := service.Open(t.Context())
	require.True(t, opened.OK, "Open() in %s process = %#v", mode, opened)
	require.NotNil(t, opened.Session)
	switch mode {
	case "execute":
		result := service.ExecuteCommandState(t.Context(), "t1", "doors")
		require.True(t, result.OK, "ExecuteCommandState() in fresh process = %#v", result)
		assert.True(t, result.Changed)
		require.NotNil(t, result.Session)
	case "reopen":
		want := domain.CommandExecutionState{CompletedName: "Двери открыты", ResultText: "Доступ в сектор разрешён."}
		assert.Equal(t, want, opened.Session.Terminals[0].CommandStates["doors"])
	default:
		require.Failf(t, "unknown helper mode", "%q", mode)
	}
	require.NoError(t, service.Shutdown(t.Context()), "mode %s", mode)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "ReadFile(%q)", path)
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
	assert.Empty(t, snapshot.Path)
	assert.Nil(t, snapshot.Session)
	assert.Zero(t, snapshot.RequestedRevision)
	assert.Zero(t, snapshot.SavedRevision)
	assert.Equal(t, SaveStateIdle, snapshot.SaveState)
}

func assertNoApplicationSupportWrites(t *testing.T, fileSystem *testutil.FakeFileSystem) {
	t.Helper()
	for _, write := range fileSystem.WriteCalls() {
		assert.False(t, strings.HasPrefix(write.Path, testLocations.ApplicationSupport+string(filepath.Separator)), "session content written to Application Support: %q", write.Path)
	}
}

func fileSystemFileData(t *testing.T, fileSystem *testutil.FakeFileSystem, path string) []byte {
	t.Helper()
	data, ok := fileSystem.File(path)
	require.True(t, ok, "file %q does not exist", path)
	return data
}

func mustEncodeSession(t *testing.T, session domain.Session) []byte {
	t.Helper()
	data, err := domain.EncodeSession(session)
	require.NoError(t, err)
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
	require.GreaterOrEqual(t, service.Snapshot().RequestedRevision, want, "requested revision never reached; state = %#v", service.Snapshot())
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

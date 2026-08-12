package session

import (
	"bytes"
	"context"
	"fmt"
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
	demoData, err := domain.EncodeSession(validSession("demo"))
	if err != nil {
		t.Fatal(err)
	}
	fileSystem.SeedFile(testLocations.BundledDemo, demoData)
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
	if got := fileSystemFileData(t, fileSystem, destination); !bytes.Equal(got, demoData) {
		t.Fatalf("writable copy = %s, want %s", got, demoData)
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

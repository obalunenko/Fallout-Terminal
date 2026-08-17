package session

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
)

func TestStorageWriteAtomicUsesPrivateSameDirectoryTemporaryFile(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	storage := NewStorage(fileSystem)
	target := "/Users/test/Documents/Fallout Terminal/Sessions/campaign.json"
	want := []byte("{\n  \"version\": 1\n}\n")

	if err := storage.WriteAtomic(target, want); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	mkdirs := fileSystem.MkdirCalls()
	if len(mkdirs) != 1 || mkdirs[0].Path != filepath.Dir(target) {
		t.Fatalf("MkdirAll calls = %#v, want target parent", mkdirs)
	}
	writes := fileSystem.WriteCalls()
	if len(writes) != 1 {
		t.Fatalf("WriteFile calls = %#v, want one temporary write", writes)
	}
	temporary := writes[0].Path
	if temporary == target || filepath.Dir(temporary) != filepath.Dir(target) {
		t.Fatalf("temporary path = %q, want a distinct path in %q", temporary, filepath.Dir(target))
	}
	if writes[0].Perm != fs.FileMode(0o600) {
		t.Fatalf("temporary permissions = %#o, want 0600", writes[0].Perm)
	}
	if !bytes.Equal(writes[0].Data, want) {
		t.Fatalf("temporary data = %q, want %q", writes[0].Data, want)
	}
	if got, wantRenames := fileSystem.RenameCalls(), []testutil.FileRename{{OldPath: temporary, NewPath: target}}; !reflect.DeepEqual(got, wantRenames) {
		t.Fatalf("Rename calls = %#v, want %#v", got, wantRenames)
	}
	if got, ok := fileSystem.File(target); !ok || !bytes.Equal(got, want) {
		t.Fatalf("target = %q, %t; want complete replacement", got, ok)
	}
	if _, ok := fileSystem.File(temporary); ok {
		t.Fatalf("temporary file %q remains after replacement", temporary)
	}
}

func TestStorageWriteAtomicKeepsOldTargetAndCleansTemporaryOnRenameFailure(t *testing.T) {
	t.Parallel()

	base := testutil.NewFakeFileSystem()
	fileSystem := &renameFailingFileSystem{FakeFileSystem: base, err: errors.New("volume unavailable")}
	storage := NewStorage(fileSystem)
	target := "/Users/test/Documents/Fallout Terminal/Sessions/campaign.json"
	oldData := []byte("old complete document\n")
	base.SeedFile(target, oldData)

	err := storage.WriteAtomic(target, []byte("new document\n"))
	if err == nil {
		t.Fatal("WriteAtomic() error = nil, want rename failure")
	}
	if got, ok := base.File(target); !ok || !bytes.Equal(got, oldData) {
		t.Fatalf("failed replacement changed target: %q, %t", got, ok)
	}
	writes := base.WriteCalls()
	if len(writes) != 1 {
		t.Fatalf("WriteFile calls = %#v, want one temporary write", writes)
	}
	if _, ok := base.File(writes[0].Path); ok {
		t.Fatalf("temporary file %q remains after rename failure", writes[0].Path)
	}
	if got := base.RemoveCalls(); !reflect.DeepEqual(got, []string{writes[0].Path}) {
		t.Fatalf("Remove calls = %#v, want temporary cleanup", got)
	}
}

func TestStorageWriteAtomicKeepsOldTargetAndSkipsRenameOnTemporaryWriteFailure(t *testing.T) {
	t.Parallel()

	base := testutil.NewFakeFileSystem()
	fileSystem := &writeFailingFileSystem{FakeFileSystem: base, err: errors.New("disk full")}
	storage := NewStorage(fileSystem)
	target := "/Users/test/Documents/Fallout Terminal/Sessions/campaign.json"
	oldData := []byte("old complete document\n")
	base.SeedFile(target, oldData)

	err := storage.WriteAtomic(target, []byte("new document\n"))
	if err == nil {
		t.Fatal("WriteAtomic() error = nil, want temporary write failure")
	}
	if got, ok := base.File(target); !ok || !bytes.Equal(got, oldData) {
		t.Fatalf("failed temporary write changed target: %q, %t", got, ok)
	}
	writes := base.WriteCalls()
	if len(writes) != 1 {
		t.Fatalf("WriteFile calls = %#v, want one failed temporary write", writes)
	}
	if renames := base.RenameCalls(); len(renames) != 0 {
		t.Fatalf("failed temporary write attempted rename: %#v", renames)
	}
	if _, ok := base.File(writes[0].Path); ok {
		t.Fatalf("failed temporary file %q remains", writes[0].Path)
	}
	if got := base.RemoveCalls(); !reflect.DeepEqual(got, []string{writes[0].Path}) {
		t.Fatalf("Remove calls = %#v, want failed temporary cleanup", got)
	}
}

func TestStorageCopyAtomicLeavesBundledDemoUnchanged(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	storage := NewStorage(fileSystem)
	source := "/Applications/Fallout Terminal.app/Contents/Resources/sessions/demo.json"
	destination := "/Users/test/Documents/Fallout Terminal/Sessions/demo-copy.json"
	demo := []byte("{\n  \"version\": 1,\n  \"name\": \"demo\",\n  \"terminals\": []\n}\n")
	fileSystem.SeedFile(source, demo)

	if err := storage.CopyAtomic(source, destination); err != nil {
		t.Fatalf("CopyAtomic() error = %v", err)
	}
	if got, ok := fileSystem.File(source); !ok || !bytes.Equal(got, demo) {
		t.Fatalf("bundled source changed: %q, %t", got, ok)
	}
	if got, ok := fileSystem.File(destination); !ok || !bytes.Equal(got, demo) {
		t.Fatalf("copied demo = %q, %t; want %q", got, ok, demo)
	}
	if got := fileSystem.ReadCalls(); !reflect.DeepEqual(got, []string{source}) {
		t.Fatalf("ReadFile calls = %v, want bundled demo only", got)
	}
	if renames := fileSystem.RenameCalls(); len(renames) != 1 || renames[0].NewPath != destination {
		t.Fatalf("Rename calls = %#v, want atomic destination replacement", renames)
	}
}

type renameFailingFileSystem struct {
	*testutil.FakeFileSystem
	err error
}

func (fileSystem *renameFailingFileSystem) Rename(oldPath, newPath string) error {
	fileSystem.RenameErrors[oldPath] = fileSystem.err
	return fileSystem.FakeFileSystem.Rename(oldPath, newPath)
}

type writeFailingFileSystem struct {
	*testutil.FakeFileSystem
	err error
}

func (fileSystem *writeFailingFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	fileSystem.WriteErrors[path] = fileSystem.err
	return fileSystem.FakeFileSystem.WriteFile(path, data, perm)
}

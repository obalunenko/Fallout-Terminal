// Package session owns durable version-1 session persistence.
package session

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
)

const (
	privateDirectoryPermissions fs.FileMode = 0o700
	privateFilePermissions      fs.FileMode = 0o600
)

// FileSystem is the narrow filesystem boundary needed for atomic session
// persistence. Implementations must keep Rename atomic when oldPath and
// newPath are in the same directory.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Rename(oldPath, newPath string) error
	Remove(path string) error
	MkdirAll(path string, perm fs.FileMode) error
}

// Storage reads user-selected documents and replaces them atomically.
type Storage struct {
	fileSystem FileSystem
	sequence   atomic.Uint64
}

// NewStorage constructs storage over fileSystem. Passing nil selects the real
// operating-system filesystem.
func NewStorage(fileSystem FileSystem) *Storage {
	if fileSystem == nil {
		fileSystem = osFileSystem{}
	}
	return &Storage{fileSystem: fileSystem}
}

// Read returns a detached copy of a session document.
func (storage *Storage) Read(path string) ([]byte, error) {
	if storage == nil || storage.fileSystem == nil {
		return nil, errors.New("session storage is not configured")
	}
	path, err := validateFilePath("session path", path)
	if err != nil {
		return nil, err
	}
	data, err := storage.fileSystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session file: %w", err)
	}
	return append([]byte(nil), data...), nil
}

// WriteAtomic writes data to a private temporary file beside path, flushes it
// when using the real filesystem, and then replaces path with one rename. The
// previous target remains intact if the temporary write or rename fails.
func (storage *Storage) WriteAtomic(path string, data []byte) error {
	if storage == nil || storage.fileSystem == nil {
		return errors.New("session storage is not configured")
	}
	path, err := validateFilePath("session path", path)
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)
	if err := storage.fileSystem.MkdirAll(directory, privateDirectoryPermissions); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	temporary := storage.temporaryPath(path)
	if err := storage.fileSystem.WriteFile(temporary, data, privateFilePermissions); err != nil {
		cleanupErr := storage.removeTemporary(temporary)
		return errors.Join(fmt.Errorf("write temporary session file: %w", err), cleanupErr)
	}
	if err := storage.fileSystem.Rename(temporary, path); err != nil {
		cleanupErr := storage.removeTemporary(temporary)
		return errors.Join(fmt.Errorf("replace session file: %w", err), cleanupErr)
	}
	return nil
}

// CopyAtomic explicitly copies a read-only bundled document to a distinct,
// user-selected destination. The source is read only and is never renamed,
// removed, or otherwise mutated.
func (storage *Storage) CopyAtomic(source, destination string) error {
	if storage == nil || storage.fileSystem == nil {
		return errors.New("session storage is not configured")
	}
	source, err := validateFilePath("copy source", source)
	if err != nil {
		return err
	}
	destination, err = validateFilePath("copy destination", destination)
	if err != nil {
		return err
	}
	if source == destination {
		return errors.New("copy destination must differ from bundled source")
	}

	data, err := storage.fileSystem.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read bundled session: %w", err)
	}
	if err := storage.WriteAtomic(destination, data); err != nil {
		return fmt.Errorf("copy bundled session: %w", err)
	}
	return nil
}

func (storage *Storage) temporaryPath(target string) string {
	sequence := storage.sequence.Add(1)
	name := fmt.Sprintf(".%s.%d.%d.tmp", filepath.Base(target), os.Getpid(), sequence)
	return filepath.Join(filepath.Dir(target), name)
}

func (storage *Storage) removeTemporary(path string) error {
	err := storage.fileSystem.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("clean temporary session file: %w", err)
}

func validateFilePath(name, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s is empty", name)
	}
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == string(filepath.Separator) {
		return "", fmt.Errorf("%s does not name a file", name)
	}
	return cleaned, nil
}

// osFileSystem flushes complete private files before Storage renames them.
// O_EXCL prevents a pre-existing path from being followed or overwritten.
type osFileSystem struct{}

func (osFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (osFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}

	written, writeErr := file.Write(data)
	if writeErr == nil && written != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

func (osFileSystem) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (osFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (osFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

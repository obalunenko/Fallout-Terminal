// Package playerconfig owns native selection and durable authored roster files.
package playerconfig

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

var errStorageUnavailable = errors.New("player config storage is unavailable")

// Dialog is the native trusted file-dialog boundary. Empty paths mean cancel.
type Dialog interface {
	OpenFile(defaultPath string) (string, error)
	SaveFile(defaultPath string) (string, error)
}

// Store is the complete-file persistence boundary.
type Store interface {
	Read(path string) ([]byte, error)
	WriteAtomic(path string, data []byte) error
}

// Result reports a selected, created, or referenced player config.
type Result struct {
	OK       bool                 `json:"ok"`
	Canceled bool                 `json:"canceled"`
	Error    string               `json:"error,omitempty"`
	FilePath string               `json:"filePath,omitempty"`
	Config   *domain.PlayerConfig `json:"config,omitempty"`
}

// Service owns player-config dialogs, strict decoding, and atomic saves.
type Service struct {
	store            Store
	dialog           Dialog
	documentsDefault string
}

// NewService constructs an idle player-config service.
func NewService(store Store, dialog Dialog, documentsDefault string) *Service {
	return &Service{store: store, dialog: dialog, documentsDefault: documentsDefault}
}

// Create asks for a path and atomically creates an empty version-1 config.
func (service *Service) Create(ctx context.Context) Result {
	if err := contextError(ctx); err != nil {
		return failure("new player config was canceled")
	}
	if service == nil || service.dialog == nil {
		return failure("new player config dialog is unavailable")
	}
	target, err := service.dialog.SaveFile(filepath.Join(service.documentsDefault, "players.json"))
	if err != nil {
		return failure("could not choose a new player config destination")
	}
	if target == "" {
		return Result{Canceled: true}
	}
	target, err = absoluteFilePath(target)
	if err != nil {
		return failure(err.Error())
	}
	config := domain.PlayerConfig{Version: 1, Name: nameFromPath(target), Roster: []domain.CharacterRosterEntry{}}
	if err := verifyPlayerConfigContract(config); err != nil {
		return failure("could not verify the new player config contract")
	}
	if err := service.write(target, config); err != nil {
		return failure("could not write the new player config")
	}
	return success(target, config)
}

// Open asks for and strictly loads one existing player config.
func (service *Service) Open(ctx context.Context) Result {
	if err := contextError(ctx); err != nil {
		return failure("open player config was canceled")
	}
	if service == nil || service.dialog == nil {
		return failure("open player config dialog is unavailable")
	}
	target, err := service.dialog.OpenFile(service.documentsDefault)
	if err != nil {
		return failure("could not choose a player config")
	}
	if target == "" {
		return Result{Canceled: true}
	}
	target, err = absoluteFilePath(target)
	if err != nil {
		return failure(err.Error())
	}
	return service.load(target)
}

// LoadReferenced resolves a normalized relative reference from a session file.
func (service *Service) LoadReferenced(sessionPath, reference string) Result {
	sessionPath, err := absoluteFilePath(sessionPath)
	if err != nil {
		return failure("active session path is invalid")
	}
	reference = strings.TrimSpace(reference)
	if reference == "" || filepath.IsAbs(reference) || filepath.Clean(reference) == "." || strings.ContainsRune(reference, '\x00') {
		return failure("player config reference must be a relative file path")
	}
	return service.load(filepath.Clean(filepath.Join(filepath.Dir(sessionPath), reference)))
}

// Save atomically replaces the complete active roster before coordinator commit.
func (service *Service) Save(handle domain.PlayerConfigHandle, roster []domain.CharacterRosterEntry) error {
	path, err := absoluteFilePath(handle.Path)
	if err != nil {
		return err
	}
	config := domain.PlayerConfig{Version: handle.Version, Name: handle.Name, Roster: cloneRoster(roster)}
	return service.write(path, config)
}

// RelativeReference normalizes configPath relative to the session directory.
func RelativeReference(sessionPath, configPath string) (string, error) {
	sessionPath, err := absoluteFilePath(sessionPath)
	if err != nil {
		return "", fmt.Errorf("active session path is invalid")
	}
	configPath, err = absoluteFilePath(configPath)
	if err != nil {
		return "", fmt.Errorf("player config path is invalid")
	}
	reference, err := filepath.Rel(filepath.Dir(sessionPath), configPath)
	if err != nil || filepath.IsAbs(reference) || filepath.Clean(reference) == "." {
		return "", fmt.Errorf("could not create a relative player config reference")
	}
	return filepath.Clean(reference), nil
}

func (service *Service) load(path string) Result {
	if service == nil || service.store == nil {
		return failure(errStorageUnavailable.Error())
	}
	data, err := service.store.Read(path)
	if err != nil {
		return failure("could not read the player config")
	}
	config, err := domain.DecodePlayerConfig(data)
	if err != nil {
		return failure("the selected file is not a valid version-1 player config")
	}
	if err := verifyPlayerConfigContract(config); err != nil {
		return failure("the selected file does not satisfy the player config contract")
	}
	return success(path, config)
}

func (service *Service) write(path string, config domain.PlayerConfig) error {
	if service == nil || service.store == nil {
		return errStorageUnavailable
	}
	if err := verifyPlayerConfigContract(config); err != nil {
		return err
	}
	data, err := domain.EncodePlayerConfig(config)
	if err != nil {
		return err
	}
	if err := service.store.WriteAtomic(path, data); err != nil {
		return fmt.Errorf("write player config: %w", err)
	}
	return nil
}

func absoluteFilePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || strings.ContainsRune(path, '\x00') {
		return "", fmt.Errorf("player config path is invalid")
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || path == string(filepath.Separator) {
		return "", fmt.Errorf("player config path must be an absolute file path")
	}
	return path, nil
}

func nameFromPath(path string) string {
	name := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if name == "" {
		return "Players"
	}
	return name
}

func success(path string, config domain.PlayerConfig) Result {
	copy := config
	copy.Roster = cloneRoster(config.Roster)
	return Result{OK: true, FilePath: path, Config: &copy}
}

func failure(message string) Result { return Result{Error: message} }

func cloneRoster(roster []domain.CharacterRosterEntry) []domain.CharacterRosterEntry {
	if roster == nil {
		return nil
	}
	cloned := make([]domain.CharacterRosterEntry, len(roster))
	copy(cloned, roster)
	return cloned
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

package statestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"git-safe/internal/domain"
	"git-safe/internal/ports"
)

const (
	stateDirName  = ".gitsafe"
	pathsFileName = "paths.json"
	keysFileName  = "keys.dat"
	version       = 1
)

type fileListDocument struct {
	Version int                 `json:"version"`
	Files   []domain.SecretFile `json:"files"`
}

type keyListDocument struct {
	Version int          `json:"version"`
	Keys    []domain.Key `json:"keys"`
}

// FileStateStore файловая реализация порта хранения состояния.
type FileStateStore struct{}

// NewFileStateStore создает файловую реализацию StateStore.
func NewFileStateStore() ports.StateStore {
	return &FileStateStore{}
}

// IsInitialized проверяет наличие каталога состояния .gitsafe.
func (s *FileStateStore) IsInitialized(repoRoot string) (bool, error) {
	_, err := os.Stat(s.stateDir(repoRoot))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Init инициализирует каталог состояния и пустые документы paths/keys.
func (s *FileStateStore) Init(repoRoot string) error {
	if err := os.MkdirAll(s.stateDir(repoRoot), 0o770); err != nil {
		return err
	}
	if err := s.StoreFiles(repoRoot, []domain.SecretFile{}); err != nil {
		return err
	}
	return s.StoreKeys(repoRoot, []domain.Key{})
}

// LoadFiles загружает список файлов из paths.json.
func (s *FileStateStore) LoadFiles(repoRoot string) ([]domain.SecretFile, error) {
	var doc fileListDocument
	if err := readJSON(s.pathsFile(repoRoot), &doc); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.SecretFile{}, nil
		}
		return nil, err
	}
	if doc.Version != 0 && doc.Version != version {
		return nil, fmt.Errorf("unsupported paths state version %d", doc.Version)
	}
	return doc.Files, nil
}

// StoreFiles сохраняет список файлов в paths.json.
func (s *FileStateStore) StoreFiles(repoRoot string, files []domain.SecretFile) error {
	doc := fileListDocument{Version: version, Files: files}
	return writeJSONAtomic(s.pathsFile(repoRoot), doc)
}

// LoadKeys загружает список ключей из keys.dat.
func (s *FileStateStore) LoadKeys(repoRoot string) ([]domain.Key, error) {
	var doc keyListDocument
	if err := readJSON(s.keysFile(repoRoot), &doc); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.Key{}, nil
		}
		return nil, err
	}
	if doc.Version != 0 && doc.Version != version {
		return nil, fmt.Errorf("unsupported keys state version %d", doc.Version)
	}
	return doc.Keys, nil
}

// StoreKeys сохраняет список ключей в keys.dat.
func (s *FileStateStore) StoreKeys(repoRoot string, keys []domain.Key) error {
	doc := keyListDocument{Version: version, Keys: keys}
	return writeJSONAtomic(s.keysFile(repoRoot), doc)
}

// stateDir возвращает путь до каталога состояния репозитория.
func (s *FileStateStore) stateDir(repoRoot string) string {
	return filepath.Join(repoRoot, stateDirName)
}

// pathsFile возвращает путь до файла paths.json.
func (s *FileStateStore) pathsFile(repoRoot string) string {
	return filepath.Join(s.stateDir(repoRoot), pathsFileName)
}

// keysFile возвращает путь до файла keys.dat.
func (s *FileStateStore) keysFile(repoRoot string) string {
	return filepath.Join(s.stateDir(repoRoot), keysFileName)
}

// readJSON читает и декодирует JSON-документ из файла.
func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

// writeJSONAtomic записывает JSON в temp-файл и атомарно переименовывает его в целевой путь.
func writeJSONAtomic(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o770); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	cleanup := func(retErr error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return retErr
	}

	if _, err := tmp.Write(data); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		return cleanup(err)
	}
	return os.Rename(tmpName, path)
}

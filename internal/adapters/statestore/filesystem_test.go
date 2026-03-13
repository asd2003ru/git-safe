package statestore

import (
	"path/filepath"
	"testing"

	"git-safe/internal/domain"
)

func TestFileStateStoreInitAndRoundTrip(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	s := &FileStateStore{}

	ok, err := s.IsInitialized(repo)
	if err != nil {
		t.Fatalf("is initialized: %v", err)
	}
	if ok {
		t.Fatalf("repo must not be initialized yet")
	}

	if err := s.Init(repo); err != nil {
		t.Fatalf("init store: %v", err)
	}

	ok, err = s.IsInitialized(repo)
	if err != nil || !ok {
		t.Fatalf("expected initialized repo, ok=%v err=%v", ok, err)
	}

	files := []domain.SecretFile{{Path: "a.txt", Hash: "h1"}}
	keys := []domain.Key{{ID: "k1", Type: domain.KeyTypeAGE, Key: "age1abc"}}
	if err := s.StoreFiles(repo, files); err != nil {
		t.Fatalf("store files: %v", err)
	}
	if err := s.StoreKeys(repo, keys); err != nil {
		t.Fatalf("store keys: %v", err)
	}

	gotFiles, err := s.LoadFiles(repo)
	if err != nil {
		t.Fatalf("load files: %v", err)
	}
	if len(gotFiles) != 1 || gotFiles[0].Path != "a.txt" {
		t.Fatalf("unexpected files state: %#v", gotFiles)
	}

	gotKeys, err := s.LoadKeys(repo)
	if err != nil {
		t.Fatalf("load keys: %v", err)
	}
	if len(gotKeys) != 1 || gotKeys[0].ID != "k1" {
		t.Fatalf("unexpected keys state: %#v", gotKeys)
	}

	if _, err := filepath.Abs(filepath.Join(repo, stateDirName, pathsFileName)); err != nil {
		t.Fatalf("paths file should exist: %v", err)
	}
}

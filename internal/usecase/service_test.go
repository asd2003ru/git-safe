package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"git-safe/cmd"
	"git-safe/internal/domain"
	"git-safe/internal/ports"
	"git-safe/internal/ucerr"
)

type fakeGit struct {
	root           string
	inside         bool
	addPatterns    []string
	removePatterns []string
}

func (g *fakeGit) IsInsideWorkTree(context.Context) (bool, error) { return g.inside, nil }
func (g *fakeGit) RepoRoot(context.Context) (string, error)       { return g.root, nil }
func (g *fakeGit) AddIgnorePattern(_ context.Context, pattern string) error {
	g.addPatterns = append(g.addPatterns, pattern)
	return nil
}
func (g *fakeGit) RemoveIgnorePattern(_ context.Context, pattern string) error {
	g.removePatterns = append(g.removePatterns, pattern)
	return nil
}

type memoryStore struct {
	initialized bool
	files       []domain.SecretFile
	keys        []domain.Key
}

func (s *memoryStore) IsInitialized(string) (bool, error) { return s.initialized, nil }
func (s *memoryStore) Init(string) error                  { s.initialized = true; return nil }
func (s *memoryStore) LoadFiles(string) ([]domain.SecretFile, error) {
	return slices.Clone(s.files), nil
}
func (s *memoryStore) StoreFiles(_ string, files []domain.SecretFile) error {
	s.files = slices.Clone(files)
	return nil
}
func (s *memoryStore) LoadKeys(string) ([]domain.Key, error) { return slices.Clone(s.keys), nil }
func (s *memoryStore) StoreKeys(_ string, keys []domain.Key) error {
	s.keys = slices.Clone(keys)
	return nil
}

type fakeCrypto struct{}

func (fakeCrypto) Encrypt(plain []byte, _ []domain.Key) ([]byte, error) {
	return append([]byte("enc:"), plain...), nil
}
func (fakeCrypto) Decrypt(cipher []byte, _ string) ([]byte, error) {
	if !strings.HasPrefix(string(cipher), "enc:") {
		return nil, errors.New("bad cipher")
	}
	return []byte(strings.TrimPrefix(string(cipher), "enc:")), nil
}
func (fakeCrypto) GenerateKeyPair() (string, string, error) { return "priv", "age1pub", nil }

type fakeKeyLoader struct{ key string }

func (k fakeKeyLoader) Load(string) (string, error) { return k.key, nil }

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Unix(0, 0) }

type fakeIO struct{ out bytes.Buffer }

func (io *fakeIO) Stdout() io.Writer { return &io.out }
func (io *fakeIO) Stderr() io.Writer { return &io.out }

func newTestService(t *testing.T, root string, store ports.StateStore) *Service {
	t.Helper()
	io := &fakeIO{}
	svc, err := NewService(Deps{
		Git:       &fakeGit{root: root, inside: true},
		Store:     store,
		Crypto:    fakeCrypto{},
		KeyLoader: fakeKeyLoader{key: "secret"},
		Clock:     fakeClock{},
		IO:        io,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func requireKind(t *testing.T, err error, want ucerr.Kind) {
	t.Helper()
	got, ok := ucerr.KindOf(err)
	if !ok {
		t.Fatalf("expected typed error, got: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected kind: got %q, want %q", got, want)
	}
}

func TestServiceAddTable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name      string
		files     []string
		initStore bool
		prepare   func(t *testing.T, root string)
		wantErr   ucerr.Kind
		wantFiles int
	}{
		{
			name:    "empty input",
			files:   nil,
			wantErr: ucerr.InvalidInput,
		},
		{
			name:  "not initialized",
			files: []string{"secret.txt"},
			prepare: func(t *testing.T, root string) {
				_ = os.WriteFile(filepath.Join(root, "secret.txt"), []byte("x"), 0o600)
			},
			wantErr: ucerr.NotInitialized,
		},
		{
			name:      "adds tracked file",
			files:     []string{"secret.txt"},
			initStore: true,
			prepare: func(t *testing.T, root string) {
				_ = os.WriteFile(filepath.Join(root, "secret.txt"), []byte("x"), 0o600)
			},
			wantFiles: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.prepare != nil {
				tt.prepare(t, root)
			}
			store := &memoryStore{initialized: tt.initStore}
			svc := newTestService(t, root, store)

			err := svc.Add(ctx, tt.files)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				requireKind(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("add failed: %v", err)
			}
			if len(store.files) != tt.wantFiles {
				t.Fatalf("tracked files mismatch: got %d, want %d", len(store.files), tt.wantFiles)
			}
		})
	}
}

func TestServiceHideRevealCycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	secretPath := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("super-secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	store := &memoryStore{
		initialized: true,
		files:       []domain.SecretFile{{Path: "secret.txt"}},
		keys:        []domain.Key{{Type: domain.KeyTypeAGE, Key: "age1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq", ID: "k1"}},
	}
	svc := newTestService(t, root, store)

	if err := svc.Hide(ctx, cmd.HideInput{Files: []string{"secret.txt"}, Clean: true}); err != nil {
		t.Fatalf("hide failed: %v", err)
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("source file should be removed after hide --clean")
	}
	if _, err := os.Stat(secretPath + ".safe"); err != nil {
		t.Fatalf("safe file should exist: %v", err)
	}

	if err := svc.Reveal(ctx, cmd.RevealInput{Files: []string{"secret.txt"}}); err != nil {
		t.Fatalf("reveal failed: %v", err)
	}
	got, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("read revealed file: %v", err)
	}
	if string(got) != "super-secret" {
		t.Fatalf("revealed content mismatch: %q", string(got))
	}
}

func TestServiceKeysAddTable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name    string
		input   cmd.KeysAddInput
		wantErr ucerr.Kind
	}{
		{
			name:    "age key requires id",
			input:   cmd.KeysAddInput{PublicKey: "age1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq"},
			wantErr: ucerr.InvalidInput,
		},
		{
			name:  "ssh key infers id",
			input: cmd.KeysAddInput{PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEc8mT4mS1WSV6xqxjWn0Q9m2i8WnB4fQ7IY2L3yVvWf user@test"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			store := &memoryStore{initialized: true}
			svc := newTestService(t, root, store)

			err := svc.KeysAdd(ctx, tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error")
				}
				requireKind(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("keys add failed: %v", err)
			}
			if len(store.keys) != 1 {
				t.Fatalf("key should be added")
			}
		})
	}
}

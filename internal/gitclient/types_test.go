package gitclient

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNewInvalidBackend(t *testing.T) {
	t.Parallel()

	_, err := New(Config{Backend: "unknown"})
	if err == nil {
		t.Fatalf("expected error for unknown backend")
	}
}

func TestNewFromEnvGoGit(t *testing.T) {
	t.Setenv(BackendEnv, BackendGoGit)
	c, err := NewFromEnv("")
	if err != nil {
		t.Fatalf("new from env: %v", err)
	}
	if _, ok := c.(*GoGitClient); !ok {
		t.Fatalf("expected GoGitClient, got %T", c)
	}
}

func TestGoGitIsIgnored(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}

	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("secret.txt\n"), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "secret.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := NewGoGit(repo)
	ignored, err := c.IsIgnored(context.Background(), "secret.txt")
	if err != nil {
		t.Fatalf("is ignored: %v", err)
	}
	if !ignored {
		t.Fatalf("file should be ignored")
	}
}

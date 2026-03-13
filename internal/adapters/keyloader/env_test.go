package keyloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvKeyLoaderPriority(t *testing.T) {
	dir := t.TempDir()
	flagFile := filepath.Join(dir, "flag.key")
	envFile := filepath.Join(dir, "env.key")
	if err := os.WriteFile(flagFile, []byte("flag-key\n"), 0o600); err != nil {
		t.Fatalf("write flag file: %v", err)
	}
	if err := os.WriteFile(envFile, []byte("file-key\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	l := &EnvKeyLoader{}
	t.Setenv(envKey, "env-key")
	t.Setenv(envKeyFile, envFile)

	got, err := l.Load(flagFile)
	if err != nil || got != "flag-key" {
		t.Fatalf("flag priority failed: got=%q err=%v", got, err)
	}

	got, err = l.Load("")
	if err != nil || got != "env-key" {
		t.Fatalf("env var priority failed: got=%q err=%v", got, err)
	}

	t.Setenv(envKey, "")
	got, err = l.Load("")
	if err != nil || got != "file-key" {
		t.Fatalf("env file fallback failed: got=%q err=%v", got, err)
	}
}

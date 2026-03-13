package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	bin := filepath.Join(t.TempDir(), "git-safe")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = projectRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build binary: %v\n%s", err, string(out))
	}
	return bin
}

func TestCLIInitAddHideRevealFlow(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)

	repo := t.TempDir()
	gitInit := exec.Command("git", "init")
	gitInit.Dir = repo
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}

	secretPath := filepath.Join(repo, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("secret-data"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init")
	run("keys", "generate", "--keyfile", "team.key", "--pubfile", "team.pub")
	run("keys", "add", "--keyfile", "team.key", "--id", "team", "--pubfile", "team.pub")
	run("add", "secret.txt")
	run("hide", "--keyfile", "team.key", "--clean")

	if _, err := os.Stat(secretPath + ".safe"); err != nil {
		t.Fatalf("safe file missing: %v", err)
	}
	if _, err := os.Stat(secretPath); !os.IsNotExist(err) {
		t.Fatalf("source file should be removed after hide --clean")
	}

	run("reveal", "--keyfile", "team.key")
	data, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("read revealed secret: %v", err)
	}
	if string(data) != "secret-data" {
		t.Fatalf("revealed content mismatch: %q", string(data))
	}
}

func TestCLIRoleAccessAndKeyRemove(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	repo := t.TempDir()

	gitInit := exec.Command("git", "init")
	gitInit.Dir = repo
	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}

	secretPath := filepath.Join(repo, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("top-secret"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	runOK := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, string(out))
		}
	}
	runFailWithCode := func(wantCode int, args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("%v should fail, output: %s", args, string(out))
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("unexpected error type: %T", err)
		}
		if exitErr.ExitCode() != wantCode {
			t.Fatalf("%v exit code mismatch: got %d, want %d, out=%s", args, exitErr.ExitCode(), wantCode, string(out))
		}
	}

	runOK("init")
	runOK("keys", "generate", "--keyfile", "rw.key", "--pubfile", "rw.pub")
	runOK("keys", "generate", "--keyfile", "ro.key", "--pubfile", "ro.pub")
	runOK("keys", "add", "--keyfile", "rw.key", "--id", "rw", "--pubfile", "rw.pub")
	runOK("keys", "add", "--keyfile", "rw.key", "--id", "ro", "--pubfile", "ro.pub", "--readonly")
	runOK("add", "secret.txt")
	runOK("hide", "--keyfile", "rw.key", "--clean")

	// RO-ключ должен иметь возможность только на reveal.
	runOK("reveal", "--keyfile", "ro.key")
	runFailWithCode(6, "keys", "list", "--keyfile", "ro.key")
	runFailWithCode(6, "keys", "add", "--keyfile", "ro.key", "--id", "x", "--pubfile", "rw.pub")
	runFailWithCode(6, "keys", "remove", "--keyfile", "ro.key", "--id", "rw")

	// RW-ключ может удалять ключи.
	runOK("keys", "remove", "--keyfile", "rw.key", "--id", "ro")
	runFailWithCode(7, "keys", "remove", "--keyfile", "rw.key", "--id", "ro")
}

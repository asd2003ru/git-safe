package tests

import (
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/adapters/cryptoage"
	"github.com/asd2003ru/git-safe/internal/adapters/gitgogit"
	"github.com/asd2003ru/git-safe/internal/adapters/keyloader"
	"github.com/asd2003ru/git-safe/internal/adapters/osfs"
	"github.com/asd2003ru/git-safe/internal/adapters/sha256hash"
	"github.com/asd2003ru/git-safe/internal/adapters/statefs"
	"github.com/asd2003ru/git-safe/internal/usecase"
)

type namedTest struct {
	name string
	test func(*testing.T, *usecase.Service)
}

type suite struct {
	name  string
	tests []namedTest
}

const oneKey = "one.key"
const onePublicKey = "one.pub"
const anotherKey = "another.key"
const anotherPublicKey = "another.pub"

var cwd string

func init() {
	cwd, _ = os.Getwd()
}

func copyFile(src string, dst string, t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newService() *usecase.Service {
	return newServiceWithInput("")
}

func newServiceWithInput(input string) *usecase.Service {
	git := gitgogit.New()
	fs := osfs.New()
	hasher := sha256hash.New()
	crypto := cryptoage.New()
	loader := keyloader.New(fs, crypto)
	state := statefs.New(git)
	return usecase.NewService(git, fs, hasher, state, loader, crypto, strings.NewReader(input), os.Stdout, os.Stderr)
}

func setupAndInit(t *testing.T) *usecase.Service {
	t.Helper()
	return setupAndInitWithInput(t, "")
}

func setupAndInitWithInput(t *testing.T, input string) *usecase.Service {
	t.Helper()

	testDir := t.TempDir()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("failed to move to test directory: %v", err)
	}

	for _, keyFile := range []string{oneKey, onePublicKey, anotherKey, anotherPublicKey} {
		copyFile(filepath.Join(cwd, "testkeys", keyFile), filepath.Join(testDir, keyFile), t)
	}

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to move to tmp directory: %v", err)
	}

	git := exec.Command("git", "init")
	if err := git.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	svc := newServiceWithInput(input)
	if err := svc.CheckSetup(); err != nil {
		t.Fatalf("check setup failed: %v", err)
	}
	if err := svc.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}
	return svc
}

func makeFile(name string, t *testing.T) {
	t.Helper()
	data := make([]byte, 5150)
	_, _ = rand.Read(data)
	if err := os.WriteFile(name, data, 0o660); err != nil {
		t.Fatalf("failed to create test file")
	}
}

func runAll(s suite, t *testing.T) {
	t.Helper()
	for _, test := range s.tests {
		svc := setupAndInit(t)
		if !t.Run(s.name+"-"+test.name, func(t *testing.T) {
			test.test(t, svc)
		}) {
			break
		}
	}
	_ = os.Chdir(cwd)
}

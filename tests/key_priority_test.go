package tests

import (
	"os"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestKeyPriorityFlagOverEnv(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}

	makeFile("secret", t)
	if err := svc.Add([]string{"secret"}); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("GIT_SAFE_KEYFILE", anotherKey); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("GIT_SAFE_KEYFILE")

	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey}); err != nil {
		t.Fatalf("expected flag keyfile to override env: %v", err)
	}
}

func TestLegacyEnvKeyFileStillWorks(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}

	makeFile("secret", t)
	if err := svc.Add([]string{"secret"}); err != nil {
		t.Fatal(err)
	}

	if err := os.Setenv("GIT_SAFE_KEY", ""); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("GIT_SAFE_KEY")
	if err := os.Setenv("GIT_SAFE_KEYFILE", ""); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("GIT_SAFE_KEYFILE")
	if err := os.Setenv("GIT_PRIVATE_KEYFILE", oneKey); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv("GIT_PRIVATE_KEYFILE")

	if err := svc.Hide(usecase.HideOptions{}); err != nil {
		t.Fatalf("expected legacy keyfile env to work: %v", err)
	}
}

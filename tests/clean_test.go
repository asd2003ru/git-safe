package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestClean(t *testing.T) {
	runAll(suite{
		name: "clean",
		tests: []namedTest{
			{name: "removes synced revealed file", test: testCleanRemovesSyncedRevealedFile},
			{name: "refuses modified file", test: testCleanRefusesModifiedFile},
			{name: "force removes modified file", test: testCleanForceRemovesModifiedFile},
			{name: "refuses missing private file", test: testCleanRefusesMissingPrivateFile},
		},
	}, t)
}

func setupHiddenFile(t *testing.T, svc *usecase.Service, name string) {
	t.Helper()
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	makeFile(name, t)
	if err := svc.Add([]string{name}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
}

func testCleanRemovesSyncedRevealedFile(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupHiddenFile(t, svc, "secret")

	if err := svc.Clean(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("secret"); !os.IsNotExist(err) {
		t.Fatal("revealed file should be removed")
	}
	if _, err := os.Stat("secret.safe"); err != nil {
		t.Fatalf("encrypted file should stay: %v", err)
	}
}

func testCleanRefusesModifiedFile(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupHiddenFile(t, svc, "secret")
	if err := os.WriteFile("secret", []byte("changed"), 0o660); err != nil {
		t.Fatal(err)
	}

	err := svc.Clean(false)
	if err == nil {
		t.Fatal("expected clean to refuse modified file")
	}
	if !strings.Contains(err.Error(), "out of sync") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat("secret"); statErr != nil {
		t.Fatalf("modified file should stay: %v", statErr)
	}
}

func testCleanForceRemovesModifiedFile(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupHiddenFile(t, svc, "secret")
	if err := os.WriteFile("secret", []byte("changed"), 0o660); err != nil {
		t.Fatal(err)
	}

	if err := svc.Clean(true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("secret"); !os.IsNotExist(err) {
		t.Fatal("force clean should remove modified file")
	}
}

func testCleanRefusesMissingPrivateFile(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupHiddenFile(t, svc, "secret")
	if err := os.Remove("secret.safe"); err != nil {
		t.Fatal(err)
	}

	err := svc.Clean(false)
	if err == nil {
		t.Fatal("expected clean to refuse file with missing private file")
	}
	if !strings.Contains(err.Error(), "missing private file") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat("secret"); statErr != nil {
		t.Fatalf("revealed file should stay: %v", statErr)
	}
}

package tests

import (
	"os"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestKeysListAndRemove(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "ro", PubFile: anotherPublicKey, KeyFile: oneKey, ReadOnly: true}); err != nil {
		t.Fatal(err)
	}

	keys, err := svc.KeysList(oneKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0].ID != "rw" || keys[1].ID != "ro" || !keys[1].ReadOnly {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	if err := svc.KeysRemove(oneKey, "ro"); err != nil {
		t.Fatal(err)
	}
	keys, err = svc.KeysList(oneKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].ID != "rw" {
		t.Fatalf("key was not removed: %#v", keys)
	}
}

func TestKeysRemoveRequiresID(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysRemove(oneKey, " "); err == nil {
		t.Fatal("expected missing id error")
	}
}

func TestKeysRemoveMissingIDFails(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := svc.KeysRemove(oneKey, "missing"); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestKeysRemoveReencryptsTrackedFiles(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "second", PubFile: anotherPublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	makeFile("secret", t)
	if err := svc.Add([]string{"secret"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}

	if err := svc.KeysRemove(oneKey, "second"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove("secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reveal(usecase.RevealOptions{KeyFile: anotherKey}); err == nil {
		t.Fatal("removed key should not decrypt re-encrypted file")
	}
	if _, err := svc.Reveal(usecase.RevealOptions{KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("secret"); err != nil {
		t.Fatalf("remaining key should reveal file: %v", err)
	}
}

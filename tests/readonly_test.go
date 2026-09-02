package tests

import (
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestReadonly(t *testing.T) {
	runAll(suite{
		name: "readonly",
		tests: []namedTest{
			{name: "adding keys fails", test: testReadonlyAddKeyFails},
			{name: "hiding fails", test: testReadonlyHidingFails},
			{name: "reveal succeeds", test: testReadonlyRevealSucceeds},
		},
	}, t)
}

func setupKeys(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "ro", PubFile: anotherPublicKey, KeyFile: oneKey, ReadOnly: true}); err != nil {
		t.Fatal(err)
	}
}

func testReadonlyAddKeyFails(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupKeys(t, svc)
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "noway", PubFile: onePublicKey, KeyFile: anotherKey}); err == nil {
		t.Fatal("adding public key with readonly key should fail")
	}
}

func testReadonlyHidingFails(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupKeys(t, svc)
	makeFile("secret", t)
	if err := svc.Add([]string{"secret"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: anotherKey}); err == nil {
		t.Fatal("hiding with readonly key should fail")
	}
}

func testReadonlyRevealSucceeds(t *testing.T, svc *usecase.Service) {
	t.Helper()
	setupKeys(t, svc)
	makeFile("secret", t)
	if err := svc.Add([]string{"secret"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey, Clean: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reveal(usecase.RevealOptions{KeyFile: anotherKey}); err != nil {
		t.Fatal(err)
	}
}

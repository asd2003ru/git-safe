package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestRemove(t *testing.T) {
	runAll(suite{
		name: "remove",
		tests: []namedTest{
			{name: "no args", test: testRemoveNoArgsFails},
			{name: "no file", test: testRemoveNonExistingFileSucceeds},
			{name: "non-revealed file", test: testRemoveNonRevealedFileSucceeds},
			{name: "tracked directory", test: testRemoveTrackedDirectorySucceeds},
		},
	}, t)
}

func testRemoveNoArgsFails(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.Remove([]string{}); err == nil {
		t.Fatal("expected error")
	}
}

func testRemoveNonExistingFileSucceeds(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.Remove([]string{"nosuchfile"}); err != nil {
		t.Fatal(err)
	}
}

func testRemoveNonRevealedFileSucceeds(t *testing.T, svc *usecase.Service) {
	t.Helper()
	makeFile("mysecrets", t)

	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "hubba", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Add([]string{"mysecrets"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey, Clean: true}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove([]string{"mysecrets"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat("mysecrets.safe"); !os.IsNotExist(err) {
		t.Fatal("safe file left behind")
	}
}

func testRemoveTrackedDirectorySucceeds(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("secrets/nested", 0o770); err != nil {
		t.Fatal(err)
	}
	makeFile("secrets/root", t)
	makeFile("secrets/nested/child", t)
	if err := svc.Add([]string{"secrets"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey, Clean: true}); err != nil {
		t.Fatal(err)
	}

	if err := svc.Remove([]string{"secrets"}); err != nil {
		t.Fatal(err)
	}
	statuses, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 0 {
		t.Fatalf("directory files should be untracked: %#v", statuses)
	}
	if _, err := os.Stat("secrets/root.safe"); !os.IsNotExist(err) {
		t.Fatal("root safe file left behind")
	}
	if _, err := os.Stat("secrets/nested/child.safe"); !os.IsNotExist(err) {
		t.Fatal("nested safe file left behind")
	}

	ignore, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ignore), "secrets/**") {
		t.Fatalf("directory ignore pattern left behind: %s", string(ignore))
	}
}

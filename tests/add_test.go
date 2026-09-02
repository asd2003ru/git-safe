package tests

import (
	"os"
	"testing"

	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestAdd(t *testing.T) {
	runAll(suite{
		name: "add",
		tests: []namedTest{
			{name: "no args", test: testAddNoArgsFails},
			{name: "no file", test: testAddNonExistingFileFails},
			{name: "add single file", test: testAddSingleFileWorks},
			{name: "add directory", test: testAddDirectoryWorks},
			{name: "track new directory files", test: testAddDirectoryTracksNewFiles},
			{name: "hide new directory files", test: testHideEncryptsNewDirectoryFiles},
		},
	}, t)
}

func TestInitUsesNativeLayout(t *testing.T) {
	_ = setupAndInit(t)

	if _, err := os.Stat(".gitsafe/paths.json"); err != nil {
		t.Fatalf("native paths file is missing: %v", err)
	}
	if _, err := os.Stat(".gitprivate"); !os.IsNotExist(err) {
		t.Fatal("legacy state directory should not be created by init")
	}
}

func testAddNoArgsFails(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.Add([]string{}); err == nil {
		t.Fatal("expected error")
	}
}

func testAddNonExistingFileFails(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.Add([]string{"nosuchfile"}); err == nil {
		t.Fatal("expected error")
	}
}

func testAddSingleFileWorks(t *testing.T, svc *usecase.Service) {
	t.Helper()
	makeFile("single", t)
	if err := svc.Add([]string{"single"}); err != nil {
		t.Fatal(err)
	}

	statuses, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].File.Path != "single" {
		t.Fatal("file not added")
	}
}

func testAddDirectoryWorks(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := os.MkdirAll("secrets/nested", 0o770); err != nil {
		t.Fatal(err)
	}
	makeFile("secrets/root", t)
	makeFile("secrets/nested/child", t)
	makeFile("secrets/root.safe", t)

	if err := svc.Add([]string{"secrets"}); err != nil {
		t.Fatal(err)
	}

	statuses, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	paths := statusPaths(statuses)
	if len(paths) != 2 || paths[0] != "secrets/nested/child" || paths[1] != "secrets/root" {
		t.Fatalf("unexpected tracked files: %#v", paths)
	}
	assertFileContains(t, ".gitignore", "secrets/**")
	assertFileContains(t, ".gitignore", "!secrets/**/*"+".safe")
}

func testAddDirectoryTracksNewFiles(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := os.MkdirAll("secrets", 0o770); err != nil {
		t.Fatal(err)
	}
	makeFile("secrets/first", t)
	if err := svc.Add([]string{"secrets"}); err != nil {
		t.Fatal(err)
	}

	makeFile("secrets/second", t)
	statuses, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}

	paths := statusPaths(statuses)
	if len(paths) != 2 || paths[0] != "secrets/first" || paths[1] != "secrets/second" {
		t.Fatalf("new directory file was not tracked: %#v", paths)
	}
}

func testHideEncryptsNewDirectoryFiles(t *testing.T, svc *usecase.Service) {
	t.Helper()
	if err := svc.KeysAdd(usecase.KeysAddOptions{ID: "rw", PubFile: onePublicKey, KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("secrets", 0o770); err != nil {
		t.Fatal(err)
	}
	makeFile("secrets/first", t)
	if err := svc.Add([]string{"secrets"}); err != nil {
		t.Fatal(err)
	}

	makeFile("secrets/second", t)
	if err := svc.Hide(usecase.HideOptions{KeyFile: oneKey}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("secrets/second.safe"); err != nil {
		t.Fatalf("new directory file was not encrypted: %v", err)
	}
}

func statusPaths(statuses []domain.FileStatus) []string {
	paths := make([]string, 0, len(statuses))
	for _, status := range statuses {
		paths = append(paths, status.File.Path)
	}
	return paths
}

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestHookRequiresKeySource(t *testing.T) {
	svc := setupAndInit(t)

	t.Setenv("GIT_SAFE_KEY", "")
	t.Setenv("GIT_SAFE_KEYFILE", "")
	t.Setenv("GIT_PRIVATE_KEY", "")
	t.Setenv("GIT_PRIVATE_KEYFILE", "")

	err := svc.Hook(usecase.HookOptions{})
	if err == nil {
		t.Fatal("expected error when key source is missing")
	}
	if !strings.Contains(err.Error(), "git-safe hook -keyfile FILE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHookWithEnvKeyWritesHooksWithoutFlag(t *testing.T) {
	svc := setupAndInit(t)
	t.Setenv("GIT_SAFE_KEYFILE", oneKey)
	t.Setenv("GIT_SAFE_KEY", "")
	t.Setenv("GIT_PRIVATE_KEYFILE", "")
	t.Setenv("GIT_PRIVATE_KEY", "")

	if err := svc.Hook(usecase.HookOptions{}); err != nil {
		t.Fatal(err)
	}

	preCommit, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	postMerge, err := os.ReadFile(".git/hooks/post-merge")
	if err != nil {
		t.Fatal(err)
	}

	preCommitText := string(preCommit)
	if !strings.Contains(preCommitText, "git-safe hide\n") {
		t.Fatalf("unexpected pre-commit hook: %s", preCommitText)
	}
	if !strings.Contains(preCommitText, "mapfile -d '' safe_files < <(git ls-files -z --others --exclude-standard -- '*.safe')\n") {
		t.Fatalf("pre-commit must stage new safe files: %s", preCommitText)
	}
	if strings.Contains(preCommitText, "-keyfile") {
		t.Fatalf("pre-commit must not contain -keyfile when env is used: %s", preCommitText)
	}

	postMergeText := string(postMerge)
	if !strings.Contains(postMergeText, "git-safe reveal\n") {
		t.Fatalf("unexpected post-merge hook: %s", postMergeText)
	}
	if strings.Contains(postMergeText, "-keyfile") {
		t.Fatalf("post-merge must not contain -keyfile when env is used: %s", postMergeText)
	}

	preStat, err := os.Stat(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	if preStat.Mode()&0o100 == 0 {
		t.Fatal("pre-commit is not executable")
	}
}

func TestHookWithKeyFileWritesFlagToHooks(t *testing.T) {
	svc := setupAndInit(t)
	keyPath := filepath.Join(cwd, "testkeys", "with space.key")

	if err := svc.Hook(usecase.HookOptions{KeyFile: "./testkeys/with space.key"}); err != nil {
		t.Fatal(err)
	}

	preCommit, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	postMerge, err := os.ReadFile(".git/hooks/post-merge")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(preCommit), "git-safe hide -keyfile '"+keyPath+"'\n") {
		t.Fatalf("unexpected pre-commit hook: %s", string(preCommit))
	}
	if !strings.Contains(string(postMerge), "git-safe reveal -keyfile '"+keyPath+"'\n") {
		t.Fatalf("unexpected post-merge hook: %s", string(postMerge))
	}
}

func TestHookWithKeyWritesFlagToHooks(t *testing.T) {
	svc := setupAndInit(t)

	keyData, err := os.ReadFile(oneKey)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimSpace(string(keyData))

	if err := svc.Hook(usecase.HookOptions{Key: key}); err != nil {
		t.Fatal(err)
	}

	preCommit, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	postMerge, err := os.ReadFile(".git/hooks/post-merge")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(preCommit), "git-safe hide -key '"+key+"'\n") {
		t.Fatalf("unexpected pre-commit hook: %s", string(preCommit))
	}
	if !strings.Contains(string(postMerge), "git-safe reveal -key '"+key+"'\n") {
		t.Fatalf("unexpected post-merge hook: %s", string(postMerge))
	}
	if strings.Contains(string(preCommit), "-keyfile") || strings.Contains(string(postMerge), "-keyfile") {
		t.Fatalf("hooks must prefer -key over -keyfile:\n%s\n%s", string(preCommit), string(postMerge))
	}
}

func TestHookAsksAndCancelsOverwrite(t *testing.T) {
	svc := setupAndInitWithInput(t, "\n")
	t.Setenv("GIT_SAFE_KEYFILE", oneKey)

	if err := os.WriteFile(".git/hooks/pre-commit", []byte("# old pre\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".git/hooks/post-merge", []byte("# old post\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := svc.Hook(usecase.HookOptions{})
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	pre, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	if string(pre) != "# old pre\n" {
		t.Fatalf("pre-commit must stay unchanged: %s", string(pre))
	}
	if _, err = os.Stat(".git/hooks/pre-commit.bak"); !os.IsNotExist(err) {
		t.Fatal("backup should not be created when overwrite is canceled")
	}
}

func TestHookOverwritesAndCreatesBackup(t *testing.T) {
	svc := setupAndInitWithInput(t, "y\n")
	t.Setenv("GIT_SAFE_KEYFILE", oneKey)

	if err := os.WriteFile(".git/hooks/pre-commit", []byte("# old pre\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".git/hooks/post-merge", []byte("# old post\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := svc.Hook(usecase.HookOptions{}); err != nil {
		t.Fatal(err)
	}

	preBak, err := os.ReadFile(".git/hooks/pre-commit.bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(preBak) != "# old pre\n" {
		t.Fatalf("unexpected pre-commit backup: %s", string(preBak))
	}

	postBak, err := os.ReadFile(".git/hooks/post-merge.bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(postBak) != "# old post\n" {
		t.Fatalf("unexpected post-merge backup: %s", string(postBak))
	}

	pre, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pre), "git-safe hide\n") {
		t.Fatalf("unexpected new pre-commit: %s", string(pre))
	}
}

func TestHookAcceptsLegacyEnvKeyFile(t *testing.T) {
	svc := setupAndInit(t)
	t.Setenv("GIT_SAFE_KEY", "")
	t.Setenv("GIT_SAFE_KEYFILE", "")
	t.Setenv("GIT_PRIVATE_KEY", "")
	t.Setenv("GIT_PRIVATE_KEYFILE", oneKey)

	if err := svc.Hook(usecase.HookOptions{}); err != nil {
		t.Fatal(err)
	}
}

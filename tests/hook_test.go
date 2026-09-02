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
	if !strings.Contains(err.Error(), "git-safe hook --keyfile FILE") {
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
	if strings.Contains(preCommitText, " --keyfile ") {
		t.Fatalf("pre-commit must not contain --keyfile when env is used: %s", preCommitText)
	}

	postMergeText := string(postMerge)
	if !strings.Contains(postMergeText, "git-safe reveal\n") {
		t.Fatalf("unexpected post-merge hook: %s", postMergeText)
	}
	if strings.Contains(postMergeText, " --keyfile ") {
		t.Fatalf("post-merge must not contain --keyfile when env is used: %s", postMergeText)
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

	if !strings.Contains(string(preCommit), "git-safe hide --keyfile '"+keyPath+"'\n") {
		t.Fatalf("unexpected pre-commit hook: %s", string(preCommit))
	}
	if !strings.Contains(string(postMerge), "git-safe reveal --keyfile '"+keyPath+"'\n") {
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

	if !strings.Contains(string(preCommit), "git-safe hide --key '"+key+"'\n") {
		t.Fatalf("unexpected pre-commit hook: %s", string(preCommit))
	}
	if !strings.Contains(string(postMerge), "git-safe reveal --key '"+key+"'\n") {
		t.Fatalf("unexpected post-merge hook: %s", string(postMerge))
	}
	if strings.Contains(string(preCommit), " --keyfile ") || strings.Contains(string(postMerge), " --keyfile ") {
		t.Fatalf("hooks must prefer --key over --keyfile:\n%s\n%s", string(preCommit), string(postMerge))
	}
}

func TestHookAppendsManagedBlockToExistingHooks(t *testing.T) {
	svc := setupAndInit(t)
	t.Setenv("GIT_SAFE_KEYFILE", oneKey)

	preExisting := "#!/bin/sh\n\necho old pre\n"
	postExisting := "#!/bin/sh\n\necho old post\n"
	if err := os.WriteFile(".git/hooks/pre-commit", []byte(preExisting), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".git/hooks/post-merge", []byte(postExisting), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := svc.Hook(usecase.HookOptions{}); err != nil {
		t.Fatal(err)
	}

	pre, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	preText := string(pre)
	if !strings.Contains(preText, "echo old pre\n") {
		t.Fatalf("pre-commit must keep existing content: %s", preText)
	}
	if !strings.Contains(preText, "# >>> git-safe\n") || !strings.Contains(preText, "# <<< git-safe\n") {
		t.Fatalf("pre-commit must contain managed block: %s", preText)
	}
	if !strings.Contains(preText, "git-safe hide\n") {
		t.Fatalf("pre-commit must contain git-safe hide: %s", preText)
	}
	if strings.Count(preText, "# >>> git-safe") != 1 {
		t.Fatalf("pre-commit must contain one managed block: %s", preText)
	}

	post, err := os.ReadFile(".git/hooks/post-merge")
	if err != nil {
		t.Fatal(err)
	}
	postText := string(post)
	if !strings.Contains(postText, "echo old post\n") {
		t.Fatalf("post-merge must keep existing content: %s", postText)
	}
	if !strings.Contains(postText, "git-safe reveal\n") {
		t.Fatalf("post-merge must contain git-safe reveal: %s", postText)
	}
}

func TestHookUpdatesExistingManagedBlock(t *testing.T) {
	svc := setupAndInit(t)
	t.Setenv("GIT_SAFE_KEYFILE", oneKey)

	oldHook := "#!/bin/sh\n\n# >>> git-safe\necho old git-safe\n# <<< git-safe\n\necho project check\n"
	if err := os.WriteFile(".git/hooks/pre-commit", []byte(oldHook), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := svc.Hook(usecase.HookOptions{KeyFile: "./testkeys/with space.key"}); err != nil {
		t.Fatal(err)
	}

	pre, err := os.ReadFile(".git/hooks/pre-commit")
	if err != nil {
		t.Fatal(err)
	}
	preText := string(pre)
	if strings.Contains(preText, "echo old git-safe") {
		t.Fatalf("old managed block must be replaced: %s", preText)
	}
	if !strings.Contains(preText, "echo project check\n") {
		t.Fatalf("project hook content must be preserved: %s", preText)
	}
	if strings.Count(preText, "# >>> git-safe") != 1 || strings.Count(preText, "# <<< git-safe") != 1 {
		t.Fatalf("managed block must not be duplicated: %s", preText)
	}
	if !strings.Contains(preText, "git-safe hide --keyfile '"+filepath.Join(cwd, "testkeys", "with space.key")+"'\n") {
		t.Fatalf("managed block must be updated with new keyfile: %s", preText)
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

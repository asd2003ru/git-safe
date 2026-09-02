package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func setupLegacyRepo(t *testing.T) *usecase.Service {
	t.Helper()

	testDir := t.TempDir()
	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to move to tmp directory: %v", err)
	}
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	if err := os.MkdirAll(".gitprivate", 0o770); err != nil {
		t.Fatal(err)
	}
	paths := `{"Version":1,"Files":[{"Path":"secret","Hash":"abc"}]}`
	if err := os.WriteFile(".gitprivate/paths.json", []byte(paths), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".gitprivate/keys.dat", []byte("encrypted keys"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("secret.private", []byte("encrypted secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".gitignore", []byte("secret\n!*.private\n"), 0o660); err != nil {
		t.Fatal(err)
	}

	svc := newService()
	if err := svc.CheckSetup(); err != nil {
		t.Fatalf("check setup failed: %v", err)
	}
	return svc
}

func TestMigrateLegacyLayoutToNativeLayout(t *testing.T) {
	svc := setupLegacyRepo(t)

	result, err := svc.Migrate(usecase.MigrateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.StateFilesCopied != 2 || result.EncryptedCopied != 1 || !result.LegacyRemoved {
		t.Fatalf("unexpected result: %+v", result)
	}

	assertFileContains(t, ".gitsafe/paths.json", `"Path":"secret"`)
	assertFileContains(t, ".gitsafe/keys.dat", "encrypted keys")
	assertFileContains(t, "secret.safe", "encrypted secret")

	if _, err := os.Stat(".gitprivate"); !os.IsNotExist(err) {
		t.Fatal("legacy state directory should be removed")
	}
	if _, err := os.Stat("secret.private"); !os.IsNotExist(err) {
		t.Fatal("legacy encrypted file should be removed")
	}

	ignore, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignore), "!*.safe") {
		t.Fatalf("native ignore exception is missing: %s", string(ignore))
	}
	if strings.Contains(string(ignore), "!*.private") {
		t.Fatalf("legacy ignore exception should be removed: %s", string(ignore))
	}
}

func TestMigrateDryRunDoesNotWriteFiles(t *testing.T) {
	svc := setupLegacyRepo(t)

	result, err := svc.Migrate(usecase.MigrateOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.StateFilesCopied != 2 || result.EncryptedCopied != 1 || result.LegacyRemoved {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if _, err := os.Stat(".gitsafe"); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create native state directory")
	}
	if _, err := os.Stat("secret.safe"); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create native encrypted file")
	}
}

func TestInitRefusesLegacyLayout(t *testing.T) {
	svc := setupLegacyRepo(t)

	err := svc.Init()
	if err == nil {
		t.Fatal("expected init to refuse legacy state")
	}
	if !strings.Contains(err.Error(), "git-safe migrate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q: %s", path, want, string(data))
	}
}

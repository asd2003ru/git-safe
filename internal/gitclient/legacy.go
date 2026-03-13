package gitclient

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// LegacyClient реализация git-клиента через вызов внешней команды git.
type LegacyClient struct {
	workDir string
}

// NewLegacy создает git-клиент на основе внешней команды git.
func NewLegacy(workDir string) *LegacyClient {
	return &LegacyClient{workDir: workDir}
}

// IsInsideWorkTree проверяет, что текущая директория находится внутри work tree.
func (c *LegacyClient) IsInsideWorkTree(ctx context.Context) (bool, error) {
	_, code, err := c.runGit(ctx, "rev-parse", "--is-inside-work-tree")
	if code == 0 {
		return true, nil
	}
	return false, err
}

// RepoRoot определяет абсолютный путь до корня репозитория.
func (c *LegacyClient) RepoRoot(ctx context.Context) (string, error) {
	out, code, err := c.runGit(ctx, "rev-parse", "--path-format=relative", "--show-toplevel")
	if code != 0 {
		return "", err
	}
	root := strings.TrimSpace(out)
	root = filepath.Clean(root)
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return root, nil
}

// IsIgnored проверяет, игнорируется ли файл правилами gitignore.
func (c *LegacyClient) IsIgnored(ctx context.Context, fileName string) (bool, error) {
	_, code, err := c.runGit(ctx, "check-ignore", "-q", fileName)
	if code == 0 {
		return true, nil
	}
	return false, err
}

// AddIgnorePattern добавляет шаблон в .gitignore корня репозитория.
func (c *LegacyClient) AddIgnorePattern(ctx context.Context, pattern string) error {
	root, err := c.RepoRoot(ctx)
	if err != nil {
		return err
	}
	return addIgnorePattern(root, pattern)
}

// RemoveIgnorePattern удаляет шаблон из .gitignore корня репозитория.
func (c *LegacyClient) RemoveIgnorePattern(ctx context.Context, pattern string) error {
	root, err := c.RepoRoot(ctx)
	if err != nil {
		return err
	}
	return removeIgnorePattern(root, pattern)
}

// runGit выполняет git-команду и возвращает stdout, exit code и ошибку выполнения.
func (c *LegacyClient) runGit(ctx context.Context, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if strings.TrimSpace(c.workDir) != "" {
		cmd.Dir = c.workDir
	}
	bytes, err := cmd.Output()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if exitCode != -1 {
		err = nil
	}
	if err != nil {
		return "", exitCode, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(bytes), exitCode, nil
}

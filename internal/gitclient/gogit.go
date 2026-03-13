package gitclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// GoGitClient реализация git-клиента на библиотеке go-git.
type GoGitClient struct {
	workDir string
}

// NewGoGit создает git-клиент на базе библиотеки go-git.
func NewGoGit(workDir string) *GoGitClient {
	return &GoGitClient{workDir: workDir}
}

// IsInsideWorkTree проверяет наличие репозитория в текущем или родительских каталогах.
func (c *GoGitClient) IsInsideWorkTree(_ context.Context) (bool, error) {
	root, err := discoverRepoRoot(c.workDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	_, err = git.PlainOpenWithOptions(root, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return false, err
	}
	return true, nil
}

// RepoRoot возвращает корень репозитория после проверки, что .git валиден.
func (c *GoGitClient) RepoRoot(_ context.Context) (string, error) {
	root, err := discoverRepoRoot(c.workDir)
	if err != nil {
		return "", err
	}
	_, err = git.PlainOpenWithOptions(root, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return "", err
	}
	return root, nil
}

// IsIgnored вычисляет игнорирование файла на основе цепочки .gitignore.
func (c *GoGitClient) IsIgnored(ctx context.Context, fileName string) (bool, error) {
	root, err := c.RepoRoot(ctx)
	if err != nil {
		return false, err
	}

	target := fileName
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(root, absTarget)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(rel, "..") {
		return false, fmt.Errorf("path %q is outside repository root %q", absTarget, root)
	}

	patterns, err := collectIgnorePatterns(root, rel)
	if err != nil {
		return false, err
	}
	matcher := gitignore.NewMatcher(patterns)
	return matcher.Match(splitPath(rel), false), nil
}

// AddIgnorePattern добавляет правило в .gitignore корня репозитория.
func (c *GoGitClient) AddIgnorePattern(ctx context.Context, pattern string) error {
	root, err := c.RepoRoot(ctx)
	if err != nil {
		return err
	}
	return addIgnorePattern(root, pattern)
}

// RemoveIgnorePattern удаляет правило из .gitignore корня репозитория.
func (c *GoGitClient) RemoveIgnorePattern(ctx context.Context, pattern string) error {
	root, err := c.RepoRoot(ctx)
	if err != nil {
		return err
	}
	return removeIgnorePattern(root, pattern)
}

// collectIgnorePatterns собирает все релевантные правила .gitignore от корня до файла.
func collectIgnorePatterns(root string, relPath string) ([]gitignore.Pattern, error) {
	var patterns []gitignore.Pattern

	dirs := ignoreSearchDirs(relPath)
	for _, dirRel := range dirs {
		ignoreFile := filepath.Join(root, dirRel, ".gitignore")
		filePatterns, err := parseIgnoreFile(ignoreFile, splitPath(dirRel))
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, filePatterns...)
	}

	return patterns, nil
}

// ignoreSearchDirs строит список директорий, в которых нужно искать .gitignore.
func ignoreSearchDirs(relPath string) []string {
	clean := filepath.Clean(relPath)
	dir := filepath.Dir(clean)
	if dir == "." {
		return []string{""}
	}

	parts := splitPath(dir)
	dirs := make([]string, 0, len(parts)+1)
	dirs = append(dirs, "")
	for i := 0; i < len(parts); i++ {
		dirs = append(dirs, filepath.Join(parts[:i+1]...))
	}
	return dirs
}

// parseIgnoreFile читает .gitignore и превращает строки в gitignore.Pattern.
func parseIgnoreFile(path string, domain []string) ([]gitignore.Pattern, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	patterns := make([]gitignore.Pattern, 0)
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, gitignore.ParsePattern(line, domain))
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

// splitPath нормализует путь и возвращает набор его сегментов.
func splitPath(path string) []string {
	if strings.TrimSpace(path) == "" || path == "." {
		return []string{}
	}
	normalized := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(normalized, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" && p != "." {
			out = append(out, p)
		}
	}
	return out
}

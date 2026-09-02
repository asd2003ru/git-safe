package gitgogit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Adapter implements GitClient using go-git without invoking the git CLI.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) IsInsideWorkTree() (bool, error) {
	_, err := a.findRepoRoot()
	if err != nil {
		if isNotRepoError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (a *Adapter) GetRootPath() (string, error) {
	return a.findRepoRoot()
}

func (a *Adapter) IsIgnored(path string) (bool, error) {
	root, err := a.GetRootPath()
	if err != nil {
		return false, err
	}

	abs := path
	if !filepath.IsAbs(abs) {
		abs, err = filepath.Abs(abs)
		if err != nil {
			return false, err
		}
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return false, nil
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return false, nil
	}

	cleanRel := filepath.ToSlash(filepath.Clean(rel))
	parts := splitPath(cleanRel)
	if len(parts) == 0 {
		return false, nil
	}

	patterns, err := a.readRootIgnorePatterns(root)
	if err != nil {
		return false, err
	}
	if len(patterns) == 0 {
		return false, nil
	}
	matcher := gitignore.NewMatcher(patterns)

	isDir := false
	if stat, statErr := os.Stat(abs); statErr == nil {
		isDir = stat.IsDir()
	}

	if matcher.Match(parts, isDir) {
		return true, nil
	}
	if matcher.Match(parts, !isDir) {
		return true, nil
	}

	return false, nil
}

func (a *Adapter) readRootIgnorePatterns(root string) ([]gitignore.Pattern, error) {
	path := filepath.Join(root, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	patterns := make([]gitignore.Pattern, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		patterns = append(patterns, gitignore.ParsePattern(trimmed, nil))
	}
	return patterns, nil
}

func (a *Adapter) AddIgnorePattern(pattern string) error {
	lines, err := a.readIgnoreFile()
	if err != nil {
		return err
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(pattern) {
			return nil
		}
	}

	lines = append(lines, pattern)
	return a.writeIgnoreFile(lines)
}

func (a *Adapter) RemoveIgnorePattern(pattern string) error {
	lines, err := a.readIgnoreFile()
	if err != nil {
		return err
	}

	updated := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(pattern) {
			updated = append(updated, line)
		}
	}

	return a.writeIgnoreFile(updated)
}

func (a *Adapter) findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		dotGit := filepath.Join(dir, ".git")
		if _, statErr := os.Stat(dotGit); statErr == nil {
			// Verify that this is really a git repository.
			if _, openErr := git.PlainOpenWithOptions(dir, &git.PlainOpenOptions{
				DetectDotGit:          false,
				EnableDotGitCommonDir: true,
			}); openErr == nil {
				return filepath.Clean(dir), nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in dir with git repo")
}

func isNotRepoError(err error) bool {
	return strings.Contains(err.Error(), "not in dir with git repo")
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func (a *Adapter) ignoreFilePath() (string, error) {
	root, err := a.GetRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".gitignore"), nil
}

func (a *Adapter) readIgnoreFile() ([]string, error) {
	path, err := a.ignoreFilePath()
	if err != nil {
		return nil, err
	}
	if _, err = os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func (a *Adapter) writeIgnoreFile(lines []string) error {
	path, err := a.ignoreFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o660)
}

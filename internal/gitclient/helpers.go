package gitclient

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// discoverRepoRoot поднимается по дереву каталогов и ищет ближайший .git.
func discoverRepoRoot(workDir string) (string, error) {
	start := workDir
	if strings.TrimSpace(start) == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	absStart, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	cur := absStart
	for {
		gitMeta := filepath.Join(cur, ".git")
		if _, err := os.Stat(gitMeta); err == nil {
			return cur, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(cur)
		if parent == cur {
			return "", os.ErrNotExist
		}
		cur = parent
	}
}

// readIgnoreLines читает строки .gitignore в корне репозитория.
func readIgnoreLines(repoRoot string) ([]string, error) {
	ignorePath := filepath.Join(repoRoot, ".gitignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

// writeIgnoreLines полностью перезаписывает .gitignore переданными строками.
func writeIgnoreLines(repoRoot string, lines []string) error {
	ignorePath := filepath.Join(repoRoot, ".gitignore")
	return os.WriteFile(ignorePath, []byte(strings.Join(lines, "\n")), 0o660)
}

// addIgnorePattern добавляет правило в .gitignore, если его там еще нет.
func addIgnorePattern(repoRoot string, pattern string) error {
	lines, err := readIgnoreLines(repoRoot)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == strings.TrimSpace(pattern) {
			return nil
		}
	}
	lines = append(lines, pattern)
	return writeIgnoreLines(repoRoot, lines)
}

// removeIgnorePattern удаляет правило из .gitignore по точному совпадению.
func removeIgnorePattern(repoRoot string, pattern string) error {
	lines, err := readIgnoreLines(repoRoot)
	if err != nil {
		return err
	}

	updated := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(pattern) {
			updated = append(updated, line)
		}
	}
	return writeIgnoreLines(repoRoot, updated)
}

package gitclient

import (
	"context"
	"fmt"
	"os"
	"strings"
)

const (
	// BackendLegacy использует внешний git CLI.
	BackendLegacy = "legacy"
	// BackendGoGit использует библиотеку go-git.
	BackendGoGit  = "go-git"
)

// BackendEnv имя переменной окружения выбора git backend.
const BackendEnv = "GIT_SAFE_GIT_BACKEND"

// Client описывает минимальные git-операции, нужные use-case слою.
type Client interface {
	IsInsideWorkTree(ctx context.Context) (bool, error)
	RepoRoot(ctx context.Context) (string, error)
	IsIgnored(ctx context.Context, fileName string) (bool, error)
	AddIgnorePattern(ctx context.Context, pattern string) error
	RemoveIgnorePattern(ctx context.Context, pattern string) error
}

// Config задает настройки создания git-клиента.
type Config struct {
	Backend string
	WorkDir string
}

// New создает клиент указанного backend типа.
func New(cfg Config) (Client, error) {
	backend := strings.TrimSpace(cfg.Backend)
	if backend == "" {
		backend = BackendLegacy
	}

	switch backend {
	case BackendLegacy:
		return NewLegacy(cfg.WorkDir), nil
	case BackendGoGit:
		return NewGoGit(cfg.WorkDir), nil
	default:
		return nil, fmt.Errorf("unsupported git backend %q", backend)
	}
}

// NewFromEnv создает клиент, выбирая backend из переменной окружения.
func NewFromEnv(workDir string) (Client, error) {
	backend := strings.TrimSpace(os.Getenv(BackendEnv))
	if backend == "" {
		backend = BackendLegacy
	}
	return New(Config{Backend: backend, WorkDir: workDir})
}

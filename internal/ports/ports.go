package ports

import (
	"context"
	"io"
	"time"

	"git-safe/internal/domain"
)

// Git порт для git-операций, необходимых бизнес-логике.
type Git interface {
	// IsInsideWorkTree проверяет, что текущий каталог находится внутри git-репозитория.
	IsInsideWorkTree(ctx context.Context) (bool, error)
	// RepoRoot возвращает абсолютный путь до корня репозитория.
	RepoRoot(ctx context.Context) (string, error)
	// AddIgnorePattern добавляет шаблон в .gitignore.
	AddIgnorePattern(ctx context.Context, pattern string) error
	// RemoveIgnorePattern удаляет шаблон из .gitignore.
	RemoveIgnorePattern(ctx context.Context, pattern string) error
}

// StateStore порт хранения состояния git-safe (файлы и ключи).
type StateStore interface {
	// IsInitialized проверяет, инициализировано ли состояние git-safe в репозитории.
	IsInitialized(repoRoot string) (bool, error)
	// Init создает начальную структуру состояния.
	Init(repoRoot string) error
	// LoadFiles загружает список отслеживаемых секретных файлов.
	LoadFiles(repoRoot string) ([]domain.SecretFile, error)
	// StoreFiles сохраняет список отслеживаемых секретных файлов.
	StoreFiles(repoRoot string, files []domain.SecretFile) error
	// LoadKeys загружает список публичных ключей.
	LoadKeys(repoRoot string) ([]domain.Key, error)
	// StoreKeys сохраняет список публичных ключей.
	StoreKeys(repoRoot string, keys []domain.Key) error
}

// Crypto порт шифрования, расшифровки и генерации ключей.
type Crypto interface {
	// Encrypt шифрует данные на набор получателей из списка ключей.
	Encrypt(plaintext []byte, keys []domain.Key) ([]byte, error)
	// Decrypt расшифровывает данные с использованием приватной identity.
	Decrypt(ciphertext []byte, identity string) ([]byte, error)
	// GenerateKeyPair генерирует новую пару AGE-ключей.
	GenerateKeyPair() (privateKey string, publicKey string, err error)
}

// KeyLoader порт загрузки приватной identity из внешних источников.
type KeyLoader interface {
	// Load загружает приватный ключ с учетом приоритетов источников.
	Load(flagKeyFile string) (string, error)
}

// Clock порт получения текущего времени.
type Clock interface {
	// Now возвращает текущее время.
	Now() time.Time
}

// IO порт для вывода сообщений в stdout/stderr.
type IO interface {
	// Stdout возвращает поток стандартного вывода.
	Stdout() io.Writer
	// Stderr возвращает поток ошибок.
	Stderr() io.Writer
}

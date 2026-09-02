package ports

import (
	"io"

	"filippo.io/age"
	"github.com/asd2003ru/git-safe/internal/domain"
)

// GitClient инкапсулирует взаимодействие с git CLI.
type GitClient interface {
	IsInsideWorkTree() (bool, error)
	GetRootPath() (string, error)
	IsIgnored(path string) (bool, error)
	AddIgnorePattern(pattern string) error
	RemoveIgnorePattern(pattern string) error
}

// FileSystem позволяет тестировать операции с файлами.
type FileSystem interface {
	Exists(path string) (bool, error)
	Abs(path string) (string, error)
	IsAbs(path string) bool
	IsDir(path string) (bool, error)
	WalkFiles(root string) ([]string, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm uint32) error
	Remove(path string) error
	Chmod(path string, perm uint32) error
	MkdirAll(path string, perm uint32) error
	Open(path string) (io.ReadCloser, error)
}

// Hasher используется для проверки синхронизации открытых файлов.
type Hasher interface {
	SHA256File(path string) (string, error)
}

// StateStore работает с native layout .gitsafe.
type StateStore interface {
	StateDir() (string, error)
	PathsFile() (string, error)
	KeysFile() (string, error)
	LoadFileList() (domain.FileList, error)
	StoreFileList(list domain.FileList) error
	ReadKeysData() ([]byte, bool, error)
	WriteKeysData(data []byte) error
}

// KeyLoader загружает приватный ключ по приоритету: key -> keyfile -> env -> env-file.
type KeyLoader interface {
	LoadIdentity(key string, keyFile string) (age.Identity, error)
}

// CryptoService инкапсулирует age/ssh крипто-операции.
type CryptoService interface {
	Encrypt(plain []byte, recipients []age.Recipient) ([]byte, error)
	Decrypt(cipher []byte, identity age.Identity) ([]byte, error)
	RecipientsFromKeyList(keys []domain.Key, access domain.KeyAccess) ([]age.Recipient, error)
	ParseAndNormalizePublicKey(key string) (keyType domain.KeyType, normalized string, sshComment string, err error)
	ParseAGERecipients(key string) (count int, err error)
	ParseAgeIdentity(data []byte) (age.Identity, error)
	GenerateAgeIdentity() (private string, public string, err error)
	EncryptWithScryptRecipient(plain []byte, passphrase []byte) ([]byte, error)
}

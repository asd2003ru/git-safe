package domain

// Основные константы native-формата и legacy-совместимости.
const (
	PrivateExtension         = ".safe"
	LegacyPrivateExtension   = ".private"
	ToolName                 = "git-safe"
	PrivateKeyVariable       = "GIT_SAFE_KEY"
	PrivateKeyFileVar        = "GIT_SAFE_KEYFILE"
	PrivateDirVariable       = "GIT_SAFE_DIR"
	LegacyPrivateKeyVar      = "GIT_PRIVATE_KEY"
	LegacyPrivateKeyFileVar  = "GIT_PRIVATE_KEYFILE"
	LegacyPrivateDirVariable = "GIT_PRIVATE_DIR"
	DefaultPrivateDirName    = ".gitsafe"
	LegacyPrivateDirName     = ".gitprivate"
	PathsFileName            = "paths.json"
	KeysFileName             = "keys.dat"
)

// KeyType описывает формат публичного ключа.
type KeyType string

const (
	SSH KeyType = "ssh"
	AGE KeyType = "age"
)

// Key хранится внутри keys.dat.
type Key struct {
	Type     KeyType `json:"Type"`
	Key      string  `json:"Key"`
	ID       string  `json:"ID"`
	ReadOnly bool    `json:"ReadOnly"`
}

// KeyList хранится в зашифрованном keys.dat.
type KeyList struct {
	Version int   `json:"Version"`
	Keys    []Key `json:"Keys"`
}

// SecureFile хранит tracked файл и hash последней скрытой версии.
type SecureFile struct {
	Path string `json:"Path"`
	Hash string `json:"Hash"`
}

// SecureDirectory хранит директорию, из которой автоматически добавляются файлы.
type SecureDirectory struct {
	Path string `json:"Path"`
}

// FileList хранится в paths.json.
type FileList struct {
	Version     int               `json:"Version"`
	Files       []SecureFile      `json:"Files"`
	Directories []SecureDirectory `json:"Directories,omitempty"`
}

type StatusCode int

const (
	NotHidden StatusCode = iota + 421
	HiddenInSync
	HiddenModified
	HiddenNotRevealed
	HiddenPrivateMissing
)

func (code StatusCode) String() string {
	switch code {
	case NotHidden:
		return "not hidden"
	case HiddenInSync:
		return "hidden, in sync"
	case HiddenModified:
		return "hidden, modified"
	case HiddenNotRevealed:
		return "hidden, not revealed"
	case HiddenPrivateMissing:
		return "WARNING: private file missing!"
	default:
		return "unknown"
	}
}

type KeyAccess string

const (
	ReadOnlyAccess  KeyAccess = "ro"
	ReadWriteAccess KeyAccess = "rw"
)

// FileStatus используется командой status/reveal/clean.
type FileStatus struct {
	File   SecureFile
	Status StatusCode
}

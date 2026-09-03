package domain

// Core constants for the native format and legacy compatibility.
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

// KeyType describes the public key format.
type KeyType string

const (
	SSH KeyType = "ssh"
	AGE KeyType = "age"
)

// Key is stored inside keys.dat.
type Key struct {
	Type     KeyType `json:"Type"`
	Key      string  `json:"Key"`
	ID       string  `json:"ID"`
	ReadOnly bool    `json:"ReadOnly"`
}

// KeyList is stored in encrypted keys.dat.
type KeyList struct {
	Version int   `json:"Version"`
	Keys    []Key `json:"Keys"`
}

// SecureFile stores a tracked file and the hash of its last hidden version.
type SecureFile struct {
	Path string `json:"Path"`
	Hash string `json:"Hash"`
}

// SecureDirectory stores a directory whose files are added automatically.
type SecureDirectory struct {
	Path string `json:"Path"`
}

// FileList is stored in paths.json.
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
	HiddenMissing
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
	case HiddenMissing:
		return "ERROR: source and private files missing!"
	default:
		return "unknown"
	}
}

type KeyAccess string

const (
	ReadOnlyAccess  KeyAccess = "ro"
	ReadWriteAccess KeyAccess = "rw"
)

// FileStatus is used by the status, reveal, and clean commands.
type FileStatus struct {
	File   SecureFile
	Status StatusCode
}

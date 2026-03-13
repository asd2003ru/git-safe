package domain

// KeyAccess определяет уровень доступа ключа к операциям git-safe.
type KeyAccess string

const (
	// KeyAccessReadWrite разрешает операции чтения и изменения ключевого списка.
	KeyAccessReadWrite KeyAccess = "rw"
	// KeyAccessReadOnly разрешает только операции чтения/расшифровки.
	KeyAccessReadOnly  KeyAccess = "ro"
)

// KeyType определяет формат ключа: SSH или AGE.
type KeyType string

const (
	// KeyTypeSSH ключ в формате SSH public key.
	KeyTypeSSH KeyType = "ssh"
	// KeyTypeAGE ключ в формате age recipient.
	KeyTypeAGE KeyType = "age"
)

// Key описывает публичный ключ, сохраненный в состоянии репозитория.
type Key struct {
	Type     KeyType   `json:"type"`
	Key      string    `json:"key"`
	ID       string    `json:"id"`
	Access   KeyAccess `json:"access"`
	ReadOnly bool      `json:"readonly"`
}

// SecretFile описывает файл, управляемый git-safe, и контрольный хеш последней скрытой версии.
type SecretFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// Status отражает текущее состояние файла относительно .safe версии.
type Status string

const (
	// StatusNotHidden файл добавлен в состояние, но еще не скрыт в .safe.
	StatusNotHidden         Status = "not hidden"
	// StatusHiddenInSync .safe версия существует и соответствует раскрытому файлу.
	StatusHiddenInSync      Status = "hidden, in sync"
	// StatusHiddenModified раскрытый файл изменен относительно последней скрытой версии.
	StatusHiddenModified    Status = "hidden, modified"
	// StatusHiddenNotRevealed .safe файл есть, а исходный файл отсутствует.
	StatusHiddenNotRevealed Status = "hidden, not revealed"
	// StatusHiddenSafeMiss исходный файл помечен как скрытый, но .safe файл отсутствует.
	StatusHiddenSafeMiss    Status = "WARNING: safe file missing!"
)

package usecase

import "git-safe/internal/ucerr"

// ErrorKind алиас типов ошибок use-case слоя.
type ErrorKind string

const (
	// ErrInvalidInput неверные входные данные для use-case операции.
	ErrInvalidInput   ErrorKind = ErrorKind(ucerr.InvalidInput)
	// ErrNotInRepo операция вызвана вне git-репозитория.
	ErrNotInRepo      ErrorKind = ErrorKind(ucerr.NotInRepo)
	// ErrNotInitialized состояние git-safe не инициализировано.
	ErrNotInitialized ErrorKind = ErrorKind(ucerr.NotInitialized)
	// ErrAlreadyInit состояние уже инициализировано.
	ErrAlreadyInit    ErrorKind = ErrorKind(ucerr.AlreadyInit)
	// ErrForbidden отказано в доступе (например, неверный ключ).
	ErrForbidden      ErrorKind = ErrorKind(ucerr.Forbidden)
	// ErrNotFound искомый ресурс не найден.
	ErrNotFound       ErrorKind = ErrorKind(ucerr.NotFound)
	// ErrConflict конфликт состояния данных.
	ErrConflict       ErrorKind = ErrorKind(ucerr.Conflict)
	// ErrInternal внутренняя ошибка выполнения.
	ErrInternal       ErrorKind = ErrorKind(ucerr.Internal)
)

// Error типизированная ошибка use-case слоя.
type Error = ucerr.Error

// wrapErr унифицирует упаковку ошибок use-case в типизированный формат.
func wrapErr(op string, kind ErrorKind, err error) error {
	return ucerr.Wrap(op, ucerr.Kind(kind), err)
}

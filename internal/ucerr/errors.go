package ucerr

import (
	"errors"
	"fmt"
)

// Kind типизирует категорию ошибки use-case слоя.
type Kind string

const (
	// InvalidInput ошибка входных данных или параметров команды.
	InvalidInput Kind = "invalid_input"
	// NotInRepo команда выполнена вне git-репозитория.
	NotInRepo Kind = "not_in_repo"
	// NotInitialized состояние git-safe еще не создано.
	NotInitialized Kind = "not_initialized"
	// AlreadyInit состояние уже было инициализировано.
	AlreadyInit Kind = "already_initialized"
	// Forbidden ошибка прав доступа или невозможность расшифровки.
	Forbidden Kind = "forbidden"
	// NotFound отсутствие требуемого файла/ключа/ресурса.
	NotFound Kind = "not_found"
	// Conflict конфликт состояния данных.
	Conflict Kind = "conflict"
	// Internal внутренняя ошибка выполнения.
	Internal Kind = "internal"
)

// Error типизированная ошибка use-case уровня.
type Error struct {
	Op   string
	Kind Kind
	Err  error
}

// Error возвращает строковое представление типизированной ошибки.
func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Op, e.Kind)
	}
	return fmt.Sprintf("%s: %s: %v", e.Op, e.Kind, e.Err)
}

// Unwrap возвращает вложенную исходную ошибку.
func (e *Error) Unwrap() error { return e.Err }

// Wrap оборачивает исходную ошибку в типизированную ошибку use-case уровня.
func Wrap(op string, kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Kind: kind, Err: err}
}

// KindOf извлекает тип ошибки, если она обернута через Error.
func KindOf(err error) (Kind, bool) {
	var ue *Error
	if errors.As(err, &ue) {
		return ue.Kind, true
	}
	return "", false
}

// Message возвращает человекочитаемое сообщение корневой ошибки без технической оболочки.
func Message(err error) string {
	if err == nil {
		return ""
	}

	// Возвращаем первопричину ошибки, а не внутренние поля op/kind.
	cur := err
	for {
		next := errors.Unwrap(cur)
		if next == nil {
			return cur.Error()
		}
		cur = next
	}
}

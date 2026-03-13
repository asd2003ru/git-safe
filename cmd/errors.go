package cmd

import (
	"strings"

	"git-safe/internal/ucerr"
)

const (
	// ExitOK успешное завершение.
	ExitOK = 0
	// ExitGeneric общий код ошибки по умолчанию.
	ExitGeneric = 1
	// ExitUsage ошибка параметров/формы запуска.
	ExitUsage = 2
	// ExitNotRepo запуск вне git-репозитория.
	ExitNotRepo = 3
	// ExitState ошибка состояния (не инициализировано / уже инициализировано).
	ExitState = 4
	// ExitConflict конфликт данных/состояния.
	ExitConflict = 5
	// ExitAccess ошибка прав доступа/аутентификации.
	ExitAccess = 6
	// ExitNotFound не найден требуемый ресурс.
	ExitNotFound = 7
)

// ExitCode маппит типизированную ошибку в стабильный exit-code CLI.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}

	kind, ok := ucerr.KindOf(err)
	if !ok {
		if isUsageError(err.Error()) {
			return ExitUsage
		}
		return ExitGeneric
	}

	switch kind {
	case ucerr.InvalidInput:
		return ExitUsage
	case ucerr.NotInRepo:
		return ExitNotRepo
	case ucerr.NotInitialized, ucerr.AlreadyInit:
		return ExitState
	case ucerr.Conflict:
		return ExitConflict
	case ucerr.Forbidden:
		return ExitAccess
	case ucerr.NotFound:
		return ExitNotFound
	default:
		return ExitGeneric
	}
}

// ErrorMessage возвращает пользовательское сообщение об ошибке.
func ErrorMessage(err error) string {
	return ucerr.Message(err)
}

// isUsageError эвристически определяет ошибки неверного синтаксиса CLI.
func isUsageError(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "unknown command") ||
		strings.Contains(m, "unknown flag") ||
		strings.Contains(m, "requires at least") ||
		strings.Contains(m, "accepts at most") ||
		strings.Contains(m, "required flag")
}

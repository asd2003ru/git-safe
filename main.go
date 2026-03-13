package main

import (
	"fmt"
	"os"

	"git-safe/cmd"
	"git-safe/internal/usecase"
)

// main точка входа CLI приложения.
func main() {
	// Этап 1: создаем use-case сервис со всеми адаптерами по умолчанию.
	uc, err := usecase.NewDefaultService()
	if err != nil {
		fmt.Fprintln(os.Stderr, cmd.ErrorMessage(err))
		os.Exit(cmd.ExitCode(err))
	}

	// Этап 2: запускаем CLI и завершаем процесс с нормализованным кодом ошибки.
	if err := cmd.Execute(uc); err != nil {
		fmt.Fprintln(os.Stderr, cmd.ErrorMessage(err))
		os.Exit(cmd.ExitCode(err))
	}
}

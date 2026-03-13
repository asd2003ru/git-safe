package cmd

import (
	"fmt"

	"git-safe/internal/ucerr"
	"github.com/spf13/cobra"
)

// Execute создает обработчики команд и запускает корневую команду Cobra.
func Execute(uc UseCases) error {
	handlers, err := NewCommandHandlers(uc)
	if err != nil {
		return err
	}
	return newRootCmd(handlers).Execute()
}

// newRootCmd формирует корневое дерево CLI-команд git-safe.
func newRootCmd(h *CommandHandlers) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "git-safe",
		Short:         "Git Safe CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		newInitCmd(h),
		newAddCmd(h),
		newRemoveCmd(h),
		newHideCmd(h),
		newRevealCmd(h),
		newCleanCmd(h),
		newStatusCmd(h),
		newKeysCmd(h),
	)

	return cmd
}

// argError создает типизированную ошибку пользовательского ввода.
func argError(format string, args ...any) error {
	return ucerr.Wrap("cli", ucerr.InvalidInput, fmt.Errorf(format, args...))
}

package cmd

import "github.com/spf13/cobra"

// newRemoveCmd создает команду удаления файлов из отслеживания git-safe.
func newRemoveCmd(h *CommandHandlers) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <FILE...>",
		Short: "Remove files from private tracking",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Remove(cmd.Context(), args)
		},
	}
}

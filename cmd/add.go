package cmd

import "github.com/spf13/cobra"

// newAddCmd создает команду добавления файлов в отслеживание git-safe.
func newAddCmd(h *CommandHandlers) *cobra.Command {
	return &cobra.Command{
		Use:   "add <FILE...>",
		Short: "Add files to private tracking",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Add(cmd.Context(), args)
		},
	}
}

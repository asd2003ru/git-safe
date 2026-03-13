package cmd

import "github.com/spf13/cobra"

// newStatusCmd создает команду просмотра статуса отслеживаемых файлов.
func newStatusCmd(h *CommandHandlers) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of tracked private files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return h.Status(cmd.Context())
		},
	}
}

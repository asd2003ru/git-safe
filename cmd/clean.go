package cmd

import "github.com/spf13/cobra"

// newCleanCmd создает команду очистки раскрытых исходных файлов.
func newCleanCmd(h *CommandHandlers) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove revealed source files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return h.Clean(cmd.Context(), CleanInput{Force: force})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force removal of out-of-sync files")

	return cmd
}

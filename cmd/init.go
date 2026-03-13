package cmd

import "github.com/spf13/cobra"

// newInitCmd создает команду инициализации состояния git-safe в репозитории.
func newInitCmd(h *CommandHandlers) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize git-safe state in repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return h.Init(cmd.Context())
		},
	}
}

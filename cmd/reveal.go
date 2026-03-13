package cmd

import "github.com/spf13/cobra"

// newRevealCmd создает команду расшифровки файлов из формата .safe.
func newRevealCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string
	var force bool
	var clean bool

	cmd := &cobra.Command{
		Use:   "reveal [FILE...]",
		Short: "Decrypt tracked files or selected files",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Reveal(cmd.Context(), RevealInput{
				KeyFile: keyFile,
				Force:   force,
				Clean:   clean,
				Files:   args,
			})
		},
	}

	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from file")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove *.safe files after reveal")

	return cmd
}

package cmd

import "github.com/spf13/cobra"

// newHideCmd создает команду шифрования файлов в формат .safe.
func newHideCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string
	var clean bool

	cmd := &cobra.Command{
		Use:   "hide [FILE...]",
		Short: "Encrypt tracked files or selected files",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return h.Hide(cmd.Context(), HideInput{
				KeyFile: keyFile,
				Clean:   clean,
				Files:   args,
			})
		},
	}

	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from file")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove source files after encryption")

	return cmd
}

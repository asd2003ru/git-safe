package cmd

import "github.com/spf13/cobra"

// newKeysCmd создает корневую группу подкоманд для управления ключами.
func newKeysCmd(h *CommandHandlers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage recipients and key material",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		newKeysListCmd(h),
		newKeysAddCmd(h),
		newKeysRemoveCmd(h),
		newKeysGenerateCmd(h),
	)

	return cmd
}

// newKeysListCmd создает подкоманду вывода списка зарегистрированных ключей.
func newKeysListCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured keys",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return h.KeysList(cmd.Context(), KeysListInput{KeyFile: keyFile})
		},
	}
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from file")
	return cmd
}

// newKeysAddCmd создает подкоманду добавления публичного ключа.
func newKeysAddCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string
	var id string
	var pubFile string
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "add [PUBLIC_KEY]",
		Short: "Add key by --pubfile or positional public key",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if pubFile == "" && len(args) == 0 {
				return argError("no public key specified: use --pubfile FILE or provide PUBLIC_KEY")
			}
			publicKey := ""
			if len(args) > 0 {
				publicKey = args[0]
			}
			return h.KeysAdd(cmd.Context(), KeysAddInput{
				KeyFile:   keyFile,
				ID:        id,
				PubFile:   pubFile,
				ReadOnly:  readOnly,
				PublicKey: publicKey,
			})
		},
	}

	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from file")
	cmd.Flags().StringVar(&id, "id", "", "Key identity")
	cmd.Flags().StringVar(&pubFile, "pubfile", "", "Load public key from file")
	cmd.Flags().BoolVar(&readOnly, "readonly", false, "Added key is read-only")

	return cmd
}

// newKeysRemoveCmd создает подкоманду удаления ключа по идентификатору.
func newKeysRemoveCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string
	var id string

	cmd := &cobra.Command{
		Use:   "remove [ID]",
		Short: "Remove key by --id or positional ID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetID := id
			if targetID == "" && len(args) > 0 {
				targetID = args[0]
			}
			if targetID == "" {
				return argError("specify key id using --id or positional ID")
			}
			return h.KeysRemove(cmd.Context(), KeysRemoveInput{
				KeyFile: keyFile,
				ID:      targetID,
			})
		},
	}

	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from file")
	cmd.Flags().StringVar(&id, "id", "", "Key identity to remove")

	return cmd
}

// newKeysGenerateCmd создает подкоманду генерации пары AGE-ключей.
func newKeysGenerateCmd(h *CommandHandlers) *cobra.Command {
	var keyFile string
	var pubFile string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate AGE keypair",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if keyFile == "" {
				return argError("--keyfile is required")
			}
			return h.KeysGenerate(cmd.Context(), KeysGenerateInput{
				KeyFile: keyFile,
				PubFile: pubFile,
			})
		},
	}

	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Target private key file")
	cmd.Flags().StringVar(&pubFile, "pubfile", "", "Target public key file")

	return cmd
}

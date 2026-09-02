package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// BuildRootCommand builds the CLI command tree.
func BuildRootCommand(service *usecase.Service) *cobra.Command {
	root := &cobra.Command{
		Use:   domain.ToolName,
		Short: "Encrypt private files in git repository",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd.Help()
			return fmt.Errorf("no command specified")
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}
			return service.CheckSetup()
		},
		SilenceUsage: true,
	}

	root.AddCommand(initCmd(service))
	root.AddCommand(addCmd(service))
	root.AddCommand(removeCmd(service))
	root.AddCommand(hideCmd(service))
	root.AddCommand(revealCmd(service))
	root.AddCommand(cleanCmd(service))
	root.AddCommand(statusCmd(service))
	root.AddCommand(hookCmd(service))
	root.AddCommand(migrateCmd(service))
	root.AddCommand(generateCmd(service))
	root.AddCommand(keysCmd(service))
	root.AddCommand(completionCmd(service))

	return root
}

func initCmd(service *usecase.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize git-safe state",
		RunE: func(_ *cobra.Command, _ []string) error {
			return service.Init()
		},
	}
}

func addCmd(service *usecase.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "add <FILE_OR_DIR...>",
		Short: "Add files or directories to private tracking",
		RunE: func(_ *cobra.Command, args []string) error {
			return service.Add(args)
		},
	}
}

func removeCmd(service *usecase.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <FILE...>",
		Short: "Remove files from private tracking",
		RunE: func(_ *cobra.Command, args []string) error {
			return service.Remove(args)
		},
	}
}

func hideCmd(service *usecase.Service) *cobra.Command {
	var key string
	var keyFile string
	var clean bool

	cmd := &cobra.Command{
		Use:   "hide [--key KEY] [--keyfile FILE] [--clean] [FILE...]",
		Short: "Encrypt tracked files",
		RunE: func(_ *cobra.Command, args []string) error {
			return service.Hide(usecase.HideOptions{Key: key, KeyFile: keyFile, Clean: clean, Files: args})
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Load private key from `value`")
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from `file`")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove source files after encryption")
	return cmd
}

func revealCmd(service *usecase.Service) *cobra.Command {
	var key string
	var keyFile string
	var force bool
	var clean bool

	cmd := &cobra.Command{
		Use:   "reveal [--key KEY] [--keyfile FILE] [--force] [--clean] [FILE...]",
		Short: "Decrypt tracked files",
		RunE: func(_ *cobra.Command, args []string) error {
			result, err := service.Reveal(usecase.RevealOptions{Key: key, KeyFile: keyFile, Force: force, Clean: clean, Files: args})
			if err != nil {
				return err
			}
			if result.InSync > 0 {
				fmt.Printf("%v file%s already in sync, ", result.InSync, pluralSuffix(result.InSync))
			}
			fmt.Printf("%v file%s revealed\n", result.Revealed, pluralSuffix(result.Revealed))
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Load private key from `value`")
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from `file`")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing target files")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove private files after revealing")
	return cmd
}

func cleanCmd(service *usecase.Service) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clean [--force]",
		Short: "Clean revealed files",
		RunE: func(_ *cobra.Command, _ []string) error {
			return service.Clean(force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force removal of out of sync files")
	return cmd
}

func statusCmd(service *usecase.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show file sync status",
		RunE: func(_ *cobra.Command, _ []string) error {
			statuses, err := service.Status()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			if len(statuses) == 0 {
				fmt.Fprintln(w, "No private files")
			}
			for _, item := range statuses {
				fmt.Fprintf(w, "%s\t[%s]\n", item.File.Path, item.Status)
			}
			return w.Flush()
		},
	}
}

func hookCmd(service *usecase.Service) *cobra.Command {
	var key string
	var keyFile string

	cmd := &cobra.Command{
		Use:   "hook [--key KEY] [--keyfile FILE]",
		Short: "Install git hooks for transparent hide/reveal",
		RunE: func(_ *cobra.Command, _ []string) error {
			return service.Hook(usecase.HookOptions{Key: key, KeyFile: keyFile})
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Use private key `value` in generated hooks")
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Use private key from `file` in generated hooks")
	return cmd
}

func migrateCmd(service *usecase.Service) *cobra.Command {
	var dryRun bool
	var force bool
	var keepLegacy bool

	cmd := &cobra.Command{
		Use:   "migrate [--dry-run] [--force] [--keep-legacy]",
		Short: "Migrate legacy git-private layout to git-safe layout",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := service.Migrate(usecase.MigrateOptions{DryRun: dryRun, Force: force, KeepLegacy: keepLegacy})
			if err != nil {
				return err
			}
			mode := "migrated"
			if dryRun {
				mode = "would migrate"
			}
			fmt.Printf("%s %d state file%s and %d encrypted file%s\n",
				mode,
				result.StateFilesCopied,
				pluralSuffix(result.StateFilesCopied),
				result.EncryptedCopied,
				pluralSuffix(result.EncryptedCopied),
			)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show migration plan without writing files")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing git-safe files")
	cmd.Flags().BoolVar(&keepLegacy, "keep-legacy", false, "Keep legacy git-private files after migration")
	return cmd
}

func generateCmd(service *usecase.Service) *cobra.Command {
	cmd := keysGenerateCmd(service)
	cmd.Use = "generate --keyfile FILE [--pubfile FILE]"
	cmd.Short = "Generate AGE keypair (alias for 'keys generate')"
	return cmd
}

func keysCmd(service *usecase.Service) *cobra.Command {
	keys := &cobra.Command{
		Use:   "keys",
		Short: "Manage key list",
	}

	keys.AddCommand(keysListCmd(service))
	keys.AddCommand(keysAddCmd(service))
	keys.AddCommand(keysRemoveCmd(service))
	keys.AddCommand(keysGenerateCmd(service))

	return keys
}

func keysListCmd(service *usecase.Service) *cobra.Command {
	var keyFile string

	cmd := &cobra.Command{
		Use:   "list [--keyfile FILE]",
		Short: "List configured keys",
		RunE: func(_ *cobra.Command, _ []string) error {
			keys, err := service.KeysList(keyFile)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			for _, key := range keys {
				mode := "rw"
				if key.ReadOnly {
					mode = "ro"
				}
				if len(key.Key) < 12 {
					fmt.Fprintf(w, "%s\t(%s/%s)\t[%s]\n", key.ID, key.Type, mode, key.Key)
					continue
				}
				fmt.Fprintf(w, "%s\t(%s/%s)\t[...%s]\n", key.ID, key.Type, mode, key.Key[len(key.Key)-12:])
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from `file`")
	return cmd
}

func keysAddCmd(service *usecase.Service) *cobra.Command {
	var keyFile string
	var pubFile string
	var id string
	var readOnly bool

	cmd := &cobra.Command{
		Use:   "add [--keyfile FILE] [--id ID] [--readonly] <--pubfile FILE | \"PUBLIC_KEY\">",
		Short: "Add public key",
		RunE: func(_ *cobra.Command, args []string) error {
			keyData := ""
			if pubFile == "" {
				if len(args) == 0 {
					return fmt.Errorf("no public key specified")
				}
				keyData = strings.Join(args, " ")
			}
			return service.KeysAdd(usecase.KeysAddOptions{KeyFile: keyFile, PubFile: pubFile, ID: id, ReadOnly: readOnly, KeyData: keyData})
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Key `identity` to add")
	cmd.Flags().StringVar(&pubFile, "pubfile", "", "Load public key from `file`")
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from `file`")
	cmd.Flags().BoolVar(&readOnly, "readonly", false, "Added key can only be used to reveal files")
	return cmd
}

func keysRemoveCmd(service *usecase.Service) *cobra.Command {
	var keyFile string
	var id string

	cmd := &cobra.Command{
		Use:   "remove [--keyfile FILE] <--id ID | ID>",
		Short: "Remove key by id",
		RunE: func(_ *cobra.Command, args []string) error {
			resolvedID := strings.TrimSpace(id)
			if resolvedID == "" && len(args) > 0 {
				resolvedID = strings.TrimSpace(args[0])
			}
			return service.KeysRemove(keyFile, resolvedID)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "Key `identity` to remove")
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Load private key from `file`")
	return cmd
}

func keysGenerateCmd(service *usecase.Service) *cobra.Command {
	var keyFile string
	var pubFile string

	cmd := &cobra.Command{
		Use:   "generate --keyfile FILE [--pubfile FILE]",
		Short: "Generate AGE keypair",
		RunE: func(_ *cobra.Command, _ []string) error {
			passphrase, err := readPassphrase("Enter passphrase:")
			if err != nil {
				return err
			}
			if len(passphrase) != 0 {
				confirmed, err := readPassphrase("Confirm passphrase:")
				if err != nil {
					return err
				}
				if !bytes.Equal(passphrase, confirmed) {
					return fmt.Errorf("passphrases do not match")
				}
			}
			return service.KeysGenerate(usecase.GenerateOptions{KeyFile: keyFile, PubFile: pubFile, Passphrase: passphrase})
		},
	}
	cmd.Flags().StringVar(&keyFile, "keyfile", "", "Store private key in `file`")
	cmd.Flags().StringVar(&pubFile, "pubfile", "", "Store public key in `file`")
	_ = cmd.MarkFlagRequired("keyfile")
	return cmd
}

// completionCmd creates the shell autocompletion command.
func completionCmd(service *usecase.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate the autocompletion script for your shell",
		Long: `Generate the autocompletion script for the specified shell.
To load completions:

  Bash:
    $ source <(git-safe completion bash)
  Zsh:
    $ git-safe completion zsh > "${fpath[1]}"/_git-safe
  Fish:
    $ git-safe completion fish | source
  PowerShell:
    PS> git-safe completion powershell | Out-String | Invoke-Expression`,
		Args: cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell type %q", args[0])
			}
		},
	}
	return cmd
}

func pluralSuffix(number int) string {
	if number == 1 {
		return ""
	}
	return "s"
}

func Execute(service *usecase.Service, args []string) error {
	root := BuildRootCommand(service)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		return err
	}
	return nil
}

func readPassphrase(prompt string) ([]byte, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	stdinFD := os.Stdin.Fd()
	state, _ := term.GetState(int(stdinFD))
	done := make(chan struct{})

	go func() {
		select {
		case sig := <-signals:
			if sig != nil && state != nil {
				_ = term.Restore(int(stdinFD), state)
				os.Exit(1)
			}
		case <-done:
		}
	}()

	defer func() {
		close(done)
		signal.Stop(signals)
		signal.Reset(syscall.SIGINT)
	}()

	fmt.Print(prompt)
	passphrase, err := term.ReadPassword(int(stdinFD))
	fmt.Println()
	if err != nil && !errors.Is(err, syscall.EINTR) {
		return nil, err
	}
	return passphrase, nil
}

package main

import (
	"fmt"
	"os"

	"github.com/asd2003ru/git-safe/cmd"
	"github.com/asd2003ru/git-safe/internal/adapters/cryptoage"
	"github.com/asd2003ru/git-safe/internal/adapters/gitgogit"
	"github.com/asd2003ru/git-safe/internal/adapters/keyloader"
	"github.com/asd2003ru/git-safe/internal/adapters/osfs"
	"github.com/asd2003ru/git-safe/internal/adapters/sha256hash"
	"github.com/asd2003ru/git-safe/internal/adapters/statefs"
	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/usecase"
)

func main() {
	git := gitgogit.New()
	fs := osfs.New()
	hasher := sha256hash.New()
	crypto := cryptoage.New()
	loader := keyloader.New(fs, crypto)
	state := statefs.New(git)
	service := usecase.NewService(git, fs, hasher, state, loader, crypto, os.Stdin, os.Stdout, os.Stderr)

	if err := cmd.Execute(service, os.Args[1:]); err != nil {
		fmt.Printf("%s error: %v\n", domain.ToolName, err)
		os.Exit(1)
	}
}

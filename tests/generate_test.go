package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/asd2003ru/git-safe/internal/usecase"
)

func TestGenerateWithoutPassphraseWritesPublicFile(t *testing.T) {
	svc := setupAndInit(t)

	if err := svc.KeysGenerate(usecase.GenerateOptions{
		KeyFile:    "plain.age",
		PubFile:    "plain.age.pub",
		Passphrase: nil,
	}); err != nil {
		t.Fatal(err)
	}

	privateData, err := os.ReadFile("plain.age")
	if err != nil {
		t.Fatal(err)
	}
	publicData, err := os.ReadFile("plain.age.pub")
	if err != nil {
		t.Fatal(err)
	}

	privateText := string(privateData)
	if !strings.Contains(privateText, "AGE-SECRET-KEY-") {
		t.Fatalf("unexpected private key contents: %s", privateText)
	}

	publicText := string(publicData)
	if !strings.HasPrefix(strings.TrimSpace(publicText), "age1") {
		t.Fatalf("unexpected public key contents: %s", publicText)
	}
	if strings.Contains(publicText, "Public key:") {
		t.Fatalf("public key file must contain raw key only: %s", publicText)
	}
}

package cryptoage

import (
	"os"
	"path/filepath"
	"testing"

	"git-safe/internal/domain"
)

func TestAdapterEncryptDecryptAGE(t *testing.T) {
	t.Parallel()

	a := &Adapter{}
	priv, pub, err := a.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}

	cipher, err := a.Encrypt([]byte("payload"), []domain.Key{{Type: domain.KeyTypeAGE, Key: pub, ID: "age"}})
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	plain, err := a.Decrypt(cipher, priv)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != "payload" {
		t.Fatalf("unexpected plaintext: %q", string(plain))
	}
}

func TestParseIdentitiesSSH(t *testing.T) {
	t.Parallel()

	keyPath := filepath.Join("..", "..", "..", "git-private", "tests", "testkeys", "one.key")
	priv, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read ssh private key fixture: %v", err)
	}

	_, err = parseIdentities(string(priv))
	if err != nil {
		t.Fatalf("parse ssh identity: %v", err)
	}
}

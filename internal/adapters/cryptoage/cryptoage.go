package cryptoage

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/asd2003ru/git-safe/internal/domain"
	"golang.org/x/crypto/ssh"
)

// Adapter implements CryptoService using age/agessh.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Encrypt(plain []byte, recipients []age.Recipient) ([]byte, error) {
	var out bytes.Buffer
	writer, err := age.Encrypt(&out, recipients...)
	if err != nil {
		return nil, err
	}
	if _, err = writer.Write(plain); err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (a *Adapter) Decrypt(cipher []byte, identity age.Identity) ([]byte, error) {
	reader := bytes.NewReader(cipher)
	plainReader, err := age.Decrypt(reader, identity)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(plainReader)
}

func (a *Adapter) RecipientsFromKeyList(keys []domain.Key, access domain.KeyAccess) ([]age.Recipient, error) {
	result := make([]age.Recipient, 0, len(keys))
	for _, key := range keys {
		if access == domain.ReadWriteAccess && key.ReadOnly {
			continue
		}

		switch key.Type {
		case domain.AGE:
			recipients, err := age.ParseRecipients(strings.NewReader(key.Key))
			if err != nil {
				return nil, err
			}
			if len(recipients) != 1 {
				return nil, fmt.Errorf("unexpected key contents")
			}
			result = append(result, recipients[0])
		case domain.SSH:
			recipient, err := agessh.ParseRecipient(key.Key)
			if err != nil {
				return nil, err
			}
			result = append(result, recipient)
		default:
			return nil, fmt.Errorf("unexpected key type %q", key.Type)
		}
	}
	return result, nil
}

func (a *Adapter) ParseAndNormalizePublicKey(key string) (domain.KeyType, string, string, error) {
	sshKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err == nil {
		normalized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshKey)))
		return domain.SSH, normalized, strings.TrimSpace(comment), nil
	}

	recipients, err := age.ParseRecipients(strings.NewReader(key))
	if err != nil {
		return "", "", "", fmt.Errorf("invalid key format")
	}
	if len(recipients) != 1 {
		if len(recipients) == 0 {
			return "", "", "", fmt.Errorf("invalid key format")
		}
		return "", "", "", fmt.Errorf("multiple keys found, add one key at a time")
	}

	return domain.AGE, strings.TrimSpace(key), "", nil
}

func (a *Adapter) ParseAGERecipients(key string) (int, error) {
	recipients, err := age.ParseRecipients(strings.NewReader(key))
	if err != nil {
		return 0, err
	}
	return len(recipients), nil
}

func (a *Adapter) ParseAgeIdentity(data []byte) (age.Identity, error) {
	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil || len(identities) != 1 {
		return nil, fmt.Errorf("invalid key or passphrase")
	}
	return identities[0], nil
}

func (a *Adapter) GenerateAgeIdentity() (string, string, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	return identity.String(), identity.Recipient().String(), nil
}

func (a *Adapter) EncryptWithScryptRecipient(plain []byte, passphrase []byte) ([]byte, error) {
	recipient, err := age.NewScryptRecipient(string(passphrase))
	if err != nil {
		return nil, err
	}
	return a.Encrypt(plain, []age.Recipient{recipient})
}

// ParseSSHIdentity tries to parse an SSH identity with passphrase support.
func ParseSSHIdentity(key []byte, passphrase []byte) (age.Identity, bool, error) {
	identity, err := agessh.ParseIdentity(key)
	if err == nil {
		return identity, false, nil
	}

	if _, needsPass := err.(*ssh.PassphraseMissingError); !needsPass {
		return nil, false, err
	}

	parsed, parseErr := ssh.ParseRawPrivateKeyWithPassphrase(key, passphrase)
	if parseErr != nil {
		return nil, true, fmt.Errorf("failed to load key, wrong passphrase?")
	}

	switch k := parsed.(type) {
	case *ed25519.PrivateKey:
		identity, err = agessh.NewEd25519Identity(*k)
	case *rsa.PrivateKey:
		identity, err = agessh.NewRSAIdentity(k)
	default:
		return nil, true, fmt.Errorf("unsupported SSH key type: %T", k)
	}
	if err != nil {
		return nil, true, err
	}

	return identity, true, nil
}

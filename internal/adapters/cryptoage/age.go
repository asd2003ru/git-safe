package cryptoage

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"git-safe/internal/domain"
	"git-safe/internal/ports"
)

// Adapter реализация порта Crypto на базе библиотеки age.
type Adapter struct{}

// New создает crypto-адаптер на базе библиотеки age.
func New() ports.Crypto {
	return &Adapter{}
}

// Encrypt шифрует открытые данные на все ключи-получатели.
func (a *Adapter) Encrypt(plaintext []byte, keys []domain.Key) ([]byte, error) {
	recipients, err := parseRecipients(keys)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, fmt.Errorf("no keys added, cannot encrypt")
	}

	var out bytes.Buffer
	w, err := age.Encrypt(&out, recipients...)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Decrypt расшифровывает ciphertext с использованием переданной identity.
func (a *Adapter) Decrypt(ciphertext []byte, identity string) ([]byte, error) {
	ids, err := parseIdentities(identity)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(ciphertext), ids...)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

// GenerateKeyPair генерирует AGE private/public ключи формата X25519.
func (a *Adapter) GenerateKeyPair() (string, string, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	return id.String(), id.Recipient().String(), nil
}

// parseRecipients преобразует доменные ключи в age recipients.
func parseRecipients(keys []domain.Key) ([]age.Recipient, error) {
	out := make([]age.Recipient, 0, len(keys))
	for _, key := range keys {
		switch key.Type {
		case domain.KeyTypeAGE:
			r, err := age.ParseX25519Recipient(strings.TrimSpace(key.Key))
			if err != nil {
				return nil, fmt.Errorf("invalid age recipient for key %q: %w", key.ID, err)
			}
			out = append(out, r)
		case domain.KeyTypeSSH:
			r, err := agessh.ParseRecipient(strings.TrimSpace(key.Key))
			if err != nil {
				return nil, fmt.Errorf("invalid ssh recipient for key %q: %w", key.ID, err)
			}
			out = append(out, r)
		default:
			return nil, fmt.Errorf("unsupported key type %q", key.Type)
		}
	}
	return out, nil
}

// parseIdentities разбирает identity как AGE или SSH приватный ключ.
func parseIdentities(identity string) ([]age.Identity, error) {
	raw := strings.TrimSpace(identity)
	if raw == "" {
		return nil, fmt.Errorf("empty identity")
	}

	if ids, err := age.ParseIdentities(strings.NewReader(raw)); err == nil && len(ids) > 0 {
		return ids, nil
	}

	sshID, err := agessh.ParseIdentity([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("unsupported private key format")
	}
	return []age.Identity{sshID}, nil
}

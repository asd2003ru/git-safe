package keyloader

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"filippo.io/age"
	"github.com/asd2003ru/git-safe/internal/adapters/cryptoage"
	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/ports"
	"golang.org/x/term"
)

// Adapter реализует загрузку приватного ключа с поддержкой legacy-переменных.
type Adapter struct {
	fs     ports.FileSystem
	crypto ports.CryptoService
}

func New(fs ports.FileSystem, crypto ports.CryptoService) *Adapter {
	return &Adapter{fs: fs, crypto: crypto}
}

func (a *Adapter) LoadIdentity(key string, keyFile string) (age.Identity, error) {
	keyData, err := a.loadKeyData(key, keyFile)
	if err != nil {
		return nil, err
	}

	if identity, _, err := cryptoage.ParseSSHIdentity(keyData, nil); err == nil {
		return identity, nil
	}

	sshNeedsPassphrase, sshParseErr := isSSHWithMissingPassphrase(keyData)
	if sshNeedsPassphrase {
		pass, err := readPassphrase("Enter SSH key passphrase:")
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase")
		}
		if identity, _, err := cryptoage.ParseSSHIdentity(keyData, pass); err == nil {
			return identity, nil
		} else {
			// Если это SSH-ключ и пароль неверный, не пытаемся трактовать его как AGE.
			return nil, err
		}
	} else if sshParseErr == nil {
		// Это валидный SSH-ключ без passphrase, но не удалось распарсить identity.
		// Возвращаем исходную SSH-ошибку и не уходим в AGE fallback.
		return nil, fmt.Errorf("failed to load SSH identity")
	}

	// Если AGE-ключ зашифрован через scrypt, сначала расшифровываем его.
	if bytes.HasPrefix(keyData, []byte("age-encryption.org/")) {
		pass, err := readPassphrase("Enter AGE key passphrase:")
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase")
		}
		scryptIdentity, err := age.NewScryptIdentity(string(pass))
		if err != nil {
			return nil, err
		}
		decrypted, err := a.crypto.Decrypt(keyData, scryptIdentity)
		if err != nil {
			return nil, err
		}
		keyData = decrypted
	}

	identity, err := a.crypto.ParseAgeIdentity(keyData)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (a *Adapter) loadKeyData(flagKey string, flagPath string) ([]byte, error) {
	if flagKey != "" {
		return []byte(flagKey), nil
	}

	if flagPath != "" {
		return readFromFileOrStdin(a.fs, flagPath)
	}

	if value := firstEnvValue(domain.PrivateKeyVariable, domain.LegacyPrivateKeyVar); value != "" {
		return []byte(value), nil
	}

	file := firstEnvValue(domain.PrivateKeyFileVar, domain.LegacyPrivateKeyFileVar)
	if file == "" {
		return nil, fmt.Errorf("no private key provided, use -key, -keyfile or environment variables %s, %s, %s or %s",
			domain.PrivateKeyVariable, domain.PrivateKeyFileVar, domain.LegacyPrivateKeyVar, domain.LegacyPrivateKeyFileVar)
	}

	data, err := a.fs.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file %q: %w", file, err)
	}
	return data, nil
}

func firstEnvValue(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func readFromFileOrStdin(fs ports.FileSystem, file string) ([]byte, error) {
	if file == "-" {
		return ioReadAll(os.Stdin)
	}
	data, err := fs.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(data), nil
}

func ioReadAll(f *os.File) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func isSSHWithMissingPassphrase(key []byte) (bool, error) {
	_, needsPass, err := cryptoage.ParseSSHIdentity(key, nil)
	return needsPass, err
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
	return passphrase, err
}

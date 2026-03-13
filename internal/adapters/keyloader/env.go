package keyloader

import (
	"fmt"
	"os"
	"strings"

	"git-safe/internal/ports"
)

const (
	envKey     = "GIT_SAFE_KEY"
	envKeyFile = "GIT_SAFE_KEYFILE"
)

// EnvKeyLoader реализация KeyLoader через флаг и переменные окружения.
type EnvKeyLoader struct{}

// NewEnvKeyLoader создает загрузчик приватного ключа из флагов и переменных окружения.
func NewEnvKeyLoader() ports.KeyLoader {
	return &EnvKeyLoader{}
}

// Load загружает ключ в приоритете: --keyfile -> GIT_SAFE_KEY -> GIT_SAFE_KEYFILE.
func (l *EnvKeyLoader) Load(flagKeyFile string) (string, error) {
	keyFile := strings.TrimSpace(flagKeyFile)
	if keyFile != "" {
		return readKeyFromFile(keyFile)
	}

	if key := strings.TrimSpace(os.Getenv(envKey)); key != "" {
		return key, nil
	}

	if envFile := strings.TrimSpace(os.Getenv(envKeyFile)); envFile != "" {
		return readKeyFromFile(envFile)
	}

	return "", fmt.Errorf("no key specified, use --keyfile, %s, or %s", envKey, envKeyFile)
}

// readKeyFromFile читает приватный ключ из файла и валидирует непустое значение.
func readKeyFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read keyfile %q: %w", path, err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("empty keyfile %q", path)
	}
	return key, nil
}

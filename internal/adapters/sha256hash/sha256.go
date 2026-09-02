package sha256hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Adapter calculates the sha256 hash of a file.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	sum := hash.Sum(nil)
	encoded := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(encoded, sum)
	return string(encoded), nil
}

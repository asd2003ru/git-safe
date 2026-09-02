package osfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Adapter реализует FileSystem поверх os.
type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (a *Adapter) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

func (a *Adapter) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

func (a *Adapter) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (a *Adapter) WalkFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (a *Adapter) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (a *Adapter) WriteFile(path string, data []byte, perm uint32) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

func (a *Adapter) Remove(path string) error {
	return os.Remove(path)
}

func (a *Adapter) Chmod(path string, perm uint32) error {
	return os.Chmod(path, os.FileMode(perm))
}

func (a *Adapter) MkdirAll(path string, perm uint32) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

func (a *Adapter) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

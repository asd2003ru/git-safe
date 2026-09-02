package statefs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/ports"
)

// Adapter хранит state-файлы в native-layout .gitsafe.
type Adapter struct {
	git ports.GitClient
}

func New(git ports.GitClient) *Adapter {
	return &Adapter{git: git}
}

func (a *Adapter) StateDir() (string, error) {
	dir := os.Getenv(domain.PrivateDirVariable)
	if dir == "" {
		dir = domain.DefaultPrivateDirName
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	root, err := a.git.GetRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, dir), nil
}

func (a *Adapter) PathsFile() (string, error) {
	dir, err := a.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, domain.PathsFileName), nil
}

func (a *Adapter) KeysFile() (string, error) {
	dir, err := a.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, domain.KeysFileName), nil
}

func (a *Adapter) LoadFileList() (domain.FileList, error) {
	path, err := a.PathsFile()
	if err != nil {
		return domain.FileList{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.FileList{}, err
	}
	var list domain.FileList
	if err = json.Unmarshal(data, &list); err != nil {
		return domain.FileList{}, err
	}
	return list, nil
}

func (a *Adapter) StoreFileList(list domain.FileList) error {
	path, err := a.PathsFile()
	if err != nil {
		return err
	}
	list.Version = 1
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (a *Adapter) ReadKeysData() ([]byte, bool, error) {
	path, err := a.KeysFile()
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func (a *Adapter) WriteKeysData(data []byte) error {
	path, err := a.KeysFile()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty keys payload")
	}
	return os.WriteFile(path, data, 0o600)
}

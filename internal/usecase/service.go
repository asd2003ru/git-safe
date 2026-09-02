package usecase

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asd2003ru/git-safe/internal/domain"
	"github.com/asd2003ru/git-safe/internal/ports"

	"filippo.io/age"
)

var errNotFound = errors.New("not found")

// Service содержит все сценарии CLI-команд.
type Service struct {
	git    ports.GitClient
	fs     ports.FileSystem
	hasher ports.Hasher
	state  ports.StateStore
	loader ports.KeyLoader
	crypto ports.CryptoService
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewService(git ports.GitClient, fs ports.FileSystem, hasher ports.Hasher, state ports.StateStore, loader ports.KeyLoader, crypto ports.CryptoService, stdin io.Reader, stdout io.Writer, stderr io.Writer) *Service {
	return &Service{
		git:    git,
		fs:     fs,
		hasher: hasher,
		state:  state,
		loader: loader,
		crypto: crypto,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

func (s *Service) CheckSetup() error {
	inTree, err := s.git.IsInsideWorkTree()
	if err != nil {
		return err
	}
	if !inTree {
		return fmt.Errorf("not in dir with git repo. Use 'git init' or 'git clone', then in repo use 'git %s init'", domain.ToolName)
	}

	stateDir, err := s.state.StateDir()
	if err != nil {
		return err
	}
	ignored, err := s.git.IsIgnored(stateDir)
	if err != nil {
		return err
	}
	if ignored {
		return fmt.Errorf("%q is in .gitignore", stateDir)
	}

	return nil
}

func (s *Service) Init() error {
	stateDir, err := s.state.StateDir()
	if err != nil {
		return err
	}
	exists, err := s.fs.Exists(stateDir)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("already initialized")
	}

	root, err := s.git.GetRootPath()
	if err != nil {
		return err
	}
	legacyExists, err := s.fs.Exists(legacyStateDir(root))
	if err != nil {
		return err
	}
	if legacyExists {
		return fmt.Errorf("legacy state found, run '%s migrate'", domain.ToolName)
	}

	if err = s.fs.MkdirAll(stateDir, 0o770); err != nil {
		return err
	}
	if err = s.state.StoreFileList(domain.FileList{}); err != nil {
		return err
	}
	return s.git.AddIgnorePattern(fmt.Sprintf("!*%s", domain.PrivateExtension))
}

func (s *Service) Add(files []string) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no files to add")
	}

	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}

	for _, file := range files {
		abs, err := s.toAbs(file)
		if err != nil {
			return err
		}
		exists, err := s.fs.Exists(abs)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no such file: %q", abs)
		}

		rel, err := s.repoRelative(abs)
		if err != nil {
			return err
		}
		rel = cleanRepoPath(rel)

		isDir, err := s.fs.IsDir(abs)
		if err != nil {
			return err
		}
		if isDir {
			if rel == "." {
				return fmt.Errorf("cannot track repository root as a directory")
			}
			if !hasDirectory(list, rel) {
				list.Directories = append(list.Directories, domain.SecureDirectory{Path: rel})
			}
			if err = s.addDirectoryIgnorePatterns(rel); err != nil {
				return err
			}
			if err = s.syncDirectoryFiles(&list, rel); err != nil {
				return err
			}
			continue
		}

		if err = s.addTrackedFile(&list, rel); err != nil {
			return err
		}
	}

	return s.state.StoreFileList(list)
}

func (s *Service) Remove(files []string) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no files to remove")
	}

	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}

	for _, file := range files {
		abs, err := s.toAbs(file)
		if err != nil {
			return err
		}
		rel, err := s.repoRelative(abs)
		if err != nil {
			return err
		}
		rel = cleanRepoPath(rel)

		if hasDirectory(list, rel) {
			if err = s.removeDirectoryIgnorePatterns(rel); err != nil {
				return err
			}
			for _, tracked := range filesInDirectory(list.Files, rel) {
				if err = s.git.RemoveIgnorePattern(tracked.Path); err != nil {
					return err
				}
				privateAbs, err := s.repoAbsolute(tracked.Path + domain.PrivateExtension)
				if err != nil {
					return err
				}
				exists, err := s.fs.Exists(privateAbs)
				if err != nil {
					return err
				}
				if exists {
					if err = s.fs.Remove(privateAbs); err != nil {
						return err
					}
				}
			}
			list = removeDirectoryFromList(list, rel)
			continue
		}

		if err = s.git.RemoveIgnorePattern(rel); err != nil {
			return err
		}
		list = removeFileFromList(list, rel)

		privateAbs, err := s.repoAbsolute(rel + domain.PrivateExtension)
		if err != nil {
			return err
		}
		exists, err := s.fs.Exists(privateAbs)
		if err != nil {
			return err
		}
		if exists {
			if err = s.fs.Remove(privateAbs); err != nil {
				return err
			}
		}
	}

	return s.state.StoreFileList(list)
}

type HideOptions struct {
	Key     string
	KeyFile string
	Clean   bool
	Files   []string
}

func (s *Service) Hide(opts HideOptions) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}

	if err := s.SyncDirectories(); err != nil {
		return err
	}

	filesToHide, err := s.resolveFilesToHide(opts.Files)
	if err != nil {
		return err
	}

	identity, err := s.loader.LoadIdentity(opts.Key, opts.KeyFile)
	if err != nil {
		return err
	}

	recipients, err := s.getRecipients(identity, domain.ReadOnlyAccess)
	if err != nil {
		return fmt.Errorf("failed to load keys, cannot encrypt: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no keys added, cannot encrypt")
	}

	for _, file := range filesToHide {
		if strings.HasSuffix(file, domain.PrivateExtension) {
			return fmt.Errorf("cannot encrypt private file:, %q", file)
		}
		if err = s.encryptFile(file, recipients); err != nil {
			return err
		}
		if err = s.updateFileHash(file); err != nil {
			return err
		}
		if opts.Clean {
			abs, err := s.repoAbsolute(file)
			if err != nil {
				return err
			}
			if err = s.fs.Remove(abs); err != nil {
				return fmt.Errorf("failed to remove source file after encryption: %w", err)
			}
		}
	}

	return nil
}

type RevealOptions struct {
	Key     string
	KeyFile string
	Force   bool
	Clean   bool
	Files   []string
}

type RevealResult struct {
	Revealed int
	InSync   int
}

func (s *Service) Reveal(opts RevealOptions) (RevealResult, error) {
	if err := s.ensureInitialized(); err != nil {
		return RevealResult{}, err
	}

	list, err := s.state.LoadFileList()
	if err != nil {
		return RevealResult{}, err
	}

	filesToReveal, err := s.resolveFilesToReveal(opts.Files, list.Files)
	if err != nil {
		return RevealResult{}, err
	}

	identity, err := s.loader.LoadIdentity(opts.Key, opts.KeyFile)
	if err != nil {
		return RevealResult{}, err
	}

	result := RevealResult{}
	for _, file := range filesToReveal {
		status, err := s.getFileStatus(file)
		if err != nil {
			return RevealResult{}, fmt.Errorf("failed to get file status: %w", err)
		}
		switch status {
		case domain.HiddenInSync:
			result.InSync++
			continue
		case domain.HiddenPrivateMissing:
			return RevealResult{}, fmt.Errorf("cannot reveal, private version of %q is missing", file.Path)
		case domain.HiddenModified:
			if !opts.Force {
				return RevealResult{}, fmt.Errorf("will not overwrite existing file %q without 'force' flag", file.Path)
			}
		case domain.NotHidden:
			return RevealResult{}, fmt.Errorf("file %q is not hidden", file.Path)
		case domain.HiddenNotRevealed:
		}

		if err = s.decryptFile(file.Path, opts.Clean, identity); err != nil {
			return RevealResult{}, fmt.Errorf("reveal failed: %w", err)
		}
		result.Revealed++
	}

	return result, nil
}

func (s *Service) Status() ([]domain.FileStatus, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	if err := s.SyncDirectories(); err != nil {
		return nil, err
	}
	list, err := s.state.LoadFileList()
	if err != nil {
		return nil, fmt.Errorf("failed to load file list: %w", err)
	}

	statuses := make([]domain.FileStatus, 0, len(list.Files))
	for _, file := range list.Files {
		status, err := s.getFileStatus(file)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, domain.FileStatus{File: file, Status: status})
	}
	return statuses, nil
}

func (s *Service) SyncDirectories() error {
	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}
	if len(list.Directories) == 0 {
		return nil
	}
	for _, dir := range list.Directories {
		if err = s.syncDirectoryFiles(&list, dir.Path); err != nil {
			return err
		}
	}
	return s.state.StoreFileList(list)
}

func (s *Service) Clean(force bool) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}

	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}

	for _, file := range list.Files {
		if !force {
			status, err := s.getFileStatus(file)
			if err != nil {
				return err
			}
			if status == domain.HiddenPrivateMissing {
				return fmt.Errorf("will not remove file %q with missing private file, use 'force' flag to override", file.Path)
			}
			if status == domain.HiddenModified {
				return fmt.Errorf("will not remove out of sync file %q, use 'force' flag to override", file.Path)
			}
		}

		abs, err := s.repoAbsolute(file.Path)
		if err != nil {
			return err
		}
		if err = s.fs.Remove(abs); err != nil {
			return fmt.Errorf("failed to remove file %q: %w", file.Path, err)
		}
	}

	return nil
}

func (s *Service) KeysList(keyFile string) ([]domain.Key, error) {
	if err := s.ensureInitialized(); err != nil {
		return nil, err
	}
	identity, err := s.loader.LoadIdentity("", keyFile)
	if err != nil {
		return nil, err
	}
	list, err := s.loadKeyList(identity)
	if err != nil {
		return nil, err
	}
	return list.Keys, nil
}

type KeysAddOptions struct {
	Key      string
	KeyFile  string
	PubFile  string
	ID       string
	ReadOnly bool
	KeyData  string
}

func (s *Service) KeysAdd(opts KeysAddOptions) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}

	key := strings.TrimSpace(opts.KeyData)
	if opts.PubFile != "" {
		data, err := s.fs.ReadFile(opts.PubFile)
		if err != nil {
			return fmt.Errorf("failed to load public key from %q: %w", opts.PubFile, err)
		}
		key = strings.TrimSpace(string(data))
	}
	if key == "" {
		return fmt.Errorf("no public key specified")
	}

	identity, err := s.loader.LoadIdentity(opts.Key, opts.KeyFile)
	if err != nil {
		return err
	}

	keyType, normalized, comment, err := s.crypto.ParseAndNormalizePublicKey(key)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(opts.ID)
	if keyType == domain.SSH {
		if id == "" {
			id = comment
		}
		if id == "" {
			return fmt.Errorf("key has no comment, and no id specified")
		}
	} else {
		if id == "" {
			return fmt.Errorf("cannot add AGE key without id")
		}
	}

	access := domain.ReadWriteAccess
	if opts.ReadOnly {
		access = domain.ReadOnlyAccess
	}

	list, err := s.loadKeyList(identity)
	if err != nil {
		return err
	}
	for _, item := range list.Keys {
		if item.ID == id {
			return fmt.Errorf("key with id %q already exists", id)
		}
	}

	list.Keys = append(list.Keys, domain.Key{Type: keyType, ID: id, Key: normalized, ReadOnly: access == domain.ReadOnlyAccess})
	if err = s.storeKeyList(identity, list); err != nil {
		return err
	}

	inSync, err := s.areFilesInSync()
	if err != nil {
		return err
	}
	if inSync {
		if err = s.reHideFiles(identity); err != nil {
			return fmt.Errorf("failed to re-encrypt files after key addition")
		}
	} else {
		_, _ = io.WriteString(s.stderr, "Files are not in sync, will not re-encrypt after key change. Use 'hide' and/or 'reveal' accordingly.\n")
	}

	return nil
}

func (s *Service) KeysRemove(keyFile, id string) error {
	if err := s.ensureInitialized(); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("specify identity of key to remove")
	}

	identity, err := s.loader.LoadIdentity("", keyFile)
	if err != nil {
		return err
	}

	list, err := s.loadKeyList(identity)
	if err != nil {
		return err
	}

	updated := domain.KeyList{Version: list.Version}
	for _, key := range list.Keys {
		if key.ID != id {
			updated.Keys = append(updated.Keys, key)
		}
	}
	if len(updated.Keys) == len(list.Keys) {
		return fmt.Errorf("key %q not found", id)
	}
	if err = s.storeKeyList(identity, updated); err != nil {
		return err
	}
	if err = s.reHideFiles(identity); err != nil {
		return fmt.Errorf("failed to re-encrypt files after key removal")
	}
	return nil
}

type GenerateOptions struct {
	KeyFile    string
	PubFile    string
	Passphrase []byte
}

type HookOptions struct {
	Key     string
	KeyFile string
}

type MigrateOptions struct {
	DryRun     bool
	Force      bool
	KeepLegacy bool
}

type MigrateResult struct {
	StateFilesCopied int
	EncryptedCopied  int
	LegacyRemoved    bool
}

func (s *Service) KeysGenerate(opts GenerateOptions) error {
	if strings.TrimSpace(opts.KeyFile) == "" {
		return fmt.Errorf("use 'keyfile' flag to specify target file for generated key")
	}
	exists, err := s.fs.Exists(opts.KeyFile)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("will not overwrite existing key file %q", opts.KeyFile)
	}

	private, public, err := s.crypto.GenerateAgeIdentity()
	if err != nil {
		return err
	}

	// В файлах ключей храним только сами ключи без метаданных.
	data := []byte(private + "\n")
	publicLine := public + "\n"

	if opts.PubFile != "" {
		if err = s.fs.WriteFile(opts.PubFile, []byte(publicLine), 0o600); err != nil {
			return fmt.Errorf("failed to write public key file: %w", err)
		}
	} else {
		_, _ = io.WriteString(s.stderr, "Public key: "+publicLine)
	}

	if len(opts.Passphrase) == 0 {
		_, _ = io.WriteString(s.stderr, "no passphrase given, generated key will be stored in clear text\n")
	} else {
		data, err = s.crypto.EncryptWithScryptRecipient(data, opts.Passphrase)
		if err != nil {
			return err
		}
	}

	if err = s.fs.WriteFile(opts.KeyFile, data, 0o600); err != nil {
		return err
	}

	return nil
}

func (s *Service) Migrate(opts MigrateOptions) (MigrateResult, error) {
	root, err := s.git.GetRootPath()
	if err != nil {
		return MigrateResult{}, err
	}

	legacyDir := legacyStateDir(root)
	nativeDir, err := s.state.StateDir()
	if err != nil {
		return MigrateResult{}, err
	}
	legacyExists, err := s.fs.Exists(legacyDir)
	if err != nil {
		return MigrateResult{}, err
	}
	if !legacyExists {
		return MigrateResult{}, fmt.Errorf("legacy state directory %q not found", legacyDir)
	}

	nativeExists, err := s.fs.Exists(nativeDir)
	if err != nil {
		return MigrateResult{}, err
	}
	if nativeExists && !opts.Force {
		return MigrateResult{}, fmt.Errorf("native state directory %q already exists, use -force to overwrite", nativeDir)
	}

	pathsData, err := s.fs.ReadFile(filepath.Join(legacyDir, domain.PathsFileName))
	if err != nil {
		return MigrateResult{}, fmt.Errorf("failed to read legacy paths file: %w", err)
	}
	var list domain.FileList
	if err = json.Unmarshal(pathsData, &list); err != nil {
		return MigrateResult{}, fmt.Errorf("failed to parse legacy paths file: %w", err)
	}

	result := MigrateResult{StateFilesCopied: 1}
	keysPath := filepath.Join(legacyDir, domain.KeysFileName)
	keysData, keysExists, err := s.readOptionalFile(keysPath)
	if err != nil {
		return MigrateResult{}, err
	}
	if keysExists {
		result.StateFilesCopied++
	}

	if opts.DryRun {
		result.EncryptedCopied = len(list.Files)
		return result, nil
	}

	if err = s.fs.MkdirAll(nativeDir, 0o770); err != nil {
		return MigrateResult{}, err
	}
	if err = s.fs.WriteFile(filepath.Join(nativeDir, domain.PathsFileName), pathsData, 0o600); err != nil {
		return MigrateResult{}, err
	}
	if keysExists {
		if err = s.fs.WriteFile(filepath.Join(nativeDir, domain.KeysFileName), keysData, 0o600); err != nil {
			return MigrateResult{}, err
		}
	}

	for _, file := range list.Files {
		legacyPath := filepath.Join(root, file.Path+domain.LegacyPrivateExtension)
		nativePath := filepath.Join(root, file.Path+domain.PrivateExtension)
		if err = s.copyEncryptedFile(legacyPath, nativePath, opts.Force); err != nil {
			return MigrateResult{}, err
		}
		result.EncryptedCopied++
	}

	if err = s.git.AddIgnorePattern(fmt.Sprintf("!*%s", domain.PrivateExtension)); err != nil {
		return MigrateResult{}, err
	}
	if !opts.KeepLegacy {
		if err = s.removeLegacyMigrationFiles(legacyDir, list.Files, keysExists); err != nil {
			return MigrateResult{}, err
		}
		if err = s.git.RemoveIgnorePattern(fmt.Sprintf("!*%s", domain.LegacyPrivateExtension)); err != nil {
			return MigrateResult{}, err
		}
		result.LegacyRemoved = true
	}

	return result, nil
}

func (s *Service) Hook(opts HookOptions) error {
	key := strings.TrimSpace(opts.Key)
	keyFile := strings.TrimSpace(opts.KeyFile)
	if key == "" && keyFile == "" {
		keyFromEnv := strings.TrimSpace(firstEnvValue(domain.PrivateKeyVariable, domain.LegacyPrivateKeyVar))
		keyFileFromEnv := strings.TrimSpace(firstEnvValue(domain.PrivateKeyFileVar, domain.LegacyPrivateKeyFileVar))
		if keyFromEnv == "" && keyFileFromEnv == "" {
			return fmt.Errorf("private key is not configured. Set %s, %s, %s or %s, or use '%s hook -key KEY' or '%s hook -keyfile FILE'",
				domain.PrivateKeyVariable, domain.PrivateKeyFileVar, domain.LegacyPrivateKeyVar, domain.LegacyPrivateKeyFileVar, domain.ToolName, domain.ToolName)
		}
	}

	root, err := s.git.GetRootPath()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(root, ".git", "hooks")
	if err = s.fs.MkdirAll(hooksDir, 0o770); err != nil {
		return err
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	postMergePath := filepath.Join(hooksDir, "post-merge")

	existing, err := s.existingHookFiles(preCommitPath, postMergePath)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		overwrite, err := s.confirmHooksOverwrite(existing)
		if err != nil {
			return err
		}
		if !overwrite {
			return fmt.Errorf("hook installation canceled")
		}
		if err = s.backupHookFiles(existing); err != nil {
			return err
		}
	}

	keyFlag := ""
	if key != "" {
		keyFlag = " -key " + shellSingleQuote(key)
	} else if keyFile != "" {
		keyFlag = " -keyfile " + shellSingleQuote(fullPath(keyFile))
	}

	preCommitScript := "#!/bin/bash\n" +
		"echo \"git-safe: running hide before commit...\"\n" +
		"git-safe hide" + keyFlag + "\n" +
		"if ! git diff-index --quiet HEAD --; then\n" +
		"  echo \"git-safe: changed files detected after hide, adding to commit...\"\n" +
		"  git add -u\n" +
		"  mapfile -d '' safe_files < <(git ls-files -z --others --exclude-standard -- '*" + domain.PrivateExtension + "')\n" +
		"  if [ ${#safe_files[@]} -gt 0 ]; then\n" +
		"    git add -- \"${safe_files[@]}\"\n" +
		"  fi\n" +
		"fi\n"
	if err = s.fs.WriteFile(preCommitPath, []byte(preCommitScript), 0o700); err != nil {
		return err
	}
	if err = s.fs.Chmod(preCommitPath, 0o700); err != nil {
		return err
	}

	postMergeScript := "#!/bin/bash\n" +
		"echo \"git-safe: revealing files after merge...\"\n" +
		"git-safe reveal" + keyFlag + "\n"
	if err = s.fs.WriteFile(postMergePath, []byte(postMergeScript), 0o700); err != nil {
		return err
	}
	if err = s.fs.Chmod(postMergePath, 0o700); err != nil {
		return err
	}

	return nil
}

func firstEnvValue(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func (s *Service) readOptionalFile(path string) ([]byte, bool, error) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to read %q: %w", path, err)
	}
	return data, true, nil
}

func (s *Service) copyEncryptedFile(src string, dst string, force bool) error {
	exists, err := s.fs.Exists(src)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("legacy encrypted file %q not found", src)
	}
	dstExists, err := s.fs.Exists(dst)
	if err != nil {
		return err
	}
	if dstExists && !force {
		return fmt.Errorf("native encrypted file %q already exists, use -force to overwrite", dst)
	}
	data, err := s.fs.ReadFile(src)
	if err != nil {
		return err
	}
	return s.fs.WriteFile(dst, data, 0o600)
}

func (s *Service) removeLegacyMigrationFiles(legacyDir string, files []domain.SecureFile, keysExists bool) error {
	root, err := s.git.GetRootPath()
	if err != nil {
		return err
	}
	for _, file := range files {
		path := filepath.Join(root, file.Path+domain.LegacyPrivateExtension)
		exists, err := s.fs.Exists(path)
		if err != nil {
			return err
		}
		if exists {
			if err = s.fs.Remove(path); err != nil {
				return err
			}
		}
	}
	if keysExists {
		if err = s.fs.Remove(filepath.Join(legacyDir, domain.KeysFileName)); err != nil {
			return err
		}
	}
	if err = s.fs.Remove(filepath.Join(legacyDir, domain.PathsFileName)); err != nil {
		return err
	}
	return s.fs.Remove(legacyDir)
}

func legacyStateDir(root string) string {
	dir := os.Getenv(domain.LegacyPrivateDirVariable)
	if dir == "" {
		dir = domain.LegacyPrivateDirName
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

func (s *Service) resolveFilesToHide(args []string) ([]string, error) {
	if len(args) == 0 {
		list, err := s.state.LoadFileList()
		if err != nil {
			return nil, err
		}
		result := make([]string, 0, len(list.Files))
		for _, file := range list.Files {
			result = append(result, file.Path)
		}
		return result, nil
	}

	result := make([]string, 0, len(args))
	for _, arg := range args {
		abs, err := s.toAbs(arg)
		if err != nil {
			return nil, err
		}
		rel, err := s.repoRelative(abs)
		if err != nil {
			return nil, err
		}
		result = append(result, rel)
	}
	return result, nil
}

func (s *Service) resolveFilesToReveal(args []string, files []domain.SecureFile) ([]domain.SecureFile, error) {
	if len(args) == 0 {
		return files, nil
	}

	result := make([]domain.SecureFile, 0, len(args))
	for _, arg := range args {
		abs, err := s.toAbs(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path to %q", arg)
		}
		file, err := s.findFile(abs, files)
		if errors.Is(err, errNotFound) {
			return nil, fmt.Errorf("file %q is not hidden", arg)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to look up file: %w", err)
		}
		result = append(result, file)
	}
	return result, nil
}

func (s *Service) findFile(abs string, files []domain.SecureFile) (domain.SecureFile, error) {
	rel, err := s.repoRelative(abs)
	if err != nil {
		return domain.SecureFile{}, err
	}
	for _, file := range files {
		if file.Path == rel {
			return file, nil
		}
	}
	return domain.SecureFile{}, errNotFound
}

func (s *Service) encryptFile(rel string, recipients []age.Recipient) error {
	fullPath, err := s.repoAbsolute(rel)
	if err != nil {
		return err
	}
	plain, err := s.fs.ReadFile(fullPath)
	if err != nil {
		return err
	}
	cipher, err := s.crypto.Encrypt(plain, recipients)
	if err != nil {
		return err
	}
	return s.fs.WriteFile(fullPath+domain.PrivateExtension, cipher, 0o600)
}

func (s *Service) decryptFile(rel string, clean bool, identity age.Identity) error {
	fullPath, err := s.repoAbsolute(rel)
	if err != nil {
		return err
	}
	privatePath := fullPath + domain.PrivateExtension
	cipher, err := s.fs.ReadFile(privatePath)
	if err != nil {
		return err
	}
	plain, err := s.crypto.Decrypt(cipher, identity)
	if err != nil {
		return err
	}
	if err = s.fs.WriteFile(fullPath, plain, 0o660); err != nil {
		return err
	}
	if clean {
		if err = s.fs.Remove(privatePath); err != nil {
			return fmt.Errorf("revealed, but failed to remove private file: %w", err)
		}
	}
	return nil
}

func (s *Service) updateFileHash(rel string) error {
	fullPath, err := s.repoAbsolute(rel)
	if err != nil {
		return err
	}
	hash, err := s.hasher.SHA256File(fullPath)
	if err != nil {
		return err
	}
	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}
	for i, file := range list.Files {
		if file.Path == rel {
			list.Files[i].Hash = hash
			return s.state.StoreFileList(list)
		}
	}
	return fmt.Errorf("file %q not in file list", rel)
}

func (s *Service) getFileStatus(file domain.SecureFile) (domain.StatusCode, error) {
	fullPath, err := s.repoAbsolute(file.Path)
	if err != nil {
		return 0, err
	}

	if file.Hash == "" {
		return domain.NotHidden, nil
	}

	privatePath := fullPath + domain.PrivateExtension
	privateExists, err := s.fs.Exists(privatePath)
	if err != nil {
		return 0, err
	}
	if !privateExists {
		return domain.HiddenPrivateMissing, nil
	}

	plainExists, err := s.fs.Exists(fullPath)
	if err != nil {
		return 0, err
	}
	if !plainExists {
		return domain.HiddenNotRevealed, nil
	}

	hash, err := s.hasher.SHA256File(fullPath)
	if err != nil {
		return 0, err
	}
	if hash == file.Hash {
		return domain.HiddenInSync, nil
	}
	return domain.HiddenModified, nil
}

func (s *Service) areFilesInSync() (bool, error) {
	list, err := s.state.LoadFileList()
	if err != nil {
		return false, err
	}
	for _, file := range list.Files {
		status, err := s.getFileStatus(file)
		if err != nil {
			return false, err
		}
		if status != domain.HiddenInSync {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) reHideFiles(identity age.Identity) error {
	list, err := s.state.LoadFileList()
	if err != nil {
		return err
	}
	recipients, err := s.getRecipients(identity, domain.ReadOnlyAccess)
	if err != nil {
		return err
	}
	for _, file := range list.Files {
		if err = s.encryptFile(file.Path, recipients); err != nil {
			return err
		}
		if err = s.updateFileHash(file.Path); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadKeyList(identity age.Identity) (domain.KeyList, error) {
	cipher, exists, err := s.state.ReadKeysData()
	if err != nil {
		return domain.KeyList{}, err
	}
	if !exists {
		return domain.KeyList{}, nil
	}
	plain, err := s.crypto.Decrypt(cipher, identity)
	if err != nil {
		return domain.KeyList{}, fmt.Errorf("key list decryption failed")
	}
	var list domain.KeyList
	if err = json.Unmarshal(plain, &list); err != nil {
		return domain.KeyList{}, err
	}
	return list, nil
}

func (s *Service) storeKeyList(identity age.Identity, list domain.KeyList) error {
	_, exists, err := s.state.ReadKeysData()
	if err != nil {
		return err
	}
	if exists {
		if _, err = s.getRecipients(identity, domain.ReadOnlyAccess); err != nil {
			return err
		}
	}

	recipients, err := s.crypto.RecipientsFromKeyList(list.Keys, domain.ReadWriteAccess)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return fmt.Errorf("cannot update key list, no keys with read/write access")
	}

	list.Version = 1
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	cipher, err := s.crypto.Encrypt(data, recipients)
	if err != nil {
		return err
	}
	return s.state.WriteKeysData(cipher)
}

func (s *Service) getRecipients(identity age.Identity, access domain.KeyAccess) ([]age.Recipient, error) {
	list, err := s.loadKeyList(identity)
	if err != nil {
		return nil, err
	}
	return s.crypto.RecipientsFromKeyList(list.Keys, access)
}

func (s *Service) ensureInitialized() error {
	dir, err := s.state.StateDir()
	if err != nil {
		return err
	}
	exists, err := s.fs.Exists(dir)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return fmt.Errorf("not initialized, run '%s init'", domain.ToolName)
}

func (s *Service) toAbs(path string) (string, error) {
	if s.fs.IsAbs(path) {
		return path, nil
	}
	return s.fs.Abs(path)
}

func (s *Service) repoAbsolute(repoRelative string) (string, error) {
	root, err := s.git.GetRootPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, repoRelative), nil
}

func (s *Service) repoRelative(absPath string) (string, error) {
	root, err := s.git.GetRootPath()
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", err
	}
	return cleanRepoPath(rel), nil
}

func (s *Service) addTrackedFile(list *domain.FileList, path string) error {
	path = cleanRepoPath(path)
	if hasFile(*list, path) {
		return nil
	}
	list.Files = append(list.Files, domain.SecureFile{Path: path})
	return s.git.AddIgnorePattern(path)
}

func (s *Service) addDirectoryIgnorePatterns(dir string) error {
	for _, pattern := range directoryIgnorePatterns(dir) {
		if err := s.git.AddIgnorePattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) removeDirectoryIgnorePatterns(dir string) error {
	for _, pattern := range directoryIgnorePatterns(dir) {
		if err := s.git.RemoveIgnorePattern(pattern); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncDirectoryFiles(list *domain.FileList, dir string) error {
	files, err := s.filesInTrackedDirectory(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err = s.addTrackedFile(list, file); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) filesInTrackedDirectory(dir string) ([]string, error) {
	absDir, err := s.repoAbsolute(dir)
	if err != nil {
		return nil, err
	}
	exists, err := s.fs.Exists(absDir)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	isDir, err := s.fs.IsDir(absDir)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fmt.Errorf("tracked directory %q is not a directory", dir)
	}

	root, err := s.git.GetRootPath()
	if err != nil {
		return nil, err
	}
	stateDir, err := s.state.StateDir()
	if err != nil {
		return nil, err
	}

	paths, err := s.fs.WalkFiles(absDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if shouldSkipDirectoryFile(path, root, stateDir) {
			continue
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		result = append(result, cleanRepoPath(rel))
	}
	sort.Strings(result)
	return result, nil
}

func shouldSkipDirectoryFile(path string, root string, stateDir string) bool {
	rel := cleanRepoPath(mustRel(root, path))
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	stateRel := cleanRepoPath(mustRel(root, stateDir))
	if rel == stateRel || strings.HasPrefix(rel, stateRel+"/") {
		return true
	}
	if strings.HasSuffix(rel, domain.PrivateExtension) || strings.HasSuffix(rel, domain.LegacyPrivateExtension) {
		return true
	}
	return false
}

func mustRel(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func cleanRepoPath(path string) string {
	cleaned := filepath.ToSlash(filepath.Clean(path))
	if cleaned == "." {
		return "."
	}
	return strings.TrimPrefix(cleaned, "./")
}

func hasFile(list domain.FileList, path string) bool {
	path = cleanRepoPath(path)
	for _, file := range list.Files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func hasDirectory(list domain.FileList, path string) bool {
	path = cleanRepoPath(path)
	for _, dir := range list.Directories {
		if dir.Path == path {
			return true
		}
	}
	return false
}

func filesInDirectory(files []domain.SecureFile, dir string) []domain.SecureFile {
	dir = cleanRepoPath(dir)
	result := []domain.SecureFile{}
	for _, file := range files {
		if pathInDirectory(file.Path, dir) {
			result = append(result, file)
		}
	}
	return result
}

func pathInDirectory(path string, dir string) bool {
	path = cleanRepoPath(path)
	dir = cleanRepoPath(dir)
	if dir == "." {
		return path != "."
	}
	return path == dir || strings.HasPrefix(path, dir+"/")
}

func directoryIgnorePatterns(dir string) []string {
	dir = cleanRepoPath(dir)
	return []string{
		dir + "/**",
		"!" + dir + "/**/",
		"!" + dir + "/**/*" + domain.PrivateExtension,
	}
}

func removeFileFromList(list domain.FileList, path string) domain.FileList {
	path = cleanRepoPath(path)
	updated := domain.FileList{Version: list.Version, Directories: list.Directories}
	for _, file := range list.Files {
		if file.Path != path {
			updated.Files = append(updated.Files, file)
		}
	}
	return updated
}

func removeDirectoryFromList(list domain.FileList, path string) domain.FileList {
	path = cleanRepoPath(path)
	updated := domain.FileList{Version: list.Version}
	for _, dir := range list.Directories {
		if dir.Path != path {
			updated.Directories = append(updated.Directories, dir)
		}
	}
	for _, file := range list.Files {
		if !pathInDirectory(file.Path, path) {
			updated.Files = append(updated.Files, file)
		}
	}
	return updated
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func fullPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(os.Getenv("PWD"), path)
}

func (s *Service) existingHookFiles(paths ...string) ([]string, error) {
	existing := make([]string, 0, len(paths))
	for _, path := range paths {
		ok, err := s.fs.Exists(path)
		if err != nil {
			return nil, err
		}
		if ok {
			existing = append(existing, path)
		}
	}
	return existing, nil
}

func (s *Service) confirmHooksOverwrite(paths []string) (bool, error) {
	if len(paths) == 0 {
		return true, nil
	}

	names := make([]string, 0, len(paths))
	for _, path := range paths {
		names = append(names, filepath.Base(path))
	}
	_, _ = fmt.Fprintf(s.stdout, "Existing hook(s) detected: %s\n", strings.Join(names, ", "))
	_, _ = io.WriteString(s.stdout, "Overwrite and create .bak backup? [y/N]: ")

	reader := bufio.NewReader(s.stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (s *Service) backupHookFiles(paths []string) error {
	for _, path := range paths {
		data, err := s.fs.ReadFile(path)
		if err != nil {
			return err
		}
		backupPath := path + ".bak"
		if err = s.fs.WriteFile(backupPath, data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

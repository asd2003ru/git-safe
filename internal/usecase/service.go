package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"git-safe/cmd"
	"git-safe/internal/adapters/cryptoage"
	"git-safe/internal/adapters/keyloader"
	runtimeadp "git-safe/internal/adapters/runtime"
	"git-safe/internal/adapters/statestore"
	"git-safe/internal/domain"
	"git-safe/internal/gitclient"
	"git-safe/internal/ports"
)

const (
	privateExt = ".safe"
	toolName   = "git-safe"
)

// Service реализует бизнес-операции git-safe через порты инфраструктуры.
type Service struct {
	git       ports.Git
	store     ports.StateStore
	crypto    ports.Crypto
	keyLoader ports.KeyLoader
	clock     ports.Clock
	io        ports.IO
}

// Deps содержит зависимости, необходимые для инициализации Service.
type Deps struct {
	Git       ports.Git
	Store     ports.StateStore
	Crypto    ports.Crypto
	KeyLoader ports.KeyLoader
	Clock     ports.Clock
	IO        ports.IO
}

// NewService создает use-case сервис и валидирует все обязательные зависимости.
func NewService(d Deps) (*Service, error) {
	if d.Git == nil || d.Store == nil || d.Crypto == nil || d.KeyLoader == nil || d.Clock == nil || d.IO == nil {
		return nil, wrapErr("new_service", ErrInvalidInput, fmt.Errorf("all service dependencies are required"))
	}
	return &Service{
		git:       d.Git,
		store:     d.Store,
		crypto:    d.Crypto,
		keyLoader: d.KeyLoader,
		clock:     d.Clock,
		io:        d.IO,
	}, nil
}

// NewDefaultService собирает сервис со стандартными адаптерами проекта.
func NewDefaultService() (*Service, error) {
	gc, err := gitclient.NewFromEnv("")
	if err != nil {
		return nil, wrapErr("new_default_service", ErrInternal, err)
	}
	return NewService(Deps{
		Git:       gc,
		Store:     statestore.NewFileStateStore(),
		Crypto:    cryptoage.New(),
		KeyLoader: keyloader.NewEnvKeyLoader(),
		Clock:     runtimeadp.NewSystemClock(),
		IO:        runtimeadp.NewStdIO(),
	})
}

var _ cmd.UseCases = (*Service)(nil)

// Init инициализирует состояние git-safe внутри текущего репозитория.
func (s *Service) Init(ctx context.Context) error {
	const op = "init"
	// Этап 1: проверяем контекст репозитория.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}

	// Этап 2: проверяем, что состояние еще не инициализировано.
	initialized, err := s.store.IsInitialized(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if initialized {
		return wrapErr(op, ErrAlreadyInit, fmt.Errorf("already initialized"))
	}

	// Этап 3: создаем состояние и добавляем исключение для файлов .safe.
	if err := s.store.Init(root); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if err := s.git.AddIgnorePattern(ctx, "!*"+privateExt); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// Add добавляет файлы в список отслеживания и в .gitignore.
func (s *Service) Add(ctx context.Context, files []string) error {
	const op = "add"
	if len(files) == 0 {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("no files to add"))
	}

	// Этап 1: проверяем репозиторий и инициализацию состояния.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}

	// Этап 2: нормализуем входные пути и проверяем существование файлов.
	paths, err := s.toRepoPaths(root, files, true)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	// Этап 3: загружаем текущее состояние и добавляем новые записи.
	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	for _, path := range paths {
		if !hasTracked(tracked, path) {
			tracked = append(tracked, domain.SecretFile{Path: path})
			if err := s.git.AddIgnorePattern(ctx, path); err != nil {
				return wrapErr(op, ErrInternal, err)
			}
		}
	}

	// Этап 4: сохраняем обновленное состояние.
	if err := s.store.StoreFiles(root, tracked); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// Remove удаляет файлы из списка отслеживания, .gitignore и связанные .safe файлы.
func (s *Service) Remove(ctx context.Context, files []string) error {
	const op = "remove"
	if len(files) == 0 {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("no files to remove"))
	}

	// Этап 1: проверяем репозиторий и инициализацию состояния.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}

	// Этап 2: нормализуем пути и загружаем текущее состояние.
	paths, err := s.toRepoPaths(root, files, false)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	// Этап 3: фильтруем список отслеживаемых файлов.
	filtered := make([]domain.SecretFile, 0, len(tracked))
	for _, item := range tracked {
		if !slices.Contains(paths, item.Path) {
			filtered = append(filtered, item)
		}
	}

	// Этап 4: удаляем правила игнорирования и побочные .safe файлы.
	for _, path := range paths {
		if err := s.git.RemoveIgnorePattern(ctx, path); err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		privatePath := filepath.Join(root, path+privateExt)
		if exists, err := pathExists(privatePath); err != nil {
			return wrapErr(op, ErrInternal, err)
		} else if exists {
			if err := os.Remove(privatePath); err != nil {
				return wrapErr(op, ErrInternal, err)
			}
		}
	}

	// Этап 5: сохраняем обновленное состояние.
	if err := s.store.StoreFiles(root, filtered); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// Hide шифрует выбранные файлы в .safe и обновляет контрольные хеши.
func (s *Service) Hide(ctx context.Context, input cmd.HideInput) error {
	const op = "hide"
	// Этап 1: проверяем окружение и доступ к ключу пользователя.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	if _, err := s.keyLoader.Load(input.KeyFile); err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	// Этап 2: загружаем состояние файлов и список ключей-получателей.
	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	keys, err := s.store.LoadKeys(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if len(keys) == 0 {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("no keys added, cannot encrypt"))
	}

	// Этап 3: определяем целевые файлы hide-операции.
	targets, err := pickTargets(tracked, input.Files)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	// Этап 4: шифруем каждый целевой файл и опционально удаляем исходник.
	for i, item := range tracked {
		if !slices.Contains(targets, item.Path) {
			continue
		}
		sourcePath := filepath.Join(root, item.Path)
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return wrapErr(op, ErrNotFound, fmt.Errorf("%q: %w", item.Path, err))
		}
		encrypted, err := s.crypto.Encrypt(data, keys)
		if err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		if err := os.WriteFile(sourcePath+privateExt, encrypted, 0o660); err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		sum := sha256.Sum256(data)
		tracked[i].Hash = hex.EncodeToString(sum[:])
		if input.Clean {
			if err := os.Remove(sourcePath); err != nil {
				return wrapErr(op, ErrInternal, fmt.Errorf("failed to remove source file after hide: %w", err))
			}
		}
	}

	// Этап 5: сохраняем обновленные хеши.
	if err := s.store.StoreFiles(root, tracked); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// Reveal расшифровывает .safe файлы обратно в исходные пути.
func (s *Service) Reveal(ctx context.Context, input cmd.RevealInput) error {
	const op = "reveal"
	// Этап 1: проверяем окружение и загружаем identity для расшифровки.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	identity, err := s.keyLoader.Load(input.KeyFile)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	// Этап 2: загружаем отслеживаемые файлы и выбираем цели.
	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	targets, err := pickTargets(tracked, input.Files)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	// Этап 3: валидируем состояние файлов и выполняем расшифровку.
	revealed := 0
	for _, item := range tracked {
		if !slices.Contains(targets, item.Path) {
			continue
		}

		st, err := s.fileStatus(root, item)
		if err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		sourcePath := filepath.Join(root, item.Path)
		safePath := sourcePath + privateExt

		switch st {
		case domain.StatusNotHidden:
			continue
		case domain.StatusHiddenSafeMiss:
			return wrapErr(op, ErrNotFound, fmt.Errorf("cannot reveal, private version of %q is missing", item.Path))
		case domain.StatusHiddenModified:
			if !input.Force {
				return wrapErr(op, ErrConflict, fmt.Errorf("file %q is out of sync, use --force", item.Path))
			}
		}

		if exists, err := pathExists(sourcePath); err != nil {
			return wrapErr(op, ErrInternal, err)
		} else if exists && !input.Force {
			if st == domain.StatusHiddenNotRevealed {
				continue
			}
			return wrapErr(op, ErrConflict, fmt.Errorf("file %q exists, use --force", item.Path))
		}

		cipher, err := os.ReadFile(safePath)
		if err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		plain, err := s.crypto.Decrypt(cipher, identity)
		if err != nil {
			return wrapErr(op, ErrForbidden, fmt.Errorf("reveal failed for %q: %w", item.Path, err))
		}
		if err := os.WriteFile(sourcePath, plain, 0o660); err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		if input.Clean {
			if err := os.Remove(safePath); err != nil {
				return wrapErr(op, ErrInternal, fmt.Errorf("revealed, but failed to remove private file: %w", err))
			}
		}
		revealed++
	}

	// Этап 4: выводим итог операции.
	if revealed > 0 {
		fmt.Fprintf(s.io.Stdout(), "%d file(s) revealed\n", revealed)
	}
	return nil
}

// Clean удаляет раскрытые исходные файлы по правилам синхронизации.
func (s *Service) Clean(ctx context.Context, input cmd.CleanInput) error {
	const op = "clean"
	// Этап 1: проверяем окружение и состояние.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}

	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	// Этап 2: проходим по отслеживаемым файлам и удаляем разрешенные.
	for _, item := range tracked {
		sourcePath := filepath.Join(root, item.Path)
		if exists, err := pathExists(sourcePath); err != nil || !exists {
			if err != nil {
				return wrapErr(op, ErrInternal, err)
			}
			continue
		}

		if !input.Force {
			st, err := s.fileStatus(root, item)
			if err != nil {
				return wrapErr(op, ErrInternal, err)
			}
			if st == domain.StatusHiddenSafeMiss {
				return wrapErr(op, ErrConflict, fmt.Errorf("will not remove %q with missing private file, use --force", item.Path))
			}
			if st == domain.StatusHiddenModified {
				return wrapErr(op, ErrConflict, fmt.Errorf("will not remove out of sync %q, use --force", item.Path))
			}
		}

		if err := os.Remove(sourcePath); err != nil {
			return wrapErr(op, ErrInternal, fmt.Errorf("failed to remove %q: %w", item.Path, err))
		}
	}
	return nil
}

// Status выводит текущий статус всех отслеживаемых файлов.
func (s *Service) Status(ctx context.Context) error {
	const op = "status"
	// Этап 1: проверяем контекст репозитория и инициализацию состояния.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}

	tracked, err := s.store.LoadFiles(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	// Этап 2: печатаем табличный отчет по состояниям файлов.
	w := tabwriter.NewWriter(s.io.Stdout(), 0, 0, 4, ' ', 0)
	if len(tracked) == 0 {
		fmt.Fprintln(w, "No private files")
		_ = w.Flush()
		return nil
	}

	for _, item := range tracked {
		st, err := s.fileStatus(root, item)
		if err != nil {
			return wrapErr(op, ErrInternal, err)
		}
		fmt.Fprintf(w, "%s\t[%s]\n", item.Path, st)
	}
	_ = w.Flush()
	return nil
}

// KeysList печатает список ключей из состояния.
func (s *Service) KeysList(ctx context.Context, input cmd.KeysListInput) error {
	const op = "keys.list"
	// Этап 1: проверяем доступ и загружаем список ключей.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	identity, err := s.keyLoader.Load(input.KeyFile)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}
	if err := s.requireReadWriteAccess(root, identity); err != nil {
		return wrapErr(op, ErrForbidden, err)
	}

	keys, err := s.store.LoadKeys(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	// Этап 2: выводим ключи в компактном табличном формате.
	for _, k := range keys {
		access := string(k.Access)
		if access == "" {
			if k.ReadOnly {
				access = string(domain.KeyAccessReadOnly)
			} else {
				access = string(domain.KeyAccessReadWrite)
			}
		}
		fmt.Fprintf(s.io.Stdout(), "%s\t%s\t%s\n", k.ID, k.Type, access)
	}
	return nil
}

// KeysAdd добавляет новый публичный ключ в состояние.
func (s *Service) KeysAdd(ctx context.Context, input cmd.KeysAddInput) error {
	const op = "keys.add"
	// Этап 1: проверяем доступ к операции.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	identity, err := s.keyLoader.Load(input.KeyFile)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}
	if err := s.requireReadWriteAccess(root, identity); err != nil {
		return wrapErr(op, ErrForbidden, err)
	}

	// Этап 2: читаем и классифицируем ключ.
	keyData, err := readKeyData(input.PubFile, input.PublicKey)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}
	typ, inferredID, err := classifyPublicKey(keyData)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}

	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = inferredID
	}
	if typ == domain.KeyTypeAGE && id == "" {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("cannot add AGE key without id"))
	}
	if id == "" {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("cannot infer key id, use --id"))
	}

	// Этап 3: валидируем уникальность идентификатора.
	keys, err := s.store.LoadKeys(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	for _, k := range keys {
		if k.ID == id {
			return wrapErr(op, ErrConflict, fmt.Errorf("key with id %q already exists", id))
		}
	}

	access := domain.KeyAccessReadWrite
	if input.ReadOnly {
		access = domain.KeyAccessReadOnly
	}
	keys = append(keys, domain.Key{
		Type:     typ,
		Key:      keyData,
		ID:       id,
		Access:   access,
		ReadOnly: input.ReadOnly,
	})
	// Этап 4: сохраняем обновленный список ключей.
	if err := s.store.StoreKeys(root, keys); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// KeysRemove удаляет ключ из состояния по идентификатору.
func (s *Service) KeysRemove(ctx context.Context, input cmd.KeysRemoveInput) error {
	const op = "keys.remove"
	// Этап 1: проверяем доступ к операции.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	identity, err := s.keyLoader.Load(input.KeyFile)
	if err != nil {
		return wrapErr(op, ErrInvalidInput, err)
	}
	if err := s.requireReadWriteAccess(root, identity); err != nil {
		return wrapErr(op, ErrForbidden, err)
	}

	targetID := strings.TrimSpace(input.ID)
	if targetID == "" {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("specify identity of key to remove"))
	}

	keys, err := s.store.LoadKeys(root)
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	// Этап 2: фильтруем список ключей и проверяем факт удаления.
	filtered := make([]domain.Key, 0, len(keys))
	removed := false
	for _, k := range keys {
		if k.ID == targetID {
			removed = true
			continue
		}
		filtered = append(filtered, k)
	}
	if !removed {
		return wrapErr(op, ErrNotFound, fmt.Errorf("key with id %q not found", targetID))
	}

	// Этап 3: сохраняем обновленный список ключей.
	if err := s.store.StoreKeys(root, filtered); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	return nil
}

// KeysGenerate генерирует пару ключей и сохраняет их в файлы.
func (s *Service) KeysGenerate(ctx context.Context, input cmd.KeysGenerateInput) error {
	const op = "keys.generate"
	// Этап 1: проверяем контекст и валидируем параметры.
	root, err := s.repoRoot(ctx)
	if err != nil {
		return err
	}
	if err := s.ensureInitialized(root); err != nil {
		return wrapErr(op, ErrNotInitialized, err)
	}
	if strings.TrimSpace(input.KeyFile) == "" {
		return wrapErr(op, ErrInvalidInput, fmt.Errorf("--keyfile is required"))
	}

	// Этап 2: вычисляем итоговые пути файлов private/public ключей.
	privateAbs := filepath.Join(root, input.KeyFile)
	if filepath.IsAbs(input.KeyFile) {
		privateAbs = input.KeyFile
	}
	publicOut := strings.TrimSpace(input.PubFile)
	if publicOut == "" {
		publicOut = input.KeyFile + ".pub"
	}
	publicAbs := filepath.Join(root, publicOut)
	if filepath.IsAbs(publicOut) {
		publicAbs = publicOut
	}

	// Этап 3: генерируем ключевую пару.
	privateKey, publicKey, err := s.crypto.GenerateKeyPair()
	if err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	// Этап 4: сохраняем ключи на диск и печатаем результат.
	if err := os.MkdirAll(filepath.Dir(privateAbs), 0o770); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if err := os.WriteFile(privateAbs, []byte(privateKey+"\n"), 0o600); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if err := os.MkdirAll(filepath.Dir(publicAbs), 0o770); err != nil {
		return wrapErr(op, ErrInternal, err)
	}
	if err := os.WriteFile(publicAbs, []byte(publicKey+"\n"), 0o660); err != nil {
		return wrapErr(op, ErrInternal, err)
	}

	fmt.Fprintf(s.io.Stdout(), "generated keypair:\nprivate: %s\npublic: %s\n", privateAbs, publicAbs)
	return nil
}

// repoRoot получает корень текущего git-репозитория.
func (s *Service) repoRoot(ctx context.Context) (string, error) {
	const op = "repo_root"
	ok, err := s.git.IsInsideWorkTree(ctx)
	if err != nil {
		return "", wrapErr(op, ErrInternal, err)
	}
	if !ok {
		return "", wrapErr(op, ErrNotInRepo, fmt.Errorf("not in git repository"))
	}
	root, err := s.git.RepoRoot(ctx)
	if err != nil {
		return "", wrapErr(op, ErrInternal, err)
	}
	return root, nil
}

// ensureInitialized проверяет, что состояние git-safe уже создано.
func (s *Service) ensureInitialized(repoRoot string) error {
	initialized, err := s.store.IsInitialized(repoRoot)
	if err != nil {
		return err
	}
	if !initialized {
		return fmt.Errorf("not initialized, run '%s init'", toolName)
	}
	return nil
}

// fileStatus вычисляет статус одного отслеживаемого файла.
func (s *Service) fileStatus(repoRoot string, item domain.SecretFile) (domain.Status, error) {
	full := filepath.Join(repoRoot, item.Path)
	if item.Hash == "" {
		return domain.StatusNotHidden, nil
	}

	safePath := full + privateExt
	safeExists, err := pathExists(safePath)
	if err != nil {
		return "", err
	}
	if !safeExists {
		return domain.StatusHiddenSafeMiss, nil
	}

	sourceExists, err := pathExists(full)
	if err != nil {
		return "", err
	}
	if !sourceExists {
		return domain.StatusHiddenNotRevealed, nil
	}

	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) == item.Hash {
		return domain.StatusHiddenInSync, nil
	}
	return domain.StatusHiddenModified, nil
}

// hasTracked проверяет наличие относительного пути в списке отслеживаемых файлов.
func hasTracked(files []domain.SecretFile, rel string) bool {
	for _, item := range files {
		if item.Path == rel {
			return true
		}
	}
	return false
}

// pickTargets выбирает целевые файлы для операции по аргументам CLI.
func pickTargets(files []domain.SecretFile, args []string) ([]string, error) {
	if len(args) == 0 {
		out := make([]string, 0, len(files))
		for _, f := range files {
			out = append(out, f.Path)
		}
		return out, nil
	}

	out := make([]string, 0, len(args))
	for _, arg := range args {
		clean := strings.TrimSpace(arg)
		if clean == "" {
			continue
		}
		if strings.HasSuffix(clean, privateExt) {
			clean = strings.TrimSuffix(clean, privateExt)
		}
		found := false
		for _, f := range files {
			if f.Path == clean {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("file %q is not tracked", clean)
		}
		out = append(out, clean)
	}
	return out, nil
}

// toRepoPaths нормализует пути к repo-relative и опционально проверяет существование.
func (s *Service) toRepoPaths(root string, files []string, mustExist bool) ([]string, error) {
	out := make([]string, 0, len(files))
	for _, file := range files {
		target := file
		if !filepath.IsAbs(target) {
			target = filepath.Join(root, target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
			return nil, fmt.Errorf("path %q is outside repository root", file)
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if mustExist {
			if exists, err := pathExists(abs); err != nil {
				return nil, err
			} else if !exists {
				return nil, fmt.Errorf("no such file: %q", file)
			}
		}
		out = append(out, rel)
	}
	return out, nil
}

// readKeyData читает публичный ключ из файла или позиционного аргумента.
func readKeyData(pubFile string, inline string) (string, error) {
	if strings.TrimSpace(pubFile) != "" {
		data, err := os.ReadFile(pubFile)
		if err != nil {
			return "", err
		}
		key := strings.TrimSpace(string(data))
		if key == "" {
			return "", fmt.Errorf("public key file is empty")
		}
		return key, nil
	}
	key := strings.TrimSpace(inline)
	if key == "" {
		return "", fmt.Errorf("no public key provided")
	}
	return key, nil
}

// classifyPublicKey определяет тип публичного ключа и пытается извлечь ID.
func classifyPublicKey(key string) (domain.KeyType, string, error) {
	if strings.HasPrefix(key, "age1") {
		return domain.KeyTypeAGE, "", nil
	}
	if strings.HasPrefix(key, "ssh-") {
		parts := strings.Fields(key)
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid ssh public key")
		}
		id := ""
		if len(parts) >= 3 {
			id = parts[2]
		}
		return domain.KeyTypeSSH, id, nil
	}
	return "", "", fmt.Errorf("unsupported public key format")
}

// pathExists проверяет существование файла или каталога.
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// requireReadWriteAccess проверяет, что identity соответствует ключу с rw-доступом.
func (s *Service) requireReadWriteAccess(repoRoot string, identity string) error {
	keys, err := s.store.LoadKeys(repoRoot)
	if err != nil {
		return err
	}
	// Пустой список ключей: bootstrap-сценарий добавления первого ключа.
	if len(keys) == 0 {
		return nil
	}

	for _, k := range keys {
		if k.Access == domain.KeyAccessReadOnly || k.ReadOnly {
			continue
		}
		ok, err := s.identityMatchesKey(identity, k)
		if err != nil {
			continue
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("provided key has no read-write access")
}

// identityMatchesKey проверяет соответствие identity конкретному публичному ключу.
func (s *Service) identityMatchesKey(identity string, key domain.Key) (bool, error) {
	const probe = "git-safe-access-probe"
	cipher, err := s.crypto.Encrypt([]byte(probe), []domain.Key{key})
	if err != nil {
		return false, err
	}
	plain, err := s.crypto.Decrypt(cipher, identity)
	if err != nil {
		return false, nil
	}
	return string(plain) == probe, nil
}

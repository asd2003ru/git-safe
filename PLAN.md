# План разработки Git Safe

## Цель
Построить новую реализацию Git Safe на Clean Architecture с совместимым CLI-поведеним относительно legacy `git-private` (только как read-only reference).

## Этап 1. Базовый каркас CLI
- [x] Подключить Cobra и создать дерево команд.
- [x] Добавить парсинг аргументов/флагов по `RUN_VARIANTS.md`.
- [x] Добавить единый слой `command handlers` (вызовы use-case вместо `noOp`).

## Этап 2. Domain и Use Cases
- [x] Описать доменные сущности: SecretFile, Key, KeyAccess, Status.
- [x] Реализовать use-cases:
  - [x] `init`
  - [x] `add` / `remove`
  - [x] `hide` / `reveal`
  - [x] `clean`
  - [x] `status`
  - [x] `keys list/add/remove/generate`
- [x] Ввести типизированные ошибки use-case уровня.

## Этап 3. Ports и Adapters
- [x] Зафиксировать ports: Git, StateStore, Crypto, KeyLoader, Clock/IO.
- [x] Создать универсальный Git модуль с backend switch (`legacy` / `go-git`).
- [x] Реализовать файловый `StateStore` (версирование структуры, атомарная запись).
- [x] Реализовать crypto adapter на `age` + SSH recipient/identity.
- [x] Реализовать key loading с приоритетом: `-keyfile` -> `GIT_SAFE_KEY` -> `GIT_SAFE_KEYFILE`.

## Этап 4. Интеграция CLI <-> Use Cases
- [x] Привязать каждый Cobra command к соответствующему use-case.
- [x] Нормализовать формат ошибок и exit codes.
- [x] Сохранить UX-фразы, критичные для совместимости.

## Этап 5. Тестирование
Для тестирования можно временно создавать тестовые ключи в директории проекта. Можно создавать тестовые файлы и тестировать на них.
- [x] Unit-тесты для use-cases (table-driven).
- [x] Тесты adapter-слоя (FS, Git backend, key parsing).
- [x] Интеграционные тесты CLI сценариев.

## Этап 6. Миграция и стабилизация
- [ ] Сравнить поведение новой реализации и legacy по всем командам.
- [ ] Зафиксировать известные несовместимости (если останутся).
- [ ] Подготовить обновление README с примерами и переменными окружения.

## Этап 7. TUI интерфейс
- [ ] Определить сценарии запуска без аргументов и карту экранов TUI.
- [ ] Реализовать базовый shell TUI на Bubble Tea (layout, navigation, quit/help).
- [ ] Добавить экран статуса приватных файлов (список, фильтрация, обновление).
- [ ] Добавить экран управления файлами (`add/remove/hide/reveal/clean`) через use-case слой.
- [ ] Добавить экран управления ключами (`list/add/remove/generate`) через use-case слой.
- [ ] Добавить обработку ошибок и подтверждений действий в TUI (modals/alerts).
- [ ] Обеспечить паритет бизнес-логики: TUI использует те же use-cases, что и CLI.
- [ ] Добавить smoke/integration тесты критичных TUI flow (запуск, навигация, основные операции).

## Технические решения
- Основной Git backend: `go-git`.
- Fallback backend: `legacy git CLI` через `GIT_SAFE_GIT_BACKEND`.
- `git-private` не редактируется, используется только для чтения контекста.

## Definition of Done (MVP)
- Все команды из `RUN_VARIANTS.md` выполняют бизнес-логику, а не заглушки.
- Основные сценарии `init/add/hide/reveal/status/keys` покрыты тестами.
- Запуск без аргументов открывает рабочий TUI с базовыми сценариями статуса, файлов и ключей.
- `go test ./...` стабильно проходит.
- Поведение документировано и воспроизводимо.

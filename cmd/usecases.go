package cmd

import "context"

// UseCases описывает бизнес-операции, доступные из CLI слоя.
type UseCases interface {
	Init(ctx context.Context) error
	Add(ctx context.Context, files []string) error
	Remove(ctx context.Context, files []string) error
	Hide(ctx context.Context, input HideInput) error
	Reveal(ctx context.Context, input RevealInput) error
	Clean(ctx context.Context, input CleanInput) error
	Status(ctx context.Context) error
	KeysList(ctx context.Context, input KeysListInput) error
	KeysAdd(ctx context.Context, input KeysAddInput) error
	KeysRemove(ctx context.Context, input KeysRemoveInput) error
	KeysGenerate(ctx context.Context, input KeysGenerateInput) error
}

// NoopUseCases пустая реализация интерфейса для заглушек и тестовых сценариев.
type NoopUseCases struct{}

// NewNoopUseCases создает экземпляр пустой реализации use-case интерфейса.
func NewNoopUseCases() NoopUseCases {
	return NoopUseCases{}
}

// Init возвращает nil и не выполняет действий.
func (NoopUseCases) Init(context.Context) error { return nil }

// Add возвращает nil и не выполняет действий.
func (NoopUseCases) Add(context.Context, []string) error { return nil }

// Remove возвращает nil и не выполняет действий.
func (NoopUseCases) Remove(context.Context, []string) error { return nil }

// Hide возвращает nil и не выполняет действий.
func (NoopUseCases) Hide(context.Context, HideInput) error { return nil }

// Reveal возвращает nil и не выполняет действий.
func (NoopUseCases) Reveal(context.Context, RevealInput) error { return nil }

// Clean возвращает nil и не выполняет действий.
func (NoopUseCases) Clean(context.Context, CleanInput) error { return nil }

// Status возвращает nil и не выполняет действий.
func (NoopUseCases) Status(context.Context) error { return nil }

// KeysList возвращает nil и не выполняет действий.
func (NoopUseCases) KeysList(context.Context, KeysListInput) error { return nil }

// KeysAdd возвращает nil и не выполняет действий.
func (NoopUseCases) KeysAdd(context.Context, KeysAddInput) error { return nil }

// KeysRemove возвращает nil и не выполняет действий.
func (NoopUseCases) KeysRemove(context.Context, KeysRemoveInput) error { return nil }

// KeysGenerate возвращает nil и не выполняет действий.
func (NoopUseCases) KeysGenerate(context.Context, KeysGenerateInput) error { return nil }

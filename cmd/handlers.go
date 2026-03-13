package cmd

import (
	"context"
	"fmt"
)

// CommandHandlers связывает CLI команды с use-case интерфейсом.
type CommandHandlers struct {
	uc UseCases
}

// NewCommandHandlers создает фасад между Cobra командами и use-case слоем.
func NewCommandHandlers(uc UseCases) (*CommandHandlers, error) {
	if uc == nil {
		return nil, fmt.Errorf("use-cases dependency is required")
	}
	return &CommandHandlers{uc: uc}, nil
}

// HideInput параметры команды hide.
type HideInput struct {
	KeyFile string
	Clean   bool
	Files   []string
}

// RevealInput параметры команды reveal.
type RevealInput struct {
	KeyFile string
	Force   bool
	Clean   bool
	Files   []string
}

// CleanInput параметры команды clean.
type CleanInput struct {
	Force bool
}

// KeysListInput параметры команды keys list.
type KeysListInput struct {
	KeyFile string
}

// KeysAddInput параметры команды keys add.
type KeysAddInput struct {
	KeyFile   string
	ID        string
	PubFile   string
	ReadOnly  bool
	PublicKey string
}

// KeysRemoveInput параметры команды keys remove.
type KeysRemoveInput struct {
	KeyFile string
	ID      string
}

// KeysGenerateInput параметры команды keys generate.
type KeysGenerateInput struct {
	KeyFile string
	PubFile string
}

// Init делегирует выполнение команды init в use-case слой.
func (h *CommandHandlers) Init(ctx context.Context) error {
	return h.uc.Init(ctx)
}

// Add делегирует выполнение команды add в use-case слой.
func (h *CommandHandlers) Add(ctx context.Context, files []string) error {
	return h.uc.Add(ctx, files)
}

// Remove делегирует выполнение команды remove в use-case слой.
func (h *CommandHandlers) Remove(ctx context.Context, files []string) error {
	return h.uc.Remove(ctx, files)
}

// Hide делегирует выполнение команды hide в use-case слой.
func (h *CommandHandlers) Hide(ctx context.Context, in HideInput) error {
	return h.uc.Hide(ctx, in)
}

// Reveal делегирует выполнение команды reveal в use-case слой.
func (h *CommandHandlers) Reveal(ctx context.Context, in RevealInput) error {
	return h.uc.Reveal(ctx, in)
}

// Clean делегирует выполнение команды clean в use-case слой.
func (h *CommandHandlers) Clean(ctx context.Context, in CleanInput) error {
	return h.uc.Clean(ctx, in)
}

// Status делегирует выполнение команды status в use-case слой.
func (h *CommandHandlers) Status(ctx context.Context) error {
	return h.uc.Status(ctx)
}

// KeysList делегирует выполнение команды keys list в use-case слой.
func (h *CommandHandlers) KeysList(ctx context.Context, in KeysListInput) error {
	return h.uc.KeysList(ctx, in)
}

// KeysAdd делегирует выполнение команды keys add в use-case слой.
func (h *CommandHandlers) KeysAdd(ctx context.Context, in KeysAddInput) error {
	return h.uc.KeysAdd(ctx, in)
}

// KeysRemove делегирует выполнение команды keys remove в use-case слой.
func (h *CommandHandlers) KeysRemove(ctx context.Context, in KeysRemoveInput) error {
	return h.uc.KeysRemove(ctx, in)
}

// KeysGenerate делегирует выполнение команды keys generate в use-case слой.
func (h *CommandHandlers) KeysGenerate(ctx context.Context, in KeysGenerateInput) error {
	return h.uc.KeysGenerate(ctx, in)
}

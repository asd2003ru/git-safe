package runtime

import (
	"io"
	"os"
	"time"

	"git-safe/internal/ports"
)

// SystemClock реализация порта Clock через системное время.
type SystemClock struct{}

// NewSystemClock создает системные часы для use-case слоя.
func NewSystemClock() ports.Clock {
	return SystemClock{}
}

// Now возвращает текущее локальное системное время.
func (SystemClock) Now() time.Time {
	return time.Now()
}

// StdIO реализация порта IO через стандартные потоки процесса.
type StdIO struct{}

// NewStdIO создает адаптер стандартных потоков ввода/вывода.
func NewStdIO() ports.IO {
	return StdIO{}
}

// Stdout возвращает поток стандартного вывода процесса.
func (StdIO) Stdout() io.Writer {
	return os.Stdout
}

// Stderr возвращает поток ошибок процесса.
func (StdIO) Stderr() io.Writer {
	return os.Stderr
}

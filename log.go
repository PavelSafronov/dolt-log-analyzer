package main

import (
	"fmt"
	"os"
	"testing"
)

type Logger interface {
	// Log formats its arguments using default formatting, analogous to Println,
	// and records the text in the error log. For tests, the text will be printed only if
	// the test fails or the -test.v flag is set. For benchmarks, the text is always
	// printed to avoid having performance depend on the value of the -test.v flag.
	Log(args ...any)

	// Logf formats its arguments according to the format, analogous to Printf, and
	// records the text in the error log. A final newline is added if not provided. For
	// tests, the text will be printed only if the test fails or the -test.v flag is
	// set. For benchmarks, the text is always printed to avoid having performance
	// depend on the value of the -test.v flag.
	Logf(format string, args ...any)
}

var _ Logger = (*ConsoleLogger)(nil)

type ConsoleLogger struct{}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

func (l *ConsoleLogger) Log(args ...any) {
	fmt.Print(args...)
}
func (l *ConsoleLogger) Logf(format string, args ...any) {
	fmt.Printf(format, args...)
}

var _ Logger = (*TestLogger)(nil)

type TestLogger struct {
	t *testing.T
}

func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}
func (l *TestLogger) Log(args ...interface{}) {
	l.t.Log(args...)
}
func (l *TestLogger) Logf(format string, args ...interface{}) {
	l.t.Logf(format, args...)
}

var _ Logger = (*FileLogger)(nil)

type FileLogger struct {
	File *os.File
}

func (f FileLogger) Log(args ...any) {
	message := fmt.Sprint(args...)
	_, err := f.File.WriteString(message)
	if err != nil {
		panic(err)
	}
}

func (f FileLogger) Logf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	_, err := f.File.WriteString(message)
	if err != nil {
		panic(err)
	}
}

func NewFileLogger(file *os.File) *FileLogger {
	return &FileLogger{File: file}
}

var _ Logger = (*MultiLogger)(nil)

type MultiLogger struct {
	Loggers []Logger
}

func (p MultiLogger) Log(args ...any) {
	for _, logger := range p.Loggers {
		if logger != nil {
			logger.Log(args...)
		}
	}
}

func (p MultiLogger) Logf(format string, args ...any) {
	for _, logger := range p.Loggers {
		if logger != nil {
			logger.Logf(format, args...)
		}
	}
}

func NewProxyLogger(loggers ...Logger) *MultiLogger {
	return &MultiLogger{Loggers: loggers}
}

type NoopLogger struct{}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (n NoopLogger) Log(args ...any) {
}

func (n NoopLogger) Logf(format string, args ...any) {
}

var _ Logger = (*NoopLogger)(nil)

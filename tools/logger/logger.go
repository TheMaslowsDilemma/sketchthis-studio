package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents logging severity
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging for the sketch studio
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	minLevel Level
	prefix   string
}

// New creates a new logger
func New(out io.Writer, minLevel Level, prefix string) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{
		out:      out,
		minLevel: minLevel,
		prefix:   prefix,
	}
}

// Default returns a default logger to stdout
func Default() *Logger {
	return New(os.Stdout, LevelInfo, "")
}

// WithPrefix creates a sub-logger with an additional prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	newPrefix := prefix
	if l.prefix != "" {
		newPrefix = l.prefix + "/" + prefix
	}
	return &Logger{
		out:      l.out,
		minLevel: l.minLevel,
		prefix:   newPrefix,
	}
}

func (l *Logger) log(level Level, format string, args ...any) {
	if level < l.minLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	prefix := ""
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}

	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s %s %s%s\n", timestamp, level.String(), prefix, msg)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, format, args...)
}

// Step logs a named step with timing
func (l *Logger) Step(name string) func() {
	start := time.Now()
	l.Info("▶ Starting: %s", name)
	return func() {
		l.Info("✓ Completed: %s (took %v)", name, time.Since(start).Round(time.Millisecond))
	}
}

// Tokens logs token usage
func (l *Logger) Tokens(input, output int) {
	l.Info("📊 Tokens - Input: %d, Output: %d, Total: %d", input, output, input+output)
}

// Section logs section information
func (l *Logger) Section(title string, description string) {
	l.Info("📐 Section [%s]: %s", title, description)
}

// Compilation logs compilation result
func (l *Logger) Compilation(success bool, path string, errors []string) {
	if success {
		l.Info("✓ Compiled successfully: %s", path)
	} else {
		l.Error("✗ Compilation failed: %s", path)
		for _, err := range errors {
			l.Error("  - %s", err)
		}
	}
}

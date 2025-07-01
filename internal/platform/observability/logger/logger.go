package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Level defines log levels.
type Level slog.Level

const (
	LevelDebug = Level(slog.LevelDebug)
	LevelInfo  = Level(slog.LevelInfo)
	LevelWarn  = Level(slog.LevelWarn)
	LevelError = Level(slog.LevelError)
)

// Logger defines the interface for our structured logger.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger // To add contextual fields
}

// SlogLogger is the concrete implementation using slog.
type SlogLogger struct {
	handler slog.Handler
}

// New creates a new logger instance based on the provided configuration.
func New(level Level, output io.Writer) Logger {
	if output == nil {
		output = os.Stdout
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: slog.Level(level),
	})

	return &SlogLogger{handler: handler}
}

// --- Implementation of the Logger interface ---

func (l *SlogLogger) log(level slog.Level, msg string, args ...any) {
	slog.New(l.handler).Log(context.Background(), level, msg, args...)
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.log(slog.LevelDebug, msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.log(slog.LevelInfo, msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.log(slog.LevelWarn, msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.log(slog.LevelError, msg, args...)
}

// With returns a new logger with the specified contextual fields.
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{handler: l.handler.WithAttrs(argsToAttr(args))}
}

// Helper to convert a flat list of key-value pairs to slog.Attr.
func argsToAttr(args []any) []slog.Attr {
	var attrs []slog.Attr
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			attrs = append(attrs, slog.Any(args[i].(string), args[i+1]))
		}
	}
	return attrs
}

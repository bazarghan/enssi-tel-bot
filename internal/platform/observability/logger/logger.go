package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the interface for our structured logger.
// This interface remains unchanged, so the rest of your app is unaffected.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger // To add contextual fields
}

// zapLogger is the concrete implementation using zap.
type zapLogger struct {
	sugaredLogger *zap.SugaredLogger
}

// New creates a new logger instance based on the provided log level string.
func New(level string) (Logger, error) {
	// Parse the log level string.
	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Create a new zap configuration.
	// We use the production config for efficient, structured JSON logging.
	// You can switch to zap.NewDevelopmentConfig() for more human-readable logs during development.
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)

	// Build the logger from the config.
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	// We use the SugaredLogger for its convenience with key-value pairs (args...).
	return &zapLogger{sugaredLogger: logger.Sugar()}, nil
}

// --- Implementation of the Logger interface ---

func (l *zapLogger) Debug(msg string, args ...any) {
	l.sugaredLogger.Debugw(msg, args...)
}

func (l *zapLogger) Info(msg string, args ...any) {
	l.sugaredLogger.Infow(msg, args...)
}

func (l *zapLogger) Warn(msg string, args ...any) {
	l.sugaredLogger.Warnw(msg, args...)
}

func (l *zapLogger) Error(msg string, args ...any) {
	l.sugaredLogger.Errorw(msg, args...)
}

// With returns a new logger with the specified contextual fields.
func (l *zapLogger) With(args ...any) Logger {
	return &zapLogger{sugaredLogger: l.sugaredLogger.With(args...)}
}

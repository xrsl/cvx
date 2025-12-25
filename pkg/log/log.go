package log

import (
	"io"
	"log/slog"
	"os"
)

var (
	// logger is the global logger instance
	logger *slog.Logger
	// level controls the log level
	level = new(slog.LevelVar)
)

func init() {
	// Default to warning level (quiet mode)
	level.Set(slog.LevelWarn)
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}

// SetVerbose enables debug logging
func SetVerbose(verbose bool) {
	if verbose {
		level.Set(slog.LevelDebug)
	} else {
		level.Set(slog.LevelWarn)
	}
}

// SetQuiet disables all logging except errors
func SetQuiet(quiet bool) {
	if quiet {
		level.Set(slog.LevelError)
	}
}

// SetOutput changes the log output destination
func SetOutput(w io.Writer) {
	logger = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	}))
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	return logger.With(args...)
}

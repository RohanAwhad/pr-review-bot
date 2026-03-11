package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func New(component string) (*slog.Logger, io.Closer, string, error) {
	logDir := filepath.Join(".", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, nil, "", err
	}

	logPath := filepath.Join(logDir, component+".log")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, "", err
	}

	handler := slog.NewTextHandler(io.MultiWriter(os.Stderr, file), &slog.HandlerOptions{
		Level: parseLevel(os.Getenv("LOGGING_LEVEL")),
	})

	logger := slog.New(handler).With("component", component)
	return logger, file, logPath, nil
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

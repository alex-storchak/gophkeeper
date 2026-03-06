package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/alex-storchak/gophkeeper/internal/client/config"
)

func New(cfg config.Log) (*slog.Logger, func() error, error) {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var writer io.Writer = os.Stderr
	var cleanup = func() error {
		return nil
	}

	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open file for log: %w", err)
		}
		writer = f
		cleanup = f.Close
	}

	h := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(h), cleanup, nil
}

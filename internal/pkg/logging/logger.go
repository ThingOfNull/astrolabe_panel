// Package logging configures slog.
//
// Handlers:
//   - console: color when TTY
//   - json: newline JSON
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config selects slog sinks.
type Config struct {
	Level  string
	Format string
}

// Init configures slog and returns setup errors.
// Stdout-only for now.
func Init(cfg Config) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	logger, err := build(os.Stdout, cfg.Format, level)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)
	return logger, nil
}

// build allows injecting writers in tests.
func build(w io.Writer, format string, level slog.Level) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: level}
	switch strings.ToLower(format) {
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case "console", "":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("unsupported log format: %q", format)
	}
}

func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %q", raw)
	}
}

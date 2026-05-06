package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestBuildJSON(t *testing.T) {
	var buf bytes.Buffer
	logger, err := build(&buf, "json", slog.LevelInfo)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	logger.Info("hello", "k", "v")
	out := buf.String()
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("expected json output, got: %s", out)
	}
}

func TestBuildConsole(t *testing.T) {
	var buf bytes.Buffer
	logger, err := build(&buf, "console", slog.LevelDebug)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	logger.Debug("dbg-msg")
	out := buf.String()
	if !strings.Contains(out, "dbg-msg") {
		t.Errorf("expected text output to contain message, got: %s", out)
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"":      slog.LevelInfo,
		"debug": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for in, want := range cases {
		got, err := parseLevel(in)
		if err != nil {
			t.Errorf("parseLevel(%q) unexpected error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := parseLevel("noisy"); err == nil {
		t.Errorf("expected error for invalid level")
	}
}

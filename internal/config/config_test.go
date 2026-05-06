package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandUser(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home: %v", err)
	}

	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"~", home},
		{"~/foo/bar", filepath.Join(home, "foo", "bar")},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
	}
	for _, tc := range cases {
		got, err := ExpandUser(tc.in)
		if err != nil {
			t.Errorf("ExpandUser(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ExpandUser(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLoadCreatesDefaults(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.json")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.ListenPort != 8080 {
		t.Errorf("default port = %d, want 8080", cfg.Server.ListenPort)
	}
	if cfg.I18n.DefaultLocale != "zh-CN" {
		t.Errorf("default locale = %q, want zh-CN", cfg.I18n.DefaultLocale)
	}
	if cfg.SourcePath != cfgPath {
		t.Errorf("source path = %q, want %q", cfg.SourcePath, cfgPath)
	}

	for _, dir := range []string{cfg.Paths.DataDir, cfg.Paths.UploadDir, cfg.Paths.LogDir} {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			t.Errorf("expected directory %q to exist: %v", dir, statErr)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %q to be a directory", dir)
		}
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal written config: %v", err)
	}
	if _, ok := generic["paths"]; !ok {
		t.Errorf("written config missing 'paths' key: %s", string(raw))
	}
}

func TestLoadOverridesFromFile(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "config.json")
	override := map[string]any{
		"server": map[string]any{
			"listen_host": "127.0.0.1",
			"listen_port": 9090,
		},
		"paths": map[string]any{
			"data_dir":   filepath.Join(tmp, "data"),
			"db_path":    filepath.Join(tmp, "data", "astro.db"),
			"upload_dir": filepath.Join(tmp, "data", "uploads"),
			"log_dir":    filepath.Join(tmp, "data", "logs"),
		},
		"log": map[string]any{
			"level":  "debug",
			"format": "json",
		},
	}
	raw, err := json.Marshal(override)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.ListenPort != 9090 {
		t.Errorf("listen_port = %d, want 9090", cfg.Server.ListenPort)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("log.format = %q, want json", cfg.Log.Format)
	}
	if cfg.Metric.RetentionMinutes != 30 {
		t.Errorf("metric default retention should fall back to 30, got %d", cfg.Metric.RetentionMinutes)
	}
}

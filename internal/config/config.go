// Package config loads config.json for the daemon.
//
// Resolution order:
//  1. CLI --config <path>
//  2. ASTROLABE_CONFIG env
//  3. Default ~/.astrolabe_panel/config.json
//
// When the path is missing, a default file and layout dirs are created.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Default directory under the user's home.
const (
	defaultDirName  = ".astrolabe_panel"
	defaultFileName = "config.json"
	envConfigPath   = "ASTROLABE_CONFIG"
)

// Config is the top-level schema for config.json.
type Config struct {
	Server  ServerConfig  `json:"server"  mapstructure:"server"`
	Paths   PathsConfig   `json:"paths"   mapstructure:"paths"`
	Log     LogConfig     `json:"log"     mapstructure:"log"`
	I18n    I18nConfig    `json:"i18n"    mapstructure:"i18n"`
	Iconify IconifyConfig `json:"iconify" mapstructure:"iconify"`
	Metric  MetricConfig  `json:"metric"  mapstructure:"metric"`
	Probe   ProbeConfig   `json:"probe"   mapstructure:"probe"`

	// SourcePath records the resolved file path at runtime.
	SourcePath string `json:"-" mapstructure:"-"`
}

// ServerConfig controls HTTP listen/bind.
type ServerConfig struct {
	ListenHost string `json:"listen_host" mapstructure:"listen_host"`
	ListenPort int    `json:"listen_port" mapstructure:"listen_port"`
	BaseURL    string `json:"base_url"    mapstructure:"base_url"`
}

// PathsConfig lists on-disk paths for data, db, uploads, logs.
type PathsConfig struct {
	DataDir   string `json:"data_dir"   mapstructure:"data_dir"`
	DBPath    string `json:"db_path"    mapstructure:"db_path"`
	UploadDir string `json:"upload_dir" mapstructure:"upload_dir"`
	LogDir    string `json:"log_dir"    mapstructure:"log_dir"`
}

// LogConfig selects slog level and format.
type LogConfig struct {
	Level  string `json:"level"  mapstructure:"level"`
	Format string `json:"format" mapstructure:"format"`
}

// I18nConfig picks default locale for server messages.
type I18nConfig struct {
	DefaultLocale string `json:"default_locale" mapstructure:"default_locale"`
}

// IconifyConfig sets upstream mirror base URL.
type IconifyConfig struct {
	Mirror string `json:"mirror" mapstructure:"mirror"`
}

// MetricConfig controls sample retention in SQLite.
type MetricConfig struct {
	RetentionMinutes       int `json:"retention_minutes"        mapstructure:"retention_minutes"`
	CleanupIntervalMinutes int `json:"cleanup_interval_minutes" mapstructure:"cleanup_interval_minutes"`
}

// ProbeConfig sets default link probe timing.
type ProbeConfig struct {
	DefaultIntervalSec int `json:"default_interval_sec" mapstructure:"default_interval_sec"`
	DefaultTimeoutSec  int `json:"default_timeout_sec"  mapstructure:"default_timeout_sec"`
}

// Default returns built-in defaults (paths may contain ~ before expansion).
func Default() Config {
	return Config{
		Server: ServerConfig{
			ListenHost: "0.0.0.0",
			ListenPort: 8080,
			BaseURL:    "",
		},
		Paths: PathsConfig{
			DataDir:   "~/.astrolabe_panel",
			DBPath:    "~/.astrolabe_panel/astrolabe.db",
			UploadDir: "~/.astrolabe_panel/uploads",
			LogDir:    "~/.astrolabe_panel/logs",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "console",
		},
		I18n: I18nConfig{
			DefaultLocale: "zh-CN",
		},
		Iconify: IconifyConfig{
			Mirror: "",
		},
		Metric: MetricConfig{
			RetentionMinutes:       30,
			CleanupIntervalMinutes: 5,
		},
		Probe: ProbeConfig{
			DefaultIntervalSec: 30,
			DefaultTimeoutSec:  4,
		},
	}
}

// Load resolves CLI > env > default path, optionally seeding defaults, then unmarshals JSON.
// If the target file is missing, defaults are written before read.
func Load(cliPath string) (*Config, error) {
	target, err := resolveConfigPath(cliPath)
	if err != nil {
		return nil, err
	}

	if err := ensureConfigFile(target); err != nil {
		return nil, fmt.Errorf("ensure config file: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(target)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", target, err)
	}

	cfg := Default()
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.SourcePath = target

	if err := cfg.expandAndValidate(); err != nil {
		return nil, err
	}

	if err := cfg.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("ensure directories: %w", err)
	}

	return &cfg, nil
}

// resolveConfigPath returns the absolute config file location.
func resolveConfigPath(cliPath string) (string, error) {
	candidate := strings.TrimSpace(cliPath)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv(envConfigPath))
	}
	if candidate == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate user home dir: %w", err)
		}
		candidate = filepath.Join(home, defaultDirName, defaultFileName)
	}

	expanded, err := ExpandUser(candidate)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("abs path %q: %w", expanded, err)
	}
	return abs, nil
}

// ensureConfigFile writes Default() JSON when path is absent.
func ensureConfigFile(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
		return mkErr
	}

	data, marshalErr := json.MarshalIndent(Default(), "", "  ")
	if marshalErr != nil {
		return marshalErr
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

// expandAndValidate expands ~ in paths and checks numeric fields.
func (c *Config) expandAndValidate() error {
	pathFields := []*string{&c.Paths.DataDir, &c.Paths.DBPath, &c.Paths.UploadDir, &c.Paths.LogDir}
	for _, p := range pathFields {
		expanded, err := ExpandUser(*p)
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(expanded)
		if err != nil {
			return err
		}
		*p = abs
	}

	if c.Server.ListenPort <= 0 || c.Server.ListenPort > 65535 {
		return fmt.Errorf("invalid server.listen_port: %d", c.Server.ListenPort)
	}
	if c.Metric.RetentionMinutes <= 0 {
		return fmt.Errorf("invalid metric.retention_minutes: %d", c.Metric.RetentionMinutes)
	}
	if c.Metric.CleanupIntervalMinutes <= 0 {
		return fmt.Errorf("invalid metric.cleanup_interval_minutes: %d", c.Metric.CleanupIntervalMinutes)
	}

	switch strings.ToLower(c.Log.Format) {
	case "console", "json":
	default:
		return fmt.Errorf("invalid log.format: %q", c.Log.Format)
	}

	return nil
}

// ensureDirectories mkdirs dirs referenced by Paths (DB file excluded).
func (c *Config) ensureDirectories() error {
	dirs := []string{
		c.Paths.DataDir,
		c.Paths.UploadDir,
		c.Paths.LogDir,
		filepath.Dir(c.Paths.DBPath),
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %q: %w", d, err)
		}
	}
	return nil
}

// Addr returns listen_host:listen_port for net.Listen.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.ListenHost, c.Server.ListenPort)
}

// ExpandUser turns leading ~ paths into absolute home-relative paths.
// Non-tilde paths are returned unchanged.
func ExpandUser(path string) (string, error) {
	if path == "" {
		return path, nil
	}
	if path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate user home dir: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, path[2:]), nil
	}
	return "", fmt.Errorf("unsupported user prefix in path: %q", path)
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	Shell       string      `json:"shell"`
	HistoryPath string      `json:"history_path"`
	Local       LocalConfig `json:"local"`
}

type LocalConfig struct {
	MaxHistory int `json:"max_history"`
}

func defaultShell() string {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("CLIAI_SHELL"))); v != "" {
		switch v {
		case "powershell", "pwsh":
			return "powershell"
		case "bash", "zsh", "fish":
			return v
		}
	}

	switch runtime.GOOS {
	case "windows":
		return "powershell"
	case "darwin":
		return "zsh"
	default:
		if shell := filepath.Base(strings.TrimSpace(os.Getenv("SHELL"))); shell != "" {
			switch shell {
			case "pwsh":
				return "powershell"
			case "bash", "zsh", "fish":
				return shell
			}
		}
		return "bash"
	}
}

func defaultHistoryPath(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	switch shell {
	case "powershell":
		if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "Microsoft", "Windows", "PowerShell", "PSReadLine", "ConsoleHost_history.txt")
		}
		return filepath.Join(home, ".local", "share", "powershell", "PSReadLine", "ConsoleHost_history.txt")
	case "zsh":
		return filepath.Join(home, ".zsh_history")
	case "fish":
		return filepath.Join(home, ".local", "share", "fish", "fish_history")
	case "bash":
		fallthrough
	default:
		return filepath.Join(home, ".bash_history")
	}
}

func Default() *Config {
	shell := defaultShell()
	return &Config{
		Shell:       shell,
		HistoryPath: defaultHistoryPath(shell),
		Local: LocalConfig{
			MaxHistory: 4000,
		},
	}
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cliai"), nil
}

func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func HistoryCachePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history_cache.json"), nil
}

func FeedbackPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "feedback.json"), nil
}

func ensureDir() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

func Load() (*Config, error) {
	cfg := Default()
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			applyEnv(cfg)
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyEnv(cfg)
	return cfg, nil
}

func Save(cfg *Config) error {
	if err := ensureDir(); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("CLIAI_SHELL"); v != "" {
		cfg.Shell = v
	}
	if v := os.Getenv("CLIAI_HISTORY_PATH"); v != "" {
		cfg.HistoryPath = v
	}
}

func Set(cfg *Config, key string, value string) error {
	switch strings.ToLower(key) {
	case "shell":
		oldShell := cfg.Shell
		oldDefaultHistoryPath := defaultHistoryPath(oldShell)
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "powershell", "pwsh":
			cfg.Shell = "powershell"
		case "bash", "zsh", "fish":
			cfg.Shell = strings.ToLower(strings.TrimSpace(value))
		default:
			return fmt.Errorf("unsupported shell: %s (supported: powershell, bash, zsh, fish)", value)
		}
		if strings.TrimSpace(cfg.HistoryPath) == "" || cfg.HistoryPath == oldDefaultHistoryPath {
			cfg.HistoryPath = defaultHistoryPath(cfg.Shell)
		}
	case "history_path":
		cfg.HistoryPath = value
	case "local.max_history":
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("local.max_history must be a number")
		}
		cfg.Local.MaxHistory = n
	default:
		return fmt.Errorf("unsupported config key: %s", key)
	}
	return nil
}

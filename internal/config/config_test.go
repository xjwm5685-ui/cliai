package config

import "testing"

func TestSetShellNormalizesPwsh(t *testing.T) {
	cfg := Default()
	if err := Set(cfg, "shell", "pwsh"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if cfg.Shell != "powershell" {
		t.Fatalf("expected powershell, got %q", cfg.Shell)
	}
}

func TestSetShellSupportsZsh(t *testing.T) {
	cfg := Default()
	if err := Set(cfg, "shell", "zsh"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if cfg.Shell != "zsh" {
		t.Fatalf("expected zsh, got %q", cfg.Shell)
	}
}

func TestSetShellRejectsUnsupportedValue(t *testing.T) {
	cfg := Default()
	if err := Set(cfg, "shell", "cmd"); err == nil {
		t.Fatalf("expected unsupported shell error")
	}
}

func TestSetShellUpdatesDefaultHistoryPath(t *testing.T) {
	cfg := &Config{
		Shell:       "powershell",
		HistoryPath: defaultHistoryPath("powershell"),
	}

	if err := Set(cfg, "shell", "zsh"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if cfg.Shell != "zsh" {
		t.Fatalf("expected zsh, got %q", cfg.Shell)
	}
	if cfg.HistoryPath != defaultHistoryPath("zsh") {
		t.Fatalf("expected zsh history path %q, got %q", defaultHistoryPath("zsh"), cfg.HistoryPath)
	}
}

func TestSetShellPreservesCustomHistoryPath(t *testing.T) {
	cfg := &Config{
		Shell:       "powershell",
		HistoryPath: "/tmp/custom-history.txt",
	}

	if err := Set(cfg, "shell", "bash"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if cfg.HistoryPath != "/tmp/custom-history.txt" {
		t.Fatalf("expected custom history path to be preserved, got %q", cfg.HistoryPath)
	}
}

func TestSetLocalMaxHistory(t *testing.T) {
	cfg := Default()
	if err := Set(cfg, "local.max_history", "5000"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if cfg.Local.MaxHistory != 5000 {
		t.Fatalf("expected max history 5000, got %d", cfg.Local.MaxHistory)
	}
}

func TestSetRejectsRemovedOpenAIKeys(t *testing.T) {
	cfg := Default()
	for _, key := range []string{
		"openai.enabled",
		"openai.base_url",
		"openai.api_key",
		"openai.model",
		"openai.timeout_seconds",
	} {
		if err := Set(cfg, key, "test"); err == nil {
			t.Fatalf("expected removed key %q to be rejected", key)
		}
	}
}

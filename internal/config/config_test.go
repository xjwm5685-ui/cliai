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

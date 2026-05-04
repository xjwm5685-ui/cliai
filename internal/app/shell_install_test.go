package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindInstallPowerShellScriptPrefersSiblingScriptsDir(t *testing.T) {
	tempDir := t.TempDir()
	exePath := filepath.Join(tempDir, "bin", "cliai.exe")
	scriptPath := filepath.Join(tempDir, "bin", "scripts", "install-powershell.ps1")

	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("prepare scripts dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("prepare exe dir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write exe stub: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write script stub: %v", err)
	}

	got := findInstallPowerShellScript(exePath)
	if got != scriptPath {
		t.Fatalf("expected %q, got %q", scriptPath, got)
	}
}

func TestFindInstallPowerShellScriptFallsBackToParentScriptsDir(t *testing.T) {
	tempDir := t.TempDir()
	exePath := filepath.Join(tempDir, "release", "cliai.exe")
	scriptPath := filepath.Join(tempDir, "scripts", "install-powershell.ps1")

	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("prepare scripts dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(exePath), 0o755); err != nil {
		t.Fatalf("prepare exe dir: %v", err)
	}
	if err := os.WriteFile(exePath, []byte("stub"), 0o644); err != nil {
		t.Fatalf("write exe stub: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("# stub"), 0o644); err != nil {
		t.Fatalf("write script stub: %v", err)
	}

	got := findInstallPowerShellScript(exePath)
	if got != scriptPath {
		t.Fatalf("expected %q, got %q", scriptPath, got)
	}
}

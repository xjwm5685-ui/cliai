package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestInstallPowerShellHelpersWritesMarkedBlock(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "Documents", "PowerShell", "Profile.ps1")

	if err := installPowerShellHelpers(profilePath); err != nil {
		t.Fatalf("installPowerShellHelpers returned error: %v", err)
	}
	if err := installPowerShellHelpers(profilePath); err != nil {
		t.Fatalf("installPowerShellHelpers second run returned error: %v", err)
	}

	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	content := string(data)
	if strings.Count(content, powerShellHelpersStartMarker) != 1 {
		t.Fatalf("expected one helper block marker, got %q", content)
	}
	if !strings.Contains(content, "Set-Alias -Name csg") || !strings.Contains(content, "Set-Alias -Name csi") || !strings.Contains(content, "Set-Alias -Name csc") {
		t.Fatalf("expected helper aliases in profile, got %q", content)
	}
	if !strings.Contains(content, "Get-CliaiExecutable") || !strings.Contains(content, "--cwd $PWD.Path") {
		t.Fatalf("expected cwd-aware helper implementation in profile, got %q", content)
	}
}

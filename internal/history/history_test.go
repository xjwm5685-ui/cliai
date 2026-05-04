package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportPowerShellSkipsSensitiveCommands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ConsoleHost_history.txt")
	content := "git status\nAuthorization: Bearer secret\nwinget install vscode\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	entries, err := ImportPowerShell(path, 10)
	if err != nil {
		t.Fatalf("ImportPowerShell returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 sanitized entries, got %d", len(entries))
	}
}

func TestImportPowerShellSkipsCommentLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ConsoleHost_history.txt")
	content := "# comment\n// note\ngo test ./...\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	entries, err := ImportPowerShell(path, 10)
	if err != nil {
		t.Fatalf("ImportPowerShell returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Command != "go test ./..." {
		t.Fatalf("unexpected sanitized entries: %#v", entries)
	}
}

func TestImportZshParsesExtendedHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".zsh_history")
	content := ": 1715000000:0;git status\n: 1715000001:0;npm test\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	entries, err := Import(path, "zsh", 10)
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %#v", entries)
	}
}

func TestImportFishParsesCmdEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fish_history")
	content := "- cmd: git status\n  when: 1715000000\n- cmd: go test ./...\n  when: 1715000001\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	entries, err := Import(path, "fish", 10)
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %#v", entries)
	}
	if entries[0].Source != "fish-history" {
		t.Fatalf("expected fish-history source, got %#v", entries)
	}
}

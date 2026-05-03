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

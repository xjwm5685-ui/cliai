package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeCommandRejectsSensitiveContent(t *testing.T) {
	tests := []string{
		`Authorization: Bearer secret-token`,
		`export TOKEN=secret-value`,
		`setx API_KEY abc123`,
		`cat ~/.ssh/id_rsa`,
	}

	for _, command := range tests {
		if got, ok := sanitizeCommand(command); ok {
			t.Fatalf("expected %q to be rejected, got %q", command, got)
		}
	}
}

func TestSanitizeCommandRejectsConcatenatedPredictionNoise(t *testing.T) {
	tests := []string{
		`cliai predict git statuscliai predict go test ./...`,
		`csg git status csg go test ./... csg`,
	}

	for _, command := range tests {
		if got, ok := sanitizeCommand(command); ok {
			t.Fatalf("expected %q to be rejected, got %q", command, got)
		}
	}
}

func TestSanitizeRejectsBareURL(t *testing.T) {
	if got, ok := sanitizeCommand(`https://github.com/foo/bar.git`); ok {
		t.Fatalf("expected bare URL to be rejected, got %q", got)
	}
}

func TestSanitizeRejectsConcatenatedCommands(t *testing.T) {
	tests := []string{
		`winget install starshipwinget install starship`,
		`npm installnpm install`,
	}
	for _, command := range tests {
		if got, ok := sanitizeCommand(command); ok {
			t.Fatalf("expected %q to be rejected, got %q", command, got)
		}
	}
}

func TestSanitizeRejectsOutputTextButKeepsURLArguments(t *testing.T) {
	rejects := []string{
		`✅ 安装完成！请重新打开 PowerShell`,
		`打开：系统属性 -> 高级系统设置`,
	}
	for _, command := range rejects {
		if got, ok := sanitizeCommand(command); ok {
			t.Fatalf("expected output text %q to be rejected, got %q", command, got)
		}
	}

	keeps := []string{
		`git clone https://github.com/foo/bar.git`,
		`curl https://example.com/install.sh`,
	}
	for _, command := range keeps {
		got, ok := sanitizeCommand(command)
		if !ok || got != command {
			t.Fatalf("expected command %q to be kept, got %q, ok=%v", command, got, ok)
		}
	}
}

func TestSanitizeCommandKeepsNormalCompoundCommand(t *testing.T) {
	tests := []string{
		`cd src && go test ./...`,
		`git status; go test ./...`,
		`cat README.md | Select-String TODO`,
	}

	for _, command := range tests {
		got, ok := sanitizeCommand(command)
		if !ok || got != command {
			t.Fatalf("expected %q to be kept, got %q, ok=%v", command, got, ok)
		}
	}
}

func TestSanitizeCommandRejectsOverlyComplexHistory(t *testing.T) {
	command := `git status && go test ./... && go vet ./... && golangci-lint run && go build ./... && go run .`
	if got, ok := sanitizeCommand(command); ok {
		t.Fatalf("expected overly complex command to be rejected, got %q", got)
	}
}

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

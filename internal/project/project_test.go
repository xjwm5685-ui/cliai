package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRecognizesGoGitAndDocker(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module test")
	mustWrite(t, filepath.Join(dir, "docker-compose.yml"), "services: {}")
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	ctx, err := Detect(dir)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if !contains(ctx.ProjectTypes, "go") || !contains(ctx.ProjectTypes, "docker") || !contains(ctx.ProjectTypes, "git") {
		t.Fatalf("unexpected project types: %#v", ctx.ProjectTypes)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

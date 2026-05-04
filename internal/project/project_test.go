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

func TestDetectWalksUpToParentProjectMarkers(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "cmd", "tool")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	mustWrite(t, filepath.Join(root, "go.mod"), "module test")
	mustWrite(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '9'")
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	mustWrite(t, filepath.Join(child, "main.go"), "package main")

	ctx, err := Detect(child)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if !contains(ctx.ProjectTypes, "go") || !contains(ctx.ProjectTypes, "node") || !contains(ctx.ProjectTypes, "git") {
		t.Fatalf("expected parent project markers to be detected, got %#v", ctx.ProjectTypes)
	}
	if ctx.PackageManager != "pnpm" {
		t.Fatalf("expected pnpm package manager from parent markers, got %q", ctx.PackageManager)
	}
	if !contains(ctx.Files, "main.go") {
		t.Fatalf("expected current directory files to be preserved, got %#v", ctx.Files)
	}
}

func mustWrite(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

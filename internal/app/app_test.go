package app

import (
	"bytes"
	"strings"
	"reflect"
	"testing"
)

func TestNormalizePredictArgsSupportsLeadingAndTrailingFlags(t *testing.T) {
	got := normalizePredictArgs([]string{"--json", "安装", "vscode", "--limit", "3"})
	want := []string{"--json", "--limit", "3", "安装", "vscode"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestNormalizePredictArgsKeepsFlagLikeQueryTokensInMiddle(t *testing.T) {
	got := normalizePredictArgs([]string{"搜索", "--json", "文件"})
	want := []string{"搜索", "--json", "文件"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestRunVersionIncludesBuildMetadata(t *testing.T) {
	oldVersion, oldCommit, oldBuildDate := Version, Commit, BuildDate
	Version, Commit, BuildDate = "1.2.3", "abc123", "2026-05-03T10:00:00"
	defer func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldBuildDate
	}()

	var stdout bytes.Buffer
	code := Run([]string{"version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "1.2.3") || !strings.Contains(output, "abc123") {
		t.Fatalf("unexpected version output: %q", output)
	}
}

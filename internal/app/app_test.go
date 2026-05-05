package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sanqiu/cliai/internal/predict"
)

type stubPredictorQueryService struct {
	suggestions []predict.Candidate
	err         error
}

func (s stubPredictorQueryService) Query(request predictorServiceRequest) ([]predict.Candidate, error) {
	return s.suggestions, s.err
}

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

func TestClipboardCommandSelectsMacOSClipboardTool(t *testing.T) {
	name, args, err := clipboardCommand("darwin", func(file string) (string, error) {
		if file == "pbcopy" {
			return "/usr/bin/pbcopy", nil
		}
		return "", errors.New("not found")
	})
	if err != nil {
		t.Fatalf("clipboardCommand returned error: %v", err)
	}
	if name != "pbcopy" {
		t.Fatalf("expected pbcopy, got %q", name)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %#v", args)
	}
}

func TestClipboardCommandSelectsLinuxClipboardTool(t *testing.T) {
	name, args, err := clipboardCommand("linux", func(file string) (string, error) {
		if file == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	})
	if err != nil {
		t.Fatalf("clipboardCommand returned error: %v", err)
	}
	if name != "xclip" {
		t.Fatalf("expected xclip, got %q", name)
	}
	wantArgs := []string{"-selection", "clipboard"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args\nwant: %#v\ngot:  %#v", wantArgs, args)
	}
}

func TestClipboardCommandReturnsHelpfulErrorWhenToolMissing(t *testing.T) {
	_, _, err := clipboardCommand("linux", func(file string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil {
		t.Fatalf("expected error when no clipboard tool is available")
	}
	if !strings.Contains(err.Error(), "wl-copy") {
		t.Fatalf("expected helpful install hint, got %q", err.Error())
	}
}

func TestPrintHelpUsesLocalFirstPositioning(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "local-first command prediction and completion CLI") {
		t.Fatalf("expected local-first help text, got %q", output)
	}
	if !strings.Contains(output, "history import           Import shell history into the local cache") {
		t.Fatalf("expected cross-shell history help text, got %q", output)
	}
	if strings.Contains(output, "--no-cloud") {
		t.Fatalf("did not expect removed --no-cloud flag in help output: %q", output)
	}
	if !strings.Contains(output, "shell init <shell>") || !strings.Contains(output, "powershell, bash, or zsh") {
		t.Fatalf("expected shell help text for powershell, bash, and zsh, got %q", output)
	}
	if !strings.Contains(output, "shell install powershell-helpers") {
		t.Fatalf("expected explicit powershell helper install help text, got %q", output)
	}
}

func TestRunPredictorUsageDoesNotMentionRemovedNoCloudFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := runPredictor([]string{}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "usage: cliai predictor serve [--limit 8] [--shell powershell]") {
		t.Fatalf("unexpected usage output: %q", output)
	}
	if strings.Contains(output, "--no-cloud") {
		t.Fatalf("did not expect removed --no-cloud flag in predictor usage: %q", output)
	}
}

func TestRunShellInitPowerShellHelpersPrintsAliasSnippet(t *testing.T) {
	var stdout bytes.Buffer
	code := runShell([]string{"init", "powershell-helpers"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	output := stdout.String()
	if !strings.Contains(output, "Set-Alias -Name csg") || !strings.Contains(output, "Set-Alias -Name csi") || !strings.Contains(output, "Set-Alias -Name csc") {
		t.Fatalf("expected helper aliases in snippet, got %q", output)
	}
	if !strings.Contains(output, "Get-CliaiExecutable") || !strings.Contains(output, "--cwd $PWD.Path") {
		t.Fatalf("expected safer helper snippet with cwd-aware execution, got %q", output)
	}
	if strings.Contains(output, "Write-Host \"cliai helper loaded") {
		t.Fatalf("did not expect noisy profile Write-Host in snippet, got %q", output)
	}
}

func TestRunPredictorStreamReturnsJSONErrorForInvalidRequest(t *testing.T) {
	var stdout bytes.Buffer
	code := runPredictorStream(stubPredictorQueryService{}, strings.NewReader("{broken-json\n"), &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	var response predictorServiceResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("expected JSON response, got error: %v, raw=%q", err, stdout.String())
	}
	if !strings.Contains(response.Error, "decode request") {
		t.Fatalf("expected decode error response, got %#v", response)
	}
}

func TestPrintDebugIncludesRejectedCandidates(t *testing.T) {
	var stderr bytes.Buffer
	printDebug(&stderr, debugInfo{
		Query:          "git st",
		Shell:          "powershell",
		CWD:            "D:\\sanqiu\\cliai",
		HistoryEntries: 10,
		FeedbackCount:  2,
	}, []predict.Candidate{
		{
			Command: "git status",
			Source:  "history",
			Score:   120,
			Risk:    "safe",
			Reason:  "history hit; matched test intent",
			Details: []string{"matches git st prefix"},
		},
	}, predict.DebugReport{
		Rejected: []predict.RejectedCandidate{
			{
				Command: "git clone https://example.com/repo.git",
				Source:  "history",
				Score:   12,
				Reason:  "rejected by gate: subcommand mismatch for explicit command prefix",
			},
		},
	})

	output := stderr.String()
	if !strings.Contains(output, "debug candidate[0]") || !strings.Contains(output, "details=") {
		t.Fatalf("expected debug output to include candidate details, got %q", output)
	}
	if !strings.Contains(output, "debug rejected[0]") || !strings.Contains(output, "subcommand mismatch") {
		t.Fatalf("expected debug output to include rejected candidate reason, got %q", output)
	}
}

func TestPrintDebugTruncatesRejectedCandidates(t *testing.T) {
	var rejected []predict.RejectedCandidate
	for i := 0; i < maxDebugRejectedCandidates+3; i++ {
		rejected = append(rejected, predict.RejectedCandidate{
			Command: "cmd",
			Source:  "history",
			Reason:  "rejected by gate",
		})
	}

	var stderr bytes.Buffer
	printDebug(&stderr, debugInfo{}, nil, predict.DebugReport{Rejected: rejected})
	output := stderr.String()
	if !strings.Contains(output, "debug rejected_truncated=3") {
		t.Fatalf("expected truncated rejected summary, got %q", output)
	}
}

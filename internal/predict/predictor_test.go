package predict

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sanqiu/cliai/internal/history"
	"github.com/sanqiu/cliai/internal/project"
)

func TestPredictPrioritizesHistoryPrefix(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "git st",
		Limit: 3,
	}, []history.Entry{
		{Command: "git status", Count: 8, LastUsed: time.Now(), Source: "test"},
		{Command: "git stash", Count: 1, LastUsed: time.Now().Add(-48 * time.Hour), Source: "test"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "git status" {
		t.Fatalf("expected git status first, got %q", results[0].Command)
	}
}

func TestPredictGitStatusDoesNotIncludeCloneHistoryInTopResults(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "git st",
		Limit: 5,
	}, []history.Entry{
		{Command: "git status", Count: 10, LastUsed: time.Now(), Source: "history"},
		{Command: "git clone https://example.com/repo-a.git", Count: 500, LastUsed: time.Now(), Source: "history"},
		{Command: "git clone https://example.com/repo-b.git", Count: 400, LastUsed: time.Now(), Source: "history"},
		{Command: "git clone https://example.com/repo-c.git", Count: 300, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "git status" {
		t.Fatalf("expected git status first, got %#v", results)
	}
	for _, item := range results {
		if strings.HasPrefix(item.Command, "git clone ") {
			t.Fatalf("did not expect git clone in top results, got %#v", results)
		}
	}
}

func TestPredictGitStatusStrongFamilyGate(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "git st",
		Limit: 5,
	}, []history.Entry{
		{Command: "git status", Count: 10, LastUsed: time.Now(), Source: "history"},
		{Command: "git clone https://github.com/foo/bar.git", Count: 500, LastUsed: time.Now(), Source: "history"},
		{Command: "go install github.com/wailsapp/wails/v2/cmd/wails@latest", Count: 450, LastUsed: time.Now(), Source: "history"},
		{Command: "winget install starshipwinget install starship", Count: 400, LastUsed: time.Now(), Source: "history"},
		{Command: "https://github.com/foo/bar.git", Count: 350, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "git status" {
		t.Fatalf("expected git status first, got %#v", results)
	}
	for _, item := range results {
		if strings.HasPrefix(item.Command, "git clone ") ||
			strings.HasPrefix(item.Command, "go install ") ||
			strings.HasPrefix(item.Command, "winget install ") ||
			strings.HasPrefix(item.Command, "https://") {
			t.Fatalf("did not expect unrelated command in top results, got %#v", results)
		}
	}
}

func TestPredictGitCloneAllowsCloneHistory(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "git cl",
		Limit: 5,
	}, []history.Entry{
		{Command: "git clone https://example.com/repo.git", Count: 50, LastUsed: time.Now(), Source: "history"},
	})

	for _, item := range results {
		if strings.HasPrefix(item.Command, "git clone https://example.com/repo.git") {
			return
		}
	}
	t.Fatalf("expected git clone history to appear, got %#v", results)
}

func TestPredictCommandSubverbPrefixForDocker(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "docker ps",
		Limit: 5,
	}, []history.Entry{
		{Command: "docker compose up -d", Count: 800, LastUsed: time.Now(), Source: "history"},
		{Command: "docker ps", Count: 10, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "docker ps" {
		t.Fatalf("expected docker ps first, got %#v", results)
	}
	if len(results) > 1 && results[1].Command == "docker compose up -d" {
		t.Fatalf("did not expect docker compose up to outrank docker ps, got %#v", results)
	}
}

func TestPredictDockerPsFamilyGate(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "docker ps",
		Limit: 5,
	}, []history.Entry{
		{Command: "docker ps", Count: 15, LastUsed: time.Now(), Source: "history"},
		{Command: "docker compose up -d", Count: 900, LastUsed: time.Now(), Source: "history"},
		{Command: "go install github.com/foo/bar@latest", Count: 700, LastUsed: time.Now(), Source: "history"},
		{Command: "https://github.com/foo/bar.git", Count: 600, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "docker ps" {
		t.Fatalf("expected docker ps first, got %#v", results)
	}
	for _, item := range results {
		if strings.HasPrefix(item.Command, "go install ") || strings.HasPrefix(item.Command, "https://") {
			t.Fatalf("did not expect non-docker garbage in top results, got %#v", results)
		}
	}
}

func TestPredictSupportsNaturalLanguageInstall(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "安装 vscode",
		Shell: "powershell",
		Limit: 5,
	}, nil)

	found := map[string]bool{}
	for _, item := range results {
		found[item.Command] = true
	}

	expectedSearch, expectedInstall, _, _, _ := packageManagerCommands("vscode")
	if !found[expectedInstall] {
		t.Fatalf("expected install template %q, got %#v", expectedInstall, results)
	}
	if !found[expectedSearch] {
		t.Fatalf("expected search template %q, got %#v", expectedSearch, results)
	}

	if runtime.GOOS == "windows" && !found["winget install vscode"] {
		t.Fatalf("expected windows install template, got %#v", results)
	}
}

func TestPredictUsesShellToFilterPowerShellSpecificCommands(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "查看文件",
		Shell: "cmd",
		Limit: 5,
	}, nil)

	for _, item := range results {
		if item.Command == "Get-ChildItem" {
			t.Fatalf("did not expect powershell-specific command for cmd shell")
		}
	}
}

func TestPredictUsesProjectContextForNodeDev(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "启动",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes:   []string{"node"},
			PackageManager: "pnpm",
		},
	}, nil)

	found := false
	for _, item := range results {
		if item.Command == "pnpm dev" && item.Source == "context" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected context-aware pnpm dev suggestion, got %#v", results)
	}
}

func TestPredictUsesProjectContextForGoRun(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "启动",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
		},
	}, nil)

	for _, item := range results {
		if item.Command == "go run ." && item.Source == "context" {
			return
		}
	}
	t.Fatalf("expected context-aware go run suggestion, got %#v", results)
}

func TestPredictUsesProjectContextForChangeDirectory(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "进入 internal",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			Directories: []string{"internal"},
		},
	}, nil)

	for _, item := range results {
		if item.Command == `Set-Location .\internal` && item.Source == "context" {
			return
		}
	}
	t.Fatalf("expected context-aware directory suggestion, got %#v", results)
}

func TestPredictUsesProjectContextForReadFile(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "打开 README",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			Files: []string{"README.md"},
		},
	}, nil)

	for _, item := range results {
		if item.Command == `Get-Content .\README.md` && item.Source == "context" {
			return
		}
	}
	t.Fatalf("expected context-aware file read suggestion, got %#v", results)
}

func TestPredictReasonExplainsGoProjectContext(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "run tests",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
		},
	}, nil)

	for _, item := range results {
		if item.Command == "go test ./..." && item.Source == "context" {
			if !strings.Contains(item.Reason, "matched test intent") || !strings.Contains(item.Reason, "Go project") {
				t.Fatalf("expected richer reason for go test candidate, got %#v", item)
			}
			if len(item.Details) == 0 {
				t.Fatalf("expected details for go test candidate, got %#v", item)
			}
			return
		}
	}
	t.Fatalf("expected context-aware go test suggestion, got %#v", results)
}

func TestRiskLevelMarksInstallCommandsAsCaution(t *testing.T) {
	if got := riskLevel("winget install vscode"); got != "caution" {
		t.Fatalf("expected install command to be caution, got %q", got)
	}
	if got := riskLevel("pip install -r requirements.txt"); got != "caution" {
		t.Fatalf("expected pip install command to be caution, got %q", got)
	}
}

func TestRiskLevelMarksServiceStartCommandsAsCaution(t *testing.T) {
	for _, command := range []string{"pnpm dev", "npm run dev", "go run .", "docker compose up -d"} {
		if got := riskLevel(command); got != "caution" {
			t.Fatalf("expected %q to be caution, got %q", command, got)
		}
	}
}

func TestRiskLevelMarksPowerShellDeleteCommandsAsDanger(t *testing.T) {
	command := `Get-Process any-api -ErrorAction SilentlyContinue | Stop-Process -Force; Remove-Item -Path ".\data.db" -Force`
	if got := riskLevel(command); got != "danger" {
		t.Fatalf("expected PowerShell delete command to be danger, got %q", got)
	}
}

func TestRiskLevelMarksAdditionalDangerousCommands(t *testing.T) {
	tests := []string{
		`rm -rf ./tmp`,
		`rm -rf /`,
		`Remove-Item -Path .\build -Recurse -Force`,
		`Remove-Item C:\ -Recurse -Force`,
		`del /s /q build`,
		`curl https://example.com/install.sh | sh`,
		`curl https://example.com/install.sh | bash`,
		`irm https://example.com/install.ps1 | iex`,
		`Invoke-WebRequest https://example.com/install.ps1 | Invoke-Expression`,
		`git reset --hard HEAD~1`,
		`git clean -fd`,
		`docker compose down -v`,
		`echo hacked > /etc/hosts`,
		`echo $TOKEN`,
	}

	for _, command := range tests {
		if got := riskLevel(command); got != "danger" {
			t.Fatalf("expected %q to be danger, got %q", command, got)
		}
	}
}

func TestRiskLevelMarksElevationCommandsAsCaution(t *testing.T) {
	tests := []string{
		`sudo apt install ripgrep`,
		`Start-Process powershell -Verb RunAs`,
		`docker system prune`,
		`chmod -R 777 .`,
		`git push --force`,
		`kubectl delete pod nginx`,
		`docker rm -f my-container`,
	}

	for _, command := range tests {
		if got := riskLevel(command); got != "caution" {
			t.Fatalf("expected %q to be caution, got %q", command, got)
		}
	}
}

func TestPredictRunTestsPrefersGoContextOverNoisyHistory(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "run tests",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
		},
	}, []history.Entry{
		{Command: "npm run dev -- --host 0.0.0.0 && npm run lint && npm run test:e2e", Count: 500, LastUsed: time.Now(), Source: "history"},
		{Command: "npm run dev", Count: 1000, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "go test ./..." {
		t.Fatalf("expected go test to rank first, got %#v", results)
	}
}

func TestPredictGoTestFamilyGate(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "go test",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
		},
	}, []history.Entry{
		{Command: "npm run dev", Count: 1000, LastUsed: time.Now(), Source: "history"},
		{Command: "winget install vscode", Count: 900, LastUsed: time.Now(), Source: "history"},
		{Command: "opencode", Count: 800, LastUsed: time.Now(), Source: "history"},
		{Command: "codex", Count: 700, LastUsed: time.Now(), Source: "history"},
	})

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != "go test ./..." {
		t.Fatalf("expected go test first, got %#v", results)
	}
	for _, item := range results {
		if item.Command == "npm run dev" || item.Command == "winget install vscode" || item.Command == "opencode" || item.Command == "codex" {
			t.Fatalf("did not expect unrelated history in top results, got %#v", results)
		}
	}
}

func TestPredictGoTestPrefersContextSourceOverHistoryForSameCommand(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "go test",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
		},
	}, []history.Entry{
		{Command: "go test ./...", Count: 50, LastUsed: time.Now(), Source: "powershell-history"},
	})

	for _, item := range results {
		if item.Command == "go test ./..." {
			if item.Source != "context" {
				t.Fatalf("expected context source to win for same command, got %#v", item)
			}
			if !strings.Contains(item.Reason, "Go project") {
				t.Fatalf("expected context-oriented reason, got %#v", item)
			}
			if !containsString(item.Details, "also seen in local history") {
				t.Fatalf("expected merged history detail, got %#v", item)
			}
			return
		}
	}
	t.Fatalf("expected go test candidate, got %#v", results)
}

func TestPredictSearchTODOPrefersFileSearchInDeveloperContext(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "查找 TODO",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
			Directories:  []string{"internal"},
		},
	}, nil)

	if len(results) == 0 {
		t.Fatalf("expected predictions")
	}
	if results[0].Command != `Select-String -Path . -Pattern "TODO"` {
		t.Fatalf("expected file search first, got %#v", results)
	}
}

func TestPredictIrrelevantInteractiveHistoryDoesNotCrowdTopResults(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "查找 TODO",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
			Directories:  []string{"internal"},
		},
	}, []history.Entry{
		{Command: "opencode", Count: 900, LastUsed: time.Now(), Source: "history"},
		{Command: "claude", Count: 800, LastUsed: time.Now(), Source: "history"},
		{Command: "codex", Count: 700, LastUsed: time.Now(), Source: "history"},
	})

	for _, item := range results {
		if item.Command == "opencode" || item.Command == "claude" || item.Command == "codex" {
			t.Fatalf("did not expect unrelated interactive tool in top results, got %#v", results)
		}
	}
}

func TestPredictSearchTODOPushesIrrelevantHistoryFurtherDown(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "查找 TODO",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			ProjectTypes: []string{"go"},
			Directories:  []string{"internal"},
		},
	}, []history.Entry{
		{Command: "pip install pycryptodome", Count: 500, LastUsed: time.Now(), Source: "history"},
		{Command: "go run .", Count: 400, LastUsed: time.Now(), Source: "history"},
	})

	for _, item := range results {
		if item.Command == "pip install pycryptodome" || item.Command == "go run ." {
			t.Fatalf("did not expect irrelevant history in top results, got %#v", results)
		}
	}
}

func TestPredictOpenReadmePushesWriteHostHistoryFurtherDown(t *testing.T) {
	engine := New()
	results := engine.Predict(Request{
		Query: "打开 README",
		Shell: "powershell",
		Limit: 5,
		Project: project.Context{
			Files: []string{"README.md"},
		},
	}, []history.Entry{
		{Command: `Write-Host "hello"`, Count: 999, LastUsed: time.Now(), Source: "history"},
	})

	for _, item := range results {
		if item.Command == `Write-Host "hello"` {
			t.Fatalf("did not expect unrelated Write-Host history in top results, got %#v", results)
		}
	}
}

func TestExtractLastMeaningfulTokenSupportsChinesePhrases(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{query: "安装一下 vscode", want: "vscode"},
		{query: "帮我进入 src 目录", want: "src"},
		{query: "查找 TODO", want: "TODO"},
		{query: `帮我进入 "internal/app" 目录`, want: "internal/app"},
	}

	for _, test := range tests {
		if got := extractLastMeaningfulToken(test.query); got != test.want {
			t.Fatalf("extractLastMeaningfulToken(%q) = %q, want %q", test.query, got, test.want)
		}
	}
}

func BenchmarkPredict(b *testing.B) {
	engine := New()
	entries := []history.Entry{
		{Command: "git status", Count: 20, LastUsed: time.Now(), Source: "history"},
		{Command: "go test ./...", Count: 10, LastUsed: time.Now(), Source: "history"},
		{Command: "pnpm dev", Count: 8, LastUsed: time.Now(), Source: "history"},
	}

	req := Request{
		Query: "run tests",
		Shell: "powershell",
		Project: project.Context{
			ProjectTypes: []string{"go", "git"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Predict(req, entries)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

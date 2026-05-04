package predict

import (
	"runtime"
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

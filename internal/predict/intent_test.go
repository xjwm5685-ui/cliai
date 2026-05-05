package predict

import (
	"testing"

	"github.com/sanqiu/cliai/internal/project"
)

func TestClassifyIntent(t *testing.T) {
	ctx := project.Context{
		ProjectTypes: []string{"go"},
		Files:        []string{"README.md", "main.go"},
		Directories:  []string{"internal"},
	}

	tests := []struct {
		query string
		kind  IntentKind
		want  string
	}{
		{query: "查找 TODO", kind: IntentSearchFiles, want: "TODO"},
		{query: "搜一下 main.go", kind: IntentSearchFiles, want: "main.go"},
		{query: "安装 vscode", kind: IntentInstallPackage, want: "vscode"},
		{query: "进入 internal", kind: IntentChangeDirectory, want: "internal"},
		{query: "打开 README", kind: IntentReadFile, want: "README"},
		{query: "跑测试", kind: IntentRunTests},
		{query: "启动项目", kind: IntentStartProject},
	}

	for _, test := range tests {
		intent := classifyIntent(test.query, ctx)
		if intent.Kind != test.kind {
			t.Fatalf("classifyIntent(%q) kind = %q, want %q", test.query, intent.Kind, test.kind)
		}
		if test.want != "" && intent.Target != test.want {
			t.Fatalf("classifyIntent(%q) target = %q, want %q", test.query, intent.Target, test.want)
		}
	}
}

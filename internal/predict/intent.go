package predict

import (
	"strings"

	"github.com/sanqiu/cliai/internal/project"
)

type IntentKind string

const (
	IntentUnknown          IntentKind = "unknown"
	IntentSearchFiles      IntentKind = "search_files"
	IntentSearchPackages   IntentKind = "search_packages"
	IntentInstallPackage   IntentKind = "install_package"
	IntentUninstallPackage IntentKind = "uninstall_package"
	IntentUpgradePackage   IntentKind = "upgrade_package"
	IntentRunTests         IntentKind = "run_tests"
	IntentChangeDirectory  IntentKind = "change_directory"
	IntentReadFile         IntentKind = "read_file"
	IntentOpenEditor       IntentKind = "open_editor"
	IntentStartProject     IntentKind = "start_project"
	IntentListFiles        IntentKind = "list_files"
)

type Intent struct {
	Kind   IntentKind
	Query  string
	Target string
}

func classifyIntent(query string, ctx project.Context) Intent {
	intent := Intent{
		Kind:   IntentUnknown,
		Query:  strings.TrimSpace(query),
		Target: extractLastMeaningfulToken(query),
	}
	if intent.Query == "" {
		return intent
	}

	norm := normalize(intent.Query)
	switch {
	case isOpenEditorIntent(norm):
		intent.Kind = IntentOpenEditor
	case containsAny(norm, "安装", "install", "setup"):
		intent.Kind = IntentInstallPackage
	case containsAny(norm, "卸载", "uninstall", "remove"):
		intent.Kind = IntentUninstallPackage
	case containsAny(norm, "升级", "更新", "upgrade", "update"):
		intent.Kind = IntentUpgradePackage
	case containsAny(norm, "跑测试", "测试", "run tests", "test"):
		intent.Kind = IntentRunTests
	case containsAny(norm, "进入", "切换目录", "go to", "cd"):
		intent.Kind = IntentChangeDirectory
	case containsAny(norm, "列出", "查看文件", "list files", "show files"):
		intent.Kind = IntentListFiles
	case isReadFileIntent(norm, intent.Target, ctx):
		intent.Kind = IntentReadFile
	case containsAny(norm, "搜索", "查找", "搜一下", "search", "find"):
		if prefersFileSearch(intent.Query, intent.Target, ctx) {
			intent.Kind = IntentSearchFiles
		} else {
			intent.Kind = IntentSearchPackages
		}
	case containsAny(norm, "启动项目", "启动", "run project", "run app", "start", "run dev", "dev"):
		intent.Kind = IntentStartProject
	}
	return intent
}

func isOpenEditorIntent(norm string) bool {
	if containsAny(norm, "code .", "open editor", "打开编辑器") {
		return true
	}
	return containsAny(norm, "打开", "open") && containsAny(norm, "vscode", "vs code", "editor")
}

func isReadFileIntent(norm string, target string, ctx project.Context) bool {
	if containsAny(norm, "read file", "cat", "读取", "打开文件") {
		return true
	}
	if !containsAny(norm, "打开", "read", "show") {
		return false
	}
	if looksLikeFileTarget(target) {
		return true
	}
	return project.MatchFile(ctx, target) != ""
}

func looksLikeFileTarget(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	if strings.Contains(target, ".") || strings.ContainsAny(target, `/\`) {
		return true
	}
	lower := strings.ToLower(target)
	for _, suffix := range []string{"md", "txt", "json", "yaml", "yml", "go", "ts", "js", "ps1", "sh"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

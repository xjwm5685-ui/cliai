package predict

import (
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/sanqiu/cliai/internal/history"
	"github.com/sanqiu/cliai/internal/project"
)

type Candidate struct {
	Command string  `json:"command"`
	Reason  string  `json:"reason"`
	Source  string  `json:"source"`
	Score   float64 `json:"score"`
	Risk    string  `json:"risk,omitempty"`
}

type Request struct {
	Query           string
	CWD             string
	Shell           string
	Limit           int
	Project         project.Context
	FeedbackBonuses map[string]float64
}

type Predictor struct {
	builtins []CommandSpec
}

func New() *Predictor {
	return &Predictor{builtins: Builtins()}
}

func (p *Predictor) Predict(req Request, entries []history.Entry) []Candidate {
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	query := strings.TrimSpace(req.Query)
	shell := normalizeShell(req.Shell)
	candidates := map[string]Candidate{}

	add := func(candidate Candidate) {
		existing, ok := candidates[candidate.Command]
		if !ok || candidate.Score > existing.Score {
			candidates[candidate.Command] = candidate
		}
	}

	for _, generated := range generateIntentCandidates(query, shell, req.Project) {
		add(generated)
	}
	for _, generated := range generateProjectCandidates(query, shell, req.Project) {
		add(generated)
	}

	for _, spec := range p.builtins {
		if !isShellCompatible(spec.Command, shell) {
			continue
		}
		score := scoreCandidate(query, spec.Command, spec.Keywords, 1, 0, projectTypeBonus(spec.ProjectTypes, req.Project.ProjectTypes))
		score += req.FeedbackBonuses[spec.Command]
		if score <= 0 {
			continue
		}
		add(Candidate{
			Command: spec.Command,
			Reason:  spec.Description,
			Source:  "builtin",
			Score:   score,
			Risk:    riskLevel(spec.Command),
		})
	}

	for _, entry := range entries {
		score := scoreCandidate(query, entry.Command, nil, entry.Count, recencyScore(entry), projectCommandBonus(entry.Command, req.Project.ProjectTypes))
		score += req.FeedbackBonuses[entry.Command]
		if score <= 0 {
			continue
		}
		add(Candidate{
			Command: entry.Command,
			Reason:  fmt.Sprintf("history hit, used %d times", entry.Count),
			Source:  entry.Source,
			Score:   score,
			Risk:    riskLevel(entry.Command),
		})
	}

	result := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return len(result[i].Command) < len(result[j].Command)
		}
		return result[i].Score > result[j].Score
	})

	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func scoreCandidate(query string, command string, keywords []string, frequency int, recency float64, contextBonus float64) float64 {
	normQuery := normalize(query)
	normCommand := normalize(command)

	if normQuery == "" {
		return math.Log(float64(max(1, frequency))+1)*5 + recency
	}

	queryTokens := tokens(normQuery)
	commandTokens := tokens(normCommand)

	var score float64
	if strings.HasPrefix(normCommand, normQuery) {
		score += 80
	}
	if strings.Contains(normCommand, normQuery) {
		score += 20
	}

	for _, qt := range queryTokens {
		tokenMatched := false
		for _, ct := range commandTokens {
			switch {
			case qt == ct:
				score += 18
				tokenMatched = true
			case strings.HasPrefix(ct, qt):
				score += 12
				tokenMatched = true
			case strings.Contains(ct, qt):
				score += 6
				tokenMatched = true
			}
		}
		if !tokenMatched {
			for _, kw := range keywords {
				normKeyword := normalize(kw)
				if strings.Contains(normKeyword, qt) || strings.Contains(qt, normKeyword) {
					score += 10
					tokenMatched = true
					break
				}
			}
		}
		if !tokenMatched {
			score -= 4
		}
	}

	for _, kw := range keywords {
		normKeyword := normalize(kw)
		if normKeyword != "" && strings.Contains(normQuery, normKeyword) {
			score += 10
		}
	}

	score += math.Log(float64(max(1, frequency))+1) * 7
	score += recency
	score += contextBonus

	if looksNaturalLanguage(query) && strings.Contains(command, "--id") {
		score -= 8
	}

	return score
}

func recencyScore(entry history.Entry) float64 {
	if entry.LastUsed.IsZero() {
		return 0
	}
	age := maxFloat(1, time.Since(entry.LastUsed).Hours())
	if age <= 24 {
		return 12
	}
	if age <= 24*7 {
		return 8
	}
	if age <= 24*30 {
		return 4
	}
	return 1
}

func generateIntentCandidates(query string, shell string, ctx project.Context) []Candidate {
	norm := normalize(query)
	if norm == "" {
		return nil
	}

	var result []Candidate
	arg := extractLastMeaningfulToken(query)
	if arg == "" {
		arg = "PACKAGE_OR_TARGET"
	}

	if containsAny(norm, "安装", "install", "setup") {
		searchCmd, installCmd, _, _, _ := packageManagerCommands(arg)
		result = append(result,
			Candidate{Command: searchCmd, Reason: "search likely package first", Source: "template", Score: 120, Risk: riskLevel(searchCmd)},
			Candidate{Command: installCmd, Reason: "install package from natural-language intent", Source: "template", Score: 128, Risk: riskLevel(installCmd)},
		)
	}
	if containsAny(norm, "卸载", "uninstall", "remove") {
		_, _, _, uninstallCmd, _ := packageManagerCommands(arg)
		result = append(result, Candidate{Command: uninstallCmd, Reason: "remove package from natural-language intent", Source: "template", Score: 124, Risk: riskLevel(uninstallCmd)})
	}
	if containsAny(norm, "升级", "更新", "upgrade", "update") {
		_, _, upgradeAllCmd, _, upgradeCmd := packageManagerCommands(arg)
		if arg == "PACKAGE_OR_TARGET" {
			result = append(result, Candidate{Command: upgradeAllCmd, Reason: "upgrade all packages", Source: "template", Score: 118, Risk: riskLevel(upgradeAllCmd)})
		} else {
			result = append(result, Candidate{Command: upgradeCmd, Reason: "upgrade specific package", Source: "template", Score: 118, Risk: riskLevel(upgradeCmd)})
		}
	}
	if containsAny(norm, "搜索", "查找", "search", "find") {
		searchCmd, _, _, _, _ := packageManagerCommands(arg)
		result = append(result, Candidate{Command: searchCmd, Reason: "search package by keyword", Source: "template", Score: 110, Risk: riskLevel(searchCmd)})
		if shell == "powershell" {
			result = append(result, Candidate{Command: "Select-String -Path . -Pattern \"" + arg + "\"", Reason: "search inside files", Source: "template", Score: 102, Risk: riskLevel("Select-String")})
		} else {
			result = append(result, Candidate{Command: "grep -R \"" + arg + "\" .", Reason: "search inside files", Source: "template", Score: 102, Risk: riskLevel("grep -R")})
		}
	}
	if shell == "powershell" && containsAny(norm, "列出", "查看文件", "list files", "show files") {
		result = append(result, Candidate{Command: "Get-ChildItem", Reason: "list current directory", Source: "template", Score: 106, Risk: riskLevel("Get-ChildItem")})
	} else if shell != "powershell" && containsAny(norm, "列出", "查看文件", "list files", "show files") {
		result = append(result, Candidate{Command: "ls -la", Reason: "list current directory", Source: "template", Score: 106, Risk: riskLevel("ls -la")})
	}
	if shell == "powershell" && containsAny(norm, "进入", "切换目录", "go to", "cd") && arg != "PACKAGE_OR_TARGET" {
		matched := project.MatchDirectory(ctx, arg)
		target := arg
		if matched != "" {
			target = ".\\" + matched
		}
		result = append(result, Candidate{Command: "Set-Location " + target, Reason: "change directory", Source: "template", Score: 106, Risk: riskLevel("Set-Location " + target)})
	} else if shell != "powershell" && containsAny(norm, "进入", "切换目录", "go to", "cd") && arg != "PACKAGE_OR_TARGET" {
		matched := project.MatchDirectory(ctx, arg)
		target := arg
		if matched != "" {
			target = "./" + matched
		}
		result = append(result, Candidate{Command: "cd " + target, Reason: "change directory", Source: "template", Score: 106, Risk: riskLevel("cd " + target)})
	}
	if containsAny(norm, "测试", "run tests", "test") {
		result = append(result, Candidate{Command: "go test ./...", Reason: "run Go tests", Source: "template", Score: 104, Risk: riskLevel("go test ./...")})
	}
	if containsAny(norm, "打开", "open vscode", "open editor") {
		result = append(result, Candidate{Command: "code .", Reason: "open current folder in VS Code", Source: "template", Score: 100, Risk: riskLevel("code .")})
	}

	return result
}

func generateProjectCandidates(query string, shell string, ctx project.Context) []Candidate {
	if strings.TrimSpace(query) == "" {
		return nil
	}

	norm := normalize(query)
	arg := extractLastMeaningfulToken(query)
	if arg == "" {
		arg = "TARGET"
	}

	var result []Candidate

	if containsAny(norm, "测试", "test", "run tests") {
		switch {
		case contains(ctx.ProjectTypes, "go"):
			result = append(result, Candidate{Command: "go test ./...", Reason: "go project detected in current directory", Source: "context", Score: 132, Risk: riskLevel("go test ./...")})
		case contains(ctx.ProjectTypes, "node") && ctx.PackageManager == "pnpm":
			result = append(result, Candidate{Command: "pnpm test", Reason: "pnpm project detected in current directory", Source: "context", Score: 132, Risk: riskLevel("pnpm test")})
		case contains(ctx.ProjectTypes, "node"):
			result = append(result, Candidate{Command: "npm test", Reason: "node project detected in current directory", Source: "context", Score: 130, Risk: riskLevel("npm test")})
		}
	}

	if containsAny(norm, "启动", "dev", "run dev", "start") && contains(ctx.ProjectTypes, "node") {
		command := "npm run dev"
		if ctx.PackageManager == "pnpm" {
			command = "pnpm dev"
		}
		result = append(result, Candidate{Command: command, Reason: "frontend project detected in current directory", Source: "context", Score: 130, Risk: riskLevel(command)})
	}

	if containsAny(norm, "启动", "run", "start") && contains(ctx.ProjectTypes, "go") {
		result = append(result, Candidate{Command: "go run .", Reason: "go project detected in current directory", Source: "context", Score: 129, Risk: riskLevel("go run .")})
	}

	if containsAny(norm, "安装依赖", "install deps", "install dependencies") && contains(ctx.ProjectTypes, "node") {
		command := "npm install"
		if ctx.PackageManager == "pnpm" {
			command = "pnpm install"
		}
		result = append(result, Candidate{Command: command, Reason: "package manager detected in current directory", Source: "context", Score: 128, Risk: riskLevel(command)})
	}

	if containsAny(norm, "build", "编译") && contains(ctx.ProjectTypes, "go") {
		result = append(result, Candidate{Command: "go build ./...", Reason: "go project detected in current directory", Source: "context", Score: 126, Risk: riskLevel("go build ./...")})
	}

	if containsAny(norm, "docker", "容器", "启动服务", "compose up") && contains(ctx.ProjectTypes, "docker") {
		result = append(result, Candidate{Command: "docker compose up -d", Reason: "docker project files detected in current directory", Source: "context", Score: 126, Risk: riskLevel("docker compose up -d")})
	}

	if shell == "powershell" && containsAny(norm, "进入", "cd", "go to", "切换目录") {
		if match := project.MatchDirectory(ctx, arg); match != "" {
			command := "Set-Location .\\" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project directory", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	} else if shell != "powershell" && containsAny(norm, "进入", "cd", "go to", "切换目录") {
		if match := project.MatchDirectory(ctx, arg); match != "" {
			command := "cd ./" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project directory", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	}

	if shell == "powershell" && containsAny(norm, "读取", "打开文件", "read file", "cat") {
		if match := project.MatchFile(ctx, arg); match != "" {
			command := "Get-Content .\\" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project file", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	} else if shell != "powershell" && containsAny(norm, "读取", "打开文件", "read file", "cat") {
		if match := project.MatchFile(ctx, arg); match != "" {
			command := "cat ./" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project file", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	}

	return result
}

func looksNaturalLanguage(query string) bool {
	for _, r := range query {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return strings.Contains(query, " ")
}

func normalize(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	replacer := strings.NewReplacer("\\", " ", "/", " ", "_", " ", "-", " ")
	in = replacer.Replace(in)
	return strings.Join(strings.Fields(in), " ")
}

func tokens(in string) []string {
	return strings.Fields(in)
}

func containsAny(in string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(in, normalize(value)) {
			return true
		}
	}
	return false
}

func extractLastMeaningfulToken(query string) string {
	fields := strings.Fields(strings.TrimSpace(query))
	stopwords := map[string]struct{}{
		"安装": {}, "卸载": {}, "更新": {}, "升级": {}, "查找": {}, "搜索": {},
		"install": {}, "uninstall": {}, "update": {}, "upgrade": {}, "search": {}, "find": {},
	}
	for i := len(fields) - 1; i >= 0; i-- {
		token := strings.Trim(fields[i], "\"'` ")
		if token == "" {
			continue
		}
		if _, ok := stopwords[strings.ToLower(token)]; ok {
			continue
		}
		return token
	}
	return ""
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeShell(shell string) string {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "", "powershell", "pwsh":
		return "powershell"
	default:
		return strings.ToLower(strings.TrimSpace(shell))
	}
}

func isShellCompatible(command string, shell string) bool {
	shell = normalizeShell(shell)
	if shell == "powershell" {
		return true
	}

	switch command {
	case "Get-ChildItem", "Get-ChildItem -Recurse", "Set-Location", "Get-Content", "Select-String":
		return false
	default:
		return true
	}
}

func projectTypeBonus(specTypes []string, activeTypes []string) float64 {
	if len(specTypes) == 0 || len(activeTypes) == 0 {
		return 0
	}
	for _, specType := range specTypes {
		for _, active := range activeTypes {
			if specType == active {
				return 14
			}
		}
	}
	return 0
}

func projectCommandBonus(command string, activeTypes []string) float64 {
	if len(activeTypes) == 0 {
		return 0
	}
	norm := normalize(command)
	switch {
	case contains(activeTypes, "go") && strings.HasPrefix(norm, "go "):
		return 12
	case contains(activeTypes, "node") && (strings.HasPrefix(norm, "npm ") || strings.HasPrefix(norm, "pnpm ") || strings.HasPrefix(norm, "yarn ")):
		return 12
	case contains(activeTypes, "docker") && strings.HasPrefix(norm, "docker "):
		return 10
	case contains(activeTypes, "git") && strings.HasPrefix(norm, "git "):
		return 8
	default:
		return 0
	}
}

func riskLevel(command string) string {
	norm := " " + normalize(command) + " "
	switch {
	case containsAnyPhrase(norm,
		" rm ",
		" del ",
		" format ",
		" remove item ",
		" shutdown ",
		" stop computer ",
	):
		return "danger"
	case containsAnyPhrase(norm,
		" install ",
		" uninstall ",
		" upgrade ",
		" checkout b ",
		" switch c ",
		" docker compose up ",
		" start process ",
		" npm run dev ",
		" pnpm dev ",
		" yarn dev ",
	) || strings.HasPrefix(strings.TrimSpace(norm), "go run "):
		return "caution"
	default:
		return "safe"
	}
}

func containsAnyPhrase(in string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(in, phrase) {
			return true
		}
	}
	return false
}

func packageManagerCommands(arg string) (search string, install string, upgradeAll string, uninstall string, upgrade string) {
	switch runtime.GOOS {
	case "darwin":
		return "brew search " + arg, "brew install " + arg, "brew upgrade", "brew uninstall " + arg, "brew upgrade " + arg
	case "linux":
		return "apt search " + arg, "sudo apt install " + arg, "sudo apt upgrade", "sudo apt remove " + arg, "sudo apt install --only-upgrade " + arg
	default:
		return "winget search " + arg, "winget install " + arg, "winget upgrade --all", "winget uninstall " + arg, "winget upgrade " + arg
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

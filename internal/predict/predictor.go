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
	Command string   `json:"command"`
	Reason  string   `json:"reason"`
	Source  string   `json:"source"`
	Score   float64  `json:"score"`
	Risk    string   `json:"risk,omitempty"`
	Details []string `json:"details,omitempty"`
}

type RejectedCandidate struct {
	Command string  `json:"command"`
	Source  string  `json:"source"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason"`
}

type DebugReport struct {
	Rejected []RejectedCandidate `json:"rejected,omitempty"`
}

type Request struct {
	Query           string
	CWD             string
	Shell           string
	Limit           int
	Debug           bool
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
	candidates, _ := p.PredictWithDebug(req, entries)
	return candidates
}

func (p *Predictor) PredictWithDebug(req Request, entries []history.Entry) ([]Candidate, DebugReport) {
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	query := strings.TrimSpace(req.Query)
	shell := normalizeShell(req.Shell)
	intent := classifyIntent(query, req.Project)
	candidates := map[string]Candidate{}
	var debugReport DebugReport

	add := func(candidate Candidate) {
		candidate = annotateCandidate(query, intent, req.Project, candidate)
		decision := candidateEligibility(query, candidate.Command, candidate.Source, req.Project)
		if !decision.Allowed {
			if req.Debug {
				debugReport.Rejected = append(debugReport.Rejected, RejectedCandidate{
					Command: candidate.Command,
					Source:  candidate.Source,
					Score:   candidate.Score,
					Reason:  decision.Reason,
				})
			}
			return
		}
		existing, ok := candidates[candidate.Command]
		if !ok {
			candidates[candidate.Command] = candidate
			return
		}
		candidates[candidate.Command] = mergeCandidate(existing, candidate)
	}

	for _, generated := range generateIntentCandidates(intent, shell, req.Project) {
		add(generated)
	}
	for _, generated := range generateProjectCandidates(intent, shell, req.Project) {
		add(generated)
	}

	for _, spec := range p.builtins {
		if !isShellCompatible(spec.Command, shell) {
			continue
		}
		score := scoreCandidate(query, spec.Command, spec.Keywords, 1, 0, projectTypeBonus(spec.ProjectTypes, req.Project.ProjectTypes))
		score += explicitCommandPrefixAdjustment(query, spec.Command, "builtin")
		score += intentAffinityAdjustment(intent, spec.Command, "builtin")
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
		score += historyQualityAdjustment(query, entry.Command, req.Project)
		score += explicitCommandPrefixAdjustment(query, entry.Command, "history")
		score += intentAffinityAdjustment(intent, entry.Command, "history")
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
	return result, debugReport
}

func scoreCandidate(query string, command string, keywords []string, frequency int, recency float64, contextBonus float64) float64 {
	normQuery := normalize(query)
	normCommand := normalize(command)

	if normQuery == "" {
		score := math.Log(float64(max(1, frequency))+1)*5 + recency
		score -= commandLengthPenalty(command)
		score -= commandComplexityPenalty(command)
		return score
	}

	queryTokens := tokens(normQuery)
	commandTokens := tokens(normCommand)

	var score float64
	if normCommand == normQuery {
		score += 24
	}
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

	if allQueryTokensCovered(queryTokens, commandTokens, keywords) && len(commandTokens) <= 4 {
		score += 10
	}

	score += math.Log(float64(max(1, frequency))+1) * 6
	score += recency
	score += contextBonus
	score -= commandLengthPenalty(command)
	score -= commandComplexityPenalty(command)

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

func generateIntentCandidates(intent Intent, shell string, ctx project.Context) []Candidate {
	if strings.TrimSpace(intent.Query) == "" {
		return nil
	}

	var result []Candidate
	arg := intent.Target
	if arg == "" {
		arg = "PACKAGE_OR_TARGET"
	}

	switch intent.Kind {
	case IntentInstallPackage:
		searchCmd, installCmd, _, _, _ := packageManagerCommands(arg)
		result = append(result,
			Candidate{Command: searchCmd, Reason: "search likely package first", Source: "template", Score: 120, Risk: riskLevel(searchCmd)},
			Candidate{Command: installCmd, Reason: "install package from natural-language intent", Source: "template", Score: 128, Risk: riskLevel(installCmd)},
		)
	case IntentUninstallPackage:
		_, _, _, uninstallCmd, _ := packageManagerCommands(arg)
		result = append(result, Candidate{Command: uninstallCmd, Reason: "remove package from natural-language intent", Source: "template", Score: 124, Risk: riskLevel(uninstallCmd)})
	case IntentUpgradePackage:
		_, _, upgradeAllCmd, _, upgradeCmd := packageManagerCommands(arg)
		if arg == "PACKAGE_OR_TARGET" {
			result = append(result, Candidate{Command: upgradeAllCmd, Reason: "upgrade all packages", Source: "template", Score: 118, Risk: riskLevel(upgradeAllCmd)})
		} else {
			result = append(result, Candidate{Command: upgradeCmd, Reason: "upgrade specific package", Source: "template", Score: 118, Risk: riskLevel(upgradeCmd)})
		}
	case IntentSearchFiles, IntentSearchPackages:
		searchCmd, _, _, _, _ := packageManagerCommands(arg)
		packageScore, fileSearchScore := searchIntentScores(intent, ctx)
		result = append(result, Candidate{Command: searchCmd, Reason: "search package by keyword", Source: "template", Score: packageScore, Risk: riskLevel(searchCmd)})
		if shell == "powershell" {
			result = append(result, Candidate{Command: "Select-String -Path . -Pattern \"" + arg + "\"", Reason: "search inside files", Source: "template", Score: fileSearchScore, Risk: riskLevel("Select-String")})
		} else {
			result = append(result, Candidate{Command: "grep -R \"" + arg + "\" .", Reason: "search inside files", Source: "template", Score: fileSearchScore, Risk: riskLevel("grep -R")})
		}
	case IntentListFiles:
		if shell == "powershell" {
			result = append(result, Candidate{Command: "Get-ChildItem", Reason: "list current directory", Source: "template", Score: 106, Risk: riskLevel("Get-ChildItem")})
		} else {
			result = append(result, Candidate{Command: "ls -la", Reason: "list current directory", Source: "template", Score: 106, Risk: riskLevel("ls -la")})
		}
	case IntentChangeDirectory:
		matched := project.MatchDirectory(ctx, arg)
		target := arg
		if matched != "" {
			target = shellPath(shell, matched)
		}
		if shell == "powershell" {
			result = append(result, Candidate{Command: "Set-Location " + target, Reason: "change directory", Source: "template", Score: 106, Risk: riskLevel("Set-Location " + target)})
		} else {
			result = append(result, Candidate{Command: "cd " + target, Reason: "change directory", Source: "template", Score: 106, Risk: riskLevel("cd " + target)})
		}
	case IntentReadFile:
		target := arg
		if matched := project.MatchFile(ctx, arg); matched != "" {
			target = shellPath(shell, matched)
		}
		if shell == "powershell" {
			result = append(result, Candidate{Command: "Get-Content " + target, Reason: "read file from natural-language intent", Source: "template", Score: 108, Risk: riskLevel("Get-Content " + target)})
		} else {
			result = append(result, Candidate{Command: "cat " + target, Reason: "read file from natural-language intent", Source: "template", Score: 108, Risk: riskLevel("cat " + target)})
		}
	case IntentRunTests:
		result = append(result, Candidate{Command: "go test ./...", Reason: "run Go tests", Source: "template", Score: 104, Risk: riskLevel("go test ./...")})
	case IntentOpenEditor:
		result = append(result, Candidate{Command: "code .", Reason: "open current folder in VS Code", Source: "template", Score: 100, Risk: riskLevel("code .")})
	}

	return result
}

func generateProjectCandidates(intent Intent, shell string, ctx project.Context) []Candidate {
	if strings.TrimSpace(intent.Query) == "" {
		return nil
	}

	norm := normalize(intent.Query)
	arg := intent.Target
	if arg == "" {
		arg = "TARGET"
	}

	var result []Candidate

	if intent.Kind == IntentRunTests {
		switch {
		case contains(ctx.ProjectTypes, "go"):
			result = append(result, Candidate{Command: "go test ./...", Reason: "go project detected in current directory", Source: "context", Score: 132, Risk: riskLevel("go test ./...")})
		case contains(ctx.ProjectTypes, "node") && ctx.PackageManager == "pnpm":
			result = append(result, Candidate{Command: "pnpm test", Reason: "pnpm project detected in current directory", Source: "context", Score: 132, Risk: riskLevel("pnpm test")})
		case contains(ctx.ProjectTypes, "node"):
			result = append(result, Candidate{Command: "npm test", Reason: "node project detected in current directory", Source: "context", Score: 130, Risk: riskLevel("npm test")})
		}
	}

	if intent.Kind == IntentStartProject && contains(ctx.ProjectTypes, "node") {
		command := "npm run dev"
		if ctx.PackageManager == "pnpm" {
			command = "pnpm dev"
		}
		result = append(result, Candidate{Command: command, Reason: "frontend project detected in current directory", Source: "context", Score: 130, Risk: riskLevel(command)})
	}

	if intent.Kind == IntentStartProject && contains(ctx.ProjectTypes, "go") {
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

	if intent.Kind == IntentChangeDirectory && shell == "powershell" {
		if match := project.MatchDirectory(ctx, arg); match != "" {
			command := "Set-Location .\\" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project directory", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	} else if intent.Kind == IntentChangeDirectory && shell != "powershell" {
		if match := project.MatchDirectory(ctx, arg); match != "" {
			command := "cd ./" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project directory", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	}

	if intent.Kind == IntentReadFile && shell == "powershell" {
		if match := project.MatchFile(ctx, arg); match != "" {
			command := "Get-Content .\\" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project file", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	} else if intent.Kind == IntentReadFile && shell != "powershell" {
		if match := project.MatchFile(ctx, arg); match != "" {
			command := "cat ./" + match
			result = append(result, Candidate{Command: command, Reason: "matched current project file", Source: "context", Score: 134, Risk: riskLevel(command)})
		}
	}

	return result
}

func annotateCandidate(query string, intent Intent, ctx project.Context, candidate Candidate) Candidate {
	details := uniqueNonEmptyStrings(candidateReasonDetails(query, intent, ctx, candidate))
	candidate.Details = details
	if len(details) > 0 {
		candidate.Reason = explainReason(candidate.Reason, details...)
	}
	return candidate
}

func candidateReasonDetails(query string, intent Intent, ctx project.Context, candidate Candidate) []string {
	var details []string

	if detail := intentReasonDetail(intent, candidate); detail != "" {
		details = append(details, detail)
	}
	if detail := projectReasonDetail(ctx, candidate); detail != "" {
		details = append(details, detail)
	}
	if detail := commandPrefixReasonDetail(query, candidate); detail != "" {
		details = append(details, detail)
	}
	if detail := historyReasonDetail(query, candidate); detail != "" {
		details = append(details, detail)
	}

	if len(details) > 2 {
		details = details[:2]
	}
	return details
}

func intentReasonDetail(intent Intent, candidate Candidate) string {
	switch intent.Kind {
	case IntentInstallPackage:
		if strings.Contains(candidate.Command, " install ") {
			return "matched install intent"
		}
	case IntentUninstallPackage:
		if strings.Contains(candidate.Command, " uninstall ") || strings.Contains(candidate.Command, " remove ") {
			return "matched uninstall intent"
		}
	case IntentUpgradePackage:
		if strings.Contains(candidate.Command, " upgrade ") || strings.Contains(candidate.Command, " update ") {
			return "matched upgrade intent"
		}
	case IntentSearchFiles:
		if candidate.Command == `Select-String -Path . -Pattern "`+intent.Target+`"` || strings.HasPrefix(candidate.Command, `grep -R "`) {
			return "matched search-files intent"
		}
	case IntentSearchPackages:
		if strings.Contains(candidate.Command, " search ") {
			return "matched package-search intent"
		}
	case IntentReadFile:
		if strings.HasPrefix(candidate.Command, "Get-Content ") || strings.HasPrefix(candidate.Command, "cat ") {
			return "matched read-file intent"
		}
	case IntentChangeDirectory:
		if strings.HasPrefix(candidate.Command, "Set-Location ") || strings.HasPrefix(candidate.Command, "cd ") {
			return "matched change-directory intent"
		}
	case IntentRunTests:
		if strings.Contains(candidate.Command, " test ") || strings.HasSuffix(candidate.Command, " test") || strings.HasPrefix(candidate.Command, "go test") {
			return "matched test intent"
		}
	case IntentOpenEditor:
		if strings.HasPrefix(candidate.Command, "code ") {
			return "matched open-editor intent"
		}
	case IntentStartProject:
		if strings.HasPrefix(candidate.Command, "go run") || strings.Contains(candidate.Command, " run dev") || strings.HasPrefix(candidate.Command, "docker compose up") {
			return "matched start-project intent"
		}
	}
	return ""
}

func projectReasonDetail(ctx project.Context, candidate Candidate) string {
	switch {
	case candidate.Source != "context":
		return ""
	case contains(ctx.ProjectTypes, "go") && strings.HasPrefix(candidate.Command, "go "):
		return "current directory looks like a Go project"
	case contains(ctx.ProjectTypes, "node") && (strings.HasPrefix(candidate.Command, "npm ") || strings.HasPrefix(candidate.Command, "pnpm ") || strings.HasPrefix(candidate.Command, "yarn ")):
		return "current directory looks like a Node project"
	case contains(ctx.ProjectTypes, "docker") && strings.HasPrefix(candidate.Command, "docker "):
		return "docker project files detected in current directory"
	default:
		return ""
	}
}

func commandPrefixReasonDetail(query string, candidate Candidate) string {
	family := queryCommandFamily(query)
	if family == CommandFamilyUnknown {
		return ""
	}
	candidateFamily := commandFamily(candidate.Command)
	if candidateFamily != family {
		return ""
	}
	subverb := querySubverbPrefix(query)
	if subverb == "" {
		return fmt.Sprintf("same %s command family as query", family)
	}
	if commandVerbMatches(subverb, commandSubverb(candidate.Command)) {
		return fmt.Sprintf("matches %s %s prefix", family, subverb)
	}
	return ""
}

func historyReasonDetail(query string, candidate Candidate) string {
	if candidate.Source != "history" && !strings.HasSuffix(candidate.Source, "-history") {
		return ""
	}
	if family := queryCommandFamily(query); family != CommandFamilyUnknown && commandFamily(candidate.Command) == family {
		return fmt.Sprintf("history kept because it stays in %s command family", family)
	}
	return ""
}

func explainReason(base string, details ...string) string {
	details = uniqueNonEmptyStrings(details)
	if len(details) == 0 {
		return base
	}
	return base + "; " + strings.Join(details, "; ")
}

func mergeCandidate(existing Candidate, incoming Candidate) Candidate {
	preferred := existing
	other := incoming
	if shouldPreferCandidate(incoming, existing) {
		preferred = incoming
		other = existing
	}

	if other.Score > preferred.Score {
		preferred.Score = other.Score
	}
	if preferred.Risk == "safe" && other.Risk != "safe" {
		preferred.Risk = other.Risk
	}

	combined := append([]string{}, preferred.Details...)
	switch {
	case strings.HasSuffix(other.Source, "-history") || other.Source == "history":
		combined = append(combined, "also seen in local history")
	case other.Source == "context":
		combined = append(combined, "also supported by current project context")
	case other.Source == "template":
		combined = append(combined, "also matched by natural-language template")
	}
	preferred.Details = uniqueNonEmptyStrings(combined)
	preferred.Reason = explainReason(baseReason(preferred.Reason), preferred.Details...)
	return preferred
}

func shouldPreferCandidate(a Candidate, b Candidate) bool {
	aRank := candidateSourcePreference(a.Source)
	bRank := candidateSourcePreference(b.Source)
	if aRank != bRank {
		return aRank > bRank
	}
	return a.Score > b.Score
}

func candidateSourcePreference(source string) int {
	switch {
	case source == "context":
		return 4
	case source == "template":
		return 3
	case source == "builtin":
		return 2
	case source == "history" || strings.HasSuffix(source, "-history"):
		return 1
	default:
		return 0
	}
}

func baseReason(reason string) string {
	if index := strings.Index(reason, "; "); index >= 0 {
		return reason[:index]
	}
	return reason
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
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
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	if quoted := extractLastQuotedToken(query); quoted != "" && !isMeaninglessQueryToken(quoted) {
		return quoted
	}

	fields := strings.Fields(query)
	for i := len(fields) - 1; i >= 0; i-- {
		token := cleanQueryToken(fields[i])
		if token == "" {
			continue
		}
		if isMeaninglessQueryToken(token) {
			continue
		}
		return token
	}
	return ""
}

func extractLastQuotedToken(query string) string {
	for _, quote := range []byte{'"', '\'', '`'} {
		end := strings.LastIndexByte(query, quote)
		if end <= 0 {
			continue
		}
		start := strings.LastIndexByte(query[:end], quote)
		if start < 0 || start >= end {
			continue
		}
		token := cleanQueryToken(query[start+1 : end])
		if token != "" {
			return token
		}
	}
	return ""
}

func cleanQueryToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, "\"'`.,;:!?()[]{}<>，。；：！？（）【】")
	token = stripQueryAffixes(token)
	token = strings.Trim(token, "\"'`.,;:!?()[]{}<>，。；：！？（）【】")
	return token
}

func stripQueryAffixes(token string) string {
	candidate := strings.TrimSpace(token)
	if candidate == "" {
		return ""
	}
	prefixes := []string{
		"帮我", "请帮我", "请", "麻烦", "帮忙", "给我", "把", "我想", "想要", "进入", "切换到", "切换", "安装一下", "安装", "查找一下", "查找", "搜索一下", "搜索", "打开", "读取", "查看",
	}
	suffixes := []string{
		"一下", "目录", "文件", "软件", "项目", "代码", "吧", "呀", "啊", "呢", "哈", "下",
		"directory", "dir", "folder", "file", "package", "pkg", "software",
	}
	for {
		changed := false
		lower := strings.ToLower(candidate)
		for _, prefix := range prefixes {
			if strings.HasPrefix(lower, prefix) {
				candidate = strings.TrimSpace(candidate[len(prefix):])
				changed = true
				break
			}
		}
		if changed {
			continue
		}
		for _, suffix := range suffixes {
			if strings.HasSuffix(lower, suffix) {
				candidate = strings.TrimSpace(candidate[:len(candidate)-len(suffix)])
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return candidate
}

func isMeaninglessQueryToken(token string) bool {
	stopwords := map[string]struct{}{
		"": {}, "安装": {}, "卸载": {}, "更新": {}, "升级": {}, "查找": {}, "搜索": {}, "进入": {}, "切换": {}, "打开": {}, "读取": {}, "查看": {},
		"帮我": {}, "请": {}, "一下": {}, "目录": {}, "文件": {}, "软件": {}, "项目": {},
		"install": {}, "uninstall": {}, "update": {}, "upgrade": {}, "search": {}, "find": {}, "open": {}, "read": {}, "directory": {}, "folder": {}, "file": {}, "package": {},
	}
	_, ok := stopwords[strings.ToLower(strings.TrimSpace(token))]
	return ok
}

func allQueryTokensCovered(queryTokens []string, commandTokens []string, keywords []string) bool {
	if len(queryTokens) == 0 {
		return false
	}
	for _, qt := range queryTokens {
		matched := false
		for _, ct := range commandTokens {
			if qt == ct || strings.HasPrefix(ct, qt) || strings.Contains(ct, qt) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		for _, kw := range keywords {
			normKeyword := normalize(kw)
			if strings.Contains(normKeyword, qt) || strings.Contains(qt, normKeyword) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func searchIntentScores(intent Intent, ctx project.Context) (packageScore float64, fileScore float64) {
	packageScore = 110
	fileScore = 102
	if intent.Kind == IntentSearchFiles || prefersFileSearch(intent.Query, intent.Target, ctx) {
		packageScore = 98
		fileScore = 124
	}
	return packageScore, fileScore
}

func prefersFileSearch(query string, arg string, ctx project.Context) bool {
	norm := normalize(query)
	if !containsAny(norm, "搜索", "查找", "搜一下", "search", "find") {
		return false
	}
	if looksLikeCodeSearchTarget(arg) {
		return true
	}
	if hasProjectContext(ctx) {
		return true
	}
	return false
}

func looksLikeCodeSearchTarget(arg string) bool {
	token := strings.TrimSpace(arg)
	if token == "" {
		return false
	}
	hasLetter := false
	allUpper := true
	for _, r := range token {
		if unicode.IsLetter(r) {
			hasLetter = true
			if unicode.ToUpper(r) != r {
				allUpper = false
			}
		}
	}
	if hasLetter && allUpper && len([]rune(token)) >= 2 {
		return true
	}
	if strings.ContainsAny(token, "/\\._-") {
		return true
	}
	return false
}

func hasProjectContext(ctx project.Context) bool {
	return len(ctx.ProjectTypes) > 0 || len(ctx.Files) > 0 || len(ctx.Directories) > 0
}

func commandLengthPenalty(command string) float64 {
	length := len(strings.TrimSpace(command))
	switch {
	case length > 320:
		return 28
	case length > 220:
		return 20
	case length > 160:
		return 12
	case length > 100:
		return 6
	default:
		return 0
	}
}

func commandComplexityPenalty(command string) float64 {
	separatorCount := strings.Count(command, ";") + strings.Count(command, "&&") + strings.Count(command, "||")
	pipeCount := strings.Count(command, "|")
	penalty := float64(separatorCount) * 4
	penalty += float64(pipeCount) * 5
	if separatorCount+pipeCount >= 3 {
		penalty += 6
	}
	return penalty
}

func historyQualityAdjustment(query string, command string, ctx project.Context) float64 {
	adjustment := 0.0
	coverage := queryCoverageScore(query, command)

	adjustment -= commandLengthPenalty(command) * 1.2
	adjustment -= commandComplexityPenalty(command) * 1.4
	if looksLikeGeneratedHistoryNoise(command) {
		adjustment -= 40
	}
	if penalty := historyCrossProjectPenalty(command, ctx); penalty > 0 {
		adjustment -= penalty
	}
	if strings.TrimSpace(query) != "" {
		switch {
		case coverage == 0:
			adjustment -= 36
		case coverage < 0.5:
			adjustment -= 16
		case coverage >= 1:
			adjustment += 8
		}
		if isShortCommand(command) {
			adjustment += 4
		}
		if isFrequentInteractiveToolCommand(command) && coverage < 0.5 {
			adjustment -= 18
		}
	}
	return adjustment
}

type commandPattern struct {
	family string
	verb   string
}

func explicitCommandPrefixAdjustment(query string, command string, source string) float64 {
	pattern, ok := parseExplicitCommandPattern(query)
	if !ok {
		return 0
	}
	candidate, ok := parseCommandPattern(command)
	if !ok || candidate.family != pattern.family {
		return 0
	}
	if candidate.verb == "" {
		return 0
	}
	if commandVerbMatches(pattern.verb, candidate.verb) {
		if candidate.verb == pattern.verb {
			return 42
		}
		return 28
	}

	switch source {
	case "history":
		return -120
	case "builtin":
		return -64
	default:
		return -24
	}
}

func parseExplicitCommandPattern(query string) (commandPattern, bool) {
	if strings.TrimSpace(query) == "" || containsHan(query) {
		return commandPattern{}, false
	}
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(fields) < 2 || !isCommandToken(fields[0]) || !isKnownCommandFamily(fields[0]) {
		return commandPattern{}, false
	}
	pattern := commandPattern{
		family: fields[0],
		verb:   parseVerb(fields),
	}
	if pattern.verb == "" {
		return commandPattern{}, false
	}
	return pattern, true
}

func parseCommandPattern(command string) (commandPattern, bool) {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(command)))
	if len(fields) < 2 || !isCommandToken(fields[0]) || !isKnownCommandFamily(fields[0]) {
		return commandPattern{}, false
	}
	pattern := commandPattern{
		family: fields[0],
		verb:   parseVerb(fields),
	}
	if pattern.verb == "" {
		return commandPattern{}, false
	}
	return pattern, true
}

func parseVerb(fields []string) string {
	if len(fields) < 2 {
		return ""
	}
	switch fields[0] {
	case "docker":
		if len(fields) >= 3 && fields[1] == "compose" {
			return fields[1] + " " + fields[2]
		}
		return fields[1]
	case "npm", "pnpm", "yarn":
		if len(fields) >= 3 && fields[1] == "run" {
			return fields[1] + " " + fields[2]
		}
		return fields[1]
	default:
		return fields[1]
	}
}

func commandVerbMatches(queryVerb string, candidateVerb string) bool {
	return strings.HasPrefix(candidateVerb, queryVerb) || strings.HasPrefix(queryVerb, candidateVerb)
}

func isCommandToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func isKnownCommandFamily(token string) bool {
	switch token {
	case "git", "docker", "go", "npm", "pnpm", "yarn", "winget", "pip", "kubectl", "brew", "apt", "cargo", "dotnet", "python", "make":
		return true
	default:
		return false
	}
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func intentAffinityAdjustment(intent Intent, command string, source string) float64 {
	if intent.Kind == IntentUnknown || !looksNaturalLanguage(intent.Query) {
		return 0
	}
	if intentMatchesCommand(intent, command) {
		return 0
	}

	switch source {
	case "history":
		return -80
	case "builtin":
		return -36
	default:
		return -12
	}
}

func intentMatchesCommand(intent Intent, command string) bool {
	norm := " " + normalize(command) + " "
	switch intent.Kind {
	case IntentSearchFiles:
		return containsAnyPhrase(norm, " select string ", " grep ", " rg ", " findstr ")
	case IntentSearchPackages:
		return containsAnyPhrase(norm, " winget search ", " brew search ", " apt search ", " choco search ", " scoop search ")
	case IntentInstallPackage:
		return containsAnyPhrase(norm, " winget install ", " brew install ", " apt install ", " pip install ", " npm install ", " pnpm install ", " choco install ", " scoop install ")
	case IntentUninstallPackage:
		return containsAnyPhrase(norm, " winget uninstall ", " brew uninstall ", " apt remove ", " pip uninstall ", " npm uninstall ", " pnpm remove ", " choco uninstall ", " scoop uninstall ")
	case IntentUpgradePackage:
		return containsAnyPhrase(norm, " winget upgrade ", " brew upgrade ", " apt upgrade ", " pip install only upgrade ", " npm update ", " pnpm update ", " choco upgrade ", " scoop update ")
	case IntentRunTests:
		return containsAnyPhrase(norm, " go test ", " npm test ", " pnpm test ", " yarn test ", " pytest ", " cargo test ")
	case IntentChangeDirectory:
		return containsAnyPhrase(norm, " set location ", " cd ")
	case IntentReadFile:
		return containsAnyPhrase(norm, " get content ", " cat ", " type ", " bat ", " less ", " more ")
	case IntentOpenEditor:
		return containsAnyPhrase(norm, " code . ", " code ")
	case IntentStartProject:
		return containsAnyPhrase(norm, " go run ", " npm run dev ", " pnpm dev ", " yarn dev ", " docker compose up ")
	case IntentListFiles:
		return containsAnyPhrase(norm, " get childitem ", " ls ", " dir ")
	default:
		return true
	}
}

func queryCoverageScore(query string, command string) float64 {
	queryTokens := tokens(normalize(query))
	if len(queryTokens) == 0 {
		return 1
	}
	commandTokens := tokens(normalize(command))
	matched := 0
	for _, qt := range queryTokens {
		for _, ct := range commandTokens {
			if qt == ct || strings.HasPrefix(ct, qt) || strings.Contains(ct, qt) {
				matched++
				break
			}
		}
	}
	return float64(matched) / float64(len(queryTokens))
}

func looksLikeGeneratedHistoryNoise(command string) bool {
	norm := normalize(command)
	return strings.Contains(norm, "cliai predict") ||
		strings.HasPrefix(norm, "cliai ") ||
		strings.HasPrefix(norm, "csg ") ||
		strings.HasPrefix(norm, "csi ") ||
		strings.HasPrefix(norm, "csc ")
}

func historyCrossProjectPenalty(command string, ctx project.Context) float64 {
	norm := normalize(command)
	if !(strings.Contains(norm, " cd ") || strings.Contains(norm, " set location ")) {
		return 0
	}
	if len(ctx.ProjectTypes) == 0 {
		return 8
	}
	for _, kind := range ctx.ProjectTypes {
		if strings.Contains(norm, kind) {
			return 0
		}
	}
	return 14
}

func isShortCommand(command string) bool {
	return len(strings.TrimSpace(command)) <= 28 && strings.Count(command, " ") <= 3
}

func isFrequentInteractiveToolCommand(command string) bool {
	norm := normalize(command)
	for _, prefix := range []string{"opencode", "claude", "codex", "cursor", "aider"} {
		if strings.HasPrefix(norm, prefix) {
			return true
		}
	}
	return false
}

func shellPath(shell string, target string) string {
	if shell == "powershell" {
		return ".\\" + target
	}
	return "./" + target
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
	rawLower := strings.ToLower(strings.TrimSpace(command))
	switch {
	case isDangerousDelete(norm),
		isPipeDownloadExecute(rawLower),
		isDangerousGit(norm),
		isDangerousDocker(norm),
		isDangerousExecutionPolicy(norm),
		overwritesCriticalPath(rawLower),
		leaksSensitiveEnvironment(rawLower),
		containsAnyPhrase(norm, " format ", " mkfs ", " diskpart clean ", " diskpart format "):
		return "danger"
	case containsAnyPhrase(norm,
		" sudo ",
		" install ",
		" uninstall ",
		" upgrade ",
		" checkout b ",
		" switch c ",
		" docker compose up ",
		" start process ",
		" docker system prune ",
		" docker compose down ",
		" start process verb runas ",
		" npm run dev ",
		" pnpm dev ",
		" yarn dev ",
		" chmod r 777 ",
		" git push force ",
		" kubectl delete ",
		" docker rm f ",
	) || strings.HasPrefix(strings.TrimSpace(norm), "go run "):
		return "caution"
	case strings.Contains(rawLower, "start-process") && strings.Contains(rawLower, "-verb runas"):
		return "caution"
	default:
		return "safe"
	}
}

func isDangerousDelete(norm string) bool {
	switch {
	case containsAnyPhrase(norm,
		" rm rf ",
		" rm fr ",
		" del s q ",
		" rmdir s q ",
	):
		return true
	case strings.Contains(norm, " remove item ") &&
		(strings.Contains(norm, " recurse ") || strings.Contains(norm, " force ")):
		return true
	default:
		return false
	}
}

func isPipeDownloadExecute(rawLower string) bool {
	if !strings.Contains(rawLower, "|") {
		return false
	}
	downloaders := []string{"curl ", "wget ", "irm ", "iwr ", "invoke-restmethod", "invoke-webrequest"}
	executors := []string{"| sh", "| bash", "| zsh", "| iex", "| invoke-expression"}
	hasDownloader := false
	for _, downloader := range downloaders {
		if strings.Contains(rawLower, downloader) {
			hasDownloader = true
			break
		}
	}
	if !hasDownloader {
		return false
	}
	for _, executor := range executors {
		if strings.Contains(rawLower, executor) {
			return true
		}
	}
	return false
}

func isDangerousGit(norm string) bool {
	return containsAnyPhrase(norm,
		" git reset hard ",
		" git clean fd ",
		" git clean fdx ",
	)
}

func isDangerousDocker(norm string) bool {
	switch {
	case containsAnyPhrase(norm,
		" docker system prune a ",
		" docker system prune volumes ",
		" docker compose down v ",
	):
		return true
	default:
		return false
	}
}

func isDangerousExecutionPolicy(norm string) bool {
	return containsAnyPhrase(norm,
		" set executionpolicy ",
		" set executionpolicy bypass ",
		" set executionpolicy unrestricted ",
	)
}

func overwritesCriticalPath(rawLower string) bool {
	if !strings.Contains(rawLower, ">") {
		return false
	}
	criticalTargets := []string{
		"/etc/hosts",
		"/etc/sudoers",
		"/etc/passwd",
		"~/.ssh/authorized_keys",
		".ssh/authorized_keys",
		".ssh\\authorized_keys",
		"c:\\windows\\system32\\drivers\\etc\\hosts",
	}
	for _, target := range criticalTargets {
		if strings.Contains(rawLower, "> "+target) || strings.Contains(rawLower, ">"+target) {
			return true
		}
	}
	return false
}

func leaksSensitiveEnvironment(rawLower string) bool {
	sensitiveNames := []string{
		"token", "password", "secret", "authorization", "bearer", "api_key", "apikey", "client_secret", "access_token", "refresh_token",
	}
	for _, name := range sensitiveNames {
		if strings.Contains(rawLower, "$env:"+name) ||
			strings.Contains(rawLower, "${"+name+"}") ||
			strings.Contains(rawLower, "$"+name) ||
			strings.Contains(rawLower, "%"+name+"%") ||
			strings.Contains(rawLower, "printenv "+name) ||
			strings.Contains(rawLower, "env |") && strings.Contains(rawLower, name) {
			return true
		}
	}
	return false
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

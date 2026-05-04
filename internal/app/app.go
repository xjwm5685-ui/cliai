package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sanqiu/cliai/internal/config"
	"github.com/sanqiu/cliai/internal/feedback"
	"github.com/sanqiu/cliai/internal/history"
	"github.com/sanqiu/cliai/internal/predict"
	"github.com/sanqiu/cliai/internal/project"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "predict":
		return runPredict(args[1:], stdout, stderr)
	case "predictor":
		return runPredictor(args[1:], stdout, stderr)
	case "history":
		return runHistory(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "feedback":
		return runFeedback(args[1:], stdout, stderr)
	case "shell":
		return runShell(args[1:], stdout, stderr)
	case "selftest":
		return runSelftest(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintf(stdout, "%s (commit=%s, built=%s)\n", Version, Commit, BuildDate)
		return 0
	case "help", "--help", "-h":
		printHelp(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 1
	}
}

func runPredict(args []string, stdout io.Writer, stderr io.Writer) int {
	args = normalizePredictArgs(args)
	fs := flag.NewFlagSet("predict", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	cwd, _ := os.Getwd()
	var (
		limit       = fs.Int("limit", 5, "number of suggestions to return")
		shell       = fs.String("shell", cfg.Shell, "shell type")
		asJSON      = fs.Bool("json", false, "output json")
		useCWD      = fs.String("cwd", cwd, "current working directory")
		debug       = fs.Bool("debug", false, "print debug information to stderr")
		copyTop     = fs.Bool("copy", false, "copy the selected or top command to clipboard")
		command     = fs.Bool("command-only", false, "print only the selected or top command")
		interactive = fs.Bool("interactive", false, "interactively choose a suggestion")
	)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		fmt.Fprintln(stderr, "usage: cliai predict [flags] <query>")
		return 1
	}

	projectCtx, err := project.Detect(*useCWD)
	if err != nil {
		fmt.Fprintf(stderr, "detect project context: %v\n", err)
		return 1
	}

	cachePath, err := config.HistoryCachePath()
	if err != nil {
		fmt.Fprintf(stderr, "resolve history cache: %v\n", err)
		return 1
	}

	cached, err := history.LoadCache(cachePath)
	if err != nil {
		fmt.Fprintf(stderr, "load cached history: %v\n", err)
		return 1
	}

	var live []history.Entry
	if cfg.HistoryPath != "" {
		live, err = history.Import(cfg.HistoryPath, cfg.Shell, cfg.Local.MaxHistory)
		if err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "read powershell history: %v\n", err)
		}
	}

	allHistory := history.Merge(cached, live)
	feedbackPath, err := config.FeedbackPath()
	if err != nil {
		fmt.Fprintf(stderr, "resolve feedback path: %v\n", err)
		return 1
	}
	feedbackEntries, err := feedback.Load(feedbackPath)
	if err != nil {
		fmt.Fprintf(stderr, "load feedback: %v\n", err)
		return 1
	}

	engine := predict.New()
	candidates := engine.Predict(predict.Request{
		Query:           query,
		CWD:             *useCWD,
		Shell:           normalizeShell(*shell),
		Limit:           *limit,
		Project:         projectCtx,
		FeedbackBonuses: feedback.CommandBonuses(query, feedbackEntries),
	}, allHistory)

	if *debug {
		printDebug(stderr, debugInfo{
			Query:          query,
			Shell:          normalizeShell(*shell),
			CWD:            *useCWD,
			HistoryEntries: len(allHistory),
			FeedbackCount:  len(feedbackEntries),
			Project:        projectCtx,
		}, candidates)
	}

	if len(candidates) == 0 {
		fmt.Fprintln(stderr, "no suggestions found")
		return 1
	}

	if *interactive {
		selected, ok := chooseCandidate(candidates, stdout, stderr)
		if !ok {
			return 1
		}
		if err := feedback.Record(feedbackPath, query, selected.Command); err != nil {
			fmt.Fprintf(stderr, "record feedback: %v\n", err)
		}
		if *copyTop {
			if err := copyToClipboard(selected.Command); err != nil {
				fmt.Fprintf(stderr, "copy to clipboard: %v\n", err)
			}
		}
		fmt.Fprintln(stdout, selected.Command)
		return 0
	}

	if *copyTop {
		if err := copyToClipboard(candidates[0].Command); err != nil {
			fmt.Fprintf(stderr, "copy to clipboard: %v\n", err)
			return 1
		}
	}

	if *command {
		fmt.Fprintln(stdout, candidates[0].Command)
		return 0
	}

	if *asJSON {
		data, err := json.MarshalIndent(candidates, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "COMMAND\tSOURCE\tRISK\tWHY")
	for _, candidate := range candidates {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", candidate.Command, candidate.Source, candidate.Risk, candidate.Reason)
	}
	_ = tw.Flush()
	return 0
}

func runHistory(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cliai history import [--file path]")
		return 1
	}
	switch args[0] {
	case "import":
		fs := flag.NewFlagSet("history import", flag.ContinueOnError)
		fs.SetOutput(stderr)
		file := fs.String("file", "", "history file to import")
		if err := fs.Parse(args[1:]); err != nil {
			return 1
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(stderr, "load config: %v\n", err)
			return 1
		}

		source := strings.TrimSpace(*file)
		if source == "" {
			source = cfg.HistoryPath
		}
		if source == "" {
			fmt.Fprintln(stderr, "history path is empty")
			return 1
		}

		entries, err := history.Import(source, cfg.Shell, cfg.Local.MaxHistory)
		if err != nil {
			fmt.Fprintf(stderr, "import history: %v\n", err)
			return 1
		}

		cachePath, err := config.HistoryCachePath()
		if err != nil {
			fmt.Fprintf(stderr, "resolve history cache: %v\n", err)
			return 1
		}
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
			fmt.Fprintf(stderr, "prepare history cache directory: %v\n", err)
			return 1
		}
		if err := history.SaveCache(cachePath, entries); err != nil {
			fmt.Fprintf(stderr, "save history cache: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "imported %d commands from %s\n", len(entries), source)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown history subcommand: %s\n", args[0])
		return 1
	}
}

func runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cliai config [show|set <key> <value>]")
		return 1
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}

	switch args[0] {
	case "show":
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode config: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, "usage: cliai config set <key> <value>")
			return 1
		}
		if err := config.Set(cfg, args[1], strings.Join(args[2:], " ")); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(stderr, "save config: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "updated %s\n", args[1])
		return 0
	default:
		fmt.Fprintf(stderr, "unknown config subcommand: %s\n", args[0])
		return 1
	}
}

func runFeedback(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cliai feedback [show|accept]")
		return 1
	}

	path, err := config.FeedbackPath()
	if err != nil {
		fmt.Fprintf(stderr, "resolve feedback path: %v\n", err)
		return 1
	}

	switch args[0] {
	case "show":
		fs := flag.NewFlagSet("feedback show", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "output json")
		if err := fs.Parse(args[1:]); err != nil {
			return 1
		}

		entries, err := feedback.Load(path)
		if err != nil {
			fmt.Fprintf(stderr, "load feedback: %v\n", err)
			return 1
		}
		if entries == nil {
			entries = []feedback.Entry{}
		}
		if *asJSON {
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				fmt.Fprintf(stderr, "encode json: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "COUNT\tQUERY\tCOMMAND\tLAST_ACCEPTED")
		for _, entry := range entries {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", entry.Count, entry.Query, entry.Command, entry.LastAccepted.Format(time.RFC3339))
		}
		_ = tw.Flush()
		return 0
	case "accept":
		fs := flag.NewFlagSet("feedback accept", flag.ContinueOnError)
		fs.SetOutput(stderr)
		query := fs.String("query", "", "original user query")
		if err := fs.Parse(args[1:]); err != nil {
			return 1
		}
		command := strings.TrimSpace(strings.Join(fs.Args(), " "))
		if command == "" {
			fmt.Fprintln(stderr, "usage: cliai feedback accept --query <query> <command>")
			return 1
		}
		if err := feedback.Record(path, *query, command); err != nil {
			fmt.Fprintf(stderr, "record feedback: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "feedback recorded")
		return 0
	default:
		fmt.Fprintf(stderr, "unknown feedback subcommand: %s\n", args[0])
		return 1
	}
}

func runShell(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		printShellUsage(stderr)
		return 1
	}

	targetShell := strings.ToLower(strings.TrimSpace(args[1]))
	switch targetShell {
	case "powershell", "bash", "zsh":
	default:
		printShellUsage(stderr)
		return 1
	}

	switch args[0] {
	case "init":
		snippet, err := shellInitSnippet(targetShell)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintln(stdout, snippet)
		return 0
	case "install":
		if targetShell == "powershell" {
			return runShellInstallPowerShell(stdout, stderr)
		}
		return runShellInstallPOSIX(targetShell, stdout, stderr)
	default:
		printShellUsage(stderr)
		return 1
	}
}

func runSelftest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("selftest", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "output json")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	cwd, _ := os.Getwd()
	projectCtx, projectErr := project.Detect(cwd)
	cfg, configErr := config.Load()
	cachePath, cacheErr := config.HistoryCachePath()
	feedbackPath, feedbackErr := config.FeedbackPath()

	type result struct {
		Version     string          `json:"version"`
		Commit      string          `json:"commit"`
		BuildDate   string          `json:"build_date"`
		ConfigOK    bool            `json:"config_ok"`
		CachePathOK bool            `json:"cache_path_ok"`
		FeedbackOK  bool            `json:"feedback_path_ok"`
		ProjectCtx  project.Context `json:"project_context"`
		Errors      []string        `json:"errors,omitempty"`
		HistoryPath string          `json:"history_path"`
	}

	out := result{
		Version:     Version,
		Commit:      Commit,
		BuildDate:   BuildDate,
		ConfigOK:    configErr == nil,
		CachePathOK: cacheErr == nil && strings.TrimSpace(cachePath) != "",
		FeedbackOK:  feedbackErr == nil && strings.TrimSpace(feedbackPath) != "",
		ProjectCtx:  projectCtx,
	}
	if configErr == nil {
		out.HistoryPath = cfg.HistoryPath
	}
	if configErr != nil {
		out.Errors = append(out.Errors, configErr.Error())
	}
	if projectErr != nil {
		out.Errors = append(out.Errors, projectErr.Error())
	}
	if cacheErr != nil {
		out.Errors = append(out.Errors, cacheErr.Error())
	}
	if feedbackErr != nil {
		out.Errors = append(out.Errors, feedbackErr.Error())
	}

	if *asJSON {
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode json: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintf(stdout, "version: %s\ncommit: %s\nbuild_date: %s\nconfig_ok: %t\ncache_path_ok: %t\nfeedback_path_ok: %t\nproject_types: %s\n",
			out.Version, out.Commit, out.BuildDate, out.ConfigOK, out.CachePathOK, out.FeedbackOK, strings.Join(out.ProjectCtx.ProjectTypes, ","))
	}
	if len(out.Errors) > 0 {
		return 1
	}
	return 0
}

func powershellSnippet() string {
	return `function Invoke-CliaiSuggestion {
  param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Query
  )
  cliai predict --limit 5 $Query
}

function Invoke-CliaiInteractiveSuggestion {
  param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Query
  )
  cliai predict --interactive --copy $Query
}

function Get-CliaiTopCommand {
  param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$Query
  )
  cliai predict --command-only $Query
}

Set-Alias csg Invoke-CliaiSuggestion
Set-Alias csi Invoke-CliaiInteractiveSuggestion
Set-Alias csc Get-CliaiTopCommand

Register-ArgumentCompleter -CommandName csg -ScriptBlock {
  param($commandName, $parameterName, $wordToComplete, $commandAst, $fakeBoundParameters)
  $query = $wordToComplete
  if ([string]::IsNullOrWhiteSpace($query)) {
    return
  }
  cliai predict --command-only $query | ForEach-Object {
    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
  }
}

Write-Host "cliai helper loaded. Use: csg '安装 vscode', csi 'git st', csc 'run tests'" -ForegroundColor Green`
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "cliai: local-first command prediction and completion CLI")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  predict <query>          Predict commands from a fragment or natural language")
	fmt.Fprintln(w, "  predictor serve          Start the local predictor bridge for PowerShell")
	fmt.Fprintln(w, "  history import           Import PowerShell history into the local cache")
	fmt.Fprintln(w, "  config show              Show current config")
	fmt.Fprintln(w, "  config set <key> <value> Update config values")
	fmt.Fprintln(w, "  feedback show            Show accepted suggestions")
	fmt.Fprintln(w, "  feedback accept          Record a selected suggestion")
	fmt.Fprintln(w, "  shell init <shell>       Print the shell integration snippet for powershell, bash, or zsh")
	fmt.Fprintln(w, "  shell install <shell>    Install the shell integration into your profile")
	fmt.Fprintln(w, "  selftest                 Run local smoke checks")
	fmt.Fprintln(w, "  version                  Print version/build metadata")
}

func printShellUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: cliai shell [init|install] [powershell|bash|zsh]")
}

func normalizePredictArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	flagValues := map[string]bool{
		"--limit": true,
		"--shell": true,
		"--cwd":   true,
	}
	flagOnly := map[string]bool{
		"--json":         true,
		"--debug":        true,
		"--copy":         true,
		"--command-only": true,
		"--interactive":  true,
	}

	consumeFront := 0
	var frontFlags []string
	for consumeFront < len(args) {
		token := args[consumeFront]
		if flagOnly[token] {
			frontFlags = append(frontFlags, token)
			consumeFront++
			continue
		}
		if flagValues[token] && consumeFront+1 < len(args) {
			frontFlags = append(frontFlags, token, args[consumeFront+1])
			consumeFront += 2
			continue
		}
		if token == "--" {
			consumeFront++
		}
		break
	}

	trailingStart := len(args)
	var trailingFlags []string
	for trailingStart > consumeFront {
		token := args[trailingStart-1]
		if flagOnly[token] {
			trailingFlags = append([]string{token}, trailingFlags...)
			trailingStart--
			continue
		}
		if trailingStart >= 2 && flagValues[args[trailingStart-2]] {
			trailingFlags = append([]string{args[trailingStart-2], args[trailingStart-1]}, trailingFlags...)
			trailingStart -= 2
			continue
		}
		break
	}

	middle := args[consumeFront:trailingStart]
	if len(middle) > 0 && middle[0] == "--" {
		middle = middle[1:]
	}
	return append(append(frontFlags, trailingFlags...), middle...)
}

func normalizeShell(shell string) string {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "", "powershell", "pwsh":
		return "powershell"
	default:
		return strings.ToLower(strings.TrimSpace(shell))
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

type debugInfo struct {
	Query          string
	Shell          string
	CWD            string
	HistoryEntries int
	FeedbackCount  int
	Project        project.Context
}

func printDebug(w io.Writer, info debugInfo, candidates []predict.Candidate) {
	fmt.Fprintf(w, "debug query=%q shell=%s cwd=%s history_entries=%d feedback_entries=%d project_types=%s package_manager=%s\n",
		info.Query, info.Shell, info.CWD, info.HistoryEntries, info.FeedbackCount, strings.Join(info.Project.ProjectTypes, ","), info.Project.PackageManager)
	for index, candidate := range candidates {
		fmt.Fprintf(w, "debug candidate[%d] score=%.2f source=%s risk=%s command=%q reason=%q\n",
			index, candidate.Score, candidate.Source, candidate.Risk, candidate.Command, candidate.Reason)
	}
}

func chooseCandidate(candidates []predict.Candidate, stdout io.Writer, stderr io.Writer) (predict.Candidate, bool) {
	reader := bufio.NewReader(os.Stdin)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tCOMMAND\tSOURCE\tRISK\tWHY")
	for index, candidate := range candidates {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", index+1, candidate.Command, candidate.Source, candidate.Risk, candidate.Reason)
	}
	_ = tw.Flush()
	fmt.Fprint(stdout, "选择编号并按回车: ")
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(stderr, "read selection: %v\n", err)
		return predict.Candidate{}, false
	}
	selected, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || selected < 1 || selected > len(candidates) {
		fmt.Fprintln(stderr, "invalid selection")
		return predict.Candidate{}, false
	}
	return candidates[selected-1], true
}

func copyToClipboard(value string) error {
	name, args, err := clipboardCommand(runtime.GOOS, exec.LookPath)
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func clipboardCommand(goos string, lookPath func(file string) (string, error)) (string, []string, error) {
	switch goos {
	case "windows":
		return "cmd", []string{"/c", "clip"}, nil
	case "darwin":
		if _, err := lookPath("pbcopy"); err != nil {
			return "", nil, fmt.Errorf("clipboard tool not found: pbcopy")
		}
		return "pbcopy", nil, nil
	default:
		type candidate struct {
			name string
			args []string
		}
		for _, item := range []candidate{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		} {
			if _, err := lookPath(item.name); err == nil {
				return item.name, item.args, nil
			}
		}
		return "", nil, errors.New("clipboard tool not found: install wl-copy, xclip, or xsel")
	}
}

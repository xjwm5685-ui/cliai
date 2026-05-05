package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
)

type Entry struct {
	Command  string    `json:"command"`
	Count    int       `json:"count"`
	LastUsed time.Time `json:"last_used"`
	Source   string    `json:"source"`
}

func Import(path string, shell string, limit int) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return importFromScanner(bufio.NewScanner(file), normalizeShell(shell), limit)
}

func ImportPowerShell(path string, limit int) ([]Entry, error) {
	return Import(path, "powershell", limit)
}

func importFromScanner(scanner *bufio.Scanner, shell string, limit int) ([]Entry, error) {
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	type aggregate struct {
		count int
		order int
	}

	lineNo := 0
	agg := map[string]aggregate{}
	for scanner.Scan() {
		lineNo++
		line, ok := normalizeHistoryLine(scanner.Text(), shell)
		if !ok {
			continue
		}
		current := agg[line]
		current.count++
		current.order = lineNo
		agg[line] = current
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}

	now := time.Now()
	entries := make([]Entry, 0, len(agg))
	for command, item := range agg {
		recencyMinutes := max(1, lineNo-item.order)
		entries = append(entries, Entry{
			Command:  command,
			Count:    item.count,
			LastUsed: now.Add(-time.Duration(recencyMinutes) * time.Minute),
			Source:   shell + "-history",
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].LastUsed.Equal(entries[j].LastUsed) {
			return entries[i].Count > entries[j].Count
		}
		return entries[i].LastUsed.After(entries[j].LastUsed)
	})

	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func normalizeShell(shell string) string {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "", "powershell", "pwsh":
		return "powershell"
	case "bash", "zsh", "fish":
		return strings.ToLower(strings.TrimSpace(shell))
	default:
		return strings.ToLower(strings.TrimSpace(shell))
	}
}

func normalizeHistoryLine(line string, shell string) (string, bool) {
	switch shell {
	case "zsh":
		if strings.HasPrefix(line, ": ") {
			if index := strings.Index(line, ";"); index >= 0 && index < len(line)-1 {
				line = line[index+1:]
			}
		}
	case "fish":
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- cmd:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "- cmd:"))
		} else {
			return "", false
		}
	}

	return sanitizeCommand(line)
}

func LoadCache(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse history cache: %w", err)
	}
	return entries, nil
}

func SaveCache(path string, entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Merge(sources ...[]Entry) []Entry {
	merged := map[string]Entry{}
	for _, source := range sources {
		for _, entry := range source {
			key := strings.TrimSpace(entry.Command)
			if key == "" {
				continue
			}
			current, ok := merged[key]
			if !ok {
				merged[key] = entry
				continue
			}
			current.Count += entry.Count
			if entry.LastUsed.After(current.LastUsed) {
				current.LastUsed = entry.LastUsed
			}
			if current.Source == "" {
				current.Source = entry.Source
			}
			merged[key] = current
		}
	}

	entries := make([]Entry, 0, len(merged))
	for _, entry := range merged {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].LastUsed.After(entries[j].LastUsed)
		}
		return entries[i].Count > entries[j].Count
	})
	return entries
}

func sanitizeCommand(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
		return "", false
	}
	if len(line) > 500 {
		return "", false
	}
	if strings.Count(line, "\x00") > 0 {
		return "", false
	}

	lower := strings.ToLower(line)
	if containsSensitiveContent(lower) {
		return "", false
	}
	if looksLikeBareURL(line) {
		return "", false
	}
	if looksLikeCommandOutputText(line) {
		return "", false
	}
	if looksLikeConcatenatedPredictionNoise(lower) {
		return "", false
	}
	if looksLikeConcatenatedCommand(lower) {
		return "", false
	}
	if isOverlyComplexHistory(line) {
		return "", false
	}
	return line, true
}

func looksLikeBareURL(line string) bool {
	trimmed := strings.Trim(strings.TrimSpace(line), "\"'`")
	return (strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) &&
		!strings.ContainsAny(trimmed, " \t")
}

func looksLikeCommandOutputText(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || commandStartsWithKnownExecutable(trimmed) || looksLikeBareURL(trimmed) {
		return false
	}
	if startsWithEmojiOrPrompt(trimmed) {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{
		"安装完成", "重新打开", "successfully", "failed", "error:", "warning:", "info:",
		"exception", "stack trace", "打开：", "open:", "at line:", "提示:",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if strings.Contains(trimmed, "->") || strings.Contains(trimmed, "=>") {
		return true
	}
	if containsMostlyHanText(trimmed) {
		return true
	}
	return false
}

func containsSensitiveContent(lower string) bool {
	sensitiveMarkers := []string{
		"authorization:",
		"authorization=",
		"api key",
		"api-key",
		"api_key",
		"apikey",
		"password",
		"passwd",
		"secret",
		"bearer ",
		"token=",
		"token ",
		" begin private key",
		" id_rsa",
		" id_dsa",
		".ssh\\",
		".ssh/",
		".pem",
		".pfx",
		".p12",
		"client_secret",
		"access_key",
		"access_token",
		"refresh_token",
	}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, name := range []string{
		"token", "password", "passwd", "secret", "authorization", "bearer", "apikey", "api_key", "api-key",
		"access_token", "refresh_token", "client_secret", "access_key",
	} {
		if containsSensitiveAssignment(lower, name) || containsSensitiveSetter(lower, name) {
			return true
		}
	}
	return false
}

func containsSensitiveAssignment(lower string, name string) bool {
	for _, operator := range []string{"=", ":"} {
		pattern := name + operator
		index := strings.Index(lower, pattern)
		if index < 0 {
			continue
		}
		value := strings.TrimSpace(lower[index+len(pattern):])
		if len(value) >= 4 {
			return true
		}
	}
	return false
}

func containsSensitiveSetter(lower string, name string) bool {
	return strings.Contains(lower, "setx "+name+" ") ||
		strings.Contains(lower, "export "+name+"=") ||
		strings.Contains(lower, "$env:"+name+" =") ||
		strings.Contains(lower, "$env:"+name+"=")
}

func looksLikeConcatenatedPredictionNoise(lower string) bool {
	cliaiPredict := strings.Count(lower, "cliai predict")
	csgMentions := countWholeWord(lower, "csg")
	if cliaiPredict >= 2 || csgMentions >= 3 {
		return true
	}
	if cliaiPredict > 0 && csgMentions > 0 {
		return true
	}
	return false
}

func looksLikeConcatenatedCommand(lower string) bool {
	phrases := []string{
		"cliai predict",
		"winget install",
		"winget upgrade",
		"npm install",
		"pnpm install",
		"npm run",
		"pnpm run",
		"go install",
		"git clone",
	}
	for _, phrase := range phrases {
		first := strings.Index(lower, phrase)
		if first < 0 {
			continue
		}
		second := strings.Index(lower[first+len(phrase):], phrase)
		if second < 0 {
			continue
		}
		second += first + len(phrase)
		if second == 0 || isSeparatorByte(lower[second-1]) {
			continue
		}
		if !hasCommandSeparator(lower[first+len(phrase) : second]) {
			return true
		}
	}
	return false
}

func isOverlyComplexHistory(line string) bool {
	length := len(line)
	semicolonCount := strings.Count(line, ";")
	andCount := strings.Count(line, "&&")
	orCount := strings.Count(line, "||")
	pipeCount := strings.Count(line, "|")
	commandSeparators := semicolonCount + andCount + orCount + pipeCount
	if pipeCount >= 3 {
		return true
	}
	if commandSeparators >= 5 {
		return true
	}
	if length > 220 && commandSeparators >= 2 {
		return true
	}
	return false
}

func commandStartsWithKnownExecutable(line string) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return false
	}
	first := strings.ToLower(strings.Trim(fields[0], "\"'`"))
	known := map[string]struct{}{
		"git": {}, "go": {}, "docker": {}, "winget": {}, "npm": {}, "pnpm": {}, "yarn": {}, "pip": {}, "python": {},
		"curl": {}, "wget": {}, "irm": {}, "iwr": {}, "kubectl": {}, "cargo": {}, "make": {}, "cmake": {}, "dotnet": {},
		"powershell": {}, "pwsh": {}, "select-string": {}, "get-content": {}, "get-childitem": {}, "set-location": {},
		"cat": {}, "grep": {}, "findstr": {}, "ls": {}, "cd": {}, "rm": {}, "del": {}, "code": {}, "write-host": {},
		"opencode": {}, "claude": {}, "codex": {}, "ssh": {}, "scp": {},
	}
	_, ok := known[first]
	return ok
}

func startsWithEmojiOrPrompt(line string) bool {
	if strings.HasPrefix(line, "PS>") || strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "$ ") {
		return true
	}
	r, _ := utf8DecodeRuneInString(line)
	if r == 0 {
		return false
	}
	return unicode.Is(unicode.So, r)
}

func containsMostlyHanText(line string) bool {
	var hanCount int
	var letterOrDigitCount int
	for _, r := range line {
		if unicode.Is(unicode.Han, r) {
			hanCount++
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letterOrDigitCount++
		}
	}
	return hanCount >= 2 && letterOrDigitCount == hanCount
}

func hasCommandSeparator(in string) bool {
	return strings.Contains(in, ";") || strings.Contains(in, "&&") || strings.Contains(in, "||") || strings.Contains(in, "|")
}

func isSeparatorByte(b byte) bool {
	switch b {
	case ' ', '\t', ';', '|', '&':
		return true
	default:
		return false
	}
}

func utf8DecodeRuneInString(value string) (rune, int) {
	for _, r := range value {
		return r, len(string(r))
	}
	return 0, 0
}

func countWholeWord(in string, target string) int {
	count := 0
	start := 0
	for {
		index := strings.Index(in[start:], target)
		if index < 0 {
			return count
		}
		index += start
		beforeOK := index == 0 || !isWordRune(rune(in[index-1]))
		afterIndex := index + len(target)
		afterOK := afterIndex >= len(in) || !isWordRune(rune(in[afterIndex]))
		if beforeOK && afterOK {
			count++
		}
		start = index + len(target)
	}
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

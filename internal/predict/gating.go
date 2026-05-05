package predict

import (
	"strings"

	"github.com/sanqiu/cliai/internal/project"
)

type CommandFamily string

const CommandFamilyUnknown CommandFamily = ""

type GateDecision struct {
	Allowed bool
	Reason  string
}

func candidateEligibility(query string, command string, source string, ctx project.Context) GateDecision {
	_ = ctx
	if source == "history" && (looksLikeBareURLCandidate(command) || looksLikeConcatenatedCommandCandidate(command) || looksLikeCommandOutputCandidate(command)) {
		switch {
		case looksLikeBareURLCandidate(command):
			return GateDecision{Allowed: false, Reason: "rejected by gate: bare URL history entry"}
		case looksLikeConcatenatedCommandCandidate(command):
			return GateDecision{Allowed: false, Reason: "rejected by gate: concatenated command history entry"}
		default:
			return GateDecision{Allowed: false, Reason: "rejected by gate: looks like shell output text"}
		}
	}

	family := queryCommandFamily(query)
	subverb := querySubverbPrefix(query)
	if family == CommandFamilyUnknown || subverb == "" {
		return GateDecision{Allowed: true}
	}

	seenFamily := false
	for _, pattern := range extractCommandPatterns(command) {
		if CommandFamily(pattern.family) != family {
			continue
		}
		seenFamily = true
		if commandVerbMatches(subverb, pattern.verb) {
			return GateDecision{Allowed: true}
		}
	}
	if seenFamily {
		return GateDecision{Allowed: false, Reason: "rejected by gate: subcommand mismatch for explicit command prefix"}
	}
	return GateDecision{Allowed: false, Reason: "rejected by gate: command family mismatch for explicit command prefix"}
}

func commandFamily(command string) CommandFamily {
	patterns := extractCommandPatterns(command)
	if len(patterns) == 0 {
		return CommandFamilyUnknown
	}
	return CommandFamily(patterns[0].family)
}

func queryCommandFamily(query string) CommandFamily {
	pattern, ok := parseExplicitCommandPattern(query)
	if !ok {
		return CommandFamilyUnknown
	}
	return CommandFamily(pattern.family)
}

func commandSubverb(command string) string {
	patterns := extractCommandPatterns(command)
	if len(patterns) == 0 {
		return ""
	}
	return patterns[0].verb
}

func querySubverbPrefix(query string) string {
	pattern, ok := parseExplicitCommandPattern(query)
	if !ok {
		return ""
	}
	return pattern.verb
}

func extractCommandPatterns(command string) []commandPattern {
	segments := splitCommandSegments(command)
	patterns := make([]commandPattern, 0, len(segments))
	for _, segment := range segments {
		pattern, ok := parseCommandPattern(segment)
		if ok {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

func splitCommandSegments(command string) []string {
	replacer := strings.NewReplacer("&&", ";", "||", ";", "|", ";")
	normalized := replacer.Replace(command)
	parts := strings.Split(normalized, ";")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

func looksLikeBareURLCandidate(command string) bool {
	trimmed := strings.Trim(strings.TrimSpace(command), "\"'`")
	return (strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) &&
		!strings.ContainsAny(trimmed, " \t")
}

func looksLikeConcatenatedCommandCandidate(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	for _, phrase := range []string{
		"winget install",
		"cliai predict",
		"npm install",
		"pnpm install",
		"npm run",
		"go install",
		"git clone",
	} {
		first := strings.Index(lower, phrase)
		if first < 0 {
			continue
		}
		second := strings.Index(lower[first+len(phrase):], phrase)
		if second < 0 {
			continue
		}
		second += first + len(phrase)
		if second == 0 || lower[second-1] == ' ' || lower[second-1] == ';' || lower[second-1] == '|' || lower[second-1] == '&' {
			continue
		}
		return true
	}
	return false
}

func looksLikeCommandOutputCandidate(command string) bool {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" || commandStartsWithKnownFamily(trimmed) || looksLikeBareURLCandidate(trimmed) {
		return false
	}
	if containsHan(trimmed) && !strings.ContainsAny(trimmed, "/\\-_.") {
		return true
	}
	if strings.HasPrefix(trimmed, "✅") || strings.HasPrefix(trimmed, "❌") || strings.HasPrefix(trimmed, "⚠") {
		return true
	}
	return strings.Contains(trimmed, "安装完成") || strings.Contains(trimmed, "重新打开") || strings.Contains(trimmed, "->")
}

func commandStartsWithKnownFamily(command string) bool {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(command)))
	if len(fields) == 0 {
		return false
	}
	return isKnownCommandFamily(fields[0])
}

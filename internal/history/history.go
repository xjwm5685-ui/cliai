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
)

type Entry struct {
	Command  string    `json:"command"`
	Count    int       `json:"count"`
	LastUsed time.Time `json:"last_used"`
	Source   string    `json:"source"`
}

func ImportPowerShell(path string, limit int) ([]Entry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	type aggregate struct {
		count int
		order int
	}

	lineNo := 0
	agg := map[string]aggregate{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lineNo++
		line, ok := sanitizeCommand(scanner.Text())
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
			Source:   "powershell-history",
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

	lower := strings.ToLower(line)
	sensitiveMarkers := []string{
		"authorization",
		"api_key",
		"apikey",
		"password",
		"passwd",
		"secret",
		"bearer ",
		"token=",
		"token ",
		"sk-",
	}
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return "", false
		}
	}

	if strings.Count(line, "\x00") > 0 {
		return "", false
	}
	return line, true
}

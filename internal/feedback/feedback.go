package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Query        string    `json:"query"`
	QueryKey     string    `json:"query_key"`
	Command      string    `json:"command"`
	Count        int       `json:"count"`
	LastAccepted time.Time `json:"last_accepted"`
}

func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse feedback: %w", err)
	}
	return entries, nil
}

func Save(path string, entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Record(path string, query string, command string) error {
	entries, err := Load(path)
	if err != nil {
		return err
	}

	queryKey := normalizeQuery(query)
	now := time.Now()
	for index := range entries {
		if entries[index].QueryKey == queryKey && entries[index].Command == command {
			entries[index].Count++
			entries[index].LastAccepted = now
			return Save(path, entries)
		}
	}

	entries = append(entries, Entry{
		Query:        strings.TrimSpace(query),
		QueryKey:     queryKey,
		Command:      command,
		Count:        1,
		LastAccepted: now,
	})

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].LastAccepted.After(entries[j].LastAccepted)
		}
		return entries[i].Count > entries[j].Count
	})
	return Save(path, entries)
}

func CommandBonuses(query string, entries []Entry) map[string]float64 {
	bonuses := make(map[string]float64)
	queryKey := normalizeQuery(query)
	for _, entry := range entries {
		if strings.TrimSpace(entry.Command) == "" {
			continue
		}

		bonus := 0.0
		if entry.QueryKey == queryKey {
			bonus += float64(entry.Count) * 20
		} else if queryKey != "" && strings.Contains(entry.QueryKey, queryKey) {
			bonus += float64(entry.Count) * 8
		}

		if entry.LastAccepted.After(time.Now().Add(-7 * 24 * time.Hour)) {
			bonus += 4
		}
		if bonus > 0 {
			bonuses[entry.Command] += bonus
		}
	}
	return bonuses
}

func normalizeQuery(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	query = strings.NewReplacer("\\", " ", "/", " ", "_", " ", "-", " ").Replace(query)
	return strings.Join(strings.Fields(query), " ")
}

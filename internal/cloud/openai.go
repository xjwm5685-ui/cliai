package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sanqiu/cliai/internal/config"
	"github.com/sanqiu/cliai/internal/predict"
)

type Client struct {
	httpClient *http.Client
	cfg        config.OpenAIConfig
}

func New(cfg config.OpenAIConfig) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		cfg:        cfg,
	}
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled && strings.TrimSpace(c.cfg.APIKey) != "" && strings.TrimSpace(c.cfg.Model) != ""
}

func (c *Client) Rerank(ctx context.Context, query string, local []predict.Candidate) ([]predict.Candidate, error) {
	if !c.Enabled() || len(local) == 0 {
		return local, nil
	}

	type selection struct {
		Index  int    `json:"index"`
		Reason string `json:"reason"`
	}
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	body := map[string]any{
		"model": c.cfg.Model,
		"messages": []message{
			{
				Role: "system",
				Content: "You are a command ranking engine for Windows PowerShell. " +
					"Return only JSON. Never invent new commands. " +
					"Only choose from the provided candidate indexes and explain the ranking briefly.",
			},
			{
				Role:    "user",
				Content: buildPrompt(query, local),
			},
		},
		"temperature": 0.1,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return local, err
	}

	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return local, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return local, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return local, err
	}
	if resp.StatusCode >= 300 {
		return local, fmt.Errorf("cloud rerank failed: %s", strings.TrimSpace(string(data)))
	}

	var raw struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return local, err
	}
	if len(raw.Choices) == 0 {
		return local, fmt.Errorf("cloud rerank returned no choices")
	}

	content := raw.Choices[0].Message.Content
	parsed, err := parseSelections(content)
	if err != nil || len(parsed) == 0 {
		return local, err
	}

	selected := make([]predict.Candidate, 0, len(local))
	used := make(map[int]struct{}, len(parsed))
	for rank, item := range parsed {
		if item.Index < 1 || item.Index > len(local) {
			continue
		}
		if _, ok := used[item.Index]; ok {
			continue
		}
		used[item.Index] = struct{}{}

		current := local[item.Index-1]
		reason := strings.TrimSpace(item.Reason)
		if reason == "" {
			reason = current.Reason
		}
		selected = append(selected, predict.Candidate{
			Command: current.Command,
			Reason:  reason,
			Source:  "cloud",
			Score:   float64(200 - rank),
		})
	}
	if len(selected) == 0 {
		return local, nil
	}

	for index, candidate := range local {
		if _, ok := used[index+1]; ok {
			continue
		}
		selected = append(selected, candidate)
	}
	return selected, nil
}

func buildPrompt(query string, local []predict.Candidate) string {
	var b strings.Builder
	b.WriteString("Query:\n")
	b.WriteString(query)
	b.WriteString("\n\nCandidates:\n")
	for i, candidate := range local {
		fmt.Fprintf(&b, "%d. %s | %s | source=%s\n", i+1, candidate.Command, candidate.Reason, candidate.Source)
	}
	b.WriteString("\nReturn a JSON array like [{\"index\":1,\"reason\":\"best match because ...\"}] with up to 5 items.")
	return b.String()
}

func parseSelections(content string) ([]struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end < start {
		return nil, fmt.Errorf("response did not contain json array")
	}

	var items []struct {
		Index  int    `json:"index"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}

package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sanqiu/cliai/internal/config"
	"github.com/sanqiu/cliai/internal/predict"
)

func TestRerankOnlySelectsExistingLocalCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": `[{"index":2,"reason":"better"},{"index":99,"reason":"invalid"}]`,
					},
				},
			},
		})
	}))
	defer server.Close()

	client := New(config.OpenAIConfig{
		Enabled: true,
		BaseURL: server.URL,
		APIKey:  "test",
		Model:   "test-model",
	})

	local := []predict.Candidate{
		{Command: "git status", Reason: "local 1", Source: "builtin", Score: 90},
		{Command: "git stash", Reason: "local 2", Source: "builtin", Score: 80},
	}

	got, err := client.Rerank(context.Background(), "git st", local)
	if err != nil {
		t.Fatalf("Rerank returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].Command != "git stash" {
		t.Fatalf("expected first command to be existing local candidate, got %q", got[0].Command)
	}
	if got[1].Command != "git status" {
		t.Fatalf("expected remaining local candidate to be preserved, got %q", got[1].Command)
	}
}

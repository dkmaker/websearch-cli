// internal/provider/perplexity_test.go
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPerplexityName(t *testing.T) {
	p := NewPerplexity("test-key", "")
	if p.Name() != "perplexity" {
		t.Errorf("expected 'perplexity', got %q", p.Name())
	}
}

func TestPerplexitySupportedModes(t *testing.T) {
	p := NewPerplexity("test-key", "")
	modes := p.SupportedModes()
	expected := map[string]bool{"ask": true, "search": true, "reason": true, "research": true}
	for _, m := range modes {
		if !expected[m] {
			t.Errorf("unexpected mode %q", m)
		}
	}
	if len(modes) != len(expected) {
		t.Errorf("expected %d modes, got %d", len(expected), len(modes))
	}
}

func TestPerplexitySearch(t *testing.T) {
	mockResp := map[string]interface{}{
		"id":    "test-id",
		"model": "sonar",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": "Test answer content",
				},
				"finish_reason": "stop",
			},
		},
		"citations": []string{
			"https://example.com/1",
			"https://example.com/2",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("expected Authorization header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["model"] != "sonar" {
			t.Errorf("expected model 'sonar', got %v", body["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewPerplexity("test-key", server.URL)
	result, err := p.Search(context.Background(), "test query", SearchOptions{
		Mode:         "ask",
		SystemPrompt: "Be concise",
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Content != "Test answer content" {
		t.Errorf("expected 'Test answer content', got %q", result.Content)
	}
	if len(result.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.Sources))
	}
	if result.Provider != "perplexity" {
		t.Errorf("expected provider 'perplexity', got %q", result.Provider)
	}
}

func TestPerplexityModeToModel(t *testing.T) {
	tests := []struct {
		mode  string
		model string
	}{
		{"ask", "sonar"},
		{"search", "sonar-pro"},
		{"reason", "sonar-reasoning-pro"},
		{"research", "sonar-deep-research"},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := modeToModel(tt.mode)
			if got != tt.model {
				t.Errorf("modeToModel(%q) = %q, want %q", tt.mode, got, tt.model)
			}
		})
	}
}

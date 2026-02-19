// internal/provider/brave_test.go
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBraveName(t *testing.T) {
	p := NewBrave("test-key", "")
	if p.Name() != "brave" {
		t.Errorf("expected 'brave', got %q", p.Name())
	}
}

func TestBraveSupportedModes(t *testing.T) {
	p := NewBrave("test-key", "")
	modes := p.SupportedModes()
	expected := map[string]bool{"web": true}
	for _, m := range modes {
		if !expected[m] {
			t.Errorf("unexpected mode %q", m)
		}
	}
}

func TestBraveSearch(t *testing.T) {
	mockResp := map[string]interface{}{
		"web": map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":       "Result 1",
					"url":         "https://example.com/1",
					"description": "First result description",
				},
				{
					"title":       "Result 2",
					"url":         "https://example.com/2",
					"description": "Second result description",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			t.Error("expected X-Subscription-Token header")
		}
		q := r.URL.Query().Get("q")
		if q != "test query" {
			t.Errorf("expected query 'test query', got %q", q)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewBrave("test-key", server.URL)
	result, err := p.Search(context.Background(), "test query", SearchOptions{
		Mode:       "web",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Provider != "brave" {
		t.Errorf("expected provider 'brave', got %q", result.Provider)
	}
	if len(result.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.Sources))
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

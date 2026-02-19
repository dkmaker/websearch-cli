package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubName(t *testing.T) {
	p := NewGitHub("", "")
	if p.Name() != "github" {
		t.Errorf("expected 'github', got %q", p.Name())
	}
}

func TestGitHubSupportedModes(t *testing.T) {
	p := NewGitHub("", "")
	modes := p.SupportedModes()
	expected := map[string]bool{"repos": true, "code": true, "issues": true}
	if len(modes) != 3 {
		t.Errorf("expected 3 modes, got %d", len(modes))
	}
	for _, m := range modes {
		if !expected[m] {
			t.Errorf("unexpected mode %q", m)
		}
	}
}

func TestGitHubSearchRepos(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        2,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"full_name":        "golang/go",
				"html_url":         "https://github.com/golang/go",
				"description":      "The Go programming language",
				"stargazers_count": 125000,
				"language":         "Go",
				"updated_at":       "2026-02-19T10:00:00Z",
				"topics":           []string{"go", "programming-language"},
			},
			{
				"full_name":        "gin-gonic/gin",
				"html_url":         "https://github.com/gin-gonic/gin",
				"description":      "Gin is a HTTP web framework written in Go",
				"stargazers_count": 80000,
				"language":         "Go",
				"updated_at":       "2026-02-18T10:00:00Z",
				"topics":           []string{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/repositories" {
			t.Errorf("expected /search/repositories, got %s", r.URL.Path)
		}
		q := r.URL.Query().Get("q")
		if q != "go web framework" {
			t.Errorf("expected query 'go web framework', got %q", q)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	result, err := p.Search(context.Background(), "go web framework", SearchOptions{
		Mode:       "repos",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Provider != "github" {
		t.Errorf("expected provider 'github', got %q", result.Provider)
	}
	if result.Mode != "repos" {
		t.Errorf("expected mode 'repos', got %q", result.Mode)
	}
	if len(result.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.Sources))
	}
	if !strings.Contains(result.Content, "golang/go") {
		t.Error("expected golang/go in content")
	}
	if !strings.Contains(result.Content, "125k") {
		t.Error("expected star count in content")
	}
	if !strings.Contains(result.Content, "Go") {
		t.Error("expected language in content")
	}
}

// internal/provider/brave_test.go
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCleanBraveHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strong tags to markdown bold",
			input: "<strong>bold text</strong>",
			want:  "**bold text**",
		},
		{
			name:  "em tags to markdown italic",
			input: "<em>italic text</em>",
			want:  "*italic text*",
		},
		{
			name:  "mixed HTML tags and entities",
			input: "<strong>Go</strong> &amp; <em>Rust</em> are &quot;fast&quot;",
			want:  "**Go** & *Rust* are \"fast\"",
		},
		{
			name:  "no HTML passes through unchanged",
			input: "plain text with no markup",
			want:  "plain text with no markup",
		},
		{
			name:  "unknown tags are stripped",
			input: "text with <span class=\"hl\">highlighted</span> and <br/>break",
			want:  "text with highlighted and break",
		},
		{
			name:  "nested tags",
			input: "<strong><em>bold italic</em></strong>",
			want:  "***bold italic***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanBraveHTML(tt.input)
			if got != tt.want {
				t.Errorf("cleanBraveHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBraveSearchHTMLEntities(t *testing.T) {
	mockResp := map[string]interface{}{
		"web": map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":       "Go &amp; Rust &mdash; A Comparison",
					"url":         "https://example.com/1",
					"description": "Find the best &quot;Go&quot; tips &amp; tricks. Use &#39;generics&#39; for &lt;type&gt; safety.",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewBrave("test-key", server.URL)
	result, err := p.Search(context.Background(), "test", SearchOptions{Mode: "web"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	// Content should have decoded HTML entities
	if strings.Contains(result.Content, "&amp;") {
		t.Errorf("content still contains &amp;: %s", result.Content)
	}
	if strings.Contains(result.Content, "&quot;") {
		t.Errorf("content still contains &quot;: %s", result.Content)
	}
	if strings.Contains(result.Content, "&#39;") {
		t.Errorf("content still contains &#39;: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"Go"`) {
		t.Errorf("expected decoded quotes in content: %s", result.Content)
	}

	// Source title should also be decoded
	if result.Sources[0].Title != "Go & Rust — A Comparison" {
		t.Errorf("expected decoded title, got %q", result.Sources[0].Title)
	}
}

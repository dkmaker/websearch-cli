// internal/output/formatter_test.go
package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dkmaker/websearch/internal/provider"
)

func TestFormatMarkdownNoSources(t *testing.T) {
	result := &provider.Result{
		Content:  "This is the answer.",
		Provider: "perplexity",
		Mode:     "ask",
		Sources: []provider.Source{
			{Title: "Source 1", URL: "https://example.com/1"},
		},
	}
	out := FormatMarkdown(result, false)
	if !strings.Contains(out, "This is the answer.") {
		t.Error("expected content in output")
	}
	if strings.Contains(out, "example.com") {
		t.Error("sources should not appear when includeSources is false")
	}
	if strings.Contains(out, "<!--") {
		t.Error("should not contain HTML comment hint")
	}
}

func TestFormatMarkdownWithSources(t *testing.T) {
	result := &provider.Result{
		Content: "This is the answer.",
		Sources: []provider.Source{
			{Title: "Source 1", URL: "https://example.com/1"},
			{Title: "Source 2", URL: "https://example.com/2"},
		},
	}
	out := FormatMarkdown(result, true)
	if !strings.Contains(out, "This is the answer.") {
		t.Error("expected content in output")
	}
	if !strings.Contains(out, "https://example.com/1") {
		t.Error("expected source URL in output")
	}
	if strings.Contains(out, "--include-sources") {
		t.Error("hint should not appear when sources are included")
	}
}

func TestFormatJSON(t *testing.T) {
	result := &provider.Result{
		Content:  "answer",
		Provider: "brave",
		Mode:     "web",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com"},
		},
	}
	out, err := FormatJSON(result, false)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed["content"] != "answer" {
		t.Errorf("expected content 'answer', got %v", parsed["content"])
	}
	if parsed["provider"] != "brave" {
		t.Errorf("expected provider 'brave', got %v", parsed["provider"])
	}
}

func TestFormatJSONWithSources(t *testing.T) {
	result := &provider.Result{
		Content: "answer",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com"},
		},
	}
	out, err := FormatJSON(result, true)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(out), &parsed)
	sources, ok := parsed["sources"].([]interface{})
	if !ok || len(sources) != 1 {
		t.Error("expected 1 source in JSON output")
	}
}

func TestFormatMarkdownNoSourcesField(t *testing.T) {
	result := &provider.Result{
		Content: "No sources here.",
	}
	out := FormatMarkdown(result, false)
	if !strings.Contains(out, "No sources here.") {
		t.Error("expected content")
	}
	if strings.Contains(out, "<!--") {
		t.Error("no hint expected when result has no sources")
	}
}

func TestFormatMarkdownStripsCitationRefs(t *testing.T) {
	result := &provider.Result{
		Content: "Go is a language [1] with great concurrency [2, 3] support.",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
		},
	}
	out := FormatMarkdown(result, false)
	if strings.Contains(out, "[1]") {
		t.Error("expected [1] to be stripped")
	}
	if strings.Contains(out, "[2, 3]") {
		t.Error("expected [2, 3] to be stripped")
	}
	if !strings.Contains(out, "Go is a language with great concurrency support.") {
		t.Errorf("expected clean text, got: %s", out)
	}
}

func TestFormatMarkdownKeepsCitationRefsWithSources(t *testing.T) {
	result := &provider.Result{
		Content: "Go is a language [1] with great support.",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
		},
	}
	out := FormatMarkdown(result, true)
	if !strings.Contains(out, "[1]") {
		t.Error("expected [1] to be preserved when sources included")
	}
}

func TestFormatJSONStripsCitationRefs(t *testing.T) {
	result := &provider.Result{
		Content:  "Answer [1] here [2, 3].",
		Provider: "perplexity",
		Mode:     "ask",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com"},
		},
	}
	out, err := FormatJSON(result, false)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}
	if strings.Contains(out, "[1]") {
		t.Error("expected [1] to be stripped from JSON content")
	}
}

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
	out := FormatMarkdown(result, false, false)
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
	out := FormatMarkdown(result, true, false)
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
	out, err := FormatJSON(result, false, false)
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
	out, err := FormatJSON(result, true, false)
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
	out := FormatMarkdown(result, false, false)
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
	out := FormatMarkdown(result, false, false)
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
	out := FormatMarkdown(result, true, false)
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
	out, err := FormatJSON(result, false, false)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}
	if strings.Contains(out, "[1]") {
		t.Error("expected [1] to be stripped from JSON content")
	}
}

// --- Metadata tests ---

func TestFormatMarkdownWithMetadata(t *testing.T) {
	result := &provider.Result{
		Content:      "The answer.",
		Provider:     "perplexity",
		Mode:         "reason",
		FinishReason: "stop",
		ModeAdjusted: false,
		Cached:       false,
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
			{Title: "S2", URL: "https://example.com/2"},
		},
	}
	out := FormatMarkdown(result, false, true)

	// Should start with YAML frontmatter
	if !strings.HasPrefix(out, "---\n") {
		t.Error("expected output to start with YAML frontmatter delimiter")
	}
	if !strings.Contains(out, "provider: perplexity") {
		t.Error("expected provider in frontmatter")
	}
	if !strings.Contains(out, "mode: reason") {
		t.Error("expected mode in frontmatter")
	}
	if !strings.Contains(out, "mode_adjusted: false") {
		t.Error("expected mode_adjusted in frontmatter")
	}
	if !strings.Contains(out, "truncated: false") {
		t.Error("expected truncated: false in frontmatter")
	}
	if !strings.Contains(out, "sources_count: 2") {
		t.Error("expected sources_count: 2 in frontmatter")
	}
	if !strings.Contains(out, "cached: false") {
		t.Error("expected cached: false in frontmatter")
	}
	// Content should still be present after frontmatter
	if !strings.Contains(out, "The answer.") {
		t.Error("expected content after frontmatter")
	}
}

func TestFormatMarkdownWithoutMetadata(t *testing.T) {
	result := &provider.Result{
		Content:  "The answer.",
		Provider: "perplexity",
		Mode:     "ask",
	}
	out := FormatMarkdown(result, false, false)

	if strings.HasPrefix(out, "---\n") {
		t.Error("expected no YAML frontmatter when metadata is disabled")
	}
	if strings.Contains(out, "provider:") {
		t.Error("expected no metadata fields when metadata is disabled")
	}
}

func TestFormatMarkdownTruncatedTrue(t *testing.T) {
	result := &provider.Result{
		Content:      "Truncated response...",
		Provider:     "perplexity",
		Mode:         "research",
		FinishReason: "length",
	}
	out := FormatMarkdown(result, false, true)

	if !strings.Contains(out, "truncated: true") {
		t.Error("expected truncated: true when FinishReason is 'length'")
	}
}

func TestFormatMarkdownModeAdjusted(t *testing.T) {
	result := &provider.Result{
		Content:      "Web results.",
		Provider:     "brave",
		Mode:         "web",
		ModeAdjusted: true,
	}
	out := FormatMarkdown(result, false, true)

	if !strings.Contains(out, "mode_adjusted: true") {
		t.Error("expected mode_adjusted: true in frontmatter")
	}
}

func TestFormatJSONWithMetadata(t *testing.T) {
	result := &provider.Result{
		Content:      "answer",
		Provider:     "perplexity",
		Mode:         "search",
		FinishReason: "stop",
		ModeAdjusted: true,
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
			{Title: "S2", URL: "https://example.com/2"},
			{Title: "S3", URL: "https://example.com/3"},
		},
	}
	out, err := FormatJSON(result, false, true)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["mode_adjusted"] != true {
		t.Errorf("expected mode_adjusted true, got %v", parsed["mode_adjusted"])
	}
	if parsed["truncated"] != false {
		t.Errorf("expected truncated false, got %v", parsed["truncated"])
	}
	if parsed["sources_count"] != float64(3) {
		t.Errorf("expected sources_count 3, got %v", parsed["sources_count"])
	}
}

func TestFormatJSONWithoutMetadata(t *testing.T) {
	result := &provider.Result{
		Content:      "answer",
		Provider:     "perplexity",
		Mode:         "ask",
		ModeAdjusted: true,
		FinishReason: "length",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
		},
	}
	out, err := FormatJSON(result, false, false)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(out), &parsed)

	// mode_adjusted and truncated should be false (zero values) when metadata is disabled
	if parsed["mode_adjusted"] != false {
		t.Errorf("expected mode_adjusted false when metadata disabled, got %v", parsed["mode_adjusted"])
	}
	if parsed["truncated"] != false {
		t.Errorf("expected truncated false when metadata disabled, got %v", parsed["truncated"])
	}
	if parsed["sources_count"] != float64(0) {
		t.Errorf("expected sources_count 0 when metadata disabled, got %v", parsed["sources_count"])
	}
}

func TestFormatJSONTruncatedTrue(t *testing.T) {
	result := &provider.Result{
		Content:      "answer",
		Provider:     "perplexity",
		Mode:         "research",
		FinishReason: "length",
	}
	out, err := FormatJSON(result, false, true)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(out), &parsed)

	if parsed["truncated"] != true {
		t.Errorf("expected truncated true when FinishReason is 'length', got %v", parsed["truncated"])
	}
}

func TestFormatJSONSourcesCountMatchesSources(t *testing.T) {
	result := &provider.Result{
		Content:  "answer",
		Provider: "perplexity",
		Mode:     "ask",
		Sources: []provider.Source{
			{Title: "S1", URL: "https://example.com/1"},
			{Title: "S2", URL: "https://example.com/2"},
			{Title: "S3", URL: "https://example.com/3"},
			{Title: "S4", URL: "https://example.com/4"},
			{Title: "S5", URL: "https://example.com/5"},
		},
	}
	out, err := FormatJSON(result, true, true)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(out), &parsed)

	sources := parsed["sources"].([]interface{})
	sourcesCount := int(parsed["sources_count"].(float64))

	if sourcesCount != len(sources) {
		t.Errorf("sources_count (%d) does not match actual sources length (%d)", sourcesCount, len(sources))
	}
}

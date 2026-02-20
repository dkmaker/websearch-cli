// internal/provider/provider_test.go
package provider

import (
	"testing"
)

func TestSearchOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    SearchOptions
		wantErr bool
	}{
		{
			name:    "valid options",
			opts:    SearchOptions{Mode: "ask", MaxTokens: 1024},
			wantErr: false,
		},
		{
			name:    "empty mode is valid (uses profile default)",
			opts:    SearchOptions{},
			wantErr: false,
		},
		{
			name:    "negative max tokens",
			opts:    SearchOptions{MaxTokens: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitHubQualifiersIsEmpty(t *testing.T) {
	q := GitHubQualifiers{}
	if !q.IsEmpty() {
		t.Error("empty qualifiers should return true")
	}
	q.Language = "go"
	if q.IsEmpty() {
		t.Error("qualifiers with language set should not be empty")
	}
}

func TestGitHubQualifiersCacheKey(t *testing.T) {
	q := GitHubQualifiers{Language: "go", Stars: ">100"}
	key := q.CacheKey()
	if key == "" {
		t.Error("expected non-empty cache key")
	}
	q2 := GitHubQualifiers{Language: "python", Stars: ">100"}
	if q.CacheKey() == q2.CacheKey() {
		t.Error("different qualifiers should produce different cache keys")
	}
	q3 := GitHubQualifiers{}
	if q3.CacheKey() != "" {
		t.Error("empty qualifiers should produce empty cache key")
	}
}

func TestResultHasSources(t *testing.T) {
	r := &Result{
		Content: "answer text",
		Sources: []Source{{Title: "src", URL: "https://example.com"}},
	}
	if len(r.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(r.Sources))
	}

	empty := &Result{Content: "no sources"}
	if len(empty.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(empty.Sources))
	}
}

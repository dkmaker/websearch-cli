// internal/provider/provider.go
package provider

import (
	"context"
	"fmt"
	"strings"
)

// Provider is the interface that search providers must implement.
type Provider interface {
	Search(ctx context.Context, query string, opts SearchOptions) (*Result, error)
	Name() string
	SupportedModes() []string
}

// GitHubQualifiers holds GitHub-specific search qualifiers exposed as CLI flags.
type GitHubQualifiers struct {
	Language  string
	User      string
	Org       string
	Repo      string
	Stars     string
	Topic     string
	License   string
	Archived  string
	Fork      string
	Pushed    string
	Created   string
	Sort      string
	Filename  string
	Extension string
	Path      string
	State     string
	Label     string
	Author    string
	Assignee  string
	In        string
}

// IsEmpty returns true if no qualifiers are set.
func (q GitHubQualifiers) IsEmpty() bool {
	return q == GitHubQualifiers{}
}

// CacheKey returns a deterministic string representation for cache keying.
// Returns empty string if no qualifiers are set.
func (q GitHubQualifiers) CacheKey() string {
	if q.IsEmpty() {
		return ""
	}
	parts := []string{
		q.Language, q.User, q.Org, q.Repo, q.Stars, q.Topic,
		q.License, q.Archived, q.Fork, q.Pushed, q.Created, q.Sort,
		q.Filename, q.Extension, q.Path, q.State, q.Label, q.Author,
		q.Assignee, q.In,
	}
	return strings.Join(parts, "\x00")
}

// SearchOptions configures a search request.
type SearchOptions struct {
	Mode              string
	MaxTokens         int
	MaxResults        int
	SystemPrompt      string
	DomainFilter      []string
	RecencyFilter     string
	IncludeSources    bool
	SearchContextSize string
	GitHub            GitHubQualifiers
}

// Validate checks that search options are valid.
func (o SearchOptions) Validate() error {
	if o.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be non-negative, got %d", o.MaxTokens)
	}
	if o.MaxResults < 0 {
		return fmt.Errorf("max_results must be non-negative, got %d", o.MaxResults)
	}
	return nil
}

// Result holds the response from a search provider.
type Result struct {
	Content      string   `json:"content"`
	Sources      []Source `json:"sources,omitempty"`
	Provider     string   `json:"provider"`
	Mode         string   `json:"mode"`
	Cached       bool     `json:"cached"`
	FinishReason string   `json:"finish_reason,omitempty"` // "stop" or "length" from API
	ModeAdjusted bool     `json:"mode_adjusted"`           // was mode changed from what was requested
}

// Source represents a citation/reference.
type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Date  string `json:"date,omitempty"`
}

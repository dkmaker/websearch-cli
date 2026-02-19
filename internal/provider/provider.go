// internal/provider/provider.go
package provider

import (
	"context"
	"fmt"
)

// Provider is the interface that search providers must implement.
type Provider interface {
	Search(ctx context.Context, query string, opts SearchOptions) (*Result, error)
	Name() string
	SupportedModes() []string
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
	Content  string   `json:"content"`
	Sources  []Source `json:"sources,omitempty"`
	Provider string   `json:"provider"`
	Mode     string   `json:"mode"`
	Cached   bool     `json:"cached"`
}

// Source represents a citation/reference.
type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Date  string `json:"date,omitempty"`
}

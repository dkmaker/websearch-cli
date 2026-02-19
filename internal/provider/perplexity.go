// internal/provider/perplexity.go
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultPerplexityURL = "https://api.perplexity.ai"

// Perplexity implements the Provider interface for the Perplexity API.
type Perplexity struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewPerplexity creates a new Perplexity provider. If baseURL is empty,
// the default Perplexity API URL is used.
func NewPerplexity(apiKey, baseURL string) *Perplexity {
	if baseURL == "" {
		baseURL = defaultPerplexityURL
	}
	return &Perplexity{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Name returns the provider name.
func (p *Perplexity) Name() string {
	return "perplexity"
}

// SupportedModes returns the modes this provider supports.
func (p *Perplexity) SupportedModes() []string {
	return []string{"ask", "search", "reason", "research"}
}

// modeToModel maps a search mode to a Perplexity model name.
func modeToModel(mode string) string {
	switch mode {
	case "ask":
		return "sonar"
	case "search":
		return "sonar-pro"
	case "reason":
		return "sonar-reasoning-pro"
	case "research":
		return "sonar-deep-research"
	default:
		return "sonar"
	}
}

type perplexityRequest struct {
	Model               string              `json:"model"`
	Messages            []perplexityMessage `json:"messages"`
	MaxTokens           int                 `json:"max_tokens,omitempty"`
	SearchDomainFilter  []string            `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter string              `json:"search_recency_filter,omitempty"`
	SearchContextSize   string              `json:"search_context_size,omitempty"`
	Stream              bool                `json:"stream"`
}

type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type perplexityResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Citations     []string               `json:"citations"`
	SearchResults []perplexitySearchResult `json:"search_results"`
}

type perplexitySearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Date  string `json:"date"`
}

// Search executes a search query against the Perplexity API.
func (p *Perplexity) Search(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	mode := opts.Mode
	if mode == "" {
		mode = "ask"
	}

	messages := []perplexityMessage{}
	if opts.SystemPrompt != "" {
		messages = append(messages, perplexityMessage{Role: "system", Content: opts.SystemPrompt})
	}
	messages = append(messages, perplexityMessage{Role: "user", Content: query})

	reqBody := perplexityRequest{
		Model:    modeToModel(mode),
		Messages: messages,
		Stream:   false,
	}
	if opts.MaxTokens > 0 {
		reqBody.MaxTokens = opts.MaxTokens
	}
	if len(opts.DomainFilter) > 0 {
		reqBody.SearchDomainFilter = opts.DomainFilter
	}
	if opts.RecencyFilter != "" {
		reqBody.SearchRecencyFilter = opts.RecencyFilter
	}
	if opts.SearchContextSize != "" {
		reqBody.SearchContextSize = opts.SearchContextSize
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("perplexity API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var pResp perplexityResponse
	if err := json.Unmarshal(respBody, &pResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if len(pResp.Choices) == 0 {
		return nil, fmt.Errorf("perplexity returned no choices")
	}

	result := &Result{
		Content:      pResp.Choices[0].Message.Content,
		Provider:     "perplexity",
		Mode:         mode,
		FinishReason: pResp.Choices[0].FinishReason,
	}

	seen := make(map[string]bool)
	for _, sr := range pResp.SearchResults {
		if !seen[sr.URL] {
			result.Sources = append(result.Sources, Source{
				Title: sr.Title,
				URL:   sr.URL,
				Date:  sr.Date,
			})
			seen[sr.URL] = true
		}
	}
	for _, cite := range pResp.Citations {
		if !seen[cite] {
			result.Sources = append(result.Sources, Source{URL: cite})
			seen[cite] = true
		}
	}

	return result, nil
}

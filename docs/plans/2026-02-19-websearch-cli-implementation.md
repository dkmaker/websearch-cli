# websearch CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool (`websearch`) for AI agents to perform web searches via Perplexity and Brave APIs, using a profile-based configuration system.

**Architecture:** Cobra CLI with a `Provider` interface abstracting Perplexity and Brave. Profiles are YAML files embedded in the binary via `embed.FS` with user overrides from `~/.config/websearch/profiles/`. File-based response cache with 60-minute TTL.

**Tech Stack:** Go, Cobra, gopkg.in/yaml.v3, standard library HTTP client

---

### Task 1: Initialize Go Module and Project Structure

**Files:**
- Create: `go.mod`
- Create: `cmd/websearch/main.go`

**Step 1: Initialize Go module**

Run: `cd /home/cp/code/dkmaker/search-cli && go mod init github.com/dkmaker/websearch`
Expected: `go.mod` created

**Step 2: Create minimal main.go**

```go
// cmd/websearch/main.go
package main

import (
	"fmt"
	"os"

	"github.com/dkmaker/websearch/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Create stub CLI package**

```go
// internal/cli/root.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "websearch [flags] <query>",
	Short: "Web search CLI for AI agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("websearch stub:", args[0])
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
```

**Step 4: Install dependencies and verify build**

Run: `cd /home/cp/code/dkmaker/search-cli && go get github.com/spf13/cobra && go get gopkg.in/yaml.v3 && go build ./cmd/websearch/`
Expected: Binary `websearch` created, no errors

**Step 5: Verify binary runs**

Run: `./websearch "test query"`
Expected: `websearch stub: test query`

**Step 6: Commit**

```bash
git add go.mod go.sum cmd/ internal/
git commit -m "feat: initialize project with Go module, Cobra CLI stub"
```

---

### Task 2: Provider Interface and Types

**Files:**
- Create: `internal/provider/provider.go`
- Test: `internal/provider/provider_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/...`
Expected: FAIL — types not defined yet

**Step 3: Write the provider interface and types**

```go
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
	Mode             string
	MaxTokens        int
	MaxResults       int
	SystemPrompt     string
	DomainFilter     []string
	RecencyFilter    string
	IncludeSources   bool
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
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/provider/
git commit -m "feat: add provider interface and search types"
```

---

### Task 3: Profile System

**Files:**
- Create: `internal/profile/profile.go`
- Create: `internal/profile/embed.go`
- Create: `internal/profile/profiles/general.yaml`
- Create: `internal/profile/profiles/nodejs.yaml`
- Create: `internal/profile/profiles/python.yaml`
- Test: `internal/profile/profile_test.go`

**Step 1: Write the test**

```go
// internal/profile/profile_test.go
package profile

import (
	"testing"
)

func TestLoadBuiltinProfile(t *testing.T) {
	p, err := Load("general")
	if err != nil {
		t.Fatalf("Load(general) error: %v", err)
	}
	if p.Name != "general" {
		t.Errorf("expected name 'general', got %q", p.Name)
	}
	if p.Provider == "" {
		t.Error("expected provider to be set")
	}
	if p.Mode == "" {
		t.Error("expected default mode to be set")
	}
}

func TestLoadBuiltinNodejs(t *testing.T) {
	p, err := Load("nodejs")
	if err != nil {
		t.Fatalf("Load(nodejs) error: %v", err)
	}
	if p.Name != "nodejs" {
		t.Errorf("expected name 'nodejs', got %q", p.Name)
	}
	if len(p.DomainFilter) == 0 {
		t.Error("expected nodejs profile to have domain filters")
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	_, err := Load("nonexistent")
	if err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestMergeProfiles(t *testing.T) {
	base := &Profile{
		Name:       "base",
		Provider:   "perplexity",
		Mode:       "ask",
		MaxTokens:  1024,
		MaxResults: 10,
	}
	override := &Profile{
		MaxTokens: 2048,
	}
	merged := Merge(base, override)
	if merged.Provider != "perplexity" {
		t.Errorf("expected provider 'perplexity', got %q", merged.Provider)
	}
	if merged.MaxTokens != 2048 {
		t.Errorf("expected max_tokens 2048, got %d", merged.MaxTokens)
	}
	if merged.MaxResults != 10 {
		t.Errorf("expected max_results 10, got %d", merged.MaxResults)
	}
}

func TestListProfiles(t *testing.T) {
	profiles := ListBuiltin()
	if len(profiles) < 3 {
		t.Errorf("expected at least 3 built-in profiles, got %d", len(profiles))
	}
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p] = true
	}
	for _, want := range []string{"general", "nodejs", "python"} {
		if !names[want] {
			t.Errorf("expected profile %q in list", want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/profile/...`
Expected: FAIL

**Step 3: Create the profile YAML files**

```yaml
# internal/profile/profiles/general.yaml
name: general
description: "General-purpose web search"
provider: perplexity
mode: ask
system_prompt: |
  Provide concise, factual answers. Use bullet points for lists.
  Include code examples when relevant. Be direct and avoid filler.
max_results: 5
max_tokens: 2048
output_format: markdown
```

```yaml
# internal/profile/profiles/nodejs.yaml
name: nodejs
description: "Node.js development searches"
provider: perplexity
mode: ask
system_prompt: |
  Focus on Node.js ecosystem. Include code examples using modern
  ESM syntax (import/export). Prefer official Node.js docs and
  well-maintained npm packages. Include version compatibility notes.
  Show package install commands when referencing npm packages.
domain_filter:
  - nodejs.org
  - developer.mozilla.org
  - npmjs.com
max_results: 5
max_tokens: 2048
output_format: markdown
```

```yaml
# internal/profile/profiles/python.yaml
name: python
description: "Python development searches"
provider: perplexity
mode: ask
system_prompt: |
  Focus on Python ecosystem. Include code examples using modern
  Python 3.10+ syntax (match/case, type hints, walrus operator).
  Prefer official Python docs and well-maintained PyPI packages.
  Include pip install commands when referencing packages.
domain_filter:
  - docs.python.org
  - pypi.org
  - realpython.com
max_results: 5
max_tokens: 2048
output_format: markdown
```

**Step 4: Write the embed and profile loading code**

```go
// internal/profile/embed.go
package profile

import "embed"

//go:embed profiles/*.yaml
var builtinProfiles embed.FS
```

```go
// internal/profile/profile.go
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile defines search configuration for a specific use case.
type Profile struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Provider     string   `yaml:"provider"`
	Mode         string   `yaml:"mode"`
	SystemPrompt string   `yaml:"system_prompt"`
	DomainFilter []string `yaml:"domain_filter"`
	MaxResults   int      `yaml:"max_results"`
	MaxTokens    int      `yaml:"max_tokens"`
	OutputFormat string   `yaml:"output_format"`
}

// Load loads a profile by name. It checks built-in profiles first,
// then user profiles in ~/.config/websearch/profiles/.
func Load(name string) (*Profile, error) {
	// Try built-in first
	builtin, err := loadBuiltin(name)
	if err == nil {
		// Check for user override
		userProfile, userErr := loadUser(name)
		if userErr == nil {
			return Merge(builtin, userProfile), nil
		}
		return builtin, nil
	}

	// Try user-only profile
	userProfile, userErr := loadUser(name)
	if userErr == nil {
		return userProfile, nil
	}

	return nil, fmt.Errorf("profile %q not found", name)
}

func loadBuiltin(name string) (*Profile, error) {
	data, err := builtinProfiles.ReadFile("profiles/" + name + ".yaml")
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing built-in profile %q: %w", name, err)
	}
	return &p, nil
}

func loadUser(name string) (*Profile, error) {
	configDir := userConfigDir()
	path := filepath.Join(configDir, "websearch", "profiles", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing user profile %q: %w", name, err)
	}
	return &p, nil
}

func userConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

// Merge overlays override fields on top of base. Zero-value fields
// in override are ignored.
func Merge(base, override *Profile) *Profile {
	merged := *base
	if override.Name != "" {
		merged.Name = override.Name
	}
	if override.Description != "" {
		merged.Description = override.Description
	}
	if override.Provider != "" {
		merged.Provider = override.Provider
	}
	if override.Mode != "" {
		merged.Mode = override.Mode
	}
	if override.SystemPrompt != "" {
		merged.SystemPrompt = override.SystemPrompt
	}
	if len(override.DomainFilter) > 0 {
		merged.DomainFilter = override.DomainFilter
	}
	if override.MaxResults > 0 {
		merged.MaxResults = override.MaxResults
	}
	if override.MaxTokens > 0 {
		merged.MaxTokens = override.MaxTokens
	}
	if override.OutputFormat != "" {
		merged.OutputFormat = override.OutputFormat
	}
	return &merged
}

// ListBuiltin returns the names of all built-in profiles.
func ListBuiltin() []string {
	entries, err := builtinProfiles.ReadDir("profiles")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	return names
}
```

**Step 5: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/profile/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/profile/
git commit -m "feat: add profile system with built-in profiles and user overrides"
```

---

### Task 4: Perplexity Provider

**Files:**
- Create: `internal/provider/perplexity.go`
- Test: `internal/provider/perplexity_test.go`

**Step 1: Write the test**

```go
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
		{"reason", "sonar-reasoning"},
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
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/... -run TestPerplexity`
Expected: FAIL

**Step 3: Implement Perplexity provider**

```go
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
// the default API URL is used.
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

func (p *Perplexity) Name() string {
	return "perplexity"
}

func (p *Perplexity) SupportedModes() []string {
	return []string{"ask", "search", "reason", "research"}
}

func modeToModel(mode string) string {
	switch mode {
	case "ask":
		return "sonar"
	case "search":
		return "sonar-pro"
	case "reason":
		return "sonar-reasoning"
	case "research":
		return "sonar-deep-research"
	default:
		return "sonar"
	}
}

type perplexityRequest struct {
	Model              string              `json:"model"`
	Messages           []perplexityMessage `json:"messages"`
	MaxTokens          int                 `json:"max_tokens,omitempty"`
	SearchDomainFilter []string            `json:"search_domain_filter,omitempty"`
	SearchRecencyFilter string             `json:"search_recency_filter,omitempty"`
	SearchContextSize  string              `json:"search_context_size,omitempty"`
	Stream             bool                `json:"stream"`
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
	Citations     []string              `json:"citations"`
	SearchResults []perplexitySearchResult `json:"search_results"`
}

type perplexitySearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Date  string `json:"date"`
}

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
		Content:  pResp.Choices[0].Message.Content,
		Provider: "perplexity",
		Mode:     mode,
	}

	// Build sources from citations and search results
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
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/... -run TestPerplexity`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/provider/perplexity.go internal/provider/perplexity_test.go
git commit -m "feat: add Perplexity provider with ask/search/reason/research modes"
```

---

### Task 5: Brave Provider

**Files:**
- Create: `internal/provider/brave.go`
- Test: `internal/provider/brave_test.go`

**Step 1: Write the test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/... -run TestBrave`
Expected: FAIL

**Step 3: Implement Brave provider**

```go
// internal/provider/brave.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultBraveURL = "https://api.search.brave.com"

// Brave implements the Provider interface for the Brave Search API.
type Brave struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewBrave creates a new Brave provider. If baseURL is empty,
// the default API URL is used.
func NewBrave(apiKey, baseURL string) *Brave {
	if baseURL == "" {
		baseURL = defaultBraveURL
	}
	return &Brave{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (b *Brave) Name() string {
	return "brave"
}

func (b *Brave) SupportedModes() []string {
	return []string{"web"}
}

type braveResponse struct {
	Web struct {
		Results []braveResult `json:"results"`
	} `json:"web"`
}

type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	PageAge     string `json:"page_age"`
}

func (b *Brave) Search(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("text_format", "markdown")
	if opts.MaxResults > 0 {
		count := opts.MaxResults
		if count > 20 {
			count = 20
		}
		params.Set("count", strconv.Itoa(count))
	}
	if opts.RecencyFilter != "" {
		params.Set("freshness", braveRecency(opts.RecencyFilter))
	}

	endpoint := b.baseURL + "/res/v1/web/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Subscription-Token", b.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var bResp braveResponse
	if err := json.Unmarshal(respBody, &bResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	result := &Result{
		Provider: "brave",
		Mode:     "web",
	}

	// Build markdown content from results
	var sb strings.Builder
	for i, r := range bResp.Web.Results {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s**\n%s", r.Title, r.Description))
		result.Sources = append(result.Sources, Source{
			Title: r.Title,
			URL:   r.URL,
			Date:  r.PageAge,
		})
	}
	result.Content = sb.String()

	return result, nil
}

// braveRecency maps generic recency values to Brave's freshness parameter.
func braveRecency(recency string) string {
	switch recency {
	case "hour", "day":
		return "pd"
	case "week":
		return "pw"
	case "month":
		return "pm"
	case "year":
		return "py"
	default:
		return recency
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/provider/... -run TestBrave`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/provider/brave.go internal/provider/brave_test.go
git commit -m "feat: add Brave Search provider with web search mode"
```

---

### Task 6: Response Cache

**Files:**
- Create: `internal/cache/cache.go`
- Test: `internal/cache/cache_test.go`

**Step 1: Write the test**

```go
// internal/cache/cache_test.go
package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	k1 := Key("query1", "general", "ask", "perplexity")
	k2 := Key("query1", "general", "ask", "perplexity")
	k3 := Key("query2", "general", "ask", "perplexity")

	if k1 != k2 {
		t.Error("same inputs should produce same key")
	}
	if k1 == k3 {
		t.Error("different inputs should produce different keys")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	key := Key("test", "general", "ask", "perplexity")
	data := []byte(`{"content":"cached answer","sources":[]}`)

	err := c.Set(key, data)
	if err != nil {
		t.Fatalf("Set error: %v", err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestCacheExpiry(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1*time.Millisecond)

	key := Key("test", "general", "ask", "perplexity")
	c.Set(key, []byte("old data"))

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get(key)
	if ok {
		t.Error("expected expired cache miss")
	}
}

func TestCachePurge(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 1*time.Millisecond)

	key := Key("test", "general", "ask", "perplexity")
	c.Set(key, []byte("old data"))

	time.Sleep(5 * time.Millisecond)

	removed := c.Purge()
	if removed != 1 {
		t.Errorf("expected 1 purged, got %d", removed)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir after purge, got %d entries", len(entries))
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, 60*time.Minute)

	// Write a corrupted cache file
	key := "badkey"
	os.WriteFile(filepath.Join(dir, key+".json"), []byte("not json{{{"), 0644)

	_, ok := c.Get(key)
	if ok {
		t.Error("corrupted file should result in cache miss")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/cache/...`
Expected: FAIL

**Step 3: Implement cache**

```go
// internal/cache/cache.go
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache provides file-based response caching with TTL.
type Cache struct {
	dir string
	ttl time.Duration
}

type cacheEntry struct {
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// New creates a new Cache in the given directory with the specified TTL.
func New(dir string, ttl time.Duration) *Cache {
	os.MkdirAll(dir, 0755)
	return &Cache{dir: dir, ttl: ttl}
}

// Key generates a cache key from the given parameters.
func Key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Get retrieves a cached entry. Returns the data and true if found and not expired.
func (c *Cache) Get(key string) ([]byte, bool) {
	path := filepath.Join(c.dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Corrupted file — treat as miss
		os.Remove(path)
		return nil, false
	}

	if time.Since(entry.CreatedAt) > c.ttl {
		os.Remove(path)
		return nil, false
	}

	return entry.Data, true
}

// Set stores data in the cache. Errors are silently ignored (e.g., disk full).
func (c *Cache) Set(key string, data []byte) error {
	entry := cacheEntry{
		Data:      data,
		CreatedAt: time.Now(),
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, key+".json")
	return os.WriteFile(path, encoded, 0644)
}

// Purge removes all expired entries. Returns the number of entries removed.
func (c *Cache) Purge() int {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			os.Remove(path)
			removed++
			continue
		}
		if time.Since(entry.CreatedAt) > c.ttl {
			os.Remove(path)
			removed++
		}
	}
	return removed
}

// DefaultDir returns the default cache directory.
func DefaultDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "websearch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "websearch")
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/cache/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cache/
git commit -m "feat: add file-based response cache with TTL and auto-purge"
```

---

### Task 7: Output Formatter

**Files:**
- Create: `internal/output/formatter.go`
- Test: `internal/output/formatter_test.go`

**Step 1: Write the test**

```go
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
	if !strings.Contains(out, "--include-sources") {
		t.Error("expected hint about --include-sources")
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
	// No hint when there are no sources at all
	if strings.Contains(out, "--include-sources") {
		t.Error("no hint expected when result has no sources")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/output/...`
Expected: FAIL

**Step 3: Implement formatter**

```go
// internal/output/formatter.go
package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dkmaker/websearch/internal/provider"
)

// FormatMarkdown formats a result as concise markdown.
func FormatMarkdown(result *provider.Result, includeSources bool) string {
	var sb strings.Builder

	sb.WriteString(result.Content)

	if includeSources && len(result.Sources) > 0 {
		sb.WriteString("\n\n---\nSources:\n")
		for _, s := range result.Sources {
			if s.Title != "" {
				sb.WriteString(fmt.Sprintf("- [%s](%s)\n", s.Title, s.URL))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", s.URL))
			}
		}
	} else if !includeSources && len(result.Sources) > 0 {
		sb.WriteString("\n<!-- run with --include-sources for citations -->")
	}

	return sb.String()
}

type jsonOutput struct {
	Content  string            `json:"content"`
	Provider string            `json:"provider"`
	Mode     string            `json:"mode"`
	Cached   bool              `json:"cached"`
	Sources  []provider.Source  `json:"sources,omitempty"`
}

// FormatJSON formats a result as JSON.
func FormatJSON(result *provider.Result, includeSources bool) (string, error) {
	out := jsonOutput{
		Content:  result.Content,
		Provider: result.Provider,
		Mode:     result.Mode,
		Cached:   result.Cached,
	}
	if includeSources {
		out.Sources = result.Sources
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/output/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat: add markdown and JSON output formatters"
```

---

### Task 8: Wire Everything Together in CLI

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `cmd/websearch/main.go`
- Test: `internal/cli/root_test.go`

**Step 1: Write the test**

```go
// internal/cli/root_test.go
package cli

import (
	"testing"
)

func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name         string
		profileProv  string
		flagProv     string
		perplexKey   string
		braveKey     string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "profile wants perplexity, key available",
			profileProv:  "perplexity",
			perplexKey:   "pplx-key",
			wantProvider: "perplexity",
		},
		{
			name:         "profile wants perplexity, only brave key",
			profileProv:  "perplexity",
			braveKey:     "brave-key",
			wantProvider: "brave",
		},
		{
			name:    "no keys at all",
			wantErr: true,
		},
		{
			name:         "flag overrides profile",
			profileProv:  "perplexity",
			flagProv:     "brave",
			perplexKey:   "pplx-key",
			braveKey:     "brave-key",
			wantProvider: "brave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := resolveProvider(tt.profileProv, tt.flagProv, tt.perplexKey, tt.braveKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.wantProvider {
				t.Errorf("got provider %q, want %q", got, tt.wantProvider)
			}
		})
	}
}

func TestValidateMode(t *testing.T) {
	tests := []struct {
		mode     string
		provider string
		wantErr  bool
	}{
		{"ask", "perplexity", false},
		{"research", "perplexity", false},
		{"web", "brave", false},
		{"research", "brave", true},
		{"", "perplexity", false},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"_"+tt.provider, func(t *testing.T) {
			err := validateMode(tt.mode, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMode(%q, %q) error = %v, wantErr %v", tt.mode, tt.provider, err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/cli/...`
Expected: FAIL

**Step 3: Implement the full CLI wiring**

```go
// internal/cli/root.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dkmaker/websearch/internal/cache"
	"github.com/dkmaker/websearch/internal/output"
	"github.com/dkmaker/websearch/internal/profile"
	"github.com/dkmaker/websearch/internal/provider"
	"github.com/spf13/cobra"
)

var (
	flagProfile        string
	flagMode           string
	flagJSON           bool
	flagProvider       string
	flagMaxResults     int
	flagMaxTokens      int
	flagIncludeSources bool
	flagNoCache        bool
	flagListProfiles   bool
)

var rootCmd = &cobra.Command{
	Use:   "websearch [flags] <query>",
	Short: "Web search CLI for AI agents",
	Long:  "A CLI tool for AI agents to perform web searches via Perplexity and Brave APIs using profile-based configuration.",
	Args: func(cmd *cobra.Command, args []string) error {
		if flagListProfiles {
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 query argument")
		}
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

func init() {
	rootCmd.Flags().StringVarP(&flagProfile, "profile", "p", "general", "Profile name")
	rootCmd.Flags().StringVarP(&flagMode, "mode", "m", "", "Search mode override")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "Provider override (perplexity, brave)")
	rootCmd.Flags().IntVarP(&flagMaxResults, "max-results", "n", 0, "Max result count")
	rootCmd.Flags().IntVar(&flagMaxTokens, "max-tokens", 0, "Max response tokens")
	rootCmd.Flags().BoolVar(&flagIncludeSources, "include-sources", false, "Include citations/sources")
	rootCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "Bypass response cache")
	rootCmd.Flags().BoolVar(&flagListProfiles, "list-profiles", false, "List available profiles")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	if flagListProfiles {
		return listProfiles()
	}

	query := args[0]

	// Load profile
	prof, err := profile.Load(flagProfile)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}

	// Apply flag overrides to profile
	if flagMaxResults > 0 {
		prof.MaxResults = flagMaxResults
	}
	if flagMaxTokens > 0 {
		prof.MaxTokens = flagMaxTokens
	}

	// Resolve mode
	mode := flagMode
	if mode == "" {
		mode = prof.Mode
	}

	// Get API keys
	perplexityKey := os.Getenv("PERPLEXITY_API_KEY")
	braveKey := os.Getenv("BRAVE_API_KEY")

	// Resolve provider
	providerName, warning, err := resolveProvider(prof.Provider, flagProvider, perplexityKey, braveKey)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// Validate mode against provider
	if err := validateMode(mode, providerName); err != nil {
		return err
	}

	// Build search options
	opts := provider.SearchOptions{
		Mode:         mode,
		MaxTokens:    prof.MaxTokens,
		MaxResults:   prof.MaxResults,
		SystemPrompt: prof.SystemPrompt,
		DomainFilter: prof.DomainFilter,
	}

	// Check cache
	responseCache := cache.New(cache.DefaultDir(), 60*time.Minute)

	// Purge stale entries in background
	go responseCache.Purge()

	cacheKey := cache.Key(query, flagProfile, mode, providerName)

	if !flagNoCache {
		if data, ok := responseCache.Get(cacheKey); ok {
			var result provider.Result
			if err := json.Unmarshal(data, &result); err == nil {
				result.Cached = true
				return outputResult(&result)
			}
		}
	}

	// Create provider
	var prov provider.Provider
	switch providerName {
	case "perplexity":
		prov = provider.NewPerplexity(perplexityKey, "")
	case "brave":
		prov = provider.NewBrave(braveKey, "")
	}

	// Execute search
	result, err := prov.Search(context.Background(), query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Cache result
	if data, err := json.Marshal(result); err == nil {
		responseCache.Set(cacheKey, data)
	}

	return outputResult(result)
}

func outputResult(result *provider.Result) error {
	if flagJSON {
		out, err := output.FormatJSON(result, flagIncludeSources)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Print(output.FormatMarkdown(result, flagIncludeSources))
	}
	return nil
}

func listProfiles() error {
	names := profile.ListBuiltin()
	for _, name := range names {
		p, err := profile.Load(name)
		if err != nil {
			fmt.Printf("  %s (error loading)\n", name)
			continue
		}
		fmt.Printf("  %-12s %s (provider: %s, mode: %s)\n", name, p.Description, p.Provider, p.Mode)
	}
	return nil
}

// resolveProvider determines which provider to use based on profile preference,
// flag override, and available API keys. Returns provider name, optional warning, and error.
func resolveProvider(profileProv, flagProv, perplexityKey, braveKey string) (string, string, error) {
	if perplexityKey == "" && braveKey == "" {
		return "", "", fmt.Errorf("no API keys found. Set PERPLEXITY_API_KEY or BRAVE_API_KEY")
	}

	// Flag override takes priority
	preferred := profileProv
	if flagProv != "" {
		preferred = flagProv
	}

	switch preferred {
	case "perplexity":
		if perplexityKey != "" {
			return "perplexity", "", nil
		}
		return "brave", "warning: PERPLEXITY_API_KEY not set, falling back to Brave", nil
	case "brave":
		if braveKey != "" {
			return "brave", "", nil
		}
		return "perplexity", "warning: BRAVE_API_KEY not set, falling back to Perplexity", nil
	default:
		// No preference — use whatever is available, prefer perplexity
		if perplexityKey != "" {
			return "perplexity", "", nil
		}
		return "brave", "", nil
	}
}

// validateMode checks that the requested mode is supported by the provider.
func validateMode(mode, providerName string) error {
	if mode == "" {
		return nil
	}

	perplexityModes := map[string]bool{"ask": true, "search": true, "reason": true, "research": true}
	braveModes := map[string]bool{"web": true}

	switch providerName {
	case "perplexity":
		if !perplexityModes[mode] {
			return fmt.Errorf("mode %q not supported by Perplexity (supported: ask, search, reason, research)", mode)
		}
	case "brave":
		if !braveModes[mode] {
			return fmt.Errorf("mode %q requires Perplexity. Set PERPLEXITY_API_KEY", mode)
		}
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./internal/cli/...`
Expected: PASS

**Step 5: Build and verify the full binary**

Run: `cd /home/cp/code/dkmaker/search-cli && go build -o websearch ./cmd/websearch/ && ./websearch --list-profiles`
Expected: Lists general, nodejs, python profiles

**Step 6: Run all tests**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./...`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/cli/ cmd/
git commit -m "feat: wire CLI with provider resolution, caching, and output formatting"
```

---

### Task 9: Add .gitignore and Build Configuration

**Files:**
- Create: `.gitignore`

**Step 1: Create .gitignore**

```
# .gitignore
websearch
*.exe
/dist/
```

**Step 2: Clean up and verify final build**

Run: `cd /home/cp/code/dkmaker/search-cli && go build -o websearch ./cmd/websearch/ && ./websearch --help`
Expected: Shows usage help with all flags

**Step 3: Run full test suite one more time**

Run: `cd /home/cp/code/dkmaker/search-cli && go test ./... -v`
Expected: All tests pass

**Step 4: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore for build artifacts"
```

---

### Task 10: Final Verification and Cleanup

**Step 1: Verify binary works end-to-end (requires API key)**

Run: `cd /home/cp/code/dkmaker/search-cli && ./websearch "what is Go 1.22"` (only if PERPLEXITY_API_KEY or BRAVE_API_KEY is set)
Expected: Returns search results

**Step 2: Verify error messages without API keys**

Run: `cd /home/cp/code/dkmaker/search-cli && unset PERPLEXITY_API_KEY && unset BRAVE_API_KEY && ./websearch "test" 2>&1; echo "exit: $?"`
Expected: Error message about missing API keys, exit code 1

**Step 3: Verify cache behavior**

Run the same query twice and confirm second is faster/cached.

**Step 4: Commit any final changes**

```bash
git add -A
git commit -m "chore: final cleanup and verification"
```

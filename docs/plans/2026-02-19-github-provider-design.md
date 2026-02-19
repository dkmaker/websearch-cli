# GitHub Provider — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GitHub as a search provider with repos, code, and issues modes.

**Architecture:** Single GitHub provider struct with mode-based routing to /search/repositories, /search/code, and /search/issues endpoints. Auth via GITHUB_TOKEN env var (optional for repos/issues, required for code). Compact markdown output per mode. New builtin github profile.

**Tech Stack:** Go 1.24+, net/http, encoding/json, httptest for tests

---

### Task 1: GitHub Provider — Repos Mode

**Files:**
- Create: `internal/provider/github.go`
- Create: `internal/provider/github_test.go`

**Step 1: Write the failing test**

Create `internal/provider/github_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestGitHub -v`
Expected: FAIL — `NewGitHub` not defined

**Step 3: Implement GitHub provider with repos mode**

Create `internal/provider/github.go`:

```go
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

const defaultGitHubURL = "https://api.github.com"

type GitHub struct {
	token   string
	baseURL string
	client  *http.Client
}

func NewGitHub(token, baseURL string) *GitHub {
	if baseURL == "" {
		baseURL = defaultGitHubURL
	}
	return &GitHub{
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) SupportedModes() []string {
	return []string{"repos", "code", "issues"}
}

// GitHub API response types

type githubSearchResponse struct {
	TotalCount        int               `json:"total_count"`
	IncompleteResults bool              `json:"incomplete_results"`
	Items             json.RawMessage   `json:"items"`
}

type githubRepo struct {
	FullName        string   `json:"full_name"`
	HTMLURL         string   `json:"html_url"`
	Description     string   `json:"description"`
	StargazersCount int      `json:"stargazers_count"`
	Language        string   `json:"language"`
	UpdatedAt       string   `json:"updated_at"`
	Topics          []string `json:"topics"`
}

type githubCodeResult struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	HTMLURL    string `json:"html_url"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	TextMatches []struct {
		Fragment string `json:"fragment"`
	} `json:"text_matches"`
}

type githubIssue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	Comments  int    `json:"comments"`
	UpdatedAt string `json:"updated_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	RepositoryURL string `json:"repository_url"`
}

func (g *GitHub) Search(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	switch opts.Mode {
	case "repos":
		return g.searchRepos(ctx, query, opts)
	case "code":
		return g.searchCode(ctx, query, opts)
	case "issues":
		return g.searchIssues(ctx, query, opts)
	default:
		return g.searchRepos(ctx, query, opts)
	}
}

func (g *GitHub) doRequest(ctx context.Context, endpoint string, acceptHeader string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	} else {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (g *GitHub) buildURL(path string, query string, maxResults int) string {
	params := url.Values{}
	params.Set("q", query)
	perPage := maxResults
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	params.Set("per_page", strconv.Itoa(perPage))
	return g.baseURL + path + "?" + params.Encode()
}

func (g *GitHub) searchRepos(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	endpoint := g.buildURL("/search/repositories", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var repos []githubRepo
	if err := json.Unmarshal(resp.Items, &repos); err != nil {
		return nil, fmt.Errorf("parsing repos: %w", err)
	}

	result := &Result{
		Provider: "github",
		Mode:     "repos",
	}

	var sb strings.Builder
	for i, r := range repos {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s** %s — %s", r.FullName, formatStars(r.StargazersCount), r.Description))
		// Metadata line
		var meta []string
		if r.Language != "" {
			meta = append(meta, "Language: "+r.Language)
		}
		if r.UpdatedAt != "" {
			meta = append(meta, "Updated: "+r.UpdatedAt[:10])
		}
		if len(r.Topics) > 0 {
			meta = append(meta, "Topics: "+strings.Join(r.Topics, ", "))
		}
		if len(meta) > 0 {
			sb.WriteString("\n  " + strings.Join(meta, " | "))
		}

		result.Sources = append(result.Sources, Source{
			Title: r.FullName,
			URL:   r.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}

func formatStars(count int) string {
	if count >= 1000 {
		return fmt.Sprintf("%dk", count/1000)
	}
	return strconv.Itoa(count)
}

// searchCode and searchIssues will be implemented in Tasks 2 and 3
```

Note: Leave `searchCode` and `searchIssues` as stubs that return errors for now — they'll be implemented in Tasks 2 and 3. Add these stubs:

```go
func (g *GitHub) searchCode(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	return nil, fmt.Errorf("code search not yet implemented")
}

func (g *GitHub) searchIssues(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	return nil, fmt.Errorf("issues search not yet implemented")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run TestGitHub -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/provider/github.go internal/provider/github_test.go
git commit -m "feat: add GitHub provider with repos search mode"
```

---

### Task 2: GitHub Provider — Code Mode

**Files:**
- Modify: `internal/provider/github.go`
- Modify: `internal/provider/github_test.go`

**Step 1: Write the failing test**

Add to `internal/provider/github_test.go`:

```go
func TestGitHubSearchCode(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        1,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"name":     "main.go",
				"path":     "cmd/server/main.go",
				"html_url": "https://github.com/owner/repo/blob/main/cmd/server/main.go",
				"repository": map[string]interface{}{
					"full_name": "owner/repo",
				},
				"text_matches": []map[string]interface{}{
					{"fragment": "func main() {\n\thttp.ListenAndServe(\":8080\", nil)\n}"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/code" {
			t.Errorf("expected /search/code, got %s", r.URL.Path)
		}
		// Code search requires text-match accept header
		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "text-match") {
			t.Error("expected text-match accept header for code search")
		}
		// Code search requires auth
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header for code search")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("ghp_testtoken", server.URL)
	result, err := p.Search(context.Background(), "http.ListenAndServe language:go", SearchOptions{
		Mode:       "code",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Mode != "code" {
		t.Errorf("expected mode 'code', got %q", result.Mode)
	}
	if !strings.Contains(result.Content, "owner/repo") {
		t.Error("expected repo name in content")
	}
	if !strings.Contains(result.Content, "cmd/server/main.go") {
		t.Error("expected file path in content")
	}
	if !strings.Contains(result.Content, "ListenAndServe") {
		t.Error("expected code fragment in content")
	}
}

func TestGitHubSearchCodeRequiresAuth(t *testing.T) {
	p := NewGitHub("", "http://localhost")
	_, err := p.Search(context.Background(), "test", SearchOptions{Mode: "code"})
	if err == nil {
		t.Error("expected error for code search without token")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("expected GITHUB_TOKEN in error, got: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestGitHubSearchCode -v`
Expected: FAIL — "code search not yet implemented"

**Step 3: Implement searchCode**

Replace the `searchCode` stub in `internal/provider/github.go`:

```go
func (g *GitHub) searchCode(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if g.token == "" {
		return nil, fmt.Errorf("code search requires authentication. Set GITHUB_TOKEN")
	}

	endpoint := g.buildURL("/search/code", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "application/vnd.github.text-match+json")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var items []githubCodeResult
	if err := json.Unmarshal(resp.Items, &items); err != nil {
		return nil, fmt.Errorf("parsing code results: %w", err)
	}

	result := &Result{
		Provider: "github",
		Mode:     "code",
	}

	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s** %s", item.Repository.FullName, item.Path))
		for _, tm := range item.TextMatches {
			if tm.Fragment != "" {
				sb.WriteString("\n  " + strings.ReplaceAll(tm.Fragment, "\n", "\n  "))
				break // only first fragment
			}
		}
		result.Sources = append(result.Sources, Source{
			Title: item.Repository.FullName + "/" + item.Path,
			URL:   item.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run TestGitHubSearchCode -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/provider/github.go internal/provider/github_test.go
git commit -m "feat: add GitHub code search mode"
```

---

### Task 3: GitHub Provider — Issues Mode

**Files:**
- Modify: `internal/provider/github.go`
- Modify: `internal/provider/github_test.go`

**Step 1: Write the failing test**

Add to `internal/provider/github_test.go`:

```go
func TestGitHubSearchIssues(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        2,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"number":         42,
				"title":          "Bug: search returns empty",
				"html_url":       "https://github.com/owner/repo/issues/42",
				"state":          "open",
				"comments":       5,
				"updated_at":     "2026-02-19T10:00:00Z",
				"repository_url": "https://api.github.com/repos/owner/repo",
				"labels": []map[string]interface{}{
					{"name": "bug"},
					{"name": "priority-high"},
				},
			},
			{
				"number":         15,
				"title":          "Feature: add search",
				"html_url":       "https://github.com/owner/repo/issues/15",
				"state":          "closed",
				"comments":       12,
				"updated_at":     "2026-01-05T10:00:00Z",
				"repository_url": "https://api.github.com/repos/owner/repo",
				"labels": []map[string]interface{}{
					{"name": "enhancement"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			t.Errorf("expected /search/issues, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	result, err := p.Search(context.Background(), "search bug", SearchOptions{
		Mode:       "issues",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if result.Mode != "issues" {
		t.Errorf("expected mode 'issues', got %q", result.Mode)
	}
	if len(result.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(result.Sources))
	}
	if !strings.Contains(result.Content, "owner/repo#42") {
		t.Error("expected owner/repo#42 in content")
	}
	if !strings.Contains(result.Content, "open") {
		t.Error("expected state in content")
	}
	if !strings.Contains(result.Content, "bug") {
		t.Error("expected label in content")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestGitHubSearchIssues -v`
Expected: FAIL — "issues search not yet implemented"

**Step 3: Implement searchIssues**

Replace the `searchIssues` stub in `internal/provider/github.go`:

```go
func (g *GitHub) searchIssues(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	endpoint := g.buildURL("/search/issues", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var issues []githubIssue
	if err := json.Unmarshal(resp.Items, &issues); err != nil {
		return nil, fmt.Errorf("parsing issues: %w", err)
	}

	result := &Result{
		Provider: "github",
		Mode:     "issues",
	}

	var sb strings.Builder
	for i, issue := range issues {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		// Extract repo name from repository_url: https://api.github.com/repos/owner/repo
		repoName := repoNameFromURL(issue.RepositoryURL)
		sb.WriteString(fmt.Sprintf("**%s#%d** %s (%s)", repoName, issue.Number, issue.Title, issue.State))

		var meta []string
		if len(issue.Labels) > 0 {
			var labelNames []string
			for _, l := range issue.Labels {
				labelNames = append(labelNames, l.Name)
			}
			meta = append(meta, "Labels: "+strings.Join(labelNames, ", "))
		}
		if issue.Comments > 0 {
			meta = append(meta, fmt.Sprintf("Comments: %d", issue.Comments))
		}
		if issue.UpdatedAt != "" {
			meta = append(meta, "Updated: "+issue.UpdatedAt[:10])
		}
		if len(meta) > 0 {
			sb.WriteString("\n  " + strings.Join(meta, " | "))
		}

		result.Sources = append(result.Sources, Source{
			Title: fmt.Sprintf("%s#%d", repoName, issue.Number),
			URL:   issue.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}

func repoNameFromURL(apiURL string) string {
	// https://api.github.com/repos/owner/repo -> owner/repo
	const prefix = "/repos/"
	idx := strings.Index(apiURL, prefix)
	if idx >= 0 {
		return apiURL[idx+len(prefix):]
	}
	return ""
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run TestGitHub -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/provider/github.go internal/provider/github_test.go
git commit -m "feat: add GitHub issues search mode"
```

---

### Task 4: GitHub Profile

**Files:**
- Create: `internal/profile/profiles/github.yaml`
- Modify: `internal/profile/profile_test.go`

**Step 1: Create the profile**

Create `internal/profile/profiles/github.yaml`:

```yaml
name: github
description: GitHub repository and code search
provider: github
mode: repos
system_prompt: ""
max_results: 10
output_format: markdown
```

**Step 2: Write the test**

Add to `internal/profile/profile_test.go`:

```go
func TestLoadBuiltinGitHub(t *testing.T) {
	p, err := Load("github")
	if err != nil {
		t.Fatalf("Load(github) error: %v", err)
	}
	if p.Name != "github" {
		t.Errorf("expected name 'github', got %q", p.Name)
	}
	if p.Provider != "github" {
		t.Errorf("expected provider 'github', got %q", p.Provider)
	}
	if p.Mode != "repos" {
		t.Errorf("expected mode 'repos', got %q", p.Mode)
	}
}
```

**Step 3: Run test**

Run: `go test ./internal/profile/ -run TestLoadBuiltinGitHub -v`
Expected: PASS

**Step 4: Update ListBuiltin test expectation**

In `internal/profile/profile_test.go`, update `TestListProfiles` to expect 4 profiles instead of 3, and add "github" to the expected list:

Change the line:
```go
if len(profiles) < 3 {
```
to:
```go
if len(profiles) < 4 {
```

And add `"github"` to the expected names slice:
```go
for _, want := range []string{"general", "github", "nodejs", "python"} {
```

**Step 5: Run all profile tests**

Run: `go test ./internal/profile/ -v`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/profile/profiles/github.yaml internal/profile/profile_test.go
git commit -m "feat: add builtin github profile"
```

---

### Task 5: Wire GitHub into CLI

**Files:**
- Modify: `internal/cli/root.go:65,114-116,119-120,167-174,220-246,249-269,271-279,281-343`
- Modify: `internal/cli/root_test.go`

**Step 1: Write failing tests**

Add to `internal/cli/root_test.go`:

```go
func TestResolveProviderGitHub(t *testing.T) {
	// GitHub with token
	got, _, err := resolveProvider("github", "", "", "", "ghp_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}

	// GitHub without token (should still work — unauthenticated OK for repos/issues)
	got, _, err = resolveProvider("github", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}

	// Flag override to github
	got, _, err = resolveProvider("perplexity", "github", "pplx-key", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}
}

func TestValidateModeGitHub(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"repos", false},
		{"code", false},
		{"issues", false},
		{"web", true},
		{"ask", true},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"_github", func(t *testing.T) {
			err := validateMode(tt.mode, "github")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMode(%q, github) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
			}
		})
	}
}

func TestSelfPrimerGitHub(t *testing.T) {
	output := buildSelfPrimer("", "", "ghp_token", false, false)
	if !strings.Contains(output, "github (ready)") {
		t.Error("expected github (ready)")
	}
	if !strings.Contains(output, "repos*") {
		t.Error("expected repos* mode")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestResolveProviderGitHub|TestValidateModeGitHub|TestSelfPrimerGitHub" -v`
Expected: FAIL — wrong function signatures, missing github handling

**Step 3: Implement CLI integration**

Modify `internal/cli/root.go`:

1. **Update resolveProvider signature** (line 220) — add `githubKey` parameter:

```go
func resolveProvider(profileProv, flagProv, perplexityKey, braveKey, githubKey string) (string, string, error) {
```

Add github case inside the switch (after the brave case at line 240):

```go
	case "github":
		return "github", "", nil
```

Update the "no keys" check (line 221-223) — GitHub doesn't strictly need a key:

```go
	if perplexityKey == "" && braveKey == "" && profileProv != "github" && flagProv != "github" {
		return "", "", fmt.Errorf("no API keys found. Set PERPLEXITY_API_KEY, BRAVE_API_KEY, or GITHUB_TOKEN")
	}
```

2. **Update validateMode** (line 249-269) — add github modes:

```go
	githubModes := map[string]bool{"repos": true, "code": true, "issues": true}
```

Add case:
```go
	case "github":
		if !githubModes[mode] {
			return fmt.Errorf("mode %q not supported by GitHub (supported: repos, code, issues)", mode)
		}
```

3. **Update defaultModeForProvider** (line 271-279) — add github:

```go
	case "github":
		return "repos"
```

4. **Update run()** (line 114-116) — add GITHUB_TOKEN:

```go
	githubKey := os.Getenv("GITHUB_TOKEN")
```

Update resolveProvider call (line 119):
```go
	providerName, warning, err := resolveProvider(prof.Provider, flagProvider, perplexityKey, braveKey, githubKey)
```

Add github case to provider creation switch (line 167-174):
```go
	case "github":
		prov = provider.NewGitHub(githubKey, "")
```

5. **Update provider flag description** (line 65):

```go
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "Provider override (perplexity, brave, github)")
```

6. **Update Long description** (line 43):

```go
	Long: "A CLI tool for AI agents to perform web searches via Perplexity, Brave, and GitHub APIs using profile-based configuration.",
```

7. **Update buildSelfPrimer** (line 281) — add githubKey parameter and github info:

Change signature:
```go
func buildSelfPrimer(perplexityKey, braveKey, githubKey string, showExamples, showProfiles bool) string {
```

Add github status after brave status (after line 295):
```go
	gStatus := "no key"
	if githubKey != "" {
		gStatus = "ready"
	}
	sb.WriteString(fmt.Sprintf("Providers: perplexity (%s), brave (%s), github (%s)\n", pStatus, bStatus, gStatus))
```

(Remove the existing line 296 that writes just perplexity and brave.)

Add github modes line after brave modes (after line 300):
```go
	sb.WriteString("Modes [github]: repos*, code, issues\n")
```

Add github example in examples section (after line 322):
```go
		sb.WriteString("  websearch -p github \"cobra CLI framework\"                  # github repos\n")
		sb.WriteString("  websearch -p github -m code \"http.ListenAndServe lang:go\"   # github code\n")
```

8. **Update self-primer call** in run() (line 86-88):

```go
		githubKey := os.Getenv("GITHUB_TOKEN")
		fmt.Print(buildSelfPrimer(perplexityKey, braveKey, githubKey, flagShowExamples, flagShowProfiles))
```

9. **Update all existing test calls to resolveProvider** in root_test.go — add empty string for githubKey parameter:

Every call `resolveProvider(a, b, c, d)` becomes `resolveProvider(a, b, c, d, "")`.

10. **Update all existing test calls to buildSelfPrimer** — add empty string for githubKey parameter:

Every call `buildSelfPrimer(a, b, c, d)` becomes `buildSelfPrimer(a, b, "", c, d)`.

**Step 4: Run all tests**

Run: `go test ./internal/cli/ -v`
Expected: ALL PASS

Run: `go test ./...`
Expected: ALL PASS

**Step 5: Build and verify**

Run: `go build -o /tmp/websearch ./cmd/websearch/ && /tmp/websearch`
Expected: Self-primer shows github provider and modes

Run: `/tmp/websearch --show-examples`
Expected: Includes github examples

**Step 6: Commit**

```bash
git add internal/cli/root.go internal/cli/root_test.go
git commit -m "feat: wire GitHub provider into CLI with self-primer support"
```

---

### Task 6: Final Verification

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 2: Build and verify all scenarios**

Run: `go build -o /tmp/websearch ./cmd/websearch/`

Verify:
- `/tmp/websearch` — shows github in self-primer
- `/tmp/websearch --show-examples` — shows github examples
- `/tmp/websearch --show-profiles` — shows github profile
- `/tmp/websearch --list-profiles` — shows github in list
- `/tmp/websearch --version` — still works

**Step 3: Commit only if fixes were needed**

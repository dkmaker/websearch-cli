// internal/provider/github_test.go
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
				"body":           "When searching for items, the result set is empty despite matching entries existing.",
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
				"body":           "It would be great to have a search feature.",
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

func TestGitHubSearchReposEmptyResults(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        0,
		"incomplete_results": false,
		"items":              []map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	_, err := p.Search(context.Background(), "nonexistent-query-xyz", SearchOptions{
		Mode:       "repos",
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for empty repos results")
	}
	if !strings.Contains(err.Error(), "no results found") {
		t.Errorf("expected 'no results found' error, got: %v", err)
	}
}

func TestGitHubSearchIssuesEmptyResults(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        0,
		"incomplete_results": false,
		"items":              []map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	_, err := p.Search(context.Background(), "nonexistent-query-xyz", SearchOptions{
		Mode:       "issues",
		MaxResults: 10,
	})
	if err == nil {
		t.Fatal("expected error for empty issues results")
	}
	if !strings.Contains(err.Error(), "no results found") {
		t.Errorf("expected 'no results found' error, got: %v", err)
	}
}

func TestGitHubSearchIssuesIncludesBody(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        1,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"number":         99,
				"title":          "Issue with body",
				"html_url":       "https://github.com/owner/repo/issues/99",
				"state":          "open",
				"body":           "This is the issue body with detailed description of the problem.",
				"comments":       3,
				"updated_at":     "2026-02-19T10:00:00Z",
				"repository_url": "https://api.github.com/repos/owner/repo",
				"labels":         []map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	result, err := p.Search(context.Background(), "issue body test", SearchOptions{
		Mode:       "issues",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if !strings.Contains(result.Content, "detailed description") {
		t.Error("expected issue body content in output")
	}
}

func TestGitHubSearchIssuesBodyTruncation(t *testing.T) {
	// Create a body longer than 500 characters
	longBody := strings.Repeat("This is a long body. ", 30) // 630 chars

	mockResp := map[string]interface{}{
		"total_count":        1,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"number":         1,
				"title":          "Long body issue",
				"html_url":       "https://github.com/owner/repo/issues/1",
				"state":          "open",
				"body":           longBody,
				"comments":       0,
				"updated_at":     "2026-02-19T10:00:00Z",
				"repository_url": "https://api.github.com/repos/owner/repo",
				"labels":         []map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	result, err := p.Search(context.Background(), "long body", SearchOptions{
		Mode:       "issues",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if !strings.Contains(result.Content, "...") {
		t.Error("expected truncation indicator '...' in long body")
	}
	// The body in the output should not contain the full longBody
	if strings.Contains(result.Content, longBody) {
		t.Error("expected body to be truncated, but found full body in output")
	}
}

func TestGitHubPrepareQuery(t *testing.T) {
	g := NewGitHub("", "")

	tests := []struct {
		name     string
		query    string
		mode     string
		expected string
	}{
		{
			name:     "issues mode appends is:issue",
			query:    "repo:owner/repo bug",
			mode:     "issues",
			expected: "repo:owner/repo bug is:issue",
		},
		{
			name:     "issues mode preserves existing is:issue",
			query:    "repo:owner/repo is:issue bug",
			mode:     "issues",
			expected: "repo:owner/repo is:issue bug",
		},
		{
			name:     "issues mode preserves is:pull-request",
			query:    "repo:owner/repo is:pull-request",
			mode:     "issues",
			expected: "repo:owner/repo is:pull-request",
		},
		{
			name:     "repos mode unchanged",
			query:    "go web framework",
			mode:     "repos",
			expected: "go web framework",
		},
		{
			name:     "code mode unchanged",
			query:    "http.ListenAndServe",
			mode:     "code",
			expected: "http.ListenAndServe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.prepareQuery(tt.query, tt.mode, GitHubQualifiers{})
			if result != tt.expected {
				t.Errorf("prepareQuery(%q, %q) = %q, want %q",
					tt.query, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestPrepareQueryWithQualifiers(t *testing.T) {
	g := NewGitHub("", "")

	tests := []struct {
		name       string
		query      string
		mode       string
		qualifiers GitHubQualifiers
		expected   string
	}{
		{
			name:       "language qualifier appended",
			query:      "web framework",
			mode:       "repos",
			qualifiers: GitHubQualifiers{Language: "go"},
			expected:   "web framework language:go",
		},
		{
			name:       "multiple qualifiers",
			query:      "cli tool",
			mode:       "repos",
			qualifiers: GitHubQualifiers{Language: "go", Stars: ">1000"},
			expected:   "cli tool language:go stars:>1000",
		},
		{
			name:       "code mode with filename",
			query:      "http.ListenAndServe",
			mode:       "code",
			qualifiers: GitHubQualifiers{Language: "go", Filename: "main.go"},
			expected:   "http.ListenAndServe language:go filename:main.go",
		},
		{
			name:       "issues mode with state and label",
			query:      "deadlock",
			mode:       "issues",
			qualifiers: GitHubQualifiers{State: "open", Label: "bug"},
			expected:   "deadlock is:open label:bug is:issue",
		},
		{
			name:       "issues mode state overrides auto is:issue",
			query:      "memory leak",
			mode:       "issues",
			qualifiers: GitHubQualifiers{State: "closed"},
			expected:   "memory leak is:closed is:issue",
		},
		{
			name:       "empty qualifiers unchanged",
			query:      "go web framework",
			mode:       "repos",
			qualifiers: GitHubQualifiers{},
			expected:   "go web framework",
		},
		{
			name:       "user and org qualifiers",
			query:      "router",
			mode:       "repos",
			qualifiers: GitHubQualifiers{User: "gorilla"},
			expected:   "router user:gorilla",
		},
		{
			name:       "repo qualifier for code search",
			query:      "func main",
			mode:       "code",
			qualifiers: GitHubQualifiers{Repo: "golang/go", Extension: "go"},
			expected:   "func main repo:golang/go extension:go",
		},
		{
			name:       "repos with topic and license",
			query:      "machine learning",
			mode:       "repos",
			qualifiers: GitHubQualifiers{Topic: "deep-learning", License: "mit"},
			expected:   "machine learning topic:deep-learning license:mit",
		},
		{
			name:       "issues with author and assignee",
			query:      "crash",
			mode:       "issues",
			qualifiers: GitHubQualifiers{Author: "octocat", Assignee: "mona"},
			expected:   "crash author:octocat assignee:mona is:issue",
		},
		{
			name:       "in qualifier",
			query:      "kubernetes",
			mode:       "repos",
			qualifiers: GitHubQualifiers{In: "name,description"},
			expected:   "kubernetes in:name,description",
		},
		{
			name:       "repos with archived and fork",
			query:      "framework",
			mode:       "repos",
			qualifiers: GitHubQualifiers{Archived: "false", Fork: "true"},
			expected:   "framework archived:false fork:true",
		},
		{
			name:       "repos with pushed and created",
			query:      "new project",
			mode:       "repos",
			qualifiers: GitHubQualifiers{Pushed: ">2024-01-01", Created: ">2023-01-01"},
			expected:   "new project pushed:>2024-01-01 created:>2023-01-01",
		},
		{
			name:       "code with path qualifier",
			query:      "config",
			mode:       "code",
			qualifiers: GitHubQualifiers{Path: "src/config"},
			expected:   "config path:src/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.prepareQuery(tt.query, tt.mode, tt.qualifiers)
			if result != tt.expected {
				t.Errorf("prepareQuery(%q, %q, %+v) = %q, want %q",
					tt.query, tt.mode, tt.qualifiers, result, tt.expected)
			}
		})
	}
}

func TestGitHubSearchIssuesAppendsIsIssue(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count":        1,
		"incomplete_results": false,
		"items": []map[string]interface{}{
			{
				"number":         1,
				"title":          "Test issue",
				"html_url":       "https://github.com/owner/repo/issues/1",
				"state":          "open",
				"comments":       0,
				"updated_at":     "2026-02-19T10:00:00Z",
				"repository_url": "https://api.github.com/repos/owner/repo",
				"labels":         []map[string]interface{}{},
			},
		},
	}

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	_, err := p.Search(context.Background(), "deadlock goroutine", SearchOptions{
		Mode:       "issues",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if !strings.Contains(capturedQuery, "is:issue") {
		t.Errorf("expected query to contain 'is:issue', got %q", capturedQuery)
	}
}

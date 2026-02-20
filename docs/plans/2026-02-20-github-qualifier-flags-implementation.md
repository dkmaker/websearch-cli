# GitHub Qualifier Flags Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Expose GitHub search qualifiers as first-class `--gh-*` CLI flags so AI agents can construct precise keyword queries.

**Architecture:** Add `GitHubQualifiers` struct to `SearchOptions`, register 20 cobra flags with `--gh-` prefix, build GitHub query string from qualifiers in `prepareQuery()`, validate flags are GitHub-only, include qualifiers in cache key.

**Tech Stack:** Go, cobra, httptest for tests

---

### Task 1: Add GitHubQualifiers struct to provider package

**Files:**
- Modify: `internal/provider/provider.go:17-26`

**Step 1: Write the failing test**

Create test in `internal/provider/provider_test.go` (if it doesn't exist, create it):

```go
package provider

import "testing"

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestGitHubQualifiers -v`
Expected: FAIL — `GitHubQualifiers` type exists but has no `IsEmpty()` or `CacheKey()` methods

**Step 3: Write implementation**

Add to `internal/provider/provider.go` after the `SearchOptions` struct:

```go
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
```

Add `"strings"` to the import block in `provider.go`.

Add `GitHub GitHubQualifiers` field to the `SearchOptions` struct:

```go
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
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/provider/ -run TestGitHubQualifiers -v`
Expected: PASS

**Step 5: Run all existing tests to check for regressions**

Run: `go test ./... -v`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/provider/provider.go internal/provider/provider_test.go
git commit -m "feat(github): add GitHubQualifiers struct to SearchOptions"
```

---

### Task 2: Update prepareQuery to build query from qualifiers

**Files:**
- Modify: `internal/provider/github.go:344-355`
- Modify: `internal/provider/github_test.go:375-425`

**Step 1: Write failing tests for qualifier-based query building**

Add to `internal/provider/github_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestPrepareQueryWithQualifiers -v`
Expected: FAIL — `prepareQuery` only takes 2 args, not 3

**Step 3: Update prepareQuery signature and implementation**

Replace `prepareQuery` in `internal/provider/github.go`:

```go
// prepareQuery builds the GitHub API query string from the free-text query
// and structured qualifiers.
func (g *GitHub) prepareQuery(query string, mode string, qualifiers GitHubQualifiers) string {
	var parts []string
	if query != "" {
		parts = append(parts, query)
	}

	// Common qualifiers (all modes)
	if qualifiers.Language != "" {
		parts = append(parts, "language:"+qualifiers.Language)
	}
	if qualifiers.User != "" {
		parts = append(parts, "user:"+qualifiers.User)
	}
	if qualifiers.Org != "" {
		parts = append(parts, "org:"+qualifiers.Org)
	}
	if qualifiers.Repo != "" {
		parts = append(parts, "repo:"+qualifiers.Repo)
	}
	if qualifiers.In != "" {
		parts = append(parts, "in:"+qualifiers.In)
	}

	// Repos-applicable qualifiers
	if qualifiers.Stars != "" {
		parts = append(parts, "stars:"+qualifiers.Stars)
	}
	if qualifiers.Topic != "" {
		parts = append(parts, "topic:"+qualifiers.Topic)
	}
	if qualifiers.License != "" {
		parts = append(parts, "license:"+qualifiers.License)
	}
	if qualifiers.Archived != "" {
		parts = append(parts, "archived:"+qualifiers.Archived)
	}
	if qualifiers.Fork != "" {
		parts = append(parts, "fork:"+qualifiers.Fork)
	}
	if qualifiers.Pushed != "" {
		parts = append(parts, "pushed:"+qualifiers.Pushed)
	}
	if qualifiers.Created != "" {
		parts = append(parts, "created:"+qualifiers.Created)
	}

	// Code-applicable qualifiers
	if qualifiers.Filename != "" {
		parts = append(parts, "filename:"+qualifiers.Filename)
	}
	if qualifiers.Extension != "" {
		parts = append(parts, "extension:"+qualifiers.Extension)
	}
	if qualifiers.Path != "" {
		parts = append(parts, "path:"+qualifiers.Path)
	}

	// Issues-applicable qualifiers
	if qualifiers.State != "" {
		parts = append(parts, "is:"+qualifiers.State)
	}
	if qualifiers.Label != "" {
		parts = append(parts, "label:"+qualifiers.Label)
	}
	if qualifiers.Author != "" {
		parts = append(parts, "author:"+qualifiers.Author)
	}
	if qualifiers.Assignee != "" {
		parts = append(parts, "assignee:"+qualifiers.Assignee)
	}

	// Issues mode: auto-append is:issue if no is: qualifier present
	if mode == "issues" {
		hasIs := false
		for _, p := range parts {
			if strings.HasPrefix(p, "is:") {
				hasIs = true
				break
			}
		}
		if !hasIs || (qualifiers.State != "" && !strings.Contains(strings.Join(parts, " "), "is:issue") && !strings.Contains(strings.Join(parts, " "), "is:pull-request")) {
			parts = append(parts, "is:issue")
		}
	}

	return strings.Join(parts, " ")
}
```

**Important:** The issues mode logic needs care. The `State` qualifier maps to `is:open` or `is:closed`, which is *different* from `is:issue`. We need `is:issue` appended when neither `is:issue` nor `is:pull-request` is in the query. Update the logic:

```go
	// Issues mode: always append is:issue unless user query already has it
	if mode == "issues" {
		joined := strings.Join(parts, " ")
		if !strings.Contains(joined, "is:issue") && !strings.Contains(joined, "is:pull-request") {
			parts = append(parts, "is:issue")
		}
	}
```

**Step 4: Update the call site in Search()**

In `github.go`, update line 94 to pass qualifiers:

```go
query = g.prepareQuery(query, opts.Mode, opts.GitHub)
```

**Step 5: Update existing prepareQuery tests**

The existing `TestGitHubPrepareQuery` tests call `prepareQuery` with 2 args. Update them to pass 3 args (empty `GitHubQualifiers{}`):

```go
result := g.prepareQuery(tt.query, tt.mode, GitHubQualifiers{})
```

**Step 6: Run all tests**

Run: `go test ./internal/provider/ -v`
Expected: All PASS

**Step 7: Commit**

```bash
git add internal/provider/github.go internal/provider/github_test.go
git commit -m "feat(github): build query string from GitHubQualifiers"
```

---

### Task 3: Add sort parameter support to buildURL

**Files:**
- Modify: `internal/provider/github.go:140-152`

**Step 1: Write failing test**

Add to `internal/provider/github_test.go`:

```go
func TestGitHubReposSortByStars(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count": 1, "incomplete_results": false,
		"items": []map[string]interface{}{
			{"full_name": "test/repo", "html_url": "https://github.com/test/repo",
				"description": "test", "stargazers_count": 100, "language": "Go",
				"updated_at": "2026-02-19T10:00:00Z", "topics": []string{}},
		},
	}

	var capturedSort, capturedOrder string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSort = r.URL.Query().Get("sort")
		capturedOrder = r.URL.Query().Get("order")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	_, err := p.Search(context.Background(), "test", SearchOptions{
		Mode:       "repos",
		MaxResults: 10,
		GitHub:     GitHubQualifiers{Sort: "stars"},
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if capturedSort != "stars" {
		t.Errorf("expected sort=stars, got %q", capturedSort)
	}
	if capturedOrder != "desc" {
		t.Errorf("expected order=desc, got %q", capturedOrder)
	}
}

func TestGitHubReposDefaultSortStars(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count": 1, "incomplete_results": false,
		"items": []map[string]interface{}{
			{"full_name": "test/repo", "html_url": "https://github.com/test/repo",
				"description": "test", "stargazers_count": 100, "language": "Go",
				"updated_at": "2026-02-19T10:00:00Z", "topics": []string{}},
		},
	}

	var capturedSort string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedSort = r.URL.Query().Get("sort")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	_, err := p.Search(context.Background(), "test", SearchOptions{
		Mode:       "repos",
		MaxResults: 10,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if capturedSort != "stars" {
		t.Errorf("expected default sort=stars for repos, got %q", capturedSort)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestGitHubReposSort -v`
Expected: FAIL — `buildURL` doesn't support sort

**Step 3: Update buildURL and searchRepos**

Update `buildURL` to accept sort/order:

```go
func (g *GitHub) buildURL(path string, query string, maxResults int, sort string) string {
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
	if sort != "" {
		params.Set("sort", sort)
		params.Set("order", "desc")
	}
	return g.baseURL + path + "?" + params.Encode()
}
```

Update `searchRepos` to default sort to `stars`:

```go
func (g *GitHub) searchRepos(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	sort := opts.GitHub.Sort
	if sort == "" {
		sort = "stars"
	}
	endpoint := g.buildURL("/search/repositories", query, opts.MaxResults, sort)
	// ... rest unchanged
}
```

Update `searchCode` and `searchIssues` to pass empty sort:

```go
endpoint := g.buildURL("/search/code", query, opts.MaxResults, "")
endpoint := g.buildURL("/search/issues", query, opts.MaxResults, "")
```

**Step 4: Run all tests**

Run: `go test ./internal/provider/ -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/provider/github.go internal/provider/github_test.go
git commit -m "feat(github): add sort parameter to repos search, default to stars"
```

---

### Task 4: Register --gh-* CLI flags and wire to SearchOptions

**Files:**
- Modify: `internal/cli/root.go:26-39` (flag vars)
- Modify: `internal/cli/root.go:62-75` (init)
- Modify: `internal/cli/root.go:153-160` (opts building)

**Step 1: Add flag variables**

Add after the existing flag vars block in `root.go`:

```go
// GitHub qualifier flags
var (
	flagGHLanguage  string
	flagGHUser      string
	flagGHOrg       string
	flagGHRepo      string
	flagGHStars     string
	flagGHTopic     string
	flagGHLicense   string
	flagGHArchived  string
	flagGHFork      string
	flagGHPushed    string
	flagGHCreated   string
	flagGHSort      string
	flagGHFilename  string
	flagGHExtension string
	flagGHPath      string
	flagGHState     string
	flagGHLabel     string
	flagGHAuthor    string
	flagGHAssignee  string
	flagGHIn        string
)
```

**Step 2: Register flags in init()**

Add after existing flag registrations:

```go
	// GitHub qualifier flags
	rootCmd.Flags().StringVar(&flagGHLanguage, "gh-language", "", "GitHub: filter by language (e.g., go, python)")
	rootCmd.Flags().StringVar(&flagGHUser, "gh-user", "", "GitHub: filter by user/owner")
	rootCmd.Flags().StringVar(&flagGHOrg, "gh-org", "", "GitHub: filter by organization")
	rootCmd.Flags().StringVar(&flagGHRepo, "gh-repo", "", "GitHub: filter by repo (owner/name)")
	rootCmd.Flags().StringVar(&flagGHStars, "gh-stars", "", "GitHub: filter by stars (e.g., >100, 10..50)")
	rootCmd.Flags().StringVar(&flagGHTopic, "gh-topic", "", "GitHub: filter by topic")
	rootCmd.Flags().StringVar(&flagGHLicense, "gh-license", "", "GitHub: filter by license (e.g., mit, apache-2.0)")
	rootCmd.Flags().StringVar(&flagGHArchived, "gh-archived", "", "GitHub: filter archived repos (true/false)")
	rootCmd.Flags().StringVar(&flagGHFork, "gh-fork", "", "GitHub: filter forks (true/only)")
	rootCmd.Flags().StringVar(&flagGHPushed, "gh-pushed", "", "GitHub: filter by last push date (e.g., >2024-01-01)")
	rootCmd.Flags().StringVar(&flagGHCreated, "gh-created", "", "GitHub: filter by creation date (e.g., >2024-01-01)")
	rootCmd.Flags().StringVar(&flagGHSort, "gh-sort", "", "GitHub: sort repos by (stars, forks, updated)")
	rootCmd.Flags().StringVar(&flagGHFilename, "gh-filename", "", "GitHub: filter code by filename")
	rootCmd.Flags().StringVar(&flagGHExtension, "gh-extension", "", "GitHub: filter code by file extension")
	rootCmd.Flags().StringVar(&flagGHPath, "gh-path", "", "GitHub: filter code by directory path")
	rootCmd.Flags().StringVar(&flagGHState, "gh-state", "", "GitHub: filter issues by state (open/closed)")
	rootCmd.Flags().StringVar(&flagGHLabel, "gh-label", "", "GitHub: filter issues by label")
	rootCmd.Flags().StringVar(&flagGHAuthor, "gh-author", "", "GitHub: filter issues by author")
	rootCmd.Flags().StringVar(&flagGHAssignee, "gh-assignee", "", "GitHub: filter issues by assignee")
	rootCmd.Flags().StringVar(&flagGHIn, "gh-in", "", "GitHub: search in fields (name,description / title,body)")
```

**Step 3: Build GitHubQualifiers in run() and add to SearchOptions**

Add a helper function and update the opts building section:

```go
func buildGitHubQualifiers() provider.GitHubQualifiers {
	return provider.GitHubQualifiers{
		Language:  flagGHLanguage,
		User:      flagGHUser,
		Org:       flagGHOrg,
		Repo:      flagGHRepo,
		Stars:     flagGHStars,
		Topic:     flagGHTopic,
		License:   flagGHLicense,
		Archived:  flagGHArchived,
		Fork:      flagGHFork,
		Pushed:    flagGHPushed,
		Created:   flagGHCreated,
		Sort:      flagGHSort,
		Filename:  flagGHFilename,
		Extension: flagGHExtension,
		Path:      flagGHPath,
		State:     flagGHState,
		Label:     flagGHLabel,
		Author:    flagGHAuthor,
		Assignee:  flagGHAssignee,
		In:        flagGHIn,
	}
}
```

Update the opts building (around line 154):

```go
	ghQualifiers := buildGitHubQualifiers()

	// Validate --gh-* flags are only used with GitHub provider
	if !ghQualifiers.IsEmpty() && providerName != "github" {
		return fmt.Errorf("--gh-* flags are only supported with the GitHub provider (current: %s)", providerName)
	}

	opts := provider.SearchOptions{
		Mode:         mode,
		MaxTokens:    prof.MaxTokens,
		MaxResults:   prof.MaxResults,
		SystemPrompt: prof.SystemPrompt,
		DomainFilter: prof.DomainFilter,
		GitHub:       ghQualifiers,
	}
```

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat(github): register --gh-* CLI flags and wire to SearchOptions"
```

---

### Task 5: Include qualifiers in cache key

**Files:**
- Modify: `internal/cli/root.go:168` (cache key line)

**Step 1: Write test**

This is a behavioral change. The existing cache key at line 168 is:

```go
cacheKey := cache.Key(query, flagProfile, mode, providerName)
```

Update to include qualifiers:

```go
cacheKey := cache.Key(query, flagProfile, mode, providerName, ghQualifiers.CacheKey())
```

The `cache.Key()` function already accepts variadic `...string`, so no changes needed to the cache package.

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat(github): include qualifiers in cache key"
```

---

### Task 6: Update self-primer with GitHub qualifier flags

**Files:**
- Modify: `internal/cli/root.go:336-406` (buildSelfPrimer)

**Step 1: Update the self-primer output**

Add after the "Key flags" line (around line 372):

```go
	sb.WriteString("GitHub flags: --gh-language, --gh-stars, --gh-sort, --gh-user, --gh-org, --gh-repo, --gh-topic,\n")
	sb.WriteString("  --gh-license, --gh-archived, --gh-fork, --gh-pushed, --gh-created, --gh-in,\n")
	sb.WriteString("  --gh-filename, --gh-extension, --gh-path (code), --gh-state, --gh-label, --gh-author, --gh-assignee (issues)\n")
```

Update the GitHub example in the `showExamples` block:

```go
	sb.WriteString("  websearch -p github --gh-language go --gh-stars '>1000' \"web framework\"  # github repos with qualifiers\n")
	sb.WriteString("  websearch -p github -m issues --gh-state open --gh-label bug \"crash\"     # github issues filtered\n")
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Build and verify self-primer output**

Run: `go build -o websearch ./cmd/websearch/ && ./websearch --show-examples`
Expected: Output includes GitHub flags section and updated examples

**Step 4: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat(github): show --gh-* flags in self-primer output"
```

---

### Task 7: Integration test — end-to-end with mock server

**Files:**
- Modify: `internal/provider/github_test.go`

**Step 1: Write integration test**

```go
func TestGitHubSearchReposWithQualifiers(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count": 1, "incomplete_results": false,
		"items": []map[string]interface{}{
			{"full_name": "spf13/cobra", "html_url": "https://github.com/spf13/cobra",
				"description": "A Commander for modern Go CLI interactions",
				"stargazers_count": 38000, "language": "Go",
				"updated_at": "2026-02-19T10:00:00Z", "topics": []string{"cli", "go"}},
		},
	}

	var capturedQuery, capturedSort string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		capturedSort = r.URL.Query().Get("sort")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewGitHub("", server.URL)
	result, err := p.Search(context.Background(), "cli framework", SearchOptions{
		Mode:       "repos",
		MaxResults: 10,
		GitHub: GitHubQualifiers{
			Language: "go",
			Stars:    ">1000",
			Topic:    "cli",
		},
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	// Verify qualifiers were appended to query
	if !strings.Contains(capturedQuery, "language:go") {
		t.Errorf("expected language:go in query, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "stars:>1000") {
		t.Errorf("expected stars:>1000 in query, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "topic:cli") {
		t.Errorf("expected topic:cli in query, got %q", capturedQuery)
	}

	// Verify default sort
	if capturedSort != "stars" {
		t.Errorf("expected sort=stars, got %q", capturedSort)
	}

	// Verify result content
	if !strings.Contains(result.Content, "spf13/cobra") {
		t.Error("expected cobra in content")
	}
}

func TestGitHubSearchIssuesWithQualifiers(t *testing.T) {
	mockResp := map[string]interface{}{
		"total_count": 1, "incomplete_results": false,
		"items": []map[string]interface{}{
			{"number": 1, "title": "Bug report", "html_url": "https://github.com/test/repo/issues/1",
				"state": "open", "body": "Details here", "comments": 2,
				"updated_at": "2026-02-19T10:00:00Z",
				"repository_url": "https://api.github.com/repos/test/repo",
				"labels": []map[string]interface{}{{"name": "bug"}}},
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
	_, err := p.Search(context.Background(), "crash", SearchOptions{
		Mode:   "issues",
		GitHub: GitHubQualifiers{State: "open", Label: "bug", Repo: "test/repo"},
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if !strings.Contains(capturedQuery, "is:open") {
		t.Errorf("expected is:open in query, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "label:bug") {
		t.Errorf("expected label:bug in query, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "repo:test/repo") {
		t.Errorf("expected repo:test/repo in query, got %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "is:issue") {
		t.Errorf("expected is:issue auto-appended, got %q", capturedQuery)
	}
}
```

**Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/provider/github_test.go
git commit -m "test(github): add integration tests for qualifier-based search"
```

---

### Task 8: Run go mod tidy and final verification

**Step 1: Run go mod tidy**

Run: `go mod tidy`

**Step 2: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

**Step 3: Build binary and smoke test**

Run: `go build -o websearch ./cmd/websearch/`
Run: `./websearch --help` — verify `--gh-*` flags appear
Run: `./websearch` — verify self-primer shows GitHub flags

**Step 4: Commit if go.mod/go.sum changed**

```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy"
```

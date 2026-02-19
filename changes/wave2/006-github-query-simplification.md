# Change 006: Smart Query Simplification for GitHub Provider

**Status:** Proposed
**Priority:** High
**Affected component:** `internal/provider/github.go`

---

## Problem

GitHub's search API performs strict AND-matching across all query keywords. Long natural-language queries — which work well with Perplexity — return 0 results or irrelevant noise from GitHub. This caused **2 hard failures** and **1 relevance failure** in the 50-query stress test:

### Hard Failures (exit code 1, empty output)

- **Query #30** (`profile: github, mode: repos`) — `"circuit breaker Go HTTP resilience half-open state language:go"`
  - GitHub returned 0 repos. The provider raised `fmt.Errorf("no results found for query")`.
  - File: `result_output/raw_30_best-practices.md` (empty)
  - Expected: sony/gobreaker, afex/hystrix-go, etc.

- **Query #41** (`profile: github, mode: repos`) — `"pyproject.toml setuptools console_scripts optional-dependencies editable install template"`
  - GitHub returned 0 repos. Same hard error.
  - File: `result_output/raw_41_bootstrapping.md` (empty)
  - Expected: modern Python project templates

### Relevance Failure

- **Query #02** (`profile: github, mode: issues`) — `"go modules incompatible major version transitive dependency conflict replace directive"`
  - GitHub returned 10 issues, but from ceylon, angular-cli, zenscript — completely unrelated to Go modules.
  - The query had enough common words (`version`, `conflict`, `module`) to match random issues, but the AND-matching with all 8 keywords yielded noise.
  - File: `result_output/raw_02_dependency-management.md`

### Root Cause

The GitHub provider passes the user query verbatim to the API (`github.go:94`). GitHub search is designed for keyword-style queries (2-5 terms), not natural-language sentences. The mismatch is fundamental — the same query that gives Perplexity great context gives GitHub too many constraints.

## Proposed Solution

Add a `simplifyQuery()` function in `github.go` that transforms natural-language queries into GitHub-friendly keyword queries before hitting the API:

```go
func (g *GitHub) simplifyQuery(query string, mode string) string {
    // 1. Extract and preserve GitHub qualifiers (language:go, stars:>100, is:issue, etc.)
    // 2. Remove common filler/stop words (the, is, how, when, what, etc.)
    // 3. If remaining non-qualifier words > 5, keep only the top 5 most distinctive
    // 4. Rejoin qualifiers + keywords
    return simplified
}
```

Apply it in `prepareQuery()` before the existing mode-specific logic:

```go
func (g *GitHub) prepareQuery(query string, mode string) string {
    query = g.simplifyQuery(query, mode)
    switch mode {
    case "issues":
        if !strings.Contains(query, "is:issue") && !strings.Contains(query, "is:pull-request") {
            query = query + " is:issue"
        }
    }
    return query
}
```

### Stop Word List (starter)

```go
var githubStopWords = map[string]bool{
    "what": true, "how": true, "the": true, "is": true, "a": true,
    "to": true, "for": true, "in": true, "of": true, "and": true,
    "or": true, "with": true, "that": true, "when": true, "where": true,
    "from": true, "by": true, "an": true, "my": true, "does": true,
    "should": true, "can": true, "way": true, "correct": true,
    "using": true, "between": true, "also": true, "versus": true,
    "without": true, "after": true, "only": true, "different": true,
}
```

### Repos Mode Enhancement

For `repos` mode, add `sort=stars` to the API request to surface higher-quality repositories first:

```go
func (g *GitHub) buildURL(path string, query string, maxResults int) string {
    params := url.Values{}
    params.Set("q", query)
    if path == "/search/repositories" {
        params.Set("sort", "stars")
        params.Set("order", "desc")
    }
    // ...
}
```

## Impact

- **Fixes 2 hard failures** (queries 30 and 41) — these would return real results
- **Improves relevance** for queries like 02 — fewer keywords = tighter matching
- **No change** for already-short queries or queries that are pure GitHub syntax
- Stars sorting surfaces community-vetted repos instead of random matches

## Test Cases

```go
func TestSimplifyQuery(t *testing.T) {
    g := NewGitHub("", "")
    tests := []struct {
        input    string
        expected string // approximate — check keyword count and qualifier preservation
    }{
        {"circuit breaker Go HTTP resilience half-open state language:go", "circuit breaker resilience half-open language:go"},
        {"pyproject.toml setuptools console_scripts optional-dependencies editable install template", "pyproject.toml setuptools console_scripts optional-dependencies template"},
        {"cobra CLI", "cobra CLI"}, // short query unchanged
        {"language:python stars:>100 fastapi", "language:python stars:>100 fastapi"}, // qualifiers preserved
    }
    // ...
}
```

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


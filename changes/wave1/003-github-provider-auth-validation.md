# Change 003: GitHub Provider Auth Validation

**Status:** Proposed
**Priority:** High
**Affected component:** `internal/cli/root.go`, `internal/provider/github.go`

---

## Problem

The GitHub provider has **three different failure modes** when `GITHUB_TOKEN` is not set, and none of them are caught early:

### Failure Mode 1: Silent empty results (repos mode)

- **Query #30** (score 0.0) — `repos` mode returned exit code 0, empty stdout, no stderr. The agent received absolutely nothing with no indication of why.
  - File: `result_output/result_30_best-practices.json`
  - Command: `./websearch --no-cache -p github "circuit breaker Go HTTP resilience half-open state language:go"`
  - `success: true, had_errors: false, response_length_words: 0`

### Failure Mode 2: Hard auth error (code mode)

- **Query #40** (score 0.0) — `code` mode correctly returns an error, but only at runtime after the agent has already committed to this search path.
  - File: `result_output/result_40_best-practices.json`
  - stderr: `Error: search failed: code search requires authentication. Set GITHUB_TOKEN`

### Failure Mode 3: Rate limit error (repos mode, unauthenticated)

- **Query #41** (score 0.0) — `repos` mode hit the unauthenticated rate limit (60 req/hour for the IP).
  - File: `result_output/result_41_bootstrapping.json`
  - stderr: `Error: search failed: github API error (status 403): {"message":"API rate limit exceeded for 80.209.65.165..."}`

All three share the same root cause: `GITHUB_TOKEN` is not set. But the agent gets three completely different experiences. The `code` mode at least has an explicit check (`github.go:212`), while `repos` and `issues` modes don't.

### Additional context: issues mode is low-value without auth

Even when the GitHub provider succeeds, `issues` mode returns only metadata (titles, URLs) without discussion content:

- **Query #2** (score 1.8) — Asked about Go module incompatibility. Got issue titles only.
- **Query #16** (score 2.3) — Asked about GitHub Actions fork PR permissions. Got issue titles only.

## Proposed Solution

### Option A: Fail fast in resolveProvider (recommended)

In `internal/cli/root.go`, the `resolveProvider()` function already checks for API key presence for Perplexity and Brave. Add the same check for GitHub:

```go
// When github is explicitly requested but GITHUB_TOKEN is missing
if requested == "github" && githubKey == "" {
    return "", "", fmt.Errorf("GitHub provider requires GITHUB_TOKEN environment variable")
}
```

If the `github` profile is loaded (not explicitly requested), fall back to another provider with a warning:

```
warning: github profile selected but GITHUB_TOKEN not set; falling back to perplexity
```

### Option B: Validate in the provider constructor

Add a token check in `NewGitHubProvider()` that returns an error if no token is provided, rather than letting individual mode handlers fail inconsistently.

### Option C: Warn but allow (least disruptive)

Keep current behavior but add a startup warning to stderr:

```
warning: GITHUB_TOKEN not set; GitHub searches will be rate-limited and code search will fail
```

## Impact

- Eliminates 3 zero-score test results (queries #30, #40, #41)
- Provides consistent, early error messages instead of three different failure modes
- Matches existing patterns for Perplexity/Brave key validation
- The agent gets a clear signal to either set the token or use a different provider

## Stress Test Context

5 of 50 queries used the GitHub provider. Results:
- 3 scored 0.0 (auth/rate-limit failures)
- 1 scored 1.8 (issues metadata only)
- 1 scored 2.3 (issues metadata only)
- **GitHub provider average: 0.82/5.0** (vs 3.19 for Perplexity, 2.88 for Brave)

## Open Questions

This is a very new profile - i think we should research what is actually possible and test with curl to see what happens and what we actually get back before we descide - make a plan how we can review what is possible and how the response looks like

## Plan

### Step 1: Map GitHub Search API capabilities (investigation)

Test each mode with and without `GITHUB_TOKEN` to document what actually works:

```bash
# --- REPOS mode, unauthenticated ---
curl -s "https://api.github.com/search/repositories?q=circuit+breaker+Go&per_page=3" \
  -H "Accept: application/json" | jq '.total_count, .items[0] | {full_name, description, stargazers_count}'

# --- REPOS mode, authenticated ---
curl -s "https://api.github.com/search/repositories?q=circuit+breaker+Go&per_page=3" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/json" | jq '.total_count, .items[0] | {full_name, description, stargazers_count}'

# --- ISSUES mode, unauthenticated ---
curl -s "https://api.github.com/search/issues?q=deadlock+goroutine&per_page=3" \
  -H "Accept: application/json" | jq '.total_count, .items[0] | {title, state, comments, html_url}'

# --- ISSUES mode, authenticated (check if body is available) ---
curl -s "https://api.github.com/search/issues?q=deadlock+goroutine&per_page=3" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/json" | jq '.items[0] | {title, body}'

# --- CODE mode (requires auth) ---
curl -s "https://api.github.com/search/code?q=circuit+breaker+language:go&per_page=3" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.text-match+json" | jq '.items[0] | {path, repository: .repository.full_name, text_matches: [.text_matches[0].fragment]}'

# --- Check rate limits ---
curl -s "https://api.github.com/rate_limit" -H "Authorization: Bearer $GITHUB_TOKEN" | jq '.resources.search'
curl -s "https://api.github.com/rate_limit" | jq '.resources.search'
```

### Step 2: Document what's returned vs what's useful

From the curl results, build a table:

| Mode | Auth Required? | Useful Fields | Missing Fields | Value for Agent |
|------|---------------|---------------|----------------|-----------------|
| repos | No (but rate-limited) | ? | ? | ? |
| issues | No (but rate-limited) | ? | body content? | ? |
| code | Yes | ? | ? | ? |

Key questions to answer:
- Does the issues search response include `body` (the issue text)? Currently we don't parse it.
- Does the code search `text_matches` fragment give enough context to be useful?
- What are the rate limits for auth vs unauth? (10 req/min unauth, 30 req/min auth for search)

### Step 3: Identify quick wins for GitHub provider quality

Based on step 2 findings, potential improvements:
- **Issues mode**: If `body` is in the response, include the first 500 chars of issue body in the output — this would dramatically improve the 1.8/2.3 scores for queries #2 and #16
- **Repos mode**: Include README content via a follow-up API call to `/repos/{owner}/{repo}/readme`
- **Code mode**: Show more text_match fragments

### Step 4: Add auth pre-check

After we know what works authenticated vs not, add appropriate validation:
- If `GITHUB_TOKEN` is missing and `github` profile is requested: warn on stderr with specific guidance about what will be limited
- Still allow unauthenticated repos/issues (they work, just rate-limited) but make the limitation clear
- Block code mode without auth (already done at `github.go:212`)

### Step 5: Re-run GitHub queries with token

Re-run queries #2, #16, #30, #40, #41 with `GITHUB_TOKEN` set and the improved output formatting to measure score improvement.

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


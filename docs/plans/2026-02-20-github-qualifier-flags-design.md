# GitHub First-Class Qualifier Flags Design

**Date:** 2026-02-20
**Status:** Approved

## Problem

GitHub's search API is keyword-based, not natural language. AI agents generate verbose queries that fail or return irrelevant results. The current implementation passes queries nearly verbatim. In the 50-query stress test, 2 queries failed completely (0 results) and 2 returned irrelevant results when using the GitHub provider.

## Solution

Expose GitHub search qualifiers as first-class CLI flags with `--gh-` prefix. Flags map directly to GitHub API query qualifiers. GitHub-specific — error if used with other providers.

## New Struct: `GitHubQualifiers`

Added to `internal/provider/provider.go`:

```go
type GitHubQualifiers struct {
    Language  string // language:X
    User      string // user:X
    Org       string // org:X
    Repo      string // repo:X
    Stars     string // stars:>100, stars:100..500
    Topic     string // topic:X
    License   string // license:mit
    Archived  string // archived:true/false
    Fork      string // fork:true/only
    Pushed    string // pushed:>2024-01-01
    Created   string // created:>2024-01-01
    Sort      string // sort param (stars, forks, updated)
    Filename  string // filename:X (code mode)
    Extension string // extension:X (code mode)
    Path      string // path:X (code mode)
    State     string // is:open/closed (issues mode)
    Label     string // label:X (issues mode)
    Author    string // author:X (issues mode)
    Assignee  string // assignee:X (issues mode)
    In        string // in:name,description (repos), in:title,body (issues)
}
```

Added to `SearchOptions`:
```go
type SearchOptions struct {
    // ... existing fields
    GitHub GitHubQualifiers
}
```

## CLI Flags

All prefixed with `--gh-`:

| Flag | Maps to | Applicable modes |
|------|---------|-----------------|
| `--gh-language` | `language:X` | all |
| `--gh-user` | `user:X` | all |
| `--gh-org` | `org:X` | all |
| `--gh-repo` | `repo:X` | all |
| `--gh-in` | `in:X` | all |
| `--gh-stars` | `stars:X` | repos |
| `--gh-topic` | `topic:X` | repos |
| `--gh-license` | `license:X` | repos |
| `--gh-archived` | `archived:X` | repos |
| `--gh-fork` | `fork:X` | repos |
| `--gh-pushed` | `pushed:X` | repos |
| `--gh-created` | `created:X` | repos |
| `--gh-sort` | sort URL param | repos |
| `--gh-filename` | `filename:X` | code |
| `--gh-extension` | `extension:X` | code |
| `--gh-path` | `path:X` | code |
| `--gh-state` | `is:X` | issues |
| `--gh-label` | `label:X` | issues |
| `--gh-author` | `author:X` | issues |
| `--gh-assignee` | `assignee:X` | issues |

## Query Building

`prepareQuery` in `github.go` builds the final `q` parameter by joining the free-text query with qualifier strings. Free-text is passed verbatim (no stop-word filtering).

## Sort Defaults

Repos mode defaults to `sort=stars&order=desc` unless `--gh-sort` specifies otherwise. This surfaces community-vetted repos first.

## Validation

- `--gh-*` flags with non-GitHub provider → error
- Mode-specific flags with wrong mode → silently ignored (not error)

## Cache Key

GitHub qualifiers included in cache key generation to prevent cross-contamination.

## Self-Primer Update

Self-primer output lists available `--gh-*` flags for agent discoverability.

## Files Modified

- `internal/provider/provider.go` — add `GitHubQualifiers` struct to `SearchOptions`
- `internal/provider/github.go` — update `prepareQuery`, `buildURL` signatures
- `internal/provider/github_test.go` — test qualifier building
- `internal/cli/root.go` — register flags, validation, pass qualifiers
- `internal/cache/cache.go` — include qualifiers in key

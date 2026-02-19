# GitHub Provider Design

## Problem

The websearch CLI supports Perplexity and Brave for web search, but AI agents often need to search GitHub for repositories, code, and issues. GitHub's Search API is free, open, and highly relevant for development tasks.

## Design Decisions

- **Single provider, multi-mode** — mirrors Perplexity pattern (one struct, three modes: repos/code/issues)
- **GITHUB_TOKEN env var** — optional for repos/issues, required for code search. Same var as `gh` CLI.
- **Compact markdown output** — token-efficient formatting per mode
- **New builtin profile** — `github.yaml` with default mode `repos`

## Provider Structure

```go
type GitHub struct {
    token   string   // GITHUB_TOKEN, may be empty
    baseURL string   // default: https://api.github.com
    client  *http.Client
}
```

**Modes:**
- `repos` (default) → `GET /search/repositories` — works without auth
- `code` → `GET /search/code` — requires auth (errors without token)
- `issues` → `GET /search/issues` — works without auth

**Constructor:** `NewGitHub(token, baseURL string) *GitHub`

**Auth:** `Authorization: Bearer <token>` header when token is set. Unauthenticated otherwise.

**Rate limits:** 10 req/min unauthenticated, 30 req/min with token. Handled by caching layer.

## Output Formatting

### Repos mode
```
**owner/repo** ⭐1.2k — Description text here
  Language: Go | Updated: 2026-02-15 | Topics: cli, search

**owner/repo2** ⭐456 — Another repo description
  Language: Python | Updated: 2026-01-20
```

### Code mode
```
**owner/repo** src/path/file.go
  ...matched code fragment...

**owner/repo2** lib/other.py
  ...matched code fragment...
```

Requires `Accept: application/vnd.github.text-match+json` header for text_matches.

### Issues mode
```
**owner/repo#42** Bug: something broken (open)
  Labels: bug, priority-high | Comments: 5 | Updated: 2026-02-10

**owner/repo#15** Feature: add search (closed)
  Labels: enhancement | Comments: 12 | Updated: 2026-01-05
```

Sources: each result's `html_url`.

## Profile

`internal/profile/profiles/github.yaml`:
```yaml
name: github
description: GitHub repository and code search
provider: github
mode: repos
system_prompt: ""
max_results: 10
output_format: markdown
```

## Integration Points

| File | Changes |
|------|---------|
| `internal/provider/github.go` | New file — GitHub provider implementation |
| `internal/provider/github_test.go` | New file — tests with httptest mock |
| `internal/cli/root.go` | Add GITHUB_TOKEN check, github case in resolveProvider/validateMode/run, update self-primer |
| `internal/cli/root_test.go` | Add github to resolveProvider and validateMode tests |
| `internal/profile/profiles/github.yaml` | New builtin profile |
| `internal/profile/profile_test.go` | Test github profile loads |

## API Details

- Base URL: `https://api.github.com`
- Auth header: `Authorization: Bearer <token>` (when available)
- Accept: `application/json` (repos/issues), `application/vnd.github.text-match+json` (code)
- Pagination: `per_page` param (max 100), `page` param
- Response: `{ total_count, incomplete_results, items[] }`

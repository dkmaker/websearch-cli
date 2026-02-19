# Change 008: Fallback Provider Chain for GitHub Failures

**Status:** Proposed
**Priority:** Medium
**Affected component:** `internal/cli/root.go`, `internal/provider/provider.go`

---

## Problem

When the GitHub provider returns 0 results, the CLI raises a hard error and the user gets nothing:

```go
// github.go:171-173
if len(repos) == 0 {
    return nil, fmt.Errorf("no results found for query")
}
```

This happened on **2 of 50 queries** (4% failure rate) in the stress test:

- **Query #30** — `"circuit breaker Go HTTP resilience half-open state language:go"` with `profile: github, mode: repos`
- **Query #41** — `"pyproject.toml setuptools console_scripts optional-dependencies editable install template"` with `profile: github, mode: repos`

Both queries are perfectly reasonable — an agent asked for information and got an error instead. Even after Change 006 (query simplification) improves hit rates, there will always be edge cases where GitHub returns 0 results. The agent should still get *something* useful.

### Current Behavior

```
$ ./websearch -p github "some obscure query"
Error: search failed: no results found for query
(exit code 1)
```

The calling agent gets exit 1 and no content. It has to decide on its own whether to retry with a different provider.

## Proposed Solution

Add a fallback mechanism in `run()` that catches "no results" errors from the GitHub provider and retries with Perplexity (if available):

```go
// In run(), after the primary search
result, err := prov.Search(context.Background(), query, opts)
if err != nil && providerName == "github" && isNoResultsError(err) {
    // Try fallback to Perplexity if key is available
    if perplexityKey != "" {
        fallbackMode := "ask"
        if flagMode != "" {
            fallbackMode = flagMode
        }
        fallbackOpts := opts
        fallbackOpts.Mode = fallbackMode
        fallbackProv := provider.NewPerplexity(perplexityKey, "")

        fmt.Fprintf(os.Stderr, "warning: GitHub returned no results, falling back to Perplexity (%s mode)\n", fallbackMode)

        result, err = fallbackProv.Search(context.Background(), query, fallbackOpts)
        if err == nil {
            result.FallbackProvider = "perplexity"
        }
    }
}
```

### Result Struct Addition

Add a metadata field so the agent knows a fallback occurred:

```go
// provider.go
type Result struct {
    // ... existing fields
    FallbackProvider string `json:"fallback_provider,omitempty"`
}
```

This shows up in both markdown metadata and JSON output:

```yaml
---
provider: github
fallback_provider: perplexity
mode: ask
---
```

### Helper Function

```go
func isNoResultsError(err error) bool {
    return strings.Contains(err.Error(), "no results found")
}
```

## Impact

- **Eliminates 0-content failures** for GitHub queries when Perplexity is available
- Agent gets a useful answer instead of an error, with metadata indicating the fallback
- If Perplexity key is not available, behavior is unchanged (error propagates)
- Slight latency increase on fallback (one additional API call), but only when GitHub already failed
- The fallback uses `ask` mode by default (cheapest/fastest Perplexity mode), not the GitHub mode

## Interaction with Change 006

These two changes are complementary:
- **006** reduces how often GitHub returns 0 results (prevention)
- **008** handles the cases that still slip through (recovery)

With both, the GitHub provider path should have near-zero hard failures.

## Open Questions

- Should the fallback also apply to Brave provider failures? Currently Brave doesn't have the same 0-result issue, but it could be generalized.
- Should the fallback mode be configurable per profile? E.g., a `fallback_provider` and `fallback_mode` field in the profile YAML.
- Should we cache the fallback result under the original cache key, or a modified key that includes the fallback provider?

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


# Change 005: Response Metadata Flag (`--meta`)

**Status:** Proposed
**Priority:** Medium
**Affected component:** `internal/cli/root.go`, `internal/output/formatter.go`

---

## Problem

AI agents using `websearch` treat it as a black box. The tool provides no structured metadata about what happened during a search — which provider was used, whether the mode was auto-adjusted, whether the response was truncated, or how many sources were found. The agent must either parse the prose output or ignore these signals entirely.

This matters in several stress test scenarios:

### Truncation is invisible

- **Queries #15, #21** (research mode) — Both responses were truncated mid-sentence. The agent has no signal that content was cut off. It would need to notice the mid-sentence ending itself, which is unreliable.
  - `result_15_architecture.json`: 1,213 words, truncated before schema-per-tenant section
  - `result_21_architecture.json`: 1,232 words, truncated before projection architecture

### Mode auto-adjustment goes to stderr only

- **All 5 Brave queries** (#11, #23, #29, #34, #48) emitted `warning: mode adjusted to "web" for brave provider` to stderr. This is good but inconsistent — truncation produces no warning, while mode adjustment does.
  - `result_48_bootstrapping.json`: stderr shows mode adjustment, but the agent may not capture stderr

### Provider fallback is silent

When a provider is unavailable and the system falls back, the agent doesn't know it's getting results from a different provider than expected. This changes the expected quality characteristics without notification.

### No source count

The agent can't tell whether a response is backed by 0, 1, or 10 sources without parsing the output. This affects how much the agent should trust the response.

## Proposed Solution

Add a `--meta` flag that appends a structured YAML block to the end of stdout, separated by a `---` delimiter:

```bash
./websearch --meta -m reason "query here"
```

Output:

```
[normal response content here]

---
provider: perplexity
mode: reason
mode_adjusted: false
max_tokens: 2048
response_tokens: ~1850
response_truncated: false
sources_count: 5
cache_hit: false
```

### Schema

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Actual provider used (after fallback) |
| `mode` | string | Actual mode used (after adjustment) |
| `mode_adjusted` | bool | Whether mode was changed from requested |
| `max_tokens` | int | Token limit that was applied |
| `response_truncated` | bool | Whether response hit the token limit |
| `sources_count` | int | Number of sources/citations returned |
| `cache_hit` | bool | Whether the result came from cache |

### Truncation detection

The `response_truncated` field can be estimated by checking if the Perplexity response's `finish_reason` is `length` (token limit hit) vs `stop` (natural completion). This is the most valuable signal — it tells the agent to retry with `--max-tokens` increased.

### Alternative: stderr JSON

Instead of appending to stdout, emit metadata as a single-line JSON to stderr:

```
{"provider":"perplexity","mode":"reason","truncated":false,"sources":5}
```

This avoids polluting stdout but requires the agent to capture stderr.

## Impact

- Enables agents to make informed retry decisions (truncated? increase tokens. mode adjusted? maybe use a different provider)
- The `--meta` flag is opt-in — no change to default behavior
- Particularly valuable for Change 001 (truncation) — with metadata, agents can detect truncation programmatically rather than guessing
- Low risk since it's additive and behind a flag

## Stress Test Context

Across 50 queries:
- 2 had response truncation (invisible to agent)
- 5 had mode auto-adjustment (stderr only)
- 3 had provider failures (errors, but no structured signal about what to try next)
- 7 had deflection responses (no structured signal)

With `--meta`, agents could handle all of these cases programmatically.

## Open Questions

I think there should be metadata no matter what if Markdown add a frontmatter header with metadata - and in json just in the json response - we could add a --no-metadata flag instead of a -metadata flag optin

## Plan

### Step 1: Design the metadata schema

Based on your preference for metadata-by-default, define what goes in the metadata:

**Markdown output** — YAML frontmatter at the top:

```markdown
---
provider: perplexity
mode: reason
mode_adjusted: false
cached: false
truncated: false
sources_count: 5
---

[response content here]
```

**JSON output** — fields added to the existing `jsonOutput` struct:

```json
{
  "provider": "perplexity",
  "mode": "reason",
  "mode_adjusted": false,
  "cached": false,
  "truncated": false,
  "sources_count": 5,
  "content": "...",
  "sources": [...]
}
```

Note: JSON already includes `provider`, `mode`, and `cached` in `formatter.go:41-46`. We just need to add the new fields (`mode_adjusted`, `truncated`, `sources_count`).

### Step 2: Extend the Result struct

Add metadata fields to `provider.Result`:

```go
// provider.go
type Result struct {
    Content      string
    Provider     string
    Mode         string
    Cached       bool
    Sources      []Source
    FinishReason string  // NEW: "stop" or "length" from API
    ModeAdjusted bool    // NEW: was mode changed from what was requested
}
```

`FinishReason` comes from the Perplexity response (already parsed but not stored — see `perplexity.go:85`). `ModeAdjusted` gets set in `root.go` when mode auto-adjustment fires.

### Step 3: Update FormatMarkdown to include frontmatter

```go
func FormatMarkdown(result *provider.Result, includeSources bool, includeMetadata bool) string {
    var sb strings.Builder
    if includeMetadata {
        sb.WriteString("---\n")
        sb.WriteString(fmt.Sprintf("provider: %s\n", result.Provider))
        sb.WriteString(fmt.Sprintf("mode: %s\n", result.Mode))
        sb.WriteString(fmt.Sprintf("truncated: %v\n", result.FinishReason == "length"))
        sb.WriteString(fmt.Sprintf("sources_count: %d\n", len(result.Sources)))
        sb.WriteString("---\n\n")
    }
    // ... existing content formatting
}
```

### Step 4: Add `--no-metadata` flag

In `root.go`, add the flag (metadata on by default):

```go
var flagNoMetadata bool
rootCmd.Flags().BoolVar(&flagNoMetadata, "no-metadata", false, "Omit response metadata")
```

Pass `!flagNoMetadata` as the `includeMetadata` parameter to the formatters.

### Step 5: Update self-primer to document the metadata

The self-primer (shown when running `./websearch` with no args) should mention that metadata is included by default and can be suppressed with `--no-metadata`.

### Step 6: Validate

Run a few test queries and verify:
- Markdown output has valid YAML frontmatter that doesn't break downstream markdown parsers
- JSON output has the extra fields
- `--no-metadata` produces the old output format
- `truncated: true` appears when `finish_reason` is `"length"` (ties back to Change 001)

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


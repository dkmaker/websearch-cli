# Change 010: Research Mode Response Summarization

**Status:** Proposed
**Priority:** Low
**Affected component:** `internal/cli/root.go`, `internal/output/formatter.go`

---

## Problem

Research mode (`-m research`, `sonar-deep-research`) produces extremely long responses — 3,500 to 4,700+ words with 40-54 sources. While thorough, these responses often exceed what downstream AI agents can efficiently consume:

### Stress Test Evidence

| Query | Words | Sources | Content |
|-------|-------|---------|---------|
| #15 | ~3,520 | 43 | Multi-tenant SaaS data isolation (RLS vs schema-per-tenant) — covers 5 compliance frameworks, 5 isolation tiers, geographic sharding |
| #21 | ~4,718 | 54 | CQRS with event sourcing — full PostgreSQL schemas, LISTEN/NOTIFY triggers, consistent hashing, monitoring queries |

Both scored 5.0/5.0 for completeness, but the sheer volume creates practical problems:

1. **Context window pressure**: An AI agent with 8k-16k context that receives a 4,700-word research response has little room left for its own reasoning
2. **Signal-to-noise ratio**: The most actionable content (architecture decisions, code patterns) is buried in comprehensive analysis
3. **Token cost**: The agent pays to process thousands of tokens of background context it may not need
4. **Diminishing returns**: After ~1,500 words, most research responses shift from core answers to edge cases, alternative approaches, and historical context

### Current Behavior

Research responses are returned in full with no length management. The `max_tokens` flag controls the *API request* token limit (Change 001 bumped this to 16k), but there's no post-processing to manage output length for the consumer.

## Proposed Solution

Add a `--max-length` flag that truncates or summarizes research output to a target word count, with clean break handling:

### Option A: Smart Truncation (simpler)

```go
rootCmd.Flags().IntVar(&flagMaxLength, "max-length", 0, "Max response length in words (0 = unlimited)")
```

When `--max-length` is set:

```go
func truncateToLength(content string, maxWords int) (string, bool) {
    words := strings.Fields(content)
    if len(words) <= maxWords {
        return content, false
    }

    // Find the last complete paragraph or section boundary before the limit
    truncated := strings.Join(words[:maxWords], " ")

    // Try to break at a paragraph boundary (double newline)
    lastPara := strings.LastIndex(truncated, "\n\n")
    if lastPara > len(truncated)/2 { // don't cut more than half
        truncated = truncated[:lastPara]
    }

    truncated += "\n\n---\n*[Response truncated at " + strconv.Itoa(maxWords) + " words. Use --max-length 0 for full response.]*"
    return truncated, true
}
```

Set `result.Truncated = true` in metadata when truncation occurs (reusing the existing `truncated` field from Result metadata).

### Option B: Two-Pass Summarization (more capable, more complex)

For a higher-quality option, pipe the full research response back through a fast Perplexity model (`sonar` / ask mode) with a summarization prompt:

```go
func summarizeResponse(ctx context.Context, content string, maxWords int, prov *Perplexity) (string, error) {
    prompt := fmt.Sprintf("Summarize the following in under %d words, preserving key decisions, code examples, and actionable recommendations:\n\n%s", maxWords, content)
    opts := SearchOptions{Mode: "ask", MaxTokens: maxWords * 2}
    result, err := prov.Search(ctx, prompt, opts)
    return result.Content, err
}
```

This preserves the most important content but costs an additional API call.

### Recommendation

Start with **Option A** (smart truncation). It's zero-cost, deterministic, and solves the primary problem. Option B can be added later as `--summarize` if demand exists.

## Impact

- Research mode responses become practical for context-constrained AI agents
- The `--max-length` flag is opt-in — default behavior is unchanged (unlimited)
- Metadata `truncated: true` signals to the agent that content was cut
- Works for all providers, not just research mode (useful for any verbose response)
- Smart paragraph-boundary truncation avoids mid-sentence cuts

## Usage Examples

```bash
# Full research response (default, unchanged)
./websearch -m research "CQRS event sourcing architecture"

# Truncated to ~1500 words for constrained context
./websearch -m research --max-length 1500 "CQRS event sourcing architecture"

# Combined with JSON output
./websearch -m research --max-length 1000 --json "multi-tenant SaaS isolation"
```

## Open Questions

- Should `--max-length` have a default value for research mode specifically? E.g., auto-set to 2000 words for research mode unless the user passes `--max-length 0` explicitly.
- Should truncation happen before or after source/citation stripping? (Currently sources are stripped by default for token efficiency — truncating after stripping is more accurate.)
- Is word count the right unit, or should this be token count? Word count is more intuitive for users but less precise for LLM context budgets.

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


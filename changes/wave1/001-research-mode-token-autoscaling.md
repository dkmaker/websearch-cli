# Change 001: Research Mode Token Auto-Scaling

**Status:** Proposed
**Priority:** High
**Affected component:** `internal/cli/root.go`, `internal/profile/profiles/*.yaml`

---

## Problem

Research mode (`-m research`) uses the Perplexity `sonar-deep-research` model, which produces long-form responses (3,000-10,000+ tokens). The default `max_tokens: 2048` set in all builtin profiles is far too low, causing responses to be **truncated mid-sentence** before reaching the most critical content.

Both research-mode queries in the stress test were cut off:

- **Query #15** (architecture, score 3.5/5.0) — asked about multi-tenant SaaS data isolation with PostgreSQL RLS vs schema-per-tenant. The response covered RLS in depth but was **truncated mid-word** at "can introduce measurable query" before ever reaching schema-per-tenant, hybrid strategies, or data residency solutions — the core of the question.
  - File: `result_output/result_15_architecture.json`
  - Evaluator: *"response is cut off mid-sentence before covering schema-per-tenant"*

- **Query #21** (architecture, score 3.8/5.0) — asked about CQRS with event sourcing under 100ms consistency SLA at 10k events/sec. Response covered event store design but was **truncated before the projection infrastructure section** — the part that actually answers the question.
  - File: `result_output/result_21_architecture.json`
  - Evaluator: *"truncated mid-sentence before addressing the core projection architecture"*

Both queries used ~1,200 words and hit the token ceiling. The agent paid for an expensive deep-research API call and received half an answer.

## Proposed Solution

Auto-scale `max_tokens` when research mode is selected and the user hasn't explicitly set `--max-tokens`:

```go
// internal/cli/root.go — after mode resolution, before provider creation
if mode == "research" && flagMaxTokens == 0 && prof.MaxTokens <= 2048 {
    prof.MaxTokens = 8192
}
```

Alternatively, the `general.yaml` profile could define a separate `research_max_tokens` field, but the simpler approach is a one-line auto-scale in the CLI orchestration.

## Impact

- Prevents truncated research responses
- No change for ask/search/reason modes (they stay at 2048)
- Both test queries would likely have scored 4.0+ with complete responses
- Cost consideration: research mode is already the most expensive mode; truncating the response wastes the API cost without delivering the value

## Open Questions

Is the problem in our logic or that the Perplexity response is limiting?

If we get all back from Perplexity:
- Could we make pagination?
- Could we set max tokens at Perplexity to be aligned with our research pattern? I still think we should bump it - research is okay to go 10k - 20k tokens!

## Plan

### Step 1: Determine where the truncation happens (investigation)

The Perplexity response includes a `finish_reason` field per choice (`perplexity.go:85`). We currently ignore it. First, we need to confirm:

- Run one of the truncated queries manually with `--no-cache` and capture the full JSON response via a debug flag or curl
- Check if `finish_reason` is `"length"` (our `max_tokens` cut it) or `"stop"` (Perplexity stopped naturally)
- Check the Perplexity docs for `sonar-deep-research` to see if it has its own output token cap independent of `max_tokens`

```bash
# Quick curl test to see raw response and finish_reason:
curl -s https://api.perplexity.ai/chat/completions \
  -H "Authorization: Bearer $PERPLEXITY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"sonar-deep-research","messages":[{"role":"user","content":"What is PostgreSQL row-level security?"}],"max_tokens":16000}' \
  | jq '.choices[0].finish_reason, .usage'
```

This tells us if Perplexity respects our higher `max_tokens` or has its own ceiling.

### Step 2: Bump research mode default to 16k

Based on your preference for 10k-20k, set research mode auto-scale to 16384 in `root.go`. This is conservative enough to avoid API errors but large enough for deep-research output:

```go
if mode == "research" && flagMaxTokens == 0 && prof.MaxTokens <= 2048 {
    prof.MaxTokens = 16384
}
```

### Step 3: Surface `finish_reason` in the Result struct

Add `FinishReason string` to `provider.Result`. Populate it from `pResp.Choices[0].FinishReason` in `perplexity.go`. This enables Change 005 (metadata) to report `response_truncated: true` when `finish_reason == "length"`.

### Step 4: Re-run queries #15 and #21 to validate

Re-execute both truncated queries with the new token limit and confirm complete responses.

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


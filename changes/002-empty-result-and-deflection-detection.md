# Change 002: Empty Result and Deflection Detection

**Status:** Proposed
**Priority:** High
**Affected component:** `internal/cli/root.go`, `internal/output/formatter.go`

---

## Problem

The CLI currently treats all non-error responses identically — exit code 0 with content on stdout. This means an AI agent cannot distinguish between:

1. **A useful answer** (the normal case)
2. **An empty response** (zero results, zero content)
3. **A deflection** (the model admits it can't answer and suggests searching elsewhere)

All three exit with code 0 and produce no stderr warning.

### Empty Result Evidence

- **Query #30** (best-practices, score 0.0/5.0) — GitHub `repos` mode returned exit code 0 with completely empty stdout. No stderr, no warning. The agent received nothing.
  - File: `result_output/result_30_best-practices.json`
  - `success: true, had_errors: false, response_length_words: 0`
  - Evaluator: *"GitHub API returned empty results without GITHUB_TOKEN"*

### Deflection Evidence

7 queries received deflection responses where Perplexity explicitly stated it couldn't find relevant information:

| Query | Score | Opening line of response |
|-------|-------|--------------------------|
| #50 | 1.8 | "The search results provided do not contain information about..." |
| #42 | 1.8 | "The search results provided do not contain specific information about..." |
| #37 | 1.8 | "The search results provided don't directly address this specific issue..." |
| #45 | 2.3 | Admits "no specific guidance found", falls back to generic advice |
| #44 | 2.3 | Provides generic packaging advice, not the specific stack asked about |
| #7  | 2.3 | "Explicitly admits the retrieved search results lack streaming SSR content" |
| #36 | 2.0 | Acknowledges insufficient search context |

These all exited with code 0. An agent has no signal that the response is a deflection unless it parses the natural language output and recognizes the pattern.

## Proposed Solution

### Part A: Empty Result Warning

After receiving a provider response, check if the content is empty:

```go
if strings.TrimSpace(result.Content) == "" {
    fmt.Fprintln(os.Stderr, "warning: search returned no results")
    os.Exit(2) // distinct from error (1) and success (0)
}
```

### Part B: Deflection Detection (optional, more complex)

Check the first 100 characters of the response for known deflection patterns:

```go
deflectionPrefixes := []string{
    "The search results provided do not contain",
    "The search results do not contain",
    "I don't have enough information",
    "The available search results don't",
}
```

If matched, emit a stderr warning:

```
warning: response may not address the query (search context insufficient)
```

This does NOT change the exit code — the response is still valid content. It gives the agent a hint to consider retrying with different terms.

## Impact

- Part A: Eliminates silent empty responses (query #30)
- Part B: Gives agents a signal for 7 deflection cases, enabling retry logic
- No change to behavior for normal successful responses
- Exit code 2 for empty results lets agents script around failures

## Open Questions

Could we use Perplexity structured response? And make perplexity output as json where there is a attribute that says if it was found and then still have the body output but just as a object in JSON?

Maybe Perplexity could do the Defelction cases - if possible? We could modify the system prompt so there is more handling in the system prompt and the actual search phrase we are taking from the user to safeguard the request/response?

I still think it should make a Exit 2 etc. not Exit 1 - because its valid its not found but 2 is okay

## Plan

### Step 1: Research Perplexity structured output (investigation)

Check if Perplexity's API supports structured/JSON output mode. Specifically:

- Check Perplexity docs for a `response_format` parameter (like OpenAI's `{ "type": "json_object" }`)
- If supported, we could ask Perplexity to return JSON with a `found: boolean` field and `content: string` field
- Test with curl to see if `sonar-reasoning-pro` respects structured output

```bash
# Test if Perplexity supports response_format:
curl -s https://api.perplexity.ai/chat/completions \
  -H "Authorization: Bearer $PERPLEXITY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"sonar-reasoning-pro","messages":[{"role":"system","content":"Respond in JSON with keys: found (boolean), confidence (1-5), content (string)"},{"role":"user","content":"What is the framwork called zxyqwerty?"}],"max_tokens":512}' \
  | jq '.choices[0].message.content'
```

### Step 2: Harden the system prompt to prevent deflection

Currently the system prompt comes from the profile YAML. We can add instructions that tell Perplexity to:
- Always attempt an answer even when search results are thin
- Clearly state confidence level rather than deflecting entirely
- Structure the response with the answer first, caveats second

Example system prompt addition:
```
Always provide your best answer based on available information.
If search results are insufficient, state what you do know and clearly
note gaps, rather than refusing to answer. Never recommend the user
search elsewhere.
```

This addresses the 7 deflection cases (#7, #36, #37, #42, #44, #45, #50) at the source.

### Step 3: Implement empty-result exit code 2

In `root.go`, after formatting the result, check for empty content:

```go
if strings.TrimSpace(result.Content) == "" {
    fmt.Fprintln(os.Stderr, "warning: search returned no results")
    os.Exit(2)
}
```

Exit 2 = valid but empty (as you specified — not an error, just "not found").

### Step 4: Add optional deflection detection in the CLI

Even with a better system prompt, add a lightweight stderr warning when the response starts with known deflection phrases. This is a safety net — the system prompt change should reduce deflections, but the CLI-side detection catches any that slip through.

### Step 5: Test the system prompt changes

Re-run 2-3 of the deflection queries (#50, #42, #37) with the improved system prompt to measure if deflection rate drops.

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


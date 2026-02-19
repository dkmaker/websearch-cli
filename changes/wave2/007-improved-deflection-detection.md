# Change 007: Improved Deflection Detection (Structured Output + Full-Content Scan)

**Status:** Proposed
**Priority:** High
**Affected component:** `internal/provider/perplexity.go`, `internal/cli/root.go`

---

## Problem

The current `detectDeflection()` function only checks if the response **starts with** a known deflection prefix (`root.go:430-438`). In the 50-query stress test, ~8 responses contained mid-response deflections that were not caught:

### Missed Deflections (no stderr warning, exit code 0)

| Query | Response Pattern | What Happened |
|-------|-----------------|---------------|
| #50 | "The search results provided focus on basic TypeScript and Express setup... and **do not contain** information about the advanced setup" | Partial answer followed by admission of failure. Model fabricated the rest from training data. |
| #42 | Perplexity explicitly says it lacks info on uv/poetry workspaces | Deflection buried after initial framing paragraph |
| #43 | "The search results do not contain detailed pytest-asyncio fixture patterns" | Mid-response gap acknowledgment |
| #45 | "The provided search results do not contain specific guidance on bootstrapping..." | Starts with framing, deflection is the second paragraph |
| #26 | "The search results did NOT cover specific native .so merging conflict scenario" | Honest admission mid-response, then generic advice |
| #37 | "did NOT find specific Go-Python compatibility issues" | Partial deflection — offers troubleshooting framework instead |
| #4  | "The search results don't directly address SIGABRT crashes when upgrading from Node.js v18 to v22" | Deflection after initial context |
| #5  | Explicitly notes it cannot cover "SQLAlchemy async session instrumentation" or "Celery worker distributed tracing" | Lists what's missing mid-response |

In all cases, the calling AI agent received exit code 0 with no stderr warning, and had no signal that the answer was partially or fully fabricated from model knowledge rather than search results.

## Proposed Solution — Three Layers

### Layer 1: Structured JSON Output from Perplexity (primary — catches deflections at the source)

Perplexity's Sonar API supports structured output via `response_format` with JSON Schema (same as OpenAI). Instead of guessing whether a natural-language response is a deflection, we can **ask the model to tell us explicitly** by wrapping every response in a structured envelope:

```json
{
  "response_format": {
    "type": "json_schema",
    "json_schema": {
      "schema": {
        "type": "object",
        "properties": {
          "answered": { "type": "boolean", "description": "true if the search results contained relevant information to answer the query" },
          "confidence": { "type": "integer", "minimum": 1, "maximum": 5, "description": "1=no relevant results, 3=partial answer, 5=fully addressed" },
          "content": { "type": "string", "description": "The answer in markdown format" }
        },
        "required": ["answered", "confidence", "content"],
        "additionalProperties": false
      }
    }
  }
}
```

#### How It Works

1. Add `response_format` to `perplexityRequest` struct in `perplexity.go`:

```go
type perplexityRequest struct {
    Model               string              `json:"model"`
    Messages            []perplexityMessage `json:"messages"`
    MaxTokens           int                 `json:"max_tokens,omitempty"`
    SearchDomainFilter  []string            `json:"search_domain_filter,omitempty"`
    SearchRecencyFilter string              `json:"search_recency_filter,omitempty"`
    SearchContextSize   string              `json:"search_context_size,omitempty"`
    ResponseFormat      *responseFormat     `json:"response_format,omitempty"`
    Stream              bool                `json:"stream"`
}

type responseFormat struct {
    Type       string      `json:"type"`
    JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
    Schema json.RawMessage `json:"schema"`
}
```

2. When structured output is enabled, parse the JSON envelope and extract `content`, `answered`, `confidence`:

```go
// After receiving the response
type structuredResponse struct {
    Answered   bool   `json:"answered"`
    Confidence int    `json:"confidence"`
    Content    string `json:"content"`
}

var sr structuredResponse
if err := json.Unmarshal([]byte(pResp.Choices[0].Message.Content), &sr); err != nil {
    // Fallback — treat raw content as the answer
    result.Content = pResp.Choices[0].Message.Content
} else {
    result.Content = sr.Content
    result.Confidence = sr.Confidence
    result.Answered = sr.Answered
}
```

3. Add fields to `Result` struct in `provider.go`:

```go
type Result struct {
    Content      string   `json:"content"`
    Sources      []Source `json:"sources,omitempty"`
    Provider     string   `json:"provider"`
    Mode         string   `json:"mode"`
    Cached       bool     `json:"cached"`
    FinishReason string   `json:"finish_reason,omitempty"`
    ModeAdjusted bool     `json:"mode_adjusted"`
    Answered     bool     `json:"answered"`       // did search results contain an answer?
    Confidence   int      `json:"confidence"`      // 1-5 confidence score
}
```

4. In `root.go`, use the structured fields for warnings:

```go
if !result.Answered || result.Confidence <= 2 {
    fmt.Fprintln(os.Stderr, "warning: response may not address the query (search context insufficient)")
}
```

#### System Prompt Addition

Augment every Perplexity system prompt with instructions that reinforce the structured output contract:

```
You MUST respond with valid JSON matching the provided schema.
Set "answered" to false if the search results do not contain information
relevant to the user's question. Set "confidence" to 1 if search results
are completely irrelevant, 3 if partially relevant, 5 if fully relevant.
The "content" field should contain your best answer in markdown format.
If search results are insufficient, still provide what you can in "content"
but set "answered" accordingly — do NOT refuse to answer or recommend
searching elsewhere.
```

This is appended to the existing profile system prompt, not replacing it.

#### Caveats

- Perplexity notes the first request with a new schema may take 10-30 seconds for preparation; subsequent requests are faster (schema is cached server-side).
- Structured output may not be supported on `sonar-deep-research` (research mode). Need to test. If not supported, fall back to Layer 2 for research mode.
- The `content` field is a JSON string, so the markdown inside it will have escaped newlines (`\n`). The Go JSON unmarshaler handles this correctly.

### Layer 2: System Prompt Hardening (reinforcement — reduces deflections at generation time)

Even without structured output, we can reduce deflections by strengthening the system prompt. The current prompts already say "always provide your best answer" but the model still deflects on ~16% of queries. Add more specific instructions:

```
When search results are insufficient:
- Still answer the question using the information available
- Clearly separate what comes from search results vs your general knowledge
- Use "Based on search results: ..." and "Additionally: ..." sections
- Never say "I recommend consulting documentation" or "search elsewhere"
- Never start your response by stating what the search results don't cover
```

This goes into each profile's `system_prompt` field in the YAML files.

### Layer 3: Full-Content Scan (safety net — catches anything that slips through Layers 1 and 2)

Keep the existing `detectDeflection()` as a fallback, but upgrade it:

**A. Scan full content, not just prefix:**

```go
func detectDeflection(content string) string {
    lower := strings.ToLower(strings.TrimSpace(content))
    for _, phrase := range deflectionPhrases {
        if strings.Contains(lower, strings.ToLower(phrase)) {
            return "warning: response may not address the query (search context insufficient)"
        }
    }
    return ""
}
```

**B. Expand the phrase list** with patterns observed in the stress test:

```go
var deflectionPhrases = []string{
    // Existing
    "the search results provided do not contain",
    "the search results do not contain",
    "the available search results don't",
    "i don't have enough information",
    // New — observed in stress test
    "do not contain information about",
    "do not contain specific guidance",
    "not covered in your search results",
    "based on my knowledge beyond these search results",
    "search results don't directly address",
    "search results did not cover",
    "search results do not provide",
    "the search results lack",
}
```

**C. Distinguish severity levels** based on position:

```go
func detectDeflection(content string) string {
    lower := strings.ToLower(strings.TrimSpace(content))
    for _, phrase := range deflectionPhrases {
        idx := strings.Index(lower, strings.ToLower(phrase))
        if idx >= 0 {
            if idx < 200 {
                return "warning: response may not address the query (search context insufficient)"
            }
            return "warning: response partially addresses the query (some search context gaps noted)"
        }
    }
    return ""
}
```

## Recommended Implementation Order

1. **Layer 3 first** — quick win, no API changes, catches most deflections today
2. **Layer 2 second** — update system prompts in profile YAMLs, reduces deflection rate at source
3. **Layer 1 last** — structured output requires Perplexity API changes, needs testing per model, highest impact but most work

Layer 3 is the safety net regardless. Layers 1 and 2 reduce how often it needs to fire.

## Impact

- **Layer 1**: Eliminates guesswork — the model explicitly reports whether it found relevant results. Confidence score (1-5) gives the agent granular signal. Enables downstream logic like "retry with different terms if confidence < 3".
- **Layer 2**: Reduces deflection rate at generation time (from ~16% to estimated ~5%)
- **Layer 3**: Catches remaining deflections that slip through with stderr warnings
- **No change to exit codes** — deflected responses still exit 0 (they have content)
- Risk: structured output adds ~2-5% latency and the JSON envelope consumes some tokens from the response budget. The `content` field inside the envelope has fewer available tokens than a raw response.

## Open Questions

- Does structured output work with all Sonar models? Need to test `sonar` (ask), `sonar-pro` (search), `sonar-reasoning-pro` (reason), and `sonar-deep-research` (research). The reasoning model may strip the schema.
- Should `confidence` be exposed in the markdown metadata frontmatter? E.g., `confidence: 3` in the `---` block.
- Should low-confidence responses (1-2) use exit code 2 (like empty results) or stay at exit 0 with just a warning?
- The phrase `"does not cover"` was removed from Layer 3's expanded list because it has high false-positive risk in legitimate technical content. Should it be included with additional context matching (e.g., only trigger if preceded by "search results")?

## Relationship to Change 002

Change 002 introduced the current deflection detection with prefix-only matching and system prompt improvements. This change builds on that foundation:
- Layer 1 (structured output) is entirely new
- Layer 2 extends 002's system prompt work
- Layer 3 extends 002's deflection detection from prefix-only to full-content scan

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


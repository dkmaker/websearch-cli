# Change 004: Strip HTML Tags from Brave Output

**Status:** Proposed
**Priority:** Low
**Affected component:** `internal/provider/brave.go` or `internal/output/formatter.go`

---

## Problem

Brave search results contain raw HTML tags (`<strong>`, `</strong>`, etc.) that leak into the markdown output. This adds noise for AI agents parsing the results, breaks clean markdown rendering, and wastes tokens on non-content characters.

### Evidence

**Query #11** (documentation, score 2.8) — Brave output for Terraform AWS AssumeRole:

```
**Using AWS AssumeRole with the AWS Terraform Provider – HashiCorp Help Center**
In your Terraform configuration, <strong>configure the AWS provider to use the
credentials for Account A and specify the assume_role block to connect to
Account B</strong>. provider "aws" { ## Credentials for the IAM user...
```

The `<strong>` tags appear throughout the raw output file `raw_11_documentation.md`. This pattern is consistent across all 5 Brave queries:

| Query | File | HTML tags present |
|-------|------|-------------------|
| #11 | `raw_11_documentation.md` | `<strong>`, `</strong>` |
| #23 | `raw_23_documentation.md` | `<strong>`, `</strong>` |
| #29 | `raw_29_documentation.md` | `<strong>`, `</strong>` |
| #34 | `raw_34_debugging.md` | `<strong>`, `</strong>` |
| #48 | `raw_48_bootstrapping.md` | `<strong>`, `</strong>` |

The Brave Search API returns highlighted snippets using HTML tags. Our formatter passes these through unmodified.

## Proposed Solution

Strip HTML tags from Brave snippet content before formatting. In the Brave provider or the output formatter:

```go
import "regexp"

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTMLTags(s string) string {
    return htmlTagRe.ReplaceAllString(s, "")
}
```

Apply to the `Description` field of each Brave web result before including it in the formatted output. This converts:

```
<strong>configure the AWS provider</strong>
```

to:

```
configure the AWS provider
```

### Alternative: Convert to Markdown

Instead of stripping, convert `<strong>` to `**` for proper markdown bold. This preserves the emphasis Brave intended:

```go
s = strings.ReplaceAll(s, "<strong>", "**")
s = strings.ReplaceAll(s, "</strong>", "**")
// Strip any remaining HTML tags
s = htmlTagRe.ReplaceAllString(s, "")
```

## Impact

- Cleaner output for all Brave queries
- Saves a few tokens per response (minor but adds up)
- No behavioral change — same content, just properly formatted
- Low risk, straightforward implementation

## Brave Provider Stress Test Summary

All 5 Brave queries scored between 2.3 and 3.8 (average 2.88). The HTML noise isn't the primary quality limiter (that's the snippet-only format), but cleaning it up improves the agent experience for the results that are returned.

## Open Questions

Cant Branve output as JSON or simimlar?

We should strip all HTML if possible and only keep Markdown but we should also be able to output json, can we do that with Brave?

## Plan

### Step 1: Check what Brave actually returns with different `text_format` values (investigation)

We already send `text_format=markdown` (`brave.go:63`). Brave's API docs say it supports `raw` (HTML) and `markdown`. But we're still getting `<strong>` tags — this might mean Brave's markdown mode is incomplete, or we're not handling it correctly.

```bash
# Test markdown mode (current behavior):
curl -s "https://api.search.brave.com/res/v1/web/search?q=vite+typescript+path+aliases&text_format=markdown&count=2" \
  -H "X-Subscription-Token: $BRAVE_API_KEY" \
  -H "Accept: application/json" | jq '.web.results[0].description'

# Test raw mode to compare:
curl -s "https://api.search.brave.com/res/v1/web/search?q=vite+typescript+path+aliases&text_format=raw&count=2" \
  -H "X-Subscription-Token: $BRAVE_API_KEY" \
  -H "Accept: application/json" | jq '.web.results[0].description'

# Check what other fields are available in the response:
curl -s "https://api.search.brave.com/res/v1/web/search?q=vite+typescript+path+aliases&count=1" \
  -H "X-Subscription-Token: $BRAVE_API_KEY" \
  -H "Accept: application/json" | jq '.web.results[0] | keys'
```

### Step 2: Strip HTML tags from description field

Regardless of what Brave returns, add a safety strip in `brave.go`. Currently we do `html.UnescapeString()` (`brave.go:114-115`) but that only handles entities (`&amp;` etc.), not tags.

Add after the unescape:

```go
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// In the Search method, after UnescapeString:
title := stripHTMLTags(html.UnescapeString(r.Title))
desc := stripHTMLTags(html.UnescapeString(r.Description))
```

### Step 3: Consider converting `<strong>` to markdown bold

Instead of just stripping, convert the emphasis:

```go
func cleanBraveHTML(s string) string {
    s = html.UnescapeString(s)
    s = strings.ReplaceAll(s, "<strong>", "**")
    s = strings.ReplaceAll(s, "</strong>", "**")
    // Strip any remaining HTML tags
    s = htmlTagRe.ReplaceAllString(s, "")
    return s
}
```

This preserves Brave's intent (highlighting matching terms) while staying in valid markdown.

### Step 4: JSON output already works

Looking at the code, `--format json` already works via `FormatJSON()` in `formatter.go`. Brave results will render as JSON with `content`, `provider`, `mode`, and `sources` fields. The HTML tags would still leak into the `content` JSON string though, so the strip in step 2 fixes both markdown and JSON output formats.

### Step 5: Validate with existing Brave raw outputs

Compare a few raw files before/after to confirm the cleanup works as expected. No need to re-run queries — we can validate the regex against the existing `raw_11_documentation.md` content.

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


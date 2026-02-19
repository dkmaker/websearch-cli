# AI Agent Optimization Design

## Problem

The websearch CLI is functional but not optimized for AI agent consumption:
1. Running with no arguments produces a bare error — agents can't discover capabilities
2. Brave provider output contains raw HTML entities (`&quot;`, `&amp;`)
3. No version flag
4. No self-describing behavior for agent priming

## Design Decisions

- **No-args = self-primer** (not an error). Agents discover capabilities by running the tool.
- **--show-examples and --show-profiles** are additive flags that expand the self-primer output.
- **HTML entity decoding** happens in the provider layer (brave.go), not the formatter.
- **Version** via Go ldflags, exposed through Cobra's built-in --version and in self-primer output.

## Self-Primer Output (no args, ~80 tokens)

```
websearch v1.0.0 — Web search CLI for AI agents

Providers: perplexity (ready), brave (no key)
Modes [perplexity]: ask*, search, reason, research
Modes [brave]: web*
Profiles: general*, nodejs, python
Output: markdown (default), json (--json)
Cache: 60m TTL (--no-cache to bypass)

Usage: websearch [flags] "query"
Key flags: -p profile, -m mode, --provider, --json, --include-sources, --no-cache
More: --show-examples, --show-profiles, --help
```

`*` marks defaults. Provider key status shown dynamically.

## --show-examples (appended to base primer)

```
Examples:
  websearch "what is Go generics"                          # general ask
  websearch -m research "AI trends 2026"                   # deep research
  websearch --provider brave "local restaurants"            # brave web search
  websearch -p python "fastapi middleware"                  # python profile
  websearch --json --include-sources "kubernetes basics"    # JSON with sources
```

## --show-profiles (appended to base primer)

```
Profiles:
  general   Perplexity/ask   General-purpose web search
  nodejs    Perplexity/ask   Node.js development (filters: nodejs.org, mdn, npmjs.com)
  python    Perplexity/ask   Python development (filters: docs.python.org, pypi.org, realpython.com)

Custom profiles: ~/.config/websearch/profiles/<name>.yaml
```

Both flags can be combined.

## Brave HTML Entity Fix

- Use `html.UnescapeString()` in `brave.go` on both `Title` and `Description` before building Result
- Decodes `&quot;` → `"`, `&amp;` → `&`, `&#39;` → `'`, etc.
- Single fix point; all formatters get clean text

## Version Support

- `version` variable in `cmd/websearch/main.go` with default `"dev"`
- Set via ldflags: `go build -ldflags "-X main.version=1.0.0"`
- Exposed via `rootCmd.Version` (Cobra --version flag)
- Displayed in self-primer header line

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/websearch/main.go` | Add version variable, pass to cli package |
| `internal/cli/root.go` | Custom no-args handling, self-primer output, --show-examples, --show-profiles flags, version integration |
| `internal/provider/brave.go` | `html.UnescapeString()` on Title and Description |
| `internal/profile/profile.go` | Add function to return profile details for --show-profiles |

## Tests to Add/Update

| File | Tests |
|------|-------|
| `internal/cli/root_test.go` | Self-primer output, --show-examples, --show-profiles |
| `internal/provider/brave_test.go` | HTML entity decoding in results |

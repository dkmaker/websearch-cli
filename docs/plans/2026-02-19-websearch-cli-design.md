# websearch CLI Design

## Overview

A Go CLI tool (`websearch`) designed primarily for AI agents to perform web searches via Perplexity and Brave APIs. Uses a profile system to abstract away provider selection, search modes, and formatting — so agents just specify a profile and query.

## Usage

```
websearch "what is the best Go HTTP router"
websearch --profile nodejs "how to use streams in Node 22"
websearch --mode research --json "compare React vs Svelte 2025"
websearch --profile python --mode reason "async patterns"
websearch --include-sources "latest Go releases"
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--profile` | `-p` | Profile name | `general` |
| `--mode` | `-m` | Search mode override | profile default |
| `--json` | | JSON output instead of markdown | `false` |
| `--provider` | | Explicit provider override | profile decides |
| `--max-results` | `-n` | Max result count | profile default |
| `--max-tokens` | | Max response tokens | profile default |
| `--include-sources` | | Include citations/sources in output | `false` |
| `--no-cache` | | Bypass response cache | `false` |
| `--list-profiles` | | List available profiles | |

Query is the positional argument.

## Architecture

### Project Structure

```
cmd/
  websearch/
    main.go              # Entry point
internal/
  cli/
    root.go              # Cobra root command, flag parsing
  provider/
    provider.go          # Provider interface
    perplexity.go        # Perplexity API client (ask, search, reason, research)
    brave.go             # Brave Search API client (web, local)
  profile/
    profile.go           # Profile struct, loading, merging
    embed.go             # Embedded built-in profiles via embed.FS
    profiles/            # Built-in YAML profile files
      general.yaml
      nodejs.yaml
      python.yaml
  cache/
    cache.go             # File-based response cache with TTL
  output/
    formatter.go         # Markdown & JSON formatters
go.mod
go.sum
```

### Provider Interface

```go
type Provider interface {
    Search(ctx context.Context, query string, opts SearchOptions) (*Result, error)
    Name() string
    SupportedModes() []string
}
```

### Profile Definition (YAML)

```yaml
name: nodejs
description: "Node.js development searches"
provider: perplexity
mode: ask
system_prompt: |
  Focus on Node.js ecosystem. Include code examples using
  modern ESM syntax. Prefer official docs and well-maintained
  packages. Include version compatibility notes.
domain_filter:
  - nodejs.org
  - developer.mozilla.org
  - npmjs.com
max_results: 10
max_tokens: 2048
output_format: markdown
```

### Profile Resolution

1. Load built-in profile (embedded in binary via `embed.FS`)
2. Check `~/.config/websearch/profiles/` for user override
3. Merge user profile on top of built-in (user fields override, unset fields inherit)
4. Apply CLI flag overrides on top of merged profile

### Provider Selection

1. Profile specifies preferred provider
2. Check env vars: `PERPLEXITY_API_KEY`, `BRAVE_API_KEY`
3. If preferred provider's key exists → use it
4. If preferred provider's key missing but other exists → fall back with stderr warning
5. No keys → error with clear message

### Request Flow

1. Parse flags + positional query
2. Load and merge profile
3. Resolve provider from profile + available API keys
4. Build request with profile's system prompt, domain filters, constraints
5. Check cache (unless `--no-cache`)
6. If cache miss: execute search via provider
7. Cache response
8. Format output (markdown or JSON) to stdout
9. If sources omitted: append `<!-- run with --include-sources for citations -->`

## Response Caching

- Location: `~/.cache/websearch/` (respects `XDG_CACHE_HOME`)
- Cache key: SHA256 hash of (query + profile + mode + provider + relevant options)
- TTL: 60 minutes
- Auto-purge stale entries on each run
- `--include-sources` on cached result reuses cached data, appends sources
- Corrupted cache files silently ignored, refetched
- Disk full → skip caching, still return result

## Token-Efficient Output

- Default output strips sources/citations — just the answer
- Concise markdown, no decorative headers or boilerplate
- `--include-sources` appends sources at the bottom
- When sources omitted: `<!-- run with --include-sources for citations -->`
- `--json` provides structured output: `{query, provider, mode, content, sources, cached}`

## Error Handling

- All errors → stderr, results → stdout (pipe-friendly for agents)
- Exit code 0 on success, 1 on error
- No API keys → `"No API keys found. Set PERPLEXITY_API_KEY or BRAVE_API_KEY"`
- Mode incompatible with provider → `"Mode 'research' requires Perplexity. Set PERPLEXITY_API_KEY"`
- Provider fallback → stderr warning, not an error

## Built-in Profiles (v1)

1. **general** — General-purpose web search, no domain restrictions
2. **nodejs** — Node.js ecosystem focus, ESM examples, npm/official docs preferred
3. **python** — Python ecosystem focus, modern patterns, PyPI/official docs preferred

## Modes by Provider

| Provider | Modes |
|----------|-------|
| Perplexity | `ask`, `search`, `reason`, `research` |
| Brave | `web`, `local` |

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `gopkg.in/yaml.v3` — YAML profile parsing
- Standard library for HTTP, caching, JSON

## Testing

- Unit tests: profile loading/merging, cache key generation, output formatting
- Integration tests: mock HTTP server for provider clients
- No E2E tests against real APIs in CI

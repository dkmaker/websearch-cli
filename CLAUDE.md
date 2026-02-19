# websearch CLI

## Project Overview

Go CLI tool (`websearch`) for AI agents to search the web via Perplexity and Brave APIs. Profile-based configuration abstracts provider selection, modes, and formatting.

## Tech Stack

- **Language:** Go 1.24+
- **CLI Framework:** github.com/spf13/cobra
- **YAML:** gopkg.in/yaml.v3
- **Module:** github.com/dkmaker/websearch

## Architecture

```
cmd/websearch/main.go          Entry point
internal/
  cli/root.go                  Cobra command, flags, run() orchestration
  provider/provider.go         Provider interface, SearchOptions, Result, Source
  provider/perplexity.go       Perplexity API client (ask/search/reason/research)
  provider/brave.go            Brave Search API client (web)
  profile/profile.go           Profile loading (builtin + user), merging
  profile/embed.go             embed.FS for builtin YAML profiles
  profile/profiles/*.yaml      Built-in profiles (general, nodejs, python)
  cache/cache.go               File-based SHA256-keyed cache, 60-min TTL
  output/formatter.go          Markdown and JSON formatters
```

## Key Patterns

- **Provider interface:** All search providers implement `Search()`, `Name()`, `SupportedModes()`
- **Profile merging:** Built-in profiles load from embed.FS, user overrides from `~/.config/websearch/profiles/`, merge non-zero fields
- **Provider resolution:** Profile preference -> flag override -> API key availability -> fallback with warning
- **Mode auto-adjustment:** When provider changes from profile default, incompatible modes auto-adjust (unless user explicitly set `--mode`)
- **Output:** Errors to stderr, results to stdout. Sources stripped by default for token efficiency.

## Commands

```bash
go build -o websearch ./cmd/websearch/   # Build
go test ./...                             # Run all tests
go test ./internal/provider/...           # Test specific package
./websearch --list-profiles               # List profiles
./websearch "query"                       # Search
```

## Environment Variables

- `PERPLEXITY_API_KEY` — Required for Perplexity provider
- `BRAVE_API_KEY` — Required for Brave provider
- `XDG_CACHE_HOME` — Override cache directory (default: `~/.cache`)
- `XDG_CONFIG_HOME` — Override user profile directory (default: `~/.config`)

## API Details

### Perplexity

- Endpoint: `POST https://api.perplexity.ai/chat/completions`
- Auth: `Authorization: Bearer <key>`
- Models: sonar (ask), sonar-pro (search), sonar-reasoning (reason), sonar-deep-research (research)
- Response fields: `choices[0].message.content`, `citations[]`, `search_results[]`

### Brave

- Endpoint: `GET https://api.search.brave.com/res/v1/web/search`
- Auth: `X-Subscription-Token: <key>`
- Max 20 results per request
- Response fields: `web.results[].title`, `.url`, `.description`

## Testing

All tests use standard `testing` package. Provider tests use `httptest.NewServer` for mock HTTP servers. Cache tests use `t.TempDir()`. No external API calls in tests.

## Adding a New Provider

1. Create `internal/provider/newprovider.go` implementing the `Provider` interface
2. Add constructor `NewProvider(apiKey, baseURL string)`
3. Add tests with httptest mock server
4. Add env var check in `internal/cli/root.go` (`resolveProvider`)
5. Add provider case in `run()` switch
6. Add mode validation in `validateMode()`

## Adding a New Profile

Create a YAML file in `internal/profile/profiles/` and rebuild. The file is automatically embedded via `embed.FS`. Fields: name, description, provider, mode, system_prompt, domain_filter, max_results, max_tokens, output_format.

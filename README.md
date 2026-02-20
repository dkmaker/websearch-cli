# websearch

A Go CLI tool for AI agents to perform web searches via Perplexity, Brave, and GitHub APIs. Uses a profile system to abstract away provider selection, search modes, and result formatting — so agents just specify a profile and query.

## Installation

Download a prebuilt binary from the [latest release](https://github.com/dkmaker/websearch-cli/releases/latest), or install with Go:

```bash
go install github.com/dkmaker/websearch/cmd/websearch@latest
```

Or build from source:

```bash
git clone https://github.com/dkmaker/websearch-cli.git
cd websearch-cli
go build -o websearch ./cmd/websearch/
```

## Setup

Set at least one API key:

```bash
export PERPLEXITY_API_KEY="pplx-..."   # For Perplexity (ask, search, reason, research)
export BRAVE_API_KEY="BSA..."          # For Brave (web search)
```

Both can be set simultaneously — the profile determines which provider to use.

## Usage

```bash
# Basic search (uses "general" profile, Perplexity ask mode)
websearch "what is the Go programming language"

# Use a specific profile
websearch --profile nodejs "how to use streams in Node 22"
websearch --profile python "async patterns in Python 3.12"

# Override the search mode
websearch --mode research "compare React vs Svelte 2025"
websearch --mode reason "why is my Docker container slow"

# Include source citations
websearch --include-sources "latest Go releases"

# JSON output (for programmatic consumption)
websearch --json "kubernetes networking"

# Override provider
websearch --provider brave "local pizza restaurants"

# Skip cache
websearch --no-cache "breaking news today"

# List available profiles
websearch --list-profiles
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--profile` | `-p` | Profile name | `general` |
| `--mode` | `-m` | Search mode override | profile default |
| `--json` | | JSON output instead of markdown | `false` |
| `--provider` | | Provider override (`perplexity`, `brave`) | profile decides |
| `--max-results` | `-n` | Max result count | profile default |
| `--max-tokens` | | Max response tokens | profile default |
| `--include-sources` | | Include citations/sources in output | `false` |
| `--no-cache` | | Bypass response cache | `false` |
| `--list-profiles` | | List available profiles | |

## Search Modes

### Perplexity Modes

| Mode | Model | Best For |
|------|-------|----------|
| `ask` | sonar | Quick factual answers |
| `search` | sonar-pro | Complex queries, more sources |
| `reason` | sonar-reasoning | Step-by-step analysis, comparisons |
| `research` | sonar-deep-research | Comprehensive multi-source reports |

### Brave Modes

| Mode | Best For |
|------|----------|
| `web` | Raw search results with snippets |

## Profiles

Profiles bundle provider, mode, system prompt, domain filters, and output constraints into a single name. Three built-in profiles ship with the binary:

### general

General-purpose web search. No domain restrictions. Good for any question.

### nodejs

Focused on the Node.js ecosystem. Prefers official docs (nodejs.org, MDN, npmjs.com). System prompt requests modern ESM syntax and version compatibility notes.

### python

Focused on the Python ecosystem. Prefers official docs (docs.python.org, pypi.org, realpython.com). System prompt requests modern Python 3.10+ syntax.

### Custom Profiles

Create YAML files in `~/.config/websearch/profiles/` to add or override profiles:

```yaml
# ~/.config/websearch/profiles/devops.yaml
name: devops
description: "DevOps and infrastructure searches"
provider: perplexity
mode: ask
system_prompt: |
  Focus on DevOps, CI/CD, containerization, and cloud infrastructure.
  Include configuration examples. Prefer official documentation.
domain_filter:
  - kubernetes.io
  - docs.docker.com
  - docs.aws.amazon.com
max_results: 5
max_tokens: 2048
output_format: markdown
```

User profiles with the same name as a built-in profile override it (fields merge — unset fields inherit from the built-in).

### Profile Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Profile identifier |
| `description` | string | Human-readable description |
| `provider` | string | Preferred provider (`perplexity` or `brave`) |
| `mode` | string | Default search mode |
| `system_prompt` | string | System prompt sent to the provider |
| `domain_filter` | string[] | Restrict results to these domains |
| `max_results` | int | Maximum number of results |
| `max_tokens` | int | Maximum response tokens |
| `output_format` | string | Output format (`markdown` or `json`) |

## Output

### Default (Markdown)

By default, output is concise markdown with sources stripped for token efficiency. A hint comment is appended:

```
The Go programming language is an open-source language developed by Google...

- Built-in concurrency via goroutines
- Fast compilation to native binaries
- Rich standard library
<!-- run with --include-sources for citations -->
```

### With Sources

```bash
websearch --include-sources "Go programming"
```

Appends a sources section:

```
...

---
Sources:
- [Go Documentation](https://go.dev/doc/)
- [Go (programming language) - Wikipedia](https://en.wikipedia.org/wiki/Go_(programming_language))
```

### JSON Output

```bash
websearch --json "Go programming"
```

```json
{
  "content": "The Go programming language is...",
  "provider": "perplexity",
  "mode": "ask",
  "cached": false
}
```

With `--include-sources --json`, the `sources` array is included.

## Caching

Responses are cached locally to avoid repeated API calls:

- **Location:** `~/.cache/websearch/` (respects `XDG_CACHE_HOME`)
- **TTL:** 60 minutes
- **Key:** SHA256 hash of query + profile + mode + provider
- **Purge:** Stale entries auto-purged on each run

Use `--no-cache` to bypass both reading and writing to the cache.

## Provider Resolution

When both API keys are set, the provider is chosen by:

1. `--provider` flag (if set)
2. Profile's `provider` field
3. If preferred provider's key is missing, falls back to the other with a stderr warning
4. If the profile's mode is incompatible with the fallback provider, mode is auto-adjusted

Errors go to stderr, results to stdout — safe for piping.

## For AI Agents

This tool is designed to be called from AI agent workflows. Key design decisions:

- **Errors to stderr, results to stdout** — pipe-friendly
- **Exit code 0** on success, **1** on error
- **Token-efficient** — sources stripped by default
- **Profiles abstract complexity** — agents don't need to know about providers or modes
- **Caching** — repeated queries are fast and free
- **JSON mode** — structured output for programmatic consumption

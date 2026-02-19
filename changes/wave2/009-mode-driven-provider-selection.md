# Change 009: Mode-Driven Provider Selection

**Status:** Proposed
**Priority:** Medium
**Affected component:** `internal/cli/root.go`

---

## Problem

When a profile specifies a mode that Brave doesn't support (e.g., `ask`, `reason`, `research`), the current logic degrades the **mode** to fit the provider rather than switching the **provider** to fit the mode. This produces significantly lower-quality results.

### Affected Queries in Stress Test

5 queries had their mode degraded from `ask` to `web` because the provider resolved to Brave:

| Query | Original Mode | Actual Mode | Result Quality |
|-------|--------------|-------------|----------------|
| #11 | ask | web (Brave) | 2.5/5.0 — snippets only, no synthesized answer |
| #23 | ask | web (Brave) | 1.8/5.0 — bare link titles, covered only 50% of the question |
| #29 | ask | web (Brave) | 3.0/5.0 — short excerpts, no lifecycle event explanation |
| #34 | ask | web (Brave) | partial — mode adjusted warning, snippets instead of analysis |
| #48 | ask | web (Brave) | snippets only, no narrative explanation |

In all cases, the mode adjustment warning fires correctly (`root.go:148-150`), but the result quality gap is severe: **Brave `web` returns link snippets while Perplexity `ask` returns synthesized answers with reasoning.**

### Current Logic

```go
// root.go:142-151
if err := validateMode(mode, providerName); err != nil {
    if flagMode != "" {
        return err  // User explicitly set mode — error
    }
    // Mode came from profile, auto-adjust
    mode = defaultModeForProvider(providerName)
    modeAdjusted = true
    fmt.Fprintf(os.Stderr, "warning: mode adjusted to %q for %s provider\n", mode, providerName)
}
```

The mode is treated as the flexible variable and the provider is fixed. But from the user's perspective, the **mode defines the intent** (I want reasoning, I want research, I want synthesis) and the provider is an implementation detail.

## Proposed Solution

When mode incompatibility is detected and the mode came from the profile (not user flag), try to switch the **provider** to one that supports the mode, before falling back to mode degradation:

```go
// After resolveProvider, before validateMode
if err := validateMode(mode, providerName); err != nil {
    if flagMode != "" {
        return err // User explicitly set mode — error
    }

    // Try to find a provider that supports this mode
    if altProvider := findProviderForMode(mode, perplexityKey, braveKey, githubKey, providerName); altProvider != "" {
        fmt.Fprintf(os.Stderr, "warning: provider switched from %s to %s to support %q mode\n", providerName, altProvider, mode)
        providerName = altProvider
        modeAdjusted = false // mode preserved, provider changed
    } else {
        // No alternative provider available — degrade mode as before
        mode = defaultModeForProvider(providerName)
        modeAdjusted = true
        fmt.Fprintf(os.Stderr, "warning: mode adjusted to %q for %s provider\n", mode, providerName)
    }
}
```

### Provider Selection Helper

```go
func findProviderForMode(mode string, perplexityKey, braveKey, githubKey, currentProvider string) string {
    perplexityModes := map[string]bool{"ask": true, "search": true, "reason": true, "research": true}
    braveModes := map[string]bool{"web": true}
    githubModes := map[string]bool{"repos": true, "code": true, "issues": true}

    // Try providers in preference order, skip the current one
    candidates := []struct {
        name  string
        key   string
        modes map[string]bool
    }{
        {"perplexity", perplexityKey, perplexityModes},
        {"brave", braveKey, braveModes},
        {"github", githubKey, githubModes},
    }

    for _, c := range candidates {
        if c.name != currentProvider && c.key != "" && c.modes[mode] {
            return c.name
        }
    }
    return ""
}
```

## Impact

- **5 queries improve** from snippet-quality to synthesized-answer quality
- Mode intent is preserved — `ask` stays `ask`, `reason` stays `reason`
- The profile's `provider` preference is still respected when mode is compatible
- Warning message changes from "mode adjusted" to "provider switched" — agent can still detect the override
- If Perplexity key is not available, behavior falls back to current mode-degradation logic (no regression)

## Interaction with Existing Profiles

Profiles that explicitly set `provider: brave` with a Perplexity-only mode (like `reason`) would trigger this switch. This is actually the correct behavior — the profile author likely set the provider for domain filtering reasons, but the mode defines the expected output quality. If they truly want Brave specifically, they should set `mode: web`.

## Open Questions

- Should this also consider the profile's `domain_filter`? If a profile specifies Brave for its domain filtering capability, switching to Perplexity loses that. We could pass `domain_filter` through to Perplexity's `search_domain_filter` parameter if supported.
- Should the warning be more prominent? Provider switches change the billing (Perplexity costs more than Brave).
- Should there be a `--strict-provider` flag to disable this behavior?

---

## Decision

<!-- Approved / Rejected / Needs revision — your notes here -->


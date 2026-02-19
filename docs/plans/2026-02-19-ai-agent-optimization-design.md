# AI Agent Optimization — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make websearch self-describing and agent-optimized — no-args shows capabilities, Brave output is clean, version is exposed.

**Architecture:** Override Cobra's no-args behavior to emit a compact self-primer. Add `--show-examples` and `--show-profiles` as additive flags. Fix HTML entities in the Brave provider layer. Wire version through ldflags.

**Tech Stack:** Go 1.24+, Cobra, html stdlib package

---

## Design Reference

- **No-args = self-primer** (not an error). Agents discover capabilities by running the tool.
- **--show-examples and --show-profiles** are additive flags that expand the self-primer output.
- **HTML entity decoding** happens in the provider layer (brave.go), not the formatter.
- **Version** via Go ldflags, exposed through Cobra's built-in --version and in self-primer output.

---

### Task 1: Brave HTML Entity Decoding

**Files:**
- Modify: `internal/provider/brave.go:1-139`
- Test: `internal/provider/brave_test.go`

**Step 1: Write the failing test**

Add to `internal/provider/brave_test.go`:

```go
func TestBraveSearchHTMLEntities(t *testing.T) {
	mockResp := map[string]interface{}{
		"web": map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":       "Go &amp; Rust &mdash; A Comparison",
					"url":         "https://example.com/1",
					"description": "Find the best &quot;Go&quot; tips &amp; tricks. Use &#39;generics&#39; for &lt;type&gt; safety.",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	p := NewBrave("test-key", server.URL)
	result, err := p.Search(context.Background(), "test", SearchOptions{Mode: "web"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	// Content should have decoded HTML entities
	if strings.Contains(result.Content, "&amp;") {
		t.Errorf("content still contains &amp;: %s", result.Content)
	}
	if strings.Contains(result.Content, "&quot;") {
		t.Errorf("content still contains &quot;: %s", result.Content)
	}
	if strings.Contains(result.Content, "&#39;") {
		t.Errorf("content still contains &#39;: %s", result.Content)
	}
	if !strings.Contains(result.Content, `"Go"`) {
		t.Errorf("expected decoded quotes in content: %s", result.Content)
	}

	// Source title should also be decoded
	if result.Sources[0].Title != "Go & Rust — A Comparison" {
		t.Errorf("expected decoded title, got %q", result.Sources[0].Title)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/ -run TestBraveSearchHTMLEntities -v`
Expected: FAIL — content still contains `&amp;`, `&quot;`, etc.

**Step 3: Implement the fix**

In `internal/provider/brave.go`:

1. Add `"html"` to imports (line 4-13)
2. Apply `html.UnescapeString()` when building results (line 109-119):

```go
	for i, r := range bResp.Web.Results {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		title := html.UnescapeString(r.Title)
		desc := html.UnescapeString(r.Description)
		sb.WriteString(fmt.Sprintf("**%s**\n%s", title, desc))
		result.Sources = append(result.Sources, Source{
			Title: title,
			URL:   r.URL,
			Date:  r.PageAge,
		})
	}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/provider/brave.go internal/provider/brave_test.go
git commit -m "fix: decode HTML entities in Brave provider output"
```

---

### Task 2: Version Support

**Files:**
- Modify: `cmd/websearch/main.go:1-16`
- Modify: `internal/cli/root.go:30-46,60-62`

**Step 1: Add version variable and pass to CLI**

In `cmd/websearch/main.go`, add a `version` variable set via ldflags and pass to `cli.SetVersion()`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/dkmaker/websearch/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Wire version into Cobra command**

In `internal/cli/root.go`, add a package-level `appVersion` var and `SetVersion` func:

```go
var appVersion = "dev"

func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}
```

Cobra's built-in `--version` flag will now work automatically.

**Step 3: Run the build and verify**

Run: `go build -o /tmp/websearch ./cmd/websearch/ && /tmp/websearch --version`
Expected: `websearch version dev`

Run: `go build -ldflags "-X main.version=1.2.3" -o /tmp/websearch ./cmd/websearch/ && /tmp/websearch --version`
Expected: `websearch version 1.2.3`

**Step 4: Run all tests**

Run: `go test ./...`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add cmd/websearch/main.go internal/cli/root.go
git commit -m "feat: add version flag via ldflags"
```

---

### Task 3: Profile Detail Function

**Files:**
- Modify: `internal/profile/profile.go`
- Test: `internal/profile/profile_test.go`

**Step 1: Write the failing test**

Add to `internal/profile/profile_test.go`:

```go
func TestLoadAllBuiltin(t *testing.T) {
	profiles, err := LoadAllBuiltin()
	if err != nil {
		t.Fatalf("LoadAllBuiltin error: %v", err)
	}
	if len(profiles) < 3 {
		t.Errorf("expected at least 3 profiles, got %d", len(profiles))
	}
	// Verify they're sorted by name
	for i := 1; i < len(profiles); i++ {
		if profiles[i].Name < profiles[i-1].Name {
			t.Errorf("profiles not sorted: %q before %q", profiles[i-1].Name, profiles[i].Name)
		}
	}
	// Verify general profile is present and loaded
	found := false
	for _, p := range profiles {
		if p.Name == "general" {
			found = true
			if p.Provider == "" {
				t.Error("expected provider on general profile")
			}
		}
	}
	if !found {
		t.Error("expected general profile in results")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/profile/ -run TestLoadAllBuiltin -v`
Expected: FAIL — `LoadAllBuiltin` not defined

**Step 3: Implement LoadAllBuiltin**

Add to `internal/profile/profile.go`:

```go
import "sort"
```

```go
// LoadAllBuiltin loads and returns all built-in profiles, sorted by name.
func LoadAllBuiltin() ([]*Profile, error) {
	names := ListBuiltin()
	sort.Strings(names)
	var profiles []*Profile
	for _, name := range names {
		p, err := loadBuiltin(name)
		if err != nil {
			return nil, fmt.Errorf("loading builtin profile %q: %w", name, err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/profile/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/profile/profile.go internal/profile/profile_test.go
git commit -m "feat: add LoadAllBuiltin for profile discovery"
```

---

### Task 4: Self-Primer (No-Args Output)

**Files:**
- Modify: `internal/cli/root.go:18-46,48-62`
- Test: `internal/cli/root_test.go`

This is the core task. The no-args behavior changes from error to self-primer output.

**Step 1: Write the failing test**

Add to `internal/cli/root_test.go`:

```go
func TestSelfPrimer(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "brave-key", false, false)

	// Must contain version
	if !strings.Contains(output, "websearch v") {
		t.Error("expected version in primer")
	}
	// Must show both providers as ready
	if !strings.Contains(output, "perplexity (ready)") {
		t.Error("expected perplexity (ready)")
	}
	if !strings.Contains(output, "brave (ready)") {
		t.Error("expected brave (ready)")
	}
	// Must show modes
	if !strings.Contains(output, "ask*") {
		t.Error("expected ask* (default mode)")
	}
	// Must show profiles
	if !strings.Contains(output, "general*") {
		t.Error("expected general* (default profile)")
	}
	// Must show usage
	if !strings.Contains(output, "Usage:") {
		t.Error("expected Usage line")
	}
	// Must NOT contain examples or profile details
	if strings.Contains(output, "Examples:") {
		t.Error("base primer should not contain examples")
	}
}

func TestSelfPrimerNoKey(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "", false, false)
	if !strings.Contains(output, "brave (no key)") {
		t.Error("expected brave (no key)")
	}
}

func TestSelfPrimerWithExamples(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "", true, false)
	if !strings.Contains(output, "Examples:") {
		t.Error("expected Examples section")
	}
}

func TestSelfPrimerWithProfiles(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "", false, true)
	if !strings.Contains(output, "Profiles:") {
		t.Error("expected Profiles section heading")
	}
	if !strings.Contains(output, "General-purpose") {
		t.Error("expected profile descriptions")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestSelfPrimer -v`
Expected: FAIL — `buildSelfPrimer` not defined

**Step 3: Implement self-primer**

In `internal/cli/root.go`:

1. Add new flag variables (alongside existing flags at line 18-28):

```go
	flagShowExamples bool
	flagShowProfiles bool
```

2. Register new flags in `init()` (after line 57):

```go
	rootCmd.Flags().BoolVar(&flagShowExamples, "show-examples", false, "Show usage examples")
	rootCmd.Flags().BoolVar(&flagShowProfiles, "show-profiles", false, "Show profile details")
```

3. Update the Args validator (line 34-42) to allow no-args when showing primer:

```go
	Args: func(cmd *cobra.Command, args []string) error {
		if flagListProfiles || flagShowExamples || flagShowProfiles {
			return nil
		}
		if len(args) == 0 {
			return nil // will trigger self-primer in run()
		}
		if len(args) != 1 {
			return fmt.Errorf("requires exactly 1 query argument")
		}
		return nil
	},
```

4. Update `run()` (line 64-69) to handle self-primer before query:

```go
func run(cmd *cobra.Command, args []string) error {
	if flagListProfiles {
		return listProfiles()
	}

	// Self-primer: no query provided
	if len(args) == 0 {
		perplexityKey := os.Getenv("PERPLEXITY_API_KEY")
		braveKey := os.Getenv("BRAVE_API_KEY")
		fmt.Print(buildSelfPrimer(perplexityKey, braveKey, flagShowExamples, flagShowProfiles))
		return nil
	}

	query := args[0]
	// ... rest of existing run() unchanged
```

5. Add the `buildSelfPrimer` function:

```go
func buildSelfPrimer(perplexityKey, braveKey string, showExamples, showProfiles bool) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("websearch v%s — Web search CLI for AI agents\n\n", appVersion))

	// Provider status
	pStatus := "no key"
	if perplexityKey != "" {
		pStatus = "ready"
	}
	bStatus := "no key"
	if braveKey != "" {
		bStatus = "ready"
	}
	sb.WriteString(fmt.Sprintf("Providers: perplexity (%s), brave (%s)\n", pStatus, bStatus))

	// Modes
	sb.WriteString("Modes [perplexity]: ask*, search, reason, research\n")
	sb.WriteString("Modes [brave]: web*\n")

	// Profiles
	names := profile.ListBuiltin()
	sb.WriteString(fmt.Sprintf("Profiles: %s\n", formatProfileNames(names)))

	// Output & Cache
	sb.WriteString("Output: markdown (default), json (--json)\n")
	sb.WriteString("Cache: 60m TTL (--no-cache to bypass)\n")

	// Usage
	sb.WriteString("\nUsage: websearch [flags] \"query\"\n")
	sb.WriteString("Key flags: -p profile, -m mode, --provider, --json, --include-sources, --no-cache\n")
	sb.WriteString("More: --show-examples, --show-profiles, --help\n")

	// Optional: examples
	if showExamples {
		sb.WriteString("\nExamples:\n")
		sb.WriteString("  websearch \"what is Go generics\"                          # general ask\n")
		sb.WriteString("  websearch -m research \"AI trends 2026\"                   # deep research\n")
		sb.WriteString("  websearch --provider brave \"local restaurants\"            # brave web search\n")
		sb.WriteString("  websearch -p python \"fastapi middleware\"                  # python profile\n")
		sb.WriteString("  websearch --json --include-sources \"kubernetes basics\"    # JSON with sources\n")
	}

	// Optional: profile details
	if showProfiles {
		sb.WriteString("\nProfiles:\n")
		profiles, err := profile.LoadAllBuiltin()
		if err == nil {
			for _, p := range profiles {
				line := fmt.Sprintf("  %-9s %s/%-8s %s", p.Name, capitalize(p.Provider), p.Mode, p.Description)
				if len(p.DomainFilter) > 0 {
					line += fmt.Sprintf(" (filters: %s)", strings.Join(p.DomainFilter, ", "))
				}
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString("\nCustom profiles: ~/.config/websearch/profiles/<name>.yaml\n")
	}

	return sb.String()
}

func formatProfileNames(names []string) string {
	for i, n := range names {
		if n == "general" {
			names[i] = n + "*"
		}
	}
	return strings.Join(names, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
```

6. Add `"strings"` to imports if not already present (it is already imported via the `strings` import for other uses — but verify).

**Step 4: Run all tests**

Run: `go test ./internal/cli/ -v`
Expected: ALL PASS

Run: `go test ./...`
Expected: ALL PASS

**Step 5: Build and manually verify**

Run: `go build -o /tmp/websearch ./cmd/websearch/ && /tmp/websearch`
Expected: Self-primer output with version, providers, modes, profiles

Run: `/tmp/websearch --show-examples`
Expected: Self-primer + examples section

Run: `/tmp/websearch --show-profiles`
Expected: Self-primer + profile details

Run: `/tmp/websearch --show-examples --show-profiles`
Expected: Self-primer + both sections

Run: `/tmp/websearch "test query"`
Expected: Normal search behavior (unchanged)

**Step 6: Commit**

```bash
git add internal/cli/root.go internal/cli/root_test.go
git commit -m "feat: self-describing no-args primer for AI agent discovery

Running websearch with no arguments now outputs a compact capability
summary showing version, provider status, modes, profiles, and usage.
--show-examples and --show-profiles expand the output."
```

---

### Task 5: Final Verification & Cleanup

**Files:**
- All modified files

**Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

**Step 2: Build and run end-to-end**

Run: `go build -o /tmp/websearch ./cmd/websearch/`

Verify each scenario:
- `/tmp/websearch` — shows self-primer
- `/tmp/websearch --version` — shows version
- `/tmp/websearch --show-examples` — shows primer + examples
- `/tmp/websearch --show-profiles` — shows primer + profiles
- `/tmp/websearch --show-examples --show-profiles` — shows all
- `/tmp/websearch --list-profiles` — existing behavior unchanged
- `/tmp/websearch --help` — Cobra help unchanged

**Step 3: Commit if any final fixes needed**

Only commit if Step 2 revealed issues that needed fixing.

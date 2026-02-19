// internal/cli/root.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dkmaker/websearch/internal/cache"
	"github.com/dkmaker/websearch/internal/output"
	"github.com/dkmaker/websearch/internal/profile"
	"github.com/dkmaker/websearch/internal/provider"
	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

var (
	flagProfile        string
	flagMode           string
	flagJSON           bool
	flagProvider       string
	flagMaxResults     int
	flagMaxTokens      int
	flagIncludeSources bool
	flagNoCache        bool
	flagListProfiles   bool
	flagShowExamples   bool
	flagShowProfiles   bool
)

var rootCmd = &cobra.Command{
	Use:   "websearch [flags] <query>",
	Short: "Web search CLI for AI agents",
	Long:  "A CLI tool for AI agents to perform web searches via Perplexity, Brave, and GitHub APIs using profile-based configuration.",
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
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          run,
}

func init() {
	rootCmd.Flags().StringVarP(&flagProfile, "profile", "p", "general", "Profile name")
	rootCmd.Flags().StringVarP(&flagMode, "mode", "m", "", "Search mode override")
	rootCmd.Flags().BoolVar(&flagJSON, "json", false, "Output as JSON")
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "Provider override (perplexity, brave, github)")
	rootCmd.Flags().IntVarP(&flagMaxResults, "max-results", "n", 0, "Max result count")
	rootCmd.Flags().IntVar(&flagMaxTokens, "max-tokens", 0, "Max response tokens")
	rootCmd.Flags().BoolVar(&flagIncludeSources, "include-sources", false, "Include citations/sources")
	rootCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "Bypass response cache")
	rootCmd.Flags().BoolVar(&flagListProfiles, "list-profiles", false, "List available profiles")
	rootCmd.Flags().BoolVar(&flagShowExamples, "show-examples", false, "Show usage examples")
	rootCmd.Flags().BoolVar(&flagShowProfiles, "show-profiles", false, "Show profile details")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	if flagListProfiles {
		return listProfiles()
	}

	// Self-primer: no query provided
	if len(args) == 0 {
		perplexityKey := os.Getenv("PERPLEXITY_API_KEY")
		braveKey := os.Getenv("BRAVE_API_KEY")
		githubKey := os.Getenv("GITHUB_TOKEN")
		fmt.Print(buildSelfPrimer(perplexityKey, braveKey, githubKey, flagShowExamples, flagShowProfiles))
		return nil
	}

	query := args[0]

	// Load profile
	prof, err := profile.Load(flagProfile)
	if err != nil {
		return fmt.Errorf("loading profile: %w", err)
	}

	// Apply flag overrides to profile
	if flagMaxResults > 0 {
		prof.MaxResults = flagMaxResults
	}
	if flagMaxTokens > 0 {
		prof.MaxTokens = flagMaxTokens
	}

	// Resolve mode
	mode := flagMode
	if mode == "" {
		mode = prof.Mode
	}

	// Get API keys
	perplexityKey := os.Getenv("PERPLEXITY_API_KEY")
	braveKey := os.Getenv("BRAVE_API_KEY")
	githubKey := os.Getenv("GITHUB_TOKEN")

	// Resolve provider
	providerName, warning, err := resolveProvider(prof.Provider, flagProvider, perplexityKey, braveKey, githubKey)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// Validate mode against provider. If the mode came from the profile
	// (not explicitly set by user) and is incompatible with the resolved
	// provider, auto-adjust to the provider's default mode.
	if err := validateMode(mode, providerName); err != nil {
		if flagMode != "" {
			// User explicitly requested this mode — error
			return err
		}
		// Mode came from profile, auto-adjust
		mode = defaultModeForProvider(providerName)
		fmt.Fprintf(os.Stderr, "warning: mode adjusted to %q for %s provider\n", mode, providerName)
	}

	// Build search options
	opts := provider.SearchOptions{
		Mode:         mode,
		MaxTokens:    prof.MaxTokens,
		MaxResults:   prof.MaxResults,
		SystemPrompt: prof.SystemPrompt,
		DomainFilter: prof.DomainFilter,
	}

	// Check cache
	responseCache := cache.New(cache.DefaultDir(), 60*time.Minute)

	// Purge stale entries in background
	go responseCache.Purge()

	cacheKey := cache.Key(query, flagProfile, mode, providerName)

	if !flagNoCache {
		if data, ok := responseCache.Get(cacheKey); ok {
			var result provider.Result
			if err := json.Unmarshal(data, &result); err == nil {
				result.Cached = true
				return outputResult(&result)
			}
		}
	}

	// Create provider
	var prov provider.Provider
	switch providerName {
	case "perplexity":
		prov = provider.NewPerplexity(perplexityKey, "")
	case "brave":
		prov = provider.NewBrave(braveKey, "")
	case "github":
		prov = provider.NewGitHub(githubKey, "")
	}

	// Execute search
	result, err := prov.Search(context.Background(), query, opts)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Cache result
	if !flagNoCache {
		if data, err := json.Marshal(result); err == nil {
			responseCache.Set(cacheKey, data)
		}
	}

	return outputResult(result)
}

func outputResult(result *provider.Result) error {
	if flagJSON {
		out, err := output.FormatJSON(result, flagIncludeSources)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Print(output.FormatMarkdown(result, flagIncludeSources))
	}
	return nil
}

func listProfiles() error {
	names := profile.ListBuiltin()
	for _, name := range names {
		p, err := profile.Load(name)
		if err != nil {
			fmt.Printf("  %s (error loading)\n", name)
			continue
		}
		fmt.Printf("  %-12s %s (provider: %s, mode: %s)\n", name, p.Description, p.Provider, p.Mode)
	}
	return nil
}

// resolveProvider determines which provider to use based on profile preference,
// flag override, and available API keys.
func resolveProvider(profileProv, flagProv, perplexityKey, braveKey, githubKey string) (string, string, error) {
	if perplexityKey == "" && braveKey == "" && profileProv != "github" && flagProv != "github" {
		return "", "", fmt.Errorf("no API keys found. Set PERPLEXITY_API_KEY, BRAVE_API_KEY, or GITHUB_TOKEN")
	}

	preferred := profileProv
	if flagProv != "" {
		preferred = flagProv
	}

	switch preferred {
	case "perplexity":
		if perplexityKey != "" {
			return "perplexity", "", nil
		}
		return "brave", "warning: PERPLEXITY_API_KEY not set, falling back to Brave", nil
	case "brave":
		if braveKey != "" {
			return "brave", "", nil
		}
		return "perplexity", "warning: BRAVE_API_KEY not set, falling back to Perplexity", nil
	case "github":
		return "github", "", nil
	default:
		if perplexityKey != "" {
			return "perplexity", "", nil
		}
		return "brave", "", nil
	}
}

// validateMode checks that the requested mode is supported by the provider.
func validateMode(mode, providerName string) error {
	if mode == "" {
		return nil
	}

	perplexityModes := map[string]bool{"ask": true, "search": true, "reason": true, "research": true}
	braveModes := map[string]bool{"web": true}
	githubModes := map[string]bool{"repos": true, "code": true, "issues": true}

	switch providerName {
	case "perplexity":
		if !perplexityModes[mode] {
			return fmt.Errorf("mode %q not supported by Perplexity (supported: ask, search, reason, research)", mode)
		}
	case "brave":
		if !braveModes[mode] {
			return fmt.Errorf("mode %q requires Perplexity. Set PERPLEXITY_API_KEY", mode)
		}
	case "github":
		if !githubModes[mode] {
			return fmt.Errorf("mode %q not supported by GitHub (supported: repos, code, issues)", mode)
		}
	}
	return nil
}

// defaultModeForProvider returns the default search mode for a provider.
func defaultModeForProvider(providerName string) string {
	switch providerName {
	case "brave":
		return "web"
	case "github":
		return "repos"
	default:
		return "ask"
	}
}

func buildSelfPrimer(perplexityKey, braveKey, githubKey string, showExamples, showProfiles bool) string {
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
	gStatus := "no key"
	if githubKey != "" {
		gStatus = "ready"
	}
	sb.WriteString(fmt.Sprintf("Providers: perplexity (%s), brave (%s), github (%s)\n", pStatus, bStatus, gStatus))

	// Modes
	sb.WriteString("Modes [perplexity]: ask*, search, reason, research\n")
	sb.WriteString("Modes [brave]: web*\n")
	sb.WriteString("Modes [github]: repos*, code, issues\n")

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
		sb.WriteString("  websearch -p github \"cobra CLI framework\"                  # github repos\n")
		sb.WriteString("  websearch -p github -m code \"http.ListenAndServe lang:go\"   # github code\n")
	}

	// Optional: profile details
	if showProfiles {
		sb.WriteString("\nProfiles:\n")
		profiles, err := profile.LoadAllBuiltin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load profiles: %v\n", err)
		} else {
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
	formatted := make([]string, len(names))
	copy(formatted, names)
	for i, n := range formatted {
		if n == "general" {
			formatted[i] = n + "*"
		}
	}
	return strings.Join(formatted, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

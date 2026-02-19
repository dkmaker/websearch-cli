// internal/cli/root.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dkmaker/websearch/internal/cache"
	"github.com/dkmaker/websearch/internal/output"
	"github.com/dkmaker/websearch/internal/profile"
	"github.com/dkmaker/websearch/internal/provider"
	"github.com/spf13/cobra"
)

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
)

var rootCmd = &cobra.Command{
	Use:   "websearch [flags] <query>",
	Short: "Web search CLI for AI agents",
	Long:  "A CLI tool for AI agents to perform web searches via Perplexity and Brave APIs using profile-based configuration.",
	Args: func(cmd *cobra.Command, args []string) error {
		if flagListProfiles {
			return nil
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
	rootCmd.Flags().StringVar(&flagProvider, "provider", "", "Provider override (perplexity, brave)")
	rootCmd.Flags().IntVarP(&flagMaxResults, "max-results", "n", 0, "Max result count")
	rootCmd.Flags().IntVar(&flagMaxTokens, "max-tokens", 0, "Max response tokens")
	rootCmd.Flags().BoolVar(&flagIncludeSources, "include-sources", false, "Include citations/sources")
	rootCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "Bypass response cache")
	rootCmd.Flags().BoolVar(&flagListProfiles, "list-profiles", false, "List available profiles")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	if flagListProfiles {
		return listProfiles()
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

	// Resolve provider
	providerName, warning, err := resolveProvider(prof.Provider, flagProvider, perplexityKey, braveKey)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// Validate mode against provider
	if err := validateMode(mode, providerName); err != nil {
		return err
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
func resolveProvider(profileProv, flagProv, perplexityKey, braveKey string) (string, string, error) {
	if perplexityKey == "" && braveKey == "" {
		return "", "", fmt.Errorf("no API keys found. Set PERPLEXITY_API_KEY or BRAVE_API_KEY")
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

	switch providerName {
	case "perplexity":
		if !perplexityModes[mode] {
			return fmt.Errorf("mode %q not supported by Perplexity (supported: ask, search, reason, research)", mode)
		}
	case "brave":
		if !braveModes[mode] {
			return fmt.Errorf("mode %q requires Perplexity. Set PERPLEXITY_API_KEY", mode)
		}
	}
	return nil
}

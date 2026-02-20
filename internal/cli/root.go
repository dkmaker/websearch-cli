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
	flagNoMetadata     bool
)

// GitHub qualifier flags
var (
	flagGHLanguage  string
	flagGHUser      string
	flagGHOrg       string
	flagGHRepo      string
	flagGHStars     string
	flagGHTopic     string
	flagGHLicense   string
	flagGHArchived  string
	flagGHFork      string
	flagGHPushed    string
	flagGHCreated   string
	flagGHSort      string
	flagGHFilename  string
	flagGHExtension string
	flagGHPath      string
	flagGHState     string
	flagGHLabel     string
	flagGHAuthor    string
	flagGHAssignee  string
	flagGHIn        string
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
	rootCmd.Flags().BoolVar(&flagNoMetadata, "no-metadata", false, "Omit response metadata from output")

	// GitHub qualifier flags
	rootCmd.Flags().StringVar(&flagGHLanguage, "gh-language", "", "GitHub: filter by language (e.g., go, python)")
	rootCmd.Flags().StringVar(&flagGHUser, "gh-user", "", "GitHub: filter by user/owner")
	rootCmd.Flags().StringVar(&flagGHOrg, "gh-org", "", "GitHub: filter by organization")
	rootCmd.Flags().StringVar(&flagGHRepo, "gh-repo", "", "GitHub: filter by repo (owner/name)")
	rootCmd.Flags().StringVar(&flagGHStars, "gh-stars", "", "GitHub: filter by stars (e.g., >100, 10..50)")
	rootCmd.Flags().StringVar(&flagGHTopic, "gh-topic", "", "GitHub: filter by topic")
	rootCmd.Flags().StringVar(&flagGHLicense, "gh-license", "", "GitHub: filter by license (e.g., mit, apache-2.0)")
	rootCmd.Flags().StringVar(&flagGHArchived, "gh-archived", "", "GitHub: filter archived repos (true/false)")
	rootCmd.Flags().StringVar(&flagGHFork, "gh-fork", "", "GitHub: filter forks (true/only)")
	rootCmd.Flags().StringVar(&flagGHPushed, "gh-pushed", "", "GitHub: filter by last push date (e.g., >2024-01-01)")
	rootCmd.Flags().StringVar(&flagGHCreated, "gh-created", "", "GitHub: filter by creation date (e.g., >2024-01-01)")
	rootCmd.Flags().StringVar(&flagGHSort, "gh-sort", "", "GitHub: sort repos by (stars, forks, updated)")
	rootCmd.Flags().StringVar(&flagGHFilename, "gh-filename", "", "GitHub: filter code by filename")
	rootCmd.Flags().StringVar(&flagGHExtension, "gh-extension", "", "GitHub: filter code by file extension")
	rootCmd.Flags().StringVar(&flagGHPath, "gh-path", "", "GitHub: filter code by directory path")
	rootCmd.Flags().StringVar(&flagGHState, "gh-state", "", "GitHub: filter issues by state (open/closed)")
	rootCmd.Flags().StringVar(&flagGHLabel, "gh-label", "", "GitHub: filter issues by label")
	rootCmd.Flags().StringVar(&flagGHAuthor, "gh-author", "", "GitHub: filter issues by author")
	rootCmd.Flags().StringVar(&flagGHAssignee, "gh-assignee", "", "GitHub: filter issues by assignee")
	rootCmd.Flags().StringVar(&flagGHIn, "gh-in", "", "GitHub: search in fields (name,description / title,body)")
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

	// Auto-scale max_tokens for research mode to prevent truncation.
	// Research mode (sonar-deep-research) produces long-form responses
	// (3,000-10,000+ tokens) that get cut off at the default 2048.
	if shouldAutoScaleTokens(mode, flagMaxTokens, prof.MaxTokens) {
		prof.MaxTokens = researchTokenScaleTarget
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
	modeAdjusted := false
	if err := validateMode(mode, providerName); err != nil {
		if flagMode != "" {
			// User explicitly requested this mode — error
			return err
		}
		// Mode came from profile, auto-adjust
		mode = defaultModeForProvider(providerName)
		modeAdjusted = true
		fmt.Fprintf(os.Stderr, "warning: mode adjusted to %q for %s provider\n", mode, providerName)
	}

	ghQualifiers := buildGitHubQualifiers()

	// Validate --gh-* flags are only used with GitHub provider
	if !ghQualifiers.IsEmpty() && providerName != "github" {
		return fmt.Errorf("--gh-* flags are only supported with the GitHub provider (current: %s)", providerName)
	}

	// Build search options
	opts := provider.SearchOptions{
		Mode:         mode,
		MaxTokens:    prof.MaxTokens,
		MaxResults:   prof.MaxResults,
		SystemPrompt: prof.SystemPrompt,
		DomainFilter: prof.DomainFilter,
		GitHub:       ghQualifiers,
	}

	// Check cache
	responseCache := cache.New(cache.DefaultDir(), 60*time.Minute)

	// Purge stale entries in background
	go responseCache.Purge()

	cacheKey := cache.Key(query, flagProfile, mode, providerName, ghQualifiers.CacheKey())

	if !flagNoCache {
		if data, ok := responseCache.Get(cacheKey); ok {
			var result provider.Result
			if err := json.Unmarshal(data, &result); err == nil {
				result.Cached = true
				result.ModeAdjusted = modeAdjusted

				// Empty result detection for cached results
				if strings.TrimSpace(result.Content) == "" {
					fmt.Fprintln(os.Stderr, "warning: search returned no results")
					os.Exit(2)
				}

				// Deflection detection for cached results
				if warning := detectDeflection(result.Content); warning != "" {
					fmt.Fprintln(os.Stderr, warning)
				}

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

	// Set mode adjustment metadata
	result.ModeAdjusted = modeAdjusted

	// Empty result detection — exit code 2
	if strings.TrimSpace(result.Content) == "" {
		fmt.Fprintln(os.Stderr, "warning: search returned no results")
		os.Exit(2)
	}

	// Deflection detection — stderr warning only, still exit 0
	if warning := detectDeflection(result.Content); warning != "" {
		fmt.Fprintln(os.Stderr, warning)
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
	includeMetadata := !flagNoMetadata
	if flagJSON {
		out, err := output.FormatJSON(result, flagIncludeSources, includeMetadata)
		if err != nil {
			return err
		}
		fmt.Println(out)
	} else {
		fmt.Print(output.FormatMarkdown(result, flagIncludeSources, includeMetadata))
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
		if githubKey == "" {
			return "github", "warning: GITHUB_TOKEN not set; GitHub searches will be rate-limited and code search will fail", nil
		}
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
	sb.WriteString("GitHub flags: --gh-language, --gh-stars, --gh-sort, --gh-user, --gh-org, --gh-repo, --gh-topic,\n")
	sb.WriteString("  --gh-license, --gh-archived, --gh-fork, --gh-pushed, --gh-created, --gh-in,\n")
	sb.WriteString("  --gh-filename, --gh-extension, --gh-path (code), --gh-state, --gh-label, --gh-author, --gh-assignee (issues)\n")
	sb.WriteString("More: --show-examples, --show-profiles, --help\n")

	// Optional: examples
	if showExamples {
		sb.WriteString("\nExamples:\n")
		sb.WriteString("  websearch \"what is Go generics\"                          # general ask\n")
		sb.WriteString("  websearch -m research \"AI trends 2026\"                   # deep research\n")
		sb.WriteString("  websearch --provider brave \"local restaurants\"            # brave web search\n")
		sb.WriteString("  websearch -p python \"fastapi middleware\"                  # python profile\n")
		sb.WriteString("  websearch --json --include-sources \"kubernetes basics\"    # JSON with sources\n")
		sb.WriteString("  websearch -p github --gh-language go --gh-stars '>1000' \"web framework\"  # github repos with qualifiers\n")
		sb.WriteString("  websearch -p github -m issues --gh-state open --gh-label bug \"crash\"     # github issues filtered\n")
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

// deflectionPrefixes are known phrases that indicate the model deflected
// rather than providing a substantive answer.
var deflectionPrefixes = []string{
	"The search results provided do not contain",
	"The search results do not contain",
	"The available search results don't",
	"I don't have enough information",
}

// detectDeflection checks the beginning of a response for known deflection
// patterns. Returns a warning string if detected, empty string otherwise.
func detectDeflection(content string) string {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	for _, prefix := range deflectionPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return "warning: response may not address the query (search context insufficient)"
		}
	}
	return ""
}

// researchTokenScaleThreshold is the max_tokens value at or below which
// research mode auto-scales to researchTokenScaleTarget.
const researchTokenScaleThreshold = 2048

// researchTokenScaleTarget is the max_tokens value used for research mode
// when auto-scaling is triggered.
const researchTokenScaleTarget = 16384

// shouldAutoScaleTokens returns true if research mode token auto-scaling
// should be applied. This happens when mode is "research", the user didn't
// explicitly set --max-tokens (flagMaxTokens == 0), and the profile's
// max_tokens is at or below the threshold.
func shouldAutoScaleTokens(mode string, flagMaxTokens, profileMaxTokens int) bool {
	return mode == "research" && flagMaxTokens == 0 && profileMaxTokens <= researchTokenScaleThreshold
}

func buildGitHubQualifiers() provider.GitHubQualifiers {
	return provider.GitHubQualifiers{
		Language:  flagGHLanguage,
		User:      flagGHUser,
		Org:       flagGHOrg,
		Repo:      flagGHRepo,
		Stars:     flagGHStars,
		Topic:     flagGHTopic,
		License:   flagGHLicense,
		Archived:  flagGHArchived,
		Fork:      flagGHFork,
		Pushed:    flagGHPushed,
		Created:   flagGHCreated,
		Sort:      flagGHSort,
		Filename:  flagGHFilename,
		Extension: flagGHExtension,
		Path:      flagGHPath,
		State:     flagGHState,
		Label:     flagGHLabel,
		Author:    flagGHAuthor,
		Assignee:  flagGHAssignee,
		In:        flagGHIn,
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

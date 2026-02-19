// internal/cli/root_test.go
package cli

import (
	"strings"
	"testing"
)

func TestSelfPrimer(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "brave-key", "", false, false)

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
	output := buildSelfPrimer("pplx-key", "", "", false, false)
	if !strings.Contains(output, "brave (no key)") {
		t.Error("expected brave (no key)")
	}
}

func TestSelfPrimerWithExamples(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "", "", true, false)
	if !strings.Contains(output, "Examples:") {
		t.Error("expected Examples section")
	}
}

func TestSelfPrimerWithProfiles(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "", "", false, true)
	if !strings.Contains(output, "Profiles:") {
		t.Error("expected Profiles section heading")
	}
	if !strings.Contains(output, "General-purpose") {
		t.Error("expected profile descriptions")
	}
}

func TestSelfPrimerWithBothFlags(t *testing.T) {
	output := buildSelfPrimer("pplx-key", "brave-key", "", true, true)
	if !strings.Contains(output, "Examples:") {
		t.Error("expected Examples section")
	}
	if !strings.Contains(output, "Profiles:") {
		t.Error("expected Profiles section")
	}
	if !strings.Contains(output, "General-purpose") {
		t.Error("expected profile descriptions")
	}
}

func TestResolveProvider(t *testing.T) {
	tests := []struct {
		name         string
		profileProv  string
		flagProv     string
		perplexKey   string
		braveKey     string
		wantProvider string
		wantErr      bool
	}{
		{
			name:         "profile wants perplexity, key available",
			profileProv:  "perplexity",
			perplexKey:   "pplx-key",
			wantProvider: "perplexity",
		},
		{
			name:         "profile wants perplexity, only brave key",
			profileProv:  "perplexity",
			braveKey:     "brave-key",
			wantProvider: "brave",
		},
		{
			name:    "no keys at all",
			wantErr: true,
		},
		{
			name:         "flag overrides profile",
			profileProv:  "perplexity",
			flagProv:     "brave",
			perplexKey:   "pplx-key",
			braveKey:     "brave-key",
			wantProvider: "brave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := resolveProvider(tt.profileProv, tt.flagProv, tt.perplexKey, tt.braveKey, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.wantProvider {
				t.Errorf("got provider %q, want %q", got, tt.wantProvider)
			}
		})
	}
}

func TestValidateMode(t *testing.T) {
	tests := []struct {
		mode     string
		provider string
		wantErr  bool
	}{
		{"ask", "perplexity", false},
		{"research", "perplexity", false},
		{"web", "brave", false},
		{"research", "brave", true},
		{"", "perplexity", false},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"_"+tt.provider, func(t *testing.T) {
			err := validateMode(tt.mode, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMode(%q, %q) error = %v, wantErr %v", tt.mode, tt.provider, err, tt.wantErr)
			}
		})
	}
}

func TestResolveProviderGitHub(t *testing.T) {
	// GitHub with token
	got, _, err := resolveProvider("github", "", "", "", "ghp_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}

	// GitHub without token (should still work — unauthenticated OK for repos/issues, but warn)
	got, warning, err := resolveProvider("github", "", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}
	if !strings.Contains(warning, "GITHUB_TOKEN not set") {
		t.Errorf("expected warning about GITHUB_TOKEN, got %q", warning)
	}

	// Flag override to github without token — should warn
	got, warning, err = resolveProvider("perplexity", "github", "pplx-key", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}
	if !strings.Contains(warning, "GITHUB_TOKEN not set") {
		t.Errorf("expected warning about GITHUB_TOKEN, got %q", warning)
	}

	// Flag override to github with token — no warning
	got, warning, err = resolveProvider("perplexity", "github", "pplx-key", "", "ghp_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "github" {
		t.Errorf("got %q, want github", got)
	}
	if warning != "" {
		t.Errorf("expected no warning with token, got %q", warning)
	}
}

func TestValidateModeGitHub(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"repos", false},
		{"code", false},
		{"issues", false},
		{"web", true},
		{"ask", true},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"_github", func(t *testing.T) {
			err := validateMode(tt.mode, "github")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMode(%q, github) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
			}
		})
	}
}

func TestSelfPrimerGitHub(t *testing.T) {
	output := buildSelfPrimer("", "", "ghp_token", false, false)
	if !strings.Contains(output, "github (ready)") {
		t.Error("expected github (ready)")
	}
	if !strings.Contains(output, "repos*") {
		t.Error("expected repos* mode")
	}
}

func TestDetectDeflection(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "exact prefix match",
			content: "The search results provided do not contain information about this topic.",
			want:    true,
		},
		{
			name:    "case insensitive match",
			content: "the search results provided do not contain relevant data.",
			want:    true,
		},
		{
			name:    "prefix with leading whitespace",
			content: "  The search results do not contain specific information about...",
			want:    true,
		},
		{
			name:    "available search results don't",
			content: "The available search results don't address this specific issue.",
			want:    true,
		},
		{
			name:    "not enough information",
			content: "I don't have enough information to answer this question.",
			want:    true,
		},
		{
			name:    "normal response",
			content: "Here is the answer to your question about Go generics...",
			want:    false,
		},
		{
			name:    "empty content",
			content: "",
			want:    false,
		},
		{
			name:    "deflection phrase mid-sentence",
			content: "Based on my analysis, the search results provided do not contain much.",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDeflection(tt.content)
			if tt.want && result == "" {
				t.Error("expected deflection warning, got empty string")
			}
			if !tt.want && result != "" {
				t.Errorf("expected no deflection warning, got %q", result)
			}
			if tt.want && result != "" {
				expected := "warning: response may not address the query (search context insufficient)"
				if result != expected {
					t.Errorf("got warning %q, want %q", result, expected)
				}
			}
		})
	}
}

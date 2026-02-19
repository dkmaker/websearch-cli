// internal/cli/root_test.go
package cli

import (
	"testing"
)

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
			got, _, err := resolveProvider(tt.profileProv, tt.flagProv, tt.perplexKey, tt.braveKey)
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

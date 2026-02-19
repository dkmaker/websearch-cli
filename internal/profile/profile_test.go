package profile

import (
	"testing"
)

func TestLoadBuiltinProfile(t *testing.T) {
	p, err := Load("general")
	if err != nil {
		t.Fatalf("Load(general) error: %v", err)
	}
	if p.Name != "general" {
		t.Errorf("expected name 'general', got %q", p.Name)
	}
	if p.Provider == "" {
		t.Error("expected provider to be set")
	}
	if p.Mode == "" {
		t.Error("expected default mode to be set")
	}
}

func TestLoadBuiltinNodejs(t *testing.T) {
	p, err := Load("nodejs")
	if err != nil {
		t.Fatalf("Load(nodejs) error: %v", err)
	}
	if p.Name != "nodejs" {
		t.Errorf("expected name 'nodejs', got %q", p.Name)
	}
	if len(p.DomainFilter) == 0 {
		t.Error("expected nodejs profile to have domain filters")
	}
}

func TestLoadUnknownProfile(t *testing.T) {
	_, err := Load("nonexistent")
	if err == nil {
		t.Error("expected error for unknown profile")
	}
}

func TestMergeProfiles(t *testing.T) {
	base := &Profile{
		Name:       "base",
		Provider:   "perplexity",
		Mode:       "ask",
		MaxTokens:  1024,
		MaxResults: 10,
	}
	override := &Profile{
		MaxTokens: 2048,
	}
	merged := Merge(base, override)
	if merged.Provider != "perplexity" {
		t.Errorf("expected provider 'perplexity', got %q", merged.Provider)
	}
	if merged.MaxTokens != 2048 {
		t.Errorf("expected max_tokens 2048, got %d", merged.MaxTokens)
	}
	if merged.MaxResults != 10 {
		t.Errorf("expected max_results 10, got %d", merged.MaxResults)
	}
}

func TestListProfiles(t *testing.T) {
	profiles := ListBuiltin()
	if len(profiles) < 3 {
		t.Errorf("expected at least 3 built-in profiles, got %d", len(profiles))
	}
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p] = true
	}
	for _, want := range []string{"general", "nodejs", "python"} {
		if !names[want] {
			t.Errorf("expected profile %q in list", want)
		}
	}
}

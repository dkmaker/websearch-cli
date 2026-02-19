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

func TestLoadBuiltinGitHub(t *testing.T) {
	p, err := Load("github")
	if err != nil {
		t.Fatalf("Load(github) error: %v", err)
	}
	if p.Name != "github" {
		t.Errorf("expected name 'github', got %q", p.Name)
	}
	if p.Provider != "github" {
		t.Errorf("expected provider 'github', got %q", p.Provider)
	}
	if p.Mode != "repos" {
		t.Errorf("expected mode 'repos', got %q", p.Mode)
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

func TestLoadAllBuiltin(t *testing.T) {
	profiles, err := LoadAllBuiltin()
	if err != nil {
		t.Fatalf("LoadAllBuiltin error: %v", err)
	}
	if len(profiles) < 4 {
		t.Errorf("expected at least 4 profiles, got %d", len(profiles))
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

func TestListProfiles(t *testing.T) {
	profiles := ListBuiltin()
	if len(profiles) < 4 {
		t.Errorf("expected at least 4 built-in profiles, got %d", len(profiles))
	}
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p] = true
	}
	for _, want := range []string{"general", "github", "nodejs", "python"} {
		if !names[want] {
			t.Errorf("expected profile %q in list", want)
		}
	}
}

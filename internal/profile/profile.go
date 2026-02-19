package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile defines search configuration for a specific use case.
type Profile struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Provider     string   `yaml:"provider"`
	Mode         string   `yaml:"mode"`
	SystemPrompt string   `yaml:"system_prompt"`
	DomainFilter []string `yaml:"domain_filter"`
	MaxResults   int      `yaml:"max_results"`
	MaxTokens    int      `yaml:"max_tokens"`
	OutputFormat string   `yaml:"output_format"`
}

// Load loads a profile by name. It checks built-in profiles first,
// then user profiles in ~/.config/websearch/profiles/.
func Load(name string) (*Profile, error) {
	builtin, err := loadBuiltin(name)
	if err == nil {
		userProfile, userErr := loadUser(name)
		if userErr == nil {
			return Merge(builtin, userProfile), nil
		}
		return builtin, nil
	}

	userProfile, userErr := loadUser(name)
	if userErr == nil {
		return userProfile, nil
	}

	return nil, fmt.Errorf("profile %q not found", name)
}

func loadBuiltin(name string) (*Profile, error) {
	data, err := builtinProfiles.ReadFile("profiles/" + name + ".yaml")
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing built-in profile %q: %w", name, err)
	}
	return &p, nil
}

func loadUser(name string) (*Profile, error) {
	configDir := userConfigDir()
	path := filepath.Join(configDir, "websearch", "profiles", name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing user profile %q: %w", name, err)
	}
	return &p, nil
}

func userConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config")
}

// Merge overlays override fields on top of base. Zero-value fields
// in override are ignored.
func Merge(base, override *Profile) *Profile {
	merged := *base
	if override.Name != "" {
		merged.Name = override.Name
	}
	if override.Description != "" {
		merged.Description = override.Description
	}
	if override.Provider != "" {
		merged.Provider = override.Provider
	}
	if override.Mode != "" {
		merged.Mode = override.Mode
	}
	if override.SystemPrompt != "" {
		merged.SystemPrompt = override.SystemPrompt
	}
	if len(override.DomainFilter) > 0 {
		merged.DomainFilter = override.DomainFilter
	}
	if override.MaxResults > 0 {
		merged.MaxResults = override.MaxResults
	}
	if override.MaxTokens > 0 {
		merged.MaxTokens = override.MaxTokens
	}
	if override.OutputFormat != "" {
		merged.OutputFormat = override.OutputFormat
	}
	return &merged
}

// ListBuiltin returns the names of all built-in profiles.
func ListBuiltin() []string {
	entries, err := builtinProfiles.ReadDir("profiles")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	return names
}

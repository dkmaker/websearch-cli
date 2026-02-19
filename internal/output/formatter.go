// internal/output/formatter.go
package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dkmaker/websearch/internal/provider"
)

// FormatMarkdown renders a provider.Result as markdown text.
// When includeSources is false and sources exist, a hint comment is appended.
func FormatMarkdown(result *provider.Result, includeSources bool) string {
	var sb strings.Builder

	sb.WriteString(result.Content)

	if includeSources && len(result.Sources) > 0 {
		sb.WriteString("\n\n---\nSources:\n")
		for _, s := range result.Sources {
			if s.Title != "" {
				sb.WriteString(fmt.Sprintf("- [%s](%s)\n", s.Title, s.URL))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", s.URL))
			}
		}
	} else if !includeSources && len(result.Sources) > 0 {
		sb.WriteString("\n<!-- run with --include-sources for citations -->")
	}

	return sb.String()
}

type jsonOutput struct {
	Content  string            `json:"content"`
	Provider string            `json:"provider"`
	Mode     string            `json:"mode"`
	Cached   bool              `json:"cached"`
	Sources  []provider.Source  `json:"sources,omitempty"`
}

// FormatJSON renders a provider.Result as pretty-printed JSON.
// Sources are only included when includeSources is true.
func FormatJSON(result *provider.Result, includeSources bool) (string, error) {
	out := jsonOutput{
		Content:  result.Content,
		Provider: result.Provider,
		Mode:     result.Mode,
		Cached:   result.Cached,
	}
	if includeSources {
		out.Sources = result.Sources
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

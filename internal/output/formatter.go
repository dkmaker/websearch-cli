// internal/output/formatter.go
package output

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dkmaker/websearch/internal/provider"
)

var citationRefRe = regexp.MustCompile(`\s*\[(\d+(?:,\s*\d+)*)\]`)

// FormatMarkdown renders a provider.Result as markdown text.
// When includeSources is false and sources exist, a hint comment is appended.
func FormatMarkdown(result *provider.Result, includeSources bool) string {
	var sb strings.Builder

	content := result.Content
	if !includeSources {
		content = citationRefRe.ReplaceAllString(content, "")
	}
	sb.WriteString(content)

	if includeSources && len(result.Sources) > 0 {
		sb.WriteString("\n\n---\nSources:\n")
		for _, s := range result.Sources {
			if s.Title != "" {
				sb.WriteString(fmt.Sprintf("- [%s](%s)\n", s.Title, s.URL))
			} else {
				sb.WriteString(fmt.Sprintf("- %s\n", s.URL))
			}
		}
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
	content := result.Content
	if !includeSources {
		content = citationRefRe.ReplaceAllString(content, "")
	}
	out := jsonOutput{
		Content:  content,
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

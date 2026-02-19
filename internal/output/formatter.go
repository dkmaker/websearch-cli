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
// When includeMetadata is true, YAML frontmatter is prepended.
// When includeSources is false and sources exist, a hint comment is appended.
func FormatMarkdown(result *provider.Result, includeSources bool, includeMetadata bool) string {
	var sb strings.Builder

	if includeMetadata {
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("provider: %s\n", result.Provider))
		sb.WriteString(fmt.Sprintf("mode: %s\n", result.Mode))
		sb.WriteString(fmt.Sprintf("mode_adjusted: %v\n", result.ModeAdjusted))
		sb.WriteString(fmt.Sprintf("truncated: %v\n", result.FinishReason == "length"))
		sb.WriteString(fmt.Sprintf("sources_count: %d\n", len(result.Sources)))
		sb.WriteString(fmt.Sprintf("cached: %v\n", result.Cached))
		sb.WriteString("---\n\n")
	}

	content := result.Content
	if !includeSources {
		content = citationRefRe.ReplaceAllString(content, "")
	}
	sb.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		sb.WriteString("\n")
	}

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
	Content      string           `json:"content"`
	Provider     string           `json:"provider"`
	Mode         string           `json:"mode"`
	Cached       bool             `json:"cached"`
	ModeAdjusted bool             `json:"mode_adjusted"`
	Truncated    bool             `json:"truncated"`
	SourcesCount int              `json:"sources_count"`
	Sources      []provider.Source `json:"sources,omitempty"`
}

// FormatJSON renders a provider.Result as pretty-printed JSON.
// Sources are only included when includeSources is true.
// When includeMetadata is true, extra metadata fields are included.
func FormatJSON(result *provider.Result, includeSources bool, includeMetadata bool) (string, error) {
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
	if includeMetadata {
		out.ModeAdjusted = result.ModeAdjusted
		out.Truncated = result.FinishReason == "length"
		out.SourcesCount = len(result.Sources)
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

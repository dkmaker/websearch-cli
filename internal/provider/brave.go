// internal/provider/brave.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultBraveURL = "https://api.search.brave.com"

type Brave struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewBrave(apiKey, baseURL string) *Brave {
	if baseURL == "" {
		baseURL = defaultBraveURL
	}
	return &Brave{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (b *Brave) Name() string {
	return "brave"
}

func (b *Brave) SupportedModes() []string {
	return []string{"web"}
}

type braveResponse struct {
	Web struct {
		Results []braveResult `json:"results"`
	} `json:"web"`
}

type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	PageAge     string `json:"page_age"`
}

func (b *Brave) Search(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("text_format", "markdown")
	if opts.MaxResults > 0 {
		count := opts.MaxResults
		if count > 20 {
			count = 20
		}
		params.Set("count", strconv.Itoa(count))
	}
	if opts.RecencyFilter != "" {
		params.Set("freshness", braveRecency(opts.RecencyFilter))
	}

	endpoint := b.baseURL + "/res/v1/web/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Subscription-Token", b.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("brave API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var bResp braveResponse
	if err := json.Unmarshal(respBody, &bResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	result := &Result{
		Provider: "brave",
		Mode:     "web",
	}

	var sb strings.Builder
	for i, r := range bResp.Web.Results {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		title := html.UnescapeString(r.Title)
		desc := html.UnescapeString(r.Description)
		sb.WriteString(fmt.Sprintf("**%s**\n%s", title, desc))
		result.Sources = append(result.Sources, Source{
			Title: title,
			URL:   r.URL,
			Date:  r.PageAge,
		})
	}
	result.Content = sb.String()

	return result, nil
}

func braveRecency(recency string) string {
	switch recency {
	case "hour", "day":
		return "pd"
	case "week":
		return "pw"
	case "month":
		return "pm"
	case "year":
		return "py"
	default:
		return recency
	}
}

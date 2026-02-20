// internal/provider/github.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultGitHubURL = "https://api.github.com"

type GitHub struct {
	token   string
	baseURL string
	client  *http.Client
}

func NewGitHub(token, baseURL string) *GitHub {
	if baseURL == "" {
		baseURL = defaultGitHubURL
	}
	return &GitHub{
		token:   token,
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (g *GitHub) Name() string {
	return "github"
}

func (g *GitHub) SupportedModes() []string {
	return []string{"repos", "code", "issues"}
}

// GitHub API response types

type githubSearchResponse struct {
	TotalCount        int             `json:"total_count"`
	IncompleteResults bool            `json:"incomplete_results"`
	Items             json.RawMessage `json:"items"`
}

type githubRepo struct {
	FullName        string   `json:"full_name"`
	HTMLURL         string   `json:"html_url"`
	Description     string   `json:"description"`
	StargazersCount int      `json:"stargazers_count"`
	Language        string   `json:"language"`
	UpdatedAt       string   `json:"updated_at"`
	Topics          []string `json:"topics"`
}

type githubCodeResult struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	HTMLURL    string `json:"html_url"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	TextMatches []struct {
		Fragment string `json:"fragment"`
	} `json:"text_matches"`
}

type githubIssue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	State     string `json:"state"`
	Body      string `json:"body"`
	Comments  int    `json:"comments"`
	UpdatedAt string `json:"updated_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	RepositoryURL string `json:"repository_url"`
}

func (g *GitHub) Search(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	query = g.prepareQuery(query, opts.Mode, opts.GitHub)

	switch opts.Mode {
	case "repos":
		return g.searchRepos(ctx, query, opts)
	case "code":
		return g.searchCode(ctx, query, opts)
	case "issues":
		return g.searchIssues(ctx, query, opts)
	default:
		return g.searchRepos(ctx, query, opts)
	}
}

func (g *GitHub) doRequest(ctx context.Context, endpoint string, acceptHeader string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	if acceptHeader != "" {
		req.Header.Set("Accept", acceptHeader)
	} else {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (g *GitHub) buildURL(path string, query string, maxResults int) string {
	params := url.Values{}
	params.Set("q", query)
	perPage := maxResults
	if perPage <= 0 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	params.Set("per_page", strconv.Itoa(perPage))
	return g.baseURL + path + "?" + params.Encode()
}

func (g *GitHub) searchRepos(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	endpoint := g.buildURL("/search/repositories", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var repos []githubRepo
	if err := json.Unmarshal(resp.Items, &repos); err != nil {
		return nil, fmt.Errorf("parsing repos: %w", err)
	}

	if len(repos) == 0 {
		return nil, fmt.Errorf("no results found for query")
	}

	result := &Result{
		Provider: "github",
		Mode:     "repos",
	}

	var sb strings.Builder
	for i, r := range repos {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s** %s — %s", r.FullName, formatStars(r.StargazersCount), r.Description))
		// Metadata line
		var meta []string
		if r.Language != "" {
			meta = append(meta, "Language: "+r.Language)
		}
		if len(r.UpdatedAt) >= 10 {
			meta = append(meta, "Updated: "+r.UpdatedAt[:10])
		}
		if len(r.Topics) > 0 {
			meta = append(meta, "Topics: "+strings.Join(r.Topics, ", "))
		}
		if len(meta) > 0 {
			sb.WriteString("\n  " + strings.Join(meta, " | "))
		}

		result.Sources = append(result.Sources, Source{
			Title: r.FullName,
			URL:   r.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}

func formatStars(count int) string {
	if count >= 1000 {
		return fmt.Sprintf("%dk", count/1000)
	}
	return strconv.Itoa(count)
}

func (g *GitHub) searchCode(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	if g.token == "" {
		return nil, fmt.Errorf("code search requires authentication. Set GITHUB_TOKEN")
	}

	endpoint := g.buildURL("/search/code", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "application/vnd.github.text-match+json")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var items []githubCodeResult
	if err := json.Unmarshal(resp.Items, &items); err != nil {
		return nil, fmt.Errorf("parsing code results: %w", err)
	}

	result := &Result{
		Provider: "github",
		Mode:     "code",
	}

	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s** %s", item.Repository.FullName, item.Path))
		for _, tm := range item.TextMatches {
			if tm.Fragment != "" {
				sb.WriteString("\n  " + strings.ReplaceAll(tm.Fragment, "\n", "\n  "))
				break // only first fragment
			}
		}
		result.Sources = append(result.Sources, Source{
			Title: item.Repository.FullName + "/" + item.Path,
			URL:   item.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}

func (g *GitHub) searchIssues(ctx context.Context, query string, opts SearchOptions) (*Result, error) {
	endpoint := g.buildURL("/search/issues", query, opts.MaxResults)
	body, err := g.doRequest(ctx, endpoint, "")
	if err != nil {
		return nil, err
	}

	var resp githubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var issues []githubIssue
	if err := json.Unmarshal(resp.Items, &issues); err != nil {
		return nil, fmt.Errorf("parsing issues: %w", err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no results found for query")
	}

	result := &Result{
		Provider: "github",
		Mode:     "issues",
	}

	var sb strings.Builder
	for i, issue := range issues {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		// Extract repo name from repository_url: https://api.github.com/repos/owner/repo
		repoName := repoNameFromURL(issue.RepositoryURL)
		sb.WriteString(fmt.Sprintf("**%s#%d** %s (%s)", repoName, issue.Number, issue.Title, issue.State))

		var meta []string
		if len(issue.Labels) > 0 {
			var labelNames []string
			for _, l := range issue.Labels {
				labelNames = append(labelNames, l.Name)
			}
			meta = append(meta, "Labels: "+strings.Join(labelNames, ", "))
		}
		if issue.Comments > 0 {
			meta = append(meta, fmt.Sprintf("Comments: %d", issue.Comments))
		}
		if len(issue.UpdatedAt) >= 10 {
			meta = append(meta, "Updated: "+issue.UpdatedAt[:10])
		}
		if len(meta) > 0 {
			sb.WriteString("\n  " + strings.Join(meta, " | "))
		}

		// Include issue body (first 500 chars) for richer context
		if body := strings.TrimSpace(issue.Body); body != "" {
			if len(body) > 500 {
				body = body[:500] + "..."
			}
			sb.WriteString("\n  " + strings.ReplaceAll(body, "\n", "\n  "))
		}

		result.Sources = append(result.Sources, Source{
			Title: fmt.Sprintf("%s#%d", repoName, issue.Number),
			URL:   issue.HTMLURL,
		})
	}
	result.Content = sb.String()
	return result, nil
}

func repoNameFromURL(apiURL string) string {
	// https://api.github.com/repos/owner/repo -> owner/repo
	const prefix = "/repos/"
	idx := strings.Index(apiURL, prefix)
	if idx >= 0 {
		return apiURL[idx+len(prefix):]
	}
	return ""
}

// prepareQuery builds the GitHub API query string from the free-text query
// and structured qualifiers.
func (g *GitHub) prepareQuery(query string, mode string, qualifiers GitHubQualifiers) string {
	var parts []string
	if query != "" {
		parts = append(parts, query)
	}

	// Common qualifiers (all modes)
	if qualifiers.Language != "" {
		parts = append(parts, "language:"+qualifiers.Language)
	}
	if qualifiers.User != "" {
		parts = append(parts, "user:"+qualifiers.User)
	}
	if qualifiers.Org != "" {
		parts = append(parts, "org:"+qualifiers.Org)
	}
	if qualifiers.Repo != "" {
		parts = append(parts, "repo:"+qualifiers.Repo)
	}
	if qualifiers.In != "" {
		parts = append(parts, "in:"+qualifiers.In)
	}

	// Repos-applicable qualifiers
	if qualifiers.Stars != "" {
		parts = append(parts, "stars:"+qualifiers.Stars)
	}
	if qualifiers.Topic != "" {
		parts = append(parts, "topic:"+qualifiers.Topic)
	}
	if qualifiers.License != "" {
		parts = append(parts, "license:"+qualifiers.License)
	}
	if qualifiers.Archived != "" {
		parts = append(parts, "archived:"+qualifiers.Archived)
	}
	if qualifiers.Fork != "" {
		parts = append(parts, "fork:"+qualifiers.Fork)
	}
	if qualifiers.Pushed != "" {
		parts = append(parts, "pushed:"+qualifiers.Pushed)
	}
	if qualifiers.Created != "" {
		parts = append(parts, "created:"+qualifiers.Created)
	}

	// Code-applicable qualifiers
	if qualifiers.Filename != "" {
		parts = append(parts, "filename:"+qualifiers.Filename)
	}
	if qualifiers.Extension != "" {
		parts = append(parts, "extension:"+qualifiers.Extension)
	}
	if qualifiers.Path != "" {
		parts = append(parts, "path:"+qualifiers.Path)
	}

	// Issues-applicable qualifiers
	if qualifiers.State != "" {
		parts = append(parts, "is:"+qualifiers.State)
	}
	if qualifiers.Label != "" {
		parts = append(parts, "label:"+qualifiers.Label)
	}
	if qualifiers.Author != "" {
		parts = append(parts, "author:"+qualifiers.Author)
	}
	if qualifiers.Assignee != "" {
		parts = append(parts, "assignee:"+qualifiers.Assignee)
	}

	// Issues mode: always append is:issue unless user query already has it
	if mode == "issues" {
		joined := strings.Join(parts, " ")
		if !strings.Contains(joined, "is:issue") && !strings.Contains(joined, "is:pull-request") {
			parts = append(parts, "is:issue")
		}
	}

	return strings.Join(parts, " ")
}

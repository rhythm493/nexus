package websearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DDGClient provides web search via DuckDuckGo HTML search
type DDGClient struct {
	httpClient *http.Client
}

// NewDDGClient creates a new DuckDuckGo search client
func NewDDGClient() *DDGClient {
	return &DDGClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

var (
	resultRx   = regexp.MustCompile(`<a rel="nofollow" class="result__a" href="([^"]+)"[^>]*>(.*?)</a>`)
	snippetRx  = regexp.MustCompile(`<a class="result__snippet"[^>]*>(.*?)</a>`)
	cleanTagRx = regexp.MustCompile(`<[^>]+>`)
	cleanURLRx = regexp.MustCompile(`uddg=([^&]+)`)
)

// Search performs a web search using DuckDuckGo HTML search
func (c *DDGClient) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults == 0 {
		maxResults = 5
	}

	params := url.Values{}
	params.Set("q", query)

	reqURL := "https://html.duckduckgo.com/html/?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	html := string(body)

	titles := resultRx.FindAllStringSubmatch(html, -1)
	snippets := snippetRx.FindAllStringSubmatch(html, -1)

	results := make([]SearchResult, 0, maxResults)
	for i := 0; i < len(titles) && len(results) < maxResults; i++ {
		title := cleanTagRx.ReplaceAllString(titles[i][2], "")

		rawURL := titles[i][1]
		decodedURL := "https://" + strings.TrimPrefix(rawURL, "//")
		if m := cleanURLRx.FindStringSubmatch(rawURL); len(m) > 1 {
			if u, err := url.QueryUnescape(m[1]); err == nil {
				decodedURL = u
			}
		}

		desc := ""
		if i < len(snippets) {
			desc = cleanTagRx.ReplaceAllString(snippets[i][1], "")
		}

		results = append(results, SearchResult{
			Title:       title,
			URL:         decodedURL,
			Description: desc,
		})
	}

	return results, nil
}

// ReadURL fetches a URL and extracts readable text content.
// Returns the page title and body text, capped at maxLen bytes.
func (c *DDGClient) ReadURL(ctx context.Context, urlStr string, maxLen int) (string, string, error) {
	if maxLen == 0 {
		maxLen = 4000
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxLen*2)))
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	html := string(body)

	// Extract title
	titleRx := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	title := ""
	if m := titleRx.FindStringSubmatch(html); len(m) > 1 {
		title = cleanTagRx.ReplaceAllString(m[1], "")
	}

	// Extract body content
	text := extractReadableText(html)

	if len(text) > maxLen {
		text = text[:maxLen] + "..."
	}

	return title, text, nil
}

// extractReadableText strips HTML and extracts text from common content tags.
func extractReadableText(html string) string {
	var parts []string

	// Extract from article, main, and body sections
	sectionRx := regexp.MustCompile(`<article[^>]*>.*?</article>|<main[^>]*>.*?</main>|<body[^>]*>.*?</body>`)
	section := sectionRx.FindString(html)
	if section == "" {
		section = html
	}

	// Extract text from common content tags
	tagRxs := []*regexp.Regexp{
		regexp.MustCompile(`<p[^>]*>(.*?)</p>`),
		regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`),
		regexp.MustCompile(`<h2[^>]*>(.*?)</h2>`),
		regexp.MustCompile(`<h3[^>]*>(.*?)</h3>`),
		regexp.MustCompile(`<h4[^>]*>(.*?)</h4>`),
		regexp.MustCompile(`<h5[^>]*>(.*?)</h5>`),
		regexp.MustCompile(`<h6[^>]*>(.*?)</h6>`),
		regexp.MustCompile(`<li[^>]*>(.*?)</li>`),
		regexp.MustCompile(`<blockquote[^>]*>(.*?)</blockquote>`),
		regexp.MustCompile(`<pre[^>]*>(.*?)</pre>`),
	}

	for _, rx := range tagRxs {
		matches := rx.FindAllStringSubmatch(section, -1)
		for _, m := range matches {
			text := strings.TrimSpace(cleanTagRx.ReplaceAllString(m[1], ""))
			if text != "" {
				parts = append(parts, text)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

// SearchResult represents a unified search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

// ToolDefinition represents a tool definition for MCP
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// ListTools returns MCP tool definitions for web search
func (c *DDGClient) ListTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "web_search",
			Description: "Search the web using DuckDuckGo. Returns relevant results with titles, URLs, and descriptions.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (e.g., 'weather in bangalore', 'python tutorial')",
					},
					"max_results": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of results to return (default: 5)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "web_read",
			Description: "Visit a URL and extract readable text content. Returns the page title and body text. Use this to read full articles, documentation, or any web page.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Full URL to visit (e.g., 'https://en.wikipedia.org/wiki/Paris')",
					},
					"max_length": map[string]interface{}{
						"type":        "number",
						"description": "Maximum characters to return (default: 4000)",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

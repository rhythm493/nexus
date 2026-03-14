package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DDGClient provides web search via DuckDuckGo
type DDGClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewDDGClient creates a new DuckDuckGo search client
func NewDDGClient() *DDGClient {
	return &DDGClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    "https://api.duckduckgo.com/",
	}
}

// Search performs a web search
func (c *DDGClient) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults == 0 {
		maxResults = 5
	}

	// DuckDuckGo Instant Answer API
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")

	reqURL := c.baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	var ddgResp DDGResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to unified format
	results := make([]SearchResult, 0, maxResults)

	// Add instant answer if available
	if ddgResp.Abstract != "" {
		results = append(results, SearchResult{
			Title:       ddgResp.Heading,
			URL:         ddgResp.AbstractURL,
			Description: ddgResp.Abstract,
			Source:      ddgResp.AbstractSource,
		})
	}

	// Add related topics
	for i, topic := range ddgResp.RelatedTopics {
		if i >= maxResults-1 { // Leave room for abstract
			break
		}
		if topic.Text != "" {
			title := topic.Text
			if len(title) > 100 {
				title = title[:100] + "..."
			}
			results = append(results, SearchResult{
				Title:       title,
				URL:         topic.FirstURL,
				Description: topic.Text,
			})
		}
	}

	return results, nil
}

// DDGResponse represents DuckDuckGo API response
type DDGResponse struct {
	Abstract       string         `json:"Abstract"`
	AbstractURL    string         `json:"AbstractURL"`
	AbstractSource string         `json:"AbstractSource"`
	Heading        string         `json:"Heading"`
	RelatedTopics  []RelatedTopic `json:"RelatedTopics"`
}

// RelatedTopic represents a related search result
type RelatedTopic struct {
	FirstURL string `json:"FirstURL"`
	Text     string `json:"Text"`
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
	}
}

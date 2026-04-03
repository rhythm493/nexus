package quickcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Client communicates with QuickCom Node.js server via REST API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new QuickCom HTTP client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:5000"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// HealthResponse from QuickCom /api/health
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Health checks if QuickCom server is reachable
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.get(ctx, "/api/health")
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}
	return &health, nil
}

// SearchRequest for POST /api/search
type SearchRequest struct {
	Query     string   `json:"query"`
	Providers []string `json:"providers,omitempty"`
	Limit     int      `json:"limit,omitempty"`
}

// SearchResponse from POST /api/search
type SearchResponse struct {
	Results      map[string]ProviderResult `json:"results"`
	SearchTimeMs int64                     `json:"searchTimeMs"`
}

// ProviderResult holds results from a single provider
type ProviderResult struct {
	Products     []Product `json:"products"`
	TotalFound   int       `json:"totalFound"`
	SearchTimeMs int64     `json:"searchTimeMs"`
	Cached       bool      `json:"cached"`
	Stale        bool      `json:"stale"`
}

// Product represents a grocery product (matches UnifiedProduct from QuickCom)
type Product struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Brand             *string  `json:"brand"`
	PricePaise        int      `json:"pricePaise"`
	PriceDisplay      string   `json:"priceDisplay"`
	MrpPaise          *int     `json:"mrpPaise"`
	MrpDisplay        *string  `json:"mrpDisplay"`
	Quantity          string   `json:"quantity"`
	QuantityValue     *float64 `json:"quantityValue"`
	QuantityUnit      *string  `json:"quantityUnit"`
	PerUnitPricePaise *int     `json:"perUnitPricePaise"`
	DeliveryTime      string   `json:"deliveryTime"`
	Discount          *string  `json:"discount"`
	DiscountPct       *int     `json:"discountPct"`
	ImageURL          string   `json:"imageUrl"`
	ProductURL        *string  `json:"productUrl"`
	Available         bool     `json:"available"`
	Rating            *float64 `json:"rating,omitempty"`
	TotalRatings      *int     `json:"totalRatings,omitempty"`
	Source            string   `json:"source"`
}

// Search searches for products across providers
func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	resp, err := c.post(ctx, "/api/search", body)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchResp, nil
}

// LocationRequest for POST /api/location
type LocationRequest struct {
	Lat       float64  `json:"lat"`
	Lon       float64  `json:"lon"`
	Label     string   `json:"label,omitempty"`
	Providers []string `json:"providers,omitempty"`
}

// SetLocation sets delivery location on QuickCom providers
func (c *Client) SetLocation(ctx context.Context, lat, lon float64, label string) error {
	body, err := json.Marshal(LocationRequest{
		Lat:   lat,
		Lon:   lon,
		Label: label,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal location request: %w", err)
	}

	resp, err := c.post(ctx, "/api/location", body)
	if err != nil {
		return fmt.Errorf("set location failed: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("QuickCom location set", "lat", lat, "lon", lon, "label", label)
	return nil
}

// ToolDefinition represents a tool definition for MCP
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// ListTools returns available QuickCom tools (for MCP integration)
func (c *Client) ListTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "grocery_search",
			Description: "Search for grocery products across Zepto, Blinkit, and Instamart. Returns product names, prices, and availability.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Product name to search for (e.g., 'milk', 'eggs', 'bread')",
					},
					"services": map[string]interface{}{
						"type":        "array",
						"description": "Which services to search (default: all)",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"blinkit", "zepto", "instamart"},
						},
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "compare_prices",
			Description: "Compare prices for a specific product across all grocery services",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"product_name": map[string]interface{}{
						"type":        "string",
						"description": "Product name to compare",
					},
				},
				"required": []string{"product_name"},
			},
		},
		{
			Name:        "location_set",
			Description: "Set delivery location using latitude and longitude coordinates",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"latitude": map[string]interface{}{
						"type":        "number",
						"description": "Latitude coordinate",
					},
					"longitude": map[string]interface{}{
						"type":        "number",
						"description": "Longitude coordinate",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable location name",
					},
				},
				"required": []string{"latitude", "longitude", "location"},
			},
		},
	}
}

// --- HTTP helpers ---

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return resp, nil
}

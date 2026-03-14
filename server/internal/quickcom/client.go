package quickcom

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

// Client communicates with QuickCom Node.js server via WebSocket
type Client struct {
	wsConn *websocket.Conn
	wsURL  string
}

// NewClient creates a new QuickCom WebSocket client
func NewClient(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = "ws://localhost:5000"
	}

	return &Client{
		wsURL: baseURL,
	}, nil
}

// Connect establishes WebSocket connection to QuickCom
func (c *Client) Connect(ctx context.Context) error {
	var err error
	c.wsConn, _, err = websocket.DefaultDialer.DialContext(ctx, c.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to QuickCom: %w", err)
	}

	slog.Info("Connected to QuickCom server", "url", c.wsURL)

	// Wait for "connected" message
	var msg map[string]interface{}
	if err := c.wsConn.ReadJSON(&msg); err != nil {
		return fmt.Errorf("failed to read connection message: %w", err)
	}

	if msgType, ok := msg["type"].(string); !ok || msgType != "connected" {
		return fmt.Errorf("unexpected connection message: %v", msg)
	}

	return nil
}

// SetLocation sets delivery location for all services
func (c *Client) SetLocation(ctx context.Context, lat, lon float64, location string) error {
	services := []string{"blinkit", "zepto", "instamart"}

	for _, svc := range services {
		msg := map[string]interface{}{
			"type":     "set-location",
			"service":  svc,
			"location": location,
		}

		if err := c.wsConn.WriteJSON(msg); err != nil {
			return fmt.Errorf("failed to set location for %s: %w", svc, err)
		}

		// Wait for confirmation
		var resp map[string]interface{}
		if err := c.wsConn.ReadJSON(&resp); err != nil {
			return fmt.Errorf("failed to read location confirmation: %w", err)
		}
	}

	slog.Info("Location set for all services", "lat", lat, "lon", lon)
	return nil
}

// SearchGrocery searches for products across services
func (c *Client) SearchGrocery(ctx context.Context, query string, services []string) ([]Product, error) {
	if len(services) == 0 {
		services = []string{"blinkit", "zepto", "instamart"}
	}

	results := make([]Product, 0)

	for _, svc := range services {
		msg := map[string]interface{}{
			"type":    "search",
			"service": svc,
			"query":   query,
		}

		if err := c.wsConn.WriteJSON(msg); err != nil {
			slog.Error("Failed to send search to service", "service", svc, "error", err)
			continue
		}

		// Read response with timeout
		c.wsConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		var resp map[string]interface{}
		if err := c.wsConn.ReadJSON(&resp); err != nil {
			slog.Error("Failed to read search response", "service", svc, "error", err)
			continue
		}
		c.wsConn.SetReadDeadline(time.Time{}) // Clear deadline

		if respType, ok := resp["type"].(string); ok && respType == "search-results" {
			if products, ok := resp["products"].([]interface{}); ok {
				for _, p := range products {
					if productMap, ok := p.(map[string]interface{}); ok {
						results = append(results, Product{
							ID:            getString(productMap, "id"),
							Name:          getString(productMap, "name"),
							Price:         getString(productMap, "price"),
							OriginalPrice: getStringPtr(productMap, "originalPrice"),
							Quantity:      getString(productMap, "quantity"),
							DeliveryTime:  getString(productMap, "deliveryTime"),
							Discount:      getStringPtr(productMap, "discount"),
							ImageURL:      getString(productMap, "imageUrl"),
							Available:     getBool(productMap, "available"),
							Source:        getString(productMap, "source"),
						})
					}
				}
			}
		}
	}

	return results, nil
}

// Product represents a grocery product
type Product struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Price         string  `json:"price"`
	OriginalPrice *string `json:"original_price,omitempty"`
	Quantity      string  `json:"quantity"`
	DeliveryTime  string  `json:"delivery_time"`
	Discount      *string `json:"discount,omitempty"`
	ImageURL      string  `json:"image_url"`
	Available     bool    `json:"available"`
	Source        string  `json:"source"`
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringPtr(m map[string]interface{}, key string) *string {
	if v, ok := m[key].(string); ok && v != "" {
		return &v
	}
	return nil
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// Close closes the WebSocket connection
func (c *Client) Close() error {
	if c.wsConn != nil {
		return c.wsConn.Close()
	}
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

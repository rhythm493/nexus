package quickcom

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

// MCPBridge adapts QuickCom HTTP client to MCP tool interface
type MCPBridge struct {
	client *Client
}

// NewMCPBridge creates a new MCP bridge for QuickCom
func NewMCPBridge(quickcomURL string) (*MCPBridge, error) {
	client := NewClient(quickcomURL)
	return &MCPBridge{client: client}, nil
}

// Client returns the underlying QuickCom HTTP client.
func (b *MCPBridge) Client() *Client { return b.client }

// Initialize verifies QuickCom server is reachable
func (b *MCPBridge) Initialize(ctx context.Context) error {
	health, err := b.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("QuickCom health check failed: %w", err)
	}
	slog.Info("QuickCom server reachable", "status", health.Status)
	return nil
}

// ListTools returns MCP-compatible tool definitions
func (b *MCPBridge) ListTools() []mcp.Tool {
	defs := b.client.ListTools()
	tools := make([]mcp.Tool, len(defs))

	for i, def := range defs {
		tools[i] = mcp.Tool{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.Parameters,
		}
	}

	return tools
}

// ExecuteTool executes a tool via QuickCom HTTP API
func (b *MCPBridge) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "grocery_search":
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query parameter required")
		}

		var providers []string
		if svcList, ok := args["services"].([]interface{}); ok {
			for _, svc := range svcList {
				if s, ok := svc.(string); ok {
					providers = append(providers, s)
				}
			}
		}

		resp, err := b.client.Search(ctx, SearchRequest{
			Query:     query,
			Providers: providers,
			Limit:     20,
		})
		if err != nil {
			return nil, err
		}
		return resp, nil

	case "compare_prices":
		productName, _ := args["product_name"].(string)
		if productName == "" {
			return nil, fmt.Errorf("product_name parameter required")
		}

		resp, err := b.client.Search(ctx, SearchRequest{
			Query: productName,
			Limit: 20,
		})
		if err != nil {
			return nil, err
		}
		return resp, nil

	case "location_set":
		lat, ok1 := args["latitude"].(float64)
		lon, ok2 := args["longitude"].(float64)
		location, _ := args["location"].(string)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("latitude and longitude must be numbers")
		}

		if err := b.client.SetLocation(ctx, lat, lon, location); err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Location set to %s (%.6f, %.6f)", location, lat, lon),
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// Close is a no-op for HTTP client (no persistent connection)
func (b *MCPBridge) Close() error {
	return nil
}

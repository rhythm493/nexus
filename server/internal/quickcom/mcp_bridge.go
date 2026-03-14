package quickcom

import (
	"context"
	"fmt"

	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

// MCPBridge adapts QuickCom client to MCP tool interface
type MCPBridge struct {
	client *Client
}

// NewMCPBridge creates a new MCP bridge
func NewMCPBridge(quickcomURL string) (*MCPBridge, error) {
	client, err := NewClient(quickcomURL)
	if err != nil {
		return nil, err
	}

	return &MCPBridge{client: client}, nil
}

// Initialize connects to QuickCom server
func (b *MCPBridge) Initialize(ctx context.Context) error {
	return b.client.Connect(ctx)
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

// ExecuteTool executes a tool via QuickCom client
func (b *MCPBridge) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "grocery_search":
		query, _ := args["query"].(string)
		if query == "" {
			return nil, fmt.Errorf("query parameter required")
		}

		services := []string{}
		if svcList, ok := args["services"].([]interface{}); ok {
			for _, svc := range svcList {
				if s, ok := svc.(string); ok {
					services = append(services, s)
				}
			}
		}
		return b.client.SearchGrocery(ctx, query, services)

	case "compare_prices":
		// Alias for grocery_search
		productName, _ := args["product_name"].(string)
		if productName == "" {
			return nil, fmt.Errorf("product_name parameter required")
		}
		return b.client.SearchGrocery(ctx, productName, []string{})

	case "location_set":
		lat, ok1 := args["latitude"].(float64)
		lon, ok2 := args["longitude"].(float64)
		location, _ := args["location"].(string)

		if !ok1 || !ok2 {
			return nil, fmt.Errorf("latitude and longitude must be numbers")
		}

		err := b.client.SetLocation(ctx, lat, lon, location)
		if err != nil {
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

// Close closes the QuickCom connection
func (b *MCPBridge) Close() error {
	return b.client.Close()
}

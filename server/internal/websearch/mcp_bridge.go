package websearch

import (
	"context"
	"fmt"

	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

// MCPBridge adapts web search client to MCP tool interface
type MCPBridge struct {
	client *DDGClient
}

// NewMCPBridge creates a new web search MCP bridge
func NewMCPBridge() *MCPBridge {
	return &MCPBridge{
		client: NewDDGClient(),
	}
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

// ExecuteTool executes a web search tool
func (b *MCPBridge) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	if toolName != "web_search" {
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter required")
	}

	maxResults := 5
	if mr, ok := args["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	return b.client.Search(ctx, query, maxResults)
}

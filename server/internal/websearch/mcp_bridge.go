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
	switch toolName {
	case "web_search":
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return nil, fmt.Errorf("query parameter required")
		}

		maxResults := 5
		if mr, ok := args["max_results"].(float64); ok {
			maxResults = int(mr)
		}

		return b.client.Search(ctx, query, maxResults)

	case "web_read":
		urlStr, ok := args["url"].(string)
		if !ok || urlStr == "" {
			return nil, fmt.Errorf("url parameter required")
		}

		maxLen := 4000
		if ml, ok := args["max_length"].(float64); ok {
			maxLen = int(ml)
		}

		title, text, err := b.client.ReadURL(ctx, urlStr, maxLen)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"title": title,
			"text":  text,
			"url":   urlStr,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

package mcp

import (
	"encoding/json"
)

// JSON-RPC 2.0 message types for MCP protocol

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolsListResult is the result of tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ToolCallParams are parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult is the result of tools/call
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents content in a tool result
type ContentItem struct {
	Type string `json:"type"` // "text", "image", etc.
	Text string `json:"text,omitempty"`
}

// InitializeParams are parameters for initialize
type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ClientCaps `json:"capabilities"`
	ClientInfo      ClientInfo `json:"clientInfo"`
}

// ClientCaps represents client capabilities
type ClientCaps struct {
	Roots    *RootsCaps    `json:"roots,omitempty"`
	Sampling *SamplingCaps `json:"sampling,omitempty"`
}

// RootsCaps represents roots capabilities
type RootsCaps struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCaps represents sampling capabilities
type SamplingCaps struct{}

// ClientInfo represents client information
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the result of initialize
type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    ServerCaps `json:"capabilities"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

// ServerCaps represents server capabilities
type ServerCaps struct {
	Tools     *ToolsCaps     `json:"tools,omitempty"`
	Resources *ResourcesCaps `json:"resources,omitempty"`
	Prompts   *PromptsCaps   `json:"prompts,omitempty"`
}

// ToolsCaps represents tools capabilities
type ToolsCaps struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCaps represents resources capabilities
type ResourcesCaps struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCaps represents prompts capabilities
type PromptsCaps struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo represents server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewRequest creates a new JSON-RPC request
func NewRequest(id int, method string, params interface{}) *Request {
	return &Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ServerConfig represents an MCP server configuration
type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// Server represents a running MCP server
type Server struct {
	Config    ServerConfig
	Transport *StdioTransport
	Tools     []Tool
}

// Host manages multiple MCP servers
type Host struct {
	configDir string
	servers   map[string]*Server
	mu        sync.RWMutex
}

// NewHost creates a new MCP host
func NewHost(configDir string) (*Host, error) {
	return &Host{
		configDir: configDir,
		servers:   make(map[string]*Server),
	}, nil
}

// StartAll discovers and starts all configured MCP servers
func (h *Host) StartAll(ctx context.Context) error {
	// Find all server configs
	configs, err := h.discoverConfigs()
	if err != nil {
		return fmt.Errorf("failed to discover MCP configs: %w", err)
	}

	if len(configs) == 0 {
		slog.Info("No MCP server configs found", "dir", h.configDir)
		return nil
	}

	// Start each server
	for _, cfg := range configs {
		if err := h.startServer(ctx, cfg); err != nil {
			slog.Error("Failed to start MCP server", "name", cfg.Name, "error", err)
			// Continue with other servers
			continue
		}
	}

	return nil
}

// discoverConfigs finds all MCP server config files
func (h *Host) discoverConfigs() ([]ServerConfig, error) {
	var configs []ServerConfig

	entries, err := os.ReadDir(h.configDir)
	if os.IsNotExist(err) {
		return configs, nil
	}
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(h.configDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("Failed to read MCP config", "path", path, "error", err)
			continue
		}

		var cfg ServerConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Warn("Failed to parse MCP config", "path", path, "error", err)
			continue
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

// startServer starts an individual MCP server
func (h *Host) startServer(ctx context.Context, cfg ServerConfig) error {
	slog.Info("Starting MCP server", "name", cfg.Name, "command", cfg.Command)

	// Convert env map to slice
	var env []string
	for k, v := range cfg.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create transport
	transport, err := NewStdioTransport(ctx, cfg.Command, cfg.Args, env)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	server := &Server{
		Config:    cfg,
		Transport: transport,
	}

	// Initialize the server
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	initParams := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: ClientCaps{
			Roots: &RootsCaps{ListChanged: true},
		},
		ClientInfo: ClientInfo{
			Name:    "pocket-assistant",
			Version: "1.0.0",
		},
	}

	resp, err := transport.Send(initCtx, "initialize", initParams)
	if err != nil {
		transport.Close()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	var initResult InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		transport.Close()
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	slog.Info("MCP server initialized",
		"name", cfg.Name,
		"serverName", initResult.ServerInfo.Name,
		"version", initResult.ServerInfo.Version,
	)

	// Send initialized notification (no response expected)
	if err := transport.Notify("notifications/initialized", nil); err != nil {
		slog.Warn("Failed to send initialized notification", "error", err)
	}

	// Discover tools
	tools, err := h.discoverTools(initCtx, transport)
	if err != nil {
		slog.Warn("Failed to discover tools", "name", cfg.Name, "error", err)
	} else {
		server.Tools = tools
		slog.Info("Discovered MCP tools", "name", cfg.Name, "count", len(tools))
	}

	h.mu.Lock()
	h.servers[cfg.Name] = server
	h.mu.Unlock()

	return nil
}

// discoverTools retrieves available tools from an MCP server
func (h *Host) discoverTools(ctx context.Context, transport *StdioTransport) ([]Tool, error) {
	resp, err := transport.Send(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tools list: %w", err)
	}

	return result.Tools, nil
}

// ListServers returns the names of running MCP servers
func (h *Host) ListServers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var names []string
	for name := range h.servers {
		names = append(names, name)
	}
	return names
}

// ListTools returns all available tools from all servers
func (h *Host) ListTools() []Tool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var tools []Tool
	for _, server := range h.servers {
		tools = append(tools, server.Tools...)
	}
	return tools
}

// ExecuteTool executes a tool on the appropriate MCP server
func (h *Host) ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	h.mu.RLock()

	// Find server that has this tool
	var server *Server
	for _, s := range h.servers {
		for _, t := range s.Tools {
			if t.Name == name {
				server = s
				break
			}
		}
		if server != nil {
			break
		}
	}
	h.mu.RUnlock()

	if server == nil {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	// Call the tool
	params := ToolCallParams{
		Name:      name,
		Arguments: args,
	}

	resp, err := server.Transport.Send(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("tool error: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("tool returned error")
	}

	// Extract text content
	var texts []string
	for _, content := range result.Content {
		if content.Type == "text" {
			texts = append(texts, content.Text)
		}
	}

	if len(texts) == 1 {
		// Try to parse as JSON
		var jsonResult interface{}
		if err := json.Unmarshal([]byte(texts[0]), &jsonResult); err == nil {
			return jsonResult, nil
		}
		return texts[0], nil
	}

	return texts, nil
}

// StopAll stops all running MCP servers
func (h *Host) StopAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, server := range h.servers {
		slog.Info("Stopping MCP server", "name", name)
		server.Transport.Close()
	}
	h.servers = make(map[string]*Server)
}

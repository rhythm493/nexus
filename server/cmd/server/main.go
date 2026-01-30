package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rhythm493/pocket-assistant/server/config"
	"github.com/rhythm493/pocket-assistant/server/internal/api"
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
	"github.com/rhythm493/pocket-assistant/server/internal/mdns"
	"github.com/rhythm493/pocket-assistant/server/internal/tls"
)

func main() {
	// Setup logging
	logLevel := os.Getenv("LOG_LEVEL")
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize LLM client (Groq)
	llmClient, err := llm.NewClient(cfg.LLMAPIKey)
	if err != nil {
		slog.Error("Failed to create LLM client", "error", err)
		os.Exit(1)
	}

	// Initialize MCP host
	mcpHost, err := mcp.NewHost(cfg.MCPServersDir)
	if err != nil {
		slog.Error("Failed to create MCP host", "error", err)
		os.Exit(1)
	}

	// Start MCP servers
	if err := mcpHost.StartAll(ctx); err != nil {
		slog.Error("Failed to start MCP servers", "error", err)
		os.Exit(1)
	}
	defer mcpHost.StopAll()

	// Load TLS configuration
	tlsConfig, err := tls.LoadServerConfig(cfg.CertsDir)
	if err != nil {
		slog.Error("Failed to load TLS config", "error", err)
		os.Exit(1)
	}

	// Create API server
	server := api.NewServer(cfg, llmClient, mcpHost, tlsConfig)

	// Start mDNS advertisement
	mdnsServer, err := mdns.Advertise(cfg.ServiceName, cfg.Port)
	if err != nil {
		slog.Error("Failed to start mDNS", "error", err)
		os.Exit(1)
	}
	defer mdnsServer.Shutdown()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()
		server.Shutdown(ctx)
	}()

	// Start server
	slog.Info("Starting pocket-assistant server",
		"port", cfg.Port,
		"service", cfg.ServiceName,
	)

	if err := server.Start(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

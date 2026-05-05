package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rhythm493/pocket-assistant/server/config"
	"github.com/rhythm493/pocket-assistant/server/internal/api"
	"github.com/rhythm493/pocket-assistant/server/internal/cart"
	"github.com/rhythm493/pocket-assistant/server/internal/discovery"
	"github.com/rhythm493/pocket-assistant/server/internal/library"
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
	"github.com/rhythm493/pocket-assistant/server/internal/mdns"
	"github.com/rhythm493/pocket-assistant/server/internal/mode"
	"github.com/rhythm493/pocket-assistant/server/internal/quickcom"
	"github.com/rhythm493/pocket-assistant/server/internal/radio"
	"github.com/rhythm493/pocket-assistant/server/internal/websearch"
	"github.com/rhythm493/pocket-assistant/server/internal/youtube"
)

func main() {
	// Load .env files (last file takes precedence, shell env vars still win)
	godotenv.Overload("../.env", ".env")

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

	// Initialize LLM provider
	factory := llm.NewProviderFactory()
	var slots []llm.SlotConfig
	for _, s := range cfg.LLM.Slots {
		slots = append(slots, llm.SlotConfig{Provider: s.Provider, Model: s.Model})
	}
	llmProvider, err := factory.CreateProvider(llm.ProviderConfig{
		Provider: cfg.LLM.Provider,
		Model:    cfg.LLM.Model,
		APIKey:   cfg.LLM.APIKey,
		BaseURL:  cfg.LLM.BaseURL,
		Slots:    slots,
	})
	if err != nil {
		slog.Error("Failed to create LLM provider", "error", err)
		os.Exit(1)
	}
	slog.Info("LLM provider initialized",
		"provider", llmProvider.Name(),
		"model", llmProvider.GetModel())

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

	// Initialize QuickCom client (if grocery mode enabled)
	var quickcomBridge *quickcom.MCPBridge
	hasGroceryMode := false
	for _, mode := range cfg.Modes {
		if mode.ID == "grocery" {
			hasGroceryMode = true
			break
		}
	}

	if hasGroceryMode {
		quickcomURL := os.Getenv("QUICKCOM_URL")
		if quickcomURL == "" {
			quickcomURL = "http://localhost:5000"
		}
		quickcomBridge, err = quickcom.NewMCPBridge(quickcomURL)
		if err != nil {
			slog.Warn("QuickCom not available", "error", err)
		} else {
			if err := quickcomBridge.Initialize(ctx); err != nil {
				slog.Warn("Failed to connect to QuickCom (make sure QuickCom server is running)", "error", err)
				quickcomBridge = nil
			} else {
				slog.Info("QuickCom client initialized")
				// Register QuickCom tools with MCP host
				mcpHost.RegisterExternalTools("quickcom", quickcomBridge)
			}
		}
	}

	// Initialize web search client (if websearch mode enabled)
	hasWebSearchMode := false
	for _, mode := range cfg.Modes {
		if mode.ID == "websearch" {
			hasWebSearchMode = true
			break
		}
	}

	if hasWebSearchMode {
		websearchBridge := websearch.NewMCPBridge()
		mcpHost.RegisterExternalTools("websearch", websearchBridge)
		slog.Info("Web search client initialized")
	}

	// Initialize library
	var lib *library.Library
	lib, err = library.New(cfg.Library.DatabasePath)
	if err != nil {
		slog.Warn("Library not available", "error", err)
		lib = nil
	} else {
		defer lib.Close()
	}

	// Initialize YouTube service (optional)
	var ytService *youtube.Service
	if cfg.YouTube.Enabled {
		// Use library data dir for downloads if available
		outputDir := cfg.YouTube.OutputDir
		if lib != nil && cfg.Library.DataDir != "" {
			outputDir = cfg.Library.DataDir
		}

		ytService, err = youtube.NewService(youtube.Config{
			BinDir:    cfg.YouTube.BinDir,
			OutputDir: outputDir,
		})
		if err != nil {
			slog.Warn("YouTube service not available", "error", err)
			ytService = nil
		}
	}

	// Initialize radio engine (optional)
	var radioEngine *radio.Engine
	var radioTools *radio.ToolHandlers
	if cfg.Radio.Enabled && lib != nil && ytService != nil {
		radioEngine = radio.New(radio.Config{
			LiquidsoapPath: cfg.Radio.LiquidsoapPath,
			ScriptPath:     cfg.Radio.ScriptPath,
			TelnetHost:     cfg.Radio.TelnetHost,
			TelnetPort:     cfg.Radio.TelnetPort,
			StreamPort:     cfg.Radio.StreamPort,
			StreamHost:     cfg.Radio.StreamHost,
			CrossfadeSecs:  cfg.Radio.CrossfadeSecs,
			AutoStart:      cfg.Radio.AutoStart,
		}, lib, ytService)

		// Create radio tool handlers
		radioTools = radio.NewToolHandlers(radioEngine, lib, ytService, mcpHost)

		// Start radio if auto-start is enabled
		if cfg.Radio.AutoStart {
			if err := radioEngine.Start(ctx); err != nil {
				slog.Warn("Failed to start radio engine (Liquidsoap may not be available)", "error", err)
				// Don't fail startup, just disable radio tools that need the engine running
			} else {
				defer radioEngine.Stop()
				slog.Info("Radio engine started",
					"stream_url", radioEngine.StreamURL(),
					"telnet", fmt.Sprintf("%s:%d", cfg.Radio.TelnetHost, cfg.Radio.TelnetPort),
				)
			}
		}
	} else {
		if !cfg.Radio.Enabled {
			slog.Info("Radio disabled in configuration")
		} else if lib == nil {
			slog.Warn("Radio disabled: library not available")
		} else if ytService == nil {
			slog.Warn("Radio disabled: YouTube service not available")
		}
	}

	// Initialize mode manager
	modeManager, err := mode.NewManager(cfg.Modes)
	if err != nil {
		slog.Error("Failed to create mode manager", "error", err)
		os.Exit(1)
	}
	slog.Info("Mode manager initialized", "modes", len(cfg.Modes))

	// Initialize cart system (uses QuickCom for search)
	var cartManager *cart.Manager
	var cartTools *cart.ToolHandlers
	if quickcomBridge != nil {
		cartManager = cart.NewManager()
		cartTools = cart.NewToolHandlers(cartManager, quickcomBridge.Client())
		slog.Info("Cart system initialized")
	}

	// Create API server
	// Get QuickCom client for direct search proxy
	var quickcomClient *quickcom.Client
	if quickcomBridge != nil {
		quickcomClient = quickcomBridge.Client()
	}

	server := api.NewServer(cfg, llmProvider, mcpHost, modeManager, ytService, radioEngine, radioTools, cartManager, cartTools, quickcomClient)

	// Start mDNS advertisement
	mdnsServer, err := mdns.Advertise(cfg.ServiceName, cfg.Port)
	if err != nil {
		slog.Error("Failed to start mDNS", "error", err)
		os.Exit(1)
	}
	defer mdnsServer.Shutdown()

	// Start UDP broadcast discovery (stops when ctx is cancelled)
	discovery.StartUDPBroadcast(ctx, cfg.Port)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Shutdown error, forcing exit", "error", err)
			os.Exit(1)
		}
	}()

	// Start server
	slog.Info("Starting Nexus server",
		"port", cfg.Port,
		"service", cfg.ServiceName,
	)

	if err := server.Start(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the server
type Config struct {
	// Server settings
	Port        int    `yaml:"port"`
	ServiceName string `yaml:"service_name"`

	// Paths
	CertsDir      string `yaml:"certs_dir"`
	MCPServersDir string `yaml:"mcp_servers_dir"`

	// API Keys (deprecated - use LLM.APIKey instead)
	LLMAPIKey string `yaml:"llm_api_key"`

	// LLM Configuration
	LLM LLMConfig `yaml:"llm"`

	// Logging
	LogLevel string `yaml:"log_level"`

	// YouTube settings
	YouTube YouTubeConfig `yaml:"youtube"`

	// Radio settings
	Radio RadioConfig `yaml:"radio"`

	// Library settings
	Library LibraryConfig `yaml:"library"`

	// Assistant Modes
	Modes []ModeConfig `yaml:"modes"`

	// Additional API keys (set via env vars)
	CerebrasAPIKey string `yaml:"-"`
}

// YouTubeConfig holds YouTube-specific configuration
type YouTubeConfig struct {
	Enabled   bool   `yaml:"enabled"`
	BinDir    string `yaml:"bin_dir"`    // Directory for yt-dlp binary (default: data/bin)
	OutputDir string `yaml:"output_dir"` // Directory for downloaded audio files (default: data/audio)
}

// RadioConfig holds radio-specific configuration
type RadioConfig struct {
	Enabled        bool    `yaml:"enabled"`
	LiquidsoapPath string  `yaml:"liquidsoap_path"` // Path to liquidsoap binary
	ScriptPath     string  `yaml:"script_path"`     // Path to radio.liq script
	TelnetHost     string  `yaml:"telnet_host"`     // Telnet control host (default: 127.0.0.1)
	TelnetPort     int     `yaml:"telnet_port"`     // Telnet control port (default: 1234)
	StreamPort     int     `yaml:"stream_port"`     // HTTP stream port (default: 8080)
	StreamHost     string  `yaml:"stream_host"`     // External hostname for stream URL (auto-detect if empty)
	CrossfadeSecs  float64 `yaml:"crossfade_secs"`  // Crossfade duration in seconds
	AutoStart      bool    `yaml:"auto_start"`      // Start Liquidsoap automatically
}

// LibraryConfig holds library-specific configuration
type LibraryConfig struct {
	DataDir      string `yaml:"data_dir"`      // Directory for downloaded audio files
	DatabasePath string `yaml:"database_path"` // Path to SQLite database
	MaxSizeMB    int    `yaml:"max_size_mb"`   // Maximum library size in MB
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider string         `yaml:"provider"` // "groq", "cerebras", "ollama", "lmstudio", "rotating"
	Model    string         `yaml:"model"`
	APIKey   string         `yaml:"api_key"`
	BaseURL  string         `yaml:"base_url"`
	Slots    []LLMSlotConfig `yaml:"slots,omitempty"`
}

// LLMSlotConfig defines a provider+model pair for the rotating provider
type LLMSlotConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
}

// ModeConfig holds mode-specific configuration
type ModeConfig struct {
	ID           string   `yaml:"id"`            // "grocery", "sonos", "websearch"
	Name         string   `yaml:"name"`          // "Grocery Shopping"
	Icon         string   `yaml:"icon"`          // "shopping_cart" (Material icon name)
	Description  string   `yaml:"description"`   // Brief description
	ToolFilters  []string `yaml:"tool_filters"`  // ["grocery_*", "compare_*"]
	SystemPrompt string   `yaml:"system_prompt"` // Mode-specific prompt template
	DefaultMode  bool     `yaml:"default_mode"`  // Is this the default?
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:          8443,
		ServiceName:   "nexus",
		CertsDir:      "../certs",
		MCPServersDir: "./mcp-servers",
		LogLevel:      "info",
		YouTube: YouTubeConfig{
			Enabled:   true,
			BinDir:    "data/bin",
			OutputDir: "data/audio",
		},
		Radio: RadioConfig{
			Enabled:        true,
			LiquidsoapPath: "liquidsoap",
			ScriptPath:     "scripts/radio.liq",
			TelnetHost:     "127.0.0.1",
			TelnetPort:     1234,
			StreamPort:     8080,
			StreamHost:     "",
			CrossfadeSecs:  3.0,
			AutoStart:      true,
		},
		Library: LibraryConfig{
			DataDir:      "data/library",
			DatabasePath: "data/library.db",
			MaxSizeMB:    10240,
		},
		LLM: LLMConfig{
			Provider: "rotating",
			Model:    "gemma-4-26b-a4b-it",
			APIKey:   "",
			BaseURL:  "",
		},
		Modes: []ModeConfig{
			{
				ID:          "sonos",
				Name:        "Music & Speakers",
				Icon:        "speaker",
				Description: "Control Sonos speakers and play music",
				ToolFilters: []string{
					// Playback
					"sonos_play", "sonos_pause", "sonos_stop", "sonos_next", "sonos_previous",
					// Volume
					"sonos_set_volume", "sonos_get_volume", "sonos_set_mute",
					// Info
					"sonos_list_devices", "sonos_get_transport_info", "sonos_get_position_info",
					// Queue
					"sonos_get_queue", "sonos_add_to_queue", "sonos_clear_queue",
					// Groups
					"sonos_get_zone_groups", "sonos_join_group", "sonos_party_mode",
					// Stream
					"sonos_play_url",
					// Radio + library + youtube
					"radio_*", "library_*", "youtube_*",
				},
				SystemPrompt: "You are a music and speaker control assistant for Sonos speakers and a personal radio stream.\n\n## How to play music\n- Use radio_play(query) — this is all you need. It searches the library, downloads from YouTube if needed, streams via Liquidsoap, and auto-connects to Sonos. One call = music plays.\n- You can optionally pass device=\"Speaker Name\" to target a specific speaker. If omitted, it auto-discovers.\n- Use radio_queue(query) to add songs to the queue without interrupting current playback.\n- Use radio_skip() to skip to the next song.\n- Use radio_status() to see what's playing and what's queued.\n\n## Volume and speaker control\n- For volume/mute/playback control, use sonos_set_volume, sonos_get_volume, sonos_play, sonos_pause etc.\n- ALWAYS call sonos_list_devices first if you need a device name for these tools.\n\n## Rules\n- Before calling any tool, briefly explain your reasoning\n- For multi-step requests, outline your plan first\n- If a tool fails, explain what went wrong and try an alternative\n- Keep responses conversational and concise\n- Don't call the same tool with the same arguments twice in a row",
				DefaultMode:  true,
			},
			{
				ID:          "grocery",
				Name:        "Grocery Shopping",
				Icon:        "shopping_cart",
				Description: "Build optimized grocery carts across Zepto, Blinkit, Instamart",
				ToolFilters: []string{"cart_*", "price_check", "find_deals"},
				SystemPrompt: "You are a grocery shopping assistant with a virtual cart system.\n\n## Cart workflow\n1. When the user asks to buy groceries, use cart_add with ALL items in one call\n   - Write SPECIFIC search queries: include product type, variant, and practical quantity\n   - Example: \"make butter chicken\" → cart_add(items: [\"boneless chicken breast 500g\", \"amul butter 100g\", \"fresh cream 200ml\", \"onion 500g\", \"tomato 500g\", \"ginger garlic paste\", \"kashmiri red chilli powder\", \"garam masala\"])\n   - BAD: \"butter\", \"chicken\", \"cream\" (too generic — matches wrong products)\n   - GOOD: \"amul butter 100g\", \"chicken breast 500g\", \"amul fresh cream 200ml\"\n2. After adding, REVIEW the matched product names in the result\n3. If a match is wrong (e.g. \"peanut butter\" instead of cooking butter), remove it and re-add with a more specific query\n4. Once satisfied, use cart_optimize to find the cheapest strategy\n5. Present the result: single-provider vs split, savings amount\n6. Wait for user approval\n\n## Tools\n- cart_add(items): Add multiple items — each string is a search query, be specific with type/brand/size\n- cart_remove(item): Remove a specific item\n- cart_view(): Show current cart with per-provider prices\n- cart_optimize(): Find cheapest ordering strategy (single vs split)\n- cart_clear(): Reset cart\n- price_check(query): Quick price lookup without touching the cart\n- find_deals(category): Browse discounted items\n\n## Rules\n- ALWAYS use cart_add for multiple items in ONE call, never one-by-one\n- Write search queries as you would type them in a grocery app\n- After cart_optimize, present both options clearly with totals in rupees\n- Keep responses concise — the user is on a phone\n- If an item isn't found, tell the user and suggest alternatives\n- Show prices in rupees (₹), not paise",
			},
			{
				ID:          "websearch",
				Name:        "Web Search",
				Icon:        "search",
				Description: "Search the web for information",
				ToolFilters: []string{"web_*", "search_*"},
				SystemPrompt: "You are a research assistant with web search capabilities.\n\n## Rules\n- Before searching, briefly explain what you're looking for\n- Always cite sources and provide relevant context\n- If search results are insufficient, try rephrasing the query\n- Be objective and comprehensive",
			},
		},
	}

	// Try to load from config file
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("Config file not found, using defaults", "path", configPath)
		} else {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Migrate old config format (backward compatibility)
	if cfg.LLMAPIKey != "" && cfg.LLM.Provider == "" {
		cfg.LLM = LLMConfig{
			Provider: "groq",
			APIKey:   cfg.LLMAPIKey,
			Model:    "gemma-4-26b-a4b-it",
		}
	}
	// If LLM.APIKey is not set but LLMAPIKey is, use it
	if cfg.LLM.APIKey == "" && cfg.LLMAPIKey != "" {
		cfg.LLM.APIKey = cfg.LLMAPIKey
	}

	// Override with environment variables
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		} else {
			slog.Warn("Invalid SERVER_PORT value, using default", "value", port, "default", cfg.Port)
		}
	}

	if name := os.Getenv("SERVICE_NAME"); name != "" {
		cfg.ServiceName = name
	}

	if dir := os.Getenv("CERTS_DIR"); dir != "" {
		cfg.CertsDir = dir
	}

	if dir := os.Getenv("MCP_SERVERS_DIR"); dir != "" {
		cfg.MCPServersDir = dir
	}

	// Check API key env vars (Groq preferred)
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		cfg.LLMAPIKey = key
		cfg.LLM.APIKey = key
	} else if key := os.Getenv("LLM_API_KEY"); key != "" {
		cfg.LLMAPIKey = key
		cfg.LLM.APIKey = key
	}

	// Cerebras API key
	if key := os.Getenv("CEREBRAS_API_KEY"); key != "" {
		cfg.CerebrasAPIKey = key
	}

	// LLM provider env overrides
	if provider := os.Getenv("LLM_PROVIDER"); provider != "" {
		cfg.LLM.Provider = provider
	}
	if model := os.Getenv("LLM_MODEL"); model != "" {
		cfg.LLM.Model = model
	}
	if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
		cfg.LLM.BaseURL = baseURL
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	// Validate required fields
	switch cfg.LLM.Provider {
	case "groq":
		if cfg.LLM.APIKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY (or LLM_API_KEY) is required for Groq provider")
		}
	case "cerebras":
		if cfg.CerebrasAPIKey == "" {
			return nil, fmt.Errorf("CEREBRAS_API_KEY is required for Cerebras provider")
		}
	case "rotating":
		// Rotating needs at least one real API key OR cliproxy available
		hasKey := cfg.LLM.APIKey != "" || cfg.CerebrasAPIKey != ""
		hasCliProxy := os.Getenv("CLIPROXY_API_KEY") != "" || os.Getenv("CLIPROXY_URL") != ""
		if !hasKey && !hasCliProxy {
			return nil, fmt.Errorf("at least one of GROQ_API_KEY, CEREBRAS_API_KEY, or CLIPROXY_API_KEY/CLIPROXY_URL is required for rotating provider")
		}
	case "cliproxy":
		// No API key required
	}

	// Convert relative paths to absolute
	if !filepath.IsAbs(cfg.CertsDir) {
		if abs, err := filepath.Abs(cfg.CertsDir); err == nil {
			cfg.CertsDir = abs
		}
	}

	if !filepath.IsAbs(cfg.MCPServersDir) {
		if abs, err := filepath.Abs(cfg.MCPServersDir); err == nil {
			cfg.MCPServersDir = abs
		}
	}

	// YouTube env overrides
	if dir := os.Getenv("YOUTUBE_BIN_DIR"); dir != "" {
		cfg.YouTube.BinDir = dir
	}
	if dir := os.Getenv("YOUTUBE_OUTPUT_DIR"); dir != "" {
		cfg.YouTube.OutputDir = dir
	}
	if enabled := os.Getenv("YOUTUBE_ENABLED"); enabled != "" {
		cfg.YouTube.Enabled = enabled == "true" || enabled == "1"
	}

	// Convert YouTube dirs to absolute
	if cfg.YouTube.BinDir != "" && !filepath.IsAbs(cfg.YouTube.BinDir) {
		if abs, err := filepath.Abs(cfg.YouTube.BinDir); err == nil {
			cfg.YouTube.BinDir = abs
		}
	}
	if cfg.YouTube.OutputDir != "" && !filepath.IsAbs(cfg.YouTube.OutputDir) {
		if abs, err := filepath.Abs(cfg.YouTube.OutputDir); err == nil {
			cfg.YouTube.OutputDir = abs
		}
	}

	// Radio env overrides
	if enabled := os.Getenv("RADIO_ENABLED"); enabled != "" {
		cfg.Radio.Enabled = enabled == "true" || enabled == "1"
	}
	if path := os.Getenv("LIQUIDSOAP_PATH"); path != "" {
		cfg.Radio.LiquidsoapPath = path
	}
	if path := os.Getenv("RADIO_SCRIPT_PATH"); path != "" {
		cfg.Radio.ScriptPath = path
	}
	if host := os.Getenv("RADIO_TELNET_HOST"); host != "" {
		cfg.Radio.TelnetHost = host
	}
	if port := os.Getenv("RADIO_TELNET_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Radio.TelnetPort = p
		}
	}
	if port := os.Getenv("RADIO_STREAM_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Radio.StreamPort = p
		}
	}
	if host := os.Getenv("RADIO_STREAM_HOST"); host != "" {
		cfg.Radio.StreamHost = host
	}
	if secs := os.Getenv("RADIO_CROSSFADE_SECS"); secs != "" {
		if f, err := strconv.ParseFloat(secs, 64); err == nil {
			cfg.Radio.CrossfadeSecs = f
		}
	}
	if autoStart := os.Getenv("RADIO_AUTO_START"); autoStart != "" {
		cfg.Radio.AutoStart = autoStart == "true" || autoStart == "1"
	}

	// Convert radio script path to absolute
	if cfg.Radio.ScriptPath != "" && !filepath.IsAbs(cfg.Radio.ScriptPath) {
		if abs, err := filepath.Abs(cfg.Radio.ScriptPath); err == nil {
			cfg.Radio.ScriptPath = abs
		}
	}

	// Library env overrides
	if dir := os.Getenv("LIBRARY_DATA_DIR"); dir != "" {
		cfg.Library.DataDir = dir
	}
	if path := os.Getenv("LIBRARY_DATABASE_PATH"); path != "" {
		cfg.Library.DatabasePath = path
	}
	if size := os.Getenv("LIBRARY_MAX_SIZE_MB"); size != "" {
		if s, err := strconv.Atoi(size); err == nil {
			cfg.Library.MaxSizeMB = s
		}
	}

	// Convert library paths to absolute
	if cfg.Library.DataDir != "" && !filepath.IsAbs(cfg.Library.DataDir) {
		if abs, err := filepath.Abs(cfg.Library.DataDir); err == nil {
			cfg.Library.DataDir = abs
		}
	}
	if cfg.Library.DatabasePath != "" && !filepath.IsAbs(cfg.Library.DatabasePath) {
		if abs, err := filepath.Abs(cfg.Library.DatabasePath); err == nil {
			cfg.Library.DatabasePath = abs
		}
	}

	return cfg, nil
}

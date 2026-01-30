package config

import (
	"fmt"
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

	// API Keys
	LLMAPIKey string `yaml:"llm_api_key"`

	// Logging
	LogLevel string `yaml:"log_level"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:          8443,
		ServiceName:   "pocket-assistant",
		CertsDir:      "../certs",
		MCPServersDir: "./mcp-servers",
		LogLevel:      "info",
	}

	// Try to load from config file
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
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

	// Check multiple API key env vars (Groq preferred, fallback to Gemini for compatibility)
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		cfg.LLMAPIKey = key
	} else if key := os.Getenv("LLM_API_KEY"); key != "" {
		cfg.LLMAPIKey = key
	} else if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		cfg.LLMAPIKey = key
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	// Validate required fields
	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("GROQ_API_KEY (or LLM_API_KEY) is required")
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

	return cfg, nil
}

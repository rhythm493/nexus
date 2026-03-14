package llm

import (
	"fmt"
	"time"
)

// ProviderConfig holds LLM provider configuration
// This mirrors the config.LLMConfig struct
type ProviderConfig struct {
	Provider string // "groq", "cerebras", "ollama", "lmstudio", "rotating"
	Model    string
	APIKey   string
	BaseURL  string
	Slots    []SlotConfig
}

// ProviderInfo holds metadata about an available provider
type ProviderInfo struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	RequiresAuth bool   `json:"requires_auth"`
	DefaultURL   string `json:"default_url"`
}

// ProviderFactory creates LLM providers
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a provider from configuration
func (f *ProviderFactory) CreateProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Provider {
	case "groq":
		return NewOpenAIProvider("groq", "https://api.groq.com/openai/v1", cfg.APIKey, cfg.Model, 60*time.Second), nil
	case "cerebras":
		return NewOpenAIProvider("cerebras", "https://api.cerebras.ai/v1", cfg.APIKey, cfg.Model, 60*time.Second), nil
	case "ollama":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		return NewOpenAIProvider("ollama", baseURL, "", cfg.Model, 120*time.Second), nil
	case "lmstudio":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:1234/v1"
		}
		return NewOpenAIProvider("lmstudio", baseURL, "", cfg.Model, 120*time.Second), nil
	case "rotating":
		return NewRotatingProvider(cfg.Slots)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// NewGroqProvider creates a new Groq provider (backward-compat wrapper)
func NewGroqProvider(apiKey, model string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key required for Groq")
	}
	if model == "" {
		model = "meta-llama/llama-4-scout-17b-16e-instruct"
	}
	return NewOpenAIProvider("groq", "https://api.groq.com/openai/v1", apiKey, model, 60*time.Second), nil
}

// NewOllamaProvider creates a new Ollama provider (backward-compat wrapper)
func NewOllamaProvider(baseURL, model string) (Provider, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	if model == "" {
		return nil, fmt.Errorf("model required for Ollama")
	}
	return NewOpenAIProvider("ollama", baseURL, "", model, 120*time.Second), nil
}

// NewLMStudioProvider creates a new LM Studio provider (backward-compat wrapper)
func NewLMStudioProvider(baseURL, model string) (Provider, error) {
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}
	if model == "" {
		return nil, fmt.Errorf("model required for LM Studio")
	}
	return NewOpenAIProvider("lmstudio", baseURL, "", model, 120*time.Second), nil
}

// GetAvailableProviders returns metadata about all available providers
func (f *ProviderFactory) GetAvailableProviders() []ProviderInfo {
	return []ProviderInfo{
		{
			Name:         "groq",
			DisplayName:  "Groq",
			RequiresAuth: true,
			DefaultURL:   "https://api.groq.com/openai/v1",
		},
		{
			Name:         "cerebras",
			DisplayName:  "Cerebras",
			RequiresAuth: true,
			DefaultURL:   "https://api.cerebras.ai/v1",
		},
		{
			Name:         "ollama",
			DisplayName:  "Ollama",
			RequiresAuth: false,
			DefaultURL:   "http://localhost:11434/v1",
		},
		{
			Name:         "lmstudio",
			DisplayName:  "LM Studio",
			RequiresAuth: false,
			DefaultURL:   "http://localhost:1234/v1",
		},
	}
}

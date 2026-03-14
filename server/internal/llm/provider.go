package llm

import "context"

// Provider defines the interface all LLM providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "groq", "ollama", "lmstudio")
	Name() string

	// Chat sends a message and returns the response
	Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error)

	// ListModels returns available models from this provider
	ListModels(ctx context.Context) ([]Model, error)

	// SupportsTools returns whether this provider supports function calling
	SupportsTools() bool

	// GetModel returns the currently configured model
	GetModel() string

	// SetModel changes the model for this provider
	SetModel(model string)
}

// Model represents an available LLM model
type Model struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	ContextSize int                    `json:"context_size,omitempty"`
	OwnedBy     string                 `json:"owned_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

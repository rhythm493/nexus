package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAICompatibleProvider implements the Provider interface using the
// go-openai SDK. Works with any OpenAI-compatible API (Groq, Cerebras,
// Ollama, LM Studio, CLIProxyAPI, etc.).
type OpenAICompatibleProvider struct {
	name    string
	baseURL string
	model   string
	client  *openai.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
// If apiKey is empty, a dummy key is used (some providers don't require auth).
func NewOpenAIProvider(name, baseURL, apiKey, model string, timeout time.Duration) *OpenAICompatibleProvider {
	if apiKey == "" {
		apiKey = "no-key"
	}

	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	cfg.HTTPClient = &http.Client{Timeout: timeout}

	return &OpenAICompatibleProvider{
		name:    name,
		baseURL: baseURL,
		model:   model,
		client:  openai.NewClientWithConfig(cfg),
	}
}

// Name returns the provider name
func (p *OpenAICompatibleProvider) Name() string { return p.name }

// GetModel returns the currently configured model
func (p *OpenAICompatibleProvider) GetModel() string { return p.model }

// SetModel changes the model for this provider
func (p *OpenAICompatibleProvider) SetModel(model string) { p.model = model }

// SupportsTools returns whether this provider supports function calling
func (p *OpenAICompatibleProvider) SupportsTools() bool { return true }

// Chat sends a message and returns the response
func (p *OpenAICompatibleProvider) Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    messagesToSDK(messages),
		Temperature: float32(0.7),
		MaxTokens:   512,
	}

	if len(tools) > 0 {
		req.Tools = toolsToSDK(tools)
		req.ToolChoice = "auto"
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, convertSDKError(err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return responseFromSDK(resp), nil
}

// ListModels returns available models from the provider
func (p *OpenAICompatibleProvider) ListModels(ctx context.Context) ([]Model, error) {
	list, err := p.client.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return modelsFromSDK(list), nil
}

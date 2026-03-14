package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAICompatibleProvider implements the Provider interface for any
// OpenAI-compatible API (Groq, Ollama, LM Studio, etc.).
type OpenAICompatibleProvider struct {
	name       string
	apiKey     string // empty means no auth header
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
// If apiKey is empty, no Authorization header will be sent.
func NewOpenAIProvider(name, baseURL, apiKey, model string, timeout time.Duration) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		name:       name,
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
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
	// Build request
	reqBody := ChatRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   512,
	}

	// Add tools if provided
	if len(tools) > 0 {
		reqBody.Tools = tools
		reqBody.ToolChoice = "auto"
	}

	// Marshal request
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for rate limiting before reading the full body
	if resp.StatusCode == http.StatusTooManyRequests {
		respBody, _ := io.ReadAll(resp.Body)
		msg := string(respBody)
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &RateLimitError{
			RetryAfter: retryAfter,
			Message:    msg,
		}
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API error
	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Build response
	choice := chatResp.Choices[0]
	return &Response{
		Text:      choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
	}, nil
}

// ListModels returns available models from the provider
func (p *OpenAICompatibleProvider) ListModels(ctx context.Context) ([]Model, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response (OpenAI format: {"data": [...], "object": "list"})
	var apiResp struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
			Context int    `json:"context_window"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our Model format
	models := make([]Model, 0, len(apiResp.Data))
	for _, m := range apiResp.Data {
		models = append(models, Model{
			ID:          m.ID,
			Name:        m.ID,
			OwnedBy:     m.OwnedBy,
			ContextSize: m.Context,
		})
	}

	return models, nil
}

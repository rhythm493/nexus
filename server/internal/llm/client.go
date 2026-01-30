package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client wraps the LLM API client (Groq with OpenAI-compatible API)
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"` // "user", "assistant", "system", or "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool represents a function that can be called
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition
type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool invocation request from the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function call details
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Response represents a chat response
type Response struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatRequest is the request body for the chat API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse is the response from the chat API
type ChatResponse struct {
	ID      string    `json:"id"`
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewClient creates a new LLM client for Groq
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    "https://api.groq.com/openai/v1",
		model:      "meta-llama/llama-4-scout-17b-16e-instruct", // 30K TPM vs 12K for llama-3.3
		httpClient: &http.Client{},
	}, nil
}

// Chat sends a message and returns the response
func (c *Client) Chat(ctx context.Context, messages []Message, tools []Tool) (*Response, error) {
	// Build request
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1024,
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
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

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
	result := &Response{
		Text:      choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
	}

	return result, nil
}

// ConvertMCPTools converts MCP tools to LLM tool format
func ConvertMCPTools(mcpTools []struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}) []Tool {
	var tools []Tool
	for _, t := range mcpTools {
		tools = append(tools, Tool{
			Type: "function",
			Function: Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return tools
}

// DefaultSystemPrompt is the fallback system prompt
const DefaultSystemPrompt = `You are a helpful voice assistant called Pocket Assistant.
You help users control their smart home devices and answer questions.
Keep responses concise and conversational since they will be spoken aloud.
When using tools, explain what you're doing briefly.`

// GenerateSystemPrompt generates a dynamic system prompt based on available tools
func GenerateSystemPrompt(tools []Tool) string {
	var b strings.Builder
	b.WriteString("You are a helpful voice assistant called Pocket Assistant.\n")
	b.WriteString("You help users control their smart home devices and answer questions.\n")
	b.WriteString("Keep responses concise and conversational since they will be spoken aloud.\n")
	b.WriteString("When using tools, explain what you're doing briefly.\n\n")

	if len(tools) == 0 {
		return b.String()
	}

	b.WriteString("You have access to tools for controlling Sonos devices. Here are some guidelines:\n")

	// Categorize and provide hints based on tool availability
	hasSonos := false
	for _, t := range tools {
		if strings.HasPrefix(t.Function.Name, "sonos_") {
			hasSonos = true
			break
		}
	}

	if hasSonos {
		b.WriteString("### Sonos Control Guidelines:\n")
		b.WriteString("1. **Discovery**: Use 'sonos_discover' to find devices on the network. Use 'sonos_list_devices' to see currently registered devices.\n")
		b.WriteString("2. **Device Selection**: Most tools require a 'name' parameter (e.g., 'Kitchen', 'Living Room'). If the user doesn't specify, use the first available device from 'sonos_list_devices'.\n")
		b.WriteString("3. **Playback**: Use 'sonos_play', 'sonos_pause', 'sonos_next', 'sonos_previous'. To play specific content, use 'sonos_play_music_service_item' or browse the library.\n")
		b.WriteString("4. **Volume**: Use 'sonos_set_volume' (0-100) or 'sonos_set_mute'.\n")
		b.WriteString("5. **Groups**: Use 'sonos_join_group' to group speakers or 'sonos_party_mode' to play everywhere.\n")
		b.WriteString("6. **Streaming**: You can browse and search music services (Spotify, etc.) using 'sonos_browse_music_service' and 'sonos_search_music_service'.\n")
		b.WriteString("7. **Status**: Use 'sonos_get_playback_state' or 'sonos_get_transport_info' to see what's currently playing.\n")
	}

	return b.String()
}

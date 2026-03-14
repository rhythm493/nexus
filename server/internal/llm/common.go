package llm

import (
	"encoding/json"
	"strings"

	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

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

// ChatRequest is the request body for the chat API (OpenAI-compatible format)
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  string    `json:"tool_choice,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse is the response from the chat API (OpenAI-compatible format)
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

// ConvertMCPTools converts MCP tools to LLM tool format
func ConvertMCPTools(mcpTools []mcp.Tool) []Tool {
	var tools []Tool
	for _, t := range mcpTools {
		// Create a deep copy of the schema to loosen numeric types
		schemaBytes, _ := json.Marshal(t.InputSchema)
		var loosenedSchema map[string]interface{}
		json.Unmarshal(schemaBytes, &loosenedSchema)

		if properties, ok := loosenedSchema["properties"].(map[string]interface{}); ok {
			for _, p := range properties {
				if prop, ok := p.(map[string]interface{}); ok {
					if propType, ok := prop["type"].(string); ok && (propType == "number" || propType == "integer") {
						// Change to allow both for maximum flexibility
						prop["type"] = []string{propType, "string"}
						// Remove default if it's a number to avoid conflict with string type if gateway is super strict
						delete(prop, "default")
					}
				}
			}
		}

		tools = append(tools, Tool{
			Type: "function",
			Function: Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  loosenedSchema,
			},
		})
	}
	return tools
}

// DefaultSystemPrompt is the fallback system prompt
const DefaultSystemPrompt = `You are Nexus, a voice assistant. Be concise.`

// GenerateSystemPrompt generates a dynamic system prompt based on available tools
// If simplified is true, returns a minimal prompt for general chat
func GenerateSystemPrompt(tools []Tool, simplified bool) string {
	// Simplified prompt for general chat - no tools needed
	if simplified || len(tools) == 0 {
		return `You are Nexus, a helpful voice assistant for smart home control.
Keep responses brief (1-2 sentences) since they will be spoken aloud.
You can control music playback and Sonos speakers. Ask if user needs help with these.`
	}

	var b strings.Builder
	b.WriteString("You are Nexus, a voice assistant for smart home and music control.\n")
	b.WriteString("Keep responses concise (1-2 sentences) for voice output.\n")
	b.WriteString("IMPORTANT: Always use tools to get real data. Never guess or make up information.\n\n")

	// Check for available tool categories
	hasRadio := false
	hasSonos := false
	for _, t := range tools {
		if strings.HasPrefix(t.Function.Name, "radio_") || t.Function.Name == "library_search" || t.Function.Name == "youtube_search" {
			hasRadio = true
		}
		if strings.HasPrefix(t.Function.Name, "sonos_") {
			hasSonos = true
		}
	}

	// Radio guidelines
	if hasRadio {
		b.WriteString("## Radio (Music Playback)\n")
		b.WriteString("- radio_play(query): Search and play a song on the radio stream\n")
		b.WriteString("- radio_queue(query): Add song to queue\n")
		b.WriteString("- radio_skip(): Skip current song\n")
		b.WriteString("- radio_status(): Get current track and queue - USE THIS when asked what's playing\n")
		if hasSonos {
			b.WriteString("- IMPORTANT: After radio_play, use sonos_play_url with url=\"http://192.168.0.129:8080/stream\" to play on Sonos\n")
		}
		b.WriteString("\n")
	}

	// Sonos guidelines
	if hasSonos {
		b.WriteString("## Sonos Speakers\n")
		b.WriteString("- sonos_list_devices: Get available speakers\n")
		b.WriteString("- sonos_set_volume(deviceId, volume): Set volume 0-100\n")
		b.WriteString("- sonos_play/pause/stop: Control playback\n")
		b.WriteString("- For Spotify: First sonos_search_music_service, then sonos_play_music_service_item with itemId\n\n")
	}

	b.WriteString("Parameter types: Use numbers for volume/count (25 not \"25\").\n")

	return b.String()
}

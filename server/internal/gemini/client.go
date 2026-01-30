package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Client wraps the Gemini API client
type Client struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

// Message represents a chat message
type Message struct {
	Role       string      `json:"role"` // "user", "assistant", or "tool"
	Content    string      `json:"content"`
	ToolCall   *ToolCall   `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// Tool represents a function that can be called
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool invocation request from the model
type ToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Name   string      `json:"name"`
	Result interface{} `json:"result"`
}

// Response represents a chat response from Gemini
type Response struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// NewClient creates a new Gemini client
func NewClient(apiKey string) (*Client, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	model := client.GenerativeModel("gemma-3-4b-it")
	model.SetTemperature(0.7)

	// Set system instruction
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(`You are a helpful voice assistant called Pocket Assistant.
You help users control their smart home devices and answer questions.
Keep responses concise and conversational since they will be spoken aloud.
When using tools, explain what you're doing briefly.`),
		},
	}

	return &Client{
		client: client,
		model:  model,
	}, nil
}

// Chat sends a message and returns the response
func (c *Client) Chat(ctx context.Context, history []Message, tools []Tool) (*Response, error) {
	// Configure tools if provided
	if len(tools) > 0 {
		var funcDecls []*genai.FunctionDeclaration
		for _, t := range tools {
			// Convert parameters to genai.Schema
			schema := convertToSchema(t.Parameters)
			funcDecls = append(funcDecls, &genai.FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			})
		}
		c.model.Tools = []*genai.Tool{{FunctionDeclarations: funcDecls}}
	}

	// Start chat session
	cs := c.model.StartChat()

	// Add history
	for _, msg := range history[:len(history)-1] {
		content := messageToContent(msg)
		if content != nil {
			cs.History = append(cs.History, content)
		}
	}

	// Get the last message (should be user message)
	lastMsg := history[len(history)-1]

	var prompt []genai.Part
	if lastMsg.ToolResult != nil {
		// Send tool result
		resultJSON, _ := json.Marshal(lastMsg.ToolResult.Result)
		prompt = []genai.Part{
			genai.FunctionResponse{
				Name:     lastMsg.ToolResult.Name,
				Response: map[string]interface{}{"result": string(resultJSON)},
			},
		}
	} else {
		prompt = []genai.Part{genai.Text(lastMsg.Content)}
	}

	// Send message
	resp, err := cs.SendMessage(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Parse response
	result := &Response{}
	for _, candidate := range resp.Candidates {
		if candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			switch p := part.(type) {
			case genai.Text:
				result.Text += string(p)
			case genai.FunctionCall:
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					Name: p.Name,
					Args: p.Args,
				})
			}
		}
	}

	return result, nil
}

// Close closes the client
func (c *Client) Close() error {
	return c.client.Close()
}

// messageToContent converts a Message to genai.Content
func messageToContent(msg Message) *genai.Content {
	switch msg.Role {
	case "user":
		return &genai.Content{
			Role:  "user",
			Parts: []genai.Part{genai.Text(msg.Content)},
		}
	case "assistant":
		if msg.ToolCall != nil {
			return &genai.Content{
				Role: "model",
				Parts: []genai.Part{
					genai.FunctionCall{
						Name: msg.ToolCall.Name,
						Args: msg.ToolCall.Args,
					},
				},
			}
		}
		return &genai.Content{
			Role:  "model",
			Parts: []genai.Part{genai.Text(msg.Content)},
		}
	case "tool":
		if msg.ToolResult != nil {
			resultJSON, _ := json.Marshal(msg.ToolResult.Result)
			return &genai.Content{
				Role: "user",
				Parts: []genai.Part{
					genai.FunctionResponse{
						Name:     msg.ToolResult.Name,
						Response: map[string]interface{}{"result": string(resultJSON)},
					},
				},
			}
		}
	}
	return nil
}

// convertToSchema converts a JSON schema map to genai.Schema
func convertToSchema(params map[string]interface{}) *genai.Schema {
	if params == nil {
		return nil
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
	}

	if props, ok := params["properties"].(map[string]interface{}); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for name, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				schema.Properties[name] = convertPropertyToSchema(propMap)
			}
		}
	}

	if required, ok := params["required"].([]interface{}); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return schema
}

func convertPropertyToSchema(prop map[string]interface{}) *genai.Schema {
	schema := &genai.Schema{}

	if t, ok := prop["type"].(string); ok {
		switch t {
		case "string":
			schema.Type = genai.TypeString
		case "number":
			schema.Type = genai.TypeNumber
		case "integer":
			schema.Type = genai.TypeInteger
		case "boolean":
			schema.Type = genai.TypeBoolean
		case "array":
			schema.Type = genai.TypeArray
			if items, ok := prop["items"].(map[string]interface{}); ok {
				schema.Items = convertPropertyToSchema(items)
			}
		case "object":
			schema.Type = genai.TypeObject
		}
	}

	if desc, ok := prop["description"].(string); ok {
		schema.Description = desc
	}

	if enum, ok := prop["enum"].([]interface{}); ok {
		for _, e := range enum {
			if s, ok := e.(string); ok {
				schema.Enum = append(schema.Enum, s)
			}
		}
	}

	return schema
}

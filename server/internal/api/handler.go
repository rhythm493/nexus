package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rhythm493/pocket-assistant/server/config"
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
)

// Server represents the API server
type Server struct {
	config        *config.Config
	llmClient     *llm.Client
	mcpHost       *mcp.Host
	tlsConfig     *tls.Config
	httpServer    *http.Server
	conversations sync.Map // map[string]*Conversation
}

// Conversation holds chat history
type Conversation struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
	mu       sync.Mutex
}

// Message represents a chat message
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRequest is the incoming chat request
type ChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id,omitempty"`
}

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type    string      `json:"type"` // "text", "tool_call", "tool_result", "error", "done"
	Content string      `json:"content,omitempty"`
	Name    string      `json:"name,omitempty"`
	Args    interface{} `json:"args,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

// HealthResponse is the health check response
type HealthResponse struct {
	Status     string   `json:"status"`
	MCPServers []string `json:"mcp_servers"`
}

// ToolsResponse lists available tools
type ToolsResponse struct {
	Tools []mcp.Tool `json:"tools"`
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, llmClient *llm.Client, mcpHost *mcp.Host, tlsConfig *tls.Config) *Server {
	return &Server{
		config:    cfg,
		llmClient: llmClient,
		mcpHost:   mcpHost,
		tlsConfig: tlsConfig,
	}
}

// Start starts the HTTPS server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("POST /api/v1/chat", s.handleChat)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/tools", s.handleTools)
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.handleGetConversation)

	// Wrap with middleware
	handler := s.logMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:      fmt.Sprintf(":%d", s.config.Port),
		Handler:   handler,
		TLSConfig: s.tlsConfig,
	}

	return s.httpServer.ListenAndServeTLS("", "")
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	servers := s.mcpHost.ListServers()

	resp := HealthResponse{
		Status:     "ok",
		MCPServers: servers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleTools returns available MCP tools
func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	tools := s.mcpHost.ListTools()

	resp := ToolsResponse{
		Tools: tools,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleGetConversation returns a conversation by ID
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	conv, ok := s.conversations.Load(id)
	if !ok {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

// handleChat processes chat messages with SSE streaming
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	slog.Info("Chat request", "message", req.Message, "conversation_id", req.ConversationID)

	// Get or create conversation
	convID := req.ConversationID
	if convID == "" {
		convID = uuid.New().String()
	}

	conv := s.getOrCreateConversation(convID)

	// Add user message
	conv.mu.Lock()
	conv.Messages = append(conv.Messages, Message{
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
	})
	conv.mu.Unlock()

	// Setup SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Conversation-ID", convID)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Get available tools from MCP
	mcpTools := s.mcpHost.ListTools()
	tools := convertToLLMTools(mcpTools)

	// Build message history
	var history []llm.Message
	// Add system prompt
	history = append(history, llm.Message{
		Role:    "system",
		Content: llm.GenerateSystemPrompt(tools),
	})
	// Add conversation history
	conv.mu.Lock()
	for _, msg := range conv.Messages {
		history = append(history, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	conv.mu.Unlock()

	ctx := r.Context()
	var assistantResponse string

	// Chat loop with tool calling
	for {
		// Call LLM
		response, err := s.llmClient.Chat(ctx, history, tools)
		if err != nil {
			s.sendSSE(w, flusher, SSEEvent{Type: "error", Content: err.Error()})
			return
		}

		// Check for tool calls
		if len(response.ToolCalls) > 0 {
			for _, toolCall := range response.ToolCalls {
				// Parse arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					slog.Error("Failed to parse tool arguments", "error", err)
					args = map[string]interface{}{}
				}

				// Send tool call event
				slog.Info("Tool call", "tool", toolCall.Function.Name, "args", args)
				s.sendSSE(w, flusher, SSEEvent{
					Type: "tool_call",
					Name: toolCall.Function.Name,
					Args: args,
				})

				// Execute tool via MCP
				result, err := s.mcpHost.ExecuteTool(ctx, toolCall.Function.Name, args)
				if err != nil {
					slog.Error("Tool execution failed", "tool", toolCall.Function.Name, "error", err)
					result = map[string]interface{}{"error": err.Error()}
				}

				// Send tool result event
				s.sendSSE(w, flusher, SSEEvent{
					Type:   "tool_result",
					Name:   toolCall.Function.Name,
					Result: result,
				})

				// Add assistant message with tool call to history
				history = append(history, llm.Message{
					Role:      "assistant",
					ToolCalls: []llm.ToolCall{toolCall},
				})

				// Add tool result to history
				resultJSON, _ := json.Marshal(result)
				history = append(history, llm.Message{
					Role:       "tool",
					Content:    string(resultJSON),
					ToolCallID: toolCall.ID,
				})
			}
			// Continue loop to get response after tool execution
			continue
		}

		// No tool calls, we have the final text response
		if response.Text != "" {
			assistantResponse = response.Text
			slog.Info("LLM response", "text", response.Text)
			s.sendSSE(w, flusher, SSEEvent{Type: "text", Content: response.Text})
		}
		break
	}

	// Save assistant response
	if assistantResponse != "" {
		conv.mu.Lock()
		conv.Messages = append(conv.Messages, Message{
			Role:      "assistant",
			Content:   assistantResponse,
			Timestamp: time.Now(),
		})
		conv.mu.Unlock()
	}

	// Send done event
	s.sendSSE(w, flusher, SSEEvent{Type: "done"})
}

func (s *Server) getOrCreateConversation(id string) *Conversation {
	if conv, ok := s.conversations.Load(id); ok {
		return conv.(*Conversation)
	}

	conv := &Conversation{
		ID:       id,
		Messages: []Message{},
	}
	s.conversations.Store(id, conv)
	return conv
}

func (s *Server) sendSSE(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// convertToLLMTools converts MCP tools to LLM tool format
func convertToLLMTools(mcpTools []mcp.Tool) []llm.Tool {
	var tools []llm.Tool
	for _, t := range mcpTools {
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return tools
}

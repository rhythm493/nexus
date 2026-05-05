package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rhythm493/pocket-assistant/server/config"
	"github.com/rhythm493/pocket-assistant/server/internal/cart"
	"github.com/rhythm493/pocket-assistant/server/internal/discovery"
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
	"github.com/rhythm493/pocket-assistant/server/internal/mode"
	"github.com/rhythm493/pocket-assistant/server/internal/quickcom"
	"github.com/rhythm493/pocket-assistant/server/internal/radio"
	"github.com/rhythm493/pocket-assistant/server/internal/youtube"
)

// Server represents the API server
type Server struct {
	config        *config.Config
	llmProvider   llm.Provider
	mcpHost       *mcp.Host
	modeManager   *mode.Manager
	httpServer    *http.Server
	conversations sync.Map // map[string]*Conversation
	youtube       *youtube.Service
	radioEngine   *radio.Engine
	radioTools    *radio.ToolHandlers
	cartManager    *cart.Manager
	cartTools      *cart.ToolHandlers
	quickcomClient *quickcom.Client
	rateLimits    sync.Map // map[string]time.Time — per-IP last request time
	done          chan struct{}
}

// Conversation holds chat history
type Conversation struct {
	ID         string    `json:"id"`
	Messages   []Message `json:"messages"`
	LastAccess time.Time `json:"last_access"`
	mu         sync.Mutex
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
	Provider       string `json:"provider,omitempty"` // Optional provider override
	Model          string `json:"model,omitempty"`    // Optional model override
	Mode           string `json:"mode,omitempty"`     // Optional mode selection
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
func NewServer(cfg *config.Config, llmProvider llm.Provider, mcpHost *mcp.Host, modeManager *mode.Manager, yt *youtube.Service, radioEngine *radio.Engine, radioTools *radio.ToolHandlers, cartManager *cart.Manager, cartTools *cart.ToolHandlers, quickcomClient *quickcom.Client) *Server {
	return &Server{
		config:         cfg,
		llmProvider:    llmProvider,
		mcpHost:        mcpHost,
		modeManager:    modeManager,
		youtube:        yt,
		radioEngine:    radioEngine,
		radioTools:     radioTools,
		cartManager:    cartManager,
		cartTools:      cartTools,
		quickcomClient: quickcomClient,
		done:           make(chan struct{}),
	}
}

// startConversationCleanup runs a background goroutine that evicts conversations
// older than 1 hour. It stops when the done channel is closed.
func (s *Server) startConversationCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				now := time.Now()
				s.conversations.Range(func(key, value any) bool {
					conv := value.(*Conversation)
					conv.mu.Lock()
					lastAccess := conv.LastAccess
					conv.mu.Unlock()
					if now.Sub(lastAccess) > 1*time.Hour {
						s.conversations.Delete(key)
						slog.Debug("Evicted stale conversation", "id", key)
					}
					return true
				})
				// Clean up stale carts
				if s.cartManager != nil {
					s.cartManager.CleanupStaleCarts(1 * time.Hour)
				}
				// Also clean up stale rate limit entries (older than 1 minute)
				s.rateLimits.Range(func(key, value any) bool {
					lastReq := value.(time.Time)
					if now.Sub(lastReq) > 1*time.Minute {
						s.rateLimits.Delete(key)
					}
					return true
				})
			}
		}
	}()
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.startConversationCleanup()

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("POST /api/v1/chat", s.handleChat)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/tools", s.handleTools)
	mux.HandleFunc("GET /api/v1/modes", s.handleGetModes)
	mux.HandleFunc("GET /api/v1/conversations/{id}", s.handleGetConversation)
	mux.HandleFunc("GET /api/v1/qrcode", discovery.HandleQRCode(s.config.Port))

	// YouTube endpoints
	if s.youtube != nil {
		mux.HandleFunc("GET /api/v1/youtube/search", s.handleYouTubeSearch)
		mux.HandleFunc("POST /api/v1/youtube/download", s.handleYouTubeDownload)
	}

	// Radio endpoints
	if s.radioEngine != nil {
		mux.HandleFunc("GET /api/v1/radio/status", s.handleRadioStatus)
		mux.HandleFunc("POST /api/v1/radio/play", s.handleRadioPlay)
		mux.HandleFunc("POST /api/v1/radio/queue", s.handleRadioQueue)
		mux.HandleFunc("POST /api/v1/radio/skip", s.handleRadioSkip)
		mux.HandleFunc("GET /api/v1/radio/stream", s.handleRadioStream)
	}

	// Cart endpoints
	if s.cartManager != nil {
		mux.HandleFunc("GET /api/v1/cart/{convId}", s.handleGetCart)
		mux.HandleFunc("GET /api/v1/cart/{convId}/full", s.handleGetCartFull)
		mux.HandleFunc("POST /api/v1/cart/{convId}/swap", s.handleSwapCartItem)
		mux.HandleFunc("POST /api/v1/cart/{convId}/add", s.handleAddCartItem)
		mux.HandleFunc("DELETE /api/v1/cart/{convId}/item/{query}", s.handleRemoveCartItem)
	}

	// Direct search proxy (for inline search in grocery UI)
	if s.quickcomClient != nil {
		mux.HandleFunc("GET /api/v1/search", s.handleSearch)
	}

	// Model download endpoint (speech recognition model served from cache)
	mux.HandleFunc("GET /api/v1/models/speech", s.handleModelDownload)

	// Provider endpoints
	mux.HandleFunc("GET /api/v1/providers", s.handleGetProviders)
	mux.HandleFunc("GET /api/v1/providers/{name}/models", s.handleGetProviderModels)

	// Wrap with middleware
	handler := s.logMiddleware(mux)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: handler,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	close(s.done)
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode health response", "error", err)
	}
}

// handleTools returns available MCP tools
func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	tools := s.mcpHost.ListTools()

	resp := ToolsResponse{
		Tools: tools,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode tools response", "error", err)
	}
}

// handleGetModes returns available assistant modes
func (s *Server) handleGetModes(w http.ResponseWriter, r *http.Request) {
	modes := s.modeManager.ListModes()

	response := map[string]interface{}{
		"modes": modes,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode modes response", "error", err)
	}
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
	if err := json.NewEncoder(w).Encode(conv); err != nil {
		slog.Error("Failed to encode conversation response", "error", err)
	}
}

// handleChat processes chat messages with SSE streaming
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	// Simple per-IP rate limiting: reject if less than 500ms since last request
	clientIP := r.RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		clientIP = ip
	}
	now := time.Now()
	if lastVal, ok := s.rateLimits.Load(clientIP); ok {
		lastTime := lastVal.(time.Time)
		if now.Sub(lastTime) < 500*time.Millisecond {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
	}
	s.rateLimits.Store(clientIP, now)

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Use provider override if provided, otherwise use default
	provider := s.llmProvider
	if req.Provider != "" && req.Model != "" {
		factory := llm.NewProviderFactory()
		cfg := llm.ProviderConfig{
			Provider: req.Provider,
			Model:    req.Model,
			APIKey:   s.config.LLM.APIKey,
			BaseURL:  s.config.LLM.BaseURL,
		}
		customProvider, err := factory.CreateProvider(cfg)
		if err == nil {
			provider = customProvider
			slog.Info("Using custom provider", "provider", req.Provider, "model", req.Model)
		} else {
			slog.Warn("Failed to create custom provider, using default", "error", err)
		}
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

	// Get mode (from request or use default)
	currentMode := s.modeManager.GetMode(req.Mode)
	slog.Info("Using mode", "mode", currentMode.ID, "name", currentMode.Name)

	// Get all MCP tools
	allMCPTools := s.mcpHost.ListTools()

	// Filter tools based on mode
	filteredMCPTools := s.modeManager.FilterTools(currentMode.ID, allMCPTools)
	tools := llm.ConvertMCPTools(filteredMCPTools)

	// Add radio tools if mode allows them
	if s.radioTools != nil {
		radioToolDefs := s.radioTools.ToolDefinitions()
		for _, def := range radioToolDefs {
			// Check if radio tools match mode filters
			if s.modeManager.MatchesFilter(def.Name, currentMode) {
				tools = append(tools, llm.Tool{
					Type: "function",
					Function: llm.Function{
						Name:        def.Name,
						Description: def.Description,
						Parameters:  def.Parameters,
					},
				})
			}
		}
	}

	// Add cart tools if mode allows them
	if s.cartTools != nil {
		cartToolDefs := s.cartTools.ToolDefinitions()
		for _, def := range cartToolDefs {
			if s.modeManager.MatchesFilter(def.Name, currentMode) {
				tools = append(tools, llm.Tool{
					Type: "function",
					Function: llm.Function{
						Name:        def.Name,
						Description: def.Description,
						Parameters:  def.Parameters,
					},
				})
			}
		}
	}

	slog.Info("Tools selected", "mode", currentMode.ID, "count", len(tools))

	// Build message history with summarization
	var history []llm.Message

	// Add mode-specific system prompt
	history = append(history, llm.Message{
		Role:    "system",
		Content: s.modeManager.GetSystemPrompt(currentMode.ID),
	})

	// Add conversation history with summarization
	const maxRecentMessages = 10 // Keep last 10 messages verbatim
	const maxTotalMessages = 20  // Summarize if more than 20

	conv.mu.Lock()
	messages := conv.Messages

	if len(messages) > maxTotalMessages {
		// Summarize older messages
		oldMessages := messages[:len(messages)-maxRecentMessages]
		summary := summarizeOldMessages(oldMessages)
		if summary != "" {
			history = append(history, llm.Message{
				Role:    "system",
				Content: summary,
			})
		}
		// Keep only recent messages
		messages = messages[len(messages)-maxRecentMessages:]
	}

	for _, msg := range messages {
		history = append(history, llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	conv.mu.Unlock()

	// Use a detached context so tool execution completes even if client disconnects
	ctx, ctxCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer ctxCancel()
	var assistantResponse string

	// Chat loop with tool calling
	const maxToolIterations = 10
	toolIterations := 0

	for {
		toolIterations++
		if toolIterations > maxToolIterations {
			s.sendSSE(w, flusher, SSEEvent{Type: "error", Content: fmt.Sprintf("Reached maximum tool call limit (%d). Stopping to prevent infinite loop.", maxToolIterations)})
			break
		}

		// Call LLM
		response, err := provider.Chat(ctx, history, tools)
		if err != nil {
			s.sendSSE(w, flusher, SSEEvent{Type: "error", Content: err.Error()})
			return
		}

		// Check for tool calls
		if len(response.ToolCalls) > 0 {
			// Emit LLM reasoning as a "thinking" event (ReAct pattern)
			if response.Text != "" {
				slog.Info("LLM reasoning", "text", response.Text)
				s.sendSSE(w, flusher, SSEEvent{Type: "thinking", Content: response.Text})
				history = append(history, llm.Message{
					Role:    "assistant",
					Content: response.Text,
				})
			}

			for _, toolCall := range response.ToolCalls {
				// Parse arguments
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					slog.Error("Failed to parse tool arguments", "tool", toolCall.Function.Name, "error", err)
					errorMsg := fmt.Sprintf("Invalid tool arguments: could not parse JSON. Error: %v", err)
					s.sendSSE(w, flusher, SSEEvent{
						Type:   "tool_result",
						Name:   toolCall.Function.Name,
						Result: map[string]interface{}{"error": errorMsg},
					})
					history = append(history, llm.Message{
						Role:      "assistant",
						ToolCalls: []llm.ToolCall{toolCall},
					})
					history = append(history, llm.Message{
						Role:       "tool",
						Content:    errorMsg,
						ToolCallID: toolCall.ID,
					})
					continue
				}

				// Send tool call event
				slog.Info("Tool call", "tool", toolCall.Function.Name, "args", args)
				s.sendSSE(w, flusher, SSEEvent{
					Type: "tool_call",
					Name: toolCall.Function.Name,
					Args: args,
				})

				// Execute tool - check cart tools, then radio tools, then MCP
				var result interface{}
				if s.cartTools != nil && cart.IsCartTool(toolCall.Function.Name) {
					result, err = s.cartTools.ExecuteTool(ctx, convID, toolCall.Function.Name, args)
					if err != nil {
						slog.Error("Cart tool execution failed", "tool", toolCall.Function.Name, "error", err)
						result = map[string]interface{}{"error": fmt.Sprintf("Tool '%s' failed: %s", toolCall.Function.Name, err.Error())}
					}
					// Emit cart-specific SSE events for Flutter UI (full detail for rich cards)
					if c := s.cartManager.GetCart(convID); c != nil {
						switch toolCall.Function.Name {
						case "cart_add", "cart_remove", "cart_clear":
							s.sendSSE(w, flusher, SSEEvent{Type: "cart_update", Result: c.FullDetail()})
						case "cart_optimize":
							s.sendSSE(w, flusher, SSEEvent{Type: "cart_update", Result: c.FullDetail()})
							if c.Optimization != nil {
								s.sendSSE(w, flusher, SSEEvent{Type: "cart_optimized", Result: c.Optimization})
							}
						}
					}
				} else if s.radioTools != nil && radio.IsRadioTool(toolCall.Function.Name) {
					result, err = s.radioTools.ExecuteTool(ctx, toolCall.Function.Name, args)
					if err != nil {
						slog.Error("Radio tool execution failed", "tool", toolCall.Function.Name, "error", err)
						var errorMsg string
						if errors.Is(err, context.DeadlineExceeded) {
							errorMsg = fmt.Sprintf("Tool '%s' timed out. The operation took too long. Try a simpler query or skip this step.", toolCall.Function.Name)
						} else if errors.Is(err, context.Canceled) {
							errorMsg = fmt.Sprintf("Tool '%s' was cancelled.", toolCall.Function.Name)
						} else if strings.Contains(err.Error(), "not found") {
							errorMsg = fmt.Sprintf("Tool '%s' error: resource not found. %s", toolCall.Function.Name, err.Error())
						} else {
							errorMsg = fmt.Sprintf("Tool '%s' failed: %s", toolCall.Function.Name, err.Error())
						}
						result = map[string]interface{}{"error": errorMsg}
					}
				} else {
					// Execute via MCP
					result, err = s.mcpHost.ExecuteTool(ctx, toolCall.Function.Name, args)
					if err != nil {
						slog.Error("Tool execution failed", "tool", toolCall.Function.Name, "error", err)
						var errorMsg string
						if errors.Is(err, context.DeadlineExceeded) {
							errorMsg = fmt.Sprintf("Tool '%s' timed out. The operation took too long. Try a simpler query or skip this step.", toolCall.Function.Name)
						} else if errors.Is(err, context.Canceled) {
							errorMsg = fmt.Sprintf("Tool '%s' was cancelled.", toolCall.Function.Name)
						} else if strings.Contains(err.Error(), "not found") {
							errorMsg = fmt.Sprintf("Tool '%s' error: resource not found. %s", toolCall.Function.Name, err.Error())
						} else {
							errorMsg = fmt.Sprintf("Tool '%s' failed: %s", toolCall.Function.Name, err.Error())
						}
						result = map[string]interface{}{"error": errorMsg}
					}
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

				// Add tool result to history (with size cap)
				resultJSON, _ := json.Marshal(result)
				const maxResultSize = 2048
				if len(resultJSON) > maxResultSize {
					// Truncate and add indicator
					resultJSON = append(resultJSON[:maxResultSize], []byte("\n... (truncated, result too large)")...)
				}
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
		c := conv.(*Conversation)
		c.mu.Lock()
		c.LastAccess = time.Now()
		c.mu.Unlock()
		return c
	}

	conv := &Conversation{
		ID:         id,
		Messages:   []Message{},
		LastAccess: time.Now(),
	}
	s.conversations.Store(id, conv)
	return conv
}

func (s *Server) sendSSE(w http.ResponseWriter, flusher http.Flusher, event SSEEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal SSE event", "error", err, "type", event.Type)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}


// Intent represents the detected user intent for tool selection
// Note: Intent detection has been replaced with mode-based tool filtering
// See mode.Manager for the new approach

// summarizeOldMessages creates a condensed summary of older messages
func summarizeOldMessages(messages []Message) string {
	if len(messages) == 0 {
		return ""
	}

	var summary strings.Builder
	summary.WriteString("[Previous conversation summary: ")

	for i, msg := range messages {
		if i > 0 {
			summary.WriteString(" | ")
		}
		// Truncate each message to 50 chars
		content := msg.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		summary.WriteString(fmt.Sprintf("%s: %s", msg.Role, content))

		// Cap total summary at 500 chars
		if summary.Len() > 500 {
			summary.WriteString(" ...]")
			return summary.String()
		}
	}

	summary.WriteString("]")

	// Final cap at 500 chars
	result := summary.String()
	if len(result) > 500 {
		return result[:497] + "...]"
	}
	return result
}

// YouTubeSearchResponse is the response for YouTube search
type YouTubeSearchResponse struct {
	Results []youtube.VideoInfo `json:"results"`
}

// YouTubeDownloadRequest is the request for downloading a YouTube audio
type YouTubeDownloadRequest struct {
	VideoID string `json:"video_id,omitempty"`
	Query   string `json:"query,omitempty"`
}

// YouTubeDownloadResponse is the response after downloading audio
type YouTubeDownloadResponse struct {
	FilePath string             `json:"file_path"`
	Format   string             `json:"format"`
	Duration int                `json:"duration"`
	Video    *youtube.VideoInfo `json:"video"`
}

// handleYouTubeSearch handles YouTube search requests
func (s *Server) handleYouTubeSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	maxResults := 5
	if max := r.URL.Query().Get("max"); max != "" {
		if n, err := fmt.Sscanf(max, "%d", &maxResults); err == nil && n == 1 {
			if maxResults < 1 {
				maxResults = 1
			}
			if maxResults > 10 {
				maxResults = 10
			}
		}
	}

	slog.Info("YouTube search", "query", query, "maxResults", maxResults)

	results, err := s.youtube.Search(r.Context(), query, maxResults)
	if err != nil {
		slog.Error("YouTube search failed", "error", err)
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(YouTubeSearchResponse{Results: results})
}

// handleYouTubeDownload downloads audio from a YouTube video
func (s *Server) handleYouTubeDownload(w http.ResponseWriter, r *http.Request) {
	var req YouTubeDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var result *youtube.DownloadResult
	var err error

	if req.VideoID != "" {
		// Download specific video by ID
		if !youtube.ValidateVideoID(req.VideoID) {
			http.Error(w, "Invalid video ID format", http.StatusBadRequest)
			return
		}

		slog.Info("Downloading YouTube audio", "videoID", req.VideoID)

		result, err = s.youtube.Download(r.Context(), req.VideoID, s.youtube.OutputDir())
		if err != nil {
			slog.Error("Failed to download audio", "error", err)
			http.Error(w, fmt.Sprintf("Failed to download audio: %v", err), http.StatusInternalServerError)
			return
		}

	} else if req.Query != "" {
		// Search and download
		slog.Info("Searching and downloading YouTube audio", "query", req.Query)

		result, err = s.youtube.DownloadByQuery(r.Context(), req.Query, s.youtube.OutputDir())
		if err != nil {
			slog.Error("Failed to search and download", "error", err)
			http.Error(w, fmt.Sprintf("Failed to download: %v", err), http.StatusInternalServerError)
			return
		}

	} else {
		http.Error(w, "Either 'video_id' or 'query' is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(YouTubeDownloadResponse{
		FilePath: result.FilePath,
		Format:   result.Format,
		Duration: result.Duration,
		Video:    result.Metadata,
	})
}

// RadioPlayRequest is the request body for radio play/queue
type RadioPlayRequest struct {
	Query string `json:"query"`
}

// handleRadioStatus returns the current radio status
func (s *Server) handleRadioStatus(w http.ResponseWriter, r *http.Request) {
	if s.radioEngine == nil {
		http.Error(w, "Radio not available", http.StatusServiceUnavailable)
		return
	}

	status, err := s.radioEngine.Status(r.Context())
	if err != nil {
		slog.Error("Failed to get radio status", "error", err)
		http.Error(w, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleRadioPlay plays a song immediately
func (s *Server) handleRadioPlay(w http.ResponseWriter, r *http.Request) {
	if s.radioEngine == nil {
		http.Error(w, "Radio not available", http.StatusServiceUnavailable)
		return
	}

	var req RadioPlayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	slog.Info("Radio play request", "query", req.Query)

	result, err := s.radioEngine.Play(r.Context(), req.Query)
	if err != nil {
		slog.Error("Failed to play", "error", err)
		http.Error(w, fmt.Sprintf("Failed to play: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRadioQueue adds a song to the queue
func (s *Server) handleRadioQueue(w http.ResponseWriter, r *http.Request) {
	if s.radioEngine == nil {
		http.Error(w, "Radio not available", http.StatusServiceUnavailable)
		return
	}

	var req RadioPlayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	slog.Info("Radio queue request", "query", req.Query)

	result, err := s.radioEngine.Queue(r.Context(), req.Query)
	if err != nil {
		slog.Error("Failed to queue", "error", err)
		http.Error(w, fmt.Sprintf("Failed to queue: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleRadioSkip skips the current song
func (s *Server) handleRadioSkip(w http.ResponseWriter, r *http.Request) {
	if s.radioEngine == nil {
		http.Error(w, "Radio not available", http.StatusServiceUnavailable)
		return
	}

	slog.Info("Radio skip request")

	err := s.radioEngine.Skip(r.Context())
	if err != nil {
		slog.Error("Failed to skip", "error", err)
		http.Error(w, fmt.Sprintf("Failed to skip: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "Skipped to next track",
	})
}

// handleRadioStream redirects to the Liquidsoap stream URL
func (s *Server) handleRadioStream(w http.ResponseWriter, r *http.Request) {
	if s.radioEngine == nil {
		http.Error(w, "Radio not available", http.StatusServiceUnavailable)
		return
	}

	streamURL := s.radioEngine.StreamURL()
	if streamURL == "" {
		http.Error(w, "Stream URL not available", http.StatusServiceUnavailable)
		return
	}

	http.Redirect(w, r, streamURL, http.StatusTemporaryRedirect)
}

// handleGetProviders returns available LLM providers
func (s *Server) handleGetProviders(w http.ResponseWriter, r *http.Request) {
	factory := llm.NewProviderFactory()
	providers := factory.GetAvailableProviders()

	response := map[string]interface{}{
		"providers": providers,
		"current": map[string]string{
			"provider": s.llmProvider.Name(),
			"model":    s.llmProvider.GetModel(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetProviderModels returns available models for a specific provider
func (s *Server) handleGetProviderModels(w http.ResponseWriter, r *http.Request) {
	providerName := r.PathValue("name")

	var provider llm.Provider
	var err error

	switch providerName {
	case "groq":
		apiKey := s.config.LLM.APIKey
		if apiKey == "" {
			http.Error(w, "Groq API key not configured", http.StatusBadRequest)
			return
		}
		provider, err = llm.NewGroqProvider(apiKey, "")
	case "cerebras":
		apiKey := os.Getenv("CEREBRAS_API_KEY")
		if apiKey == "" {
			http.Error(w, "Cerebras API key not configured", http.StatusBadRequest)
			return
		}
		provider = llm.NewOpenAIProvider("cerebras", "https://api.cerebras.ai/v1", apiKey, "dummy", 60*time.Second)
		err = nil
	case "ollama":
		baseURL := r.URL.Query().Get("base_url")
		if baseURL == "" {
			baseURL = s.config.LLM.BaseURL
			if baseURL == "" {
				baseURL = "http://localhost:11434/v1"
			}
		}
		provider, err = llm.NewOllamaProvider(baseURL, "dummy")
	case "lmstudio":
		baseURL := r.URL.Query().Get("base_url")
		if baseURL == "" {
			baseURL = s.config.LLM.BaseURL
			if baseURL == "" {
				baseURL = "http://localhost:1234/v1"
			}
		}
		provider, err = llm.NewLMStudioProvider(baseURL, "dummy")
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	models, err := provider.ListModels(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch models: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
}

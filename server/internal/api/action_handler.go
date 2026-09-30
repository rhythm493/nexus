package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/rhythm493/pocket-assistant/server/internal/cart"
	"github.com/rhythm493/pocket-assistant/server/internal/llm"
	"github.com/rhythm493/pocket-assistant/server/internal/mcp"
	"github.com/rhythm493/pocket-assistant/server/internal/radio"
	"github.com/rhythm493/pocket-assistant/server/internal/tools/results"
)

// ActionRequest is the request body for POST /api/v1/action
type ActionRequest struct {
	Action         string                 `json:"action"`
	Args           map[string]interface{} `json:"args"`
	ConversationID string                 `json:"conversation_id,omitempty"`
}

// handleAction executes a tool action (from an interactive UI component)
// and streams the result back via SSE.
//
// This endpoint is called by the Flutter app when the user interacts with
// an interactive component (button, form, etc.).
func (s *Server) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Action == "" {
		http.Error(w, "Action is required", http.StatusBadRequest)
		return
	}

	slog.Info("Action request", "action", req.Action, "args", req.Args)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	// Route the action to the appropriate tool handler (matching chat handler pattern)
	var result interface{}
	var err error

	if s.cartTools != nil && cart.IsCartTool(req.Action) {
		result, err = s.cartTools.ExecuteTool(ctx, req.ConversationID, req.Action, req.Args)
		if err != nil {
			slog.Error("Cart tool execution failed", "tool", req.Action, "error", err)
			result = map[string]interface{}{"error": err.Error()}
		} else if req.ConversationID != "" {
			if c := s.cartManager.GetCart(req.ConversationID); c != nil {
				s.sendSSE(w, flusher, SSEEvent{Type: "cart_update", Result: c.FullDetail()})
			}
		}
	} else if s.radioTools != nil && radio.IsRadioTool(req.Action) {
		result, err = s.radioTools.ExecuteTool(ctx, req.Action, req.Args)
		if err != nil {
			slog.Error("Radio tool execution failed", "tool", req.Action, "error", err)
			result = map[string]interface{}{"error": err.Error()}
		}
	} else {
		result, err = s.mcpHost.ExecuteTool(ctx, req.Action, req.Args)
		if err != nil {
			slog.Error("MCP tool execution failed", "tool", req.Action, "error", err)
			result = map[string]interface{}{"error": err.Error()}
		}
	}

	// Send tool result event with formatted UI blocks if available
	blocks := results.FormatResult(req.Action, result, req.Args)
	s.sendSSE(w, flusher, SSEEvent{
		Type:   "tool_result",
		Name:   req.Action,
		Result: result,
		Blocks: blocks,
	})

	// If result contains RichContentItem with component/image types, send ui_blocks
	if richItems, ok := result.([]mcp.RichContentItem); ok {
		var blocks []llm.ComponentBlock
		for _, item := range richItems {
			if item.Type == "component" {
				if dataMap, ok := item.Data.(map[string]interface{}); ok {
					compType, _ := dataMap["type"].(string)
					blocks = append(blocks, llm.ComponentBlock{Type: compType, Data: dataMap})
				}
			}
		}
		if len(blocks) > 0 {
			s.sendSSE(w, flusher, SSEEvent{Type: "ui_blocks", Blocks: blocks})
		}
	}

	s.sendSSE(w, flusher, SSEEvent{Type: "done"})
}

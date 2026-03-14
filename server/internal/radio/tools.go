package radio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/rhythm493/pocket-assistant/server/internal/library"
	"github.com/rhythm493/pocket-assistant/server/internal/youtube"
)

// MCPExecutor is an interface for executing MCP tools (avoids circular import with mcp package).
type MCPExecutor interface {
	ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

// ToolHandlers provides LLM tool handlers for radio functionality.
type ToolHandlers struct {
	engine      *Engine
	library     *library.Library
	youtube     *youtube.Service
	mcpExecutor MCPExecutor
	// cachedDevices stores discovered Sonos devices to avoid repeated discovery
	cachedDevices []string
}

// NewToolHandlers creates a new ToolHandlers instance.
func NewToolHandlers(engine *Engine, lib *library.Library, yt *youtube.Service, mcpExec MCPExecutor) *ToolHandlers {
	return &ToolHandlers{
		engine:      engine,
		library:     lib,
		youtube:     yt,
		mcpExecutor: mcpExec,
	}
}

// ToolDefinition represents a tool definition for the LLM.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// RadioPlayResult is the result of a radio_play tool call.
type RadioPlayResult struct {
	Success   bool           `json:"success"`
	Message   string         `json:"message"`
	Track     *library.Track `json:"track,omitempty"`
	Position  int            `json:"position,omitempty"`
	Cached    bool           `json:"cached,omitempty"`
	StreamURL string         `json:"stream_url,omitempty"`
}

// RadioQueueResult is the result of a radio_queue tool call.
type RadioQueueResult struct {
	Success  bool           `json:"success"`
	Message  string         `json:"message"`
	Track    *library.Track `json:"track,omitempty"`
	Position int            `json:"position,omitempty"`
	Cached   bool           `json:"cached,omitempty"`
}

// RadioSkipResult is the result of a radio_skip tool call.
type RadioSkipResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// RadioStatusResult is the result of a radio_status tool call.
type RadioStatusResult struct {
	Success      bool           `json:"success"`
	Running      bool           `json:"running"`
	CurrentTrack *library.Track `json:"current_track,omitempty"`
	QueueLength  int            `json:"queue_length"`
	Queue        []*QueueItem   `json:"queue,omitempty"`
	StreamURL    string         `json:"stream_url"`
	Message      string         `json:"message,omitempty"`
}

// LibrarySearchResult is the result of a library_search tool call.
type LibrarySearchResult struct {
	Success bool             `json:"success"`
	Count   int              `json:"count"`
	Tracks  []*library.Track `json:"tracks"`
	Message string           `json:"message,omitempty"`
}

// YouTubeSearchResult is the result of a youtube_search tool call.
type YouTubeSearchResult struct {
	Success bool                `json:"success"`
	Count   int                 `json:"count"`
	Videos  []youtube.VideoInfo `json:"videos"`
	Message string              `json:"message,omitempty"`
}

// RadioPlay searches for a track and plays it immediately.
func (h *ToolHandlers) RadioPlay(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return RadioPlayResult{
			Success: false,
			Message: "query parameter is required",
		}, nil
	}

	if len(query) > 200 {
		return RadioPlayResult{Success: false, Message: "Query too long (max 200 characters)"}, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return RadioPlayResult{Success: false, Message: "Query is empty after trimming"}, nil
	}

	if h.engine == nil {
		return RadioPlayResult{
			Success: false,
			Message: "radio engine is not available",
		}, nil
	}

	result, err := h.engine.Play(ctx, query)
	if err != nil {
		return RadioPlayResult{
			Success: false,
			Message: fmt.Sprintf("failed to play: %v", err),
		}, nil
	}

	// Get stream URL for Sonos - replace localhost with LAN IP
	streamURL := h.engine.StreamURL()
	if lanIP := getLANIP(); lanIP != "" {
		port := h.engine.Config().StreamPort
		streamURL = fmt.Sprintf("http://%s:%d/stream", lanIP, port)
	}

	// Auto-connect to Sonos if device specified or auto-discoverable
	device, _ := args["device"].(string)
	sonosConnected := false
	sonosMessage := ""

	if h.mcpExecutor != nil {
		if device == "" {
			// Auto-discover: find available devices
			device = h.discoverDefaultDevice(ctx)
		}

		if device != "" {
			title := result.Track.Title
			artist := result.Track.Artist
			if artist == "" {
				artist = "Unknown Artist"
			}

			sonosArgs := map[string]interface{}{
				"deviceId": device,
				"url":      streamURL,
				"title":    title,
				"artist":   artist,
			}

			_, err := h.mcpExecutor.ExecuteTool(ctx, "sonos_play_url", sonosArgs)
			if err != nil {
				slog.Warn("Failed to connect Sonos", "device", device, "error", err)
				sonosMessage = fmt.Sprintf("Stream is playing but failed to connect to Sonos '%s': %v. Stream URL: %s", device, err, streamURL)
			} else {
				sonosConnected = true
				sonosMessage = fmt.Sprintf("Playing on Sonos '%s'", device)
				slog.Info("Auto-connected Sonos", "device", device, "track", title)
			}
		}
	}

	msg := result.Message
	if sonosConnected {
		msg += " — " + sonosMessage
	} else if sonosMessage != "" {
		msg += " — " + sonosMessage
	}

	return RadioPlayResult{
		Success:   true,
		Message:   msg,
		Track:     result.Track,
		Position:  result.Position,
		Cached:    result.Cached,
		StreamURL: streamURL,
	}, nil
}

// RadioQueue adds a track to the end of the queue.
func (h *ToolHandlers) RadioQueue(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return RadioQueueResult{
			Success: false,
			Message: "query parameter is required",
		}, nil
	}

	if len(query) > 200 {
		return RadioQueueResult{Success: false, Message: "Query too long (max 200 characters)"}, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return RadioQueueResult{Success: false, Message: "Query is empty after trimming"}, nil
	}

	if h.engine == nil {
		return RadioQueueResult{
			Success: false,
			Message: "radio engine is not available",
		}, nil
	}

	result, err := h.engine.Queue(ctx, query)
	if err != nil {
		return RadioQueueResult{
			Success: false,
			Message: fmt.Sprintf("failed to queue: %v", err),
		}, nil
	}

	return RadioQueueResult{
		Success:  true,
		Message:  result.Message,
		Track:    result.Track,
		Position: result.Position,
		Cached:   result.Cached,
	}, nil
}

// RadioSkip skips to the next track.
func (h *ToolHandlers) RadioSkip(ctx context.Context, args map[string]any) (any, error) {
	if h.engine == nil {
		return RadioSkipResult{
			Success: false,
			Message: "radio engine is not available",
		}, nil
	}

	err := h.engine.Skip(ctx)
	if err != nil {
		return RadioSkipResult{
			Success: false,
			Message: fmt.Sprintf("failed to skip: %v", err),
		}, nil
	}

	return RadioSkipResult{
		Success: true,
		Message: "Skipped to next track",
	}, nil
}

// RadioStatus returns the current radio status.
func (h *ToolHandlers) RadioStatus(ctx context.Context, args map[string]any) (any, error) {
	if h.engine == nil {
		return RadioStatusResult{
			Success: false,
			Running: false,
			Message: "radio engine is not available",
		}, nil
	}

	status, err := h.engine.Status(ctx)
	if err != nil {
		return RadioStatusResult{
			Success: false,
			Running: h.engine.IsRunning(),
			Message: fmt.Sprintf("failed to get status: %v", err),
		}, nil
	}

	// Limit queue items to prevent context overflow
	maxQueueItems := 5
	queue := status.Queue
	if len(queue) > maxQueueItems {
		queue = queue[:maxQueueItems]
	}

	return RadioStatusResult{
		Success:      true,
		Running:      status.Running,
		CurrentTrack: status.CurrentTrack,
		QueueLength:  status.QueueLength,
		Queue:        queue,
		StreamURL:    status.StreamURL,
	}, nil
}

// LibrarySearch searches for tracks in the local library.
func (h *ToolHandlers) LibrarySearch(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return LibrarySearchResult{
			Success: false,
			Message: "query parameter is required",
		}, nil
	}

	if len(query) > 200 {
		return LibrarySearchResult{Success: false, Message: "Query too long (max 200 characters)"}, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return LibrarySearchResult{Success: false, Message: "Query is empty after trimming"}, nil
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit < 1 {
			limit = 1
		}
		if limit > 50 {
			limit = 50
		}
	}

	if h.library == nil {
		return LibrarySearchResult{
			Success: false,
			Message: "library is not available",
		}, nil
	}

	tracks, err := h.library.Search(ctx, query, limit)
	if err != nil {
		return LibrarySearchResult{
			Success: false,
			Message: fmt.Sprintf("search failed: %v", err),
		}, nil
	}

	return LibrarySearchResult{
		Success: true,
		Count:   len(tracks),
		Tracks:  tracks,
	}, nil
}

// YouTubeSearch searches YouTube for videos.
func (h *ToolHandlers) YouTubeSearch(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return YouTubeSearchResult{
			Success: false,
			Message: "query parameter is required",
		}, nil
	}

	if len(query) > 200 {
		return YouTubeSearchResult{Success: false, Message: "Query too long (max 200 characters)"}, nil
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return YouTubeSearchResult{Success: false, Message: "Query is empty after trimming"}, nil
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit < 1 {
			limit = 1
		}
		if limit > 10 {
			limit = 10
		}
	}

	if h.youtube == nil {
		return YouTubeSearchResult{
			Success: false,
			Message: "youtube service is not available",
		}, nil
	}

	videos, err := h.youtube.Search(ctx, query, limit)
	if err != nil {
		return YouTubeSearchResult{
			Success: false,
			Message: fmt.Sprintf("search failed: %v", err),
		}, nil
	}

	return YouTubeSearchResult{
		Success: true,
		Count:   len(videos),
		Videos:  videos,
	}, nil
}

// ToolDefinitions returns the tool definitions for the LLM.
func (h *ToolHandlers) ToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "radio_play",
			Description: "Search for a song (checks local library first, then YouTube), download if needed, and play it. Automatically connects to Sonos speakers — if only one speaker exists, it connects automatically. Use this when the user wants to hear music. This is the PRIMARY tool for playing music.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Song title and/or artist name to search for. Examples: 'Shape of You Ed Sheeran', 'Blinding Lights', 'Avicii'. Case-insensitive.",
					},
					"device": map[string]any{
						"type":        "string",
						"description": "Sonos speaker name to play on (e.g., 'Living Room'). Optional — if omitted, auto-discovers and uses the available speaker.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "radio_queue",
			Description: "Search for a song and add it to the end of the radio queue without interrupting current playback. Use this when the user wants to add songs for later, or when building a playlist. Returns the queued track info and queue position.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Song title and/or artist name to search for. Examples: 'Shape of You Ed Sheeran', 'Blinding Lights', 'Avicii'. Case-insensitive.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "radio_skip",
			Description: "Skip the currently playing song and move to the next track in the queue with a crossfade transition. Use this when the user wants to skip or change the current song. No parameters needed.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "radio_status",
			Description: "Get the current radio status: what's playing now, upcoming queue (up to 5 tracks), and stream URL. Use this to check what's currently playing before making changes.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "library_search",
			Description: "Search the local downloaded music library using full-text search. Use this to check if a song is already downloaded before calling radio_play. Returns matching tracks with metadata. Does NOT play anything.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Song title and/or artist name to search for. Examples: 'Shape of You Ed Sheeran', 'Blinding Lights', 'Avicii'. Case-insensitive.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (default: 10, max: 50)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "youtube_search",
			Description: "Search YouTube for videos matching the query. Use this to show the user search results or to find specific videos. Does NOT download or play anything. For playing music, use radio_play instead.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Song title and/or artist name to search for. Examples: 'Shape of You Ed Sheeran', 'Blinding Lights', 'Avicii'. Case-insensitive.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of results (default: 5, max: 10)",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// GetToolHandler returns a handler function for the given tool name.
func (h *ToolHandlers) GetToolHandler(name string) func(context.Context, map[string]any) (any, error) {
	switch name {
	case "radio_play":
		return h.RadioPlay
	case "radio_queue":
		return h.RadioQueue
	case "radio_skip":
		return h.RadioSkip
	case "radio_status":
		return h.RadioStatus
	case "library_search":
		return h.LibrarySearch
	case "youtube_search":
		return h.YouTubeSearch
	default:
		return nil
	}
}

// ExecuteTool executes a tool by name with the given arguments.
func (h *ToolHandlers) ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error) {
	handler := h.GetToolHandler(name)
	if handler == nil {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, args)
}

// IsRadioTool returns true if the tool name is a radio tool.
func IsRadioTool(name string) bool {
	switch name {
	case "radio_play", "radio_queue", "radio_skip", "radio_status", "library_search", "youtube_search":
		return true
	default:
		return false
	}
}

// discoverDefaultDevice finds the default Sonos device to play on.
// Uses cached devices if available, otherwise discovers via MCP.
// If only one device exists, returns it. If multiple, returns empty (LLM should ask user).
func (h *ToolHandlers) discoverDefaultDevice(ctx context.Context) string {
	// Return cached if available
	if len(h.cachedDevices) == 1 {
		return h.cachedDevices[0]
	}
	if len(h.cachedDevices) > 1 {
		// Multiple devices — use the first one as default
		return h.cachedDevices[0]
	}

	// Discover via MCP
	if h.mcpExecutor == nil {
		return ""
	}

	result, err := h.mcpExecutor.ExecuteTool(ctx, "sonos_list_devices", map[string]interface{}{})
	if err != nil {
		slog.Warn("Failed to discover Sonos devices", "error", err)
		return ""
	}

	// Parse the result — it's a []interface{} of device objects
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return ""
	}

	var devices []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(resultJSON, &devices); err != nil {
		slog.Debug("Failed to parse device list", "error", err)
		return ""
	}

	// Cache the device names
	h.cachedDevices = make([]string, 0, len(devices))
	for _, d := range devices {
		if d.Name != "" {
			h.cachedDevices = append(h.cachedDevices, d.Name)
		}
	}

	slog.Info("Discovered Sonos devices", "count", len(h.cachedDevices), "devices", h.cachedDevices)

	if len(h.cachedDevices) >= 1 {
		return h.cachedDevices[0]
	}
	return ""
}

// getLANIP returns the first LAN IPv4 address of the machine.
// Returns empty string if no LAN IP is found.
func getLANIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		slog.Debug("Failed to get network interfaces", "error", err)
		return ""
	}

	for _, iface := range interfaces {
		// Skip loopback, down, and known virtual interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			ip4 := ip.To4()
			if ip4 == nil {
				continue
			}

			// 192.168.x.x or 10.x.x.x (common LAN ranges)
			if ip4[0] == 192 && ip4[1] == 168 {
				return ip.String()
			}
			if ip4[0] == 10 {
				return ip.String()
			}
			// Skip all 172.x.x.x — too often Docker/container IPs
			// LAN should use 192.168.x.x or 10.x.x.x
		}
	}

	return ""
}

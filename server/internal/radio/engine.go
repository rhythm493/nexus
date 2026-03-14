package radio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rhythm493/pocket-assistant/server/internal/library"
	"github.com/rhythm493/pocket-assistant/server/internal/youtube"
)

// Common engine errors
var (
	ErrNotRunning    = errors.New("radio is not running")
	ErrAlreadyRunning = errors.New("radio is already running")
	ErrNoResults     = errors.New("no results found for query")
)

// Config holds the radio engine configuration.
type Config struct {
	LiquidsoapPath string        // Path to liquidsoap binary
	ScriptPath     string        // Path to radio.liq script
	TelnetHost     string        // Telnet host (default: 127.0.0.1)
	TelnetPort     int           // Telnet port (default: 1234)
	StreamPort     int           // HTTP stream port (default: 8080)
	StreamHost     string        // External hostname for stream URL
	CrossfadeSecs  float64       // Crossfade duration in seconds
	AutoStart      bool          // Start Liquidsoap automatically
	StartupTimeout time.Duration // Timeout for Liquidsoap startup
}

// DefaultConfig returns the default radio configuration.
func DefaultConfig() Config {
	return Config{
		LiquidsoapPath: "liquidsoap",
		ScriptPath:     "scripts/radio.liq",
		TelnetHost:     "127.0.0.1",
		TelnetPort:     1234,
		StreamPort:     8080,
		StreamHost:     "localhost",
		CrossfadeSecs:  3.0,
		AutoStart:      true,
		StartupTimeout: 10 * time.Second,
	}
}

// PlayResult contains the result of a Play or Queue operation.
type PlayResult struct {
	Track    *library.Track `json:"track"`
	Position int            `json:"position"`
	Message  string         `json:"message"`
	Cached   bool           `json:"cached"` // True if track was in library
}

// RadioStatus contains the current radio status.
type RadioStatus struct {
	Running      bool         `json:"running"`
	CurrentTrack *library.Track `json:"current_track,omitempty"`
	Queue        []*QueueItem `json:"queue"`
	QueueLength  int          `json:"queue_length"`
	StreamURL    string       `json:"stream_url"`
	Uptime       string       `json:"uptime,omitempty"`
	Version      string       `json:"version,omitempty"`
}

// Engine orchestrates the radio functionality.
type Engine struct {
	cfg       Config
	library   *library.Library
	youtube   *youtube.Service
	liq       *Client
	queue     *Queue
	process   *exec.Cmd
	streamURL string
	startedAt time.Time
	running   bool
	mu        sync.RWMutex
}

// New creates a new radio engine.
func New(cfg Config, lib *library.Library, yt *youtube.Service) *Engine {
	// Apply defaults
	if cfg.TelnetHost == "" {
		cfg.TelnetHost = "127.0.0.1"
	}
	if cfg.TelnetPort == 0 {
		cfg.TelnetPort = 1234
	}
	if cfg.StreamPort == 0 {
		cfg.StreamPort = 8080
	}
	if cfg.StreamHost == "" {
		cfg.StreamHost = "localhost"
	}
	if cfg.StartupTimeout == 0 {
		cfg.StartupTimeout = 10 * time.Second
	}

	// Build stream URL
	streamURL := fmt.Sprintf("http://%s:%d/stream", cfg.StreamHost, cfg.StreamPort)

	return &Engine{
		cfg:       cfg,
		library:   lib,
		youtube:   yt,
		liq:       NewClient(cfg.TelnetHost, cfg.TelnetPort),
		queue:     NewQueue(),
		streamURL: streamURL,
	}
}

// Start starts the Liquidsoap process and connects to it.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return ErrAlreadyRunning
	}

	slog.Info("Starting radio engine",
		"script", e.cfg.ScriptPath,
		"telnet", fmt.Sprintf("%s:%d", e.cfg.TelnetHost, e.cfg.TelnetPort),
		"stream", e.streamURL,
	)

	// Verify script exists
	if _, err := os.Stat(e.cfg.ScriptPath); err != nil {
		return fmt.Errorf("liquidsoap script not found: %s: %w", e.cfg.ScriptPath, err)
	}

	// Find Liquidsoap binary
	liquidsoapPath, err := exec.LookPath(e.cfg.LiquidsoapPath)
	if err != nil {
		return fmt.Errorf("liquidsoap binary not found: %w", err)
	}

	// Start Liquidsoap process
	scriptPath, err := filepath.Abs(e.cfg.ScriptPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute script path: %w", err)
	}

	e.process = exec.Command(liquidsoapPath, scriptPath)
	e.process.Stdout = os.Stdout
	e.process.Stderr = os.Stderr

	if err := e.process.Start(); err != nil {
		return fmt.Errorf("failed to start liquidsoap: %w", err)
	}

	slog.Info("Liquidsoap process started", "pid", e.process.Process.Pid)

	// Wait for Liquidsoap to be ready and connect
	connectCtx, cancel := context.WithTimeout(ctx, e.cfg.StartupTimeout)
	defer cancel()

	if err := e.waitForLiquidsoap(connectCtx); err != nil {
		// Kill the process if we can't connect
		e.killProcess()
		return fmt.Errorf("failed to connect to liquidsoap: %w", err)
	}

	e.running = true
	e.startedAt = time.Now()

	slog.Info("Radio engine started successfully")
	return nil
}

// waitForLiquidsoap waits for Liquidsoap to be ready and connects.
func (e *Engine) waitForLiquidsoap(ctx context.Context) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Try to connect
			if err := e.liq.Connect(ctx); err == nil {
				// Verify connection works
				if _, err := e.liq.Status(); err == nil {
					return nil
				}
				e.liq.Close()
			}
		}
	}
}

// Stop stops the Liquidsoap process.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrNotRunning
	}

	slog.Info("Stopping radio engine")

	// Close telnet connection
	if err := e.liq.Close(); err != nil {
		slog.Warn("Error closing liquidsoap connection", "error", err)
	}

	// Stop the process
	if e.process != nil && e.process.Process != nil {
		// Try graceful shutdown first
		if err := e.process.Process.Signal(os.Interrupt); err != nil {
			slog.Debug("Failed to send interrupt to Liquidsoap, will kill", "error", err)
			e.killProcess()
		} else {
			// Wait for process to exit (with timeout)
			done := make(chan error, 1)
			go func() {
				done <- e.process.Wait()
			}()

			select {
			case <-done:
				e.process = nil
			case <-time.After(5 * time.Second):
				slog.Warn("Liquidsoap did not exit gracefully, killing")
				e.killProcess()
			}
		}
	}

	e.running = false
	e.queue.Clear()

	slog.Info("Radio engine stopped")
	return nil
}

// IsRunning returns true if the radio is running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// Play searches for a track and starts playing it immediately.
// If there's currently a track playing, it will skip to this one.
func (e *Engine) Play(ctx context.Context, query string) (*PlayResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil, ErrNotRunning
	}

	slog.Info("Radio play request", "query", query)

	// Find or download the track
	track, cached, err := e.findOrDownload(ctx, query)
	if err != nil {
		return nil, err
	}

	// Clear the queue and push this track
	if err := e.liq.Flush(); err != nil {
		slog.Warn("Failed to flush queue", "error", err)
	}
	e.queue.Clear()

	// Push to Liquidsoap
	requestID, err := e.liq.Push(track.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to push track to liquidsoap: %w", err)
	}

	// Add to our queue
	position := e.queue.AddWithRequestID(track, requestID)

	// Update play count in library
	go func() {
		if err := e.library.IncrementPlayCount(context.Background(), track.ID); err != nil {
			slog.Warn("Failed to increment play count", "trackID", track.ID, "error", err)
		}
	}()

	result := &PlayResult{
		Track:    track,
		Position: position,
		Cached:   cached,
	}

	if cached {
		result.Message = fmt.Sprintf("Now playing: %s by %s (from library)", track.Title, track.Artist)
	} else {
		result.Message = fmt.Sprintf("Now playing: %s by %s (downloaded)", track.Title, track.Artist)
	}

	slog.Info("Radio playing track",
		"title", track.Title,
		"artist", track.Artist,
		"cached", cached,
	)

	return result, nil
}

// Queue adds a track to the end of the queue.
func (e *Engine) Queue(ctx context.Context, query string) (*PlayResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil, ErrNotRunning
	}

	slog.Info("Radio queue request", "query", query)

	// Find or download the track
	track, cached, err := e.findOrDownload(ctx, query)
	if err != nil {
		return nil, err
	}

	// Push to Liquidsoap
	requestID, err := e.liq.Push(track.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to push track to liquidsoap: %w", err)
	}

	// Add to our queue
	position := e.queue.AddWithRequestID(track, requestID)

	result := &PlayResult{
		Track:    track,
		Position: position,
		Cached:   cached,
	}

	if cached {
		result.Message = fmt.Sprintf("Added to queue (position %d): %s by %s", position, track.Title, track.Artist)
	} else {
		result.Message = fmt.Sprintf("Downloaded and added to queue (position %d): %s by %s", position, track.Title, track.Artist)
	}

	slog.Info("Track added to queue",
		"title", track.Title,
		"artist", track.Artist,
		"position", position,
		"cached", cached,
	)

	return result, nil
}

// Skip skips to the next track.
func (e *Engine) Skip(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrNotRunning
	}

	slog.Info("Skipping current track")

	if err := e.liq.Skip(); err != nil {
		return fmt.Errorf("failed to skip: %w", err)
	}

	// Advance our queue
	e.queue.Advance()

	return nil
}

// Status returns the current radio status.
func (e *Engine) Status(ctx context.Context) (*RadioStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := &RadioStatus{
		Running:   e.running,
		StreamURL: e.streamURL,
		Queue:     e.queue.Upcoming(),
	}

	if !e.running {
		return status, nil
	}

	// Get current track
	current := e.queue.Current()
	if current != nil {
		status.CurrentTrack = current.Track
	}

	// Get queue length
	status.QueueLength = e.queue.RemainingLen()

	// Get Liquidsoap status
	liqStatus, err := e.liq.Status()
	if err == nil && liqStatus != nil {
		status.Uptime = liqStatus.Uptime
		status.Version = liqStatus.Version
	}

	return status, nil
}

// StreamURL returns the stream URL.
func (e *Engine) StreamURL() string {
	return e.streamURL
}

// Config returns the engine configuration.
func (e *Engine) Config() Config {
	return e.cfg
}

// killProcess forcefully kills the Liquidsoap process and reaps the zombie.
// Must only be called when e.process is non-nil.
func (e *Engine) killProcess() {
	if e.process == nil || e.process.Process == nil {
		return
	}

	if err := e.process.Process.Kill(); err != nil {
		slog.Error("Failed to kill Liquidsoap process", "error", err)
	}

	// Reap zombie process
	done := make(chan error, 1)
	go func() { done <- e.process.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		slog.Warn("Liquidsoap process did not exit after kill")
	}

	e.process = nil
}

// findOrDownload finds a track in the library or downloads it from YouTube.
// Returns the track, whether it was cached (in library), and any error.
func (e *Engine) findOrDownload(ctx context.Context, query string) (*library.Track, bool, error) {
	// First, search the library
	tracks, err := e.library.Search(ctx, query, 1)
	if err != nil {
		slog.Debug("Library search failed", "query", query, "error", err)
	} else if len(tracks) > 0 {
		slog.Info("Found track in library",
			"title", tracks[0].Title,
			"artist", tracks[0].Artist,
		)
		return tracks[0], true, nil
	}

	// Not in library, search YouTube
	slog.Info("Track not in library, searching YouTube", "query", query)

	videos, err := e.youtube.Search(ctx, query, 1)
	if err != nil {
		return nil, false, fmt.Errorf("youtube search failed: %w", err)
	}
	if len(videos) == 0 {
		return nil, false, ErrNoResults
	}

	video := videos[0]
	slog.Info("Found on YouTube",
		"title", video.Title,
		"artist", video.Artist,
		"videoID", video.ID,
	)

	// Check if we already have this video downloaded
	existing, err := e.library.GetByYouTubeID(ctx, video.ID)
	if err != nil {
		slog.Debug("Failed to check library by YouTube ID", "videoID", video.ID, "error", err)
	} else if existing != nil {
		slog.Info("Video already in library",
			"title", existing.Title,
			"youtubeID", video.ID,
		)
		return existing, true, nil
	}

	// Download from YouTube
	result, err := e.youtube.Download(ctx, video.ID, e.youtube.OutputDir())
	if err != nil {
		return nil, false, fmt.Errorf("download failed: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(result.FilePath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	// Fallback for empty metadata
	artist := video.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}
	title := video.Title
	if title == "" {
		title = query
	}

	// Create track
	track := &library.Track{
		ID:           uuid.New().String(),
		YouTubeID:    video.ID,
		Title:        title,
		Artist:       artist,
		Duration:     result.Duration,
		FilePath:     result.FilePath,
		FileSize:     fileInfo.Size(),
		Format:       result.Format,
		Thumbnail:    video.Thumbnail,
		DownloadedAt: time.Now(),
	}

	// Add to library
	if err := e.library.AddTrack(ctx, track); err != nil {
		// Log but don't fail - we can still play the file
		slog.Warn("Failed to add track to library", "error", err)
	}

	return track, false, nil
}

// ClearQueue clears all upcoming tracks from the queue.
func (e *Engine) ClearQueue(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return ErrNotRunning
	}

	if err := e.liq.Flush(); err != nil {
		return fmt.Errorf("failed to flush liquidsoap queue: %w", err)
	}

	e.queue.Clear()
	slog.Info("Queue cleared")

	return nil
}

// GetQueue returns the current queue.
func (e *Engine) GetQueue() []*QueueItem {
	return e.queue.Upcoming()
}

// CurrentTrack returns the currently playing track.
func (e *Engine) CurrentTrack() *library.Track {
	current := e.queue.Current()
	if current == nil {
		return nil
	}
	return current.Track
}

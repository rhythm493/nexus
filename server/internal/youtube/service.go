package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/rhythm493/pocket-assistant/server/internal/binutil"
)

// Service handles YouTube search and audio download
type Service struct {
	ytdlpPath string
	outputDir string
}

// VideoInfo contains metadata about a YouTube video
type VideoInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`      // Channel name or extracted artist
	Channel     string `json:"channel"`     // YouTube channel name
	Duration    int    `json:"duration"`    // Duration in seconds
	Thumbnail   string `json:"thumbnail"`   // Thumbnail URL
	Description string `json:"description"` // Video description
}

// ytdlpThumbnail represents a thumbnail from yt-dlp
type ytdlpThumbnail struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// ytdlpOutput represents the JSON output from yt-dlp
type ytdlpOutput struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Channel     string           `json:"channel"`
	Uploader    string           `json:"uploader"`
	Duration    float64          `json:"duration"`
	Thumbnail   string           `json:"thumbnail"`   // Full video info
	Thumbnails  []ytdlpThumbnail `json:"thumbnails"`  // Flat playlist search
	Description string           `json:"description"`
	Track       string           `json:"track"`
	Artist      string           `json:"artist"`
	Album       string           `json:"album"`
	WebpageURL  string           `json:"webpage_url"`
}

// Config holds YouTube service configuration
type Config struct {
	BinDir    string // Directory for yt-dlp binary (default: data/bin)
	OutputDir string // Directory for downloaded audio files
}

// NewService creates a new YouTube service.
// It ensures yt-dlp is available, downloading it if necessary.
func NewService(cfg Config) (*Service, error) {
	// Set defaults
	if cfg.BinDir == "" {
		cfg.BinDir = "data/bin"
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "data/audio"
	}

	// Ensure yt-dlp binary exists
	ytdlpPath, err := binutil.EnsureYtdlp(cfg.BinDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure yt-dlp: %w", err)
	}

	// Log yt-dlp version
	if version, err := binutil.GetVersion(ytdlpPath); err == nil {
		slog.Info("YouTube service initialized",
			"ytdlp", ytdlpPath,
			"version", version,
			"outputDir", cfg.OutputDir,
		)
	} else {
		slog.Info("YouTube service initialized",
			"ytdlp", ytdlpPath,
			"outputDir", cfg.OutputDir,
		)
	}

	return &Service{
		ytdlpPath: ytdlpPath,
		outputDir: cfg.OutputDir,
	}, nil
}

// Search searches YouTube for videos matching the query
func (s *Service) Search(ctx context.Context, query string, maxResults int) ([]VideoInfo, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	slog.Debug("Searching YouTube", "query", query, "maxResults", maxResults)

	// Use yt-dlp to search YouTube
	// ytsearch<N>: prefix searches YouTube and returns N results
	args := []string{
		fmt.Sprintf("ytsearch%d:%s", maxResults, query),
		"--dump-json",
		"--no-playlist",
		"--flat-playlist",
		"--no-warnings",
		"--ignore-errors",
	}

	cmd := exec.CommandContext(ctx, s.ytdlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp search failed: %w", err)
	}

	// Parse JSON output (one JSON object per line)
	var results []VideoInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var ytInfo ytdlpOutput
		if err := json.Unmarshal([]byte(line), &ytInfo); err != nil {
			slog.Debug("Failed to parse yt-dlp output line", "error", err)
			continue
		}

		// Determine artist - prefer track artist, fall back to channel
		artist := ytInfo.Artist
		if artist == "" {
			artist = ytInfo.Channel
			if artist == "" {
				artist = ytInfo.Uploader
			}
		}

		// Use track title if available, otherwise video title
		title := ytInfo.Track
		if title == "" {
			title = ytInfo.Title
		}

		// Get thumbnail - from Thumbnail or first in Thumbnails array
		thumbnail := ytInfo.Thumbnail
		if thumbnail == "" && len(ytInfo.Thumbnails) > 0 {
			thumbnail = ytInfo.Thumbnails[0].URL
		}

		results = append(results, VideoInfo{
			ID:          ytInfo.ID,
			Title:       title,
			Artist:      artist,
			Channel:     ytInfo.Channel,
			Duration:    int(ytInfo.Duration),
			Thumbnail:   thumbnail,
			Description: ytInfo.Description,
		})
	}

	slog.Debug("YouTube search complete", "results", len(results))
	return results, nil
}

// GetVideoInfo retrieves detailed metadata for a specific video
func (s *Service) GetVideoInfo(ctx context.Context, videoID string) (*VideoInfo, error) {
	slog.Debug("Getting video info", "videoID", videoID)

	args := []string{
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
	}

	cmd := exec.CommandContext(ctx, s.ytdlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp info failed: %w", err)
	}

	var ytInfo ytdlpOutput
	if err := json.Unmarshal(output, &ytInfo); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	// Determine artist
	artist := ytInfo.Artist
	if artist == "" {
		artist = ytInfo.Channel
		if artist == "" {
			artist = ytInfo.Uploader
		}
	}

	// Use track title if available
	title := ytInfo.Track
	if title == "" {
		title = ytInfo.Title
	}

	// Get thumbnail - from Thumbnail or first in Thumbnails array
	thumbnail := ytInfo.Thumbnail
	if thumbnail == "" && len(ytInfo.Thumbnails) > 0 {
		thumbnail = ytInfo.Thumbnails[0].URL
	}

	return &VideoInfo{
		ID:          ytInfo.ID,
		Title:       title,
		Artist:      artist,
		Channel:     ytInfo.Channel,
		Duration:    int(ytInfo.Duration),
		Thumbnail:   thumbnail,
		Description: ytInfo.Description,
	}, nil
}

// YtdlpPath returns the path to the yt-dlp binary
func (s *Service) YtdlpPath() string {
	return s.ytdlpPath
}

// OutputDir returns the output directory for downloads
func (s *Service) OutputDir() string {
	return s.outputDir
}

// UpdateYtdlp updates yt-dlp to the latest version
func (s *Service) UpdateYtdlp() error {
	return binutil.UpdateYtdlp(s.ytdlpPath)
}

// GetYtdlpVersion returns the current yt-dlp version
func (s *Service) GetYtdlpVersion() (string, error) {
	return binutil.GetVersion(s.ytdlpPath)
}

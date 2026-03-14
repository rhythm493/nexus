package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DownloadResult contains the result of a YouTube audio download.
type DownloadResult struct {
	FilePath string     // Path to the downloaded file
	Format   string     // Audio format (opus, m4a, webm, etc.)
	Duration int        // Duration in seconds
	Metadata *VideoInfo // Video metadata
}

// ytdlpDownloadOutput represents the JSON output from yt-dlp during download.
type ytdlpDownloadOutput struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Channel     string  `json:"channel"`
	Uploader    string  `json:"uploader"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`
	Description string  `json:"description"`
	Track       string  `json:"track"`
	Artist      string  `json:"artist"`
	Album       string  `json:"album"`
	Ext         string  `json:"ext"`      // File extension
	Filename    string  `json:"filename"` // Full path to downloaded file
	Filesize    int64   `json:"filesize"` // File size in bytes
}

// Download downloads audio from a YouTube video and converts to mp3.
// It uses yt-dlp with '-x --audio-format mp3' for Liquidsoap compatibility.
// Returns the download result with file path and metadata.
func (s *Service) Download(ctx context.Context, videoID, outputDir string) (*DownloadResult, error) {
	slog.Info("Downloading audio", "videoID", videoID, "outputDir", outputDir)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build output template - yt-dlp will fill in the extension
	outputTemplate := filepath.Join(outputDir, fmt.Sprintf("%s.%%(ext)s", videoID))

	// yt-dlp command:
	// -x --audio-format mp3 - extract audio and convert to mp3
	// -o template - output path template
	// --print-json - print metadata as JSON
	// --no-playlist - don't download playlists
	// --no-progress - no progress bar (cleaner output)
	args := []string{
		"-x", "--audio-format", "mp3",
		"-o", outputTemplate,
		"--print-json",
		"--no-playlist",
		"--no-progress",
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
	}

	slog.Debug("Running yt-dlp", "args", args)

	// Create command with timeout context
	downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(downloadCtx, s.ytdlpPath, args...)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	output, err := cmd.Output()
	if err != nil {
		stderrStr := strings.TrimSpace(stderrBuf.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("yt-dlp download failed: %w\nstderr: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("yt-dlp download failed: %w", err)
	}

	// Parse JSON output
	var ytInfo ytdlpDownloadOutput
	if err := json.Unmarshal(output, &ytInfo); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	// Determine the actual file path
	filePath := ytInfo.Filename
	if filePath == "" {
		// Fall back to constructing the path from videoID and extension
		filePath = filepath.Join(outputDir, fmt.Sprintf("%s.%s", videoID, ytInfo.Ext))
	}

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("downloaded file not found at %s: %w", filePath, err)
	}

	// Build metadata
	artist := ytInfo.Artist
	if artist == "" {
		artist = ytInfo.Channel
		if artist == "" {
			artist = ytInfo.Uploader
		}
	}

	title := ytInfo.Track
	if title == "" {
		title = ytInfo.Title
	}

	metadata := &VideoInfo{
		ID:          ytInfo.ID,
		Title:       title,
		Artist:      artist,
		Channel:     ytInfo.Channel,
		Duration:    int(ytInfo.Duration),
		Thumbnail:   ytInfo.Thumbnail,
		Description: ytInfo.Description,
	}

	result := &DownloadResult{
		FilePath: filePath,
		Format:   ytInfo.Ext,
		Duration: int(ytInfo.Duration),
		Metadata: metadata,
	}

	slog.Info("Download complete",
		"videoID", videoID,
		"format", result.Format,
		"path", result.FilePath,
		"duration", result.Duration,
	)

	return result, nil
}

// DownloadByQuery searches for a video and downloads its audio.
// This is a convenience method that combines search + download.
func (s *Service) DownloadByQuery(ctx context.Context, query, outputDir string) (*DownloadResult, error) {
	// Search for the video
	results, err := s.Search(ctx, query, 1)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query: %s", query)
	}

	video := results[0]

	// Download the audio
	return s.Download(ctx, video.ID, outputDir)
}

// GetDownloadedFile checks if a video has already been downloaded.
// Returns the file path if found, empty string otherwise.
func GetDownloadedFile(videoID, outputDir string) string {
	// Check common audio extensions
	extensions := []string{"opus", "m4a", "webm", "ogg", "mp3"}

	for _, ext := range extensions {
		path := filepath.Join(outputDir, fmt.Sprintf("%s.%s", videoID, ext))
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// CleanupDownload removes a downloaded file.
func CleanupDownload(filePath string) error {
	if filePath == "" {
		return nil
	}
	return os.Remove(filePath)
}

// ValidateVideoID checks if a string looks like a valid YouTube video ID.
func ValidateVideoID(videoID string) bool {
	// YouTube video IDs are 11 characters, alphanumeric plus - and _
	if len(videoID) != 11 {
		return false
	}

	for _, c := range videoID {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_') {
			return false
		}
	}

	return true
}

// ExtractVideoID extracts a video ID from a YouTube URL.
func ExtractVideoID(input string) string {
	// Handle direct video IDs
	if ValidateVideoID(input) {
		return input
	}

	// Handle various YouTube URL formats
	input = strings.TrimSpace(input)

	// Check for youtu.be format
	if strings.Contains(input, "youtu.be/") {
		parts := strings.Split(input, "youtu.be/")
		if len(parts) >= 2 {
			id := strings.Split(parts[1], "?")[0]
			id = strings.Split(id, "&")[0]
			id = strings.Split(id, "#")[0]
			if ValidateVideoID(id) {
				return id
			}
		}
	}

	// Check for youtube.com formats
	if strings.Contains(input, "youtube.com") {
		// Try v= parameter
		if strings.Contains(input, "v=") {
			parts := strings.Split(input, "v=")
			if len(parts) >= 2 {
				id := strings.Split(parts[1], "&")[0]
				id = strings.Split(id, "#")[0]
				if ValidateVideoID(id) {
					return id
				}
			}
		}

		// Try /embed/ format
		if strings.Contains(input, "/embed/") {
			parts := strings.Split(input, "/embed/")
			if len(parts) >= 2 {
				id := strings.Split(parts[1], "?")[0]
				id = strings.Split(id, "&")[0]
				if ValidateVideoID(id) {
					return id
				}
			}
		}

		// Try /v/ format
		if strings.Contains(input, "/v/") {
			parts := strings.Split(input, "/v/")
			if len(parts) >= 2 {
				id := strings.Split(parts[1], "?")[0]
				id = strings.Split(id, "&")[0]
				if ValidateVideoID(id) {
					return id
				}
			}
		}
	}

	return ""
}

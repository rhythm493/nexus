package binutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	// YtdlpURL is the download URL for the latest yt-dlp release
	YtdlpURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	// DefaultPath is the default location for the yt-dlp binary
	DefaultPath = "data/bin/yt-dlp"
)

// EnsureYtdlp ensures yt-dlp binary exists, downloading it if necessary.
// It returns the path to the binary.
func EnsureYtdlp(binDir string) (string, error) {
	binPath := filepath.Join(binDir, "yt-dlp")

	// Check if binary already exists and is executable
	if info, err := os.Stat(binPath); err == nil {
		if info.Mode()&0111 != 0 {
			slog.Debug("yt-dlp binary exists", "path", binPath)
			return binPath, nil
		}
		// Make it executable if it exists but isn't executable
		if err := os.Chmod(binPath, 0755); err != nil {
			return "", fmt.Errorf("failed to make yt-dlp executable: %w", err)
		}
		return binPath, nil
	}

	// Binary doesn't exist, download it
	slog.Info("yt-dlp not found, downloading...", "dest", binPath)
	if err := downloadYtdlp(binPath); err != nil {
		return "", err
	}

	return binPath, nil
}

// UpdateYtdlp updates yt-dlp to the latest version.
func UpdateYtdlp(binPath string) error {
	// First try using yt-dlp's built-in update
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "-U")
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Debug("yt-dlp -U failed, downloading fresh copy", "error", err, "output", string(output))
		// Fall back to downloading fresh copy
		return downloadYtdlp(binPath)
	}

	slog.Info("yt-dlp updated successfully", "output", strings.TrimSpace(string(output)))
	return nil
}

// GetVersion returns the version of yt-dlp.
func GetVersion(binPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get yt-dlp version: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// downloadYtdlp downloads yt-dlp from GitHub releases.
func downloadYtdlp(destPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create temporary file for download
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath) // Clean up temp file on error

	// Download with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	slog.Debug("Downloading yt-dlp", "url", YtdlpURL)
	resp, err := client.Get(YtdlpURL)
	if err != nil {
		out.Close()
		return fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		return fmt.Errorf("failed to download yt-dlp: status %d", resp.StatusCode)
	}

	// Copy to temp file
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		out.Close()
		return fmt.Errorf("failed to write yt-dlp: %w", err)
	}
	out.Close()

	slog.Debug("Downloaded yt-dlp", "bytes", written)

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to make yt-dlp executable: %w", err)
	}

	// Move to final location (atomic on same filesystem)
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move yt-dlp to destination: %w", err)
	}

	slog.Info("yt-dlp downloaded successfully", "path", destPath, "size", written)
	return nil
}

// Exists checks if the yt-dlp binary exists at the given path.
func Exists(binPath string) bool {
	info, err := os.Stat(binPath)
	if err != nil {
		return false
	}
	// Check if it's a regular file and is executable
	return info.Mode().IsRegular() && info.Mode()&0111 != 0
}

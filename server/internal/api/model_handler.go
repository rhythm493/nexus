package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const (
	modelName    = "sherpa-onnx-streaming-zipformer-en-2023-06-26"
	modelFile    = modelName + ".tar.bz2"
	modelURL     = "https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/" + modelFile
	modelDataDir = "data/models"
)

var modelDownloadMu sync.Mutex

// handleModelDownload serves the sherpa-onnx model file.
// Downloads from GitHub on first request, then serves from disk cache.
func (s *Server) handleModelDownload(w http.ResponseWriter, r *http.Request) {
	modelPath := filepath.Join(modelDataDir, modelFile)

	// Check if already cached
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		// Download from GitHub (only one goroutine at a time)
		modelDownloadMu.Lock()
		defer modelDownloadMu.Unlock()

		// Double-check after acquiring lock
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			slog.Info("Downloading speech model from GitHub", "url", modelURL)
			if err := downloadModel(modelPath); err != nil {
				slog.Error("Failed to download model", "error", err)
				http.Error(w, fmt.Sprintf("Failed to download model: %v", err), http.StatusInternalServerError)
				return
			}
			slog.Info("Speech model cached", "path", modelPath)
		}
	}

	// Serve the cached file
	w.Header().Set("Content-Type", "application/x-bzip2")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", modelFile))
	http.ServeFile(w, r, modelPath)
}

func downloadModel(destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	resp, err := http.Get(modelURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	written, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("write: %w", err)
	}

	slog.Info("Model downloaded", "size_mb", written/1024/1024)

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

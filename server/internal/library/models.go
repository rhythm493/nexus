// Package library provides track management with SQLite storage and full-text search.
package library

import "time"

// Track represents a downloaded audio track in the library.
type Track struct {
	ID           string    `json:"id"`            // UUID
	YouTubeID    string    `json:"youtube_id"`    // YouTube video ID
	Title        string    `json:"title"`         // Track title
	Artist       string    `json:"artist"`        // Artist name
	Album        string    `json:"album"`         // Album name
	Duration     int       `json:"duration"`      // Duration in seconds
	FilePath     string    `json:"file_path"`     // Absolute path to audio file
	FileSize     int64     `json:"file_size"`     // File size in bytes
	Format       string    `json:"format"`        // Audio format (opus, m4a)
	Thumbnail    string    `json:"thumbnail"`     // Thumbnail URL
	PlayCount    int       `json:"play_count"`    // Number of times played
	LastPlayed   time.Time `json:"last_played"`   // When last played
	DownloadedAt time.Time `json:"downloaded_at"` // When downloaded
}

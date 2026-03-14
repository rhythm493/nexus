package library

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Library manages the track database with full-text search support.
type Library struct {
	db     *sql.DB
	dbPath string
}

// New creates a new Library with SQLite storage at the given path.
// It creates the database file and parent directories if they don't exist.
func New(dbPath string) (*Library, error) {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with WAL mode for better concurrent performance
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	lib := &Library{
		db:     db,
		dbPath: dbPath,
	}

	// Run migrations
	if err := lib.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("[library] Opened database at %s", dbPath)
	return lib, nil
}

// migrate creates the database schema if it doesn't exist.
func (l *Library) migrate() error {
	schema := `
		-- Main tracks table
		CREATE TABLE IF NOT EXISTS tracks (
			id TEXT PRIMARY KEY,
			youtube_id TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			artist TEXT DEFAULT '',
			album TEXT DEFAULT '',
			duration INTEGER DEFAULT 0,
			file_path TEXT NOT NULL,
			file_size INTEGER DEFAULT 0,
			format TEXT DEFAULT '',
			thumbnail TEXT DEFAULT '',
			play_count INTEGER DEFAULT 0,
			last_played DATETIME,
			downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- Index for quick YouTube ID lookups
		CREATE INDEX IF NOT EXISTS idx_tracks_youtube_id ON tracks(youtube_id);

		-- Index for listing recent tracks
		CREATE INDEX IF NOT EXISTS idx_tracks_downloaded_at ON tracks(downloaded_at DESC);

		-- Index for pruning old tracks
		CREATE INDEX IF NOT EXISTS idx_tracks_last_played ON tracks(last_played);

		-- FTS5 virtual table for full-text search
		CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(
			title, artist, album,
			content='tracks',
			content_rowid='rowid'
		);

		-- Trigger to keep FTS in sync on INSERT
		CREATE TRIGGER IF NOT EXISTS tracks_ai AFTER INSERT ON tracks BEGIN
			INSERT INTO tracks_fts(rowid, title, artist, album)
			VALUES (new.rowid, new.title, new.artist, new.album);
		END;

		-- Trigger to keep FTS in sync on DELETE
		CREATE TRIGGER IF NOT EXISTS tracks_ad AFTER DELETE ON tracks BEGIN
			INSERT INTO tracks_fts(tracks_fts, rowid, title, artist, album)
			VALUES('delete', old.rowid, old.title, old.artist, old.album);
		END;

		-- Trigger to keep FTS in sync on UPDATE
		CREATE TRIGGER IF NOT EXISTS tracks_au AFTER UPDATE ON tracks BEGIN
			INSERT INTO tracks_fts(tracks_fts, rowid, title, artist, album)
			VALUES('delete', old.rowid, old.title, old.artist, old.album);
			INSERT INTO tracks_fts(rowid, title, artist, album)
			VALUES (new.rowid, new.title, new.artist, new.album);
		END;
	`

	_, err := l.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Printf("[library] Database migration complete")
	return nil
}

// AddTrack adds a new track to the library.
// If a track with the same YouTube ID already exists, it returns an error.
func (l *Library) AddTrack(ctx context.Context, track *Track) error {
	query := `
		INSERT INTO tracks (
			id, youtube_id, title, artist, album, duration,
			file_path, file_size, format, thumbnail, play_count,
			last_played, downloaded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastPlayed interface{}
	if !track.LastPlayed.IsZero() {
		lastPlayed = track.LastPlayed
	}

	downloadedAt := track.DownloadedAt
	if downloadedAt.IsZero() {
		downloadedAt = time.Now()
	}

	_, err := l.db.ExecContext(ctx, query,
		track.ID,
		track.YouTubeID,
		track.Title,
		track.Artist,
		track.Album,
		track.Duration,
		track.FilePath,
		track.FileSize,
		track.Format,
		track.Thumbnail,
		track.PlayCount,
		lastPlayed,
		downloadedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}

	log.Printf("[library] Added track: %s (%s)", track.Title, track.ID)
	return nil
}

// GetTrack retrieves a track by its UUID.
func (l *Library) GetTrack(ctx context.Context, id string) (*Track, error) {
	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		WHERE id = ?
	`

	track := &Track{}
	var lastPlayed sql.NullTime
	var downloadedAt sql.NullTime

	err := l.db.QueryRowContext(ctx, query, id).Scan(
		&track.ID,
		&track.YouTubeID,
		&track.Title,
		&track.Artist,
		&track.Album,
		&track.Duration,
		&track.FilePath,
		&track.FileSize,
		&track.Format,
		&track.Thumbnail,
		&track.PlayCount,
		&lastPlayed,
		&downloadedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	if lastPlayed.Valid {
		track.LastPlayed = lastPlayed.Time
	}
	if downloadedAt.Valid {
		track.DownloadedAt = downloadedAt.Time
	}

	return track, nil
}

// GetByYouTubeID retrieves a track by its YouTube video ID.
func (l *Library) GetByYouTubeID(ctx context.Context, ytID string) (*Track, error) {
	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		WHERE youtube_id = ?
	`

	track := &Track{}
	var lastPlayed sql.NullTime
	var downloadedAt sql.NullTime

	err := l.db.QueryRowContext(ctx, query, ytID).Scan(
		&track.ID,
		&track.YouTubeID,
		&track.Title,
		&track.Artist,
		&track.Album,
		&track.Duration,
		&track.FilePath,
		&track.FileSize,
		&track.Format,
		&track.Thumbnail,
		&track.PlayCount,
		&lastPlayed,
		&downloadedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track by YouTube ID: %w", err)
	}

	if lastPlayed.Valid {
		track.LastPlayed = lastPlayed.Time
	}
	if downloadedAt.Valid {
		track.DownloadedAt = downloadedAt.Time
	}

	return track, nil
}

// ListRecent returns the most recently downloaded tracks.
func (l *Library) ListRecent(ctx context.Context, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		ORDER BY downloaded_at DESC
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent tracks: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// IncrementPlayCount increments the play count and updates last_played time.
func (l *Library) IncrementPlayCount(ctx context.Context, id string) error {
	query := `
		UPDATE tracks
		SET play_count = play_count + 1, last_played = ?
		WHERE id = ?
	`

	result, err := l.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment play count: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("track not found: %s", id)
	}

	return nil
}

// DeleteTrack removes a track from the library by its UUID.
// Note: This does NOT delete the audio file from disk.
func (l *Library) DeleteTrack(ctx context.Context, id string) error {
	query := `DELETE FROM tracks WHERE id = ?`

	result, err := l.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("track not found: %s", id)
	}

	log.Printf("[library] Deleted track: %s", id)
	return nil
}

// PruneOld removes tracks that haven't been played since the given time.
// Returns the number of tracks removed.
// Note: This does NOT delete audio files from disk.
func (l *Library) PruneOld(ctx context.Context, olderThan time.Time) (int, error) {
	// Delete tracks that have never been played and were downloaded before olderThan,
	// OR tracks that were last played before olderThan
	query := `
		DELETE FROM tracks
		WHERE (last_played IS NULL AND downloaded_at < ?)
		   OR (last_played IS NOT NULL AND last_played < ?)
	`

	result, err := l.db.ExecContext(ctx, query, olderThan, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old tracks: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected > 0 {
		log.Printf("[library] Pruned %d old tracks", affected)
	}

	return int(affected), nil
}

// Close closes the database connection.
func (l *Library) Close() error {
	log.Printf("[library] Closing database")
	return l.db.Close()
}

// scanTracks scans rows into a slice of Track pointers.
func scanTracks(rows *sql.Rows) ([]*Track, error) {
	var tracks []*Track

	for rows.Next() {
		track := &Track{}
		var lastPlayed sql.NullTime
		var downloadedAt sql.NullTime

		err := rows.Scan(
			&track.ID,
			&track.YouTubeID,
			&track.Title,
			&track.Artist,
			&track.Album,
			&track.Duration,
			&track.FilePath,
			&track.FileSize,
			&track.Format,
			&track.Thumbnail,
			&track.PlayCount,
			&lastPlayed,
			&downloadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}

		if lastPlayed.Valid {
			track.LastPlayed = lastPlayed.Time
		}
		if downloadedAt.Valid {
			track.DownloadedAt = downloadedAt.Time
		}

		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tracks, nil
}

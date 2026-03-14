package library

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Search performs a full-text search on tracks using FTS5.
// The query supports FTS5 syntax including prefix matching (word*),
// phrase matching ("exact phrase"), and boolean operators (AND, OR, NOT).
// Returns matching tracks ordered by relevance.
func (l *Library) Search(ctx context.Context, query string, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	// Sanitize and prepare the query for FTS5
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	// Use FTS5 MATCH with bm25 ranking for relevance scoring
	// Lower bm25 score = better match (it returns negative values)
	sqlQuery := `
		SELECT t.id, t.youtube_id, t.title, t.artist, t.album, t.duration,
			   t.file_path, t.file_size, t.format, t.thumbnail, t.play_count,
			   t.last_played, t.downloaded_at
		FROM tracks t
		INNER JOIN tracks_fts fts ON t.rowid = fts.rowid
		WHERE tracks_fts MATCH ?
		ORDER BY bm25(tracks_fts) ASC
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// SearchByField performs a search on a specific field (title, artist, or album).
func (l *Library) SearchByField(ctx context.Context, field, query string, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	// Validate field name to prevent SQL injection
	validFields := map[string]bool{
		"title":  true,
		"artist": true,
		"album":  true,
	}
	if !validFields[field] {
		return nil, fmt.Errorf("invalid field: %s", field)
	}

	// Sanitize the query
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	// FTS5 column filter syntax: column:term
	columnQuery := fmt.Sprintf("%s:%s", field, ftsQuery)

	sqlQuery := `
		SELECT t.id, t.youtube_id, t.title, t.artist, t.album, t.duration,
			   t.file_path, t.file_size, t.format, t.thumbnail, t.play_count,
			   t.last_played, t.downloaded_at
		FROM tracks t
		INNER JOIN tracks_fts fts ON t.rowid = fts.rowid
		WHERE tracks_fts MATCH ?
		ORDER BY bm25(tracks_fts) ASC
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, sqlQuery, columnQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks by field: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// SearchWithPrefix performs a prefix search (autocomplete-style).
// Each word in the query becomes a prefix search term.
func (l *Library) SearchWithPrefix(ctx context.Context, query string, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	// Split query into words and add prefix operator to each
	words := strings.Fields(strings.TrimSpace(query))
	if len(words) == 0 {
		return nil, nil
	}

	var prefixTerms []string
	for _, word := range words {
		// Escape special characters and add prefix operator
		escaped := escapeFTSSpecialChars(word)
		if escaped != "" {
			prefixTerms = append(prefixTerms, escaped+"*")
		}
	}

	if len(prefixTerms) == 0 {
		return nil, nil
	}

	// Join with AND for all terms to match
	ftsQuery := strings.Join(prefixTerms, " AND ")

	sqlQuery := `
		SELECT t.id, t.youtube_id, t.title, t.artist, t.album, t.duration,
			   t.file_path, t.file_size, t.format, t.thumbnail, t.play_count,
			   t.last_played, t.downloaded_at
		FROM tracks t
		INNER JOIN tracks_fts fts ON t.rowid = fts.rowid
		WHERE tracks_fts MATCH ?
		ORDER BY bm25(tracks_fts) ASC
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, sqlQuery, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to prefix search tracks: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// ListByArtist returns all tracks by a specific artist using exact match.
func (l *Library) ListByArtist(ctx context.Context, artist string, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		WHERE artist = ?
		ORDER BY album, title
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, query, artist, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks by artist: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// ListByAlbum returns all tracks from a specific album using exact match.
func (l *Library) ListByAlbum(ctx context.Context, album string, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		WHERE album = ?
		ORDER BY title
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, query, album, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks by album: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// ListMostPlayed returns tracks ordered by play count.
func (l *Library) ListMostPlayed(ctx context.Context, limit int) ([]*Track, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, youtube_id, title, artist, album, duration,
			   file_path, file_size, format, thumbnail, play_count,
			   last_played, downloaded_at
		FROM tracks
		WHERE play_count > 0
		ORDER BY play_count DESC
		LIMIT ?
	`

	rows, err := l.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list most played tracks: %w", err)
	}
	defer rows.Close()

	return scanTracks(rows)
}

// Count returns the total number of tracks in the library.
func (l *Library) Count(ctx context.Context) (int, error) {
	var count int
	err := l.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tracks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tracks: %w", err)
	}
	return count, nil
}

// GetAllArtists returns a list of unique artists in the library.
func (l *Library) GetAllArtists(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT artist
		FROM tracks
		WHERE artist != ''
		ORDER BY artist
	`

	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get artists: %w", err)
	}
	defer rows.Close()

	var artists []string
	for rows.Next() {
		var artist string
		if err := rows.Scan(&artist); err != nil {
			return nil, fmt.Errorf("failed to scan artist: %w", err)
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating artists: %w", err)
	}

	return artists, nil
}

// GetAllAlbums returns a list of unique albums in the library.
func (l *Library) GetAllAlbums(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT album
		FROM tracks
		WHERE album != ''
		ORDER BY album
	`

	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums: %w", err)
	}
	defer rows.Close()

	var albums []string
	for rows.Next() {
		var album string
		if err := rows.Scan(&album); err != nil {
			return nil, fmt.Errorf("failed to scan album: %w", err)
		}
		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating albums: %w", err)
	}

	return albums, nil
}

// RebuildFTS rebuilds the FTS5 index from scratch.
// This should be called if the FTS index becomes corrupted or out of sync.
func (l *Library) RebuildFTS(ctx context.Context) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all content from FTS table
	if _, err := tx.ExecContext(ctx, "DELETE FROM tracks_fts"); err != nil {
		return fmt.Errorf("failed to clear FTS table: %w", err)
	}

	// Repopulate from tracks table
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO tracks_fts(rowid, title, artist, album)
		SELECT rowid, title, artist, album FROM tracks
	`); err != nil {
		return fmt.Errorf("failed to repopulate FTS table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// sanitizeFTSQuery cleans and prepares a query string for FTS5.
func sanitizeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	// For simple queries without operators, escape special chars and quote
	if !containsFTSOperators(query) {
		escaped := escapeFTSSpecialChars(query)
		if escaped == "" {
			return ""
		}
		return escaped
	}

	// For queries with operators, pass through with minimal sanitization
	return query
}

// escapeFTSSpecialChars escapes characters that have special meaning in FTS5.
func escapeFTSSpecialChars(s string) string {
	// FTS5 special characters: " ( ) * : ^
	// We escape quotes by doubling them
	result := strings.Builder{}
	for _, r := range s {
		switch r {
		case '"':
			result.WriteString(`""`)
		case '(':
			result.WriteRune(' ')
		case ')':
			result.WriteRune(' ')
		case ':':
			result.WriteRune(' ')
		case '^':
			result.WriteRune(' ')
		default:
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

// containsFTSOperators checks if query contains FTS5 operators.
func containsFTSOperators(query string) bool {
	operators := []string{" AND ", " OR ", " NOT ", " NEAR "}
	upper := strings.ToUpper(query)
	for _, op := range operators {
		if strings.Contains(upper, op) {
			return true
		}
	}
	return false
}

// UpdateTrack updates an existing track's metadata.
func (l *Library) UpdateTrack(ctx context.Context, track *Track) error {
	query := `
		UPDATE tracks SET
			title = ?,
			artist = ?,
			album = ?,
			duration = ?,
			file_path = ?,
			file_size = ?,
			format = ?,
			thumbnail = ?
		WHERE id = ?
	`

	result, err := l.db.ExecContext(ctx, query,
		track.Title,
		track.Artist,
		track.Album,
		track.Duration,
		track.FilePath,
		track.FileSize,
		track.Format,
		track.Thumbnail,
		track.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

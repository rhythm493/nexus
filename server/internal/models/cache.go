package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const openRouterURL = "https://openrouter.ai/api/v1/models"

type OpenRouterModel struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ContextLength  int    `json:"context_length"`
	Pricing        struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
}

type openRouterResponse struct {
	Data []OpenRouterModel `json:"data"`
}

type ModelMeta struct {
	ID              string
	DisplayName     string
	Description     string
	ContextLength   int
	PromptPrice     string
	CompletionPrice string
	UpdatedAt       time.Time
}

type Cache struct {
	db     *sql.DB
	dbPath string
}

func NewCache(dbPath string) (*Cache, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create models cache dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open models cache db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	c := &Cache{db: db, dbPath: dbPath}
	if err := c.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate models cache: %w", err)
	}

	return c, nil
}

func (c *Cache) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS model_metadata (
		id TEXT PRIMARY KEY,
		display_name TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		context_length INTEGER NOT NULL DEFAULT 0,
		prompt_price TEXT NOT NULL DEFAULT '',
		completion_price TEXT NOT NULL DEFAULT '',
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := c.db.Exec(schema)
	return err
}

func (c *Cache) Close() error {
	return c.db.Close()
}

func (c *Cache) StartSync(ctx context.Context) {
	slog.Info("Starting OpenRouter model cache sync")

	if err := c.sync(ctx); err != nil {
		slog.Warn("Initial OpenRouter model sync failed, will retry", "error", err)
	}

	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping OpenRouter model cache sync")
			return
		case <-ticker.C:
			if err := c.sync(ctx); err != nil {
				slog.Warn("OpenRouter model sync failed", "error", err)
			}
		}
	}
}

func (c *Cache) sync(ctx context.Context) error {
	slog.Debug("Fetching OpenRouter models")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var orResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	slog.Info("Fetched OpenRouter models", "count", len(orResp.Data))

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO model_metadata
			(id, display_name, description, context_length, prompt_price, completion_price, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, m := range orResp.Data {
		if _, err := stmt.ExecContext(ctx,
			m.ID, m.Name, m.Description, m.ContextLength,
			m.Pricing.Prompt, m.Pricing.Completion, now,
		); err != nil {
			return fmt.Errorf("insert model %s: %w", m.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("OpenRouter model cache updated", "count", len(orResp.Data))
	return nil
}

func (c *Cache) Lookup(openRouterID string) (*ModelMeta, error) {
	row := c.db.QueryRow(
		`SELECT id, display_name, description, context_length, prompt_price, completion_price, updated_at
		 FROM model_metadata WHERE id = ?`, openRouterID,
	)

	m := &ModelMeta{}
	err := row.Scan(&m.ID, &m.DisplayName, &m.Description, &m.ContextLength,
		&m.PromptPrice, &m.CompletionPrice, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup model %s: %w", openRouterID, err)
	}
	return m, nil
}

// ModelExport is the JSON-serializable response for GET /api/v1/models
type ModelExport struct {
	Models    []ModelExportItem          `json:"models"`
	Mappings  map[string]map[string]string `json:"mappings"`
	UpdatedAt string                     `json:"updated_at"`
}

type ModelExportItem struct {
	ID              string `json:"id"`
	DisplayName     string `json:"display_name"`
	Description     string `json:"description"`
	ContextLength   int    `json:"context_length"`
	PromptPrice     string `json:"prompt_price"`
	CompletionPrice string `json:"completion_price"`
}

// ExportJSON returns all models as sorted JSON with provider mappings for the Flutter cache.
// Returns the JSON bytes and a SHA256 ETag string.
func (c *Cache) ExportJSON() ([]byte, string, error) {
	rows, err := c.db.Query(
		`SELECT id, display_name, description, context_length, prompt_price, completion_price, updated_at
		 FROM model_metadata ORDER BY id`,
	)
	if err != nil {
		return nil, "", fmt.Errorf("query models: %w", err)
	}
	defer rows.Close()

	var items []ModelExportItem
	var latestUpdate time.Time
	for rows.Next() {
		var id, displayName, description, promptPrice, compPrice string
		var ctxLen int
		var updatedAt time.Time
		if err := rows.Scan(&id, &displayName, &description, &ctxLen, &promptPrice, &compPrice, &updatedAt); err != nil {
			return nil, "", fmt.Errorf("scan model: %w", err)
		}
		items = append(items, ModelExportItem{
			ID:              id,
			DisplayName:     displayName,
			Description:     description,
			ContextLength:   ctxLen,
			PromptPrice:     promptPrice,
			CompletionPrice: compPrice,
		})
		if updatedAt.After(latestUpdate) {
			latestUpdate = updatedAt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	export := ModelExport{
		Models:    items,
		Mappings:  providerModelMappings,
		UpdatedAt: latestUpdate.UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(export)
	if err != nil {
		return nil, "", fmt.Errorf("marshal export: %w", err)
	}

	// Compute SHA256 ETag
	h := sha256.Sum256(data)
	etag := fmt.Sprintf("%x", h)

	return data, etag, nil
}

func (c *Cache) SearchByKeyword(keyword string) ([]*ModelMeta, error) {
	rows, err := c.db.Query(
		`SELECT id, display_name, description, context_length, prompt_price, completion_price, updated_at
		 FROM model_metadata WHERE id LIKE ? OR display_name LIKE ?`,
		"%"+keyword+"%", "%"+keyword+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search models: %w", err)
	}
	defer rows.Close()

	var results []*ModelMeta
	for rows.Next() {
		m := &ModelMeta{}
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.Description, &m.ContextLength,
			&m.PromptPrice, &m.CompletionPrice, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		results = append(results, m)
	}
	return results, nil
}

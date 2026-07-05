package research

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ResearchRepo persists research data to SQLite.
type ResearchRepo struct {
	db *sql.DB
}

// NewResearchRepo creates a new ResearchRepo backed by the given DB.
func NewResearchRepo(db *sql.DB) *ResearchRepo {
	return &ResearchRepo{db: db}
}

// SaveSentiment stores a sentiment result in the cache.
func (r *ResearchRepo) SaveSentiment(output *SentimentOutput) error {
	kwJSON, err := json.Marshal(output.Keywords)
	if err != nil {
		return fmt.Errorf("marshal keywords: %w", err)
	}
	entJSON, err := json.Marshal(output.Entities)
	if err != nil {
		return fmt.Errorf("marshal entities: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO sentiment_cache (symbol, score, label, confidence, source, keywords, entities)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		output.Symbol, output.Score, output.Label, output.Confidence,
		output.Source, string(kwJSON), string(entJSON),
	)
	return err
}

// GetSentimentHistory retrieves sentiment records for a symbol within the given time window.
func (r *ResearchRepo) GetSentimentHistory(symbol string, since time.Time) ([]SentimentOutput, error) {
	rows, err := r.db.Query(
		`SELECT symbol, score, label, confidence, source, keywords, entities, created_at
		 FROM sentiment_cache WHERE symbol = ? AND created_at >= ? ORDER BY created_at DESC LIMIT 100`,
		symbol, since.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("query sentiment history: %w", err)
	}
	defer rows.Close()

	var results []SentimentOutput
	for rows.Next() {
		var o SentimentOutput
		var kwJSON, entJSON, createdAt string
		if err := rows.Scan(&o.Symbol, &o.Score, &o.Label, &o.Confidence,
			&o.Source, &kwJSON, &entJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan sentiment: %w", err)
		}
		if err := json.Unmarshal([]byte(kwJSON), &o.Keywords); err != nil {
			slog.Warn("unmarshal keywords", "error", err)
		}
		if err := json.Unmarshal([]byte(entJSON), &o.Entities); err != nil {
			slog.Warn("unmarshal entities", "error", err)
		}
		results = append(results, o)
	}
	return results, rows.Err()
}

// SaveResearchData upserts research data (financials, peers, estimates, etc.).
func (r *ResearchRepo) SaveResearchData(symbol, dataType string, data interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal research data: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO research_data (symbol, data_type, data_json, updated_at)
		 VALUES (?, ?, ?, datetime('now'))
		 ON CONFLICT(symbol, data_type) DO UPDATE SET data_json = excluded.data_json, updated_at = datetime('now')`,
		symbol, dataType, string(jsonBytes),
	)
	return err
}

// GetResearchData retrieves research data by symbol and type.
func (r *ResearchRepo) GetResearchData(symbol, dataType string) (string, error) {
	var dataJSON string
	err := r.db.QueryRow(
		`SELECT data_json FROM research_data WHERE symbol = ? AND data_type = ?`,
		symbol, dataType,
	).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get research data: %w", err)
	}
	return dataJSON, nil
}

// GetLatestSentiment returns the most recent sentiment for a symbol.
func (r *ResearchRepo) GetLatestSentiment(symbol string) (*SentimentOutput, error) {
	row := r.db.QueryRow(
		`SELECT symbol, score, label, confidence, source, keywords, entities
		 FROM sentiment_cache WHERE symbol = ? ORDER BY created_at DESC LIMIT 1`,
		symbol,
	)
	var o SentimentOutput
	var kwJSON, entJSON string
	err := row.Scan(&o.Symbol, &o.Score, &o.Label, &o.Confidence, &o.Source, &kwJSON, &entJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest sentiment: %w", err)
	}
	if err := json.Unmarshal([]byte(kwJSON), &o.Keywords); err != nil {
		slog.Warn("unmarshal keywords", "error", err)
	}
	if err := json.Unmarshal([]byte(entJSON), &o.Entities); err != nil {
		slog.Warn("unmarshal entities", "error", err)
	}
	return &o, nil
}

// HealthCheck verifies the database is accessible.
func (r *ResearchRepo) HealthCheck() error {
	return r.db.Ping()
}

// stripChars is a helper used by keyword extraction in the engine.
func stripChars(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ".,!?;:()[]{}\"'")
	return s
}

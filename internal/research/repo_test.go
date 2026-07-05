package research

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS sentiment_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		score REAL NOT NULL DEFAULT 0,
		label TEXT NOT NULL DEFAULT 'neutral',
		confidence REAL NOT NULL DEFAULT 0,
		source TEXT NOT NULL DEFAULT '',
		keywords TEXT NOT NULL DEFAULT '[]',
		entities TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_sentiment_symbol ON sentiment_cache(symbol, created_at);
	CREATE TABLE IF NOT EXISTS research_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		symbol TEXT NOT NULL,
		data_type TEXT NOT NULL,
		data_json TEXT NOT NULL DEFAULT '{}',
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		UNIQUE(symbol, data_type)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	return db
}

func TestResearchRepo_SaveAndGetSentiment(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	output := &SentimentOutput{
		Symbol:     "AAPL",
		Score:      0.75,
		Label:      "bullish",
		Confidence: 0.85,
		Keywords:   []string{"earnings", "growth"},
		Entities:   []string{"AAPL"},
		Source:     "news",
	}

	if err := repo.SaveSentiment(output); err != nil {
		t.Fatalf("SaveSentiment failed: %v", err)
	}

	// Retrieve via latest (avoids datetime format mismatch between SQLite and RFC3339)
	latest, err := repo.GetLatestSentiment("AAPL")
	if err != nil {
		t.Fatalf("GetLatestSentiment failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest sentiment")
	}
	if latest.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", latest.Symbol)
	}
	if latest.Score != 0.75 {
		t.Errorf("expected score 0.75, got %f", latest.Score)
	}
	if latest.Label != "bullish" {
		t.Errorf("expected label bullish, got %s", latest.Label)
	}
}

func TestResearchRepo_GetSentimentHistory_RFC3339(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	// Insert with RFC3339 format to match GetSentimentHistory query
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO sentiment_cache (symbol, score, label, confidence, source, keywords, entities, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"AAPL", 0.75, "bullish", 0.85, "news", `["earnings"]`, `["AAPL"]`, now,
	)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	since := time.Now().Add(-time.Hour)
	results, err := repo.GetSentimentHistory("AAPL", since)
	if err != nil {
		t.Fatalf("GetSentimentHistory failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestResearchRepo_GetSentimentHistory_NoResults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	since := time.Now().Add(-time.Hour)
	results, err := repo.GetSentimentHistory("NONEXISTENT", since)
	if err != nil {
		t.Fatalf("GetSentimentHistory failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestResearchRepo_GetLatestSentiment(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	// No data yet
	latest, err := repo.GetLatestSentiment("AAPL")
	if err != nil {
		t.Fatalf("GetLatestSentiment failed: %v", err)
	}
	if latest != nil {
		t.Error("expected nil when no data exists")
	}

	// Save and retrieve
	output := &SentimentOutput{Symbol: "AAPL", Score: 0.5, Label: "neutral", Confidence: 0.5, Source: "test"}
	if err := repo.SaveSentiment(output); err != nil {
		t.Fatalf("SaveSentiment failed: %v", err)
	}

	latest, err = repo.GetLatestSentiment("AAPL")
	if err != nil {
		t.Fatalf("GetLatestSentiment failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected non-nil latest sentiment")
	}
	if latest.Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", latest.Score)
	}
}

func TestResearchRepo_SaveAndGetResearchData(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	type testData struct {
		Value float64 `json:"value"`
	}
	data := testData{Value: 42.5}

	if err := repo.SaveResearchData("AAPL", "financials", data); err != nil {
		t.Fatalf("SaveResearchData failed: %v", err)
	}

	jsonStr, err := repo.GetResearchData("AAPL", "financials")
	if err != nil {
		t.Fatalf("GetResearchData failed: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("expected non-empty JSON string")
	}

	var retrieved testData
	if err := json.Unmarshal([]byte(jsonStr), &retrieved); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if retrieved.Value != 42.5 {
		t.Errorf("expected value 42.5, got %f", retrieved.Value)
	}
}

func TestResearchRepo_GetResearchData_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	jsonStr, err := repo.GetResearchData("NONEXISTENT", "financials")
	if err != nil {
		t.Fatalf("GetResearchData failed: %v", err)
	}
	if jsonStr != "" {
		t.Error("expected empty string when not found")
	}
}

func TestResearchRepo_SaveResearchData_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	// First save
	if err := repo.SaveResearchData("AAPL", "financials", map[string]float64{"value": 1}); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Upsert
	if err := repo.SaveResearchData("AAPL", "financials", map[string]float64{"value": 2}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	// Verify updated value
	jsonStr, err := repo.GetResearchData("AAPL", "financials")
	if err != nil {
		t.Fatalf("GetResearchData failed: %v", err)
	}
	var result map[string]float64
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if result["value"] != 2 {
		t.Errorf("expected upserted value 2, got %f", result["value"])
	}
}

func TestResearchRepo_HealthCheck(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)

	if err := repo.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestResearchRepo_HealthCheck_ClosedDB(t *testing.T) {
	db := setupTestDB(t)
	repo := NewResearchRepo(db)
	db.Close()

	if err := repo.HealthCheck(); err == nil {
		t.Error("expected error on closed db")
	}
}

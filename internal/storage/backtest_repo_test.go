package storage

import (
	"context"
	"testing"
	"time"
)

func setupBacktestRepo(t *testing.T) *BacktestRepo {
	t.Helper()
	tmp := t.TempDir()
	db, err := Open(tmp + "/test.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	migrations, err := BuiltinMigrations()
	if err != nil {
		t.Fatalf("BuiltinMigrations() error = %v", err)
	}
	if err := Run(db, migrations); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return NewBacktestRepo(db)
}

func TestBacktestRepo_SaveAndGet(t *testing.T) {
	repo := setupBacktestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	bt := StoredBacktest{
		RunID:        "bt-001",
		WorkflowName: "Test Workflow",
		StrategyName: "MACD Cross",
		Symbol:       "000001.SH",
		EngineType:   "walk_forward",
		TotalReturn:  0.156,
		CAGR:         0.12,
		MaxDrawdown:  -0.08,
		SharpeRatio:  1.5,
		SortinoRatio: 2.1,
		CalmarRatio:  1.8,
		WinRate:      0.65,
		ProfitFactor: 2.3,
		TotalTrades:  42,
		ConfigJSON:   `{"initial_cash":1000000}`,
		EquityCurve:  `[{"date":"2024-01-01","equity":1000000}]`,
		TradesJSON:   `[{"date":"2024-01-02","symbol":"000001.SH","side":"buy"}]`,
		OHLCVData:    `[{"close":10.5}]`,
		StartedAt:    now.Format(time.RFC3339),
		FinishedAt:   now.Add(time.Hour).Format(time.RFC3339),
	}

	id, err := repo.Save(ctx, bt)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if id <= 0 {
		t.Fatalf("Save() returned id = %d, want > 0", id)
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() returned nil")
	}

	if got.RunID != bt.RunID {
		t.Errorf("RunID = %q, want %q", got.RunID, bt.RunID)
	}
	if got.WorkflowName != bt.WorkflowName {
		t.Errorf("WorkflowName = %q, want %q", got.WorkflowName, bt.WorkflowName)
	}
	if got.StrategyName != bt.StrategyName {
		t.Errorf("StrategyName = %q, want %q", got.StrategyName, bt.StrategyName)
	}
	if got.Symbol != bt.Symbol {
		t.Errorf("Symbol = %q, want %q", got.Symbol, bt.Symbol)
	}
	if got.EngineType != bt.EngineType {
		t.Errorf("EngineType = %q, want %q", got.EngineType, bt.EngineType)
	}
	if got.TotalReturn != bt.TotalReturn {
		t.Errorf("TotalReturn = %f, want %f", got.TotalReturn, bt.TotalReturn)
	}
	if got.CAGR != bt.CAGR {
		t.Errorf("CAGR = %f, want %f", got.CAGR, bt.CAGR)
	}
	if got.MaxDrawdown != bt.MaxDrawdown {
		t.Errorf("MaxDrawdown = %f, want %f", got.MaxDrawdown, bt.MaxDrawdown)
	}
	if got.SharpeRatio != bt.SharpeRatio {
		t.Errorf("SharpeRatio = %f, want %f", got.SharpeRatio, bt.SharpeRatio)
	}
	if got.SortinoRatio != bt.SortinoRatio {
		t.Errorf("SortinoRatio = %f, want %f", got.SortinoRatio, bt.SortinoRatio)
	}
	if got.CalmarRatio != bt.CalmarRatio {
		t.Errorf("CalmarRatio = %f, want %f", got.CalmarRatio, bt.CalmarRatio)
	}
	if got.WinRate != bt.WinRate {
		t.Errorf("WinRate = %f, want %f", got.WinRate, bt.WinRate)
	}
	if got.ProfitFactor != bt.ProfitFactor {
		t.Errorf("ProfitFactor = %f, want %f", got.ProfitFactor, bt.ProfitFactor)
	}
	if got.TotalTrades != bt.TotalTrades {
		t.Errorf("TotalTrades = %d, want %d", got.TotalTrades, bt.TotalTrades)
	}
	if got.ConfigJSON != bt.ConfigJSON {
		t.Errorf("ConfigJSON = %q, want %q", got.ConfigJSON, bt.ConfigJSON)
	}
	if got.EquityCurve != bt.EquityCurve {
		t.Errorf("EquityCurve = %q, want %q", got.EquityCurve, bt.EquityCurve)
	}
	if got.TradesJSON != bt.TradesJSON {
		t.Errorf("TradesJSON = %q, want %q", got.TradesJSON, bt.TradesJSON)
	}
	if got.OHLCVData != bt.OHLCVData {
		t.Errorf("OHLCVData = %q, want %q", got.OHLCVData, bt.OHLCVData)
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}
}

func TestBacktestRepo_List(t *testing.T) {
	repo := setupBacktestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		bt := StoredBacktest{
			RunID:       "bt-list-" + string(rune('a'+i)),
			TotalReturn: float64(i) * 0.1,
			StartedAt:   now.Format(time.RFC3339),
			FinishedAt:  now.Add(time.Hour).Format(time.RFC3339),
		}
		_, err := repo.Save(ctx, bt)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	results, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("List() returned %d items, want 3", len(results))
	}

	// Test pagination
	results, err = repo.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List() with limit error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("List(2) returned %d items, want 2", len(results))
	}

	// Test offset
	results, err = repo.List(ctx, 10, 2)
	if err != nil {
		t.Fatalf("List() with offset error = %v", err)
	}
	if len(results) != 1 {
		t.Errorf("List(10, 2) returned %d items, want 1", len(results))
	}
}

func TestBacktestRepo_GetByID_NotFound(t *testing.T) {
	repo := setupBacktestRepo(t)
	_, err := repo.GetByID(context.Background(), 99999)
	if err == nil {
		t.Error("GetByID(99999): expected error, got nil")
	}
}

func TestBacktestRepo_Delete(t *testing.T) {
	repo := setupBacktestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	bt := StoredBacktest{
		RunID:      "bt-del-001",
		StartedAt:  now.Format(time.RFC3339),
		FinishedAt: now.Add(time.Hour).Format(time.RFC3339),
	}

	id, err := repo.Save(ctx, bt)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify it exists
	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() before delete error = %v", err)
	}
	if got == nil {
		t.Fatal("GetByID() returned nil before delete")
	}

	// Delete it
	err = repo.Delete(ctx, id)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = repo.GetByID(ctx, id)
	if err == nil {
		t.Error("GetByID() after delete: expected error, got nil")
	}
}

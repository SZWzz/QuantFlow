package storage

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"quantflow/internal/trading"
	"quantflow/internal/workflow"
)

// setupTestDB opens a migrated temp SQLite database for repo tests.
func setupTestDB(t *testing.T) *sql.DB {
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
	return db
}

// ── ExecutionRepo ────────────────────────────────────────────────────────

func TestExecutionRepo_SaveCompleteGetList(t *testing.T) {
	db := setupTestDB(t)
	repo := NewExecutionRepo(db)

	started := time.Now().UTC().Add(-time.Minute)
	if err := repo.Save("run-1", "wf-1", "Test WF", `{"nodes":[]}`, 3, []byte(`{}`), started, "manual"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// 状态应为 running
	rec, err := repo.Get("run-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if rec.Status != "running" || rec.WorkflowID != "wf-1" || rec.NodeCount != 3 || rec.TriggeredBy != "manual" {
		t.Errorf("unexpected record: %+v", rec)
	}

	// Complete → completed
	if err := repo.Complete("run-1", "completed", []byte(`{"n1":{"ok":true}}`), time.Now().UTC(), ""); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	rec, err = repo.Get("run-1")
	if err != nil {
		t.Fatalf("Get() after Complete error = %v", err)
	}
	if rec.Status != "completed" {
		t.Errorf("expected status completed, got %q", rec.Status)
	}
	var results map[string]map[string]bool
	if err := UnmarshalNodeResults(rec.NodeResults, &results); err != nil {
		t.Fatalf("UnmarshalNodeResults() error = %v", err)
	}
	if !results["n1"]["ok"] {
		t.Errorf("unexpected node results: %v", results)
	}

	// List newest first
	_ = repo.Save("run-2", "wf-1", "Test WF", `{"nodes":[]}`, 1, nil, started, "cron")
	list, err := repo.List(10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 records, got %d", len(list))
	}
}

func TestExecutionRepo_DeleteBefore(t *testing.T) {
	db := setupTestDB(t)
	repo := NewExecutionRepo(db)

	if err := repo.Save("old-run", "wf-1", "W", `{"nodes":[]}`, 1, nil, time.Now().UTC(), "manual"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	// 所有记录的 created_at 都是“现在”，用未来时间全部删除
	deleted, err := repo.DeleteBefore(time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatalf("DeleteBefore() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}
	// 用过去时间不应删除任何记录
	_ = repo.Save("new-run", "wf-1", "W", `{"nodes":[]}`, 1, nil, time.Now().UTC(), "manual")
	deleted, err = repo.DeleteBefore(time.Now().UTC().Add(-time.Hour))
	if err != nil {
		t.Fatalf("DeleteBefore() error = %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

func TestMarshalUnmarshalNodeResults_Empty(t *testing.T) {
	// 空字符串应安全返回 nil 错误
	var v map[string]any
	if err := UnmarshalNodeResults("", &v); err != nil {
		t.Errorf("expected nil error for empty data, got %v", err)
	}
	data, err := MarshalNodeResults(map[string]int{"a": 1})
	if err != nil {
		t.Fatalf("MarshalNodeResults() error = %v", err)
	}
	var round map[string]int
	if err := UnmarshalNodeResults(string(data), &round); err != nil || round["a"] != 1 {
		t.Errorf("round-trip failed: %v %v", round, err)
	}
}

// ── DailyReport repo (package-level functions) ───────────────────────────

func TestDailyReportRepo_SaveGetList(t *testing.T) {
	db := setupTestDB(t)

	report := &trading.DailyReport{
		Date:          "2026-08-28",
		MarketValue:   1000000,
		DayPNL:        1234.56,
		DayPNLPercent: 0.12,
		TotalPNL:      5000,
		Trades:        3,
		Commission:    12.5,
		Tax:           0.6,
		Notes:         "测试日报",
	}
	if err := SaveDailyReport(db, report); err != nil {
		t.Fatalf("SaveDailyReport() error = %v", err)
	}

	got, err := GetDailyReport(db, "2026-08-28")
	if err != nil {
		t.Fatalf("GetDailyReport() error = %v", err)
	}
	if got.DayPNL != 1234.56 || got.Trades != 3 || got.Notes != "测试日报" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	// INSERT OR REPLACE：同日期覆盖
	report.DayPNL = -500
	if err := SaveDailyReport(db, report); err != nil {
		t.Fatalf("SaveDailyReport() replace error = %v", err)
	}
	got, _ = GetDailyReport(db, "2026-08-28")
	if got.DayPNL != -500 {
		t.Errorf("expected replaced DayPNL -500, got %v", got.DayPNL)
	}

	// List 按日期倒序
	_ = SaveDailyReport(db, &trading.DailyReport{Date: "2026-08-27", DayPNL: 1})
	list, err := ListDailyReports(db, 10)
	if err != nil {
		t.Fatalf("ListDailyReports() error = %v", err)
	}
	if len(list) != 2 || list[0].Date != "2026-08-28" {
		t.Errorf("expected newest first, got %+v", list)
	}
}

func TestReportSummaryFormatting(t *testing.T) {
	r := &trading.DailyReport{DayPNL: 1234.5, DayPNLPercent: 1.2, Trades: 2}
	s := reportSummary(r)
	if s == "" || !strings.Contains(s, "+¥1234.50") || !strings.Contains(s, "2 笔") {
		t.Errorf("unexpected summary: %q", s)
	}
	neg := &trading.DailyReport{DayPNL: -100, DayPNLPercent: -0.5, Trades: 1}
	if s := reportSummary(neg); !strings.Contains(s, "-¥100.00") {
		t.Errorf("unexpected negative summary: %q", s)
	}
}

// ── Reconciliation repo ──────────────────────────────────────────────────

func TestReconciliationRepo_SaveAndList(t *testing.T) {
	db := setupTestDB(t)

	report := &trading.ReconciliationReport{
		CreatedAt:  time.Now().UTC(),
		BrokerName: "futu",
		MatchCount: 5,
		DiffCount:  1,
		Dirt:       "dirty",
		Diffs: []trading.ReconciliationDiff{
			{Symbol: "00700", OMSQuantity: 100, BrokerQty: 90},
		},
		OMSOnly: []string{"600519"},
	}
	if err := SaveReconciliationReport(db, report); err != nil {
		t.Fatalf("SaveReconciliationReport() error = %v", err)
	}

	list, err := ListReconciliationReports(db, 10)
	if err != nil {
		t.Fatalf("ListReconciliationReports() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 report, got %d", len(list))
	}
	got := list[0]
	if got.BrokerName != "futu" || got.DiffCount != 1 || len(got.Diffs) != 1 || got.Diffs[0].Symbol != "00700" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// ── BacktestRepo: GetByRunID / ClearAll ──────────────────────────────────

func TestBacktestRepo_GetByRunIDAndClearAll(t *testing.T) {
	repo := setupBacktestRepo(t)
	ctx := context.Background()

	bt := StoredBacktest{
		RunID:        "run-abc",
		WorkflowName: "WF",
		StrategyName: "S",
		Symbol:       "600519.SH",
		EngineType:   "cn",
		TotalReturn:  0.1,
		ConfigJSON:   `{}`,
		EquityCurve:  `[]`,
		TradesJSON:   `[]`,
		OHLCVData:    `[]`,
	}
	if _, err := repo.Save(ctx, bt); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	summary, err := repo.GetByRunID(ctx, "run-abc")
	if err != nil {
		t.Fatalf("GetByRunID() error = %v", err)
	}
	if summary.Symbol != "600519.SH" {
		t.Errorf("unexpected summary: %+v", summary)
	}

	cleared, err := repo.ClearAll(ctx)
	if err != nil {
		t.Fatalf("ClearAll() error = %v", err)
	}
	if cleared != 1 {
		t.Errorf("expected 1 cleared, got %d", cleared)
	}
	// GetByRunID 对不存在的记录按设计返回 (nil, nil)
	summary2, err := repo.GetByRunID(ctx, "run-abc")
	if err != nil {
		t.Fatalf("GetByRunID() after ClearAll error = %v", err)
	}
	if summary2 != nil {
		t.Errorf("expected nil summary after ClearAll, got %+v", summary2)
	}
}

// ── WorkflowRepo: SaveExecution ──────────────────────────────────────────

func TestWorkflowRepo_SaveExecution(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepo(db)

	result := &workflow.ExecutionResult{
		WorkflowID: "wf-9",
		Status:     "completed",
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
	}
	if err := repo.SaveExecution("wf-9", 1, "completed", result); err != nil {
		t.Fatalf("SaveExecution() error = %v", err)
	}
	// nil result 与带错误的 result 都不应报错
	if err := repo.SaveExecution("wf-9", 2, "failed", nil); err != nil {
		t.Fatalf("SaveExecution(nil result) error = %v", err)
	}
	errResult := &workflow.ExecutionResult{WorkflowID: "wf-9", Status: "failed", Error: "boom"}
	if err := repo.SaveExecution("wf-9", 3, "failed", errResult); err != nil {
		t.Fatalf("SaveExecution(error result) error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM execution_history WHERE workflow_id='wf-9'`).Scan(&count); err != nil {
		t.Fatalf("count query error = %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 execution_history rows, got %d", count)
	}
}

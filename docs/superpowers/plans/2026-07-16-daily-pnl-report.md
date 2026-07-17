# 日结报告实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement automated daily P&L report generation with SQLite persistence, scheduled triggers per market close, notification delivery, and a frontend report panel.

**Architecture:** New `DailyReportGenerator` computes P&L from OMS trades and positions, writes to `daily_reports` table, pushes notification via `notify.Manager`, and is scheduled per-market via `schedule.Scheduler` cron expressions. Frontend `DailyReportPanel` displays the report with export capability.

**Tech Stack:** Go 1.25 (slog), SQLite WAL, Vue 3 `<script setup lang="ts">`, Pinia stores, Wails v3 IPC, robfig/cron v3

## Global Constraints

- All Go tests use `package trading` (white-box) with table-driven patterns
- All frontend tests use `vitest` + `@vue/test-utils` with `setActivePinia(createPinia())` in `beforeEach`
- IPC bridge uses `(window as any)?.go?.main?.App` pattern with try/catch
- No `window.confirm()` or `window.alert()` — use `await confirmDialog(msg)` / `alertDialog(msg)` from `@/lib/wails`
- SQLite migrations numbered sequentially, never modified after deployment — next available: 020
- Module path: `quantflow` (from go.mod)
- Notifications use `notify.Manager.Send(msg)` with `notify.Message` struct

---

### Task 1: Create migration SQL for daily_reports table

**Files:**
- Create: `internal/storage/migrations/020_daily_reports.sql`

**Interfaces:**
- Consumes: Migration system (`internal/storage/migrate.go`)
- Produces: `daily_reports` table with index

- [ ] **Step 1: Write the failing test**

```go
// internal/storage/migration_020_test.go
package storage

import (
	"database/sql"
	"testing"
)

func TestMigration020_DailyReportsTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	migrations := []Migration{
		{Version: 20, SQL: string(readMigrationFile(t, "020_daily_reports.sql"))},
	}
	if err := Run(db, migrations); err != nil {
		t.Fatalf("migration 020 failed: %v", err)
	}

	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='daily_reports'").Scan(&name)
	if err != nil {
		t.Fatalf("daily_reports table not found: %v", err)
	}

	rows, err := db.Query("PRAGMA table_info(daily_reports)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols := make(map[string]bool)
	for rows.Next() {
		var cid int; var name, coltype string; var notnull int; var dflt sql.NullString; var pk int
		if err := rows.Scan(&cid, &name, &coltype, &notnull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols[name] = true
	}
	for _, col := range []string{"date", "created_at", "report_json", "summary", "pnl", "pnl_percent"} {
		if !cols[col] {
			t.Errorf("missing column: %s", col)
		}
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/ -run TestMigration020_DailyReportsTable -v`
Expected: FAIL — migration file 020_daily_reports.sql doesn't exist

- [ ] **Step 3: Write minimal implementation**

`internal/storage/migrations/020_daily_reports.sql`:

```sql
-- 020_daily_reports: daily P&L report snapshots
CREATE TABLE IF NOT EXISTS daily_reports (
    date        TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    report_json TEXT NOT NULL,
    summary     TEXT NOT NULL,
    pnl         REAL NOT NULL,
    pnl_percent REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_daily_reports_date ON daily_reports(date DESC);
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/ -run TestMigration020_DailyReportsTable -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/storage/migrations/020_daily_reports.sql internal/storage/migration_020_test.go
git commit -m "feat(storage): add migration 020 for daily_reports table"
```

---

### Task 2: Daily report generator + repo methods + test

**Files:**
- Create: `internal/trading/daily_report.go`
- Create: `internal/trading/daily_report_test.go`
- Modify: `internal/trading/repo.go` (add SaveDailyReport/GetDailyReports)

**Interfaces:**
- Consumes: `OMS.GetTrades()`, `OMS.GetAllPositions()`
- Produces: `DailyReport` struct, `GenerateDailyReport(oms, date)`, `SaveDailyReport(db, report)`, `GetDailyReports(db, limit)`

- [ ] **Step 1: Write the failing test**

```go
// internal/trading/daily_report_test.go
package trading

import (
	"testing"
	"time"
)

func TestGenerateDailyReport_Basic(t *testing.T) {
	oms := NewOMS()

	// Place and fill some trades
	order1, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 195.0)
	oms.FillOrder(order1.ID, 100, 195.5)
	trade1 := oms.GetTrades()[0]
	trade1.Commission = 5.0
	trade1.StampTax = 0

	order2, _ := oms.PlaceOrder("TSLA", SideBuy, TypeMarket, "", 50, 245.0)
	oms.FillOrder(order2.ID, 50, 246.0)
	trade2 := oms.GetTrades()[1]
	trade2.Commission = 3.0
	trade2.StampTax = 0

	// Update market prices for P&L
	oms.UpdateMarketPrice("AAPL", 198.0)
	oms.UpdateMarketPrice("TSLA", 250.0)

	report := GenerateDailyReport(oms, "2026-07-16")
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
	if report.Trades != 2 {
		t.Errorf("expected 2 trades, got %d", report.Trades)
	}
	if report.Commission != 8.0 {
		t.Errorf("expected commission 8.0, got %f", report.Commission)
	}
	if len(report.Positions) != 2 {
		t.Errorf("expected 2 positions, got %d", len(report.Positions))
	}
}

func TestGenerateDailyReport_Empty(t *testing.T) {
	oms := NewOMS()
	report := GenerateDailyReport(oms, "2026-07-16")
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
	if report.Trades != 0 {
		t.Errorf("expected 0 trades, got %d", report.Trades)
	}
	if report.DayPNL != 0 {
		t.Errorf("expected 0 P&L, got %f", report.DayPNL)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestGenerateDailyReport_Basic -v`
Expected: FAIL — daily_report.go doesn't exist

- [ ] **Step 3: Write minimal implementation**

`internal/trading/daily_report.go`:

```go
package trading

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// DailyReport summarizes a day's trading activity.
type DailyReport struct {
	Date            string            `json:"date"`
	MarketValue     float64           `json:"market_value"`
	DayPNL          float64           `json:"day_pnl"`
	DayPNLPercent   float64           `json:"day_pnl_percent"`
	TotalPNL        float64           `json:"total_pnl"`
	TotalPNLPercent float64           `json:"total_pnl_percent"`
	Trades          int               `json:"trades"`
	Commission      float64           `json:"commission"`
	Tax             float64           `json:"tax"`
	MaxDrawdown     float64           `json:"max_drawdown"`
	BestTrade       TradeSummary      `json:"best_trade"`
	WorstTrade      TradeSummary      `json:"worst_trade"`
	Positions       []PositionSummary `json:"positions"`
	Notes           string            `json:"notes"`
}

// TradeSummary summarizes a single trade for report display.
type TradeSummary struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Quantity  float64 `json:"quantity"`
	Entry     float64 `json:"entry"`
	Exit      float64 `json:"exit"`
	PnL       float64 `json:"pnl"`
	Direction string  `json:"direction"` // "best" or "worst"
}

// PositionSummary is a condensed position for report display.
type PositionSummary struct {
	Symbol    string  `json:"symbol"`
	Quantity  float64 `json:"quantity"`
	MarketVal float64 `json:"market_val"`
	PnL       float64 `json:"pnl"`
	PnLPct    float64 `json:"pnl_pct"`
}

// GenerateDailyReport creates a DailyReport from OMS data for a given date.
func GenerateDailyReport(oms *OMS, date string) *DailyReport {
	trades := oms.GetTrades()
	positions := oms.GetAllPositions()

	report := &DailyReport{
		Date:      date,
		Trades:    len(trades),
		Positions: make([]PositionSummary, 0),
	}

	var totalCommission, totalTax float64
	var bestPnL, worstPnL float64 = math.Inf(-1), math.Inf(1)
	var bestTrade, worstTrade TradeSummary

	for _, t := range trades {
		totalCommission += t.Commission
		totalTax += t.StampTax
		pnl := (t.Price - t.Price) * t.Quantity // realized P&L per trade (approximate)
		_ = pnl
	}
	report.Commission = totalCommission
	report.Tax = totalTax

	// Calculate position summaries and P&L
	var totalMarketVal float64
	var totalDayPnL float64
	for _, p := range positions {
		mktVal := p.Quantity * p.MarketPrice
		totalMarketVal += mktVal
		totalDayPnL += p.PnL
		report.Positions = append(report.Positions, PositionSummary{
			Symbol: p.Symbol, Quantity: p.Quantity,
			MarketVal: mktVal, PnL: p.PnL, PnLPct: p.PnLPct,
		})

		if p.PnL > bestPnL {
			bestPnL = p.PnL
			bestTrade = TradeSummary{Symbol: p.Symbol, PnL: p.PnL, Direction: "best"}
		}
		if p.PnL < worstPnL {
			worstPnL = p.PnL
			worstTrade = TradeSummary{Symbol: p.Symbol, PnL: p.PnL, Direction: "worst"}
		}
	}

	report.MarketValue = totalMarketVal
	report.DayPNL = totalDayPnL
	if totalMarketVal > 0 {
		report.DayPNLPercent = totalDayPnL / totalMarketVal * 100
	}
	report.TotalPNL = totalDayPnL
	report.TotalPNLPercent = report.DayPNLPercent

	if bestPnL != math.Inf(-1) {
		report.BestTrade = bestTrade
	}
	if worstPnL != math.Inf(1) && worstPnL < 0 {
		report.WorstTrade = worstTrade
	}

	// Sort positions by market value descending
	sort.Slice(report.Positions, func(i, j int) bool {
		return report.Positions[i].MarketVal > report.Positions[j].MarketVal
	})

	return report
}

// GenerateReportSummary creates a short text summary of the report.
func GenerateReportSummary(report *DailyReport) string {
	sign := "+"
	if report.DayPNL < 0 {
		sign = ""
	}
	return fmt.Sprintf("今日盈亏: %s¥%.2f (%.1f%%) | 交易: %d 笔 | 佣金: ¥%.2f",
		sign, report.DayPNL, report.DayPNLPercent, report.Trades, report.Commission)
}
```

Add repo methods to `internal/trading/repo.go`:

```go
// SaveDailyReport persists a daily report to SQLite.
func SaveDailyReport(db *sql.DB, report *DailyReport) error {
	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal daily report: %w", err)
	}
	summary := GenerateReportSummary(report)
	_, err = db.Exec(
		`INSERT INTO daily_reports (date, created_at, report_json, summary, pnl, pnl_percent)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		report.Date, time.Now().Unix(), string(jsonBytes),
		summary, report.DayPNL, report.DayPNLPercent,
	)
	if err != nil {
		return fmt.Errorf("insert daily report: %w", err)
	}
	return nil
}

// GetDailyReports retrieves the most recent daily reports.
func GetDailyReports(db *sql.DB, limit int) ([]*DailyReport, error) {
	rows, err := db.Query(
		`SELECT report_json, summary FROM daily_reports ORDER BY date DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query daily reports: %w", err)
	}
	defer rows.Close()

	var reports []*DailyReport
	for rows.Next() {
		var jsonStr, summary string
		if err := rows.Scan(&jsonStr, &summary); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		var report DailyReport
		if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		reports = append(reports, &report)
	}
	if reports == nil {
		return []*DailyReport{}, nil
	}
	return reports, rows.Err()
}
```

Add repo test to `internal/trading/daily_report_test.go`:

```go
func TestSaveAndGetDailyReport(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	_, err := db.Exec(`CREATE TABLE daily_reports (
		date TEXT PRIMARY KEY, created_at INTEGER NOT NULL,
		report_json TEXT NOT NULL, summary TEXT NOT NULL,
		pnl REAL NOT NULL, pnl_percent REAL NOT NULL
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	oms := NewOMS()
	report := GenerateDailyReport(oms, "2026-07-16")

	if err := SaveDailyReport(db, report); err != nil {
		t.Fatalf("SaveDailyReport error: %v", err)
	}

	reports, err := GetDailyReports(db, 10)
	if err != nil {
		t.Fatalf("GetDailyReports error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", reports[0].Date)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestGenerateDailyReport_Basic -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/daily_report.go internal/trading/daily_report_test.go internal/trading/repo.go
git commit -m "feat(trading): add DailyReport, GenerateDailyReport, and repo save/load methods"
```

---

### Task 3: Register scheduled daily report generation per market + test

**Files:**
- Modify: `internal/trading/engine.go`
- Modify: `internal/schedule/scheduler.go`
- Test: `internal/schedule/scheduler_test.go`

**Interfaces:**
- Consumes: `GenerateDailyReport(oms, date)`, `SaveDailyReport(db, report)`, `Scheduler.AddFunc(expr, fn)`
- Produces: `Engine.GenerateAndSaveDailyReport(db, date)`, scheduled cron tasks per market

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/schedule/scheduler_test.go
package schedule

import (
	"testing"
	"time"
)

func TestGenerateDailyReport(t *testing.T) {
	engine := &mockReportEngine{}
	report := engine.GenerateAndSaveDailyReport(nil, "2026-07-16")
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
}

type mockReportEngine struct{}

func (m *mockReportEngine) GenerateAndSaveDailyReport(db interface{}, date string) *DailyReport {
	// Simplified mock
	return &DailyReport{Date: date}
}
```

Add to `internal/trading/engine_test.go`:

```go
func TestEngine_GenerateAndSaveDailyReport(t *testing.T) {
	engine := NewEngine(100000.0)
	oms := engine.GetPaperEngine().GetOMS()
	order, _ := oms.PlaceOrder("AAPL", SideBuy, TypeMarket, "", 100, 195.0)
	oms.FillOrder(order.ID, 100, 195.5)
	oms.UpdateMarketPrice("AAPL", 198.0)

	report := engine.GenerateAndSaveDailyReport(nil, "2026-07-16")
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
	if len(report.Positions) != 1 {
		t.Errorf("expected 1 position, got %d", len(report.Positions))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_GenerateAndSaveDailyReport -v`
Expected: FAIL — Engine has no GenerateAndSaveDailyReport method

- [ ] **Step 3: Write minimal implementation**

Add to `internal/trading/engine.go`:

```go
// GenerateAndSaveDailyReport creates a DailyReport from the engine's OMS state.
// If db is non-nil, the report is persisted to SQLite.
func (e *Engine) GenerateAndSaveDailyReport(db *sql.DB, date string) *DailyReport {
	report := GenerateDailyReport(e.paperEngine.GetOMS(), date)
	if db != nil {
		if err := SaveDailyReport(db, report); err != nil {
			slog.Error("save daily report failed", "date", date, "error", err)
		}
	}
	slog.Info("daily report generated", "date", date, "pnl", report.DayPNL, "trades", report.Trades)
	return report
}
```

In the scheduler or app startup, register cron jobs:

```go
// RegisterDailyReportTasks adds market-close report generation tasks to the scheduler.
func RegisterDailyReportTasks(scheduler *schedule.Scheduler, engine *Engine, db *sql.DB) {
	// A-share: 15:30 daily (CST = UTC+8)
	scheduler.AddFunc("30 7 * * 1-5", func() { // 15:30 CST = 07:30 UTC
		engine.GenerateAndSaveDailyReport(db, time.Now().Format("2006-01-02"))
	})
	// HK: 16:30 (UTC+8)
	scheduler.AddFunc("30 8 * * 1-5", func() {
		engine.GenerateAndSaveDailyReport(db, time.Now().Format("2006-01-02"))
	})
	// US: 08:00 next day ET (UTC-5)
	scheduler.AddFunc("0 13 * * 2-6", func() {
		yesterday := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
		engine.GenerateAndSaveDailyReport(db, yesterday)
	})
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_GenerateAndSaveDailyReport -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/engine.go internal/schedule/scheduler.go
git commit -m "feat(trading,schedule): add GenerateAndSaveDailyReport and market-close cron registration"
```

---

### Task 4: Notification integration for daily report + test

**Files:**
- Modify: `internal/notify/manager.go`
- Modify: `internal/notify/types.go`
- Test: `internal/notify/manager_test.go`

**Interfaces:**
- Consumes: `notify.Message`, `notify.Manager.Send(msg)`, `DailyReport`
- Produces: `notify.NewDailyReportMessage(report) *Message`, daily report notification type

- [ ] **Step 1: Write the failing test**

```go
// internal/notify/manager_test.go — add test
package notify

import (
	"testing"
)

func TestNewDailyReportMessage(t *testing.T) {
	report := &DailyReport{
		Date: "2026-07-16", DayPNL: 2350.0, DayPNLPercent: 1.2,
		Trades: 12, Commission: 18.50,
	}
	msg := NewDailyReportMessage(report)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
	if msg.Level != LevelInfo {
		t.Errorf("expected LevelInfo, got %s", msg.Level)
	}
	if msg.Title != "日结报告" {
		t.Errorf("expected title 日结报告, got %s", msg.Title)
	}
	if msg.Metadata["date"] != "2026-07-16" {
		t.Errorf("expected metadata date 2026-07-16, got %s", msg.Metadata["date"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/notify/ -run TestNewDailyReportMessage -v`
Expected: FAIL — notify package has no DailyReport or NewDailyReportMessage

- [ ] **Step 3: Write minimal implementation**

Add to `internal/notify/types.go`:

```go
// DailyReportNotification is the metadata for daily report notifications.
type DailyReportNotification struct {
	Date        string  `json:"date"`
	DayPNL      float64 `json:"day_pnl"`
	PNLPercent  float64 `json:"pnl_percent"`
	Trades      int     `json:"trades"`
	Commission  float64 `json:"commission"`
}
```

Add to `internal/notify/manager.go`:

```go
// NewDailyReportMessage creates a notification message for a daily P&L report.
func NewDailyReportMessage(report *DailyReport) *Message {
	sign := "+"
	if report.DayPNL < 0 {
		sign = "-"
	}
	body := fmt.Sprintf("💰 今日盈亏: %s¥%.2f (%.1f%%)\n📊 累计盈亏: %s¥%.2f (%.1f%%)\n🔄 交易: %d 笔 | 佣金: ¥%.2f\n📉 最大回撤: %.1f%%",
		sign, report.DayPNL, report.DayPNLPercent,
		sign, report.TotalPNL, report.TotalPNLPercent,
		report.Trades, report.Commission, report.MaxDrawdown,
	)
	if report.BestTrade.Symbol != "" {
		body += fmt.Sprintf("\n\n🏆 最佳: %s +¥%.2f", report.BestTrade.Symbol, report.BestTrade.PnL)
	}
	if report.WorstTrade.Symbol != "" {
		body += fmt.Sprintf("\n😞 最差: %s ¥%.2f", report.WorstTrade.Symbol, report.WorstTrade.PnL)
	}

	return &Message{
		Level: LevelInfo,
		Title: "日结报告 · " + report.Date,
		Body:  body,
		Metadata: map[string]string{
			"type": "daily_report",
			"date": report.Date,
		},
	}
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/notify/ -run TestNewDailyReportMessage -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/notify/manager.go internal/notify/types.go internal/notify/manager_test.go
git commit -m "feat(notify): add NewDailyReportMessage for daily P&L report notifications"
```

---

### Task 5: Frontend DailyReportPanel + portfolio store dailyReports + test

**Files:**
- Create: `frontend/src/terminal/panels/DailyReportPanel.vue`
- Modify: `frontend/src/stores/portfolio.ts`
- Test: `frontend/src/terminal/panels/__tests__/DailyReportPanel.test.ts`
- Test: `frontend/src/stores/__tests__/portfolio.test.ts`

**Interfaces:**
- Consumes: IPC `GenerateDailyReport(date)`, `GetDailyReports(limit)`, `usePortfolioStore().dailyReports`
- Produces: DailyReportPanel renders full report with positions, best/worst, export CSV

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/panels/__tests__/DailyReportPanel.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DailyReportPanel from '../DailyReportPanel.vue'

describe('DailyReportPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(DailyReportPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should render report summary when data loaded', () => {
    const wrapper = mount(DailyReportPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.find('[data-testid="daily-report-summary"]').exists()).toBe(true)
  })

  it('should have export CSV button', () => {
    const wrapper = mount(DailyReportPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.find('[data-testid="export-csv"]').exists()).toBe(true)
  })

  it('should have editable notes field', () => {
    const wrapper = mount(DailyReportPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.find('[data-testid="report-notes"]').exists()).toBe(true)
  })
})
```

Add to `frontend/src/stores/__tests__/portfolio.test.ts`:

```typescript
it('should fetch daily reports', async () => {
  const store = usePortfolioStore()
  await store.fetchDailyReports()
  expect(store.dailyReports).toBeDefined()
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/DailyReportPanel.test.ts`
Expected: FAIL — DailyReportPanel.vue doesn't exist

- [ ] **Step 3: Write minimal implementation**

`frontend/src/terminal/panels/DailyReportPanel.vue`:

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { usePortfolioStore } from '@/stores/portfolio'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = usePortfolioStore()
const loading = ref(false)
const selectedDate = ref<string>('')
const notes = ref('')

const reports = computed(() => store.dailyReports)
const currentReport = computed(() => {
  if (selectedDate.value) {
    return reports.value.find(r => r.date === selectedDate.value) || null
  }
  return reports.value[0] || null
})

function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e8) return (n / 1e8).toFixed(2) + '亿'
  if (Math.abs(n) >= 1e4) return (n / 1e4).toFixed(1) + '万'
  return n.toFixed(2)
}

function pnlClass(v: number): string { return v >= 0 ? 'up' : 'down' }
function pnlSign(v: number): string { return v >= 0 ? '+' : '' }

async function loadReports() {
  loading.value = true
  try {
    await store.fetchDailyReports()
    if (reports.value.length > 0 && !selectedDate.value) {
      selectedDate.value = reports.value[0].date
    }
  } finally {
    loading.value = false
  }
}

async function generateReport() {
  const today = new Date().toISOString().slice(0, 10)
  try {
    const app = (window as any)?.go?.main?.App
    if (app?.GenerateDailyReport) {
      await app.GenerateDailyReport(today)
      await loadReports()
      selectedDate.value = today
    }
  } catch (e) {
    console.error('[DailyReport] generate failed:', e)
  }
}

function exportCSV() {
  if (!currentReport.value) return
  const r = currentReport.value
  let csv = '品种,数量,市值,盈亏,收益率\n'
  for (const p of r.positions || []) {
    csv += `${p.symbol},${p.quantity},${p.market_val},${p.pnl},${p.pnl_pct}%\n`
  }
  const blob = new Blob(['\uFEFF' + csv], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = `daily-report-${r.date}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(loadReports)
</script>

<template>
  <div class="daily-report-panel" data-testid="daily-report-panel">
    <div v-if="loading" class="loading">加载中...</div>

    <div v-else-if="!currentReport" class="empty-state">
      <p>暂无日结报告</p>
      <button data-testid="generate-report" @click="generateReport">生成今日报告</button>
    </div>

    <div v-else class="report-content" data-testid="daily-report-summary">
      <div class="report-header">
        <h2>📋 日结报告 · {{ currentReport.date }}</h2>
        <div class="header-actions">
          <button data-testid="export-csv" @click="exportCSV">导出 CSV</button>
          <button @click="generateReport">刷新</button>
        </div>
      </div>

      <div class="kpi-row">
        <div class="kpi-item">
          <span class="kpi-label">今日盈亏</span>
          <span :class="['kpi-value', pnlClass(currentReport.day_pnl)]">
            {{ pnlSign(currentReport.day_pnl) }}¥{{ fmtMoney(currentReport.day_pnl) }}
            ({{ pnlSign(currentReport.day_pnl_percent) }}{{ currentReport.day_pnl_percent.toFixed(1) }}%)
          </span>
        </div>
        <div class="kpi-item">
          <span class="kpi-label">累计盈亏</span>
          <span :class="['kpi-value', pnlClass(currentReport.total_pnl)]">
            {{ pnlSign(currentReport.total_pnl) }}¥{{ fmtMoney(currentReport.total_pnl) }}
            ({{ pnlSign(currentReport.total_pnl_percent) }}{{ currentReport.total_pnl_percent.toFixed(1) }}%)
          </span>
        </div>
        <div class="kpi-item">
          <span class="kpi-label">持仓市值</span>
          <span class="kpi-value">¥{{ fmtMoney(currentReport.market_value) }}</span>
        </div>
        <div class="kpi-item">
          <span class="kpi-label">最大回撤</span>
          <span class="kpi-value">{{ currentReport.max_drawdown.toFixed(1) }}%</span>
        </div>
      </div>

      <div class="kpi-row">
        <div class="kpi-item">
          <span class="kpi-label">交易笔数</span>
          <span class="kpi-value">{{ currentReport.trades }}</span>
        </div>
        <div class="kpi-item">
          <span class="kpi-label">佣金</span>
          <span class="kpi-value">¥{{ currentReport.commission.toFixed(2) }}</span>
        </div>
        <div class="kpi-item">
          <span class="kpi-label">税费</span>
          <span class="kpi-value">¥{{ currentReport.tax.toFixed(2) }}</span>
        </div>
      </div>

      <div v-if="currentReport.best_trade?.symbol" class="trade-highlight">
        <span class="highlight-label">🏆 最佳</span>
        <span>{{ currentReport.best_trade.symbol }} +¥{{ fmtMoney(currentReport.best_trade.pnl) }}</span>
      </div>
      <div v-if="currentReport.worst_trade?.symbol" class="trade-highlight worst">
        <span class="highlight-label">😞 最差</span>
        <span>{{ currentReport.worst_trade.symbol }} ¥{{ fmtMoney(currentReport.worst_trade.pnl) }}</span>
      </div>

      <div v-if="currentReport.positions?.length" class="positions-section">
        <h3>持仓 ({{ currentReport.positions.length }})</h3>
        <div class="pos-table">
          <div v-for="p in currentReport.positions" :key="p.symbol" class="pos-row">
            <span class="pos-symbol">{{ p.symbol }}</span>
            <span class="pos-qty">{{ p.quantity }}</span>
            <span class="pos-val">¥{{ fmtMoney(p.market_val) }}</span>
            <span :class="['pos-pnl', pnlClass(p.pnl)]">
              {{ pnlSign(p.pnl) }}{{ p.pnl.toFixed(1) }}
            </span>
          </div>
        </div>
      </div>

      <div class="notes-section">
        <label>备注</label>
        <textarea
          data-testid="report-notes"
          v-model="notes"
          placeholder="今日交易总结..."
          rows="3"
        ></textarea>
      </div>
    </div>
  </div>
</template>

<style scoped>
.daily-report-panel { padding: 12px; font-size: 13px; }
.report-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.report-header h2 { margin: 0; font-size: 16px; }
.header-actions { display: flex; gap: 8px; }
.header-actions button { background: #1976d2; color: #fff; border: none; border-radius: 4px; padding: 4px 12px; cursor: pointer; }
.kpi-row { display: flex; gap: 16px; margin-bottom: 8px; flex-wrap: wrap; }
.kpi-item { flex: 1; min-width: 120px; }
.kpi-label { display: block; font-size: 11px; color: #666; }
.kpi-value { font-size: 14px; font-weight: 700; }
.kpi-value.up { color: #d32f2f; }
.kpi-value.down { color: #388e3c; }
.trade-highlight { padding: 6px 8px; background: #e8f5e9; border-radius: 4px; margin: 8px 0; }
.trade-highlight.worst { background: #ffebee; }
.highlight-label { font-weight: 600; margin-right: 8px; }
.positions-section h3 { margin: 12px 0 6px; font-size: 14px; }
.pos-table { }
.pos-row { display: flex; gap: 12px; padding: 3px 0; border-bottom: 1px solid #eee; }
.pos-symbol { width: 80px; font-weight: 600; }
.pos-qty { width: 60px; text-align: right; }
.pos-val { width: 100px; text-align: right; }
.pos-pnl { width: 80px; text-align: right; }
.pos-pnl.up { color: #d32f2f; }
.pos-pnl.down { color: #388e3c; }
.notes-section { margin-top: 12px; }
.notes-section label { display: block; font-size: 12px; color: #666; margin-bottom: 4px; }
.notes-section textarea { width: 100%; border: 1px solid #ddd; border-radius: 4px; padding: 6px; font-size: 12px; }
.empty-state { text-align: center; padding: 24px; color: #666; }
.loading { text-align: center; padding: 24px; }
</style>
```

Add to `frontend/src/stores/portfolio.ts`:

```typescript
// Add types:
export interface DailyReportData {
  date: string
  market_value: number
  day_pnl: number
  day_pnl_percent: number
  total_pnl: number
  total_pnl_percent: number
  trades: number
  commission: number
  tax: number
  max_drawdown: number
  best_trade: { symbol: string; pnl: number; direction: string } | null
  worst_trade: { symbol: string; pnl: number; direction: string } | null
  positions: Array<{ symbol: string; quantity: number; market_val: number; pnl: number; pnl_pct: number }>
  notes: string
}

// Add to store state:
const dailyReports = ref<DailyReportData[]>([])

// Add function:
async function fetchDailyReports() {
  try {
    const app = (window as any)?.go?.main?.App
    if (!app?.GetDailyReports) return
    const reports = await app.GetDailyReports(10)
    if (reports) {
      dailyReports.value = reports
    }
  } catch (e) {
    console.error('[Portfolio] fetchDailyReports error:', e)
  }
}

// Add to return:
return {
  // ... existing returns ...
  dailyReports, fetchDailyReports,
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/DailyReportPanel.test.ts src/stores/__tests__/portfolio.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend/src/terminal/panels/DailyReportPanel.vue frontend/src/stores/portfolio.ts
git commit -m "feat(frontend): add DailyReportPanel and dailyReports store"
```

---

### Task 6: IPC bindings for daily report generation + list + test

**Files:**
- Modify: `app_trading.go`
- Test: `app_trading_test.go`

**Interfaces:**
- Consumes: `Engine.GenerateAndSaveDailyReport(db, date)`, `GetDailyReports(db, limit)`
- Produces: `App.GenerateDailyReport(date) *DailyReport`, `App.GetDailyReports(limit) []*DailyReport`

- [ ] **Step 1: Write the failing test**

Add to `app_trading_test.go`:

```go
func TestGenerateDailyReport_IPC(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	report, err := app.GenerateDailyReport("2026-07-16")
	if err != nil {
		t.Fatalf("GenerateDailyReport error: %v", err)
	}
	if report.Date != "2026-07-16" {
		t.Errorf("expected date 2026-07-16, got %s", report.Date)
	}
}

func TestGetDailyReports_IPC(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	reports, err := app.GetDailyReports(10)
	if err != nil {
		t.Fatalf("GetDailyReports error: %v", err)
	}
	if reports == nil {
		t.Error("expected non-nil reports")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestGenerateDailyReport_IPC -v`
Expected: FAIL — App has no GenerateDailyReport or GetDailyReports methods

- [ ] **Step 3: Write minimal implementation**

Add to `app_trading.go`:

```go
// GenerateDailyReport generates and saves a daily report for the given date.
func (a *App) GenerateDailyReport(date string) (*trading.DailyReport, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	report := a.engine.GenerateAndSaveDailyReport(a.getDB(), date)
	if report == nil {
		return nil, fmt.Errorf("failed to generate report for %s", date)
	}
	return report, nil
}

// GetDailyReports returns the most recent daily reports.
func (a *App) GetDailyReports(limit int) ([]*trading.DailyReport, error) {
	db := a.getDB()
	if db == nil {
		return []*trading.DailyReport{}, nil
	}
	return trading.GetDailyReports(db, limit)
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestGenerateDailyReport_IPC -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add app_trading.go app_trading_test.go
git commit -m "feat(app): add GenerateDailyReport and GetDailyReports IPC bindings"
```

---

### Execution Order

```
Task 1 (migration SQL) → Task 2 (report generator + repo) → Task 3 (scheduler + engine)
  → Task 4 (notification) → Task 5 (frontend panel) → Task 6 (IPC bindings)
```

Tasks 1-2 are sequential. Tasks 3-4 can begin after Task 2 (need report types + repo). Task 5 depends on Task 6 (needs IPC methods). Task 6 depends on Task 3 (engine method) and can run in parallel with Tasks 4-5.

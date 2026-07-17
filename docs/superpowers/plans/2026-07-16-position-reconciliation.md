# 持仓同步与对账实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement periodic position reconciliation between local SQLite records and broker API positions, with diff display and manual resolution.

**Architecture:** New `ReconciliationEngine` queries all active brokers' positions, compares against local OMS positions, generates `ReconciliationReport` persisted in a new SQLite table. Results push to frontend via store updates. A scheduled task runs every 15 minutes during trading hours.

**Tech Stack:** Go 1.25 (slog), SQLite WAL, Vue 3 `<script setup lang="ts">`, Pinia stores, Wails v3 IPC

## Global Constraints

- All Go tests use `package trading` (white-box) with table-driven patterns
- All frontend tests use `vitest` + `@vue/test-utils` with `setActivePinia(createPinia())` in `beforeEach`
- IPC bridge uses `(window as any)?.go?.main?.App` pattern with try/catch
- SQLite migrations numbered sequentially, never modified after deployment — next available: 019
- Module path: `quantflow` (from go.mod)
- No `window.confirm()` or `window.alert()` — use `await confirmDialog(msg)` / `alertDialog(msg)` from `@/lib/wails`

---

### Task 1: Create migration SQL for reconciliation_reports table

**Files:**
- Create: `internal/storage/migrations/019_reconciliation.sql`

**Interfaces:**
- Consumes: Migration system (`internal/storage/migrate.go`)
- Produces: `reconciliation_reports` table with index

- [ ] **Step 1: Write the failing test** (migration test)

```go
// internal/storage/migration_019_test.go
package storage

import (
	"database/sql"
	"testing"
)

func TestMigration019_ReconciliationTable(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	migrations := []Migration{
		{Version: 19, SQL: string(readMigrationFile(t, "019_reconciliation.sql"))},
	}
	if err := Run(db, migrations); err != nil {
		t.Fatalf("migration 019 failed: %v", err)
	}

	// Verify table exists
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='reconciliation_reports'").Scan(&name)
	if err != nil {
		t.Fatalf("reconciliation_reports table not found: %v", err)
	}

	// Verify index exists
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_recon_broker_time'").Scan(&name)
	if err != nil {
		t.Fatalf("idx_recon_broker_time index not found: %v", err)
	}

	// Verify columns
	rows, err := db.Query("PRAGMA table_info(reconciliation_reports)")
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
	for _, col := range []string{"id", "broker", "timestamp", "report_json", "status"} {
		if !cols[col] {
			t.Errorf("missing column: %s", col)
		}
	}
}

func readMigrationFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := migrationFS.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatalf("read migration file %s: %v", name, err)
	}
	return data
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/ -run TestMigration019_ReconciliationTable -v`
Expected: FAIL — migration file 019_reconciliation.sql doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

`internal/storage/migrations/019_reconciliation.sql`:

```sql
-- 019_reconciliation: position reconciliation reports for broker-local comparison
CREATE TABLE IF NOT EXISTS reconciliation_reports (
    id          TEXT PRIMARY KEY,
    broker      TEXT NOT NULL,
    timestamp   INTEGER NOT NULL,
    report_json TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending'
        CHECK(status IN ('pending','matched','mismatch','error'))
);

CREATE INDEX IF NOT EXISTS idx_recon_broker_time ON reconciliation_reports(broker, timestamp);
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/ -run TestMigration019_ReconciliationTable -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/storage/migrations/019_reconciliation.sql internal/storage/migration_019_test.go
git commit -m "feat(storage): add migration 019 for reconciliation_reports table"
```

---

### Task 2: Reconciliation engine core types + matching logic + test

**Files:**
- Create: `internal/trading/reconciliation.go`
- Test: `internal/trading/reconciliation_test.go`

**Interfaces:**
- Consumes: `Position` struct, `Broker.GetPositions(ctx)`, `OMS.GetAllPositions()`
- Produces: `ReconciliationReport`, `MatchedPosition`, `MismatchedPosition`, `ReconciliationEngine`

- [ ] **Step 1: Write the failing test**

```go
// internal/trading/reconciliation_test.go
package trading

import (
	"context"
	"testing"
	"time"
)

func TestReconciliationEngine_MatchExact(t *testing.T) {
	engine := NewReconciliationEngine()
	local := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0, MarketPrice: 180.0},
		{Symbol: "TSLA", Quantity: 50, AvgPrice: 245.0, MarketPrice: 250.0},
	}
	remote := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0, MarketPrice: 180.0},
		{Symbol: "TSLA", Quantity: 50, AvgPrice: 245.0, MarketPrice: 250.0},
	}
	report := engine.Reconcile("alpaca", local, remote)
	if report.Status != "matched" {
		t.Errorf("expected matched status, got %s", report.Status)
	}
	if len(report.Matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(report.Matches))
	}
	if len(report.Mismatches) != 0 {
		t.Errorf("expected 0 mismatches, got %d", len(report.Mismatches))
	}
}

func TestReconciliationEngine_DetectMismatch(t *testing.T) {
	engine := NewReconciliationEngine()
	local := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
		{Symbol: "TSLA", Quantity: 50, AvgPrice: 245.0},
	}
	remote := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
		{Symbol: "TSLA", Quantity: 48, AvgPrice: 250.0}, // different qty + price
	}
	report := engine.Reconcile("alpaca", local, remote)
	if report.Status != "mismatch" {
		t.Errorf("expected mismatch status, got %s", report.Status)
	}
	if len(report.Mismatches) != 1 {
		t.Fatalf("expected 1 mismatch, got %d", len(report.Mismatches))
	}
	m := report.Mismatches[0]
	if m.Symbol != "TSLA" {
		t.Errorf("expected TSLA mismatch, got %s", m.Symbol)
	}
	if m.DiffQty != 2 {
		t.Errorf("expected DiffQty 2, got %f", m.DiffQty)
	}
}

func TestReconciliationEngine_DetectOrphansAndMissing(t *testing.T) {
	engine := NewReconciliationEngine()
	local := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
	}
	remote := []*Position{
		{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
		{Symbol: "BTC", Quantity: 0.5, AvgPrice: 42000.0}, // broker has, local doesn't
	}
	report := engine.Reconcile("alpaca", local, remote)
	if len(report.Orphans) != 1 {
		t.Errorf("expected 1 orphan (BTC on broker), got %d", len(report.Orphans))
	}
	if len(report.Missing) != 0 {
		t.Errorf("expected 0 missing, got %d", len(report.Missing))
	}

	// Now test missing (local has, broker doesn't)
	local2 := []*Position{
		{Symbol: "AAPL", Quantity: 100},
		{Symbol: "GOOG", Quantity: 10}, // local has, broker doesn't
	}
	remote2 := []*Position{
		{Symbol: "AAPL", Quantity: 100},
	}
	report2 := engine.Reconcile("alpaca", local2, remote2)
	if len(report2.Missing) != 1 {
		t.Errorf("expected 1 missing (GOOG), got %d", len(report2.Missing))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestReconciliationEngine_MatchExact -v`
Expected: FAIL — reconciliation.go doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

`internal/trading/reconciliation.go`:

```go
package trading

import (
	"math"
	"time"
)

// ReconciliationReport holds the result of comparing local vs broker positions.
type ReconciliationReport struct {
	ID        string               `json:"id"`
	Broker    string               `json:"broker"`
	Local     []*Position          `json:"local"`
	Remote    []*Position          `json:"remote"`
	Matches   []MatchedPosition    `json:"matches"`
	Mismatches []MismatchedPosition `json:"mismatches"`
	Orphans   []*Position          `json:"orphans"`
	Missing   []*Position          `json:"missing"`
	Timestamp time.Time            `json:"timestamp"`
	Status    string               `json:"status"` // pending|matched|mismatch|error
}

// MatchedPosition is a position that matches between local and remote.
type MatchedPosition struct {
	Symbol       string  `json:"symbol"`
	Quantity     float64 `json:"quantity"`
	AvgPrice     float64 `json:"avg_price"`
	MarketPrice  float64 `json:"market_price"`
}

// MismatchedPosition is a position with differing values.
type MismatchedPosition struct {
	Symbol      string  `json:"symbol"`
	LocalQty    float64 `json:"local_qty"`
	RemoteQty   float64 `json:"remote_qty"`
	LocalCost   float64 `json:"local_cost"`
	RemoteCost  float64 `json:"remote_cost"`
	DiffQty     float64 `json:"diff_qty"`
	DiffCost    float64 `json:"diff_cost"`
}

// ReconciliationEngine compares local and broker positions.
type ReconciliationEngine struct{}

// NewReconciliationEngine creates a new reconciliation engine.
func NewReconciliationEngine() *ReconciliationEngine {
	return &ReconciliationEngine{}
}

// Reconcile compares local positions against remote (broker) positions and
// produces a detailed report of matches, mismatches, orphans, and missing.
func (re *ReconciliationEngine) Reconcile(broker string, local, remote []*Position) *ReconciliationReport {
	report := &ReconciliationReport{
		ID:        broker + "-" + time.Now().Format("20060102-150405"),
		Broker:    broker,
		Local:     local,
		Remote:    remote,
		Timestamp: time.Now(),
		Status:    "matched",
	}

	localMap := make(map[string]*Position)
	for _, p := range local {
		localMap[p.Symbol] = p
	}
	remoteMap := make(map[string]*Position)
	for _, p := range remote {
		remoteMap[p.Symbol] = p
	}

	// Check all remote positions
	for sym, rp := range remoteMap {
		lp, found := localMap[sym]
		if !found {
			report.Orphans = append(report.Orphans, rp)
			report.Status = "mismatch"
			continue
		}
		if approxEqual(lp.Quantity, rp.Quantity, 0.001) && approxEqual(lp.AvgPrice, rp.AvgPrice, 0.01) {
			report.Matches = append(report.Matches, MatchedPosition{
				Symbol: sym, Quantity: lp.Quantity,
				AvgPrice: lp.AvgPrice, MarketPrice: lp.MarketPrice,
			})
		} else {
			report.Mismatches = append(report.Mismatches, MismatchedPosition{
				Symbol: sym, LocalQty: lp.Quantity, RemoteQty: rp.Quantity,
				LocalCost: lp.AvgPrice, RemoteCost: rp.AvgPrice,
				DiffQty: lp.Quantity - rp.Quantity,
				DiffCost: lp.AvgPrice - rp.AvgPrice,
			})
			report.Status = "mismatch"
		}
	}

	// Check for local-only positions (missing from remote)
	for sym, lp := range localMap {
		if _, found := remoteMap[sym]; !found {
			report.Missing = append(report.Missing, lp)
			report.Status = "mismatch"
		}
	}

	return report
}

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) <= epsilon
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestReconciliationEngine_MatchExact -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/reconciliation.go internal/trading/reconciliation_test.go
git commit -m "feat(trading): add ReconciliationEngine with match/mismatch/orphan/missing detection"
```

---

### Task 3: Repository methods for saving/loading reconciliation reports + test

**Files:**
- Create: `internal/trading/repo.go`
- Test: `internal/trading/repo_test.go`

**Interfaces:**
- Consumes: `*sql.DB`, `ReconciliationReport` (from Task 2)
- Produces: `SaveReconciliationReport(db, report) error`, `GetReconciliationReports(db, broker, limit) ([]ReconciliationReport, error)`

- [ ] **Step 1: Write the failing test**

```go
// internal/trading/repo_test.go
package trading

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS reconciliation_reports (
			id TEXT PRIMARY KEY,
			broker TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			report_json TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending'
		);
		CREATE INDEX IF NOT EXISTS idx_recon_broker_time ON reconciliation_reports(broker, timestamp);
	`)
	if err != nil {
		t.Fatalf("create test table: %v", err)
	}
	return db
}

func TestSaveAndGetReconciliationReport(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	report := &ReconciliationReport{
		ID:        "test-001",
		Broker:    "alpaca",
		Timestamp: time.Now(),
		Status:    "matched",
		Matches: []MatchedPosition{
			{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
		},
	}

	if err := SaveReconciliationReport(db, report); err != nil {
		t.Fatalf("SaveReconciliationReport error: %v", err)
	}

	reports, err := GetReconciliationReports(db, "alpaca", 10)
	if err != nil {
		t.Fatalf("GetReconciliationReports error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if reports[0].ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", reports[0].ID)
	}
	if reports[0].Status != "matched" {
		t.Errorf("expected status matched, got %s", reports[0].Status)
	}
}

func TestGetReconciliationReports_Limit(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	for i := 0; i < 5; i++ {
		r := &ReconciliationReport{
			ID:     "r-" + string(rune('0'+i)),
			Broker: "alpaca", Timestamp: time.Now(), Status: "matched",
		}
		SaveReconciliationReport(db, r)
	}

	reports, err := GetReconciliationReports(db, "alpaca", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 3 {
		t.Errorf("expected 3 reports, got %d", len(reports))
	}

	// Test wrong broker returns empty
	reports, _ = GetReconciliationReports(db, "binance", 10)
	if len(reports) != 0 {
		t.Errorf("expected 0 reports for binance, got %d", len(reports))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestSaveAndGetReconciliationReport -v`
Expected: FAIL — repo.go doesn't exist yet

- [ ] **Step 3: Write minimal implementation**

`internal/trading/repo.go`:

```go
package trading

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SaveReconciliationReport persists a reconciliation report to SQLite.
func SaveReconciliationReport(db *sql.DB, report *ReconciliationReport) error {
	jsonBytes, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	_, err = db.Exec(
		`INSERT INTO reconciliation_reports (id, broker, timestamp, report_json, status)
		 VALUES (?, ?, ?, ?, ?)`,
		report.ID, report.Broker, report.Timestamp.Unix(), string(jsonBytes), report.Status,
	)
	if err != nil {
		return fmt.Errorf("insert reconciliation report: %w", err)
	}
	return nil
}

// GetReconciliationReports retrieves the most recent reports for a broker.
func GetReconciliationReports(db *sql.DB, broker string, limit int) ([]*ReconciliationReport, error) {
	rows, err := db.Query(
		`SELECT report_json, status FROM reconciliation_reports
		 WHERE broker = ? ORDER BY timestamp DESC LIMIT ?`,
		broker, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query reconciliation reports: %w", err)
	}
	defer rows.Close()

	var reports []*ReconciliationReport
	for rows.Next() {
		var jsonStr, status string
		if err := rows.Scan(&jsonStr, &status); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		var report ReconciliationReport
		if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
			return nil, fmt.Errorf("unmarshal report: %w", err)
		}
		report.Status = status
		reports = append(reports, &report)
	}
	if reports == nil {
		return []*ReconciliationReport{}, rows.Err()
	}
	return reports, rows.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestSaveAndGetReconciliationReport -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/repo.go internal/trading/repo_test.go
git commit -m "feat(trading): add repo functions for saving/loading reconciliation reports"
```

---

### Task 4: Engine ReconcileAll + scheduled reconciliation + test

**Files:**
- Modify: `internal/trading/engine.go`
- Test: `internal/trading/engine_test.go`

**Interfaces:**
- Consumes: `ReconciliationEngine.Reconcile()`, `Broker.GetPositions(ctx)`, `SaveReconciliationReport()`
- Produces: `Engine.ReconcileAll()`, `Engine.ScheduleReconciliation(interval time.Duration)`

- [ ] **Step 1: Write the failing test**

Add to `internal/trading/engine_test.go`:

```go
func TestEngine_ReconcileAllWithMockBroker(t *testing.T) {
	engine := NewEngine(100000.0)

	// Register a mock broker that returns known positions
	mockBroker := &mockBroker{
		positions: []*Position{
			{Symbol: "AAPL", Quantity: 100, AvgPrice: 175.0},
			{Symbol: "TSLA", Quantity: 50, AvgPrice: 245.0},
		},
		connected: true,
	}
	engine.RegisterBroker("alpaca", mockBroker)

	// Add matching positions to OMS
	engine.GetPaperEngine().GetOMS().PlaceOrder("AAPL", SideBuy, TypeMarket, "mock", 100, 175.0)
	engine.GetPaperEngine().GetOMS().PlaceOrder("TSLA", SideBuy, TypeMarket, "mock", 50, 245.0)

	// Need a test DB for repo calls
	report, err := engine.ReconcileAll(nil) // nil db should be handled
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "matched" {
		t.Errorf("expected matched, got %s", report.Status)
	}
}

type mockBroker struct {
	Broker
	positions []*Position
	connected bool
}

func (m *mockBroker) IsConnected() bool { return m.connected }
func (m *mockBroker) Name() string      { return "mock" }
func (m *mockBroker) GetPositions(ctx context.Context) ([]*Position, error) {
	return m.positions, nil
}
func (m *mockBroker) CancelAllOrders(ctx context.Context) error    { return nil }
func (m *mockBroker) CloseAllPositions(ctx context.Context) error  { return nil }
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_ReconcileAllWithMockBroker -v`
Expected: FAIL — Engine doesn't have ReconcileAll

- [ ] **Step 3: Write minimal implementation**

Add to `internal/trading/engine.go`:

```go
import (
	"database/sql"
)

// ReconcileAll runs reconciliation for all registered brokers.
// If db is non-nil, reports are persisted.
func (e *Engine) ReconcileAll(db *sql.DB) (*ReconciliationReport, error) {
	reconEngine := NewReconciliationEngine()
	var lastReport *ReconciliationReport

	for name, broker := range e.brokers {
		if !broker.IsConnected() {
			slog.Debug("skip reconciliation for disconnected broker", "broker", name)
			continue
		}

		ctx := context.Background()
		remote, err := broker.GetPositions(ctx)
		if err != nil {
			slog.Error("reconciliation: get remote positions failed", "broker", name, "error", err)
			continue
		}
		if remote == nil {
			remote = []*Position{}
		}

		local := e.paperEngine.GetOMS().GetAllPositions()
		if local == nil {
			local = []*Position{}
		}

		report := reconEngine.Reconcile(name, local, remote)
		lastReport = report

		if db != nil {
			if err := SaveReconciliationReport(db, report); err != nil {
				slog.Error("reconciliation: save report failed", "broker", name, "error", err)
			}
		}

		slog.Info("reconciliation complete",
			"broker", name,
			"status", report.Status,
			"matches", len(report.Matches),
			"mismatches", len(report.Mismatches),
			"orphans", len(report.Orphans),
			"missing", len(report.Missing),
		)
	}

	return lastReport, nil
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_ReconcileAllWithMockBroker -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/engine.go internal/trading/engine_test.go
git commit -m "feat(trading): add ReconcileAll to Engine with broker iteration and report persistence"
```

---

### Task 5: Frontend portfolio store reconciliation state + PositionPanel diff display + test

**Files:**
- Modify: `frontend/src/stores/portfolio.ts`
- Modify: `frontend/src/terminal/panels/PositionPanel.vue`
- Test: `frontend/src/stores/__tests__/portfolio.test.ts`
- Test: `frontend/src/terminal/panels/__tests__/PositionPanel.test.ts`

**Interfaces:**
- Consumes: IPC `ReconcileAll()`, `GetReconciliationReports(broker, limit)`
- Produces: `usePortfolioStore().reconciliation reports[]`, `usePortfolioStore().fetchReconciliation()`, PositionPanel shows diff column

- [ ] **Step 1: Write the failing test**

```typescript
// Add to frontend/src/stores/__tests__/portfolio.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePortfolioStore } from '../portfolio'

describe('usePortfolioStore - reconciliation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty reconciliation', () => {
    const store = usePortfolioStore()
    expect(store.reconciliation).toEqual({})
  })

  it('should fetch reconciliation reports', async () => {
    const store = usePortfolioStore()
    await store.fetchReconciliation()
    expect(store.reconciliation).toBeDefined()
  })
})
```

Add to `frontend/src/terminal/panels/__tests__/PositionPanel.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import PositionPanel from '../PositionPanel.vue'

describe('PositionPanel - reconciliation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should show reconciliation timestamp when available', () => {
    const wrapper = mount(PositionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    // The panel should have a reconciliation section
    expect(wrapper.find('[data-testid="recon-status"]').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/stores/__tests__/portfolio.test.ts src/terminal/panels/__tests__/PositionPanel.test.ts`
Expected: FAIL — portfolio store has no reconciliation property, PositionPanel panel doesn't have reconciliation UI

- [ ] **Step 3: Write minimal implementation**

Add to `frontend/src/stores/portfolio.ts`:

```typescript
// Add types:
export interface ReconReport {
  id: string
  broker: string
  timestamp: string
  status: string
  matches: Array<{symbol: string; quantity: number; avg_price: number; market_price: number}>
  mismatches: Array<{
    symbol: string; local_qty: number; remote_qty: number; diff_qty: number
  }>
  orphans: Array<{symbol: string; quantity: number}>
  missing: Array<{symbol: string; quantity: number}>
}

// Add to store state:
const reconciliation = ref<Record<string, ReconReport[]>>({})

// Add function:
async function fetchReconciliation() {
  try {
    const app = (window as any)?.go?.main?.App
    if (!app?.GetReconciliationReports) return
    const brokers = ['alpaca', 'binance', 'futu', 'ibkr']
    for (const broker of brokers) {
      const reports = await app.GetReconciliationReports(broker, 5)
      if (reports) {
        reconciliation.value[broker] = reports
      }
    }
  } catch (e) {
    console.error('[Portfolio] fetchReconciliation error:', e)
  }
}

// Add to return:
return {
  // ... existing returns ...
  reconciliation, fetchReconciliation,
}
```

Update `frontend/src/terminal/panels/PositionPanel.vue`:

```vue
<script setup lang="ts">
// Add to imports:
import { useTerminalStore } from '@/stores/terminal'

// Add to component:
const terminalStore = useTerminalStore()
const lastReconTime = computed(() => {
  const recon = store.reconciliation
  for (const broker of Object.keys(recon)) {
    const reports = recon[broker]
    if (reports && reports.length > 0) {
      return reports[0].timestamp
    }
  }
  return null
})

const reconStatus = computed(() => {
  const recon = store.reconciliation
  for (const broker of Object.keys(recon)) {
    const reports = recon[broker]
    if (reports && reports.length > 0) {
      return reports[0].status
    }
  }
  return 'unknown'
})

async function runReconciliation() {
  try {
    const app = (window as any)?.go?.main?.App
    if (app?.ReconcileAll) {
      await app.ReconcileAll()
      await store.fetchReconciliation()
    }
  } catch (e) {
    console.error('[Position] reconciliation failed:', e)
  }
}
</script>

<!-- Add to template, e.g. after summary-row -->
<div class="recon-bar" data-testid="recon-status">
  <span class="recon-label">对账状态:</span>
  <span :class="['recon-status', reconStatus]">{{ reconStatus }}</span>
  <span v-if="lastReconTime" class="recon-time">最后对账: {{ lastReconTime }}</span>
  <button class="recon-btn" @click="runReconciliation">同步对账</button>
</div>

<!-- In position-row, add diff column if reconciliation data exists -->
<div v-if="reconStatus === 'mismatch'" class="pos-diff">
  <span v-if="reconMismatch(pos.symbol)" class="diff-warning">⚠️ Diff: {{ reconMismatch(pos.symbol) }}</span>
</div>
```

Add a computed to detect symbol mismatches:

```typescript
function reconMismatch(symbol: string): string | null {
  const recon = store.reconciliation
  for (const broker of Object.keys(recon)) {
    const reports = recon[broker]
    if (!reports || reports.length === 0) continue
    for (const m of reports[0].mismatches) {
      if (m.symbol === symbol) {
        return `${m.diff_qty > 0 ? '+' : ''}${m.diff_qty}`
      }
    }
  }
  return null
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/stores/__tests__/portfolio.test.ts src/terminal/panels/__tests__/PositionPanel.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend/src/stores/portfolio.ts frontend/src/terminal/panels/PositionPanel.vue
git commit -m "feat(frontend): add reconciliation state to portfolio store; show diff in PositionPanel"
```

---

### Task 6: IPC binding for ReconcileAll + GetReconciliationReports + test

**Files:**
- Modify: `app_trading.go`
- Test: `app_trading_test.go`

**Interfaces:**
- Consumes: `Engine.ReconcileAll(db)`, `GetReconciliationReports(db, broker, limit)`
- Produces: `App.ReconcileAll()`, `App.GetReconciliationReports(broker, limit)`

- [ ] **Step 1: Write the failing test**

Add to `app_trading_test.go`:

```go
func TestReconcileAll_IPC(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	// With no brokers registered, should still return a nil report (not crash)
	report, err := app.ReconcileAll()
	if err != nil {
		t.Fatalf("ReconcileAll error: %v", err)
	}
	_ = report // nil is acceptable with no brokers
}

func TestGetReconciliationReports_IPC(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	reports, err := app.GetReconciliationReports("alpaca", 10)
	if err != nil {
		t.Fatalf("GetReconciliationReports error: %v", err)
	}
	if reports == nil {
		t.Error("expected non-nil reports slice")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestReconcileAll_IPC -v`
Expected: FAIL — App has no ReconcileAll or GetReconciliationReports methods

- [ ] **Step 3: Write minimal implementation**

Add to `app_trading.go`:

```go
import (
	"quantflow/internal/storage"
)

// ReconcileAll triggers reconciliation for all brokers and returns the report.
func (a *App) ReconcileAll() (*trading.ReconciliationReport, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	db := a.getDB()
	return a.engine.ReconcileAll(db)
}

// GetReconciliationReports returns recent reconciliation reports for a broker.
func (a *App) GetReconciliationReports(broker string, limit int) ([]*trading.ReconciliationReport, error) {
	db := a.getDB()
	if db == nil {
		return []*trading.ReconciliationReport{}, nil
	}
	return trading.GetReconciliationReports(db, broker, limit)
}

// getDB returns the SQLite database handle from the app context.
func (a *App) getDB() *sql.DB {
	if a.db != nil {
		return a.db
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestReconcileAll_IPC -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add app_trading.go app_trading_test.go
git commit -m "feat(app): add ReconcileAll and GetReconciliationReports IPC bindings"
```

---

### Execution Order

```
Task 1 (migration SQL) → Task 2 (reconciliation engine) → Task 3 (repo layer)
  → Task 4 (Engine.ReconcileAll) → Task 5 (frontend) → Task 6 (IPC bindings)
```

Tasks 1-4 are sequential. Task 5 can begin after Task 2 (needs report types). Task 6 wires frontend to backend and can run in parallel with Task 5.

# Backtest History Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist backtest results from workflow execution to SQLite, provide terminal panel for browsing/deleting historical backtests with full detail view.

**Architecture:** New `backtest_results` SQLite table stores result + OHLCV data as JSON columns, indexed by metrics for sorting. `executionSaver` callback in app.go detects backtest node outputs and persists them. New `BacktestHistoryPanel.vue` lists history with delete. Existing `BacktestResultPanel.vue` enhanced to load from stored data via `storeId` param.

**Tech Stack:** Go 1.22+, SQLite (WAL), Vue 3 + TypeScript, Wails v3

## Global Constraints

- SQLite is the only database — no PostgreSQL/Redis
- All Go backtest engines must persist through the same hook
- Frontend panel registry pattern must be followed
- CHANGELOG.md must be updated every change
- Version date must be checked before every commit

---

### Task 1: Migration SQL + BacktestRepo

**Files:**
- Create: `internal/storage/migrations/015_backtest_results.sql`
- Modify: `internal/storage/migrate.go` — add migration to list
- Create: `internal/storage/backtest_repo.go` — full repository
- Create: `internal/storage/backtest_repo_test.go` — tests

**Interfaces:**
- Produces: `BacktestRepo` struct with `Save`, `List`, `GetByID`, `Delete` methods

- [ ] **Step 1: Create the migration SQL**

```sql
-- 015_backtest_results.sql
-- Store backtest results from workflow executions for terminal-mode history.

CREATE TABLE IF NOT EXISTS backtest_results (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id          TEXT    NOT NULL,
    workflow_name   TEXT    NOT NULL DEFAULT '',
    strategy_name   TEXT    NOT NULL DEFAULT '',
    symbol          TEXT    NOT NULL DEFAULT '',
    engine_type     TEXT    NOT NULL DEFAULT '',

    total_return    REAL    NOT NULL DEFAULT 0,
    cagr            REAL    NOT NULL DEFAULT 0,
    max_drawdown    REAL    NOT NULL DEFAULT 0,
    sharpe_ratio    REAL    NOT NULL DEFAULT 0,
    sortino_ratio   REAL    NOT NULL DEFAULT 0,
    calmar_ratio    REAL    NOT NULL DEFAULT 0,
    win_rate        REAL    NOT NULL DEFAULT 0,
    profit_factor   REAL    NOT NULL DEFAULT 0,
    total_trades    INTEGER NOT NULL DEFAULT 0,

    config_json     TEXT    NOT NULL DEFAULT '{}',
    equity_curve    TEXT    NOT NULL DEFAULT '[]',
    trades_json     TEXT    NOT NULL DEFAULT '[]',
    ohlcv_data      TEXT    NOT NULL DEFAULT '[]',

    started_at      TEXT    NOT NULL,
    finished_at     TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_bt_results_finished ON backtest_results(finished_at DESC);
CREATE INDEX idx_bt_results_symbol ON backtest_results(symbol);
```

- [ ] **Step 2: Register migration in migrate.go**

Read `internal/storage/migrate.go` and add `015_backtest_results.sql` to the list, likely after the last `case` in the migration switch or however the slice is defined. Append `.sql` filename to the ordered list.

Find the line where migrations are defined (likely a slice or switch), add `"015_backtest_results.sql"`.

- [ ] **Step 3: Write the failing test**

```go
package storage

import (
	"context"
	"testing"

	"quantflow/internal/backtest"
)

func TestBacktestRepo_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBacktestRepo(db)

	err := repo.Save(context.Background(), &StoredBacktest{
		RunID:        "run-1",
		WorkflowName: "Test Workflow",
		StrategyName: "SMA Cross",
		Symbol:       "600519",
		EngineType:   "cn",
		Metrics: backtest.Metrics{
			TotalReturn: 0.15,
			CAGR:        0.12,
			MaxDrawdown: -0.08,
			SharpeRatio: 1.5,
			TotalTrades: 10,
		},
		ConfigJSON:  `{"initial_cash":100000}`,
		EquityCurve: `[{"date":"2024-01-01","equity":100000}]`,
		TradesJSON:  `[]`,
		OHLCVData:   `[]`,
		StartedAt:   "2024-01-01",
		FinishedAt:  "2024-12-31",
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.StrategyName != "SMA Cross" {
		t.Errorf("expected SMA Cross, got %s", got.StrategyName)
	}
}
```

```go
func TestBacktestRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBacktestRepo(db)

	for i := 0; i < 3; i++ {
		repo.Save(context.Background(), &StoredBacktest{
			RunID: fmt.Sprintf("run-%d", i), WorkflowName: "WF", FinishedAt: "2024-01-0%d" + fmt.Sprintf("%d", i+1),
		})
	}

	results, err := repo.List(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3, got %d", len(results))
	}
}
```

```go
func TestBacktestRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewBacktestRepo(db)

	repo.Save(context.Background(), &StoredBacktest{RunID: "run-1", WorkflowName: "WF", FinishedAt: "2024-01-01"})
	err := repo.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(context.Background(), 1)
	if err == nil {
		t.Error("expected error for deleted record")
	}
}
```

Need a `setupTestDB` helper:
```go
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	// Run migration
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS backtest_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		run_id TEXT NOT NULL,
		workflow_name TEXT NOT NULL DEFAULT '',
		strategy_name TEXT NOT NULL DEFAULT '',
		symbol TEXT NOT NULL DEFAULT '',
		engine_type TEXT NOT NULL DEFAULT '',
		total_return REAL NOT NULL DEFAULT 0,
		cagr REAL NOT NULL DEFAULT 0,
		max_drawdown REAL NOT NULL DEFAULT 0,
		sharpe_ratio REAL NOT NULL DEFAULT 0,
		sortino_ratio REAL NOT NULL DEFAULT 0,
		calmar_ratio REAL NOT NULL DEFAULT 0,
		win_rate REAL NOT NULL DEFAULT 0,
		profit_factor REAL NOT NULL DEFAULT 0,
		total_trades INTEGER NOT NULL DEFAULT 0,
		config_json TEXT NOT NULL DEFAULT '{}',
		equity_curve TEXT NOT NULL DEFAULT '[]',
		trades_json TEXT NOT NULL DEFAULT '[]',
		ohlcv_data TEXT NOT NULL DEFAULT '[]',
		started_at TEXT NOT NULL,
		finished_at TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/... -run TestBacktestRepo -v -count=1
```
Expected: FAIL (BacktestRepo not defined)

- [ ] **Step 5: Write BacktestRepo implementation**

```go
package storage

import (
	"context"
	"database/sql"
	"fmt"

	"quantflow/internal/backtest"
)

type StoredBacktest struct {
	ID           int              `json:"id"`
	RunID        string           `json:"run_id"`
	WorkflowName string           `json:"workflow_name"`
	StrategyName string           `json:"strategy_name"`
	Symbol       string           `json:"symbol"`
	EngineType   string           `json:"engine_type"`
	Metrics      backtest.Metrics `json:"metrics"`
	ConfigJSON   string           `json:"config_json"`
	EquityCurve  string           `json:"equity_curve"`
	TradesJSON   string           `json:"trades_json"`
	OHLCVData    string           `json:"ohlcv_data"`
	StartedAt    string           `json:"started_at"`
	FinishedAt   string           `json:"finished_at"`
	CreatedAt    string           `json:"created_at"`
}

type StoredBacktestSummary struct {
	ID           int              `json:"id"`
	RunID        string           `json:"run_id"`
	WorkflowName string           `json:"workflow_name"`
	StrategyName string           `json:"strategy_name"`
	Symbol       string           `json:"symbol"`
	EngineType   string           `json:"engine_type"`
	Metrics      backtest.Metrics `json:"metrics"`
	StartedAt    string           `json:"started_at"`
	FinishedAt   string           `json:"finished_at"`
	CreatedAt    string           `json:"created_at"`
}

type BacktestRepo struct {
	db *sql.DB
}

func NewBacktestRepo(db *sql.DB) *BacktestRepo {
	return &BacktestRepo{db: db}
}

func (r *BacktestRepo) Save(ctx context.Context, bt *StoredBacktest) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO backtest_results
			(run_id, workflow_name, strategy_name, symbol, engine_type,
			 total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio, calmar_ratio, win_rate, profit_factor, total_trades,
			 config_json, equity_curve, trades_json, ohlcv_data,
			 started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		bt.RunID, bt.WorkflowName, bt.StrategyName, bt.Symbol, bt.EngineType,
		bt.Metrics.TotalReturn, bt.Metrics.CAGR, bt.Metrics.MaxDrawdown,
		bt.Metrics.SharpeRatio, bt.Metrics.SortinoRatio, bt.Metrics.CalmarRatio,
		bt.Metrics.WinRate, bt.Metrics.ProfitFactor, bt.Metrics.TotalTrades,
		bt.ConfigJSON, bt.EquityCurve, bt.TradesJSON, bt.OHLCVData,
		bt.StartedAt, bt.FinishedAt)
	return err
}

func (r *BacktestRepo) List(ctx context.Context, limit, offset int) ([]StoredBacktestSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, run_id, workflow_name, strategy_name, symbol, engine_type,
		       total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio, calmar_ratio, win_rate, profit_factor, total_trades,
		       started_at, finished_at, created_at
		FROM backtest_results
		ORDER BY finished_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StoredBacktestSummary
	for rows.Next() {
		var s StoredBacktestSummary
		if err := rows.Scan(&s.ID, &s.RunID, &s.WorkflowName, &s.StrategyName, &s.Symbol, &s.EngineType,
			&s.Metrics.TotalReturn, &s.Metrics.CAGR, &s.Metrics.MaxDrawdown,
			&s.Metrics.SharpeRatio, &s.Metrics.SortinoRatio, &s.Metrics.CalmarRatio,
			&s.Metrics.WinRate, &s.Metrics.ProfitFactor, &s.Metrics.TotalTrades,
			&s.StartedAt, &s.FinishedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *BacktestRepo) GetByID(ctx context.Context, id int) (*StoredBacktest, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, run_id, workflow_name, strategy_name, symbol, engine_type,
		       total_return, cagr, max_drawdown, sharpe_ratio, sortino_ratio, calmar_ratio, win_rate, profit_factor, total_trades,
		       config_json, equity_curve, trades_json, ohlcv_data,
		       started_at, finished_at, created_at
		FROM backtest_results WHERE id = ?`, id)

	var s StoredBacktest
	if err := row.Scan(&s.ID, &s.RunID, &s.WorkflowName, &s.StrategyName, &s.Symbol, &s.EngineType,
		&s.Metrics.TotalReturn, &s.Metrics.CAGR, &s.Metrics.MaxDrawdown,
		&s.Metrics.SharpeRatio, &s.Metrics.SortinoRatio, &s.Metrics.CalmarRatio,
		&s.Metrics.WinRate, &s.Metrics.ProfitFactor, &s.Metrics.TotalTrades,
		&s.ConfigJSON, &s.EquityCurve, &s.TradesJSON, &s.OHLCVData,
		&s.StartedAt, &s.FinishedAt, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *BacktestRepo) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM backtest_results WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("backtest result %d not found", id)
	}
	return nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/... -run TestBacktestRepo -v -count=1
```
Expected: PASS (3/3)

- [ ] **Step 7: Commit**

```bash
git add internal/storage/migrations/015_backtest_results.sql internal/storage/migrate.go internal/storage/backtest_repo.go internal/storage/backtest_repo_test.go
git commit -m "feat(storage): add backtest_results table and BacktestRepo [spec:2026-07-04-backtest-history-persistence]"
```

---

### Task 2: App.go — executionSaver hook + Wails bindings

**Files:**
- Modify: `app.go` — add BacktestRepo, executionSaver hook, 3 new bindings

**Interfaces:**
- Consumes: `BacktestRepo` from Task 1, `workflow.ExecutionResult`, `workflow.Workflow` (for edge tracing)
- Produces: `ListBacktestHistory(limit, offset)`, `GetStoredBacktestResult(id)`, `DeleteBacktestResult(id)` Wails bindings

- [ ] **Step 1: Read app.go to understand ServiceStartup and executionSaver structure**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && head -50 app.go
```

Look for:
- `type App struct` fields
- `ServiceStartup` method
- `SetExecutionSaver` call
- Any existing repo fields (e.g., `execRepo`)

- [ ] **Step 2: Add BacktestRepo field to App struct**

Find the App struct definition, add:
```go
btRepo *storage.BacktestRepo
```

- [ ] **Step 3: Initialize BacktestRepo in ServiceStartup**

After existing repo init lines (near where `execRepo` is initialized), add:
```go
a.btRepo = storage.NewBacktestRepo(a.db)
```

- [ ] **Step 4: Add executionSaver hook logic**

Find the `SetExecutionSaver` block (the callback that saves execution results). Inside it, after saving the execution record, add backtest-specific persistence.

The logic:
```go
// Persist backtest results
for _, nr := range result.NodeResults {
    if nr.NodeType != "backtest" || nr.Status != "success" {
        continue
    }
    btResult, err := extractBacktestResult(nr.Outputs)
    if err != nil {
        slog.Warn("extract backtest result", "node", nr.NodeID, "error", err)
        continue
    }
    // Find strategy_name from params or node outputs
    strategyName := extractString(nr.Outputs, "strategy_name")
    if strategyName == "" {
        strategyName = wf.Name
    }

    // Find data_loader OHLCV by tracing edges
    ohlcvJSON := "[]"
    ohlcv := findUpstreamOhlcv(wf, nr.NodeID, result.NodeResults)
    if ohlcv != nil {
        if b, err := json.Marshal(ohlcv); err == nil {
            ohlcvJSON = string(b)
        }
    }

    configJSON, _ := json.Marshal(btResult.Config)
    equityJSON, _ := json.Marshal(btResult.EquityCurve)
    tradesJSON, _ := json.Marshal(btResult.Trades)

    bt := &storage.StoredBacktest{
        RunID:        runID,
        WorkflowName: wf.Name,
        StrategyName: strategyName,
        Symbol:       extractString(nr.Outputs, "symbol"),
        EngineType:   extractString(nr.Outputs, "engine_type"),
        Metrics:      btResult.Metrics,
        ConfigJSON:   string(configJSON),
        EquityCurve:  string(equityJSON),
        TradesJSON:   string(tradesJSON),
        OHLCVData:    ohlcvJSON,
        StartedAt:    result.StartedAt.Format(time.RFC3339),
        FinishedAt:   result.FinishedAt.Format(time.RFC3339),
    }
    if err := a.btRepo.Save(context.Background(), bt); err != nil {
        slog.Error("save backtest result", "error", err)
    }
}
```

And the helper functions:
```go
func extractBacktestResult(outputs map[string]any) (*backtest.Result, error) {
    raw, ok := outputs["result"]
    if !ok {
        return nil, fmt.Errorf("no 'result' key in backtest outputs")
    }
    // Try direct type assertion first, then JSON round-trip
    if r, ok := raw.(*backtest.Result); ok {
        return r, nil
    }
    // JSON marshal/unmarshal for map[string]any form
    b, err := json.Marshal(raw)
    if err != nil {
        return nil, err
    }
    var r backtest.Result
    if err := json.Unmarshal(b, &r); err != nil {
        return nil, err
    }
    return &r, nil
}

func extractString(outputs map[string]any, key string) string {
    if v, ok := outputs[key]; ok {
        if s, ok := v.(string); ok {
            return s
        }
    }
    return ""
}

func findUpstreamOhlcv(wf *workflow.Workflow, backtestNodeID string, nodeResults []workflow.NodeResult) any {
    // Find edge: backtestNode.ohlcv_data ← sourceNode.ohlcv
    for _, edge := range wf.Edges {
        if edge.ToNode == backtestNodeID && edge.ToPort == "ohlcv_data" {
            // Find the source node's output
            for _, nr := range nodeResults {
                if nr.NodeID == edge.FromNode {
                    if ohlcv, ok := nr.Outputs["ohlcv"]; ok {
                        return ohlcv
                    }
                }
            }
        }
    }
    return nil
}
```

- [ ] **Step 5: Add Wails bindings**

Find the area where other binding methods are defined (e.g., `GetExecutionHistory`). Add three new methods:

```go
func (a *App) ListBacktestHistory(ctx context.Context, limit, offset int) ([]storage.StoredBacktestSummary, error) {
    if limit <= 0 || limit > 100 {
        limit = 50
    }
    return a.btRepo.List(ctx, limit, offset)
}

func (a *App) GetStoredBacktestResult(ctx context.Context, id int) (*storage.StoredBacktest, error) {
    return a.btRepo.GetByID(ctx, id)
}

func (a *App) DeleteBacktestResult(ctx context.Context, id int) error {
    return a.btRepo.Delete(ctx, id)
}
```

- [ ] **Step 6: Add imports**

Ensure app.go imports `"quantflow/internal/backtest"`, `"quantflow/internal/storage"`, `"quantflow/internal/workflow"`, `"encoding/json"`, `"log/slog"`, `"time"`, `"context"`, `"fmt"`.

- [ ] **Step 7: Build to verify**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./... 2>&1 | grep -v "ld: warning"
```
Expected: no errors (warnings are fine)

- [ ] **Step 8: Commit**

```bash
git add app.go
git commit -m "feat(app): persist backtest results from workflow execution, add Wails bindings [spec:2026-07-04-backtest-history-persistence]"
```

---

### Task 3: BacktestHistoryPanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/BacktestHistoryPanel.vue`

- [ ] **Step 1: Create the Vue panel**

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'

interface BacktestSummary {
  id: number
  run_id: string
  workflow_name: string
  strategy_name: string
  symbol: string
  engine_type: string
  metrics: {
    total_return: number
    cagr: number
    max_drawdown: number
    sharpe_ratio: number
    sortino_ratio: number
    calmar_ratio: number
    win_rate: number
    profit_factor: number
    total_trades: number
    annual_volatility: number
  }
  started_at: string
  finished_at: string
  created_at: string
}

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()
const terminal = useTerminalStore()

const items = ref<BacktestSummary[]>([])
const loading = ref(false)
const selectedIds = ref<Set<number>>(new Set())
const sortField = ref<'finished_at' | 'total_return' | 'sharpe_ratio'>('finished_at')
const sortAsc = ref(false)

async function loadData() {
  loading.value = true
  try {
    const res = await (window as any).go.main.App.ListBacktestHistory(100, 0)
    items.value = res || []
  } catch (e) {
    console.error('ListBacktestHistory failed:', e)
  } finally {
    loading.value = false
  }
}

function sortedItems() {
  const sorted = [...items.value]
  sorted.sort((a, b) => {
    let va: any, vb: any
    if (sortField.value === 'finished_at') {
      va = a.finished_at; vb = b.finished_at
    } else {
      va = a.metrics[sortField.value]; vb = b.metrics[sortField.value]
    }
    if (sortAsc.value) return va > vb ? 1 : -1
    return va < vb ? 1 : -1
  })
  return sorted
}

function toggleSort(field: typeof sortField.value) {
  if (sortField.value === field) sortAsc.value = !sortAsc.value
  else { sortField.value = field; sortAsc.value = false }
}

function toggleSelect(id: number) {
  if (selectedIds.value.has(id)) selectedIds.value.delete(id)
  else selectedIds.value.add(id)
}

function openDetail(id: number) {
  terminal.openPanel('backtest-result', { storeId: id })
}

async function deleteSelected() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  if (!confirm(`确定删除选中的 ${ids.length} 条回测记录？`)) return
  for (const id of ids) {
    try {
      await (window as any).go.main.App.DeleteBacktestResult(id)
    } catch (e) {
      console.error('DeleteBacktestResult failed:', e)
    }
  }
  selectedIds.value.clear()
  await loadData()
}

async function deleteSingle(id: number) {
  if (!confirm('确定删除此回测记录？')) return
  try {
    await (window as any).go.main.App.DeleteBacktestResult(id)
    selectedIds.value.delete(id)
    await loadData()
  } catch (e) {
    console.error('DeleteBacktestResult failed:', e)
  }
}

function fmt(v: number, decimals = 2) {
  if (v == null || isNaN(v)) return '-'
  return v.toFixed(decimals)
}
function pct(v: number) {
  if (v == null || isNaN(v)) return '-'
  return (v * 100).toFixed(2) + '%'
}

onMounted(loadData)
</script>

<template>
  <div class="backtest-history-panel">
    <div class="panel-toolbar">
      <span class="panel-title">回测历史 ({{ items.length }})</span>
      <div class="toolbar-actions">
        <button v-if="selectedIds.size > 0" class="btn btn-danger btn-sm" @click="deleteSelected">
          删除选中 ({{ selectedIds.size }})
        </button>
        <button class="btn btn-sm" @click="loadData">刷新</button>
      </div>
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="items.length === 0" class="empty">暂无回测记录</div>
    <table v-else class="history-table">
      <thead>
        <tr>
          <th class="col-check"><input type="checkbox" @change="toggleSelectAll" /></th>
          <th class="col-date sortable" @click="toggleSort('finished_at')">
            日期 {{ sortField === 'finished_at' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-wf">工作流</th>
          <th class="col-strategy">策略</th>
          <th class="col-symbol">标的</th>
          <th class="col-return sortable" @click="toggleSort('total_return')">
            收益率 {{ sortField === 'total_return' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-sharpe sortable" @click="toggleSort('sharpe_ratio')">
            Sharpe {{ sortField === 'sharpe_ratio' ? (sortAsc ? '↑' : '↓') : '' }}
          </th>
          <th class="col-trades">交易</th>
          <th class="col-actions">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in sortedItems()" :key="item.id"
            :class="{ selected: selectedIds.has(item.id) }"
            @click="openDetail(item.id)">
          <td class="col-check" @click.stop><input type="checkbox" :checked="selectedIds.has(item.id)" @change="toggleSelect(item.id)" /></td>
          <td>{{ item.finished_at?.slice(0, 10) }}</td>
          <td>{{ item.workflow_name }}</td>
          <td>{{ item.strategy_name }}</td>
          <td>{{ item.symbol }}</td>
          <td :class="item.metrics.total_return >= 0 ? 'positive' : 'negative'">{{ pct(item.metrics.total_return) }}</td>
          <td>{{ fmt(item.metrics.sharpe_ratio) }}</td>
          <td>{{ item.metrics.total_trades }}</td>
          <td class="col-actions" @click.stop>
            <button class="btn-icon" title="删除" @click="deleteSingle(item.id)">🗑</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.backtest-history-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 8px;
}
.panel-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.toolbar-actions { display: flex; gap: 4px; }
.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.history-table th {
  text-align: left;
  padding: 6px 4px;
  border-bottom: 1px solid var(--border-color, #334);
  font-weight: 600;
  white-space: nowrap;
}
.sortable { cursor: pointer; user-select: none; }
.sortable:hover { color: var(--accent, #3b82f6); }
.history-table td {
  padding: 4px;
  border-bottom: 1px solid var(--border-color, #2a2a3a);
  cursor: pointer;
}
.history-table tr:hover { background: var(--hover-bg, rgba(59,130,246,0.08)); }
.history-table tr.selected { background: var(--selected-bg, rgba(59,130,246,0.15)); }
.col-check { width: 24px; }
.col-date { width: 90px; }
.col-wf { min-width: 100px; }
.col-strategy { min-width: 80px; }
.col-symbol { width: 70px; }
.col-return { width: 80px; text-align: right; }
.col-sharpe { width: 70px; text-align: right; }
.col-trades { width: 50px; text-align: right; }
.col-actions { width: 40px; text-align: center; }
.positive { color: #ef4444; }
.negative { color: #22c55e; }
.loading, .empty { padding: 20px; text-align: center; color: #888; }
.btn-sm { padding: 2px 8px; font-size: 11px; }
.btn-danger { background: #dc2626; color: #fff; border: none; border-radius: 3px; }
.btn-icon { background: none; border: none; cursor: pointer; padding: 2px 4px; font-size: 13px; }
</style>
```

Note: The `toggleSelectAll` ref is a template reference. Change the checkbox to use a computed pattern or add a method:

```typescript
function toggleSelectAll(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  if (checked) {
    selectedIds.value = new Set(items.value.map(i => i.id))
  } else {
    selectedIds.value.clear()
  }
}
```

Add this after `deleteSingle`.

- [ ] **Step 2: Verify build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | head -20
```
Expected: no errors related to our new file

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/BacktestHistoryPanel.vue
git commit -m "feat(frontend): BacktestHistoryPanel with list, sort, multi-select delete [spec:2026-07-04-backtest-history-persistence]"
```

---

### Task 4: BacktestResultPanel.vue — storeId support

**Files:**
- Modify: `frontend/src/terminal/panels/BacktestResultPanel.vue`

- [ ] **Step 1: Read the current BacktestResultPanel.vue**

```bash
wc -l frontend/src/terminal/panels/BacktestResultPanel.vue
```

- [ ] **Step 2: Add storeId parameter support**

At the top of `<script setup>`, after the props definition, add:

```typescript
const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

// Support both runtime mode (from workflow.nodeOutputs) and history mode (from storeId)
const storeId = computed(() => props.params?.storeId as number | undefined)
const storedResult = ref<any>(null)
const loadingStored = ref(false)
```

Replace the existing `findBacktestOutput()` with a conditional that checks `storeId` first:

```typescript
// btOutput = computed from storedResult if storeId, otherwise from workflow store
const btOutput = computed(() => {
  if (storeId.value) return storedResult.value
  // fallback to runtime mode
  for (const [, outputs] of workflow.nodeOutputs) {
    if (outputs && outputs.equity_curve) return outputs
  }
  return null
})

// Load from backend when storeId is set
watch(storeId, async (id) => {
  if (!id) return
  loadingStored.value = true
  try {
    const res = await (window as any).go.main.App.GetStoredBacktestResult(id)
    if (res) {
      storedResult.value = {
        equity_curve: JSON.parse(res.equity_curve || '[]').map((p: any) => p.equity),
        metrics: res.metrics,
        trades: JSON.parse(res.trades_json || '[]'),
        storedConfig: JSON.parse(res.config_json || '{}'),
        storedOhlcv: JSON.parse(res.ohlcv_data || '[]'),
      }
    }
  } catch (e) {
    console.error('GetStoredBacktestResult failed:', e)
  } finally {
    loadingStored.value = false
  }
}, { immediate: true })
```

Update the `findDataLoaderOhlcv()` function to also check stored OHLCV:

```typescript
function findDataLoaderOhlcv(): any[] | null {
  if (storeId.value && storedResult.value?.storedOhlcv?.length) {
    return storedResult.value.storedOhlcv
  }
  for (const edge of workflow.edges) {
    if (edge.targetHandle === 'ohlcv_data') {
      const srcOutputs = workflow.nodeOutputs.get(edge.source)
      if (srcOutputs?.ohlcv && Array.isArray(srcOutputs.ohlcv)) return srcOutputs.ohlcv
    }
  }
  return null
}
```

Add a loading state in the template (if `loadingStored`, show spinner).

- [ ] **Step 3: Verify build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | head -20
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/BacktestResultPanel.vue
git commit -m "feat(frontend): BacktestResultPanel loads from storeId for history mode [spec:2026-07-04-backtest-history-persistence]"
```

---

### Task 5: Registry + final wiring

**Files:**
- Modify: `frontend/src/terminal/panels/registry.ts`

- [ ] **Step 1: Register backtest-history panel**

Add to registry.ts:
```typescript
register('backtest-history', () => import('./BacktestHistoryPanel.vue'),
  { label: '回测历史', category: '量化分析', description: '浏览和管理历史回测记录' })
```

- [ ] **Step 2: Verify frontend build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1
```

- [ ] **Step 3: Verify Go backend build**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go build ./... 2>&1 | grep -v "ld: warning"
```

- [ ] **Step 4: Run all storage tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/storage/... -v -count=1 2>&1 | tail -15
```

Expected: tests in storage pass (including new BacktestRepo tests)

- [ ] **Step 5: Update CHANGELOG.md**

Add entry:
```markdown
### Added
- [Storage] `backtest_results` SQLite table (migration 015) with indexed metrics columns for sorting
- [Storage] `BacktestRepo` — Save/List/GetByID/Delete for backtest results
- [Terminal] `BacktestHistoryPanel` — 历史回测浏览面板，支持按日期/收益率/Sharpe排序、多选批量删除
- [Terminal] BacktestResultPanel 支持 `storeId` 参数加载历史回测数据，保留完整 K 线图+买卖点+净值曲线
- [Workflow] 工作流执行完成后自动将 backtest 节点结果（含上游 OHLCV 数据）持久化到 SQLite
```

- [ ] **Step 6: Update version to 2026.7.4 if needed**

Check `frontend/package.json`, `README.md` version badge, ensure they reflect current date.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/panels/registry.ts CHANGELOG.md
git add frontend/package.json README.md  # if version updated
git commit -m "feat: register BacktestHistoryPanel, update CHANGELOG [spec:2026-07-04-backtest-history-persistence]"
```

---

## Self-Review Checklist

- [x] Spec coverage: migration, BacktestRepo, app.go hook, Wails bindings, history panel, result panel enhancement, registry — all spec requirements covered including delete
- [x] No placeholders: every step has complete code
- [x] Type consistency: StoredBacktest/StoredBacktestSummary structs match between storage package and frontend interface; storeId param type matches between panel open call and consume
- [x] All methods referenced exist: Save/List/GetByID/Delete on BacktestRepo, ListBacktestHistory/GetStoredBacktestResult/DeleteBacktestResult on App

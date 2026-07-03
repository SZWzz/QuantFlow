# Backtest History Persistence

## Motivation

当前回测结果仅在工作流运行时驻留于前端 `nodeOutputs`（内存），切换工作流或刷新页面后丢失。用户需要在终端模式下浏览、排序、查看所有历史回测记录，包括完整的 K 线图、净值曲线、指标网格和交易记录。

## Design

### Data Flow

```
Workflow Run completes
  → executionSaver callback (app.go)
    → detect backtest node in NodeResults
      → extract backtest result + upstream ohlcv data
        → SaveBacktestResult() → SQLite backtest_results table
          → Terminal BacktestHistoryPanel queries table
            → click row → BacktestResultPanel loads from storeId
```

### 1. SQLite Schema (Migration 015)

```sql
CREATE TABLE IF NOT EXISTS backtest_results (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id          TEXT    NOT NULL,
    workflow_name   TEXT    NOT NULL DEFAULT '',
    strategy_name   TEXT    NOT NULL DEFAULT '',
    symbol          TEXT    NOT NULL DEFAULT '',
    engine_type     TEXT    NOT NULL DEFAULT '',

    -- Sortable metric columns
    total_return    REAL    NOT NULL DEFAULT 0,
    cagr            REAL    NOT NULL DEFAULT 0,
    max_drawdown    REAL    NOT NULL DEFAULT 0,
    sharpe_ratio    REAL    NOT NULL DEFAULT 0,
    sortino_ratio   REAL    NOT NULL DEFAULT 0,
    calmar_ratio    REAL    NOT NULL DEFAULT 0,
    win_rate        REAL    NOT NULL DEFAULT 0,
    profit_factor   REAL    NOT NULL DEFAULT 0,
    total_trades    INTEGER NOT NULL DEFAULT 0,

    -- Full data (JSON blobs for rendering)
    config_json     TEXT    NOT NULL DEFAULT '{}',
    equity_curve    TEXT    NOT NULL DEFAULT '[]',
    trades_json     TEXT    NOT NULL DEFAULT '[]',
    ohlcv_data      TEXT    NOT NULL DEFAULT '[]',

    -- Timeline
    started_at      TEXT    NOT NULL,
    finished_at     TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_bt_results_finished ON backtest_results(finished_at DESC);
CREATE INDEX idx_bt_results_symbol ON backtest_results(symbol);
```

### 2. New/Modified Go Files

#### `internal/storage/backtest_repo.go` (new)

```go
type StoredBacktest struct {
    ID           int               `json:"id"`
    RunID        string            `json:"run_id"`
    WorkflowName string            `json:"workflow_name"`
    StrategyName string            `json:"strategy_name"`
    Symbol       string            `json:"symbol"`
    EngineType   string            `json:"engine_type"`
    Metrics      backtest.Metrics  `json:"metrics"`       // embedded
    ConfigJSON   string            `json:"config_json"`   // raw JSON
    EquityCurve  string            `json:"equity_curve"`  // JSON array of EquityPoint
    TradesJSON   string            `json:"trades_json"`   // JSON array of TradeRecord
    OHLCVData    string            `json:"ohlcv_data"`    // JSON array of OHLCVBar
    StartedAt    string            `json:"started_at"`
    FinishedAt   string            `json:"finished_at"`
    CreatedAt    string            `json:"created_at"`
}

func (r *BacktestRepo) Save(ctx context.Context, result *StoredBacktest) error
func (r *BacktestRepo) List(ctx context.Context, limit, offset int) ([]StoredBacktestSummary, error)
func (r *BacktestRepo) GetByID(ctx context.Context, id int) (*StoredBacktest, error)
```

`StoredBacktestSummary` is a lighter struct with only metadata + metrics (no JSON blobs) for list display.

#### `app.go` modifications

1. Store `BacktestRepo` on `App` struct, init in `ServiceStartup`
2. In `executionSaver` callback: iterate `result.NodeResults`, find nodes with type `"backtest"`, extract outputs + upstream data_loader OHLCV, call `backtestRepo.Save()`
3. New bindings:
   - `ListBacktestHistory(limit, offset)` → `[]StoredBacktestSummary`
   - `GetStoredBacktestResult(id)` → `*StoredBacktest`

#### `internal/storage/migrate.go`

Add `015_backtest_results.sql` to embedded migrations.

### 3. New/Modified Frontend Files

#### `BacktestHistoryPanel.vue` (new terminal panel)

- Table view: columns = Date | Workflow | Strategy | Symbol | Return | Sharpe | Trades
- Default sort: `finished_at DESC`
- Clickable column headers for sorting
- Each row clickable → opens `BacktestResultPanel` with `{ storeId: row.id }`

#### `BacktestResultPanel.vue` (modified)

- Accept `storeId` param in addition to existing runtime-mode
- New `storeId` mode:
  - Calls `app.GetStoredBacktestResult(storeId)` on mount
  - Deserializes `config_json`, `equity_curve`, `trades_json`, `ohlcv_data`
  - Renders identically to the runtime mode (K-line, equity curve, metrics, trades)
- Existing runtime mode (`nodeOutputs`) unchanged
- Mutually exclusive: if `storeId` provided, ignore `nodeOutputs`

#### `registry.ts`

Register `backtest-history` panel:
```ts
register('backtest-history', () => import('./BacktestHistoryPanel.vue'),
  { label: '回测历史', category: '量化分析', description: '历史回测记录浏览' })
```

### 4. OHLCV Data Capture

Backtest node runs after `data_loader` node in the workflow DAG. The `executionSaver` callback has access to ALL `NodeResult` entries. Strategy:

1. Backtest node outputs: `{ result, equity_curve, metrics, trades }`
2. Scan other NodeResults for node type `"data_loader"` with output port `"ohlcv"`
3. Extract OHLCV bars
4. Store alongside backtest result

If no data_loader OHLCV found (e.g. signals came from a different path), `ohlcv_data` stores `[]` and K-line section shows "数据不可用" message.

## Acceptance Criteria

- [ ] 工作流运行包含 backtest 节点时，回测结果自动持久化到 SQLite
- [ ] 终端模式下「回测历史」面板列出所有历史回测（日期/策略/标的/收益率/Sharpe），支持按列排序
- [ ] 点击历史行打开 `BacktestResultPanel`，显示完整的 K 线图 + 买卖点 + 净值曲线 + 指标网格 + 交易记录
- [ ] 数据在应用重启后仍可查看（SQLite 持久化）
- [ ] 旧有运行时 `BacktestResultPanel` 不受影响（兼容模式）

## Risks / Trade-offs

- **OHLCV 数据重复存储**：工作流运行时的 OHLCV 数据可能已在 `ohlcv_cache` 表中缓存，但为了简化持久化逻辑，直接存储到 `backtest_results.ohlcv_data` 列。桌面端 SQLite 单用户场景下存储成本可忽略。
- **大型回测性能**：equity_curve 和 ohlcv_data 可能较大（数万根 K 线）。`ListBacktestHistory` 只返回摘要（不含 JSON 列），只有在用户点开详情时才加载完整数据，避免列表加载慢。
- **executionSaver 耦合**：回测结果持久化依赖 `executionSaver` 回调。若未来执行引擎重构，需确保回调仍被调用。

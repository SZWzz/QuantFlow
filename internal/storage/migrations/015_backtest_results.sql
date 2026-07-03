-- 015_backtest_results: persist backtest run results for history and comparison

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

CREATE INDEX IF NOT EXISTS idx_bt_results_finished ON backtest_results(finished_at DESC);
CREATE INDEX IF NOT EXISTS idx_bt_results_symbol ON backtest_results(symbol);

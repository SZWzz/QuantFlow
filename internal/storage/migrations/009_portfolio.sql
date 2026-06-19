-- 009_portfolio: daily P&L snapshots and position history
CREATE TABLE IF NOT EXISTS daily_pnl (
    date TEXT NOT NULL,
    total_value REAL NOT NULL,
    cash REAL NOT NULL,
    market_value REAL NOT NULL,
    pnl REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    PRIMARY KEY (date)
);

CREATE TABLE IF NOT EXISTS position_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol TEXT NOT NULL,
    date TEXT NOT NULL,
    quantity REAL NOT NULL,
    avg_price REAL NOT NULL,
    market_price REAL NOT NULL,
    pnl REAL NOT NULL,
    pnl_pct REAL NOT NULL,
    UNIQUE(symbol, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_pnl_date ON daily_pnl(date);
CREATE INDEX IF NOT EXISTS idx_position_snapshots_date ON position_snapshots(date);

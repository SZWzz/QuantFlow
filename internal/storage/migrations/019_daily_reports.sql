-- 019_daily_reports: daily P&L report snapshots
CREATE TABLE IF NOT EXISTS daily_reports (
    date        TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    report_json TEXT NOT NULL,
    summary     TEXT NOT NULL,
    pnl         REAL NOT NULL,
    pnl_percent REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_daily_reports_date ON daily_reports(date DESC);

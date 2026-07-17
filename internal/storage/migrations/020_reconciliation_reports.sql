-- 020_reconciliation_reports: position reconciliation audit trail
CREATE TABLE IF NOT EXISTS reconciliation_reports (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at  INTEGER NOT NULL,
    report_json TEXT NOT NULL,
    match_count INTEGER NOT NULL,
    diff_count  INTEGER NOT NULL,
    dirt        TEXT NOT NULL, -- "clean" | "dirty"
    broker_name TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rec_reports_created ON reconciliation_reports(created_at DESC);

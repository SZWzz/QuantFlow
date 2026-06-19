-- 008_schedule: cron-based scheduled task definitions
CREATE TABLE IF NOT EXISTS schedule_tasks (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    timeout_sec INTEGER DEFAULT 1800,
    last_run_at TEXT,
    last_run_status TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

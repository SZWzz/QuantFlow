-- 013_executions: persist workflow execution history for replay and comparison

CREATE TABLE IF NOT EXISTS executions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT    NOT NULL UNIQUE,         -- e.g. "run-20260703-120000-abc123"
    workflow_id TEXT    NOT NULL,                -- original workflow id
    workflow_name TEXT  NOT NULL DEFAULT '',     -- human-readable name
    workflow_json TEXT  NOT NULL,                -- full workflow snapshot at execution time
    status      TEXT    NOT NULL DEFAULT 'running', -- running|completed|failed
    node_count  INTEGER NOT NULL DEFAULT 0,
    node_results TEXT,                           -- JSON array of NodeResult
    started_at  TEXT    NOT NULL,                -- ISO 8601
    finished_at TEXT,                            -- ISO 8601
    triggered_by TEXT   NOT NULL DEFAULT 'manual', -- manual|schedule|webhook
    error       TEXT,                            -- top-level error message (if failed)
    created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_executions_wf_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_created ON executions(created_at DESC);

-- 003_checkpoints: execution checkpoint table for breakpoint recovery.

CREATE TABLE IF NOT EXISTS execution_checkpoints (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id        TEXT NOT NULL,
    workflow_id   TEXT NOT NULL,
    node_id       TEXT NOT NULL,
    input_hash    TEXT NOT NULL,
    outputs_json  TEXT NOT NULL DEFAULT '{}',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_run_node
    ON execution_checkpoints(run_id, node_id);

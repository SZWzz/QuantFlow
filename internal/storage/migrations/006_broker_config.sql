-- 006_broker_config: store broker API credentials and settings
-- SECURITY: config_json stores broker API keys. In production, encrypt this
-- column at the application layer before writing (see internal/auth/encrypt.go
-- for the AES-256-GCM envelope). Until encryption is wired, set restrictive
-- file permissions on the SQLite database (chmod 600).
CREATE TABLE IF NOT EXISTS broker_config (
    broker_name TEXT PRIMARY KEY,
    enabled INTEGER DEFAULT 1,
    config_json TEXT NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

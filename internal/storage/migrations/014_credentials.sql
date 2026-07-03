-- 014_credentials: encrypted API key and credential storage

CREATE TABLE IF NOT EXISTS credentials (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,       -- e.g. "Binance Main", "Alpaca Paper"
    type        TEXT    NOT NULL DEFAULT 'api_key', -- api_key, oauth, basic_auth
    data        TEXT    NOT NULL,             -- JSON with encrypted values
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

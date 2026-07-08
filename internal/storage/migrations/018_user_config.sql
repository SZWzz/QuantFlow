-- 018_user_config: key-value config store for UI state (layouts, preferences)
CREATE TABLE IF NOT EXISTS user_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

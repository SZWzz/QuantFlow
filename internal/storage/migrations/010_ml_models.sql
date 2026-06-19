-- 010_ml_models: ML model storage, predictions, and evaluations
CREATE TABLE IF NOT EXISTS ml_models (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    model_type  TEXT NOT NULL,
    category    TEXT NOT NULL,
    hyperparams TEXT DEFAULT '{}',
    metrics     TEXT DEFAULT '{}',
    file_path   TEXT,
    file_bytes  BLOB,
    status      TEXT DEFAULT 'training',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_models_type ON ml_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ml_models_category ON ml_models(category);
CREATE INDEX IF NOT EXISTS idx_ml_models_status ON ml_models(status);

CREATE TABLE IF NOT EXISTS ml_predictions (
    id          TEXT PRIMARY KEY,
    model_id    TEXT NOT NULL REFERENCES ml_models(id) ON DELETE CASCADE,
    symbol      TEXT NOT NULL,
    date        TEXT NOT NULL,
    prediction  REAL NOT NULL,
    actual      REAL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_predictions_model ON ml_predictions(model_id);
CREATE INDEX IF NOT EXISTS idx_ml_predictions_symbol_date ON ml_predictions(symbol, date);

CREATE TABLE IF NOT EXISTS ml_evaluations (
    id          TEXT PRIMARY KEY,
    model_id    TEXT NOT NULL REFERENCES ml_models(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    value       REAL NOT NULL,
    period      TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_evaluations_model ON ml_evaluations(model_id);

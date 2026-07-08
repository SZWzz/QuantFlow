-- 017_data_archive: compressed archival storage for OHLCV/minute/backtest data

CREATE TABLE IF NOT EXISTS data_archive (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT    NOT NULL,
    symbol      TEXT    NOT NULL,
    interval    TEXT    NOT NULL DEFAULT '',
    date_from   TEXT    NOT NULL,
    date_to     TEXT    NOT NULL,
    row_count   INTEGER NOT NULL,
    data        BLOB    NOT NULL,
    archived_at TEXT    NOT NULL DEFAULT (datetime('now')),
    checksum    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_archive_source ON data_archive(source, symbol);
CREATE INDEX IF NOT EXISTS idx_archive_date   ON data_archive(date_from, date_to);

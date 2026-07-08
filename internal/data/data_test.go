package data

const ohlcvCacheDDL = `
CREATE TABLE IF NOT EXISTS ohlcv_cache (
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    ts INTEGER NOT NULL,
    open REAL, high REAL, low REAL, close REAL, volume REAL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (symbol, interval, ts)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS idx_ohlcv_cache_fetched ON ohlcv_cache(symbol, interval, fetched_at);
`

const minuteCacheDDL = `
CREATE TABLE IF NOT EXISTS minute_cache (
    symbol    TEXT    NOT NULL,
    date      TEXT    NOT NULL,
    tick_time TEXT    NOT NULL,
    price     REAL    NOT NULL,
    volume    REAL    NOT NULL,
    avg_price REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, date, tick_time)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS idx_minute_sym_date ON minute_cache(symbol, date);
`

const dataArchiveDDL = `
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
`

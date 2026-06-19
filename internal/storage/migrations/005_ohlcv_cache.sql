-- 005_ohlcv_cache: OHLCV data cache with TTL-based eviction

CREATE TABLE IF NOT EXISTS ohlcv_cache (
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    ts INTEGER NOT NULL,
    open REAL,
    high REAL,
    low REAL,
    close REAL,
    volume REAL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (symbol, interval, ts)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_ohlcv_cache_fetched ON ohlcv_cache(symbol, interval, fetched_at);

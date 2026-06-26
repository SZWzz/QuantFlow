-- 012_minute_cache: 分时数据持久化缓存（当日分钟级 tick）

CREATE TABLE IF NOT EXISTS minute_cache (
    symbol    TEXT    NOT NULL,
    date      TEXT    NOT NULL,   -- '2026-06-26'
    tick_time TEXT    NOT NULL,   -- '09:30'
    price     REAL    NOT NULL,
    volume    REAL    NOT NULL,
    avg_price REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, date, tick_time)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_minute_sym_date ON minute_cache(symbol, date);

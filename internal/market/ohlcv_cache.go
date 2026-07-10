package market

import (
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// OHLCVCache is a two-tier cache for OHLCV bar data:
//   - Hot tier: in-memory LRU (size 200, key "symbol:interval").
//   - Cold tier: SQLite ohlcv_cache table.
//
// All methods are safe for concurrent use.
type OHLCVCache struct {
	db  *sql.DB
	lru *lru.Cache[string, []OHLCVBar]
	mu  sync.RWMutex
}

// NewOHLCVCache initializes the cache and creates the backing SQLite
// table if it does not exist (defensive — migration 005 should have it).
func NewOHLCVCache(db *sql.DB) (*OHLCVCache, error) {
	lruCache, err := lru.New[string, []OHLCVBar](200)
	if err != nil {
		return nil, fmt.Errorf("ohlcv_cache: create lru: %w", err)
	}

	if _, err := db.Exec(ohlcvCacheDDL); err != nil {
		return nil, fmt.Errorf("ohlcv_cache: ensure table: %w", err)
	}

	return &OHLCVCache{
		db:  db,
		lru: lruCache,
	}, nil
}

// Get returns cached OHLCV bars for the given symbol/interval in the
// [start, end) time range (unix timestamps, seconds). Returns nil if no
// cached data is found. The returned slice is sorted by date ascending.
func (oc *OHLCVCache) Get(symbol, interval string, start, end int64) ([]OHLCVBar, error) {
	key := symbol + ":" + interval

	oc.mu.RLock()
	cached, ok := oc.lru.Get(key)
	oc.mu.RUnlock()

	if ok {
		return filterOHLCVByRange(cached, start, end), nil
	}

	oc.mu.Lock()
	defer oc.mu.Unlock()

	if cached, ok := oc.lru.Get(key); ok {
		return filterOHLCVByRange(cached, start, end), nil
	}

	bars, err := oc.loadFromDB(symbol, interval, start, end)
	if err != nil {
		return nil, fmt.Errorf("ohlcv_cache: load %s %s: %w", symbol, interval, err)
	}

	if len(bars) > 0 {
		oc.lru.Add(key, bars)
	}
	return filterOHLCVByRange(bars, start, end), nil
}

// Set stores OHLCV bars for a symbol/interval. Existing rows (same
// primary key) are silently skipped via INSERT OR IGNORE. The full bar
// list replaces the LRU entry for this key.
func (oc *OHLCVCache) Set(symbol, interval string, bars []OHLCVBar) error {
	if len(bars) == 0 {
		return nil
	}

	key := symbol + ":" + interval

	oc.mu.Lock()
	defer oc.mu.Unlock()

	oc.lru.Add(key, bars)

	return oc.saveToDB(symbol, interval, bars)
}

// GetIncremental returns cached bars whose timestamp is strictly later
// than since (unix seconds). Useful for refreshing after initial load.
func (oc *OHLCVCache) GetIncremental(symbol, interval string, since int64) ([]OHLCVBar, error) {
	return oc.Get(symbol, interval, since, time.Now().Unix())
}

// HasAtLeast checks whether the cache contains at least n bars for the
// given symbol/interval. Returns true and the bar count if so.
func (oc *OHLCVCache) HasAtLeast(symbol, interval string, n int) (bool, int, error) {
	key := symbol + ":" + interval

	oc.mu.RLock()
	cached, ok := oc.lru.Get(key)
	oc.mu.RUnlock()

	if ok && len(cached) >= n {
		return true, len(cached), nil
	}

	oc.mu.Lock()
	defer oc.mu.Unlock()

	if cached, ok := oc.lru.Get(key); ok && len(cached) >= n {
		return true, len(cached), nil
	}

	count, err := oc.countInDB(symbol, interval)
	if err != nil {
		return false, 0, err
	}
	return count >= n, count, nil
}

// Close releases resources. The underlying sql.DB is not closed.
func (oc *OHLCVCache) Close() error {
	oc.lru.Purge()
	return nil
}

// ── internal helpers ────────────────────────────────────────────────

const ohlcvCacheDDL = `
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
`

// dateToTs converts "2006-01-02" to a unix timestamp (seconds) at
// midnight in the Asia/Shanghai timezone. Returns 0 on parse failure.
func dateToTs(date string) int64 {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// tsToDate converts a unix timestamp to "2006-01-02" format in
// Asia/Shanghai timezone. Returns empty string for 0.
func tsToDate(ts int64) string {
	if ts == 0 {
		return ""
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return time.Unix(ts, 0).In(loc).Format("2006-01-02")
}

func (oc *OHLCVCache) loadFromDB(symbol, interval string, start, end int64) ([]OHLCVBar, error) {
	rows, err := oc.db.Query(
		"SELECT ts, open, high, low, close, volume FROM ohlcv_cache WHERE symbol=? AND interval=? AND ts >= ? AND ts < ? ORDER BY ts",
		symbol, interval, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bars []OHLCVBar
	for rows.Next() {
		var ts int64
		var bar OHLCVBar
		bar.Symbol = symbol
		if err := rows.Scan(&ts, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume); err != nil {
			return nil, err
		}
		bar.Date = tsToDate(ts)
		bars = append(bars, bar)
	}
	return bars, rows.Err()
}

func (oc *OHLCVCache) countInDB(symbol, interval string) (int, error) {
	var count int
	err := oc.db.QueryRow(
		"SELECT COUNT(*) FROM ohlcv_cache WHERE symbol=? AND interval=?",
		symbol, interval,
	).Scan(&count)
	return count, err
}

func (oc *OHLCVCache) saveToDB(symbol, interval string, bars []OHLCVBar) error {
	now := time.Now().Unix()
	tx, err := oc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT OR REPLACE INTO ohlcv_cache (symbol, interval, ts, open, high, low, close, volume, fetched_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, bar := range bars {
		ts := dateToTs(bar.Date)
		if ts == 0 {
			slog.Warn("ohlcv_cache: skipping bar with unparsable date", "symbol", symbol, "date", bar.Date)
			continue
		}
		if _, err := stmt.Exec(symbol, interval, ts, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume, now); err != nil {
			slog.Warn("ohlcv_cache: insert failed", "symbol", symbol, "date", bar.Date, "err", err)
		}
	}

	return tx.Commit()
}

// filterOHLCVByRange returns bars whose date falls within [start, end).
func filterOHLCVByRange(bars []OHLCVBar, start, end int64) []OHLCVBar {
	if len(bars) == 0 || (start == 0 && end == 0) {
		return bars
	}
	result := make([]OHLCVBar, 0, len(bars))
	for _, bar := range bars {
		ts := dateToTs(bar.Date)
		if ts >= start && (end == 0 || ts < end) {
			result = append(result, bar)
		}
	}
	return result
}

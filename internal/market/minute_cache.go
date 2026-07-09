package market

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// MinuteCache is a two-tier cache for intraday minute-line data:
//   - Hot tier: in-memory LRU (size 500, key "symbol:date").
//   - Cold tier: SQLite minute_cache table.
//
// All methods are safe for concurrent use.
type MinuteCache struct {
	db  *sql.DB
	lru *lru.Cache[string, []MinuteTick]
	mu  sync.RWMutex
}

// NewMinuteCache initializes the cache and creates the backing
// SQLite table if it does not exist.
func NewMinuteCache(db *sql.DB) (*MinuteCache, error) {
	lruCache, err := lru.New[string, []MinuteTick](500)
	if err != nil {
		return nil, fmt.Errorf("minute_cache: create lru: %w", err)
	}

	if _, err := db.Exec(minuteCacheDDL); err != nil {
		return nil, fmt.Errorf("minute_cache: ensure table: %w", err)
	}

	return &MinuteCache{
		db:  db,
		lru: lruCache,
	}, nil
}

// GetIncremental returns minute ticks for the given symbol on today's
// date. If since is 0, returns all ticks for today in chronological order.
// If since > 0, returns only ticks whose time is strictly after the given
// Unix timestamp. The returned slice is nil-safe (never nil).
func (mc *MinuteCache) GetIncremental(symbol string, since int64) ([]MinuteTick, error) {
	var date string
	if since == 0 {
		date = time.Now().Format("2006-01-02")
	} else {
		loc, err := time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8 * 3600)
		}
		date = time.Unix(since, 0).In(loc).Format("2006-01-02")
	}
	key := symbol + ":" + date

	// 1. Check LRU
	mc.mu.RLock()
	if cached, ok := mc.lru.Get(key); ok {
		mc.mu.RUnlock()
		return filterSince(cached, since), nil
	}
	mc.mu.RUnlock()

	// 2. Try SQLite
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Double-check LRU (may have been filled by concurrent call).
	if cached, ok := mc.lru.Get(key); ok {
		return filterSince(cached, since), nil
	}

	ticks, err := mc.loadFromDB(symbol, date)
	if err != nil {
		return nil, fmt.Errorf("minute_cache: load %s %s: %w", symbol, date, err)
	}

	if ticks != nil {
		mc.lru.Add(key, ticks)
	}
	return filterSince(ticks, since), nil
}

// SaveTicks persists a batch of minute ticks for a symbol on a date.
// Existing rows (same primary key) are silently skipped via INSERT OR IGNORE.
func (mc *MinuteCache) SaveTicks(symbol, date string, ticks []MinuteTick) error {
	if len(ticks) == 0 {
		return nil
	}

	key := symbol + ":" + date

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Upsert: merge new ticks with existing LRU entry.
	existing := make(map[string]MinuteTick)
	if cached, ok := mc.lru.Get(key); ok {
		for _, t := range cached {
			existing[t.Time] = t
		}
	}
	for _, t := range ticks {
		existing[t.Time] = t
	}

	merged := make([]MinuteTick, 0, len(existing))
	for _, t := range existing {
		merged = append(merged, t)
	}
	// Sort ascending by time.
	sortMinuteTicks(merged)
	mc.lru.Add(key, merged)

	// Write to SQLite.
	return mc.saveToDB(symbol, date, ticks)
}

// GetRecentTicks returns minute ticks from the most recent trading day found in
// cache. Looks back up to lookbackDays (max 10). Returns nil if no cached data.
func (mc *MinuteCache) GetRecentTicks(symbol string, lookbackDays int) ([]MinuteTick, string, error) {
	if lookbackDays <= 0 {
		lookbackDays = 5
	}
	if lookbackDays > 10 {
		lookbackDays = 10
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for i := 0; i < lookbackDays; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		key := symbol + ":" + date

		// Check LRU first
		if cached, ok := mc.lru.Get(key); ok && len(cached) > 0 {
			return cached, date, nil
		}
		// Check SQLite
		ticks, err := mc.loadFromDB(symbol, date)
		if err != nil {
			continue
		}
		if len(ticks) > 0 {
			mc.mu.RUnlock()
			mc.mu.Lock()
			mc.lru.Add(key, ticks)
			mc.mu.Unlock()
			mc.mu.RLock()
			return ticks, date, nil
		}
	}
	return nil, "", nil
}

// Truncate clears all minute cache entries (both LRU and SQLite).
// Useful for cache invalidation after data source fixes.
func (mc *MinuteCache) Truncate() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.lru.Purge()
	_, err := mc.db.Exec("DELETE FROM minute_cache")
	return err
}

// Close releases resources. The underlying sql.DB is not closed.
func (mc *MinuteCache) Close() error {
	mc.lru.Purge()
	return nil
}

// ── internal helpers ────────────────────────────────────────────────

const minuteCacheDDL = `
CREATE TABLE IF NOT EXISTS minute_cache (
    symbol    TEXT    NOT NULL,
    date      TEXT    NOT NULL,
    tick_time TEXT    NOT NULL,
    price     REAL    NOT NULL,
    volume    REAL    NOT NULL,
    avg_price REAL    NOT NULL DEFAULT 0,
    amount    REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (symbol, date, tick_time)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_minute_sym_date ON minute_cache(symbol, date);
`

func (mc *MinuteCache) loadFromDB(symbol, date string) ([]MinuteTick, error) {
	rows, err := mc.db.Query(
		"SELECT tick_time, price, volume, avg_price, amount FROM minute_cache WHERE symbol=? AND date=? ORDER BY tick_time",
		symbol, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ticks []MinuteTick
	for rows.Next() {
		var t MinuteTick
		if err := rows.Scan(&t.Time, &t.Price, &t.Volume, &t.AvgPrice, &t.Amount); err != nil {
			return nil, err
		}
		ticks = append(ticks, t)
	}
	return ticks, rows.Err()
}

func (mc *MinuteCache) saveToDB(symbol, date string, ticks []MinuteTick) error {
	tx, err := mc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT OR IGNORE INTO minute_cache (symbol, date, tick_time, price, volume, avg_price, amount) VALUES (?, ?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range ticks {
		if _, err := stmt.Exec(symbol, date, t.Time, t.Price, t.Volume, t.AvgPrice, t.Amount); err != nil {
			slog.Warn("minute_cache: insert failed", "symbol", symbol, "time", t.Time, "err", err)
			// Don't fail the whole batch.
		}
	}

	return tx.Commit()
}

// filterSince returns ticks whose time string is later than the reference.
// since=0 means return all ticks.
func filterSince(ticks []MinuteTick, since int64) []MinuteTick {
	if since == 0 || len(ticks) == 0 {
		return ticks
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8 * 3600)
	}
	ref := time.Unix(since, 0).In(loc).Format("15:04")
	for i := len(ticks) - 1; i >= 0; i-- {
		if strings.Compare(ticks[i].Time, ref) <= 0 {
			return ticks[i+1:]
		}
	}
	return ticks
}

// sortMinuteTicks sorts a slice of MinuteTick by Time ascending in place.
func sortMinuteTicks(ticks []MinuteTick) {
	// Insertion sort — small slices (≤240 ticks per day).
	for i := 1; i < len(ticks); i++ {
		key := ticks[i]
		j := i - 1
		for j >= 0 && ticks[j].Time > key.Time {
			ticks[j+1] = ticks[j]
			j--
		}
		ticks[j+1] = key
	}
}

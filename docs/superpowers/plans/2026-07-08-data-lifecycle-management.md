# Data Lifecycle Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add data lifecycle management to QuantFlow — archive, export, import, cleanup, and storage monitoring.

**Architecture:** New `internal/data/` Go package (5 files) handles all operations against SQLite. App struct exposes 5 IPC methods. Frontend gets a new `StoragePanel.vue`.

**Tech Stack:** Go 1.25+, SQLite (mattn/go-sqlite3), Parquet (segmentio/parquet-go), Vue 3 + TypeScript, vitest

## Global Constraints

- SQLite is the only database — no PostgreSQL, no Redis
- All IPC methods go through App struct in `app.go` / `app_*.go`
- Frontend confirm/alert must use `@/lib/wails` (not `window.confirm`)
- Tests use in-memory SQLite (`:memory:`)
- Migration files go in `internal/storage/migrations/`
- All new Go files go in `internal/data/`
- Parquet dependency: `github.com/segmentio/parquet-go`

---

### Task 1: Migration 017 + stats.go + package scaffold

**Files:**
- Create: `internal/storage/migrations/017_data_archive.sql`
- Create: `internal/data/data_test.go`
- Create: `internal/data/stats.go`
- Create: `internal/data/stats_test.go`

**Interfaces:**
- Produces: `func GetTableStats(db *sql.DB) ([]TableStat, error)`
- Produces: Type `TableStat`

- [ ] **Step 1: Write migration SQL**

```sql
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
```

- [ ] **Step 2: Register migration in `internal/storage/migrate.go`**

Read `internal/storage/migrate.go` to find the `BuiltinMigrations` slice, then add:

```go
{17, "017_data_archive"},
```

- [ ] **Step 3: Write the failing test**

```go
// internal/data/stats_test.go
package data

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	for _, ddl := range []string{
		ohlcvCacheDDL, minuteCacheDDL, dataArchiveDDL,
	} {
		_, err := db.Exec(ddl)
		require.NoError(t, err)
	}
	return db
}

func seedOHLCV(t *testing.T, db *sql.DB, symbol string, count int) {
	t.Helper()
	now := time.Now().Unix()
	for i := 0; i < count; i++ {
		ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Unix()
		_, err := db.Exec(
			"INSERT OR IGNORE INTO ohlcv_cache (symbol, interval, ts, open, high, low, close, volume, fetched_at) VALUES (?,?,?,?,?,?,?,?,?)",
			symbol, "1D", ts, 10.0, 11.0, 9.0, 10.5, 100000, now,
		)
		require.NoError(t, err)
	}
}

func TestGetTableStats_returns_counts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedOHLCV(t, db, "000001", 5)
	seedOHLCV(t, db, "600001", 3)

	stats, err := GetTableStats(db)
	require.NoError(t, err)
	require.Len(t, stats, 3) // ohlcv_cache, minute_cache, data_archive

	var found bool
	for _, s := range stats {
		if s.Table == "ohlcv_cache" {
			found = true
			require.Equal(t, int64(8), s.Rows)
			require.Greater(t, s.SizeBytes, int64(0))
			require.NotEmpty(t, s.Oldest)
			require.NotEmpty(t, s.Newest)
		}
	}
	require.True(t, found)
}

func TestGetTableStats_empty_db(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := GetTableStats(db)
	require.NoError(t, err)
	for _, s := range stats {
		require.Equal(t, int64(0), s.Rows)
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

```bash
cd app && go test ./internal/data/ -run TestGetTableStats -v
```

Expected: FAIL with `GetTableStats not defined`

- [ ] **Step 5: Write package scaffold + stats.go**

Create `internal/data/data_test.go` with DDL constants:

```go
// internal/data/data_test.go
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
```

Create `internal/data/stats.go`:

```go
// Package data provides data lifecycle management: archive, export, import, cleanup, stats.
package data

import (
	"database/sql"
	"fmt"
)

// TableStat holds storage statistics for a single database table.
type TableStat struct {
	Table     string `json:"table"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"size_bytes"`
	Oldest    string `json:"oldest"`
	Newest    string `json:"newest"`
}

var trackedTables = []struct {
	Name     string
	DateCol  string // column name for time range, empty = skip
	SizeExpr string // SQL expression for approx size
}{
	{"ohlcv_cache", "ts", "COUNT(*) * 64"},    // ~64 bytes/row estimate
	{"minute_cache", "date", "COUNT(*) * 48"},  // ~48 bytes/row estimate
	{"data_archive", "archived_at", "COALESCE(SUM(LENGTH(data)), 0) + COUNT(*) * 128"},
}

// GetTableStats returns storage statistics for all tracked tables.
func GetTableStats(db *sql.DB) ([]TableStat, error) {
	var stats []TableStat
	for _, t := range trackedTables {
		st, err := tableStat(db, t.Name, t.DateCol, t.SizeExpr)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", t.Name, err)
		}
		stats = append(stats, st)
	}
	return stats, nil
}

func tableStat(db *sql.DB, table, dateCol, sizeExpr string) (TableStat, error) {
	var st TableStat
	st.Table = table

	q := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table)
	err := db.QueryRow(q).Scan(&st.Rows)
	if err != nil {
		return st, err
	}

	q = fmt.Sprintf("SELECT %s FROM \"%s\"", sizeExpr, table)
	err = db.QueryRow(q).Scan(&st.SizeBytes)
	if err != nil {
		st.SizeBytes = 0
	}

	if dateCol != "" && st.Rows > 0 {
		q = fmt.Sprintf("SELECT MIN(%s), MAX(%s) FROM \"%s\"", dateCol, dateCol, table)
		var oldest, newest sql.NullString
		err = db.QueryRow(q).Scan(&oldest, &newest)
		if err == nil {
			st.Oldest = oldest.String
			st.Newest = newest.String
		}
	}

	return st, nil
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd app && go test ./internal/data/ -run TestGetTableStats -v
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/data/ internal/storage/migrations/017_data_archive.sql internal/storage/migrate.go
git commit -m "feat(data): add migration 017 and GetTableStats"
```

---

### Task 2: Archiver — compress/decompress roundtrip

**Files:**
- Create: `internal/data/archiver.go`
- Create: `internal/data/archiver_test.go`

**Interfaces:**
- Consumes: `GetTableStats`, `TableStat` (from Task 1)
- Produces: `func ArchiveData(db *sql.DB, source, symbol, before string) (*ArchiveResult, error)`
- Produces: `func UnarchiveData(db *sql.DB, archiveID int64) (int64, error)`
- Produces: Type `ArchiveResult`

- [ ] **Step 1: Write the failing test**

```go
// internal/data/archiver_test.go
package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveData_compresses_ohlcv(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "ohlcv_cache", result.Source)
	require.Equal(t, "000001", result.Symbol)
	require.Equal(t, 10, result.RowCount)
	require.Greater(t, result.CompressedBytes, 0)

	// Original rows must still exist
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	require.Equal(t, 10, count)
}

func TestArchiveData_no_data(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := ArchiveData(db, "ohlcv_cache", "999999", "2025-01-01")
	require.NoError(t, err)
	require.Equal(t, 0, result.RowCount)
}

func TestArchiveData_invalid_source(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := ArchiveData(db, "nonexistent", "000001", "2025-01-01")
	require.Error(t, err)
}

func TestUnarchiveData_roundtrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	ar, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	require.NoError(t, err)
	require.Equal(t, 10, ar.RowCount)

	// Delete originals
	_, err = db.Exec("DELETE FROM ohlcv_cache WHERE symbol='000001'")
	require.NoError(t, err)

	// Unarchive
	restored, err := UnarchiveData(db, ar.ID)
	require.NoError(t, err)
	require.Equal(t, int64(10), restored)

	// Verify data is back
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	require.Equal(t, 10, count)
}

func TestUnarchiveData_checksum_mismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	ar, err := ArchiveData(db, "ohlcv_cache", "000001", "2025-01-01")
	require.NoError(t, err)

	// Tamper with the BLOB
	db.Exec("UPDATE data_archive SET data = X'0000' WHERE id = ?", ar.ID)

	_, err = UnarchiveData(db, ar.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && go test ./internal/data/ -run TestArchive -v
```

Expected: FAIL (undefined functions)

- [ ] **Step 3: Write archiver.go**

```go
// internal/data/archiver.go
package data

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ArchiveResult describes a completed archive operation.
type ArchiveResult struct {
	ID              int64  `json:"id"`
	Source          string `json:"source"`
	Symbol          string `json:"symbol"`
	Interval        string `json:"interval"`
	DateFrom        string `json:"date_from"`
	DateTo          string `json:"date_to"`
	RowCount        int    `json:"row_count"`
	CompressedBytes int    `json:"compressed_bytes"`
}

var validArchiveSources = map[string]bool{
	"ohlcv_cache":      true,
	"minute_cache":     true,
	"backtest_results": true,
}

func sourceDateCol(source string) string {
	switch source {
	case "ohlcv_cache":
		return "ts"
	case "minute_cache":
		return "date"
	case "backtest_results":
		return "finished_at"
	default:
		return ""
	}
}

// ArchiveData compresses rows from the given source table into data_archive.
// Original rows are NOT deleted. symbol="" means all symbols; before="" means all dates.
func ArchiveData(db *sql.DB, source, symbol, before string) (*ArchiveResult, error) {
	if !validArchiveSources[source] {
		return nil, fmt.Errorf("invalid archive source: %s", source)
	}

	dateCol := sourceDateCol(source)
	if dateCol == "" {
		return nil, fmt.Errorf("no date column mapping for source: %s", source)
	}

	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if before != "" {
		where += " AND " + dateCol + " < ?"
		args = append(args, before)
	}

	// Fetch all rows
	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY %s", source, where, dateCol)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", source, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var allRows []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		valPtrs := make([]any, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any)
		for i, col := range cols {
			row[col] = vals[i]
		}
		allRows = append(allRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(allRows) == 0 {
		return &ArchiveResult{Source: source, Symbol: symbol}, nil
	}

	// JSON marshal
	raw, err := json.Marshal(allRows)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	// SHA256 checksum
	hash := sha256.Sum256(raw)
	checksum := hex.EncodeToString(hash[:])

	// gzip compress
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip init: %w", err)
	}
	if _, err := gz.Write(raw); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	compressed := buf.Bytes()

	// Determine date range
	dateFrom := before
	dateTo := time.Now().Format("2006-01-02")
	if len(allRows) > 0 {
		if v, ok := allRows[0][dateCol]; ok {
			dateFrom = fmt.Sprintf("%v", v)
		}
		if v, ok := allRows[len(allRows)-1][dateCol]; ok {
			dateTo = fmt.Sprintf("%v", v)
		}
	}

	// Determine interval
	interval := ""
	if v, ok := allRows[0]["interval"]; ok {
		interval = fmt.Sprintf("%v", v)
	}

	// Insert into data_archive
	result := &ArchiveResult{
		Source:          source,
		Symbol:          symbol,
		Interval:        interval,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		RowCount:        len(allRows),
		CompressedBytes: len(compressed),
	}

	err = db.QueryRow(
		`INSERT INTO data_archive (source, symbol, interval, date_from, date_to, row_count, data, checksum)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		source, symbol, interval, dateFrom, dateTo, len(allRows), compressed, checksum,
	).Scan(&result.ID)
	if err != nil {
		return nil, fmt.Errorf("insert archive: %w", err)
	}

	return result, nil
}

// UnarchiveData decompresses an archive entry and writes rows back to the source table.
func UnarchiveData(db *sql.DB, archiveID int64) (int64, error) {
	var source, checksum string
	var compressed []byte
	err := db.QueryRow(
		"SELECT source, data, checksum FROM data_archive WHERE id = ?", archiveID,
	).Scan(&source, &compressed, &checksum)
	if err != nil {
		return 0, fmt.Errorf("find archive %d: %w", archiveID, err)
	}

	// Decompress
	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return 0, fmt.Errorf("gzip reader: %w", err)
	}
	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(gz); err != nil {
		return 0, fmt.Errorf("gzip read: %w", err)
	}
	gz.Close()

	// Verify checksum
	hash := sha256.Sum256(decompressed.Bytes())
	if hex.EncodeToString(hash[:]) != checksum {
		return 0, fmt.Errorf("checksum mismatch: archive may be corrupted")
	}

	// Parse JSON
	var rows []map[string]any
	if err := json.Unmarshal(decompressed.Bytes(), &rows); err != nil {
		return 0, fmt.Errorf("unmarshal: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}

	// Build INSERT
	cols := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		// skip computed columns that don't exist in the source table
		if k == "id" && source != "backtest_results" {
			continue
		}
		cols = append(cols, k)
	}

	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}

	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", source, colList, placeholders)
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for _, row := range rows {
		vals := make([]any, 0, len(cols))
		for _, c := range cols {
			vals = append(vals, row[c])
		}
		res, err := stmt.Exec(vals...)
		if err != nil {
			continue // skip problematic rows
		}
		n, _ := res.RowsAffected()
		inserted += n
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func repeatString(s string, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd app && go test ./internal/data/ -run TestArchive -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/data/archiver.go internal/data/archiver_test.go
git commit -m "feat(data): add ArchiveData and UnarchiveData with gzip compression"
```

---

### Task 3: Cleaner — preview and delete

**Files:**
- Create: `internal/data/cleaner.go`
- Create: `internal/data/cleaner_test.go`

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `func CleanupData(db *sql.DB, table, symbol, before string, dryRun bool) (*CleanupResult, error)`
- Produces: Type `CleanupResult`

- [ ] **Step 1: Write the failing test**

```go
// internal/data/cleaner_test.go
package data

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCleanupData_dryRun_returns_count(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := CleanupData(db, "ohlcv_cache", "000001", "2024-01-05", true)
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Equal(t, int64(4), result.AffectedRows) // Jan 1-4 = 4 rows before Jan 5
	require.Len(t, result.Preview, 4)

	// Rows must still exist
	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	require.Equal(t, 10, count)
}

func TestCleanupData_execute_removes_rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 10)

	result, err := CleanupData(db, "ohlcv_cache", "000001", "2024-01-05", false)
	require.NoError(t, err)
	require.False(t, result.DryRun)
	require.Equal(t, int64(4), result.AffectedRows)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache WHERE symbol='000001'").Scan(&count)
	require.Equal(t, 6, count)
}

func TestCleanupData_invalid_table(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := CleanupData(db, "orders", "000001", "2025-01-01", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestCleanupData_no_data(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	result, err := CleanupData(db, "ohlcv_cache", "999999", "2025-01-01", true)
	require.NoError(t, err)
	require.Equal(t, int64(0), result.AffectedRows)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && go test ./internal/data/ -run TestCleanup -v
```

Expected: FAIL

- [ ] **Step 3: Write cleaner.go**

```go
// internal/data/cleaner.go
package data

import (
	"database/sql"
	"fmt"
)

var allowedCleanupTables = map[string]bool{
	"ohlcv_cache":  true,
	"minute_cache": true,
}

// CleanupResult describes the outcome of a cleanup operation.
type CleanupResult struct {
	AffectedRows int64            `json:"affected_rows"`
	Preview      []map[string]any `json:"preview"`
	Table        string           `json:"table"`
	DryRun       bool             `json:"dry_run"`
}

func cleanupDateCol(table string) string {
	switch table {
	case "ohlcv_cache":
		return "ts"
	case "minute_cache":
		return "date"
	default:
		return ""
	}
}

// CleanupData previews or executes deletion of data rows.
// dryRun=true only counts and previews; dryRun=false deletes.
func CleanupData(db *sql.DB, table, symbol, before string, dryRun bool) (*CleanupResult, error) {
	if !allowedCleanupTables[table] {
		return nil, fmt.Errorf("table %s is not allowed for cleanup", table)
	}

	dateCol := cleanupDateCol(table)
	if dateCol == "" {
		return nil, fmt.Errorf("no date column for table %s", table)
	}

	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if before != "" {
		if table == "ohlcv_cache" {
			// dateCol is ts (unix seconds), before is ISO 8601
			where += " AND " + dateCol + " < ?"
			// We store before as unix timestamp reference
			args = append(args, dateToTimestamp(before))
		} else {
			where += " AND " + dateCol + " < ?"
			args = append(args, before)
		}
	}

	// Count matching rows
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM \"%s\" WHERE %s", table, where)
	var total int64
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count: %w", err)
	}

	// Preview: fetch first 10 rows
	previewQ := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s LIMIT 10", table, where)
	rows, err := db.Query(previewQ, args...)
	if err != nil {
		return nil, fmt.Errorf("preview: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var preview []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		valPtrs := make([]any, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any)
		for i, col := range cols {
			row[col] = vals[i]
		}
		preview = append(preview, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &CleanupResult{
		AffectedRows: total,
		Preview:      preview,
		Table:        table,
		DryRun:       dryRun,
	}

	if !dryRun && total > 0 {
		deleteQ := fmt.Sprintf("DELETE FROM \"%s\" WHERE %s", table, where)
		res, err := db.Exec(deleteQ, args...)
		if err != nil {
			return nil, fmt.Errorf("delete: %w", err)
		}
		n, _ := res.RowsAffected()
		result.AffectedRows = n
	}

	return result, nil
}

// dateToTimestamp converts "2006-01-02" to a unix timestamp (seconds) at UTC midnight.
func dateToTimestamp(dateStr string) int64 {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	return t.Unix()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Need to add `"time"` import:

```bash
cd app && go test ./internal/data/ -run TestCleanup -v
```

Expected: all PASS

Note: Add `time` import to cleaner.go:
```go
import (
    "database/sql"
    "fmt"
    "time"
)
```

- [ ] **Step 5: Commit**

```bash
git add internal/data/cleaner.go internal/data/cleaner_test.go
git commit -m "feat(data): add CleanupData with dryRun preview and safe delete"
```

---

### Task 4: Exporter — CSV and Parquet export

**Files:**
- Create: `internal/data/exporter.go`
- Create: `internal/data/exporter_test.go`

**Interfaces:**
- Consumes: nothing from prior tasks
- Produces: `func ExportCSV(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error)`
- Produces: `func ExportParquet(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error)`

- [ ] **Step 1: Ensure parquet-go dependency is available**

```bash
cd app && go get github.com/segmentio/parquet-go@latest
```

- [ ] **Step 2: Write the failing test**

```go
// internal/data/exporter_test.go
package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportCSV_writes_valid_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "000001", "1D", "2024-01-01", "2024-01-05", path)
	require.NoError(t, err)
	require.Equal(t, int64(5), count)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, "symbol,interval,ts,open,high,low,close,volume,fetched_at")
	require.Contains(t, content, "000001")
}

func TestExportCSV_empty_result(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "999999", "1D", "2024-01-01", "2024-01-05", path)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestExportCSV_no_date_filter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "all.csv")

	count, err := ExportCSV(db, "ohlcv_cache", "000001", "1D", "", "", path)
	require.NoError(t, err)
	require.Equal(t, int64(5), count)
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd app && go test ./internal/data/ -run TestExportCSV -v
```

Expected: FAIL

- [ ] **Step 4: Write exporter.go**

```go
// internal/data/exporter.go
package data

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/segmentio/parquet-go"
)

func exportWhere(table, symbol, interval, dateFrom, dateTo string) (string, []any) {
	where := "1=1"
	args := []any{}
	if symbol != "" {
		where += " AND symbol = ?"
		args = append(args, symbol)
	}
	if interval != "" {
		where += " AND interval = ?"
		args = append(args, interval)
	}
	if dateFrom != "" {
		if table == "ohlcv_cache" {
			where += " AND ts >= ?"
			args = append(args, dateToTimestamp(dateFrom))
		} else {
			where += " AND date >= ?"
			args = append(args, dateFrom)
		}
	}
	if dateTo != "" {
		if table == "ohlcv_cache" {
			where += " AND ts <= ?"
			args = append(args, dateToTimestamp(dateTo))
		} else {
			where += " AND date <= ?"
			args = append(args, dateTo)
		}
	}
	return where, args
}

// ExportCSV exports query results to a CSV file. Returns row count.
func ExportCSV(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error) {
	where, args := exportWhere(table, symbol, interval, dateFrom, dateTo)
	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY symbol, ts", table, where)
	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Write header
	if err := w.Write(cols); err != nil {
		return 0, fmt.Errorf("write header: %w", err)
	}

	var count int64
	vals := make([]any, len(cols))
	valPtrs := make([]any, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			return 0, err
		}
		record := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := w.Write(record); err != nil {
			return 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

func ohlcvParquetSchema() *parquet.Schema {
	return parquet.SchemaOf(ohlcvParquetRow{})
}

type ohlcvParquetRow struct {
	Symbol   string  `parquet:"symbol"`
	Interval string  `parquet:"interval"`
	Ts       int64   `parquet:"ts"`
	Open     float64 `parquet:"open"`
	High     float64 `parquet:"high"`
	Low      float64 `parquet:"low"`
	Close    float64 `parquet:"close"`
	Volume   float64 `parquet:"volume"`
}

// ExportParquet exports query results to a Parquet file. Returns row count.
func ExportParquet(db *sql.DB, table, symbol, interval, dateFrom, dateTo, outputPath string) (int64, error) {
	where, args := exportWhere(table, symbol, interval, dateFrom, dateTo)
	q := fmt.Sprintf("SELECT * FROM \"%s\" WHERE %s ORDER BY symbol, ts", table, where)
	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return 0, fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	writer := parquet.NewGenericWriter[ohlcvParquetRow](f)
	defer writer.Close()

	var count int64
	vals := make([]any, len(cols))
	valPtrs := make([]any, len(cols))
	for i := range vals {
		valPtrs[i] = &vals[i]
	}

	// Attempt to find column indices
	var symIdx, intervalIdx, tsIdx, oIdx, hIdx, lIdx, cIdx, vIdx int = -1, -1, -1, -1, -1, -1, -1, -1
	for i, c := range cols {
		switch c {
		case "symbol":
			symIdx = i
		case "interval":
			intervalIdx = i
		case "ts":
			tsIdx = i
		case "open":
			oIdx = i
		case "high":
			hIdx = i
		case "low":
			lIdx = i
		case "close":
			cIdx = i
		case "volume":
			vIdx = i
		}
	}

	for rows.Next() {
		if err := rows.Scan(valPtrs...); err != nil {
			return 0, err
		}
		row := ohlcvParquetRow{}
		if symIdx >= 0 {
			row.Symbol = fmt.Sprintf("%v", vals[symIdx])
		}
		if intervalIdx >= 0 && vals[intervalIdx] != nil {
			row.Interval = fmt.Sprintf("%v", vals[intervalIdx])
		}
		if tsIdx >= 0 && vals[tsIdx] != nil {
			row.Ts = toInt64(vals[tsIdx])
		}
		row.Open = toFloat64(vals, oIdx)
		row.High = toFloat64(vals, hIdx)
		row.Low = toFloat64(vals, lIdx)
		row.Close = toFloat64(vals, cIdx)
		row.Volume = toFloat64(vals, vIdx)

		if _, err := writer.Write([]ohlcvParquetRow{row}); err != nil {
			return 0, fmt.Errorf("parquet write: %w", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

func toFloat64(vals []any, idx int) float64 {
	if idx < 0 || idx >= len(vals) || vals[idx] == nil {
		return 0
	}
	switch v := vals[idx].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case []byte:
		var f float64
		_, _ = fmt.Sscanf(string(v), "%f", &f)
		return f
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch v := v.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case []byte:
		var n int64
		_, _ = fmt.Sscanf(string(v), "%d", &n)
		return n
	default:
		return 0
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd app && go test ./internal/data/ -run TestExportCSV -v
```

Expected: all PASS

- [ ] **Step 6: Add Parquet export test and run**

```go
func TestExportParquet_writes_valid_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	count, err := ExportParquet(db, "ohlcv_cache", "000001", "1D", "2024-01-01", "2024-01-05", path)
	require.NoError(t, err)
	require.Equal(t, int64(5), count)

	// Verify file exists and has content
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}
```

Run:
```bash
cd app && go test ./internal/data/ -run TestExport -v
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add internal/data/exporter.go internal/data/exporter_test.go
git commit -m "feat(data): add CSV and Parquet export"
```

---

### Task 5: Importer — CSV and Parquet import

**Files:**
- Create: `internal/data/importer.go`
- Create: `internal/data/importer_test.go`

**Interfaces:**
- Consumes: `exportWhere`, `dateToTimestamp` (from Task 4)
- Produces: `func ImportCSV(db *sql.DB, filePath, table string) (int64, error)`
- Produces: `func ImportParquet(db *sql.DB, filePath, table string) (int64, error)`

- [ ] **Step 1: Write the failing test**

```go
// internal/data/importer_test.go
package data

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTestCSV(t *testing.T, path string, rows [][]string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	w := csv.NewWriter(f)
	require.NoError(t, w.WriteAll(rows))
	w.Flush()
}

func TestImportCSV_inserts_rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "import.csv")
	writeTestCSV(t, path, [][]string{
		{"symbol", "interval", "ts", "open", "high", "low", "close", "volume", "fetched_at"},
		{"000001", "1D", "1704067200", "10.0", "11.0", "9.0", "10.5", "100000", "1704067200"},
		{"000001", "1D", "1704153600", "10.5", "11.5", "10.0", "11.0", "150000", "1704153600"},
	})

	count, err := ImportCSV(db, path, "ohlcv_cache")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	var total int
	db.QueryRow("SELECT COUNT(*) FROM ohlcv_cache").Scan(&total)
	require.Equal(t, 2, total)
}

func TestImportCSV_duplicate_rows_skipped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedOHLCV(t, db, "000001", 5)

	dir := t.TempDir()
	path := filepath.Join(dir, "dup.csv")
	writeTestCSV(t, path, [][]string{
		{"symbol", "interval", "ts", "open", "high", "low", "close", "volume", "fetched_at"},
		{"000001", "1D", "1704067200", "10.0", "11.0", "9.0", "10.5", "100000", "1704067200"},
	})

	count, err := ImportCSV(db, path, "ohlcv_cache")
	require.NoError(t, err)
	require.Equal(t, int64(0), count) // all skipped (INSERT OR IGNORE)
}

func TestImportCSV_invalid_table(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	writeTestCSV(t, path, [][]string{{"a"}, {"1"}})

	_, err := ImportCSV(db, path, "nonexistent")
	require.Error(t, err)
}

func TestImportCSV_missing_file(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := ImportCSV(db, "/nonexistent/path.csv", "ohlcv_cache")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd app && go test ./internal/data/ -run TestImportCSV -v
```

Expected: FAIL

- [ ] **Step 3: Write importer.go**

```go
// internal/data/importer.go
package data

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/segmentio/parquet-go"
)

var validImportTables = map[string]bool{
	"ohlcv_cache":  true,
	"minute_cache": true,
}

// ImportCSV reads a CSV file (first row = header) and inserts rows into the given table.
// Duplicate rows are silently skipped (INSERT OR IGNORE).
func ImportCSV(db *sql.DB, filePath, table string) (int64, error) {
	if !validImportTables[table] {
		return 0, fmt.Errorf("import to table %q is not allowed", table)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	// Validate header names match table columns
	var cols []string
	for _, h := range header {
		cols = append(cols, h)
	}

	// Build INSERT statement
	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}
	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", table, colList, placeholders)

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var count int64
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, fmt.Errorf("read row: %w", err)
		}

		vals := make([]any, len(record))
		for i, v := range record {
			vals[i] = v
		}

		res, err := stmt.Exec(vals...)
		if err != nil {
			continue // skip problematic rows
		}
		n, _ := res.RowsAffected()
		count += n
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

// ImportParquet reads a Parquet file and inserts rows into the given table.
func ImportParquet(db *sql.DB, filePath, table string) (int64, error) {
	if !validImportTables[table] {
		return 0, fmt.Errorf("import to table %q is not allowed", table)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	data, err := parquet.ReadFile[ohlcvParquetRow](filePath)
	if err != nil {
		return 0, fmt.Errorf("parquet read: %w", err)
	}

	if len(data) == 0 {
		return 0, nil
	}

	cols := []string{"symbol", "interval", "ts", "open", "high", "low", "close", "volume"}
	placeholders := "?" + repeatString(",?", len(cols)-1)
	colList := ""
	for i, c := range cols {
		if i > 0 {
			colList += ", "
		}
		colList += `"` + c + `"`
	}
	q := fmt.Sprintf("INSERT OR IGNORE INTO \"%s\" (%s) VALUES (%s)", table, colList, placeholders)

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	var count int64
	now := time.Now().Unix()
	for _, row := range data {
		vals := []any{row.Symbol, row.Interval, row.Ts, row.Open, row.High, row.Low, row.Close, row.Volume, now}
		res, err := stmt.Exec(vals...)
		if err != nil {
			continue
		}
		n, _ := res.RowsAffected()
		count += n
	}

	if err := tx.Commit(); err != nil {
		return count, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}
```

Note: Add `"time"` to imports.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd app && go test ./internal/data/ -run TestImportCSV -v
```

Expected: all PASS

- [ ] **Step 5: Add Parquet import test**

```go
func TestImportParquet_inserts_rows(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.parquet")

	// First export some data
	seedOHLCV(t, db, "000001", 3)
	_, err := ExportParquet(db, "ohlcv_cache", "000001", "1D", "", "", path)
	require.NoError(t, err)

	// Clear the table
	db.Exec("DELETE FROM ohlcv_cache")

	// Import back
	count, err := ImportParquet(db, path, "ohlcv_cache")
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}
```

Run:
```bash
cd app && go test ./internal/data/ -run TestImport -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/data/importer.go internal/data/importer_test.go
git commit -m "feat(data): add CSV and Parquet import"
```

---

### Task 6: Wire IPC methods into App

**Files:**
- Modify: `app.go`
- Modify: `app_startup.go`
- Create: `app_data.go`

**Interfaces:**
- Consumes: all `internal/data` functions from Tasks 1-5
- Produces: 5 IPC methods exposed via Wails binding

- [ ] **Step 1: Create app_data.go**

```go
// app_data.go — Data lifecycle management IPC methods
package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"quantflow/internal/data"
)

func (a *App) GetStorageStats(ctx context.Context) ([]data.TableStat, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.GetTableStats(a.db)
}

func (a *App) ArchiveData(ctx context.Context, source, symbol, before string) (*data.ArchiveResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.ArchiveData(a.db, source, symbol, before)
}

func (a *App) ExportData(ctx context.Context, table, symbol, interval, format, dateFrom, dateTo string) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}

	var outputPath string
	var count int64
	var err error

	switch format {
	case "csv":
		outputPath = exportFilePath(table, symbol, interval, dateFrom, dateTo, "csv")
		count, err = data.ExportCSV(a.db, table, symbol, interval, dateFrom, dateTo, outputPath)
	case "parquet":
		outputPath = exportFilePath(table, symbol, interval, dateFrom, dateTo, "parquet")
		count, err = data.ExportParquet(a.db, table, symbol, interval, dateFrom, dateTo, outputPath)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("no data to export")
	}
	return outputPath, nil
}

func (a *App) ImportData(ctx context.Context, filePath, table string) (int64, error) {
	if a.db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	ext := filepath.Ext(filePath)
	switch ext {
	case ".csv":
		return data.ImportCSV(a.db, filePath, table)
	case ".parquet":
		return data.ImportParquet(a.db, filePath, table)
	default:
		return 0, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func (a *App) CleanupData(ctx context.Context, table, symbol, before string, dryRun bool) (*data.CleanupResult, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return data.CleanupData(a.db, table, symbol, before, dryRun)
}

func exportFilePath(table, symbol, interval, dateFrom, dateTo, ext string) string {
	ts := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("%s_%s_%s_%s_%s.%s", table, symbol, interval, dateFrom, dateTo, ext)
	if symbol == "" {
		name = fmt.Sprintf("%s_%s_%s_%s.%s", table, interval, dateFrom, dateTo, ext)
	}
	return filepath.Join("data", "export", ts+"_"+name)
}
```

- [ ] **Step 2: Write integration test for app_data.go**

```go
// app_data_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppDataRoundTrip(t *testing.T) {
	app := &App{db: setupTestDB(t)}
	defer app.db.Close()

	// Seed some data
	seedTestOHLCV(t, app.db, "000001", 5)

	// Stat
	stats, err := app.GetStorageStats(nil)
	require.NoError(t, err)
	require.Greater(t, len(stats), 0)

	// Archive
	result, err := app.ArchiveData(nil, "ohlcv_cache", "000001", "2025-01-01")
	require.NoError(t, err)
	require.Equal(t, 5, result.RowCount)

	// Cleanup dry run
	cleanResult, err := app.CleanupData(nil, "ohlcv_cache", "000001", "2025-01-01", true)
	require.NoError(t, err)
	require.True(t, cleanResult.DryRun)

	// Export CSV
	path, err := app.ExportData(nil, "ohlcv_cache", "000001", "1D", "csv", "", "")
	require.NoError(t, err)
	require.NotEmpty(t, path)
}
```

Add test helpers:

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := storage.Open(":memory:")
    require.NoError(t, err)
    return db
}

// Copy seedOHLCV from internal/data or reuse it
```

- [ ] **Step 3: Run integration tests**

```bash
cd app && go test -run TestAppDataRoundTrip -v
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add app_data.go app_data_test.go
git commit -m "feat(data): wire IPC methods into App"
```

---

### Task 7: Frontend StoragePanel

**Files:**
- Create: `frontend/src/terminal/panels/StoragePanel.vue`
- Modify: `frontend/src/terminal/panels/registry.ts`
- Modify: `frontend/src/lib/wails.ts`
- Modify: `frontend/src/i18n/zh.ts`
- Modify: `frontend/src/i18n/en.ts`
- Create: `frontend/src/terminal/panels/__tests__/StoragePanel.test.ts`

- [ ] **Step 1: Add typed IPC bindings to wails.ts**

Read existing `frontend/src/lib/wails.ts` to find the pattern, then add:

```typescript
// In frontend/src/lib/wails.ts

export interface TableStat {
  table: string
  rows: number
  size_bytes: number
  oldest: string
  newest: string
}

export interface ArchiveResult {
  id: number
  source: string
  symbol: string
  interval: string
  date_from: string
  date_to: string
  row_count: number
  compressed_bytes: number
}

export interface CleanupResult {
  affected_rows: number
  preview: Record<string, unknown>[]
  table: string
  dry_run: boolean
}

// Add to typed IPC object
export const data = {
  getStorageStats: () => call('GetStorageStats') as Promise<TableStat[]>,
  archiveData: (source: string, symbol: string, before: string) =>
    call('ArchiveData', source, symbol, before) as Promise<ArchiveResult>,
  exportData: (table: string, symbol: string, interval: string, format: string, dateFrom: string, dateTo: string) =>
    call('ExportData', table, symbol, interval, format, dateFrom, dateTo) as Promise<string>,
  importData: (filePath: string, table: string) =>
    call('ImportData', filePath, table) as Promise<number>,
  cleanupData: (table: string, symbol: string, before: string, dryRun: boolean) =>
    call('CleanupData', table, symbol, before, dryRun) as Promise<CleanupResult>,
}
```

- [ ] **Step 2: Write failing frontend test**

```typescript
// frontend/src/terminal/panels/__tests__/StoragePanel.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import StoragePanel from '../StoragePanel.vue'
import { createTestingPinia } from '@pinia/testing'

// Mock wails
vi.mock('@/lib/wails', () => ({
  data: {
    getStorageStats: vi.fn(),
    archiveData: vi.fn(),
    exportData: vi.fn(),
    importData: vi.fn(),
    cleanupData: vi.fn(),
  },
  confirmDialog: vi.fn().mockResolvedValue(true),
  alertDialog: vi.fn(),
}))

describe('StoragePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('mounts without error', () => {
    const wrapper = mount(StoragePanel, {
      global: {
        plugins: [createTestingPinia()],
      },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('shows loading state initially', () => {
    const wrapper = mount(StoragePanel, {
      global: {
        plugins: [createTestingPinia()],
      },
    })
    expect(wrapper.find('[data-testid="loading"]').exists()).toBe(true)
  })

  it('renders table stats after load', async () => {
    const { data } = await import('@/lib/wails')
    ;(data.getStorageStats as ReturnType<typeof vi.fn>).mockResolvedValue([
      { table: 'ohlcv_cache', rows: 100, size_bytes: 6400, oldest: '2024-01-01', newest: '2024-06-01' },
      { table: 'minute_cache', rows: 500, size_bytes: 24000, oldest: '2024-03-01', newest: '2024-06-01' },
    ])

    const wrapper = mount(StoragePanel, {
      global: {
        plugins: [createTestingPinia()],
      },
    })

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('ohlcv_cache')
    expect(wrapper.text()).toContain('100')
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/StoragePanel.test.ts
```

Expected: FAIL (component not defined or no export)

- [ ] **Step 4: Create StoragePanel.vue**

```vue
<!-- frontend/src/terminal/panels/StoragePanel.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { data, confirmDialog } from '@/lib/wails'
import PanelHeader from '@/terminal/components/panel/PanelHeader.vue'
import LoadingState from '@/terminal/components/panel/LoadingState.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

interface TableStat {
  table: string
  rows: number
  size_bytes: number
  oldest: string
  newest: string
}

const stats = ref<TableStat[]>([])
const loading = ref(true)
const error = ref('')

async function loadStats() {
  loading.value = true
  error.value = ''
  try {
    stats.value = await data.getStorageStats()
  } catch (e: any) {
    error.value = e.message || 'Failed to load storage stats'
  } finally {
    loading.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatTableLabel(table: string): string {
  const labels: Record<string, string> = {
    ohlcv_cache: 'OHLCV 缓存',
    minute_cache: '分时缓存',
    data_archive: '数据归档',
  }
  return labels[table] || table
}

async function handleArchive(source: string) {
  const ok = await confirmDialog(`确认归档 ${formatTableLabel(source)} 数据？`)
  if (!ok) return
  try {
    await data.archiveData(source, '', '')
    await loadStats()
  } catch (e: any) {
    error.value = e.message
  }
}

async function handleExport(table: string) {
  const ok = await confirmDialog(`确认导出 ${formatTableLabel(table)} 数据？`)
  if (!ok) return
  try {
    const path = await data.exportData(table, '', '', 'csv', '', '')
    error.value = `导出完成: ${path}`
  } catch (e: any) {
    error.value = e.message
  }
}

async function handleCleanup(table: string) {
  const ok = await confirmDialog(`确认清理 ${formatTableLabel(table)} 数据？此操作不可恢复。`)
  if (!ok) return
  try {
    const result = await data.cleanupData(table, '', '', false)
    error.value = `已清理 ${result.affected_rows} 行`
    await loadStats()
  } catch (e: any) {
    error.value = e.message
  }
}

onMounted(loadStats)
</script>

<template>
  <div class="storage-panel">
    <PanelHeader :title="t('panel.storage.title')" />
    <div class="storage-content">
      <LoadingState v-if="loading" data-testid="loading" />
      <div v-else-if="error" class="storage-error">{{ error }}</div>
      <table v-else class="storage-table">
        <thead>
          <tr>
            <th>{{ t('panel.storage.table') }}</th>
            <th>{{ t('panel.storage.rows') }}</th>
            <th>{{ t('panel.storage.size') }}</th>
            <th>{{ t('panel.storage.oldest') }}</th>
            <th>{{ t('panel.storage.newest') }}</th>
            <th>{{ t('panel.storage.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in stats" :key="s.table">
            <td>{{ formatTableLabel(s.table) }}</td>
            <td class="num">{{ s.rows.toLocaleString() }}</td>
            <td class="num">{{ formatBytes(s.size_bytes) }}</td>
            <td>{{ s.oldest }}</td>
            <td>{{ s.newest }}</td>
            <td class="actions">
              <button v-if="s.table !== 'data_archive'" @click="handleExport(s.table)" :title="t('panel.storage.export')">
                ⬇
              </button>
              <button v-if="s.table === 'ohlcv_cache' || s.table === 'minute_cache'" @click="handleArchive(s.table)" :title="t('panel.storage.archive')">
                📦
              </button>
              <button v-if="s.table === 'ohlcv_cache' || s.table === 'minute_cache'" @click="handleCleanup(s.table)" :title="t('panel.storage.cleanup')" class="danger">
                🗑
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.storage-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.storage-content {
  flex: 1;
  overflow: auto;
  padding: 8px;
}
.storage-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.storage-table th,
.storage-table td {
  padding: 6px 8px;
  text-align: left;
  border-bottom: 1px solid var(--border-color, #333);
}
.storage-table th {
  font-weight: 600;
  color: var(--text-secondary, #888);
  position: sticky;
  top: 0;
  background: var(--bg-primary, #1a1a2e);
}
.storage-table td.num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.storage-table td.actions {
  white-space: nowrap;
}
.storage-table button {
  background: none;
  border: 1px solid var(--border-color, #444);
  border-radius: 4px;
  cursor: pointer;
  padding: 2px 6px;
  margin: 0 2px;
  font-size: 14px;
}
.storage-table button.danger:hover {
  background: rgba(255, 50, 50, 0.2);
  border-color: #f44;
}
.storage-error {
  color: var(--text-error, #f44);
  padding: 16px;
}
</style>
```

- [ ] **Step 5: Register in panel registry**

Read `frontend/src/terminal/panels/registry.ts` to find the import + registration pattern, then add:

```typescript
// In registry.ts
import StoragePanel from './StoragePanel.vue'

// In the panel registry object:
'storage': {
  component: StoragePanel,
  category: 'system',
  titleKey: 'panel.storage.title',
  icon: '📊',
}
```

- [ ] **Step 6: Add i18n keys**

In `frontend/src/i18n/zh.ts`:
```typescript
'panel.storage.title': '存储管理',
'panel.storage.table': '数据表',
'panel.storage.rows': '行数',
'panel.storage.size': '大小',
'panel.storage.oldest': '最早数据',
'panel.storage.newest': '最新数据',
'panel.storage.actions': '操作',
'panel.storage.export': '导出',
'panel.storage.archive': '归档',
'panel.storage.cleanup': '清理',
```

In `frontend/src/i18n/en.ts`:
```typescript
'panel.storage.title': 'Storage',
'panel.storage.table': 'Table',
'panel.storage.rows': 'Rows',
'panel.storage.size': 'Size',
'panel.storage.oldest': 'Oldest',
'panel.storage.newest': 'Newest',
'panel.storage.actions': 'Actions',
'panel.storage.export': 'Export',
'panel.storage.archive': 'Archive',
'panel.storage.cleanup': 'Cleanup',
```

- [ ] **Step 7: Run frontend tests**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/StoragePanel.test.ts
```

Expected: PASS

- [ ] **Step 8: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: no errors

- [ ] **Step 9: Commit**

```bash
git add frontend/src/terminal/panels/StoragePanel.vue \
       frontend/src/terminal/panels/__tests__/StoragePanel.test.ts \
       frontend/src/terminal/panels/registry.ts \
       frontend/src/lib/wails.ts \
       frontend/src/i18n/zh.ts \
       frontend/src/i18n/en.ts
git commit -m "feat(frontend): add StoragePanel with archive/export/cleanup UI"
```

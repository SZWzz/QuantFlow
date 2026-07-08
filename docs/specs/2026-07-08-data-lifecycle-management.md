# Data Lifecycle Management — 数据生命周期管理

## Motivation

QuantFlow 的 SQLite 数据库随使用持续膨胀。当前 `ohlcv_cache` 和 `minute_cache` 表无止境增长，`backtest_results` 存储完整 OHLCV JSON blob，用户没有工具查看用量或控制数据规模。

用户反馈需要的 5 项能力：
1. 归档/压缩旧数据（保持可查但缩小体积）
2. CSV/Parquet 导出（外部分析用）
3. CSV/Parquet 导入（历史数据灌入）
4. 手动清理工具（预览 + 安全删除）
5. 存储用量监控面板（查看各表大小）

## Design

### Architecture

新增 `internal/data/` 包，与现有 `internal/storage/` 职责分离。

```
internal/data/
├── archiver.go     归档 → 压缩 BLOB → 写入 data_archive 表
├── exporter.go     导出 → SQLite → CSV / Parquet → data/export/
├── importer.go     导入 → CSV / Parquet → SQLite
├── cleaner.go      清理 → 预览 + 删除 (dryRun 安全模式)
└── stats.go        统计 → 各表行数 / 大小 / 时间范围
```

App struct 新增 5 个 IPC 方法，暴露给前端：

```go
func (a *App) GetStorageStats(ctx context.Context) ([]TableStat, error)
func (a *App) ArchiveData(ctx context.Context, source, symbol, before string) (ArchiveResult, error)
func (a *App) ExportData(ctx context.Context, table, symbol, interval, format, dateFrom, dateTo string) (string, error)
func (a *App) ImportData(ctx context.Context, filePath, table string) (int64, error)
func (a *App) CleanupData(ctx context.Context, table, symbol, before string, dryRun bool) (CleanupResult, error)
```

### Archive Table (Migration 017)

```sql
CREATE TABLE data_archive (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source      TEXT    NOT NULL,       -- 'ohlcv_cache' | 'minute_cache' | 'backtest_results'
    symbol      TEXT    NOT NULL,
    interval    TEXT    NOT NULL DEFAULT '',
    date_from   TEXT    NOT NULL,       -- ISO 8601
    date_to     TEXT    NOT NULL,
    row_count   INTEGER NOT NULL,
    data        BLOB    NOT NULL,       -- gzip(JSON)
    archived_at TEXT    NOT NULL DEFAULT (datetime('now')),
    checksum    TEXT    NOT NULL        -- SHA256 of uncompressed JSON
);

CREATE INDEX idx_archive_source ON data_archive(source, symbol);
CREATE INDEX idx_archive_date   ON data_archive(date_from, date_to);
```

### Data Flow

```
                    ArchiveData
原始表 ──────────────────────────→ data_archive (压缩 BLOB)
ohlcv_cache                          原始行保留
minute_cache
backtest_results

                    ExportData
原始表 / data_archive ─────────────→ data/export/<table>/symbol_interval_from_to.{csv,parquet}
ohlcv_cache
minute_cache

                    ImportData
{CSV, Parquet} 文件 ──────────────→ 原始表
                                    (INSERT OR IGNORE)

                    CleanupData (dryRun=false)
原始表 ────────────────────────────→ DELETE WHERE symbol + date 条件
ohlcv_cache
minute_cache

                    CleanupData (dryRun=true)
原始表 ────────────────────────────→ {rowCount, preview[10]}
                                    (只查询，不删除)

StoragePanel.vue
    ↓ GetStorageStats()
    ↓
[{table, rows, size_bytes, oldest, newest}, ...]
```

### New / Modified Files

| File | Action | Purpose |
|------|--------|---------|
| `internal/data/archiver.go` | CREATE | Archive/unarchive logic |
| `internal/data/exporter.go` | CREATE | CSV & Parquet export |
| `internal/data/importer.go` | CREATE | CSV & Parquet import |
| `internal/data/cleaner.go` | CREATE | Preview + delete |
| `internal/data/stats.go` | CREATE | Storage statistics |
| `internal/storage/migrations/017_data_archive.sql` | CREATE | Archive table DDL |
| `app.go` | MODIFY | Register 5 new IPC methods |
| `app_startup.go` | MODIFY | Init data service |
| `frontend/src/terminal/panels/StoragePanel.vue` | CREATE | Storage monitoring panel |
| `frontend/src/terminal/panels/registry.ts` | MODIFY | Register StoragePanel |
| `frontend/src/lib/wails.ts` | MODIFY | Add typed IPC bindings |
| `frontend/src/i18n/zh.ts` | MODIFY | Chinese labels |
| `frontend/src/i18n/en.ts` | MODIFY | English labels |
| `go.mod` | MODIFY | Add `github.com/segmentio/parquet-go` |

### Exported API

**Go types:**

```go
type TableStat struct {
    Table    string `json:"table"`
    Rows     int64  `json:"rows"`
    SizeBytes int64 `json:"size_bytes"`
    Oldest   string `json:"oldest"`
    Newest   string `json:"newest"`
}

type ArchiveResult struct {
    Table      string `json:"table"`
    Symbol     string `json:"symbol"`
    DateFrom   string `json:"date_from"`
    DateTo     string `json:"date_to"`
    RowCount   int    `json:"row_count"`
    Compressed int    `json:"compressed_bytes"`
}

type CleanupResult struct {
    AffectedRows int64         `json:"affected_rows"`
    Preview      []map[string]any `json:"preview"`       // first 10 rows
    Table        string        `json:"table"`
    DryRun       bool          `json:"dry_run"`
}
```

**Frontend Pinia store:** None needed. StoragePanel is self-contained via `useWailsApp()`.

### Cleanup Safety Rules

- `CleanupData` 只允许操作 `ohlcv_cache` 和 `minute_cache` 两张表
- 不允许清理 orders/trades/portfolio/backtest_results/credentials 等业务表
- `dryRun=true` 是默认值，需要用户在前端确认 `confirmDialog()` 后才传 `dryRun=false`
- 清理条件按 `symbol + before (date)` 过滤，不传 symbol = 全量（需再次确认）

### Export Path Convention

```
data/export/<table>/<symbol>_<interval>_<dateFrom>_<dateTo>.<fmt>
```

示例:
```
data/export/ohlcv_cache/000001_1D_2024-01-01_2024-12-31.csv
data/export/minute_cache/000001_1m_2026-07-01_2026-07-08.parquet
```

## Acceptance Criteria

- [ ] `GetStorageStats` 返回所有数据表的行数/大小/时间范围，准确度 ±10%
- [ ] `ArchiveData` 将指定数据压缩为 gzip BLOB 写入 `data_archive`，原始行不受影响
- [ ] `ExportData` CSV 输出可直接用 Excel/Pandas 打开，Parquet 输出可用 PyArrow/Polars 读取
- [ ] `ImportData` 能正确解析 CSV 和 Parquet 并写入对应表，重复行自动跳过 (INSERT OR IGNORE)
- [ ] `CleanupData` dryRun 返回准确行数 + 预览，实际执行后不可恢复（用户通过 `confirmDialog` 确认）
- [ ] `StoragePanel.vue` 展示所有表用量、提供归档/导出/清理操作入口
- [ ] 所有操作有错误处理和前端 loading/error 状态

## Testing

### Go Tests

All tests use an in-memory SQLite (`:memory:`) with the same schema as production. Tests run via `go test ./internal/data/...` in `app/` directory.

| File | Tests | Coverage Target |
|------|-------|-----------------|
| `internal/data/archiver_test.go` | 归档 → BLOB 压缩/解压缩 roundtrip, checksum 校验, 无效源表处理, 无数据归档返回零行 | 90%+ |
| `internal/data/exporter_test.go` | CSV 写入正确 header + 行, Parquet schema 匹配, 空数据集导出返回空文件, 无效 format 返回错误 | 85%+ |
| `internal/data/importer_test.go` | CSV 正常导入行数, Parquet 正常导入, 重复行 INSERT OR IGNORE 不报错, 列名不匹配返回友好错误 | 85%+ |
| `internal/data/cleaner_test.go` | dryRun 返回准确行数 + 预览, 实际删除后 COUNT=0, 不允许的表名返回错误, 不传 symbol 全量删除的边界 | 90%+ |
| `internal/data/stats_test.go` | 插入已知行后统计行数匹配, 大小 >0, 时间范围正确, 空库返回零值 | 90%+ |

**测试模式：**

```go
func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    // apply migrations 005 (ohlcv_cache) + 012 (minute_cache) + 017 (data_archive)
    for _, ddl := range []string{ohlcvCacheDDL, minuteCacheDDL, dataArchiveDDL} {
        _, err := db.Exec(ddl)
        require.NoError(t, err)
    }
    return db
}
```

### Frontend Test

| File | Tests |
|------|-------|
| `StoragePanel.test.ts` | mount 时不报错, mock `GetStorageStats` 返回数据后渲染行, loading 状态展示骨架屏, error 状态展示 ErrorBoundary |

使用 `vitest` + `@vue/test-utils`，mock IPC 通过 `vi.mock('@/lib/wails')`。

### Test Data

```go
func seedOHLCV(db *sql.DB, symbol string, count int) {
    for i := 0; i < count; i++ {
        ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Unix()
        db.Exec("INSERT OR IGNORE INTO ohlcv_cache VALUES (?,?,?,?,?,?,?,?)",
            symbol, "1D", ts, 10.0, 11.0, 9.0, 10.5, 100000, time.Now().Unix())
    }
}
```

## Risks / Trade-offs

- **归档不解压直接查询**：压缩数据无法直接 SELECT，如需查询历史需解归档到原始表。对于"偶尔查旧数据"场景可接受；频繁查询历史不合适。
- **Parquet 依赖纯 Go 库**：`segmentio/parquet-go` 功能完整但 Columnar 写放大可能比 CSV 慢。大文件导出建议异步 + 进度通知。
- **单文件 SQLite 膨胀**：即使归档，数据仍在一个文件内。SQLite 单文件上限 140TB，实际使用中远低于此。
- **导入格式限制**：CSV 需第一行为 header，列名必须匹配目标表列名。Parquet 需 schema 兼容。不提供智能列映射。

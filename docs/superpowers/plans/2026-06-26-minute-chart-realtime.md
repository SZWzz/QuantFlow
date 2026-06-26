# 分时图实时绘制 & 数据持久化 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让分时图像同花顺一样平滑实时绘制，消除"刷新闪烁"，且切 Tab/重启后数据不丢。

**Architecture:** Go 后端新增 `MinuteCache` 服务层（SQLite 持久化 + LRU 热缓存），前端 CandlestickPanel 改为增量数据请求 + ECharts 静默合并更新。

**Tech Stack:** Go 1.22+, hashicorp/golang-lru/v2, SQLite WAL, Vue 3 + vue-echarts (ECharts 5)

## Global Constraints

- SQLite 是唯一数据库，新增 `012_minute_cache` 迁移
- 前端通过 Wails v3 `Call.ByName` 调用 Go 方法
- `MinuteTick` 类型沿用 `internal/market/minuteline.go:5`
- 轮询间隔保持 10s
- LRU 上限 500 entries，使用已存在的 `github.com/hashicorp/golang-lru/v2`

---

### Task 1: SQLite 迁移 — minute_cache 表

**Files:**
- Create: `internal/storage/migrations/012_minute_cache.sql`

**Interfaces:**
- Produces: `minute_cache` 表（symbol, date, tick_time, price, volume, avg_price）

- [ ] **Step 1: 创建迁移文件**

创建 `internal/storage/migrations/012_minute_cache.sql`：

```sql
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
```

- [ ] **Step 2: 验证迁移**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
# 确认文件格式正确（数字前缀 _ 描述 .sql）
ls -la internal/storage/migrations/012_minute_cache.sql
```

- [ ] **Step 3: Commit**

```bash
git add internal/storage/migrations/012_minute_cache.sql
git commit -m "feat: add minute_cache SQLite migration"
```

---

### Task 2: Go 后端 — MinuteCache 服务

**Files:**
- Create: `internal/market/minute_cache.go`
- Create: `internal/market/minute_cache_test.go`

**Interfaces:**
- Produces:
  - `func NewMinuteCache(db *sql.DB) (*MinuteCache, error)` — 构造函数，建表 + 初始化 LRU
  - `func (mc *MinuteCache) GetIncremental(symbol string, since int64) ([]MinuteTick, error)` — 增量获取
  - `func (mc *MinuteCache) Close() error` — 关闭
- Consumes: `market.MinuteTick` from `internal/market/minuteline.go`

- [ ] **Step 1: 编写单元测试**

创建 `internal/market/minute_cache_test.go`：

```go
package market

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMinuteCache_GetIncremental_FirstCall(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	// First call: no data yet, should return empty
	ticks, err := mc.GetIncremental("600519", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 0 {
		t.Errorf("expected 0 ticks, got %d", len(ticks))
	}
}

func TestMinuteCache_SaveAndGet(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	// Save some ticks
	input := []MinuteTick{
		{Time: "09:30", Price: 100.5, Volume: 1000, AvgPrice: 100.5},
		{Time: "09:31", Price: 100.8, Volume: 2000, AvgPrice: 100.65},
	}
	if err := mc.SaveTicks("600519", "2026-06-26", input); err != nil {
		t.Fatal(err)
	}

	// Get all ticks
	ticks, err := mc.GetIncremental("600519", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 2 {
		t.Fatalf("expected 2 ticks, got %d", len(ticks))
	}
	if ticks[0].Time != "09:30" || ticks[0].Price != 100.5 {
		t.Errorf("unexpected tick[0]: %+v", ticks[0])
	}
}

func TestMinuteCache_GetIncremental_Subsequent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	input := []MinuteTick{
		{Time: "09:30", Price: 100.0, Volume: 100, AvgPrice: 100.0},
		{Time: "09:31", Price: 101.0, Volume: 200, AvgPrice: 100.5},
		{Time: "09:32", Price: 102.0, Volume: 150, AvgPrice: 101.0},
	}
	if err := mc.SaveTicks("600519", "2026-06-26", input); err != nil {
		t.Fatal(err)
	}

	// since = unix of 09:31:00
	sinceUnix := parseTimeToUnix("2026-06-26", "09:31")
	ticks, err := mc.GetIncremental("600519", sinceUnix)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 1 {
		t.Fatalf("expected 1 new tick after 09:31, got %d", len(ticks))
	}
	if ticks[0].Time != "09:32" {
		t.Errorf("expected 09:32, got %s", ticks[0].Time)
	}
}

func TestMinuteCache_LRUFull(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mc, err := NewMinuteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	defer mc.Close()

	// Fill LRU beyond capacity (500 entries is huge, simulate with small cache)
	// This test verifies the LRU eviction doesn't panic
	for i := 0; i < 10; i++ {
		symbol := "6005" + string(rune('0'+i))
		ticks := []MinuteTick{{Time: "09:30", Price: 100.0, Volume: 100, AvgPrice: 100.0}}
		if err := mc.SaveTicks(symbol, "2026-06-26", ticks); err != nil {
			t.Fatal(err)
		}
	}

	// Verify first inserted symbol can still be loaded from DB
	ticks, err := mc.GetIncremental("60050", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ticks) != 1 {
		t.Errorf("expected 1 tick from DB fallback, got %d", len(ticks))
	}
}

func parseTimeToUnix(date, timeStr string) int64 {
	// Helper for test — parse "2026-06-26 09:31" to Unix timestamp
	t, _ := time.Parse("2006-01-02 15:04", date+" "+timeStr)
	return t.Unix()
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go test ./internal/market/ -run TestMinuteCache -v
```

预期：编译失败，`NewMinuteCache` 未定义。

- [ ] **Step 3: 实现 MinuteCache**

创建 `internal/market/minute_cache.go`：

```go
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
	today := time.Now().Format("2006-01-02")
	key := symbol + ":" + today

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

	ticks, err := mc.loadFromDB(symbol, today)
	if err != nil {
		return nil, fmt.Errorf("minute_cache: load %s %s: %w", symbol, today, err)
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
    PRIMARY KEY (symbol, date, tick_time)
) WITHOUT ROWID;

CREATE INDEX IF NOT EXISTS idx_minute_sym_date ON minute_cache(symbol, date);
`

func (mc *MinuteCache) loadFromDB(symbol, date string) ([]MinuteTick, error) {
	rows, err := mc.db.Query(
		"SELECT tick_time, price, volume, avg_price FROM minute_cache WHERE symbol=? AND date=? ORDER BY tick_time",
		symbol, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ticks []MinuteTick
	for rows.Next() {
		var t MinuteTick
		if err := rows.Scan(&t.Time, &t.Price, &t.Volume, &t.AvgPrice); err != nil {
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
		"INSERT OR IGNORE INTO minute_cache (symbol, date, tick_time, price, volume, avg_price) VALUES (?, ?, ?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range ticks {
		if _, err := stmt.Exec(symbol, date, t.Time, t.Price, t.Volume, t.AvgPrice); err != nil {
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
	ref := time.Unix(since, 0).Format("15:04")
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
```

- [ ] **Step 4: 运行测试确认通过**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go test ./internal/market/ -run TestMinuteCache -v
```

预期：所有 4 个测试通过。

- [ ] **Step 5: Commit**

```bash
git add internal/market/minute_cache.go internal/market/minute_cache_test.go
git commit -m "feat: add MinuteCache service with SQLite persistence + LRU"
```

---

### Task 3: Go 后端 — 集成 MinuteCache 到 App

**Files:**
- Modify: `app.go` — App struct 加 `minuteCache` 字段
- Modify: `app.go` — `GetMinuteLine` 改用 `MinuteCache`
- Modify: `app.go` — `Startup` 中初始化 `minuteCache`

**Interfaces:**
- Modifies: `GetMinuteLine(ctx context.Context, symbol string)` → `GetMinuteLine(ctx context.Context, symbol string, sinceTimestamp int64)`
- Consumes: `MinuteCache.GetIncremental`, `MinuteCache.SaveTicks`

- [ ] **Step 1: 在 App struct 添加 minuteCache 字段**

编辑 `app.go`，在 `App` struct 中添加字段（在 `emitter` 行下方）：

```go
// Read the current struct
```

找到 `app.go:46` 的 `emitter` 字段行，在其下方添加：

```go
minuteCache   *market.MinuteCache
```

- [ ] **Step 2: 在 Startup 中初始化 minuteCache**

编辑 `app.go`，在 `Startup` 方法中初始化数据库之后添加：

```go
mc, err := market.NewMinuteCache(a.db)
if err != nil {
    slog.Error("failed to init minute cache", "err", err)
} else {
    a.minuteCache = mc
}
```

- [ ] **Step 3: 重写 GetMinuteLine 使用 MinuteCache**

编辑 `app.go`，替换 `GetMinuteLine` 方法（app.go:563-576）：

```go
// GetMinuteLine returns today's intraday minute-by-minute ticks for a CN symbol.
// If sinceTimestamp is 0, returns all ticks for today.
// If sinceTimestamp > 0, returns only ticks after the given Unix timestamp.
// Data is cached in SQLite + LRU; source data comes from mootdx when not cached.
func (a *App) GetMinuteLine(ctx context.Context, symbol string, sinceTimestamp int64) ([]market.MinuteTick, string, error) {
	if a.minuteCache == nil {
		return nil, "unavailable", fmt.Errorf("minute cache not initialized")
	}

	// 1. Try cache first (SQLite + LRU).
	ticks, err := a.minuteCache.GetIncremental(symbol, sinceTimestamp)
	if err != nil {
		slog.Warn("minute_cache: get failed", "symbol", symbol, "err", err)
		// Fall through to live fetch.
	}

	// 2. If cache has data and the request is incremental (since > 0),
	//    return cached data. For initial load (since == 0), if cache
	//    is empty, fall through to live fetch.
	if len(ticks) > 0 || sinceTimestamp > 0 {
		return ticks, "cache", nil
	}

	// 3. Live fetch via mootdx.
	adpt := a.getMootdxAdapter()
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	liveTicks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}

	// 4. Persist live data to cache.
	if len(liveTicks) > 0 {
		today := time.Now().Format("2006-01-02")
		if err := a.minuteCache.SaveTicks(symbol, today, liveTicks); err != nil {
			slog.Warn("minute_cache: save failed", "symbol", symbol, "err", err)
		}
	}

	return liveTicks, "mootdx", nil
}
```

- [ ] **Step 4: 编译验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go build -o /dev/null .
```

预期：编译成功，无错误。

- [ ] **Step 5: Commit**

```bash
git add app.go
git commit -m "feat: integrate MinuteCache into GetMinuteLine with incremental support"
```

---

### Task 4: 前端 — CandlestickPanel 增量渲染

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue` — `loadMinuteLine` 改为增量调用
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue` — `minuteChartOption` 禁止动画
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue` — 添加 `inject` 获取共享数据 Map

**Interfaces:**
- Consumes: `provide('minuteDataCache', reactiveMap)` from DockView
- Modifies: `loadMinuteLine()` → 传 `sinceTimestamp`

- [ ] **Step 1: 修改分钟图 option，关闭动画**

编辑 `frontend/src/terminal/panels/CandlestickPanel.vue` 的 `minuteChartOption` computed（line 221），在返回对象的顶层添加：

```typescript
animation: false,
animationDurationUpdate: 0,
animationEasingUpdate: 'linear',
```

完整 diff：

```typescript
const minuteChartOption = computed(() => {
  if (!minuteTicks.value.length) return {}
  // ... existing code ...
  return {
    animation: false,           // <-- 新增
    animationDurationUpdate: 0, // <-- 新增
    backgroundColor: 'transparent',
    // ... rest unchanged ...
  }
})
```

- [ ] **Step 2: 引入 inject 读取共享分钟数据**

在 `CandlestickPanel.vue` script 顶部 import 区域附近添加：

```typescript
import { inject, ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'

// 从父级 DockView 获取共享的分时数据 Map
const minuteDataCache = inject<Map<string, MinuteTick[]>>('minuteDataCache', new Map())
```

- [ ] **Step 3: 修改 loadMinuteLine 支持增量调用**

编辑 `loadMinuteLine` 函数（line 83），改为：

```typescript
async function loadMinuteLine() {
  const app = (window as any).go?.main?.App
  if (!app) return
  minuteLoading.value = true
  try {
    // 计算上次最后一个 tick 的时间戳作为增量锚点
    const lastTick = minuteTicks.value.length > 0
      ? minuteTicks.value[minuteTicks.value.length - 1]
      : null
    const sinceTimestamp = lastTick
      ? parseMinuteTimeToUnix(lastTick.time)
      : 0

    const result = await app.GetMinuteLine(symbol.value, sinceTimestamp)
    const ticks: MinuteTick[] = Array.isArray(result) ? result[0] : result
    if (!Array.isArray(ticks) || ticks.length === 0) {
      return
    }

    if (sinceTimestamp === 0) {
      // 首次加载：全量替换
      minuteTicks.value = ticks
    } else {
      // 增量更新：追加新 tick
      const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
      for (const t of ticks) {
        existing.set(t.time, t)
      }
      minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
    }

    if (prevClose.value === 0 && minuteTicks.value.length > 0) {
      prevClose.value = minuteTicks.value[0].price
    }

    // 更新共享缓存
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  } catch {
    // silent
  } finally {
    minuteLoading.value = false
  }
}
```

- [ ] **Step 4: 添加辅助函数**

在文件顶部附近（和其他辅助函数一起）添加：

```typescript
function getTodayDateString(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function parseMinuteTimeToUnix(timeStr: string): number {
  // timeStr like "09:30", combine with today's date
  const today = getTodayDateString()
  const d = new Date(`${today}T${timeStr}:00+08:00`)
  return Math.floor(d.getTime() / 1000)
}
```

- [ ] **Step 5: 修改 symbol watch，优先读取缓存**

编辑 `watch(() => symbol.value, ...)` (line 160)，改为先查缓存：

```typescript
watch(() => symbol.value, (newSymbol, oldSymbol) => {
  // 保存旧 symbol 数据到共享缓存
  if (oldSymbol) {
    const cacheKey = `${oldSymbol}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }

  // 尝试从共享缓存恢复新 symbol 数据
  const cacheKey = `${newSymbol}:${getTodayDateString()}`
  const cached = minuteDataCache.get(cacheKey)
  if (cached && cached.length > 0) {
    minuteTicks.value = cached
    prevClose.value = cached[0].price
  } else {
    minuteTicks.value = []
    prevClose.value = 0
  }

  if (activeTab.value === 'minute') {
    loadMinuteLine()
  }
})
```

- [ ] **Step 6: 修改 onUnmounted，保存当前数据**

在 `onUnmounted` 回调中（line 157 附近）添加：

```typescript
onUnmounted(() => {
  stopMinutePolling()
  // 保存当前数据到共享缓存，组件销毁后数据不丢
  if (symbol.value && minuteTicks.value.length > 0) {
    const cacheKey = `${symbol.value}:${getTodayDateString()}`
    minuteDataCache.set(cacheKey, minuteTicks.value)
  }
})
```

- [ ] **Step 7: 构建前端验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend
npm run build -q
```

预期：构建成功，无 TS/Vue 编译错误。

- [ ] **Step 8: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue
git commit -m "feat: minute chart incremental rendering with animation disabled"
```

---

### Task 5: 前端 — DockView 提供共享分钟数据

**Files:**
- Modify: `frontend/src/terminal/DockView/DockView.vue` — 添加 `provide('minuteDataCache', ...)`

**Interfaces:**
- Produces: `provide('minuteDataCache', reactiveMap)` 供 CandlestickPanel inject

- [ ] **Step 1: 在 DockView 中添加 provide**

编辑 `DockView.vue`，在 `<script setup>` 顶层添加：

```typescript
import { provide, reactive } from 'vue'
import type { MinuteTick } from '@/terminal/panels/CandlestickPanel.vue'

// 跨组件共享的分时数据缓存：key = "symbol:date"
const minuteDataCache = reactive(new Map<string, MinuteTick[]>())
provide('minuteDataCache', minuteDataCache)
```

> 注意：如果 DockView 组件嵌套多层，确认 provide 在渲染 CandlestickPanel 的祖先组件中。

- [ ] **Step 2: 构建验证**

```bash
cd /Volumes/etx/coding/rebuild/quantflow/frontend
npm run build -q
```

预期：构建成功。

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/DockView/DockView.vue
git commit -m "feat: provide shared minute data cache across panels"
```

---

### Task 6: 全栈打包验证

**Files:**
- 无新文件，验证所有改动正确集成。

- [ ] **Step 1: 完整构建**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
# 前端
cd frontend && npm run build -q && cd ..
# Go
go build -o build/quantflow .
# Python sidecar
rsync -a --delete --exclude='.venv/' --exclude='__pycache__/' --exclude='*.pyc' --exclude='tests/' --exclude='.DS_Store' python/ build/python/
test -d python/.venv && ln -sfn "$(pwd)/python/.venv" build/python/.venv || true
```

预期：三个步骤均成功。

- [ ] **Step 2: 运行 Go 测试**

```bash
cd /Volumes/etx/coding/rebuild/quantflow
go test ./internal/market/ -run TestMinuteCache -v
```

预期：所有 MinuteCache 测试通过。

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat: complete minute chart realtime rendering with persistence"
```

# 实施计划：Off-Hours Data Cache

参考：`docs/specs/2026-07-05-off-hours-data-cache.md`

## Task 1: 创建通用 OffHoursCache 工具

**文件**: `internal/market/offhours.go`（新建）

```go
package market

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"log/slog"
)

// OffHoursCache provides sync.Map + JSON persistence for off-hours data.
// Type parameter T must be JSON-serializable.
type OffHoursCache[T any] struct {
	mu   sync.Mutex
	data map[string]T
	path string
	name string
}

func NewOffHoursCache[T any](name string) *OffHoursCache[T] {
	return &OffHoursCache[T]{
		data: make(map[string]T),
		name: name,
	}
}

func (c *OffHoursCache[T]) SetPath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.path = path
}

func (c *OffHoursCache[T]) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" {
		return nil
	}
	b, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var data map[string]T
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	c.data = data
	slog.Info("loaded off-hours cache", "name", c.name, "count", len(data), "path", c.path)
	return nil
}

func (c *OffHoursCache[T]) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.path == "" {
		return nil
	}
	if len(c.data) == 0 {
		return nil
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, c.path); err != nil {
		return err
	}
	slog.Debug("saved off-hours cache", "name", c.name, "count", len(c.data))
	return nil
}

func (c *OffHoursCache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.data[key]
	return v, ok
}

func (c *OffHoursCache[T]) Set(key string, val T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = val
}

func (c *OffHoursCache[T]) GetAll() map[string]T {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[string]T, len(c.data))
	for k, v := range c.data {
		cp[k] = v
	}
	return cp
}

func (c *OffHoursCache[T]) SetAll(data map[string]T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
}
```

**测试**: `internal/market/offhours_test.go`

```go
package market

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOffHoursCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	type Item struct {
		Value int `json:"v"`
	}

	cache := NewOffHoursCache[Item]("test")
	cache.SetPath(path)

	// Set and save
	cache.Set("a", Item{Value: 1})
	cache.Set("b", Item{Value: 2})
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}

	// Load into new instance
	cache2 := NewOffHoursCache[Item]("test")
	cache2.SetPath(path)
	if err := cache2.Load(); err != nil {
		t.Fatal(err)
	}

	v, ok := cache2.Get("a")
	if !ok || v.Value != 1 {
		t.Fatalf("expected a=1, got %+v", v)
	}
	v, ok = cache2.Get("b")
	if !ok || v.Value != 2 {
		t.Fatalf("expected b=2, got %+v", v)
	}

	// Missing key
	_, ok = cache2.Get("c")
	if ok {
		t.Fatal("expected c to be missing")
	}

	// GetAll
	all := cache2.GetAll()
	if len(all) != 2 {
		t.Fatalf("expected 2 items, got %d", len(all))
	}

	// SetAll
	cache2.SetAll(map[string]Item{"x": {Value: 9}})
	if err := cache2.Save(); err != nil {
		t.Fatal(err)
	}

	cache3 := NewOffHoursCache[Item]("test")
	cache3.SetPath(path)
	cache3.Load()
	v, ok = cache3.Get("x")
	if !ok || v.Value != 9 {
		t.Fatalf("expected x=9, got %+v", v)
	}
	_, ok = cache3.Get("a")
	if ok {
		t.Fatal("expected a to be removed after SetAll")
	}

	// Missing file
	cache4 := NewOffHoursCache[Item]("test")
	cache4.SetPath(filepath.Join(dir, "nonexistent.json"))
	if err := cache4.Load(); err != nil {
		t.Fatal("expected no error for missing file:", err)
	}
}
```

运行: `cd app && go test ./internal/market/ -run TestOffHoursCache -v -count=1`

---

## Task 2: Wire GetIndustryRanks

**文件**: `app.go`

**修改**:
1. `App` 结构体加字段 `industryRanksCache *market.OffHoursCache[[]market.IndustryRank]`
2. `GetIndustryRanks` 方法改造

**app.go App 结构体** — 增加:
```go
industryRanksCache *market.OffHoursCache[[]market.IndustryRank]
```

**app.go Init 函数** — 注册路径:
```go
a.industryRanksCache = market.NewOffHoursCache[[]market.IndustryRank]("industry_ranks")
offDir := filepath.Join(filepath.Dir(a.resolvedDBPath), "offhours")
a.industryRanksCache.SetPath(filepath.Join(offDir, "industry_ranks.json"))
if err := a.industryRanksCache.Load(); err != nil {
    slog.Warn("load industry ranks cache", "error", err)
}
```

**app.go GetIndustryRanks**:
```go
func (a *App) GetIndustryRanks(mkt string, topN int) ([]market.IndustryRank, error) {
	if topN <= 0 {
		topN = 20
	}
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	if !market.IsTradingHours(a.resolveMarket(mkt)) {
		if cached, ok := a.industryRanksCache.Get(mkt); ok {
			if len(cached) > topN {
				cached = cached[:topN]
			}
			return cached, nil
		}
		return nil, fmt.Errorf("market %q is currently closed (no cached data)", mkt)
	}
	ranks, err := reg.FetchIndustryRanksWithFallback(context.Background(), mkt, topN)
	if err != nil {
		return nil, err
	}
	a.industryRanksCache.Set(mkt, ranks)
	go func() {
		if e := a.industryRanksCache.Save(); e != nil {
			slog.Warn("save industry ranks cache", "error", e)
		}
	}()
	return ranks, nil
}
```

运行: `cd app && go test ./... -count=1`

---

## Task 3: Wire GetFundFlow

**文件**: `app_market.go`

**类型**: 需要定义 FundFlow 的缓存类型。检查 `GetFundFlow` 返回 `interface{}`，实际返回 `map[string]interface{}`（包含 daily_flow, minute_flow 等）。

使用 `map[string]interface{}` 作为缓存类型。

1. `App` 结构体增加 `fundFlowCache *market.OffHoursCache[map[string]interface{}]`
2. Init 加载
3. 方法加 `IsTradingHours` 守卫

---

## Task 4: Wire GetDepth

**文件**: `app_market.go`

`GetDepth` 返回 `*market.DepthSnapshot`。缓存 key = symbol。

1. `App` 结构体增加 `depthCache *market.OffHoursCache[*market.DepthSnapshot]`
2. Init 加载
3. 方法加 `IsTradingHours` 守卫

---

## Task 5: Wire GetAbnormalStocks

**文件**: `app_research.go`

`GetAbnormalStocks` 返回 `[]adapters.AbnormalStock`。缓存 key = market (`"SH"`/`"SZ"`)。

1. `App` 结构体加 `abnormalStocksCache *market.OffHoursCache[[]adapters.AbnormalStock]`
2. Init 加载
3. 方法加 `IsTradingHours` 守卫

---

## Task 6-7: Wire GetDragonTiger / GetLimitUpDown

- 找对应方法签名
- 加 Cache 字段 + Init + `IsTradingHours` 守卫

---

## Task 8: Init — 统一注册

在 `app.go` Init 函数收尾:

```go
offDir := filepath.Join(filepath.Dir(a.resolvedDBPath), "offhours")

// IndustryRanks
a.industryRanksCache = market.NewOffHoursCache[[]market.IndustryRank]("industry_ranks")
a.industryRanksCache.SetPath(filepath.Join(offDir, "industry_ranks.json"))
if err := a.industryRanksCache.Load(); err != nil {
    slog.Warn("load industry ranks cache", "error", err)
}

// FundFlow
a.fundFlowCache = market.NewOffHoursCache[map[string]interface{}]("fund_flow")
a.fundFlowCache.SetPath(filepath.Join(offDir, "fund_flow.json"))
if err := a.fundFlowCache.Load(); err != nil {
    slog.Warn("load fund flow cache", "error", err)
}

// Depth
a.depthCache = market.NewOffHoursCache[*market.DepthSnapshot]("depth")
a.depthCache.SetPath(filepath.Join(offDir, "depth.json"))
if err := a.depthCache.Load(); err != nil {
    slog.Warn("load depth cache", "error", err)
}

// AbnormalStocks
a.abnormalStocksCache = market.NewOffHoursCache[[]adapters.AbnormalStock]("abnormal_stocks")
a.abnormalStocksCache.SetPath(filepath.Join(offDir, "abnormal_stocks.json"))
if err := a.abnormalStocksCache.Load(); err != nil {
    slog.Warn("load abnormal stocks cache", "error", err)
}

// DragonTiger
a.dragonTigerCache = market.NewOffHoursCache[[]adapters.DragonTigerRecord]("dragon_tiger")
a.dragonTigerCache.SetPath(filepath.Join(offDir, "dragon_tiger.json"))
if err := a.dragonTigerCache.Load(); err != nil {
    slog.Warn("load dragon tiger cache", "error", err)
}

// LimitUpDown
a.limitUpDownCache = market.NewOffHoursCache[[]adapters.LimitUpDownStock]("limit_up_down")
a.limitUpDownCache.SetPath(filepath.Join(offDir, "limit_up_down.json"))
if err := a.limitUpDownCache.Load(); err != nil {
    slog.Warn("load limit up down cache", "error", err)
}
```

还要增加 `resolveMarket` 辅助方法（把 `"SH"/"SZ"` 映射到 `"CN"`）用于 `IsTradingHours` 检查。

---

## Task 9: Build & Verify

- `cd app && go vet ./...`
- `cd app && go test ./... -count=1`
- `cd frontend && npx vue-tsc --noEmit && npx vitest run`
- `wails3 build`
- 手动验证：运行二进制，检查 `build/data/offhours/` 目录

**注意**: 需要在 Go 测试中添加对 `market.IsTradingHours` 的 mock 或使用特定的测试时间，否则在交易时间外 `IsTradingHours` 返回 false 导致缓存路径无法覆盖。

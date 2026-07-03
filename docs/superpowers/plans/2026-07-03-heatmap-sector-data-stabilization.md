# Heatmap Sector Data Stabilization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix heatmap panel so CN sector data has retry + error propagation, HK/US get their own sector data sources, and the market cache is per-market.

**Architecture:** Move `IndustryRank` to `market` package → define `IndustryRankProvider` interface there → Tencent gets HK sector method, Finnhub gets US sector method → `AdapterRegistry` gets `FetchIndustryRanksWithFallback` → `GetIndustryRanks(mkt, topN)` uses it → `marketOverviewCache` keyed by market.

**Tech Stack:** Go 1.22+, Wails v3, Vue 3 + Pinia

## Global Constraints

- No new external dependencies (use existing adapters: Tencent for HK, Finnhub for US)
- EastMoney push2 for CN with retry only (no viable CN fallback exists)
- `IndustryRank` moves from `adapters` to `market` package (to avoid circular dep)
- Error propagation: no silent swallow — frontend must show meaningful error
- All existing tests must continue to pass

---

### Task 1: Move `IndustryRank` to `market` package + define `IndustryRankProvider`

**Files:**
- Modify: `internal/market/adapter.go` — add `IndustryRank` and `IndustryRankProvider`
- Modify: `internal/market/adapters/eastmoney_signals.go` — remove local `IndustryRank`, use `market.IndustryRank`
- Modify: `app.go` — change import from `adapters.IndustryRank` to `market.IndustryRank`
- Modify: `app_market.go` — import `market.IndustryRank` if needed
- Modify: `internal/research/peer_comparison_service.go` — change import from `adapters.IndustryRank` to `market.IndustryRank`

**Why:** `IndustryRankProvider` needs to be in `market` package so `registry.go` (also in `market`) can reference it without importing `adapters`. The `adapters` package already imports `market` — moving avoids circular dependency.

- [ ] **Step 1: Add `IndustryRank` and `IndustryRankProvider` to `market/adapter.go`**

Append to `internal/market/adapter.go`:

```go
// IndustryRank represents a single industry/sector ranking entry.
type IndustryRank struct {
	Rank      int     `json:"rank"`
	Name      string  `json:"name"`
	Code      string  `json:"code"`
	ChangePct float64 `json:"change_pct"`
	UpCount   int     `json:"up_count"`
	DownCount int     `json:"down_count"`
	Leader    string  `json:"leader"`
	LeaderChg float64 `json:"leader_change"`
}

// IndustryRankProvider is an optional interface for adapters that can provide
// industry/sector ranking data (涨跌板块排名).
type IndustryRankProvider interface {
	// FetchIndustryRanks returns top N industry/sector rankings by change percent.
	// market is "CN", "HK", or "US".
	FetchIndustryRanks(ctx context.Context, market string, topN int) ([]IndustryRank, error)
}
```

- [ ] **Step 2: Remove local `IndustryRank` from `eastmoney_signals.go`**

In `internal/market/adapters/eastmoney_signals.go`:
- Delete lines 42-52 (the `IndustryRank` struct definition)
- Change `FetchIndustryRanks` return type from `[]IndustryRank` to `[]market.IndustryRank`
- Change the local var `ranks := make([]IndustryRank, ...)` to `ranks := make([]market.IndustryRank, ...)`
- Change `ranks = append(ranks, IndustryRank{...})` to `ranks = append(ranks, market.IndustryRank{...})`
- Change the `IsAvailable` call to match

Result:

```go
func (a *EastMoneySignalsAdapter) FetchIndustryRanks(ctx context.Context, topN int) ([]market.IndustryRank, error) {
	// ... same implementation ...
	ranks := make([]market.IndustryRank, 0, len(result.Data.Diff))
	for _, item := range result.Data.Diff {
		ranks = append(ranks, market.IndustryRank{
			Rank:      len(ranks) + 1,
			Name:      item.Name,
			Code:      item.Code,
			ChangePct: item.ChangePct,
			UpCount:   item.UpCount,
			DownCount: item.DownCount,
			Leader:    item.Leader,
			LeaderChg: item.LeaderChg,
		})
	}
	return ranks, nil
}
```

Also update `IsAvailable` to use the proper import.

- [ ] **Step 3: Update `app.go`**

Change `adapters.IndustryRank` to `market.IndustryRank` in `GetIndustryRanks`:
```go
func (a *App) GetIndustryRanks(topN int) ([]market.IndustryRank, error) {
```

- [ ] **Step 4: Update `peer_comparison_service.go`**

Change `adapters.IndustryRank` to `market.IndustryRank` in return type:
```go
func (s *PeerComparisonService) GetIndustryRanks(ctx context.Context, topN int) ([]market.IndustryRank, error) {
```

- [ ] **Step 5: Build to verify**

Run: `cd app && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/market/adapter.go internal/market/adapters/eastmoney_signals.go app.go app_market.go internal/research/peer_comparison_service.go
git commit -m "refactor: move IndustryRank to market package, add IndustryRankProvider interface"
```

---

### Task 2: Add HK sector ranking to TencentAdapter

**Files:**
- Modify: `internal/market/adapters/tencent.go`
- Test: `internal/market/adapters/tencent_test.go` (create if not exists)

**Interfaces:**
- Consumes: `market.IndustryRankProvider` interface, `market.IndustryRank` struct
- Produces: `TencentAdapter.FetchIndustryRanks` for HK market

- [ ] **Step 1: Write the test**

In `internal/market/adapters/tencent_test.go`:

```go
package adapters

import (
	"context"
	"testing"
)

func TestTencentAdapter_FetchIndustryRanks(t *testing.T) {
	adapter := NewTencentAdapter()
	if !adapter.IsAvailable(context.Background()) {
		t.Skip("tencent adapter not available (network)")
	}

	ranks, err := adapter.FetchIndustryRanks(context.Background(), "HK", 30)
	if err != nil {
		t.Fatalf("FetchIndustryRanks failed: %v", err)
	}
	if len(ranks) == 0 {
		t.Fatal("expected non-empty industry ranks")
	}
	t.Logf("fetched %d HK industry ranks", len(ranks))
	for i, r := range ranks {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s: %.2f%%", r.Rank, r.Name, r.ChangePct)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/market/adapters/ -run TestTencentAdapter_FetchIndustryRanks -v`
Expected: FAIL — FetchIndustryRanks not defined

- [ ] **Step 3: Implement `FetchIndustryRanks` on TencentAdapter**

Add to `internal/market/adapters/tencent.go` (at the end, before package-closing line):

```go
const tencentHKRankingURL = "http://web.ifzq.gtimg.cn/appstock/app/HK/hkranking"

// tencentHKIndustryResp represents the Tencent HK industry ranking response.
type tencentHKIndustryResp struct {
	Data []struct {
		Name      string  `json:"name"`
		ChangePct float64 `json:"change_pct"`
		UpCount   int     `json:"up_count"`
		DownCount int     `json:"down_count"`
		Leader    string  `json:"leader"`
	} `json:"data"`
}

// FetchIndustryRanks returns HK industry rankings via Tencent Finance.
// Only supports market="HK"; for other markets returns empty slice.
func (a *TencentAdapter) FetchIndustryRanks(ctx context.Context, market string, topN int) ([]market.IndustryRank, error) {
	if market != "HK" {
		return []market.IndustryRank{}, nil
	}
	if topN <= 0 {
		topN = 20
	}

	u := fmt.Sprintf("%s?type=industry", tencentHKRankingURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("tencent: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tencent: fetch HK ranking: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent: HK ranking status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tencent: read body: %w", err)
	}

	var apiResp tencentHKIndustryResp
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("tencent: parse HK ranking: %w", err)
	}

	ranks := make([]market.IndustryRank, 0, min(topN, len(apiResp.Data)))
	for i, item := range apiResp.Data {
		if i >= topN {
			break
		}
		ranks = append(ranks, market.IndustryRank{
			Rank:      i + 1,
			Name:      item.Name,
			ChangePct: item.ChangePct,
			UpCount:   item.UpCount,
			DownCount: item.DownCount,
			Leader:    item.Leader,
		})
	}
	return ranks, nil
}
```

Also add `"quantflow/internal/market"` to the imports if not already present (check existing imports first — tencent.go likely doesn't import market).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/market/adapters/ -run TestTencentAdapter_FetchIndustryRanks -v`
Expected: PASS (or SKIP if network unavailable)

- [ ] **Step 5: Commit**

```bash
git add internal/market/adapters/tencent.go internal/market/adapters/tencent_test.go
git commit -m "feat: add HK industry ranking via Tencent Finance adapter"
```

---

### Task 3: Add US sector ranking to FinnhubAdapter

**Files:**
- Modify: `internal/market/adapters/finnhub.go`
- Test: `internal/market/adapters/finnhub_test.go`

**Interfaces:**
- Consumes: `market.IndustryRankProvider` interface, `market.IndustryRank` struct
- Produces: `FinnhubAdapter.FetchIndustryRanks` for US market

- [ ] **Step 1: Write the test**

In `internal/market/adapters/finnhub_test.go`:

```go
package adapters

import (
	"context"
	"testing"
)

func TestFinnhubAdapter_FetchIndustryRanks(t *testing.T) {
	adapter := NewFinnhubAdapter()
	if !adapter.IsAvailable(context.Background()) {
		t.Skip("finnhub not available (no API key or network)")
	}

	ranks, err := adapter.FetchIndustryRanks(context.Background(), "US", 30)
	if err != nil {
		t.Fatalf("FetchIndustryRanks failed: %v", err)
	}
	if len(ranks) == 0 {
		t.Fatal("expected non-empty industry ranks")
	}
	t.Logf("fetched %d US industry ranks", len(ranks))
	for i, r := range ranks {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s: %.2f%%", r.Rank, r.Name, r.ChangePct)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/market/adapters/ -run TestFinnhubAdapter_FetchIndustryRanks -v`
Expected: FAIL — FetchIndustryRanks not defined

- [ ] **Step 3: Implement `FetchIndustryRanks` on FinnhubAdapter**

Add to `internal/market/adapters/finnhub.go`:

```go
// finnhubSectorResp represents Finnhub's sector performance response.
type finnhubSectorResp struct {
	Data []struct {
		Name      string  `json:"name"`
		ChangePct float64 `json:"changePct"`
	} `json:"sector"`
}

// FetchIndustryRanks returns US sector rankings via Finnhub.
// Only supports market="US"; for other markets returns empty slice.
func (a *FinnhubAdapter) FetchIndustryRanks(ctx context.Context, market string, topN int) ([]market.IndustryRank, error) {
	if market != "US" {
		return []market.IndustryRank{}, nil
	}
	if a.apiKey == "" {
		return nil, fmt.Errorf("finnhub: API key not set")
	}
	if topN <= 0 {
		topN = 20
	}

	u := fmt.Sprintf("%s/sector?token=%s", finnhubBaseURL, a.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("finnhub: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("finnhub: fetch sector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("finnhub: sector status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("finnhub: read body: %w", err)
	}

	var apiResp finnhubSectorResp
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("finnhub: parse sector: %w", err)
	}

	ranks := make([]market.IndustryRank, 0, min(topN, len(apiResp.Data)))
	for i, item := range apiResp.Data {
		if i >= topN {
			break
		}
		ranks = append(ranks, market.IndustryRank{
			Rank:      i + 1,
			Name:      item.Name,
			ChangePct: item.ChangePct,
		})
	}
	return ranks, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/market/adapters/ -run TestFinnhubAdapter_FetchIndustryRanks -v`
Expected: PASS (or SKIP if no API key)

- [ ] **Step 5: Commit**

```bash
git add internal/market/adapters/finnhub.go internal/market/adapters/finnhub_test.go
git commit -m "feat: add US sector ranking via Finnhub adapter"
```

---

### Task 4: Add `FetchIndustryRanksWithFallback` to registry

**Files:**
- Modify: `internal/market/registry.go`
- Modify: `internal/market/registry_test.go`

**Interfaces:**
- Consumes: `market.IndustryRankProvider` interface (defined in `adapter.go`)
- Produces: `AdapterRegistry.FetchIndustryRanksWithFallback` (consumed by Task 5 `app.go`)

- [ ] **Step 1: Write failing test**

In `internal/market/registry_test.go`:

```go
package market

import (
	"context"
	"fmt"
	"testing"
)

// mockIndustryRankProvider implements IndustryRankProvider for testing.
type mockIndustryRankProvider struct {
	name  string
	fail  bool
	ranks []IndustryRank
}

func (m *mockIndustryRankProvider) Name() string           { return m.name }
func (m *mockIndustryRankProvider) IsAvailable(ctx context.Context) bool { return true }
func (m *mockIndustryRankProvider) FetchIndustryRanks(ctx context.Context, market string, topN int) ([]IndustryRank, error) {
	if m.fail {
		return nil, fmt.Errorf("mock: %s failed", m.name)
	}
	if m.ranks != nil {
		return m.ranks, nil
	}
	return []IndustryRank{{Rank: 1, Name: "Mock", ChangePct: 1.0}}, nil
}

func TestFetchIndustryRanksWithFallback(t *testing.T) {
	reg := NewAdapterRegistry()

	// Register mock that fails for unknown market
	reg.Register(&mockIndustryRankProvider{name: "test"})

	ranks, err := reg.FetchIndustryRanksWithFallback(context.Background(), "ZZ", 10)
	if err == nil {
		t.Fatal("expected error for unknown market")
	}
	if ranks != nil {
		t.Fatal("expected nil ranks for unknown market")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd app && go test ./internal/market/ -run TestFetchIndustryRanksWithFallback -v`
Expected: FAIL — FetchIndustryRanksWithFallback not defined

- [ ] **Step 3: Implement `FetchIndustryRanksWithFallback`**

Add to `internal/market/registry.go` after the `FetchOHLCVWithFallback` method:

```go
// IndustryRankChains defines the priority-ordered list of adapter names for
// industry/sector ranking data for each market.
var IndustryRankChains = map[string][]string{
	"CN": {"eastmoney_signals"},
	"HK": {"tencent"},
	"US": {"finnhub"},
}

// FetchIndustryRanksWithFallback tries each adapter in the market's industry
// rank chain until one succeeds. Returns the ranked list and any error.
func (r *AdapterRegistry) FetchIndustryRanksWithFallback(ctx context.Context, market string, topN int) ([]IndustryRank, error) {
	chain, ok := IndustryRankChains[market]
	if !ok {
		return nil, fmt.Errorf("no industry rank chain for market %q", market)
	}

	var lastErr error
	for _, name := range chain {
		adapter := r.Get(name)
		if adapter == nil {
			slog.Debug("adapter not registered, skipping", "name", name, "market", market)
			continue
		}
		provider, ok := adapter.(IndustryRankProvider)
		if !ok {
			slog.Debug("adapter does not implement IndustryRankProvider", "name", name)
			continue
		}
		if !adapter.IsAvailable(ctx) {
			slog.Debug("adapter unavailable, skipping", "name", name, "market", market)
			continue
		}

		ranks, err := RetryWithBudget(
			func() ([]IndustryRank, error) { return provider.FetchIndustryRanks(ctx, market, topN) },
			DefaultRetryConfig(name),
		)
		if err != nil {
			slog.Warn("industry rank fetch failed, trying next", "name", name, "market", market, "error", err)
			lastErr = err
			continue
		}

		if len(ranks) > 0 {
			return ranks, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all industry rank adapters failed for market %q: %w", market, lastErr)
	}
	return nil, fmt.Errorf("no available industry rank adapter for market %q (chain: %v)", market, chain)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd app && go test ./internal/market/ -run TestFetchIndustryRanksWithFallback -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/market/registry.go internal/market/registry_test.go
git commit -m "feat: add FetchIndustryRanksWithFallback to AdapterRegistry"
```

---

### Task 5: Update `marketOverviewCache` to be per-market

**Files:**
- Modify: `app_market.go`

- [ ] **Step 1: Rewrite `marketOverviewCache` struct**

In `app_market.go`, replace (lines 21-89):

```go
type marketOverviewCache struct {
	mu      sync.Mutex
	data    map[string]interface{}
	expires time.Time
}

var overviewCache = &marketOverviewCache{}

func (c *marketOverviewCache) get(mkt string) (map[string]interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil && time.Now().Before(c.expires) {
		return c.data, true
	}
	return nil, false
}

func (c *marketOverviewCache) set(data map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.expires = time.Now().Add(60 * time.Second)
}
```

With:

```go
type marketOverviewCache struct {
	mu      sync.Mutex
	entries map[string]*marketCacheEntry
}

type marketCacheEntry struct {
	data    map[string]interface{}
	expires time.Time
}

var overviewCache = &marketOverviewCache{
	entries: make(map[string]*marketCacheEntry),
}

func (c *marketOverviewCache) get(mkt string) (map[string]interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[mkt]
	if !ok || time.Now().After(entry.expires) {
		if ok {
			delete(c.entries, mkt)
		}
		return nil, false
	}
	return entry.data, true
}

func (c *marketOverviewCache) set(mkt string, data map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[mkt] = &marketCacheEntry{
		data:    data,
		expires: time.Now().Add(60 * time.Second),
	}
}
```

- [ ] **Step 2: Update the `set` call site**

In `GetMarketOverview` (around line 474), change:
```go
overviewCache.set(out)
```
To:
```go
overviewCache.set(mkt, out)
```

- [ ] **Step 3: Build to verify**

Run: `cd app && go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add app_market.go
git commit -m "fix: marketOverviewCache now stores per-market entries"
```

---

### Task 6: Update `GetIndustryRanks` with market parameter + fallback chain

**Files:**
- Modify: `app.go`

- [ ] **Step 1: Update `GetIndustryRanks` signature and implementation**

Replace `app.go` lines 820-835 with:

```go
// GetIndustryRanks returns industry ranking by change percent for a given market.
// Uses per-market fallback chains: CN→eastmoney_signals, HK→tencent, US→finnhub.
func (a *App) GetIndustryRanks(mkt string, topN int) ([]market.IndustryRank, error) {
	if topN <= 0 {
		topN = 20
	}
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	return reg.FetchIndustryRanksWithFallback(context.Background(), mkt, topN)
}
```

- [ ] **Step 2: Build to verify**

Run: `cd app && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add app.go
git commit -m "feat: GetIndustryRanks now accepts market parameter with per-market fallback"
```

---

### Task 7: Update frontend data store to pass market to `GetIndustryRanks`

**Files:**
- Modify: `frontend/src/stores/data.ts`

- [ ] **Step 1: Update `fetchMarketOverview` call**

In `data.ts` line 135, change:
```ts
return await app.GetIndustryRanks(30)
```
To:
```ts
return await app.GetIndustryRanks(market, 30)
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/data.ts
git commit -m "feat: pass market to GetIndustryRanks in frontend data store"
```

---

### Task 8: Update HeatmapPanel for market-aware empty states

**Files:**
- Modify: `frontend/src/terminal/panels/HeatmapPanel.vue`
- Modify: `frontend/src/lib/i18n/zh.ts`
- Modify: `frontend/src/lib/i18n/en.ts`

- [ ] **Step 1: Add i18n keys**

In `frontend/src/lib/i18n/zh.ts` add under the misc section:
```ts
no_hk_sector_data: '港股板块数据暂不可用',
no_us_sector_data: '美股板块数据暂不可用',
```

In `frontend/src/lib/i18n/en.ts` add under the misc section:
```ts
no_hk_sector_data: 'HK sector data unavailable',
no_us_sector_data: 'US sector data unavailable',
```

- [ ] **Step 2: Update empty state in HeatmapPanel**

Replace the single empty state div in `HeatmapPanel.vue`:

```vue
<div v-else class="empty-state">
  {{ activeMarket === 'HK' ? $t('misc.no_hk_sector_data') :
     activeMarket === 'US' ? $t('misc.no_us_sector_data') :
     $t('misc.no_sector_data') }}
</div>
```

- [ ] **Step 3: Verify TypeScript compilation**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/HeatmapPanel.vue frontend/src/lib/i18n/zh.ts frontend/src/lib/i18n/en.ts
git commit -m "feat: market-aware empty states for heatmap panel"
```

---

### Task 9: Run full check and update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Run backend tests**

Run: `cd app && go vet ./... && go test ./... -v -count=1 2>&1 | tail -30`
Expected: All tests pass

- [ ] **Step 2: Run frontend checks**

Run: `cd frontend && npx vue-tsc --noEmit && npx vitest run 2>&1 | tail -20`
Expected: All checks pass

- [ ] **Step 3: Update CHANGELOG.md**

Add under today's date:

```markdown
## [2026.7.3] - 2026-07-03

### Fixed
- [Terminal] Heatmap sector data: CN EastMoney push2 now retries 3x with error propagation instead of silent empty
- [Terminal] Heatmap market cache: fixed per-market isolation bug (was returning CN data for HK/US)
- [MarketData] GetIndustryRanks now accepts market parameter; HK uses Tencent, US uses Finnhub

### Added
- [MarketData] HK sector ranking via Tencent Finance adapter (web.ifzq.gtimg.cn)
- [MarketData] US sector ranking via Finnhub adapter (/v1/sector endpoint)
- [MarketData] IndustryRankProvider interface for future sector data adapters
- [MarketData] FetchIndustryRanksWithFallback in AdapterRegistry with per-market chains
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for heatmap sector data stabilization"
```

# 港股市场概况数据 Implementation Plan

> Spec: [2026-07-09-hk-market-overview.md](../../specs/2026-07-09-hk-market-overview.md)

**Goal:** MarketOverview HK 标签页完整可用——指数报价、分时图、K线图、行业排行

---

### Task 1: 指数代码统一 + 报价修复

**Files:**
- Modify: `app_market.go`

- [ ] **Step 1: 修改 idxDef**

将 HK 指数代码从 Yahoo 格式改为标准格式：

```go
// Before:
case "HK":
    indices = []idxDef{
        {"^HSI", "恒生指数"},
        {"^HSCE", "国企指数"},
        {"^HSTECH", "恒生科技"},
    }

// After:
case "HK":
    indices = []idxDef{
        {"HSI.HK", "恒生指数"},
        {"HSCEI.HK", "国企指数"},
        {"HSTECH.HK", "恒生科技"},
    }
```

- [ ] **Step 2: Verify Go compiles**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add app_market.go
git commit -m "fix(market): use standard .HK suffix for HK index codes in GetMarketOverview"
```

---

### Task 2: 分时图支持

**Files:**
- Modify: `app_market.go` — 去除非 CN 硬限制
- Modify: `python/src/data/fetcher.py` — 港股指数走 Tencent

- [ ] **Step 1: Go 侧去除非 CN 限制**

```go
// Before:
mkt := market.MarketForSymbol(symbol)
if mkt != "CN" {
    return nil, "unavailable", fmt.Errorf("minute data not available for market %s", mkt)
}

// After: 对所有市场尝试获取分时数据，港股走 Tencent
mkt := market.MarketForSymbol(symbol)
_ = mkt // 保留但不硬限制
```

- [ ] **Step 2: Python 侧加港股指数识别**

在 `_is_index_code` 中增加港股指数前缀 `HSI`、`HSCEI`、`HSTECH`：

```python
def _is_index_code(symbol: str) -> bool:
    s = symbol.strip().upper()
    for suffix in (".SH", ".SS", ".SZ", ".BJ", ".HK"):
        s = s.replace(suffix, "")
    # CN indices
    if len(s) == 6 and s.isdigit() and s[:3] in ("000", "399", "688"):
        return True
    # HK indices
    if s in ("HSI", "HSCEI", "HSTECH"):
        return True
    return False
```

- [ ] **Step 3: Python 侧加 Tencent 港股分时**

在 `_fetch_mootdx_minute` 中，让 `.HK` 后缀也走 Tencent：

```python
# Already works: _to_tencent_code("HSI.HK") → "hkHSI"
# Already works: _fetch_tencent_index_minute("hkHSI") → Tencent API
```

`_to_tencent_code` 已处理 `.HK` 后缀，无需修改。`_fetch_tencent_index_minute` 传入 `hkHSI` 即可。

- [ ] **Step 4: Verify Python syntax**

```bash
python -m py_compile python/src/data/fetcher.py
```

- [ ] **Step 5: Commit**

---

### Task 3: K 线图修复

**Files:**
- 可能需修改: Yahoo 适配器（如果 `HSI.HK` → `^HSI` 映射缺失）

- [ ] **Step 1: 测试现有 fallback 链**

`FetchOHLCVWithFallback(ctx, "HK", "HSI.HK", "1D", "", start, end)`:
- Yahoo: 需要 `^HSI` 格式 → 可能失败
- Tencent: `toTencentCode("HSI.HK")` → `hkHSI` → 可能成功

- [ ] **Step 2: 如果 Yahoo 失败，增加格式兼容**

在 `internal/market/adapters/` 中增加港股指数 Yahoo 格式映射：

```go
// 如果 Yahoo adapter 不支持 HSI.HK 格式，映射到 ^HSI
func normalizeHKSymbol(symbol string) string {
    hkIndexMap := map[string]string{
        "HSI.HK":    "^HSI",
        "HSCEI.HK":  "^HSCE",
        "HSTECH.HK": "^HSTECH",
    }
    if mapped, ok := hkIndexMap[symbol]; ok {
        return mapped
    }
    return symbol
}
```

- [ ] **Step 3: Commit**

---

### Task 4: 前端适配

**Files:**
- 可能需修改: `frontend/src/terminal/panels/MarketOverviewPanel.vue`

- [ ] **Step 1: 确认 HK 行业排行数据流**

`dataStore.fetchMarketOverview("HK")` → `app.GetIndustryRanks("HK", 30)` → Tencent → 前端 `sectors` → 行业排行。

- [ ] **Step 2: 确认前端 HK 切换逻辑**

`switchMarket("HK")` → `refresh()` → `fetchMarketOverview("HK")` + `loadChart()`

- [ ] **Step 3: 测试并 commit**

---

### Task 5: CHANGELOG

- [ ] Add under `[2026.7.9]` → `### Added`:

```markdown
- [Market] HK index quotes now use standard .HK suffix (HSI.HK, HSCEI.HK, HSTECH.HK)
- [Market] GetMinuteLine no longer blocks non-CN markets; HK indices go through Tencent minute API
- [Frontend] MarketOverview HK tab now shows index cards, minute chart, K-line chart, and sector rankings
```

- [ ] Commit

---

## Revision Notes

- 涨跌家数/情绪条对港股暂时隐藏（`breadthTotal > 1` 守卫自动处理）
- 行业排行已有 Tencent 适配器支持，无需额外开发
- HK 分时数据走 Tencent API（和 A 股指数同一套 fallback 代码）

# 港股市场概况数据 Spec

## Motivation

MarketOverviewPanel 的 HK 标签页目前为空——切换到港股后无指数行情、无分时图、无 K 线图、无涨跌家数。需要补齐港股市场概况的完整数据链路。

## 当前状态 vs 目标

| 区域 | 当前 | 目标 |
|------|------|------|
| 指数卡片 | 无数据 | 恒生指数、国企指数、恒生科技 实时报价 |
| 分时图 | "market closed" 错误 | Tencent API 获取港股指数分时 |
| K 线图 | 无数据 | Yahoo/Tencent 日 K |
| 涨跌家数/情绪 | 无数据 | 港股市场广度（可选，先跳过） |
| 行业排行 | 无数据 | Tencent `FetchIndustryRanks` (已有) |
| 市场切换 | 空页面 | HK 标签页完整可用 |

### 港股三大指数

| 名称 | Yahoo 符号 | Tencent 代码 | 说明 |
|------|-----------|-------------|------|
| 恒生指数 | `^HSI` | `hkHSI` | 港股基准指数 |
| 国企指数 | `^HSCE` | `hkHSCEI` | H 股指数 |
| 恒生科技 | `^HSTECH` | `hkHSTECH` | 科技板块指数 |

## Design

### 1. 指数代码适配

**问题**：Go 侧 `idxDef` 使用 Yahoo 格式 (`^HSI`)，但 Tencent 适配器用 `toTencentCode()` 只处理 `.HK` 后缀。Yahoo 格式 `^HSI` 到适配器的转换链中会失败。

**方案**：`idxDef` 改用 Tencent 格式（`hkHSI` 等），Yahoo 适配器内部做兼容映射；或者在 `toTencentCode` 中增加 Yahoo 格式 → Tencent 格式的映射表。

```go
// 推荐方案：idxDef 直接用 Tencent 格式
case "HK":
    indices = []idxDef{
        {"hkHSI", "恒生指数"},
        {"hkHSCEI", "国企指数"},
        {"hkHSTECH", "恒生科技"},
    }
```

Yahoo 适配器的 `FetchQuote`/`FetchOHLCV` 增加 Yahoo 格式兼容（`^HSI` ↔ `hkHSI` 映射）——或者直接将 Tencent 格式传给 Yahoo（Yahoo 也接受 `0700.HK` 这样的格式）。

**实际方案**：用 `00700.HK` 格式（标准港股代码格式），所有适配器都支持。

指数代码映射：
| 名称 | 标准代码 | 说明 |
|------|---------|------|
| 恒生指数 | `HSI` 或 `^HSI` | Yahoo 接受 `^HSI` |
| 国企指数 | `HSCE` 或 `^HSCE` | |
| 恒生科技 | `HSTECH` 或 `^HSTECH` | |

**最终决定**：使用 `HSI.HK` / `HSCE.HK` / `HSTECH.HK` 格式（统一后缀规范），Yahoo fallback 链自动处理。

### 2. 实时报价

**现有能力**：`FetchQuoteWithFallback(ctx, "HK", symbol)` 走 yahoo→tencent→sina→akshare 链。

对于 `HSI.HK`：
- Yahoo: API 接受 `^HSI` 格式，需要映射
- Tencent: `toTencentCode("HSI.HK")` → `hkHSI` ✓（后缀 `.HK` 被识别）

**修改**：
- `toTencentCode` 已处理 `.HK` 后缀（`hkHSI`），无需修改
- Yahoo 适配器需增加 `^` 前缀兼容（`HSI.HK` → `^HSI`）

或者直接：
- Yahoo 适配器新增 `normalizeHKSymbol()` 函数：`HSI.HK` → `^HSI`

### 3. 分时图

**问题**：`GetMinuteLine` 只支持 CN 市场（`MarketForSymbol(symbol) != "CN"` 时直接返回错误）。

**方案**：对港股指数，用 Tencent 分时 API（和 A 股指数相同模式）。

Tencent 港股分时 API：
```
http://ifzq.gtimg.cn/appstock/app/minute/query?_var=min_data&code=hkHSI
```

在 Python `_fetch_mootdx_minute` 中增加港股指数判断（通过 `.HK` 后缀），走 Tencent fallback。

**Go 侧修改**：
- `GetMinuteLine` 去除 `mkt != "CN"` 的硬限制
- 对非 CN 市场，直接走 Python sidecar 的 Tencent fallback 路径
- 或新建 `fetchHKMinuteLine` 函数

### 4. K 线图

**现有能力**：`FetchOHLCVWithFallback(ctx, "HK", symbol, ...)` 走 yahoo→tencent 链。

- Yahoo: `FetchOHLCV("HK", "^HSI", ...)` — 需要符号格式兼容
- Tencent: `toTencentCode("HSI.HK")` → `hkHSI` — 需要确认 Tencent K 线支持港股指数

**方案**：保持现有 fallback 链，修复符号格式兼容即可。

### 5. 行业排行

**已有**：Tencent 适配器 `FetchIndustryRanks("HK", topN)` → `tencentHKRankingURL`

Go 侧 `GetIndustryRanks("HK", 30)` 已实现。前端 `data.ts` 的 `fetchMarketOverview` 也传 `activeMarket`，自动切换市场。

**无需修改**。

### 6. 涨跌家数/情绪

**跳过低优先级**：港股涨跌家数数据源不明确（Tencent/Sina 无直接接口）。先用 `v-if="breadthTotal > 1"` 守卫隐藏该区域。后续可用 HKEX 官方数据或 Wind/BBG 补充。

## Implementation Tasks

### Task 1：指数代码统一 + 报价修复

**Files:**
- Modify: `app_market.go` — idxDef 改为 `HSI.HK` 等标准格式
- Modify: `internal/market/adapters/adapters.go` (or yahoo.go) — 增加 `^HSI` 兼容
- Expected: HK 指数卡片显示实时报价

### Task 2：分时图支持

**Files:**
- Modify: `app_market.go` — `GetMinuteLine` 去除非 CN 硬限制，增加 Tencent 港股分时路径
- Modify: `python/src/data/fetcher.py` — `_fetch_mootdx_minute` 支持 `.HK` 后缀走 Tencent fallback
- Expected: 港股指数分时图可用

### Task 3：K 线图修复

**Files:**
- Modify: Yahoo 适配器符号兼容（如有需要）
- Expected: 港股指数日 K 线图可用

### Task 4：CHANGELOG

## Acceptance Criteria

- [ ] 切换到 HK 标签页，恒生指数/国企指数/恒生科技 三张卡片显示实时报价
- [ ] 点击指数卡片，分时图正常显示（Tencent API 数据）
- [ ] 日 K 线图正常显示（Yahoo/Tencent 数据）
- [ ] 行业排行显示港股板块涨跌
- [ ] 涨跌家数区域隐藏（港股暂无数据源）
- [ ] CN/HK/US 切换正常，数据不混淆

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| Yahoo API 对港股指数有限制 | Fallback 到 Tencent（确认支持指数 K 线） |
| Tencent 分时 API 港股数据时效性 | 与 A 股相同的 5s 刷新间隔 |
| 港股交易时间与 A 股不同（9:30-16:00） | `IsTradingHours` 已支持多市场 |
| `^HSI` 格式未被所有适配器识别 | 统一改用 `HSI.HK` 格式 |

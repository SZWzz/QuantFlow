# Multi-Market Full Support — A股 / 港股 / 美股 / 加密

> 父 spec: 无（独立专项）
> 前序: 调研报告（后端适配器链 + 前端面板覆盖矩阵）
> 日期: 2026-06-26

## Motivation

项目声称支持 A 股 / 港股 / 美股 / 加密四大市场，但实际覆盖严重不均：

**后端 P0 缺陷：**
- `GetMarketSnapshot`、`GetCorrelationMatrix`、`GetReturnDistribution`、`GetVolatilitySurface` 4 个 API 硬编码 `"CN"`，传 `AAPL` 也会走 CN 链
- `gateio` 适配器已注册但**不在** `FallbackChains["CRYPTO"]` 中，国内用户加密数据全部不可用
- `GetMinuteLine` 仅支持 CN（依赖 mootdx）
- Polygon 适配器是完全的 stub（`"not implemented"`）

**前端面板缺陷：**
- `TickerTapePanel` 硬编码 8 个 A 股代码，港股/美股用户看不到跑马灯
- `MarketOverviewPanel` / `HeatmapPanel` 仅返回 A 股指数和行业数据
- `OrderEntryPanel` 市场参数写死 `'CN'`，下单 AAPL 也走 CN 链
- `PositionDetail` 全是 mock 数据
- `CandlestickPanel` 交易时段写死 A 股时间（09:30-11:30/13:00-15:00 北京时间）

## Design

### Scope

按优先级分为三阶段：

**Phase A（P0 修复 — 后端正确性）：**
1. `GetMarketSnapshot` 等 4 个 API 使用 `MarketForSymbol()` 动态路由
2. `gateio` 加入 CRYPTO 回退链
3. Polygon 适配器实现（去掉 stub）或从 US 链移除
4. `OrderEntryPanel` 使用 `detectMarket()` 而非硬编码 `'CN'`
5. `GetMinuteLine` 扩展为按市场路由

**Phase B（P1 补齐 — 前端 HK/US 面板）：**
6. `TickerTapePanel` 按市场分组显示，支持 HK/US/CRYPTO
7. `MarketOverviewPanel` 支持切换市场（CN/HK/US）
8. `HeatmapPanel` 支持多市场
9. `PositionDetail` 接入真实数据（复用 portfolio store）
10. `CandlestickPanel` 交易时段按市场动态计算

**Phase C（P2 增强 — 体验完善）：**
11. 前端市场选择器 UI（Settings 或 Header 下拉）
12. `SymbolSearch` 搜索结果增加市场标签筛选
13. 多市场颜色方案自动切换（CN 红涨绿跌 / US 绿涨红跌）

### 详细设计

#### Phase A — P0 修复

##### A1. 4 个 Go API 使用动态市场路由

```go
// GetMarketSnapshot — 原代码硬编码 "CN"
// 改为使用 MarketForSymbol 逐 symbol 判断
func (a *App) GetMarketSnapshot(ctx context.Context, symbols []string) ([]map[string]interface{}, error) {
    reg := a.getMarketReg()
    result := make([]map[string]interface{}, 0, len(symbols))
    for _, sym := range symbols {
        market := market.MarketForSymbol(sym)
        snap, _, err := reg.FetchQuoteWithFallback(ctx, market, sym)
        if err != nil { continue }
        result = append(result, /* ... */)
    }
    return result, nil
}
```

`GetCorrelationMatrix` / `GetReturnDistribution` / `GetVolatilitySurface` 同理，每个 symbol 独立推断市场。

##### A2. gateio 加入 CRYPTO 回退链

```go
// registry.go
var FallbackChains = map[string][]string{
    "CRYPTO": {"binance", "okx", "coingecko", "gateio"},
}
```

##### A3. Polygon 适配器

选项：实现简化版 REST 调用，或从 US 链移除。鉴于 Polygon 需要 API key 且免费额度有限，选择**将其从 US 链末尾移除**，US 链变为 `yahoo → sina → finnhub`。

##### A4. OrderEntryPanel 修复

```typescript
// 原代码: const snap = await GetQuote('CN', symbol.value)
// 改为:
const market = detectMarket(symbol.value)
const snap = await GetQuote(market, symbol.value)
```

##### A5. GetMinuteLine 多市场路由

```go
func (a *App) GetMinuteLine(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
    market := market.MarketForSymbol(symbol)
    switch market {
    case "CN":
        return a.getMinuteLineCN(ctx, symbol)
    case "US":
        return a.getMinuteLineUS(ctx, symbol) // via yahoo 1m OHLCV
    case "HK":
        return a.getMinuteLineHK(ctx, symbol) // via tencent
    default:
        return nil, fmt.Errorf("minute data not supported for %s", market)
    }
}
```

#### Phase B — 前端面板补齐

##### B6. TickerTapePanel 多市场

```typescript
// 从硬编码 8 个 CN 代码改为按市场分组默认列表
const defaultSymbols = {
    CN:  ['600519', '000001', '300750', '601318'],
    HK:  ['00700.HK', '09988.HK', '00388.HK', '00941.HK'],
    US:  ['AAPL', 'MSFT', 'NVDA', 'TSLA'],
}
// 支持通过 props 或设置切换市场分组
```

##### B7. MarketOverviewPanel 多市场

新增 API `GetMarketOverview(market string)` 或现有 API 增加 market 参数。CN 返回沪/深指数，HK 返回恒指/国企指数，US 返回 SPX/IXIC/DJI。

##### B8. CandlestickPanel 动态交易时段

```typescript
function isTradingHours(market: string): boolean {
    const now = new Date()
    const hk = now.getUTCHours() * 60 + now.getUTCMinutes()
    switch (market) {
        case 'CN':  // UTC+8, 09:30-11:30, 13:00-15:00
            return (hk+480 >= 570 && hk+480 < 690) || (hk+480 >= 780 && hk+480 < 900)
        case 'HK':  // UTC+8, 09:30-16:00
            return (hk+480 >= 570 && hk+480 < 960)
        case 'US':  // UTC-5/UTC-4, 09:30-16:00 ET
            const et = hk - 300 // EDT = UTC-4
            return (et >= 570 && et < 960)
        default: return false
    }
}
```

### 数据流

```
用户输入 symbol (如 "AAPL" / "00700.HK" / "600519")
  → 前端 detectMarket(symbol) → market string ("US" / "HK" / "CN")
  → 后端 MarketForSymbol(symbol) → 同样逻辑
  → 适配器链根据 market 选择回退链
  → 返回统一格式的 QuoteSnapshot / OHLCVBar[]
  → 前端按市场动态渲染（颜色方案、交易时段、小数位数）
```

### 文件变更清单

| 文件 | Phase | 改动 |
|------|-------|------|
| `internal/market/registry.go` | A | `gateio` 加入 CRYPTO 回退链；Polygon 从 US 链移除 |
| `internal/market/adapters/polygon.go` | A | 标记为 deprecated 或实现简化版 |
| `app.go` (GetMarketSnapshot) | A | `"CN"` → `MarketForSymbol(sym)` |
| `app.go` (GetCorrelationMatrix) | A | `"CN"` → `MarketForSymbol(sym)` 逐 symbol |
| `app.go` (GetReturnDistribution) | A | `"CN"` → `MarketForSymbol(sym)` |
| `app.go` (GetVolatilitySurface) | A | `"CN"` → `MarketForSymbol(sym)` |
| `app.go` (GetMinuteLine) | A | 按市场路由到不同适配器 |
| `frontend/.../OrderEntryPanel.vue` | A | `'CN'` → `detectMarket(sym)` |
| `frontend/.../TickerTapePanel.vue` | B | 多市场默认列表 |
| `frontend/.../MarketOverviewPanel.vue` | B | 支持 market 参数 |
| `frontend/.../HeatmapPanel.vue` | B | 支持多市场 |
| `frontend/.../CandlestickPanel.vue` | B | `isTradingHours(market)` |
| `frontend/.../PositionDetail.vue` | B | mock → 真实 API |
| `frontend/src/stores/data.ts` | B | market-aware caching |

### Acceptance Criteria

- [ ] `GetMarketSnapshot(["AAPL", "00700.HK", "600519"])` 分别走 US/HK/CN 链
- [ ] CRYPTO 查询从国内网络可用（gateio 回退）
- [ ] `OrderEntryPanel` 输入 `AAPL` 时走 US 链而非 CN
- [ ] TickerTape 显示美股和港股行情
- [ ] MarketOverview 可切换查看不同市场指数
- [ ] CandlestickPanel 在美股交易时段正确轮询
- [ ] PositionDetail 显示真实持仓数据
- [ ] 所有 Go 测试通过，`vue-tsc --noEmit` 通过

### Risks / Trade-offs

- **Polygon 移除减少 US 回退深度**：但 polygon 是 stub 本来就不工作，移除后 US 链 3 层（yahoo → sina → finnhub）仍可用。finnhub 需要 API key，免费套餐 60 req/min。
- **GetMinuteLine 多市场依赖适配器能力**：yahoo 1m OHLCV 可能受限，HK 分钟线可能需要新适配器。
- **MarketOverview 多市场新增数据依赖**：HK 指数数据需要确认来源（tencent 支持恒指）。

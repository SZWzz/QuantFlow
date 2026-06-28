# Crypto Derivatives Panels (Funding Rate + Liquidation)

## Motivation

加密永续合约的资金费率是衡量市场情绪的核心指标：正费率=多头占优（需付空头费用），负费率=空头占优。当费率极端偏离（>0.1% 或 < -0.1%）时，通常预示方向反转。爆仓数据揭示去杠杆化程度，连续大额爆仓可能引发连锁清算。

当前 `CryptoOverviewPanel` 仅展示现货价格，缺少永续合约和衍生品数据。

## Design

### Go Backend — BinanceFuturesAdapter 扩展

在 `app/internal/market/adapters/binance_futures.go` 中新增方法：

```go
// Binance API: GET /fapi/v1/premiumIndex
type FundingRateData struct {
    Symbol          string  `json:"symbol"`
    MarkPrice       float64 `json:"markPrice"`
    IndexPrice      float64 `json:"indexPrice"`
    FundingRate     float64 `json:"lastFundingRate"`
    NextFundingTime int64   `json:"nextFundingTime"`
}

// Binance API: GET /fapi/v1/allForceOrders
type LiquidationData struct {
    Symbol     string  `json:"symbol"`
    Side       string  `json:"side"`       // BUY/SELL
    Price      float64 `json:"price"`
    Qty        float64 `json:"qty"`
    Amount     float64 `json:"amount"`     // quote amount
    Time       int64   `json:"time"`
    OrderSide  string  `json:"orderSide"`  // SELL for longs, BUY for shorts
}
```

暴露到 `*App`：

```go
func (a *App) GetCryptoFundingRates(symbols []string) ([]FundingRateData, error)
func (a *App) GetCryptoLiquidations(symbol string, limit int) ([]LiquidationData, error)
```

### Frontend — FundingRatePanel

```
┌─────────────────────────────────────────────┐
│ [Sortable Table]                    [auto⟳]│
├─────────────────────────────────────────────┤
│ 品种  │标记价 │指数价 │费率    │下期结算  │趋势│
│ BTC   │68,234 │68,210 │+0.005% │12:00 UTC │⬆️ │
│ ETH   │3,521  │3,518  │+0.012% │12:00 UTC │⬆️ │
│ SOL   │142.5  │142.3  │-0.008% │12:00 UTC │⬇️ │
│ ...                                            │
│ ◆ 费率异常: ETH > 0.01%, 注意多空失衡           │
│ ◆ 8h 资金费率合计可估算多头持仓成本               │
└─────────────────────────────────────────────┘
```

### Frontend — LiquidationPanel

```
┌─────────────────────────────────────────────┐
│ [Symbol search] [24h│7d]         [auto⟳ 30s]│
├─────────────────────────────────────────────┤
│ ┌─ Stats ──────────────────────────────┐   │
│ │ 24h 爆仓总额: $342M  最大单笔: $12.8M │   │
│ │ 多头爆仓: $198M    空头爆仓: $144M    │   │
│ └──────────────────────────────────────┘   │
│                                             │
│ ┌─ Table ──────────────────────────────┐   │
│ │时间    │品种 │方向 │价格  │数量  │金额  │   │
│ │10:32  │BTC  │多→空│67,800│12.5 │$847K │   │
│ │10:31  │ETH  │多→空│3,480 │85.2 │$297K │   │
│ └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

### 跨面板联动

| 联动 | 机制 |
|------|------|
| 费率异常预警 → 爆仓面板高亮 | 同 panel 上下文 |
| 爆仓 symbol 点击 → CryptoOverview | symbolContext（加密 symbol） |
| 费率面板和爆仓面板 | 数据共享同一 adapter |

### Files

| File | Change |
|------|--------|
| `app/internal/market/adapters/binance_futures.go` | 新增 `FetchFundingRates`, `FetchLiquidations` |
| `app/app.go` | 新增 `GetCryptoFundingRates`, `GetCryptoLiquidations` |
| `frontend/src/terminal/panels/FundingRatePanel.vue` | **新增** |
| `frontend/src/terminal/panels/LiquidationPanel.vue` | **新增** |
| `frontend/src/terminal/panels/registry.ts` | 注册 |
| `frontend/src/lib/i18n/zh.ts` + `en.ts` | i18n keys |

## Acceptance Criteria

- [ ] `GetCryptoFundingRates` 返回 TOP20 交易对资金费率
- [ ] `GetCryptoLiquidations` 返回最近 100 条爆仓记录
- [ ] 费率面板：品种、标记价、指数价、费率、下期结算时间
- [ ] 费率面板：按费率排序（高→低）
- [ ] 爆仓面板：24h 统计 + 历史列表
- [ ] 爆仓面板：多/空方向颜色标识
- [ ] 30s 自动刷新
- [ ] 骨架屏 loading

## Risks

- Binance API 可能被屏蔽（近期加密监管趋势），需 fallback 数据源
- `allForceOrders` API 频率限制（20次/分钟）

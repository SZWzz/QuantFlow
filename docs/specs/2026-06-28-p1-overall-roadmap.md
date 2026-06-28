# P1-P2 Terminal Panel Enhancement Roadmap

## Motivation

终端模式 64 个面板已覆盖核心功能，但在以下维度存在缺口：(1) A 股短线信号（龙虎榜、涨跌停）；(2) 港股市场（港股通）；(3) 加密衍生品（资金费率、爆仓）；(4) 宏观决策（经济日历、板块轮动）；(5) UX 体验（骨架屏、错误边界、快捷键）。补齐这些面板可覆盖量化交易者的完整决策链路。

## Design — Panel Interconnections

新面板之间的数据联动关系如下：

```
                   SymbolBar (4 link groups)
                  /        |         |        \
            [Red组]    [Green组]  [Amber组]  [Blue组]
            ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐
            │龙虎榜 │  │港股通 │  │资金费率│  │板块轮动│
            │涨跌停 │  │      │  │爆仓  │  │经济日历 │
            └──────┘  └──────┘  └──────┘  └──────┘
                  │         │         │         │
                  ▼         ▼         ▼         ▼
              dataStore.cache (共享 TTL 缓存层)
                  │         │         │         │
                  ▼         ▼         ▼         ▼
              Existing panels (行情详情, K线, 市场概况...)
```

### 跨面板联动

| 触发事件 | 受影响面板 | 联动机制 |
|----------|-----------|----------|
| 龙虎榜 symbol 点击 | 涨跌停监控、行情详情、K线 | symbolContext.setGroupSymbol |
| 涨跌停列表点击 | 行情详情、K线、资金流向 | symbolContext.setGroupSymbol |
| 港股通净流入异动 | 市场概况(CN)、板块轮动 | dataStore.broadcast(event) |
| 资金费率过高 | 爆仓追踪(风险预警) | 同 panel 内 tab 联动 |
| 经济日历事件 | 板块轮动、市场概况 | dataStore 事件总线 |
| 板块轮动切换 | 自选股、热力图 | 通过 dataStore sector 数据 |

### 新增面板一览

| # | Panel ID | Category | Go Backend | 工作量 |
|---|----------|----------|-----------|--------|
| 1 | `dragon-tiger` | 市场行情 | ✅ 已有 (`GetDailyDragonTiger`, `GetDragonTiger`) | frontend only |
| 2 | `limit-up-down` | 市场行情 | ✅ 复用 `GetAbnormalStocks` + 前端过滤 | frontend only |
| 3 | `hk-connect` | 市场行情 | ✅ 已有 (`GetNorthboundFlow`) | frontend only |
| 4 | `funding-rate` | 市场行情 | ❌ 需新增 Binance 永续费率和爆仓 API | frontend + backend |
| 5 | `liquidation` | 市场行情 | ❌ 同上，同 adapter | frontend + backend |
| 6 | `sector-rotation` | 研究分析 | ✅ 复用 `GetIndustryRanks` + `GetMarketOverview` | frontend only |
| 7 | `economic-calendar` | 研究分析 | ✅ 复用 `GetEconomicIndicators` + 前端转换 | frontend only |
| P2 | 骨架屏系统 | 系统 | — | frontend component |
| P2 | ErrorBoundary | 系统 | — | frontend component |
| P2 | 快捷键扩展 | 系统 | — | frontend logic |

### 文件影响

| 文件 | 改动 |
|------|------|
| `frontend/src/terminal/panels/DragonTigerPanel.vue` | **新增** — 龙虎榜面板 |
| `frontend/src/terminal/panels/LimitUpDownPanel.vue` | **新增** — 涨跌停监控面板 |
| `frontend/src/terminal/panels/HKConnectPanel.vue` | **新增** — 港股通面板 |
| `frontend/src/terminal/panels/FundingRatePanel.vue` | **新增** — 资金费率面板 |
| `frontend/src/terminal/panels/LiquidationPanel.vue` | **新增** — 爆仓追踪面板 |
| `frontend/src/terminal/panels/SectorRotationPanel.vue` | **新增** — 板块轮动/RRG面板 |
| `frontend/src/terminal/panels/EconomicCalendarPanel.vue` | **新增** — 经济日历面板 |
| `frontend/src/terminal/panels/registry.ts` | 注册 7 个新面板 |
| `frontend/src/terminal/DockView/DockTab.vue` | 包裹 ErrorBoundary |
| `frontend/src/terminal/components/ErrorBoundary.vue` | **新增** — 面板级错误边界 |
| `frontend/src/terminal/components/SkeletonPanel.vue` | **新增** — 骨架屏组件 |
| `frontend/src/terminal/TerminalMode.vue` | 快捷键扩展 |
| `app/internal/market/adapters/binance_futures.go` | 新增 funding rate + liquidation 方法 |
| `app/app.go` | 新增 `GetCryptoFundingRates`, `GetCryptoLiquidations` |
| `frontend/src/lib/i18n/zh.ts` + `en.ts` | 新增面板 i18n keys |

## Acceptance Criteria

- [ ] 7 个新面板全部在 registry.ts 注册，CommandBar 可搜索
- [ ] 龙虎榜点击 → 涨跌停/行情详情联动
- [ ] 港股通显示北上资金实时流入/累计/历史
- [ ] 加密面板显示实时资金费率 + 爆仓历史
- [ ] 板块轮动显示 RRG 图表风格相对强度
- [ ] 经济日历显示全球宏观事件 + 预期/实际值
- [ ] ErrorBoundary 捕获面板异常，显示备用 UI
- [ ] SkeletonPanel 用于所有新面板的 loading 态
- [ ] 快捷键 `Ctrl+Shift+D` 龙虎榜、`Ctrl+Shift+H` 港股通等

## Risks / Trade-offs

- 加密面板依赖 Binance API 可用性，若被封需 fallback
- 港股通数据已就绪但缺少南下资金数据源
- 经济日历数据源需从 `GetEconomicIndicators` 转换（当前返回时间序列而非事件列表）
- P2 项目不影响功能但影响用户感知的流畅度

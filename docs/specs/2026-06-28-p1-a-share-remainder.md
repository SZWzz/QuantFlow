# A股残局: IPO 新股日历 + 分红除权日历 + 可转债套利

## Motivation

当前 A 股面板仍缺三个重要功能：
1. **IPO 新股日历** — 新股发行/申购/上市一览，是打新策略的基础
2. **分红除权日历** — 全市场除权除息日程，用于股息捕获策略和复权调整参考
3. **可转债套利** — 基于集思录的溢价率/转股价/强赎预警，现有 BondsPanel 只有行情没有套利分析

这三个面板覆盖了 A 股投资者最常查询的另类数据缺口，且数据源（EastMoney datacenter + Python AKShare sidecar）均已就位，无需引入新依赖。

## Design

### 数据流

```
IPO 新股日历:
  EastMoney API (datacenter RPT_NEW_SHARE_ISSUE)
    → EastMoneySignalsAdapter.FetchIPOCalendar()
      → App.GetIPOCalendar(startDate, endDate)
        → IPOCalendarPanel.vue

分红除权日历:
  EastMoney API (datacenter RPT_SHAREBONUS_DET, date-range filter)
    → EastMoneyCapitalAdapter.FetchExDividendCalendar(startDate, endDate)
      → new research service method or direct adapter call
        → ExDividendPanel.vue

可转债套利:
  Python AKShare (get_bond_cb_jsl + get_bond_cb_redeem_jsl)
    → fetcher.py routes → PythonBridge
      → App.FetchData("akshare", "cb_arbitrage", ...)
        → CBArbitragePanel.vue
```

### 修改/新增文件

**Go Backend:**
- `internal/market/adapters/eastmoney_signals.go` — 新增 `FetchIPOCalendar(ctx, startDate, endDate, pageSize)` 方法
- `internal/market/adapters/eastmoney_capital.go` — 新增 `FetchExDividendCalendar(ctx, startDate, endDate)` 方法 (全市场日期范围版, 不同单个股票)
- `app_market.go` 或 `app_research.go` — 新增 `GetIPOCalendar` / `GetExDividendCalendar` / `GetCBArbitrageData` 三个 Wails 方法

**Python Sidecar:**
- `python/src/data/fetcher.py` — 在 `_AKSHARE_ROUTES` 中添加 `cb_arbitrage` → `get_bond_cb_jsl` 和 `cb_redeem` → `get_bond_cb_redeem_jsl` 路由

**Frontend:**
- `frontend/src/terminal/panels/IPOCalendarPanel.vue` — 新股日历面板
- `frontend/src/terminal/panels/ExDividendPanel.vue` — 分红除权面板
- `frontend/src/terminal/panels/CBArbitragePanel.vue` — 可转债套利面板
- `frontend/src/terminal/panels/registry.ts` — 注册 3 个新面板
- `frontend/src/lib/i18n/zh.ts` + `en.ts` — i18n keys
- `frontend/src/types/wails-runtime.d.ts` — TypeScript 类型声明

### 新面板设计

#### IPOCalendarPanel.vue
- **布局**: 表格列表，按日期分组（近期上市/待申购/已上市）
- **列**: 股票代码, 股票名称, 发行价, 市盈率, 申购日期, 上市日期, 中签率
- **Tab1**: 今日申购 — 当天可申购新股
- **Tab2**: 即将上市 — 未来 1-2 周上市
- **Tab3**: 近期上市 — 过去 1 周涨幅表现
- **数据**: 通过 `app.GetIPOCalendar(start, end)` 获取
- **Empty state**: "暂无新股数据" / "今日无新股申购"

#### ExDividendPanel.vue
- **布局**: 表格列表，日期倒序
- **列**: 股票代码, 股票名称, 除权除息日, 每股派息(税前), 每10股转增, 每10股送股, 进度, 股息率
- **Tab1**: 今日除权 — 当天除权除息
- **Tab2**: 本周除权 — 本周除权除息日历
- **Tab3**: 本月除权 — 本月除权除息日历
- **数据**: 通过 `app.GetExDividendCalendar(start, end)` 获取
- **特色**: 点击代码 → symbol context 跳转
- **计算字段**: 股息率 = 每股派息 / 前收盘价

#### CBArbitragePanel.vue
- **布局**: 可转债套利列表 + 强赎预警
- **列**: 转债代码, 转债名称, 正股代码, 正股价, 转股价, 转股价值, 溢价率%, 税前收益率, 强赎触发价, 强赎天数
- **Tab1**: 套利机会 — 按溢价率排序（负溢价最前）
- **Tab2**: 强赎预警 — 已触发/将触发强赎的转债
- **Tab3**: 回售机会 — 进入回售期的转债
- **数据**: 通过 `FetchData("akshare", "cb_arbitrage", ["all"])` 获取
- **特色**: 溢价率<0 时绿色高亮（套利机会），溢价率>50% 时红色高亮
- **强赎预警**: 已触发强赎的转债标红

## Acceptance Criteria

- [ ] `IPOCalendarPanel` displays upcoming/recent IPO calendar grouped by date
- [ ] `ExDividendPanel` displays ex-dividend calendar with today/this-week/this-month tabs
- [ ] `CBArbitragePanel` shows convertible bond arbitrage data with premium rate highlighting
- [ ] All three panels use SkeletonPanel for loading state
- [ ] All three panels handle empty/error states gracefully
- [ ] All three panels implement symbol click → context linking
- [ ] `GetIPOCalendar` / `GetExDividendCalendar` / `GetCBArbitrageData` exposed as Wails methods
- [ ] Python fetcher.py routes `cb_arbitrage` and `cb_redeem` data types
- [ ] Go backend compiles (`go vet ./...` passes)
- [ ] i18n zh/en keys for all new strings
- [ ] Panels discoverable via Ctrl+K (registry.ts)

## Risks / Trade-offs

- **EastMoney API 稳定性**: IPO 日历和分红日历依赖 EastMoney datacenter API，如果 report name 不对或字段变更需要适配
- **Python 依赖**: 可转债套利数据来自 AKShare (集思录)，需要 Python sidecar 运行；如果 Python 不可用则面板显示引导信息
- **数据时效**: 新股申购数据每天更新一次，非实时；分红日历在季报密集期数据量大，需要分页
- **EastMoney datacenter rate limit**: 500ms + random jitter 限制，3 个面板同时刷新可能排队

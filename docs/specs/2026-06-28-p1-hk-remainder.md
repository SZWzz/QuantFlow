# 港股补缺: 香港IPO + 牛熊证/涡轮 + 交收规则

## Motivation

港股是项目第二大市场（仅次于A股），但目前只有基础行情/OHLCV面板有港股支持。缺少三个重要功能：
1. **香港IPO日历** — 新股认购/分配/上市追踪，港股打新用户的核心需求
2. **牛熊证/涡轮** — 香港独有的衍生品结构产品，对应彭博 `CBBC` / `WARR` 面板
3. **交收规则** — T+2 交收日历、费用计算、假日标记，交易执行必查信息

## Design

### 数据流

```
香港IPO:
  Python AKShare (stock_hk_ipo_subscription + stock_hk_ipo_record)
    → fincept/hk.py → fetcher.py routes
      → App.FetchData("akshare", "hk_ipo", ...)
        → HKIPOPanel.vue

牛熊证/涡轮:
  Python AKShare (stock_hk_cbbc + stock_hk_warrants)
    → fincept/hk.py → fetcher.py routes
      → App.FetchData("akshare", "hk_cbbc", ...)
      → App.FetchData("akshare", "hk_warrants", ...)
        → HKDerivativesPanel.vue

交收规则:
  Go 静态规则 (T+2, stamp duty, exchange fee, SFC levy)
    + Python AKShare (tool_trade_date_hist) 或硬编码假日日历
    → App.GetHKSettlementInfo()
    → App.GetHKTradingCalendar(year)
      → HKSettlementPanel.vue
```

### 修改/新增文件

**Python Sidecar:**
- `python/src/data/fincept/hk.py` — 新模块，封装 AKShare HK 数据函数

**Go Backend:**
- `app_market.go` 或 `app_research.go` — 新增 `GetHKIPOCalendar` / `GetHKDerivatives` / `GetHKSettlementInfo` / `GetHKTradingCalendar`

**Frontend:**
- `frontend/src/terminal/panels/HKIPOPanel.vue` — 香港IPO面板
- `frontend/src/terminal/panels/HKDerivativesPanel.vue` — 牛熊证/涡轮面板
- `frontend/src/terminal/panels/HKSettlementPanel.vue` — 交收规则面板
- registry.ts + i18n + CHANGELOG

### 新面板设计

#### HKIPOPanel.vue
- Tab1: 正在认购 — 可申购的新股 (认购价、截止日、入场费)
- Tab2: 即将上市 — 已分配待上市 (上市日、发行价、认购倍数)
- Tab3: 历史表现 — 近期上市涨跌幅

#### HKDerivativesPanel.vue
- Tab1: 牛证(牛熊证) — 牛证列表
- Tab2: 熊证 — 熊证列表
- Tab3: 涡轮 — 认股证列表
- 列: 代码, 名称, 行使价, 到期日, 换股比率, 溢价率, 杠杆比率, 街货量

#### HKSettlementPanel.vue
- 交收时间线: Trade Date → T+2 → Settlement Date
- 费用计算器: 输入成交金额 → 自动计算各种费用
- 假日日历: 全年港股通/港股休市日

## Acceptance Criteria
- [ ] HKIPOPanel displays HK IPO subscription + listing data
- [ ] HKDerivativesPanel shows CBBC and warrant data with key columns
- [ ] HKSettlementPanel shows T+2 timeline and fee calculator
- [ ] Python `fincept/hk.py` module created and routes registered
- [ ] Go wrapper methods exposed
- [ ] `go vet ./...` passes
- [ ] i18n keys for all three panels

## Risks / Trade-offs
- AKShare HK 数据依赖 Python sidecar
- 牛熊证/涡轮数据需要 AKShare 的 `stock_hk_cbbc` 和 `stock_hk_warrants` 函数可用
- 假日日历建议硬编码 2025-2026 年 HKEX 日历+动态计算

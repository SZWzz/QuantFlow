# Implementation Plan: A股残局 — IPO 新股日历 + 分红除权 + 可转债套利

## Batch N: IPO 新股日历 (IPOCalendarPanel)

**依赖**: Batch A (SkeletonPanel + ErrorBoundary 已就位)

- [ ] **N1**: `eastmoney_signals.go` 添加 `FetchIPOCalendar` 方法
- [ ] **N2**: `app_research.go` 或 `app_market.go` 添加 `GetIPOCalendar` Wails 方法
- [ ] **N3**: `wails-runtime.d.ts` 添加 TypeScript 签名
- [ ] **N4**: 创建 `IPOCalendarPanel.vue`
- [ ] **N5**: i18n keys + register

## Batch O: 分红除权日历 (ExDividendPanel)

**依赖**: Batch A

- [ ] **O1**: `eastmoney_capital.go` 添加 `FetchExDividendCalendar` 方法 (日期范围查询)
- [ ] **O2**: `app_market.go` 添加 `GetExDividendCalendar` Wails 方法
- [ ] **O3**: `wails-runtime.d.ts` 添加 TypeScript 签名
- [ ] **O4**: 创建 `ExDividendPanel.vue`
- [ ] **O5**: i18n keys + register

## Batch P: 可转债套利 (CBArbitragePanel)

**依赖**: Batch A + Python sidecar

- [ ] **P1**: `python/src/data/fetcher.py` 添加 `cb_arbitrage` + `cb_redeem` 路由
- [ ] **P2**: `app_market.go` 添加 `GetCBArbitrageData` Wails 方法
- [ ] **P3**: `wails-runtime.d.ts` 添加 TypeScript 签名
- [ ] **P4**: 创建 `CBArbitragePanel.vue`
- [ ] **P5**: i18n keys + register

## Batch Q: 收尾

- [ ] **Q1**: 更新 `CHANGELOG.md`
- [ ] **Q2**: `go vet ./...` 验证

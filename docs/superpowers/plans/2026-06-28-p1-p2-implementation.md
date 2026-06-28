# Implementation Plan: P1-P2 Terminal Panel Enhancements

## Overview

7 个新面板 + 4 个 UX 增强。分 8 个 Batch 执行，每个 Batch 独立可交付。

## Batch A: Foundation Components (Skeleton + ErrorBoundary)

**依赖:** 无。是所有新面板的前提。

- [ ] **A1**: 创建 `SkeletonPanel.vue` (table/card/chart 三种类型)
- [ ] **A2**: 创建 `ErrorBoundary.vue` (捕获 errorCaptured, 显示备用 UI)
- [ ] **A3**: `DockTab.vue` 包裹 ErrorBoundary

## Batch B: Dragon Tiger Panel (龙虎榜)

**依赖:** Batch A。Go backend 无需改动。

- [ ] **B1**: 创建 `DragonTigerPanel.vue` — 日榜单 tab (调用 `GetDailyDragonTiger`)
- [ ] **B2**: 添加个股历史 tab (调用 `GetDragonTiger`)
- [ ] **B3**: 展开行显示买入/卖出 TOP5 营业部
- [ ] **B4**: i18n zh/en keys
- [ ] **B5**: registry.ts 注册

## Batch C: Limit Up/Down Panel (涨跌停)

**依赖:** Batch A。Go backend 无需改动。

- [ ] **C1**: 创建 `LimitUpDownPanel.vue` — 从 `GetAbnormalStocks` 过滤
- [ ] **C2**: 涨停/跌停 tab 分离 + 统计计数
- [ ] **C3**: i18n + registry

## Batch D: HK Connect Panel (港股通)

**依赖:** Batch A。Go backend 无需改动。

- [ ] **D1**: 创建 `HKConnectPanel.vue` — 北向资金 tab
- [ ] **D2**: 分时走势 ECharts 面积图
- [ ] **D3**: 每日历史表格
- [ ] **D4**: 额度概览 tab
- [ ] **D5**: i18n + registry

## Batch E: Sector Rotation Panel (板块轮动/RRG)

**依赖:** Batch A。Go backend 无需改动（需验证 sector 数据历史序列是否足够）。

- [ ] **E1**: 创建 `SectorRotationPanel.vue`
- [ ] **E2**: RRG 散点图 ECharts (JdK Ratio, Rs Ratio)
- [ ] **E3**: 板块强度表格
- [ ] **E4**: i18n + registry

## Batch F: Economic Calendar Panel (经济日历)

**依赖:** Batch A。Go backend 无需改动。

- [ ] **F1**: 创建 `EconomicCalendarPanel.vue`
- [ ] **F2**: 时间线 UI + 日期分组
- [ ] **F3**: 前值/预期/实际 三级展示
- [ ] **F4**: i18n + registry

## Batch G: Crypto Derivatives (Funding Rate + Liquidation)

**依赖:** Batch A + Go backend changes。

- [ ] **G1**: `binance_futures.go` 添加 `FetchFundingRates`
- [ ] **G2**: `binance_futures.go` 添加 `FetchLiquidations`
- [ ] **G3**: `app.go` 暴露 `GetCryptoFundingRates`, `GetCryptoLiquidations`
- [ ] **G4**: 创建 `FundingRatePanel.vue`
- [ ] **G5**: 创建 `LiquidationPanel.vue`
- [ ] **G6**: registry + i18n

## Batch H: Shortcuts + WelcomePanel

**依赖:** 面板已注册。

- [ ] **H1**: `TerminalMode.vue` 注册快捷键监听
- [ ] **H2**: WelcomePanel 动态化（最近面板+市场快照）
- [ ] **H3**: session store 添加 recentPanels

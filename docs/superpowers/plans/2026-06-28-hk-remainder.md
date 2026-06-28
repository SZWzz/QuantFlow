# Implementation Plan: 港股补缺 — 香港IPO + 牛熊证/涡轮 + 交收规则

## Batch R: Python HK Data Module + Go Backend

- [ ] **R1**: 创建 `python/src/data/fincept/hk.py` — AKShare HK 数据封装
- [ ] **R2**: `fetcher.py` 添加 `hk_ipo` / `hk_cbbc` / `hk_warrants` / `hk_trade_cal` 路由
- [ ] **R3**: `app_market.go` 添加 Wails 方法 + `app_research.go` 添加交收规则方法
- [ ] **R4**: TypeScript 类型声明

## Batch S: 前端面板

- [ ] **S1**: 创建 `HKIPOPanel.vue`
- [ ] **S2**: 创建 `HKDerivativesPanel.vue`
- [ ] **S3**: 创建 `HKSettlementPanel.vue`
- [ ] **S4**: registry.ts + i18n + CHANGELOG

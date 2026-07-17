# 日结报告 (Daily P&L Report)

## Motivation

用户交易一天后，需要一份清晰的日结报告：今日盈亏、持仓市值、交易笔数、佣金税费、最大回撤。当前 Portfolio 模块有 analytics 计算能力，但没有自动生成和推送日结报告的机制。

用户需要在收市后（或睡前）收到一份完整摘要，无论是否打开终端。

## Design

### 报告结构

```go
// internal/trading/daily_report.go
type DailyReport struct {
    Date         string        // "2026-07-16"
    MarketValue  float64       // 持仓市值
    DayPNL       float64       // 今日盈亏
    DayPNLPercent float64      // 今日收益率
    TotalPNL     float64       // 累计盈亏
    TotalPNLPercent float64    // 累计收益率
    Trades       int           // 今日成交笔数
    Commission   float64       // 今日佣金
    Tax          float64       // 今日税费
    MaxDrawdown  float64       // 今日最大回撤
    BestTrade    TradeSummary  // 今日最佳
    WorstTrade   TradeSummary  // 今日最差
    Positions    []PositionSummary // 持仓摘要
    Notes        string        // 用户备注
}
```

### 数据流

```
收市后（定时任务 15:00 / 16:00 / 次日开盘前）
  ↓
TradingEngine.GenerateDailyReport()
  ↓
存储到 SQLite daily_reports 表
  ↓
ws push "trading:daily-report"
  ↓
前端：终端弹出报告面板 (非阻塞)
  + 通知: notify.Manager.Send("daily_report", "今日盈亏: +¥2,350 (1.2%)")
  + Telegram 推送 (如配置)
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/trading/daily_report.go` | 新建 | 报告生成引擎 |
| `internal/trading/repo.go` | 修改 | 新增 `SaveDailyReport` / `GetDailyReports` |
| `internal/storage/migrations/020_daily_reports.sql` | 新建 | `daily_reports` 表 |
| `internal/schedule/scheduler.go` | 修改 | 注册每日收市报告任务 |
| `internal/notify/manager.go` | 修改 | 支持 daily_report 通知类型 |
| `frontend/src/terminal/panels/DailyReportPanel.vue` | 新建 | 报告展示面板 |
| `frontend/src/stores/portfolio.ts` | 修改 | 新增 `dailyReports` state |

### SQLite Schema

```sql
-- migration 020
CREATE TABLE daily_reports (
    date        TEXT PRIMARY KEY,  -- "2026-07-16"
    created_at  INTEGER NOT NULL,
    report_json TEXT NOT NULL,
    summary     TEXT NOT NULL,     -- "今日盈亏: +¥2,350 (1.2%) | 交易: 12 笔"
    pnl         REAL NOT NULL,
    pnl_percent REAL NOT NULL
);

CREATE INDEX idx_daily_reports_date ON daily_reports(date DESC);
```

### 前端面板

```
┌──────────────────────────────────────────────┐
│  📋 日结报告 · 2026-07-16                     │
├──────────────────────────────────────────────┤
│  💰 今日盈亏   +¥2,350  (+1.2%)              │
│  📊 累计盈亏   +¥12,800 (+6.8%)               │
│  🏦 持仓市值   ¥198,500                       │
│  📉 最大回撤   -0.8%                          │
│  🔄 交易 12 笔 | 佣金 ¥18.50 | 税费 ¥5.20    │
├──────────────────────────────────────────────┤
│  🏆 最佳: TSLA +¥520 (买入 ¥242 → ¥247)    │
│  😞 最差: AAPL -¥180 (卖出 ¥178 → ¥176)     │
├──────────────────────────────────────────────┤
│  持仓 (5)                                    │
│  AAPL  100  ¥17,520  +0.3%                   │
│  TSLA  50   ¥12,250  +2.1%                   │
│  ...                                        │
├──────────────────────────────────────────────┤
│  📝 备注: [今日减仓 AAPL，加仓 TSLA...]       │
│                                              │
│  [导出 CSV]  [分享摘要]  [固定到面板]          │
└──────────────────────────────────────────────┘
```

### 定时触发条件

| 市场 | 触发时间 | 说明 |
|------|---------|------|
| A 股 | 15:30 (收市后 30min) | T+1 结算，当日数据 |
| 港股 | 16:30 | 16:00 收盘 |
| 美股 | 实时 (16:00 ET) / 次日 08:00 | 盘后数据就绪 |
| 加密 | 每日 00:00 UTC | 按日结算 |

### 通知模板

```
📊 QuantFlow 日结报告 — 2026-07-16

💰 今日盈亏: +¥2,350 (+1.2%)
📊 累计盈亏: +¥12,800 (+6.8%)
🔄 交易: 12 笔 | 佣金: ¥18.50
📉 最大回撤: -0.8%

🏆 最佳: TSLA +¥520
😞 最差: AAPL -¥180

查看详情: quantflow://daily-report/2026-07-16
```

## Acceptance Criteria

- [ ] GenerateDailyReport 汇总当日成交、持仓、盈亏、最大回撤
- [ ] 报告按市场收市时间自动触发（A股 15:30，港股 16:30，美股次晨 08:00）
- [ ] 报告持久化到 SQLite，可按日期查询历史
- [ ] 前端 DailyReportPanel 展示完整报告（盈亏、最佳/最差交易、持仓摘要）
- [ ] 报告生成后 ws 推送，前端自动弹出非阻塞通知
- [ ] 如配置了 Telegram，推送文本摘要
- [ ] 报告包含用户备注字段
- [ ] 导出 CSV 功能
- [ ] Go 测试覆盖报告生成逻辑（mock 成交记录 → 验证 P&L 计算）
- [ ] 迁移 SQL 向前兼容

## Risks / Trade-offs

- **风险**: 收市时间因交易所节假日变化。→ 通过 `trading_hours.go` 获取实际收市时间，不硬编码
- **风险**: Paper 模式和 Live 模式的报告混在一起。→ 按 TradingMode 分开生成，标注明细
- **Trade-off**: v1 不做多账户合并报告（用户有多个 Alpaca 账户），每个 broker 单独出报告

# Paper→Live 实盘切换流程

## Motivation

当前 Paper Engine 和 Broker Adapter 各自独立工作：PaperTrading 用 cash ledger 虚拟成交，Broker 直接对接交易所。没有从"模拟"切换到"实盘"的仪式感——用户可能在 Paper 模式下测了几天策略，突然想上实盘时发现没有清晰路径。

这有真实资金风险：用户以为还在模拟，实际已真实下单。

## Design

### 核心概念：TradingMode

`internal/trading/types.go` 新增：

```go
type TradingMode string
const (
    TradingModeInvalid TradingMode = ""
    TradingModePaper   TradingMode = "paper"
    TradingModeLive    TradingMode = "live"
)
```

TradingEngine 持有当前模式，所有下单 API 检查模式并明确提示。

### 切换流程

```
用户点击 "Go Live"
  ↓
安全检查面板
  ┌──────────────────────────────────────────┐
  │  ⚠️  即将切换到实盘模式                     │
  │                                           │
  │  检查清单:                                 │
  │  ✅ Broker 连接正常 (Alpaca paper env)    │
  │  ✅ API Key 已配置                         │
  │  ✅ 风控规则已加载 (最大单笔 10%)            │
  │  ⚠️  日亏损限额 -2% 未设置                  │
  │  ❌ 紧急关停按钮未测试                      │
  │                                           │
  │  [取消]     [强制切换]     [修复后切换]     │
  └──────────────────────────────────────────┘
  ↓
执行切换
  ↓
TradingEngine 写入 TradingMode=live
  → 所有 PlaceOrder 调用 broker.PlaceOrder (非 paper.Execute)
  → UI 显示 🔴 LIVE MODE 横幅 (全终端可见)
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/trading/types.go` | 修改 | 新增 `TradingMode`, `SafetyCheck` struct |
| `internal/trading/engine.go` | 修改 | `SetMode(mode)` + 安全检查方法 |
| `internal/trading/paper_engine.go` | 修改 | 支持从 Paper 导出仓位到 Live |
| `internal/trading/risk_pipeline.go` | 修改 | Live 模式强制启用风控 |
| `app_trading.go` | 追加 | `SwitchToLive(skipChecks bool) SafetyReport` IPC |
| `frontend/src/terminal/panels/TradingJournalPanel.vue` | 新建 | 切换面板 + 模式指示器 |
| `frontend/src/terminal/components/LiveModeBanner.vue` | 新建 | 🔴 永久横幅 |
| `frontend/src/stores/terminal.ts` | 修改 | 新增 `tradingMode` state |

### 模式指示器

```
终端顶部 (固定在 MenuBar 下方):

[Paper Mode]  [🔒]  ← 点击弹出切换面板
    ↑ 绿色标签

[🔴 LIVE MODE]  [🛑 紧急平仓]  ← 点击弹出确认
    ↑ 红色闪烁横幅
```

### 紧急关停

Live 模式时有红色"紧急平仓"按钮：
1. `confirmDialog("确定平掉所有持仓？此操作不可撤销")` → await
2. 并行向所有活跃 broker 发送 `CancelAllOrders` + `CloseAllPositions`
3. 切换回 Paper 模式
4. 弹出结果报告（已平仓位数 + 未成交撤销数）

### Paper→Live 仓位迁移

用户可选项：
- **清空 Paper 仓位**：Paper 和历史隔离，Live 从零开始
- **迁移 Paper 仓位**：将 Paper 持仓按当前市价导入 Live 模式（按实际价格成交）
- **镜像模式**：Paper 继续运行，下单时同时发给 Paper + Live（比例可配 1:10）

## Acceptance Criteria

- [ ] TradingEngine 持有当前模式，默认 Paper
- [ ] SwitchToLive 弹出安全检查面板（Broker 连接 / API Key / 风控 / 限额）
- [ ] 安全检查不合格时列出具体失败项，"强制切换"需二次确认
- [ ] Live 模式终端顶部显示 🔴 红色横幅，不可关闭
- [ ] 所有下单 API 在 Live 模式调用真实 Broker
- [ ] Paper→Live 仓位迁移选项（清空/迁移/镜像）
- [ ] 紧急关停按钮 → confirmDialog → 并行平仓 → 报告
- [ ] 紧急关停后自动回退 Paper 模式
- [ ] TradingMode 持久化到 SQLite（重启保留）
- [ ] 前端测试覆盖 LiveModeBanner + 切换面板
- [ ] Go 测试覆盖 SafetyCheck + 模式切换

## Risks / Trade-offs

- **风险**: 用户误触"Go Live"导致真金白银下单。→ 三确认：安全检查 → confirmDialog → PIN 二次验证（可选）
- **风险**: 紧急关停不保证全部成交（流动性不足）。→ 显示"已发出指令"而非"已全部平仓"
- **Trade-off**: 不实现镜像模式 v1（太复杂），v2 再补

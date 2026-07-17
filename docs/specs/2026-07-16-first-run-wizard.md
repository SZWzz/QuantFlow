# 首次启动向导 (First-Run Wizard)

## Motivation

新用户下载 QuantFlow 后打开终端，面对空白 DockView + 93 个面板无从下手。当前无任何引导，用户不知道如何配数据源、加自选股、切换模式。首次启动体验直接决定留存率。

## Design

### 检测机制

`frontend/src/lib/` 新增 `useFirstRun.ts` composable：

```
App 启动 → localStorage 检查 quantflow_first_run_done flag
           ↓
        未标记 → 弹出向导 (多层 Step)
        已标记 → 正常加载默认布局
```

向导完成后设 flag，可通过 SettingsPanel 手动重放。

### 向导流程（3 步）

```
Step 1: 市场选择
  ┌──────────────────────────────────┐
  │  选择你需要覆盖的市场              │
  │  ☑ A 股  ☑ 港股  ☐ 美股  ☐ 加密   │
  │                                   │
  │  [上一步]              [下一步 →]  │
  └──────────────────────────────────┘

Step 2: 数据源配置
  ┌──────────────────────────────────┐
  │  配置数据源 API Key               │
  │  (根据 Step 1 筛选展示)            │
  │                                   │
  │  TuShare:   [················] 🔑 │
  │  Polygon:   [················] 🔑 │
  │  QOS:       [················] 🔑 │
  │  ...                              │
  │  跳过 → 以后在 Settings 中配置      │
  │                                   │
  │  [上一步]              [下一步 →]  │
  └──────────────────────────────────┘

Step 3: 默认布局 + 快速开始
  ┌──────────────────────────────────┐
  │  选择你的角色                      │
  │                                   │
  │  📊 日内交易                      │
  │     Watchlist + Candlestick + TickerTape│
  │                                   │
  │  📈 波段/趋势                     │
  │     MarketOverview + Heatmap + SectorRotation│
  │                                   │
  │  🔬 量化研究                      │
  │     Backtest + FactorAnalysis + AIChat│
  │                                   │
  │  📋 通用                          │
  │     默认多面板布局                  │
  │                                   │
  │  [上一步]              [完成 ✨]   │
  └──────────────────────────────────┘
```

### 新文件

| 文件 | 说明 |
|------|------|
| `frontend/src/terminal/components/SetupWizard.vue` | 向导主组件 (Stepper + 3 Step) |
| `frontend/src/terminal/components/SetupStepMarket.vue` | Step 1: 市场选择 |
| `frontend/src/terminal/components/SetupStepAPIKeys.vue` | Step 2: API Key 配置 |
| `frontend/src/terminal/components/SetupStepProfile.vue` | Step 3: 角色选布局 |
| `frontend/src/lib/useFirstRun.ts` | first-run 检测 + flag 管理 |

### 修改文件

| 文件 | 修改 |
|------|------|
| `frontend/src/App.vue` | 启动时调用 `useFirstRun().check()` |
| `frontend/src/terminal/DockView/DockView.vue` | 提供默认布局应用接口 |
| `frontend/src/stores/terminal.ts` | 新增 `applyDefaultLayout(profile)` action |

### 数据流

```
App.vue mount
  → useFirstRun().check()
    → localStorage.getItem("quantflow_first_run_done")
      → null → 渲染 SetupWizard (overlay, 不可关闭)
      → "done" → 正常渲染
```

### 默认布局定义

内置 4 种 profile，每种是一个预定义面板 ID 数组 + split 方向：

```typescript
const PROFILES = {
  intraday: {
    name: '日内交易',
    icon: 'chart-line',
    panels: ['WatchlistPanel', 'CandlestickPanel', 'TickerTapePanel',
             'OrderEntryPanel', 'BrokerStatusPanel', 'DepthComparisonPanel'],
    layout: 'horizontal-3split'
  },
  swing: { ... },
  quant: { ... },
  general: { ... }
}
```

## Acceptance Criteria

- [ ] 全新安装首次启动弹出向导，覆盖整个窗口（不可跳过，不可关闭）
- [ ] Step 1 选择市场后 Step 2 只展示相关数据源
- [ ] API Key 输入后通过 CredentialManager 加密存储
- [ ] Step 3 选择角色后自动布局生效
- [ ] 完成后设 flag，第二次启动直接进入终端
- [ ] SettingsPanel 有"重新运行向导"按钮
- [ ] 向导组件全量测试覆盖（vitest）
- [ ] `localStorage` flag + SQLite fallback 双保险

## Risks / Trade-offs

- **风险**: 向导阻止用户直接使用终端，急性子用户可能反感。→ 加"跳过所有，直接使用"按钮（但警告缺少数据源）
- **风险**: API Key 输入在向导中可能被用户视为"太重"。→ Step 2 可完全跳过，Settings 中补
- **Trade-off**: 不引入任何新依赖，纯 Vue 组件 + localStorage

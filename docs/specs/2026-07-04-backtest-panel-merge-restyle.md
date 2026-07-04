# Backtest Panel 合并 + 统一卡片风格

## Motivation

当前有两个独立的面板：
1. **BacktestHistoryPanel** — 回测历史列表（仅列表，浏览/删除）
2. **BacktestResultPanel** — 回测结果详情（K线+买卖点+净值曲线+指标+交易记录）

两者功能互补却分散在两个标签页中，用户在历史面板中点击一行后会新开一个结果面板，体验割裂。同时 BacktestResultPanel 使用自建样式，未遵循统一的卡片风格（PanelHeader / PanelCard / PanelTable / EmptyState / LoadingState）。

合并后：一个面板内包含列表→详情的完整工作流，且全部使用统一设计语言。

## Design

### 数据流

```
Go: ListBacktestHistory(limit, offset)
  → Frontend: items[] (列表)
    → 用户点击某行
      → Go: GetStoredBacktestResult(id)
        → storedData (详情：K线/净值/指标/Trades)
          → ECharts 渲染
```

### 新面板结构

单个面板 `BacktestPanel.vue`，替代 `BacktestHistoryPanel.vue` + `BacktestResultPanel.vue`。

**视图状态机**：
```
[list] ←→ [detail]
   ↑ 点击行     ← 返回按钮
```

#### List 视图
- `PanelHeader` — 标题"回测历史"，刷新按钮
- `PanelTable` — 历史记录列表（日期/工作流/策略/标的/收益率/Sharpe/交易次数）
- `EmptyState` — 无数据
- 行点击 → detail 视图

#### Detail 视图
- `PanelHeader` — 面包屑导航（← 返回 + "策略名 | 标的" 子标题），删除按钮
- ECharts K 线图（含买卖点标记）
- ECharts 净值曲线
- **Metrics 卡片网格** — 使用 `PanelCard` 显示：总收益率、年化、最大回撤、夏普、索提诺、卡玛、胜率、盈亏比、年化波动率、总交易次数
- **交易记录** — 使用 `PanelTable` 显示交易明细
- `LoadingState` — 加载中骨架屏
- `EmptyState` — 无数据 / 已删除

### 组件层级

```
BacktestPanel.vue
├── PanelHeader (title + controls: refresh / back + delete)
├── [view=list]
│   ├── EmptyState / LoadingState
│   └── PanelTable (columns: date, workflow, strategy, symbol, return, sharpe, trades, action)
├── [view=detail]
│   ├── EmptyState / LoadingState
│   ├── VChart (K-line + buy/sell marks)
│   ├── VChart (equity curve)
│   ├── PanelCard grid (10 metrics)
│   └── PanelTable (trades)
```

### 修改的文件

| 文件 | 动作 |
|------|------|
| `frontend/src/terminal/panels/BacktestPanel.vue` | **新建** — 合并后的面板 |
| `frontend/src/terminal/panels/BacktestHistoryPanel.vue` | **删除** |
| `frontend/src/terminal/panels/BacktestResultPanel.vue` | **删除** |
| `frontend/src/terminal/panels/registry.ts` | 注册 `backtest` 取代 `backtest-history` 和 `backtest-result` |
| `frontend/src/stores/terminal.ts` | 如有必要，更新默认布局 |
| `CHANGELOG.md` | 记录变更 |

### Go 后端

无变更。复用现有的 `ListBacktestHistory` / `GetStoredBacktestResult` / `DeleteBacktestResult`。

## Acceptance Criteria

- [ ] 新建面板注册为 `'backtest'`，分类"量化分析"
- [ ] 打开面板默认显示历史列表，使用 `PanelTable` 和 `PanelHeader`
- [ ] 列表行点击切换到详情视图，显示 K 线 + 净值曲线 + 指标卡片
- [ ] 详情视图顶部有"← 返回"按钮返回列表
- [ ] 指标使用 `PanelCard` 网格，遵循统一样式
- [ ] 交易记录使用 `PanelTable`
- [ ] 无数据时显示 `EmptyState`
- [ ] 加载中显示 `LoadingState`
- [ ] 删除逻辑正常工作（列表批量删除 + 详情单条删除）
- [ ] 旧面板文件被删除，registry 中 `backtest-history` 和 `backtest-result` 不再注册

## Risks / Trade-offs

- 现有工作流和 Welcome 面板中指向 `backtest-result` 面板的链接需要更新为 `backtest`。
- 向后兼容：如果用户有保存的布局引用了 `backtest-result` 或 `backtest-history`，会显示"Panel not registered"。可接受，因为布局通常不会持久化已关闭的面板。
- ECharts 的 `VChart` 组件保持不变，不重构为 PanelCard。

# Workflow Mode Enhancements (P1-P3)

## Motivation

工作流模式有 100+ 节点和扎实的 DAG 引擎，但缺少三个层次的能力：

1. **P1 可用性**：新用户面对空白画布不知从何开始，端口无类型校验
2. **P2 体验**：节点视觉雷同，无多工作流管理
3. **P3 量化深度**：只支持单次执行，无滚动回测和参数优化

## Design

### P1.1 — 预置工作流模板 (7 个)

NodePalette 底部新增模板区，点击即插入完整工作流：

| 模板 | 节点链路 |
|------|---------|
| 金叉选股 | data_loader → SMA(5) + SMA(20) → cross_over → entry_signal |
| MACD 底背离 | data_loader → MACD → 价格找低点 + DIF 找高点 → compare → entry_signal |
| 多因子打分 | data_loader → 动量因子+波动因子+质量因子 → scale → merge → rank_select |
| 布林带突破 | data_loader → BB(20,2) → 价格<下轨 → entry_signal + 价格>上轨 → exit_signal |
| RSI 超卖反弹 | data_loader → RSI(14) → threshold_signal(30/70) → entry/exit |
| 均值回归配对 | data_loader×2 → std_dev(价差) → zscore>2 → entry_signal |
| MACD+RSI 共振 | data_loader → MACD + RSI → signal_combine → entry_signal |

每个模板是预定义 JSON，插入画布后可自由修改。

### P1.2 — 端口类型校验

`ListNodes` 返回每个节点的端口定义（名称+类型）。前端连线时检查：

- 同类型 → 绿色实线
- 兼容（如 ohlcv→series）→ 黄色虚线 + tooltip
- 不兼容 → 阻止连接 + 红色闪烁

### P2.1 — 多工作流管理

工具栏新增「工作流列表」按钮，弹出侧边抽屉展示已保存工作流。支持新建/打开/重命名/删除。存储从单键升级为 `quantflow-workflows` 数组。

### P2.2 — 节点视觉差异化

按类别分配颜色边框 + emoji 图标：
data(蓝📊) / indicator(绿📈) / signal(橙⚡) / trading(红💰) / risk(深红🛡️) / portfolio(青📦) / strategy(天蓝🧠) / ml(紫🤖) / output(浅紫📝) / control(靛蓝🔀)

### P3.1 — Walk-Forward 回测

新增 `ExecuteBacktest(ctx, wf, config)` — 滚动窗口训练/测试分离，输出每期 Sharpe/回撤曲线。

### P3.2 — 参数优化

新增 `OptimizeParams(ctx, wf, config)` — 网格搜索遍历参数空间，返回 Top 5 参数组合及对应绩效。

## Acceptance Criteria

- [ ] 7 个预置模板可一键插入画布
- [ ] 端口类型兼容性检查，不兼容阻止连接
- [ ] 工作流保存/加载/重命名/删除，列表管理
- [ ] 节点按类别显示不同颜色和图标
- [ ] Walk-Forward 输出滚动窗口绩效曲线
- [ ] 参数优化返回 Top 5 参数组合

## Risks

- `ListNodes` 返回结构需向后兼容
- Walk-Forward 时间长需进度反馈
- 参数优化穷举型，参数空间大时可能超时

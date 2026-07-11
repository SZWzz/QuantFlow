# P1 — 技术债清理：代码质量与正确性

> 日期: 2026-07-11 | 优先级: P1 | 预计工作量: 12-16 小时

## Motivation

评审发现 12 项已批准但未执行的 P1 技术债（来自 `2026-07-11-comprehensive-review.md`）+ 3 项新发现的代码质量问题。这些债务如果不清理，将：

1. **持续腐蚀回测可信度** — wash sale 计算错误、美股默认 100 股导致回测 PnL 失真
2. **阻塞性能优化** — channel leak、busy-wait 100% CPU 空转浪费资源
3. **阻碍团队协作** — 工具函数散落 4 个文件、上帝方法不可维护
4. **降低用户体验** — CandlestickPanel 1147 行巨型组件难以加载和调试

## Design

### Part A: Go 后端质量修复（6 项，来自已有 Spec）

| # | 修复项 | 文件 | 改动量 |
|---|--------|------|:------:|
| A1 | Wash sale 计算修复 — 亏损比较卖出价 vs 原始成本价 | `internal/trading/wash_sale.go` | ~20 行 |
| A2 | Stamp duty 四舍五入 — `math.Round(tradeValue*rate*100)/100` | `internal/trading/backtest/engine_cn.go` | ~5 行 |
| A3 | 美股默认 1 股 — `qty = 1`（非 100） | `internal/trading/backtest/engine_us.go` | ~3 行 |
| A4 | Sharpe 无风险利率可配置 — 新增 `MetricsConfig.RiskFreeRate` | `internal/trading/backtest/metrics.go` | ~15 行 |
| A5 | Channel leak 修复 — `hub.go` 新增 `subscriber` struct 封装 channel + close 语义 | `internal/market/hub.go` | ~30 行 |
| A6 | Busy-wait 修复 — `queue.go` 用 `sync.Cond` 替换忙等循环 | `internal/workflow/queue.go` | ~20 行 |

### Part B: 工具函数集中化（新发现）

**当前状态**:
- `extractFloatSlice` 定义在 `macd.go`，被 56 个文件 import——位置不合理
- `extractFloat64Slice` 定义在 `floatutil.go`——与上面功能重复
- `getStringParam/getFloatParam/getIntParam` 散落在 `factor.go` 和 `strategy.go`

**方案**: 创建 `internal/workflow/nodes/utils.go`，迁移所有通用工具函数:
```go
// utils.go — shared parameter extraction and type conversion helpers
func extractFloatSlice(inputs map[string]any, port string) ([]float64, error) { ... }
func getStringParam(params map[string]any, key, defaultVal string) string { ... }
func getFloatParam(params map[string]any, key string, defaultVal float64) float64 { ... }
func getIntParam(params map[string]any, key string, defaultVal int) int { ... }
```

然后更新所有 56+ 处 import 路径。

### Part C: CandlestickPanel 拆分

**当前**: `CandlestickPanel.vue` 1147 行，混合了 K 线图、工具栏、指标叠加层、分时图切换。

**拆分方案**:
```
CandlestickPanel.vue (主容器, ~300 行)
├── ChartToolbar.vue (工具栏: 周期切换、指标选择、导出, ~150 行)
├── KlineChart.vue (K 线图本体: ECharts 配置 + dataZoom, ~350 行)
├── IndicatorOverlay.vue (指标叠加: MA/EMA/Bollinger/MACD 等, ~200 行)
└── MinuteChart.vue (分时图: 复用 useMinuteChart composable, ~150 行)
```

### Part D: godoc 补充

- `internal/workflow/engine.go` — 添加 Execute 管道的完整文档
- `internal/workflow/nodes/register.go` — 添加节点注册机制文档

## Acceptance Criteria

- [ ] A1: wash sale 亏损计算使用原始成本价，有单元测试
- [ ] A2: stamp duty 四舍五入到分（0.01 元），有测试
- [ ] A3: 美股默认交易 1 股，`engine_us_test.go` 通过
- [ ] A4: Sharpe/Sortino 支持自定义 `RiskFreeRate`，默认 0.02
- [ ] A5: Channel subscriber 使用 struct 封装，unsubscribe 时 close channel
- [ ] A6: `sync.Cond` 替换忙等循环，CPU 使用率下降
- [ ] B1: `utils.go` 存在且包含所有通用函数，56 处引用更新完毕
- [ ] B2: `extractFloatSlice` 和 `extractFloat64Slice` 合并为统一函数
- [ ] C1: CandlestickPanel 拆分后每个文件 <400 行
- [ ] C2: 拆分后所有 K 线相关测试通过
- [ ] `go test ./internal/...` 全部通过
- [ ] `npx vitest run` 全部通过

## Risks / Trade-offs

- **Wash sale 修复**: 需要确认原始成本价的计算方式——是 FIFO 还是加权平均。当前 OMS 使用加权平均，wash sale 检测也应一致。
- **工具函数迁移**: 56 处 import 更新量大但机械化——可以用 `grep` + `sed` 批量替换。
- **CandlestickPanel 拆分**: 涉及 props/events 的重新设计，需要保持与父组件 `DockTab` 的接口兼容。
- **sync.Cond**: 需要确保 Signal/Broadcast 时机正确，否则可能引入新的死锁。

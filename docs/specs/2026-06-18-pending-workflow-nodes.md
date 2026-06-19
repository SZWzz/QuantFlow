# [待开发] 缺失工作流节点

> **Status**: PENDING — 后续开发
> **Proposal ref**: NEW_PROJECT_PROPOSAL.md §3.2 (节点类型体系), §4 (功能模块映射)
> **Priority**: 🟡 中

## Motivation

规划 200+ 节点，目前仅实现 54 个。缺失节点主要分布在 Alpha 因子、归因分析、研究分析、另类数据等类别。补齐节点是将 Terminal 功能全面 Workflow 化的关键。

## 缺失节点清单

### Alpha 因子 (规划 ~20, 现有 1, 缺失 ~19)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `alpha_zoo` | 450 因子包批量计算 | 🔴 |
| `gp_evolution` | 遗传规划因子挖掘 | 🟡 |
| `factor_ic` | IC/IR 分析 | 🔴 |
| `factor_rank` | 因子排序/分组 | 🟡 |
| `factor_decay` | 因子衰减检测 | 🟡 |
| `factor_turnover` | 因子换手率 | 🟢 |
| `factor_neutralize` | 因子中性化（市值/行业） | 🟡 |

### 归因分析 (规划 ~8, 现有 0, 缺失 ~8)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `brinson_attribution` | Brinson 归因（配置/选股/交互） | 🟡 |
| `factor_attribution` | 因子归因分解 | 🟡 |
| `industry_attribution` | 行业归因 | 🟢 |
| `risk_decomposition` | 风险分解（系统/特质） | 🟡 |
| `performance_summary` | 绩效归因报告 | 🟢 |

### 交易执行 (规划 ~20, 现有 6, 缺失 ~14)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `twap_execution` | TWAP 执行算法 | 🟡 |
| `vwap_execution` | VWAP 执行算法 | 🟡 |
| `iceberg_order` | 冰山订单 | 🟢 |
| `basket_order` | 篮子交易 | 🟡 |
| `pair_trade` | 配对交易 | 🟡 |
| `arbitrage` | 套利检测 | 🟢 |
| `oco_order` | OCO 订单 (One-Cancels-Other) | 🟡 |
| `trailing_stop` | 追踪止损 | 🟡 |

### 风控分析 (规划 ~8, 现有 3, 缺失 ~5)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `stress_test` | 压力测试（历史场景/蒙特卡洛） | 🟡 |
| `var_analysis` | VaR 分解 | 🟡 |
| `greeks` | 期权 Greeks 分析 | 🟢 |
| `margin_check` | 保证金检查 | 🟡 |
| `concentration` | 集中度检测 | 🟢 |

### AI 智能体 (规划 ~10, 现有 1, 缺失 ~9)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `multi_agent` | 多智能体协作 | 🟡 |
| `debate_agent` | 多智能体辩论 (Bull vs Bear) | 🟡 |
| `report_agent` | 报告生成智能体 | 🟢 |
| `code_gen_agent` | 策略代码生成 | 🟡 |

### 市场数据 (规划 ~18, 现有 1, 缺失 ~17)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `market_screener` | 市场筛选器 | 🟡 |
| `market_regime` | 市场状态检测 | 🟡 |
| `option_chain` | 期权链 | 🟡 |
| `order_book` | 深度行情 | 🟢 |
| `tick_data` | Tick 级数据 | 🟢 |

### 投资组合 (规划 ~12, 现有 3, 缺失 ~9)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `black_litterman` | Black-Litterman 优化 | 🟡 |
| `mean_variance` | 均值-方差优化 | 🟡 |
| `risk_parity` | 风险平价 | 🟡 |
| `monte_carlo_sim` | 蒙特卡洛模拟 | 🟡 |
| `optimizer` | 组合优化器 | 🟡 |

### 控制流 (规划 ~8, 现有 5, 缺失 ~3)

| 节点 | 说明 | 优先级 |
|------|------|--------|
| `switch_case` | 多路分支 | 🟡 |
| `for_each` | 批量遍历 | 🟡 |
| `retry` | 错误重试 | 🟢 |

## 实现模式

所有节点遵循现有实现模式:
```go
type XxxNode struct {
    BaseNode
    Param1 string `json:"param1"`
}

func (n *XxxNode) NodeType() string { return "xxx" }
func (n *XxxNode) Category() string { return "xxx" }
func (n *XxxNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) { ... }
```

需在 `internal/workflow/nodes/register.go` 中注册:
```go
r.Register(&XxxNode{BaseNode: BaseNode{Name: "xxx"}})
```

## Acceptance Criteria

- [ ] 每个新节点有独立的 `.go` 文件，遵循现有命名/接口
- [ ] 注册到 `register.go`
- [ ] 有对应的 `_test.go` 文件
- [ ] 在 PropertyPanel 中参数可编辑
- [ ] 现有 251 Go 测试 + 76 前端测试通过

## 工作量估算

- Alpha 因子节点 (7): ~4 天
- 归因分析节点 (5): ~3 天
- 交易执行节点 (7): ~4 天
- 风控节点 (3): ~1.5 天
- AI 节点 (4): ~2 天
- 市场节点 (3): ~1.5 天
- 组合节点 (5): ~3 天
- 控制流 (3): ~1 天
- 测试: ~3 天
- **合计: ~23 天**

## Risks / Trade-offs

- 优先级按 🔴🟡🟢 排列，可分批实现
- 很多节点依赖 Python sidecar（AlphaZoo/归因等），需同时开发 Python 端

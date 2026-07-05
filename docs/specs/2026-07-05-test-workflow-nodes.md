# Test Coverage: Workflow Nodes (97 files, currently 14 tests)

## Motivation

`internal/workflow/nodes` 是项目最大的包（97 个源文件），但只有 14 个测试文件，覆盖率约 14%。用户可以将任意节点拖到画布上串联执行，节点间通过 `map[string]any` 传递数据 —— 没有编译期类型检查。每个节点的 `Execute` 方法都需要独立的测试保证：

1. **输入验证** — 缺失必要输入返回清晰错误
2. **正常路径** — 给定正确输入，输出符合预期
3. **边界条件** — 空输入、零值、极端值不 panic

## Design

### 分类策略

97 个节点按模式分类，同类节点共享测试模板：

#### A 类：技术指标（26 个）

`ema`, `sma`, `macd`, `rsi`, `bollinger`, `kdj`, `cci`, `atr`, `dmi`, `obi`, `mfi`, `sar`, `wr`, `roc`, `psy`, `bias`, `brar`, `mass`, `aroon`, `asi`, `bbi`, `vwap`, `std_dev`, `delta`, `pct_change`, `zhuoyao`

**测试模板**：给定已知输入序列 → 计算指标 → 与已知输出比较
```go
func TestEma_Execute(t *testing.T) {
    n := &EMANode{Period: 3}
    out, err := n.Execute(ctx, map[string]any{"values": []float64{1,2,3,4,5}})
    if err != nil { t.Fatal(err) }
    result, ok := out["ema"].([]float64)
    if !ok { t.Fatal("expected []float64 output") }
    if len(result) == 0 { t.Fatal("expected non-empty result") }
}
```

#### B 类：信号生成（5 个）

`cross_signal`, `cross_over`, `entry_signal`, `exit_signal`, `hold_signal`, `threshold_signal`

**测试模板**：输入两个序列 → 验证信号生成逻辑
```go
func TestCrossOver_Execute(t *testing.T) {
    n := &CrossOverNode{}
    out, err := n.Execute(ctx, map[string]any{
        "fast": []float64{1,2,3,4,5},
        "slow": []float64{5,4,3,2,1},
    })
    // 在交叉点应产生信号
}
```

#### C 类：数据获取（12 个）

`data_loader`, `eastmoney_data`, `news_fetcher`, `sentiment`, `financials`, `peer_compare`, `insider_trades`, `geopolitics`, `gov_data`, `prediction_market`, `satellite`, `stock_research`

**测试模板**：mock adapter → 验证 Execute 调用正确的 adapter 方法

#### D 类：逻辑控制（8 个）

`if_condition`, `if_else`, `compare`, `filter`, `merge`, `loop`, `wait`, `schedule`

**测试模板**：输入不同条件的值 → 验证分支/循环/等待逻辑

#### E 类：交易操作（8 个）

`place_order`, `cancel_order`, `order_query`, `position_query`, `position_sizer`, `stop_loss`, `rebalance`, `portfolio_summary`

**测试模板**：mock broker → 验证订单创建/查询/风控逻辑

#### F 类：ML/AI（6 个）

`train_model`, `predict`, `evaluate_model`, `rl_train`, `rl_predict`, `rl_env`, `factor`, `feature_engineer`, `alpha_mining`, `risk_model`

**测试模板**：mock ML client → 验证输入参数传递和输出解析

#### G 类：工具节点（22 个）

`arithmetic`, `math_op`, `scale`, `resample`, `rank`, `rank_select`, `rolling_zscore`, `rolling_maxmin`, `log_output`, `notify`, `alert`, `webhook_trigger`, `chart_data`, `signal_combine`, `bool_combine`, `agent`, `sub_workflow`, `strategy`, `json_parse`, `http_request`, `allocation`, `risk_metrics`

**测试模板**：给定输入 → 验证输出正确性

#### H 类：已有测试的（14 个）

不重复覆盖，但检查现有测试是否覆盖了边界条件。

### 测试总数目标

- 97 个节点 × 平均 2 个测试/节点 = ~194 个测试
- 覆盖所有节点的正常路径 + 错误路径

## Acceptance Criteria

- [ ] 每个节点至少 2 个测试（正常路径 + 缺失输入错误路径）
- [ ] 技术指标节点用已知序列验证计算精度
- [ ] 逻辑控制节点覆盖 true/false/空输入分支
- [ ] 交易操作节点验证错误输入不 panic
- [ ] `go test ./internal/workflow/nodes/... -count=1` 全部通过
- [ ] 节点包总行覆盖率 > 60%

## Risks / Trade-offs

- 97 个节点 × 2 测试 ≈ 194 个测试，预计 4-6 小时工作量。可分批执行：A 类指标节点（26 个）批量相似，用代码生成或参数化测试。
- 部分节点（如 `rl_train`）依赖 Python gRPC，执行需要 sidecar。测试设计为：Python 不可用时返回 `ErrSidecarNotAvailable` 错误路径。
- `http_request` 节点测试需要 mock HTTP server（`httptest.NewServer`），避免真实网络调用。

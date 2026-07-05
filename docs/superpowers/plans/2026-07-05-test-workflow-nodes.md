# 实施计划：Workflow Nodes 测试全覆盖

参考：`docs/specs/2026-07-05-test-workflow-nodes.md`

## 策略

97 个节点按 A-G 类分组，同类用参数化测试模板。测试文件命名为 `*_test.go` 与源文件一一对应。

## A 类：技术指标（26 个文件）

26 个指标节点共用同一模板，用已知输入验证计算结果。

**参数化模板**：在 `indicator_test_util.go` 中定义：

```go
package nodes

import (
    "context"
    "testing"
)

type indicatorTestCase struct {
    name    string
    node    Node
    inputs  map[string]any
    check   func(t *testing.T, output map[string]any)
}

func runIndicatorTest(t *testing.T, tc indicatorTestCase) {
    t.Helper()
    out, err := tc.node.Execute(context.Background(), tc.inputs)
    if err != nil {
        t.Fatalf("%s: unexpected error: %v", tc.name, err)
    }
    tc.check(t, out)
}
```

示例 — `ema_test.go`：
```go
func TestEMANode_Execute(t *testing.T) {
    runIndicatorTest(t, indicatorTestCase{
        name: "basic EMA",
        node: &EMANode{Period: 3},
        inputs: map[string]any{"values": []float64{1,2,3,4,5}},
        check: func(t *testing.T, out map[string]any) {
            ema, ok := out["ema"].([]float64)
            if !ok { t.Fatal("expected []float64") }
            if len(ema) == 0 { t.Fatal("expected non-empty") }
        },
    })
}
```

```go
func TestEMANode_MissingInput(t *testing.T) {
    n := &EMANode{Period: 3}
    _, err := n.Execute(context.Background(), map[string]any{})
    if err == nil {
        t.Error("expected error for missing 'values' input")
    }
}
```

**覆盖的 26 个文件**：`ema_test.go`, `sma_test.go`, `macd_test.go`, `rsi_test.go`, `bollinger_test.go`, `kdj_test.go`, `cci_test.go`, `atr_test.go`, `dmi_test.go`, `obi_test.go`, `mfi_test.go`, `sar_test.go`, `wr_test.go`, `roc_test.go`, `psy_test.go`, `bias_test.go`, `brar_test.go`, `mass_test.go`, `aroon_test.go`, `asi_test.go`, `bbi_test.go`, `vwap_test.go`, `std_dev_test.go`, `delta_test.go`, `pct_change_test.go`, `zhuoyao_test.go`

## B 类：信号生成（5 个文件）

测试信号生成的逻辑正确性：

`cross_signal_test.go`、`cross_over_test.go`、`entry_signal_test.go`、`exit_signal_test.go`、`hold_signal_test.go`、`threshold_signal_test.go`

## C 类：数据获取（12 个文件）

用 nil adapter 测试 mock fallback：

`data_loader_test.go`、`news_fetcher_test.go`、`sentiment_test.go`、`financials_test.go`、`peer_compare_test.go`、`insider_trades_test.go`、`geopolitics_test.go`、`gov_data_test.go`、`prediction_market_test.go`、`satellite_test.go`、`stock_research_test.go`

## D 类：逻辑控制（8 个文件）

`if_condition_test.go`、`if_else_test.go`、`compare_test.go`、`filter_test.go`、`merge_test.go`、`loop_test.go`、`wait_test.go`、`schedule_test.go`

## E 类：交易操作（8 个文件）

`place_order_test.go`、`cancel_order_test.go`、`order_query_test.go`、`position_query_test.go`、`position_sizer_test.go`、`stop_loss_test.go`、`rebalance_test.go`、`portfolio_summary_test.go`

## F 类：ML/AI（6 个文件）

`train_model_test.go`、`predict_test.go`、`evaluate_model_test.go`、`rl_train_test.go`、`rl_predict_test.go`、`rl_env_test.go`、`factor_test.go`、`feature_engineer_test.go`、`alpha_mining_test.go`、`risk_model_test.go`

## G 类：工具（22 个文件）

`arithmetic_test.go`、`math_op_test.go`、`scale_test.go`、`resample_test.go`、`rank_test.go`、`rank_select_test.go`、`rolling_zscore_test.go`、`rolling_maxmin_test.go`、`log_output_test.go`、`notify_test.go`、`alert_test.go`、`webhook_trigger_test.go`、`chart_data_test.go`、`signal_combine_test.go`、`bool_combine_test.go`、`agent_test.go`、`sub_workflow_test.go`、`strategy_test.go`、`json_parse_test.go`、`http_request_test.go`、`allocation_test.go`、`risk_metrics_test.go`

## 验证

```bash
go test ./internal/workflow/nodes/... -v -count=1
go test ./internal/workflow/nodes/... -race -count=1
```

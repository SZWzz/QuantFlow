# Phase 8: Workflow Node Expansion (20 → 34)

## Motivation
当前 20 个节点覆盖基础场景，扩展到 34 个支撑完整量化策略开发。

## Design

### 新增 14 个节点

**Indicator (4)**: MACD, RSI, BollingerBands, EMA
**Data (3)**: Merge, Filter, Resample
**Signal (2)**: ThresholdSignal, SignalCombine
**Risk (2)**: StopLoss, PositionSizer
**Utility (3)**: HTTPRequest, MathOperation, JSONParse

### 文件
```
internal/workflow/nodes/
├── macd.go, rsi.go, bollinger.go, ema.go
├── merge.go, filter.go, resample.go
├── threshold_signal.go, signal_combine.go
├── stop_loss.go, position_sizer.go
├── http_request.go, math_op.go, json_parse.go
└── register.go (modify: +14)
```

### Acceptance Criteria
- [ ] 14 新节点注册，总数 34
- [ ] go build + test 通过

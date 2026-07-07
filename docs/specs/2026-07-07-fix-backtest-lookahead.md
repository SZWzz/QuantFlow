# Fix Backtest Look-Ahead Bias

## Motivation

`SignalFunc` in both `engine_cn.go` and `engine_us.go` receives the full OHLCV bar, including Close/High/Low which are future information at the time of open-price execution. A strategy can trivially cheat by checking `bar.Close > bar.Open` before deciding to buy "at the open". This invalidates all backtest results.

## Design

### Data flow change

```
Before: SignalFunc(bar OHLCV, portfolio) → signal
After:  SignalFunc(openPrice, prevBar *OHLCV, portfolio) → signal
```

The signal function only receives:
- `openPrice` (当前 bar 的开盘价，唯一可用的实时价格)
- `prevBar` (前一完整 bar，用于计算指标如均线、RSI)
- `portfolio` (持仓信息，不变)

### Modified files

1. **`internal/backtest/types.go`** — Change `SignalFunc` type signature:
   ```go
   type SignalFunc func(openPrice float64, prevBar *OHLCV, portfolio Portfolio) Signal
   ```

2. **`internal/backtest/engine_cn.go`** (lines 145-147) — Pass only `bar.Open` and `prevBar` instead of full `bar`

3. **`internal/backtest/engine_us.go`** (lines 150-152) — Same fix

4. **`internal/backtest/engine_cn_test.go`**, **`engine_us_test.go`** — Update all test callbacks to match new signature

5. **`internal/strategy/`** — Update all strategy implementations (e.g., `sma_cross.go`, `macd_strategy.go`) to use `prevBar` for indicator computation

### API changes

- `SignalFunc` signature changes — **breaking change** for all strategy implementations
- No new exported functions, no gRPC changes, no Pinia store changes

## Acceptance Criteria

- [ ] `SignalFunc` no longer receives current bar's Close/High/Low
- [ ] All existing strategy implementations compile with new signature
- [ ] Existing backtest tests pass with same numerical results (or explainable differences)
- [ ] All Go tests pass (`cd app && go test ./...`)

## Risks / Trade-offs

- **Breaking change**: All strategy callbacks must be updated. This is intentional — we want to force strategy authors to write correct code.
- **Slight performance impact**: Every bar iteration now needs to track `prevBar`. Negligible (single pointer copy).
- **Still possible to cheat**: A strategy could cache bars in a package-level variable. This is a contract fix, not a sandbox. Future work: run strategies in a subprocess with no shared state.

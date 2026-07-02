# Multi-Day Minute Data Fix

## Motivation
多日分时（multi-day minute line）在 K 线面板中显示"无数据"。根因是 TDX MAC TCP 直连不可达且无 fallback，加上 dayDate 硬编码、avg_price 未计算。

## Design

### Data Flow Before
```
Frontend → app.GetMultiDayMinute("600519", 3)
  → macAdpt.GetMultiDayMinute()   // TCP 7709, 7 IPs, 无 fallback
  → 若 TCP 不通 → error → [] → 无数据
```

### Data Flow After
```
Frontend → app.GetMultiDayMinute("600519", 3)
  → try macAdpt.GetMultiDayMinute()       // 快路径
  → if err or empty:
      → mootdxAdpt.FetchMultiDayMinute()  // fallback via Python sidecar
         → dataClient.FetchData{source:"mootdx", data_type:"multi_minute"}
         → _fetch_mootdx_multi_minute()
            → client.bars(frequency='1m') for each day
            → convert 1m OHLCV → MinuteTick {time, price=close, volume, avg_price}
```

### Changes

1. **`internal/market/adapters/mac.go`** — Fix dayDate parsing (epoch 1990-12-19), compute avg_price, add connection error logging
2. **`python/src/data/fetcher.py`** — Add `_fetch_mootdx_multi_minute()`, route `multi_minute` in `_handle_mootdx`
3. **`internal/market/adapters/mootdx_minuteline.go`** — Add `FetchMultiDayMinute()`
4. **`app_research.go`** — Add mootdx fallback when MAC fails

## Acceptance Criteria
- [ ] MAC 路径可用时走 MAC（带正确日期和均价）
- [ ] MAC 不可用时自动 fallback 到 mootdx/Python 侧车
- [ ] 前端日期下拉正确显示实际交易日
- [ ] 均价线正常绘制
- [ ] 构建通过

## Risks / Trade-offs
- Python sidecar 必须运行才能 fallback；若侧车也挂了，仍无数据
- 1m OHLCV 的 close 近似分时价格，非精确逐笔均价，有微小偏差

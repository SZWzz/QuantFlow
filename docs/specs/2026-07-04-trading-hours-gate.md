# Trading Hours Gate — 非交易时段跳过行情请求

## Motivation

非交易时段（如 A 股午盘休市 11:30-13:00、夜间、周末），`FetchQuoteWithFallback` 仍然遍历整个适配器链并逐一重试，导致：

1. mootdx 等第一优先适配器反复报 `"no quote data"` WARN 日志，干扰问题排查
2. 适配器链逐个重试耗时 10-15 秒，前端挂起等待后收到空数据
3. 无意义地对 TDX / 新浪等免费数据源产生无效请求，有被限流风险

## Design

在 `internal/market` 包中新增 `trading_hours.go`，提供 `IsTradingHours(market string) bool` 函数。

然后在 `registry.go` 的 `FetchQuoteWithFallback` 入口处检查：若当前市场不在交易时段，直接返回错误而不遍历适配器链。`FetchOHLCVWithFallback` 不受影响（历史 K 线非交易时段也能拉取）。

### 交易时段定义

| Market | Session | Time (Local) |
|--------|---------|-------------|
| CN | 上午 | 09:30 – 11:30 |
| CN | 下午 | 13:00 – 15:00 |
| HK | 上午 | 09:30 – 12:00 |
| HK | 下午 | 13:00 – 16:00 |
| US | 盘前 | 04:00 – 09:30 ET |
| US | 盘中 | 09:30 – 16:00 ET |
| CRYPTO | 全天 | 24/7 |

- 所有市场仅周一至周五判断交易时段
- CN/HK 以北京时间（UTC+8）判断
- US 以美东时间（UTC-4/UTC-5）判断
- CRYPTO 始终返回 `true`

### 数据流

```
FetchQuoteWithFallback(ctx, market, symbol)
  │
  ├── !IsTradingHours(market) → return error "market closed"
  │
  └── (原有逻辑) 检查缓存 → 遍历适配器链
```

### 修改文件

- `internal/market/trading_hours.go` — 新增，`IsTradingHours()` 实现
- `internal/market/registry.go` — `FetchQuoteWithFallback` 入口处增加 `IsTradingHours` 检查

### 不变

- `FetchOHLCVWithFallback` — 不检查交易时段，历史 K 线随时可取
- 各适配器本身的 `IsAvailable()` — 不变
- `FetchQuote` 在单个适配器内的行为 — 不变
- 前端调用方式 — 不变，只是更快返回 "market closed" 错误

## Acceptance Criteria

- [ ] A 股交易时段内（周一至周五 9:30-11:30 或 13:00-15:00）行情请求正常走适配器链
- [ ] A 股非交易时段（午休、夜间、周末）`FetchQuoteWithFallback` 立即返回 "market closed" 不进入适配器链
- [ ] 美股/港股按各自时区判断
- [ ] 加密货币始终可通过
- [ ] OHLCV 历史 K 线请求不受交易时段限制
- [ ] 已有测试全部通过

## Risks / Trade-offs

- 系统时间必须正确，否则判断错误。桌面应用场景可接受。
- US 夏令时/冬令时转换：用 `time.LoadLocation("America/New_York")` 自动处理。
- 节假日（春节、国庆、圣诞）不处理——这些日子适配器本身会返回错误，不会产生额外 WARN（因为交易时段检查会通过，但适配器链仍会尝试并失败）。如需静默节假日需引入交易日历，复杂度太高暂不纳入。

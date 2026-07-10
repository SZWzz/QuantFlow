# OHLCV Fixes: Data Range & Cache Wiring

## Motivation

两个用户反馈的 K 线问题：
1. **数据范围不足** — 指数 K 线（000001.SH 等）只返回 90 天数据，远少于之前的全历史
2. **拉取速度慢** — mootdx 在 CN 链条排第一，但对指数符号每次都 timeout，叠加 `RequestCtx` 10s 超时后所有 adapter cascading 失败

## Design

### 1. 扩展指数 K 线数据范围

`app_market.go:585` 把 `now.AddDate(0, 0, -90)` 改为 `now.AddDate(-10, 0, 0)`（10 年），配合腾讯 API 的 2000 条限制取最大可用历史。对于 2005 年后上市的指数（如 000688.SH 科创50），远端返回多少算多少。

### 2. 接线 OHLCVCache

`internal/market/ohlcv_cache.go` 的 `OHLCVCache`（SQLite + LRU 双层）是已经写好的死代码，从未被初始化。在 `app_startup.go` 初始化并挂到 `App` 上，在 `FetchOHLCVWithFallback` 中先查缓存。

### 3. 优化 CN OHLCV adapter 链

- `FetchOHLCVWithFallback` 中加入缓存检查路径
- 将 `tencent` 提升为 CN 市场 OHLCV 的首选 adapter（mootdx 只有 quote 有价值，OHLCV 经常超时）
- `_fetch_mootdx_ohlcv` 增加指数符号的特殊处理（同 `_fetch_mootdx_minute` 的做法）

## Data Flow

```
Frontend request (FetchOHLCV or GetMarketOverview)
  → App.GetOHLCV (new) / GetMarketOverview
    → OHLCVCache.Get → hit? return
    → FetchOHLCVWithFallback (tencent first for CN)
      → adapter.FetchOHLCV
    → OHLCVCache.Set
    → return
```

## Files Modified

| File | Change |
|------|--------|
| `app_market.go:585` | `-90` → `-10*365` |
| `app_startup.go` | Init `OHLCVCache` |
| `internal/market/registry.go` | Add cache check in `FetchOHLCVWithFallback` |
| `internal/market/ohlcv_cache.go` | Minor: add `Has()` method |
| `internal/market/adapters/mootdx.go` | Add `Categories()` for OHLCV |
| `python/src/data/fetcher.py` | Index handling in `_fetch_mootdx_ohlcv` |

## Acceptance Criteria

- [ ] 指数 K 线返回 >5 年数据（10 年最佳）
- [ ] 第二次请求同一指数 OHLCV 时走内存/SQLite 缓存，毫秒级返回
- [ ] CN 市场 OHLCV 不再因 mootdx timeout 而延迟
- [ ] 重新打包后可正常运行

## Risks / Trade-offs

- 10 年数据多拉一次不影响：`indexOHLCVCache` 已有 60s TTL，不会频繁重复拉取
- tencent 第一顺位可能在某些网络环境下不可用（境外用户），但原 fallback 链保留，tencent 失败后走 eastmoney → baidu → akshare

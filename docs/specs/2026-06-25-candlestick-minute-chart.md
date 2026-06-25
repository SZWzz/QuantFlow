# Spec: CandlestickPanel 分时图 Tab + K线实时刷新

> 日期: 2026-06-25
> 状态: approved
> 策略: 策略一（CandlestickPanel 内嵌 tab 切换）

## 目标

在 CandlestickPanel 内增加「分时图」tab，与「K线」tab 切换，实现同花顺式股票详情面板。分时图显示当日分钟走势线 + 成交量，每 10s 自动刷新。

## 布局

```
┌──────────────────────────────────────────┐
│ [600519]  [📈 K线] [⏱ 分时]  1m 5m 15m 1h 1d 1w │
├──────────────────────────────────────────┤
│                                          │
│     ECharts 图表                          │
│     - K线 tab: 蜡烛图 + MA                 │
│     - 分时 tab: 走势折线 + 成交量柱 + 昨收线  │
│                                          │
├──────────────────────────────────────────┤
│     成交量柱 (K线 tab) / 指标 (分时 tab)     │
└──────────────────────────────────────────┘
```

## 数据结构

### 分时 tick

```ts
interface MinuteTick {
  time: string       // "09:35"
  price: number      // 该分钟成交均价
  volume: number     // 该分钟成交量
  avgPrice: number   // 日内均价（用于均线）
}
```

### Go 后端新增 API

```go
// GetMinuteLine returns today's minute-line ticks for a CN symbol.
// Falls through the adapter chain; mootdx is the primary provider.
func (a *App) GetMinuteLine(ctx context.Context, symbol string) ([]MinuteTick, string, error)
```

`MinuteTick` 结构体在 `internal/market/types.go` 中新增。mootdx 适配器通过 `FetchData(data_type="quote")` 获取原始分钟线，Go 侧做标准化。

### 前端数据流

```
分时 tab 激活时（仅当 symbol.endsWith('.SZ' | '.SH') 或 6 位纯数字）:

1. 首次加载: App.GetMinuteLine('CN', symbol) → 全量分钟 tick
2. 每 10s 轮询: App.GetMinuteLine('CN', symbol) → 增量合并新 tick
3. 兜底: 如 mootdx 不可用，回退到 GetQuote('CN', symbol) 追加当前价
```

K线 tab 分钟级 interval 时（1m/5m/15m/30m/60m）：每 30s 调 FetchOHLCV 刷新最后几根 bar。

## 组件改动

### CandlestickPanel.vue

**新增状态**：
- `activeTab: ref<'kline' | 'minute'>` — 默认 `'kline'`
- `minuteTicks: ref<MinuteTick[]>` — 分时数据
- `prevClose: ref<number>` — 昨收价（用于分时图参考线）
- `minuteTimer` — 10s 轮询计时器

**分时图 ECharts 配置**：
- 双 Y 轴：左轴价格、右轴成交量
- 走势线颜色：最新价 >= 昨收 → `#ef4444` (红，涨)；否则 `#22c55e` (绿，跌)
- 昨收参考线：水平虚线 `#6b7280`
- 成交量柱颜色：对应分钟涨跌
- 横轴时间范围：09:30 ~ 15:00
- 支持 tooltip、十字光标

**K线 tab 实时刷新**：
- `watch(interval)`: 分钟级 interval → 启动 30s 定时器调 `loadOHLCV()`；日线以上 → 清除定时器
- `onUnmounted`: 清除所有定时器

### Go 后端

**app.go**: 新增 `GetMinuteLine(ctx, symbol)` 方法
**internal/market/adapters/mootdx.go**: 新增 `FetchMinuteLine(ctx, symbol)` → 调用 Python sidecar `FetchData(data_type="quote")`
**internal/market/types.go**: 新增 `MinuteTick` 结构体

### Python sidecar

`fetcher.py` 已有 `_fetch_mootdx_quote()` 和 gRPC `FetchData` 的 `data_type="quote"` 通路，无需改动。

## 边界情况

| 场景 | 处理 |
|------|------|
| 非交易时段（盘前/盘后/周末） | mootdx minute() 返回空 → 显示"暂无分时数据" + 昨收水平线 |
| mootdx sidecar 不可用 | 显示"通达信未连接，分时数据不可用" + 回落按钮 |
| symbol 变化 | 清空 tick 数据 + 复拉 |
| 当天首次加载（无历史 tick） | 等待 sidecar 响应，显示 loading 骨架 |
| 盘中断开重连 | mootdx 自动重试服务器，Go 侧 catch error 后标记 `minuteUnavailable` |

## 验收标准

- [ ] CandlestickPanel 顶部有 K线/分时 两个 tab 可切换
- [ ] 分时图显示当日分钟走势线 + 成交量 + 昨收参考线
- [ ] 分时图每 10s 自动刷新，走势线实时追加
- [ ] K线 tab 分钟级 interval 每 30s 刷新最后 bar
- [ ] K线 tab 日线及以上 interval 不自动刷新
- [ ] mootdx 不可用时显示友好提示
- [ ] vue-tsc + go build 通过

# Minute Chart Professional Features

## Motivation
分时图当前落后同花顺专业版，缺少三个关键功能：
1. **十字线浮窗** — 鼠标悬停显示时间/价格/均价/成交量/成交额
2. **量比副图** — 量比(Volume Ratio)是 A 股分时图标准指标
3. **底部指标扩展** — 分时图仅支持 volume/MACD/KDJ，而 K 线图有 7 种

## Design

### 1. 十字线浮窗 (Crosshair Tooltip)
**现状**: `buildMinuteOption` 已有 tooltip formatter（行 336-347），但只显示 series 名称+值，缺少成交额。

**改动**: 增强 tooltip formatter，在分时图显示：
- 时间 / Time
- 价格 / Price (当前价, 涨跌额, 涨跌幅)
- 均价 / Avg Price
- 成交量 / Volume (手)
- 成交额 / Amount (万元)

成交额需要从 Python 侧透传或前端计算 `price × volume`。Python 侧 `_fetch_mootdx_minute` 新增 `amount` 字段。

### 2. 量比副图 (Volume Ratio)
量比 = 当前分钟成交量 / 过去 5 日同分钟平均成交量。

**数据来源**: 量比需要 5 日历史分钟数据，当前 minute 接口只返回当日。新增 `GetVolumeRatio` Go 接口：
- 取过去 5 个交易日同 symbol 的分时数据
- 按分钟对齐，计算历史平均
- 当前分钟量比 = volume / 历史同分钟均值

**前端**: 新增 `minuteBottomMode` 选项 `'vr'`，在 `buildMinuteOption` 中渲染量比柱状图（红色 >1，绿色 <1）。

### 3. 底部指标扩展
`minuteBottomMode` 从 `<'volume' | 'macd' | 'kdj'>` 扩展为 `<'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv'>`。

复用 K 线图已有的 `rsi`/`wr`/`cci`/`obv` 指标函数（`useIndicators.ts`）。

### 4. 成交额字段透传
`MinuteTick` 新增 `amount` 字段，Python fetcher 计算 `p * v` 作为每笔成交额。
Go 层 `MinuteTick`、SQLite `minute_cache`、Go adapter 同步更新。

## Files Changed
| File | Change |
|------|--------|
| `python/src/data/fetcher.py` | `_fetch_mootdx_minute`: 新增 `amount` 字段 |
| `frontend/src/lib/buildChartOption.ts` | `buildMinuteOption`: 增强 tooltip + 量比副图 + 指标扩展 |
| `frontend/src/terminal/panels/CandlestickPanel.vue` | `minuteBottomMode` 类型扩展 + 新增按钮 |
| `internal/market/minuteline.go` | `MinuteTick` 新增 `Amount` 字段 |
| `internal/market/minute_cache.go` | SQLite DDL + 读写 `amount` 列 |
| `internal/market/adapters/mootdx_minuteline.go` | `rawMinuteTick` 透传 amount |
| `CHANGELOG.md` | 更新条目 |

## Acceptance Criteria
- [ ] 分时图 tooltip 显示时间/价格/涨跌/均价/成交量/成交额
- [ ] 量比副图可用，红绿柱区分高于/低于均值
- [ ] 分时图底部支持 RSI/WR/CCI/OBV
- [ ] build 通过，Python 测试通过

## Risks / Trade-offs
- 量比精度受限于 5 日历史数据可用性（非交易日/新股不足 5 日则降级为 1x）

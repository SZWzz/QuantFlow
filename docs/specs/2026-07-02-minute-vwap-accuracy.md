# Minute Chart VWAP (均价线) Accuracy Fix

## Motivation
分时图的均价线（黄色虚线）当前使用迭代 VWAP 公式 `cum_avg = (cum_avg * cum_vol + p * v) / (cum_vol + v)`，存在两个精度问题：
1. 首个 tick volume=0 时错误地将 `avg_price` 设为该 tick 价格
2. 累积除法传播浮点误差

正确的均价线应为：**累计成交额 / 累计成交量**。

## Design
**修改文件**: `python/src/data/fetcher.py` 中 `_fetch_mootdx_minute()`

**改动**: 用直接累计法替代迭代 VWAP：
- 追踪 `cum_amount`（累计成交额）= sum(price × volume)
- 追踪 `cum_vol`（累计成交量），仅在 volume > 0 时更新
- 计算 `avg_price = cum_amount / cum_vol`（当 cum_vol > 0），否则为 0

TDX 协议分时数据不含成交额字段，`price × volume` 是行业标准代理。

**数据流**: Python fetcher.py → JSON → Go adapter → SQLite minute_cache → Vue buildChartOption → ECharts

## Acceptance Criteria
- [ ] VWAP 公式改为 `cum_amount / cum_vol`
- [ ] Volume=0 的 tick 不更新 cum_amount/cum_vol，avg_price 沿用上一值
- [ ] `wails3 build` 通过
- [ ] Python 测试通过

## Risks / Trade-offs
- 仍使用 `price * volume` 近似成交额，而非真实逐笔成交额。这是免费数据源的根本限制。

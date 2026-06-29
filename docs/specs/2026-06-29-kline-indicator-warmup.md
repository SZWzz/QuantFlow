# K线指标预热数据不足修复

## Motivation

日K线图默认回溯 90 个自然日，仅能获取约 58 个交易日数据。MA60 需要 60 根 K 线才能算出第一个值，导致 MA60 在整个图表上**完全没有有效值**。MA20/布林带/MACD/KDJ/RSI 等虽然在尾部有值，但前段大片空白，给用户"指标无法完整生成"的观感。

周K线（1w）同样存在类似问题：回溯 180 天 ≈ 26 根周K线，MA60 需要 60 周K线才能开始计算，不满足。

## Design

### Root Cause

`frontend/src/terminal/panels/CandlestickPanel.vue:114` 中 `lookbackDays` 计算：

```javascript
const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 180 : 90
```

各间隔的回溯不足：
- **日K（1d）**：90 自然日 → ~58 交易日 < MA60（需 60 天）
- **周K（1w）**：180 自然日 → ~26 周K线 < MA60（需 60 周 ≈ 420 天）

### Data Flow

```
CandlestickPanel.vue (lookback: 日K=365, 周K=450)
  → app.FetchOHLCV(symbol, "1D"/"1W", start, end)
    → AdapterRegistry.FetchOHLCVWithFallback
      → MootdxAdapter.FetchOHLCV  (首试，支持分页到 80,000 根)
        → Python sidecar mootdx → TDX TCP (10年+数据)
```

国产数据源（mootdx/东方财富/腾讯）均能轻松返回所需数据量：
- **Mootdx**: 80,000 根上限，分页获取
- **东方财富**: 无限制，日期范围返回
- **腾讯**: 上限 2,000 根，365 天日K / 450 天周K均远低于此限制

### Changed Files

| File | Change |
|------|--------|
| `frontend/src/terminal/panels/CandlestickPanel.vue:114` | 日K `90` → `365`，周K `180` → `450` |

### Not Changed

- 分钟级间隔（`1m/5m/15m/30m/1h`）：保持 5 天回溯，分时数据回取更长时间无意义（数据量大且大部分数据源不提供历史分时）
- Go 后端代码：无修改
- SQLite/缓存：无修改

## Acceptance Criteria

- [ ] 日K默认视图下 MA60 线条能完整显示（前 59 根 K 线无值属正常预热行为，第 60 根起有值即代表修复成功）
- [ ] 周K（1w）视图下 MA60 能有足够预热数据
- [ ] MA5/MA10/MA20/布林带/MACD/KDJ/RSI/WR 在日K下前段空白显著缩短
- [ ] 数据加载性能无明显下降（日K 365 根 / 周K ~64 根，对 echarts 无压力）

## Risks / Trade-offs

- **性能**: 365 根日K / 64 根周K数据点约 2KB JSON，对浏览器和 echarts 无任何压力
- **首次加载**: 数据略有增加，用户无感知
- **新股**: 上市不足回溯期的股票，数据源返回自上市日起的数据，不影响显示
- **分钟级间隔不调整**: 分时数据回取 365 天数据量巨大（约 80,000 根 1m K线），且大部分数据源不提供历史分时，保持现状
- **替代方案**: ① 自适应计算最小回溯（根据当前选中的指标动态计算）—— 过度设计；② 首次加载长回溯，之后增量更新 —— 可行但当前初始化逻辑已足够

# Market Overview 指数 K 线图

## Motivation

Market Overview 面板顶部展示 5 (CN) / 3 (HK) / 3 (US) 个主要指数卡片，目前使用 SVG 折线 sparkline（仅收盘价连线）且 sparkline 数组实际为空。用户希望在指数卡片中看到真实的 K 线图，以更专业的方式判断指数趋势。

## Design

### Data Flow

```
Go GetMarketOverview(mkt)
  │
  ├── 现有: 并行 FetchQuote → {price, change, change_pct}
  │
  └── 新增: 并行 FetchOHLCV(mkt, code, "1D", "", now-60d, now)
        │
        └── fallback 链: mootdx → tencent → eastmoney → baidu → akshare
              → EastMoney 实测支持指数 secid (1.000001 / 0.399001)
              → fqt=0 (不复权) 返回指数原始 OHLC
              → 返回最近 60 根日 K，前端取最后 30 根渲染

Frontend fetchMarketOverview()
  │
  ├── mapping 中透传 idx.ohlcv → IndexSnapshot.ohlcv[]
  │
  └── PanelCard 渲染:
        ├── 优先: SVG mini candlestick (20-30 candles)
        └── fallback: 原折线 sparkline (ohlcv 为空时)
```

### Modified Files

| File | Change |
|------|--------|
| `app_market.go` | `GetMarketOverview` 中每个 index goroutine 增加 `FetchOHLCV` 调用；result map 追加 `"ohlcv"` 字段 |
| `frontend/src/stores/data.ts` | `IndexSnapshot` 新增 `ohlcv?: OHLCVBar[]` |
| `frontend/src/stores/data.ts` | `fetchMarketOverview` mapping 透传 `idx.ohlcv` |
| `frontend/src/terminal/components/panel/PanelCard.vue` | 新增 `ohlcv` prop；template 中渲染 SVG mini candlestick（优先）或折线 sparkline（fallback） |

### No Changes To

- Python sidecar（无需改动）
- SQLite schema（无需存储，仅实时展示）
- CandlestickPanel（index overlay 已有独立 OHLCV 获取逻辑）
- i18n（不需新增文案）

### Mini SVG Candlestick Design

viewBox `100×30`，渲染最后 30 根日 K：

```
每根蜡烛:
  <line>       ← 上下影线 (high-low)
  <rect>       ← 实体 (open-close)
阳线: 实体 red, 影线 red
阴线: 实体 green, 影线 green
```

计算方法：先找 high/low 归一化到 viewBox 30px 高度，每个 candle 宽度 ≈ 3.3px，左边缘 `x = idx * 3.33`，实体宽度 2px，居中。

## Acceptance Criteria

- [ ] Market Overview (CN tab) 的 5 个指数卡片显示 mini candlestick 而非空白/折线
- [ ] HK / US tab 同理显示对应指数的 mini K 线
- [ ] 切换 tab 后 K 线正确更新
- [ ] 数据源不可用时（如网络断开），优雅 fallback 到折线 sparkline 或空白
- [ ] `wails3 build` 通过，`go vet ./...` 无警告

## Risks / Trade-offs

- **mootdx 可能不支持指数**: TDX 协议使用不同 category 区分指数/股票。但不影响最终渲染——fallback 链中 EastMoney 一定可用。
- **性能**: 为 8 个指数每人加一次 `FetchOHLCV`，但 goroutine 并行，整体 latency 与单次 OHLCV 请求相当（≈200–500ms）。加上现有 30s overviewCache，影响可控。
- **SVG 精度**: 30px 高度的 mini candle 只能显示趋势，无法精确读取 OHLC 值（卡片数字已显示最新价/涨跌幅，K 线仅作视觉参考）。
- **HK/US 指数**: `^HSI`、`^GSPC` 等符号可能不适用 EastMoney。这些指数的 OHLCV 通过 Yahoo 或 Finnhub fallback 获取（Yahoo adapter 支持 `^HSI` 格式）。

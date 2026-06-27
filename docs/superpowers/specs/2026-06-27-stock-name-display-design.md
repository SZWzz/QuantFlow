# 面板股票名称显示 — 设计文档

> 问题：21 个面板只显示股票代码（如 600519）不显示公司名称（贵州茅台），用户体验差。

## 根因

| 类别 | 根因 | 涉及面板 |
|------|------|---------|
| Go 数据类型缺字段 | Trade、Order、Position 结构体无 `Name` 字段 | PositionPanel、OrderBlotter、TradeHistory、ExecutionPanel、PortfolioSummary 等 |
| 前端未解析名称 | 面板只取 symbol 不做 name lookup | FinancialsPanel、SentimentPanel、AnalystEstimates、CandlestickPanel、ChanlunPanel 等 |

## 方案

### 一、Go 后端 — 数据类型补 Name 字段

| 结构体 | 文件 | 改动 |
|--------|------|------|
| `Trade` | `internal/trading/types.go` | 加 `Name string` |
| `Order` | `internal/trading/types.go` | 加 `Name string` |
| `Position` | `internal/trading/types.go` | 加 `Name string` |

**填充逻辑**：OMS 中已有 `symbol`，每次 `FillOrder` / `UpdateMarketPrice` 时从 `quoteCache`（新增轻量 map）读取名称填充。quoteCache 由 `GetQuote` 调用时自动写入。

### 二、前端 — `useStockName` composable

```typescript
// src/lib/composables/useStockName.ts
export function useStockName(symbol: Ref<string>) {
  const name = ref('')
  watch(symbol, async (sym) => {
    if (!sym) return
    try {
      const result = await (window as any).go?.main?.App?.GetQuote('CN', sym)
      if (result?.[0]?.name) name.value = result[0].name
    } catch { /* silent */ }
  }, { immediate: true })
  return { name }
}
```

### 三、面板改动范围

**Priority HIGH（7 个）**：
- PositionPanel — 表格加名称列
- PositionDetail — 标题显示名称
- PortfolioSummary — 持仓表加名称列
- OrderBlotterPanel — 订单表加名称列
- OrderEntryPanel — 标题显示名称
- TradeHistory — 成交/订单表加名称列
- ExecutionPanel — 成交表加名称列

**Priority MEDIUM（4 个）**：
- CandlestickPanel — 标题 `600519` → `600519 贵州茅台`
- DrawingPanel — 标题显示名称
- RebalancePanel — 交易列表加名称
- BasketOrderPanel — 表格加名称列

**Priority LOW（10 个）**：
- FinancialsPanel、AnalystEstimatesPanel、SentimentPanel、InsiderTradingPanel、StockResearchPanel、ChanlunPanel、IndicatorPanel、CorrelationPanel、NewsPanel、PeerComparisonPanel — 标题加名称

## 验证标准

1. 自选股面板显示代码+名称 ✅（已有）
2. 持仓表每行显示代码+名称
3. 订单表每行显示代码+名称
4. K线图标题显示代码+名称
5. 研究类面板标题显示代码+名称
6. npm run build + go build 通过

# 股票名称显示 — 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended).

**Goal:** 21 个面板从只显示代码 → 显示代码+名称。

**Architecture:** Go 端 Trade/Order/Position 加 Name 字段 + quoteCache 自动填充；前端新增 useStockName composable 统一解析。

**Tech Stack:** Go 1.22+, Vue 3 Composition API

## Global Constraints

- Go 结构体字段首字母大写（Wails JSON 序列化要求）
- 名称从 `GetQuote` 获取，不额外新增 API
- 面板改动最小化，仅加一列/一行

---

### Task 1: Go 后端 — 数据类型加 Name + quoteCache

**Files:**
- Modify: `internal/trading/types.go` — Trade/Order/Position 加 Name
- Modify: `internal/trading/oms.go` — OMS 加 quoteCache + 自动填充
- Modify: `app_market.go` — GetQuote 写入 quoteCache

- [ ] **Step 1: types.go 加字段**

```go
type Trade struct {
    // ... existing fields ...
    Name string `json:"name"`  // 股票名称
}

type Order struct {
    // ... existing fields ...
    Name string `json:"name"`
}

type Position struct {
    // ... existing fields ...
    Name string `json:"name"`
}
```

- [ ] **Step 2: OMS 加 quoteCache**

```go
type OMS struct {
    // ... existing fields ...
    quoteCache map[string]string  // symbol → name
}

func (o *OMS) SetQuoteName(symbol, name string) {
    o.mu.Lock()
    defer o.mu.Unlock()
    if o.quoteCache == nil {
        o.quoteCache = make(map[string]string)
    }
    o.quoteCache[symbol] = name
}

func (o *OMS) getQuoteName(symbol string) string {
    o.mu.RLock()
    defer o.mu.RUnlock()
    return o.quoteCache[symbol]
}
```

在 `PlaceOrder`/`FillOrder` 中调用 `getQuoteName` 填充 `order.Name`/`trade.Name`。

- [ ] **Step 3: GetQuote 写入缓存**

`app_market.go` 中 `GetQuote` 方法获取到 quote 后，写入 OMS quoteCache：
```go
if a.oms != nil && quote.Name != "" {
    a.oms.SetQuoteName(symbol, quote.Name)
}
```

- [ ] **Step 4: 构建验证**

```bash
go build -o /dev/null .
```

---

### Task 2: 前端 — useStockName composable

**Files:**
- Create: `frontend/src/lib/composables/useStockName.ts`

- [ ] **Step 1: 创建 composable**

```typescript
import { ref, watch, type Ref } from 'vue'

export function useStockName(symbol: Ref<string | undefined>) {
  const name = ref('')

  watch(symbol, async (sym) => {
    if (!sym) { name.value = ''; return }
    try {
      const app = (window as any).go?.main?.App
      if (!app) return
      const result = await app.GetQuote('CN', sym)
      const quote = Array.isArray(result) ? result[0] : result
      if (quote?.name) name.value = quote.name
    } catch { /* name resolution is best-effort */ }
  }, { immediate: true })

  return { name }
}
```

- [ ] **Step 2: 构建验证**

```bash
cd frontend && npm run build -q | tail -1
```

---

### Task 3: 面板批量改动 — HIGH priority（7 个）

**Files:** PositionPanel, PositionDetail, PortfolioSummary, OrderBlotterPanel, OrderEntryPanel, TradeHistory, ExecutionPanel

改动模式统一：
1. 表格每行：`<td>{{ pos.Symbol }}</td>` → `<td>{{ pos.Symbol }}<br/><small>{{ pos.Name || '--' }}</small></td>`
2. 标题区域：`{{ symbol }}` → `{{ symbol }} {{ name }}`

---

### Task 4: 面板批量改动 — MEDIUM priority（4 个）

**Files:** CandlestickPanel, DrawingPanel, RebalancePanel, BasketOrderPanel

---

### Task 5: 面板批量改动 — LOW priority（10 个）

**Files:** FinancialsPanel、AnalystEstimatesPanel、SentimentPanel、InsiderTradingPanel、StockResearchPanel、ChanlunPanel、IndicatorPanel、CorrelationPanel、NewsPanel、PeerComparisonPanel

对输入型面板：import useStockName，标题改用 `{{ symbol }} {{ name || '' }}`

---

### Task 6: 全栈打包验证

```bash
cd frontend && npm run build -q && cd .. && go build -o build/quantflow .
```

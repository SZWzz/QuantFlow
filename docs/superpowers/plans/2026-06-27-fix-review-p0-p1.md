# 评审 P0 问题修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复全面评审中发现的全部 9 个 P0 致命问题和 5 个关键 P1 问题

**Architecture:** 分 Phase 0（P0 立即修复，9 任务）和 Phase 1（P1 本轮修复，5 任务）。Go 后端修改遵循 TDD，Python 修改需配套测试，前端修改需 vite build 通过。

**Tech Stack:** Go 1.22, Vue 3 + TypeScript, Python 3.12, gRPC

**Spec:** docs/specs/fix-review-issues.md

## Global Constraints

- Go: `go.mod` version → `go 1.22`（非 1.26.4）
- 前端: `vite build` 不新增编译错误
- Python: `pytest` 所有测试必须通过
- 每个任务独立可测试，一任务一提交
- 统一版本号 `0.2.0` 写入 `python/pyproject.toml`

---

## Phase 0: P0 致命修复

### Task 1: 修复 OMS 自死锁 + go.mod 版本号

**Files:**
- Modify: `internal/trading/oms.go:389-393`
- Modify: `go.mod:3`

**Interfaces:**
- Consumes: None
- Produces: `getName` 不再持锁；`go.mod` 版本号正确

- [ ] **Step 1: 修复 getName 死锁**

```go
// internal/trading/oms.go:389-393
// Replace the existing getName method:
func (o *OMS) getName(symbol string) string {
    return o.quoteCache[symbol]
}
```

- [ ] **Step 2: 修复 go.mod 版本号**

```
go 1.22
```

- [ ] **Step 3: 运行测试**

```bash
cd /Volumes/etx/coding/rebuild/quantflow && go test ./internal/trading/ -run TestOMS -v -timeout 30s
```

Expected: PASS, No deadlock timeout

- [ ] **Step 4: 运行回测测试**

```bash
go test ./internal/backtest/ -v -timeout 30s
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/trading/oms.go go.mod
git commit -m "fix(P0): resolve OMS self-deadlock in getName + go.mod version"
```

---

### Task 2: 修复 DrawingPanel 翻译损坏 + Canvas CSS 变量

**Files:**
- Modify: `frontend/src/terminal/panels/DrawingPanel.vue:29,142,146,173,201,218,224`
- Create: `frontend/src/lib/canvas-theme.ts`

**Interfaces:**
- Consumes: None
- Produces: `useCanvasTheme()` composable 返回 ColorScheme

- [ ] **Step 1: 创建 Canvas 主题 composable**

```typescript
// frontend/src/lib/canvas-theme.ts
export interface ColorScheme {
  bg: string
  grid: string
  text: string
}

export function useCanvasTheme(): ColorScheme {
  const isDark = document.documentElement.classList.contains('theme-dark')
  return {
    bg: isDark ? '#1a1a2e' : '#ffffff',
    grid: isDark ? '#2d2d44' : '#e0e0e0',
    text: isDark ? '#e0e0e0' : '#333333',
  }
}
```

- [ ] **Step 2: 修复 DrawingPanel.vue — 变量名和 API**

Replace `active颜色` → `activeColor` (line 29, 173)
Replace `fill文字(` → `fillText(` (line 201, 218, 224)
Replace `var(--color-bg-elevated)` → `scheme.bg` (line 142)
Replace `var(--color-border-strong)` → `scheme.grid` (line 146)

```typescript
// Line 29
const activeColor = ref('#58a6ff')

// Line 142 — 在 renderCanvas() 中使用 useCanvasTheme()
import { useCanvasTheme } from '@/lib/canvas-theme'
// 在 renderCanvas() 开头:
const scheme = useCanvasTheme()
ctx.fillStyle = scheme.bg
ctx.strokeStyle = scheme.grid

// Line 173
color: activeColor.value,

// Line 201
ctx.fillText(b.y.toFixed(0), 6, b.y - 4)

// Line 218
ctx.fillText((ratios[i] * 100).toFixed(1) + '%', 6, y - 4)

// Line 224
ctx.fillText(d.text || '', p.x, p.y)
```

- [ ] **Step 3: 验证构建**

```bash
cd frontend && npx vite build 2>&1 | tail -5
```

Expected: No new errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/DrawingPanel.vue frontend/src/lib/canvas-theme.ts
git commit -m "fix(P0): repair DrawingPanel translation damage + Canvas CSS var"
```

---

### Task 3: 修复 ECharts CSS 变量不可用

**Files:**
- Create: `frontend/src/lib/composables/useChartTheme.ts`
- Modify: `frontend/src/terminal/panels/EquityCurvePanel.vue` (示例)
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue` (示例)
- Modify: `frontend/src/terminal/panels/MonteCarloPanel.vue` (示例)

**Interfaces:**
- Consumes: `useChartTheme()` → `{ textColor, axisColor, splitColor, bgColor }`
- Produces: 所有图表面板接入主题颜色

- [ ] **Step 1: 创建 useChartTheme**

```typescript
// frontend/src/lib/composables/useChartTheme.ts
export interface ChartTheme {
  textColor: string
  axisColor: string
  splitColor: string
  bgColor: string
}

export function useChartTheme(): ChartTheme {
  const styles = getComputedStyle(document.documentElement)
  return {
    textColor: styles.getPropertyValue('--color-text-primary').trim() || '#333333',
    axisColor: styles.getPropertyValue('--color-text-tertiary').trim() || '#888888',
    splitColor: styles.getPropertyValue('--color-border-subtle').trim() || '#e8e8e8',
    bgColor: styles.getPropertyValue('--color-bg-elevated').trim() || '#ffffff',
  }
}
```

- [ ] **Step 2: 在面板中使用 useChartTheme**

在 EquityCurvePanel.vue 等面板中，读取 computed style 写入 ECharts option:

```typescript
import { useChartTheme } from '@/lib/composables/useChartTheme'

// 在 setup 或 getOption() 中:
const theme = useChartTheme()
const option = {
  // ...
  backgroundColor: theme.bgColor,
  xAxis: {
    axisLabel: { color: theme.axisColor },
    splitLine: { lineStyle: { color: theme.splitColor } },
  },
  yAxis: {
    axisLabel: { color: theme.axisColor },
    splitLine: { lineStyle: { color: theme.splitColor } },
  },
}
```

- [ ] **Step 3: 验证构建**

```bash
cd frontend && npx vite build 2>&1 | tail -5
```

Expected: No new errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/composables/useChartTheme.ts frontend/src/terminal/panels/EquityCurvePanel.vue frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/MonteCarloPanel.vue
git commit -m "fix(P0): replace CSS vars with computed colors in ECharts panels"
```

---

### Task 4: 修复版本号不一致

**Files:**
- Modify: `python/src/server.py:55`
- Modify: `python/pyproject.toml:3`
- Modify: `python/tests/test_factor_engine.py:69`
- Modify: `internal/python/sidecar.go:25`

- [ ] **Step 1: 统一 pyproject.toml 版本号**

```toml
# python/pyproject.toml
version = "0.2.0"
```

- [ ] **Step 2: server.py 改为动态读取版本**

```python
# python/src/server.py:55
# Replace hardcoded "2026.6.26" with:
try:
    from importlib.metadata import version as pkg_version
    VERSION = pkg_version("quantflow")
except ImportError:
    VERSION = "0.2.0"
```

- [ ] **Step 3: 更新测试断言**

```python
# python/tests/test_factor_engine.py:69
assert response.version == "0.2.0"  # was "2026.6.17"
```

- [ ] **Step 4: 同步 Go 侧版本**

```go
// internal/python/sidecar.go:25
ExpectedSidecarVersion = "0.2.0"  // was "2026.6.26"
```

- [ ] **Step 5: 运行 Python 测试**

```bash
cd python && python -m pytest tests/test_factor_engine.py -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add python/pyproject.toml python/src/server.py python/tests/test_factor_engine.py internal/python/sidecar.go
git commit -m "fix(P0): unify version numbers to 0.2.0 across Go/Python"
```

---

### Task 5: 修复实盘涨跌停校验

**Files:**
- Modify: `app_trading.go:12-17` (PlaceOrder)
- Modify: `app.go` (App struct 增加 lastClose)
- Modify: `app_market.go` (行情更新时记录 lastClose)

- [ ] **Step 1: App struct 增加 lastClose**

```go
// app.go — 在 App struct 中增加
lastClose map[string]float64
```

- [ ] **Step 2: 在 PlaceOrder 前配置涨跌停**

```go
// app_trading.go:12-17 — 修改 PlaceOrder
func (a *App) PlaceOrder(symbol, side, orderType string, qty, price float64) (*trading.Order, error) {
    if a.oms == nil {
        return nil, fmt.Errorf("OMS not initialized")
    }
    
    // Configure price limits from cached prevClose
    a.mu.RLock()
    if prevClose, ok := a.lastClose[symbol]; ok && prevClose > 0 {
        ratio := PriceLimitRatio(symbol)
        a.oms.SetPriceLimit(symbol, prevClose, ratio)
    }
    a.mu.RUnlock()
    
    return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), qty, price)
}
```

- [ ] **Step 3: 添加 PriceLimitRatio helper**

```go
// app_trading.go — 新增函数
func PriceLimitRatio(symbol string) float64 {
    // Main board: ±10%, ChiNext (300/301): ±20%, STAR (688): ±20%, ST: ±5%
    if strings.HasPrefix(symbol, "300") || strings.HasPrefix(symbol, "301") || strings.HasPrefix(symbol, "688") {
        return 0.20
    }
    return 0.10
}
```

- [ ] **Step 4: 记录行情 prevClose**

```go
// app_market.go — GetQuote 等方法返回行情后记录
// (在现有行情获取逻辑中增加)
if quote.PrevClose > 0 {
    a.mu.Lock()
    a.lastClose[quote.Symbol] = quote.PrevClose
    a.mu.Unlock()
}
```

- [ ] **Step 5: 验证构建**

```bash
go build -o /dev/null . 2>&1
```

Expected: 0 errors (lastClose 访问需要实际行情流程，先保证编译通过)

- [ ] **Step 6: Commit**

```bash
git add app_trading.go app.go app_market.go
git commit -m "fix(P0): enable price limit check in live trading via SetPriceLimit"
```

---

### Task 6: 修复 OMS T+1 锁整个持仓（改为可用份额模型）

**Files:**
- Modify: `internal/trading/oms.go:37, 175-177, 235` (t1Lock 类型和使用)
- Modify: `internal/trading/oms_test.go` (更新 T+1 测试)
- Create: `internal/trading/t1.go` (T+1 日期切换辅助)

- [ ] **Step 1: 创建 t1.go**

```go
// internal/trading/t1.go
package trading

import "sync"

// T1Tracker tracks shares locked by T+1 settlement rule.
// Shares bought today cannot be sold until the next trading day.
type T1Tracker struct {
    mu     sync.Mutex
    locked map[string]float64 // symbol → locked quantity from today's buys
}

func NewT1Tracker() *T1Tracker {
    return &T1Tracker{locked: make(map[string]float64)}
}

func (t *T1Tracker) Lock(symbol string, qty float64) {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.locked[symbol] += qty
}

func (t *T1Tracker) Available(symbol string, totalQty float64) float64 {
    t.mu.Lock()
    defer t.mu.Unlock()
    locked := t.locked[symbol]
    avail := totalQty - locked
    if avail <= 0 {
        return 0
    }
    return avail
}

func (t *T1Tracker) Clear() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.locked = make(map[string]float64)
}
```

- [ ] **Step 2: OMS struct 改用 T1Tracker**

```go
// oms.go:37 — 替换字段
t1Lock *T1Tracker  // was: map[string]time.Time
```

```go
// oms.go:57 — NewOMS 初始化
t1Lock: NewT1Tracker(),
```

- [ ] **Step 3: FillOrder 卖出校验改为可用份额**

Replace `oms.go:162-177` sell-side section:

```go
// oms.go:162-177 — 修改卖出校验
pos, ok := o.positions[order.Symbol]
if order.Side == SideSell {
    if !ok || pos.Quantity <= 0 {
        return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
    }
    
    // Calculate available quantity respecting T+1 lock
    availableQty := o.t1Lock.Available(order.Symbol, pos.Quantity)
    if availableQty <= 0 {
        return nil, fmt.Errorf("T+1 lock: all %s shares bought today cannot be sold", order.Symbol)
    }
    if fillQty > availableQty {
        fillQty = availableQty
    }
    if fillQty <= 0 {
        return nil, fmt.Errorf("fill %s: no sellable shares for %s", order.ID, order.Symbol)
    }
}
```

- [ ] **Step 4: FillOrder 买入后记录 T+1**

Replace `oms.go:234-235`:

```go
// oms.go:234-235 — 买入后
o.t1Lock.Lock(order.Symbol, fillQty)
```

- [ ] **Step 5: 删除 isSameDay 函数** (no longer needed)

删除 `oms.go:423-428` 的 `isSameDay` 函数。

- [ ] **Step 6: 添加 ClearT1Lock 公开方法**

```go
// oms.go — 新增
func (o *OMS) ClearT1Lock() {
    o.t1Lock.Clear()
}
```

- [ ] **Step 7: 更新测试**

更新 `internal/trading/oms_test.go` 中的 T+1 测试：使用 T1Tracker 而非直接操作 map。

- [ ] **Step 8: 运行测试验证**

```bash
go test ./internal/trading/ -v -timeout 30s
go test ./internal/backtest/ -v -timeout 30s
```

Expected: All PASS

- [ ] **Step 9: Commit**

```bash
git add internal/trading/t1.go internal/trading/oms.go internal/trading/oms_test.go
git commit -m "fix(P0): refactor T+1 lock to available-shares model"
```

---

### Task 7: 修复回测前视偏差

**Files:**
- Modify: `internal/backtest/engine_cn.go:110,119,128,151,213,238,278` (所有成交价用 bar.Open)
- Modify: `internal/backtest/runner.go:65,73,104,140,166,198` (所有成交价用 bar.Open)
- Modify: `internal/backtest/engine_cn.go:151` (SignalFunc 调用加文档注释)

- [ ] **Step 1: engine_cn.go — processCNBuySignal 成交价改 bar.Open**

```go
// engine_cn.go:213 — 替换
effectivePrice := bar.Open * (1 + slippage)  // was: bar.Close
```

- [ ] **Step 2: engine_cn.go — processCNSellSignal 成交价改 bar.Open**

```go
// engine_cn.go:278 — 替换
effectivePrice := bar.Open * (1 - slippage)  // was: bar.Close
```

- [ ] **Step 3: engine_cn.go — 止损 FillOrder 改 bar.Open**

```go
// engine_cn.go:128 — 替换
e.oms.FillOrder(order.ID, availableQty, bar.Open)  // was: bar.Close
```

- [ ] **Step 4: engine_cn.go — SignalFunc 调用加注释**

```go
// engine_cn.go:150-151 — 增加注释
// NOTE: SignalFunc receives the full OHLCV bar for context.
// Strategies MUST NOT use bar.Close as an execution price — execution happens at bar.Open.
// Using bar.Close in signals is acceptable for momentum/trend calculation
// but constitutes look-ahead bias for execution decisions.
signal := strategy.SignalFunc(bar, portfolio)
```

- [ ] **Step 5: runner.go — processBuySignal 成交价改 bar.Open**

```go
// runner.go:140 — 替换
effectivePrice := bar.Open * (1 + r.config.Slippage)  // was: bar.Close
```

- [ ] **Step 6: runner.go — processSellSignal 成交价改 bar.Open**

```go
// runner.go:198 — 替换
effectivePrice := bar.Open * (1 - r.config.Slippage)  // was: bar.Close
```

- [ ] **Step 7: runner.go — 止损/止盈 FillOrder 改 bar.Open**

```go
// runner.go:73,88 — 替换
r.oms.FillOrder(order.ID, pos.Quantity, bar.Open)  // was: bar.Close
```

- [ ] **Step 8: 运行回测测试**

```bash
go test ./internal/backtest/ -v -timeout 30s
```

Expected: All PASS (可能有些测试预期值会变，需调整)

- [ ] **Step 9: Commit**

```bash
git add internal/backtest/engine_cn.go internal/backtest/runner.go
git commit -m "fix(P0): eliminate look-ahead bias, execute at bar.Open not bar.Close"
```

---

### Task 8: 修复多标的权益计算（全量价格 map）

**Files:**
- Modify: `internal/backtest/engine_cn.go:166-179` (Run loop 中权益计算)
- Modify: `internal/backtest/runner.go:114-121` (Run loop 中权益计算)
- Modify: `internal/backtest/config.go:48-56` (Equity 方法 — 已正确，无需修改)

- [ ] **Step 1: engine_cn.go — 维护 latestPrices**

```go
// engine_cn.go — Run loop 中增加 latestPrices
func (e *CNEngine) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
    // ... existing code ...
    portfolio := NewPortfolio(e.config.InitialCash)
    var equityCurve []EquityPoint
    var tradeRecords []TradeRecord
    latestPrices := make(map[string]float64)  // NEW: track all symbols' latest prices
    var lastDate string                         // NEW: track date changes

    for _, bar := range bars {
        // ... existing ctx check ...
        
        // Update latest price for current symbol
        latestPrices[bar.Symbol] = bar.Close
        
        // ... existing UpdateMarketPrice ...
```

- [ ] **Step 2: engine_cn.go — 替换 recordEquityCN label**

```go
// engine_cn.go:166-179 — 替换 recordEquityCN 代码块
recordEquityCN:
    // Update prevClose for next day's price limit check
    e.prevClose[bar.Symbol] = bar.Close

    // Clear T+1 lock only on date change (not every bar)
    if lastDate != "" && bar.Date != lastDate {
        e.t1Lock.locked = make(map[string]float64)
    }
    lastDate = bar.Date

    // Record equity with ALL known prices
    equityCurve = append(equityCurve, EquityPoint{
        Date:   bar.Date,
        Equity: portfolio.Equity(latestPrices),
        Cash:   portfolio.Cash,
    })
```

- [ ] **Step 3: runner.go — 同样增加 latestPrices**

```go
// runner.go:45-122 — Run loop
func (r *Runner) Run(ctx context.Context, strategy Strategy, bars []trading.OHLCVBar) (*Result, error) {
    // ... existing ...
    latestPrices := make(map[string]float64)  // NEW

    for _, bar := range bars {
        latestPrices[bar.Symbol] = bar.Close   // NEW
        
        // ... rest of loop ...

    recordEquity:
        equityCurve = append(equityCurve, EquityPoint{
            Date:   bar.Date,
            Equity: portfolio.Equity(latestPrices),  // was: map[string]float64{bar.Symbol: bar.Close}
            Cash:   portfolio.Cash,
        })
    }
```

- [ ] **Step 4: 运行测试**

```bash
go test ./internal/backtest/ -v -timeout 30s
```

- [ ] **Step 5: Commit**

```bash
git add internal/backtest/engine_cn.go internal/backtest/runner.go
git commit -m "fix(P0): fix multi-symbol equity with full latestPrices map"
```

---

### Task 9: 修复 RL TradingEnv 组合账户计算

**Files:**
- Modify: `python/src/ml/rl/env.py:51-73` (step 方法重写)
- Create: `python/tests/test_rl_env.py` (测试)

- [ ] **Step 1: 重写 step 方法**

```python
# python/src/ml/rl/env.py:51-73 — 替换 step 方法
def step(self, action):
    if self.action_type == "discrete":
        action_val = action - 1  # 0->sell(-1), 1->hold(0), 2->buy(1)
    else:
        action_val = float(action)

    prev_position = self.position
    self.position = np.clip(action_val, -1.0, 1.0)

    price = self.ohlcv[self.current_step, 0]
    prev_price = self.ohlcv[self.current_step - 1, 0] if self.current_step > 0 else price
    price_return = (price - prev_price) / prev_price if prev_price > 0 else 0.0

    # Correct portfolio model:
    # - Cash changes only when we trade (buy/sell)
    # - Portfolio value = cash + position_value
    # - position_value = position * previous_portfolio_value (for long)
    #   For short (position < 0), the value change is inverted
    portfolio_value_before = self.portfolio_value
    
    # Compute trade
    if prev_position != self.position:
        # How much portfolio value to reallocate
        delta = self.position - prev_position
        trade_value = delta * portfolio_value_before  # positive = buy, negative = sell
        trade_cost = abs(trade_value) * 0.001
        self.cash -= trade_value + trade_cost  # buy: cash decreases; sell: cash increases
        self.cash = max(self.cash, 0)  # cash cannot go negative
    
    # Update portfolio value after price change
    position_value = self.position * portfolio_value_before * (1 + price_return)
    self.portfolio_value = self.cash + position_value
    self.portfolio_value = max(self.portfolio_value, 0)

    reward = (self.portfolio_value - self.prev_value) / self.prev_value if self.prev_value > 0 else 0.0
    self.prev_value = self.portfolio_value
    self.current_step += 1

    done = self.current_step >= len(self.ohlcv) - 1
    truncated = False

    return self._get_state(), reward, done, truncated, {"portfolio_value": self.portfolio_value}
```

- [ ] **Step 2: 写测试验证**

```python
# python/tests/test_rl_env.py
import numpy as np
import pytest

@pytest.fixture
def simple_ohlcv():
    """10 periods of uptrend OHLCV data."""
    prices = np.linspace(100, 110, 10)
    data = np.column_stack([prices, prices*1.02, prices*0.98, prices, np.ones(10)*1000])
    return data.astype(np.float32)

def test_buy_and_hold_increases_value(simple_ohlcv):
    from src.ml.rl.env import TradingEnv
    env = TradingEnv(simple_ohlcv, window_size=3, initial_cash=10000)
    env.reset()
    
    # Buy (action=2 for discrete)
    state, reward, done, truncated, info = env.step(2)
    assert reward >= 0, f"Expected non-negative reward on buy in uptrend, got {reward}"
    assert env.portfolio_value >= 10000, f"Portfolio should grow in uptrend"

def test_hold_preserves_cash(simple_ohlcv):
    from src.ml.rl.env import TradingEnv
    env = TradingEnv(simple_ohlcv, window_size=3, initial_cash=10000)
    env.reset()
    
    initial_cash = env.cash
    state, reward, done, truncated, info = env.step(1)  # hold
    
    assert env.cash == initial_cash, f"Hold should not change cash, got {env.cash} vs {initial_cash}"

def test_full_buy_then_sell_no_neg_portfolio(simple_ohlcv):
    from src.ml.rl.env import TradingEnv
    env = TradingEnv(simple_ohlcv, window_size=3, initial_cash=10000)
    env.reset()
    
    env.step(2)  # buy (full position since action_val=1)
    for _ in range(3):
        env.step(1)  # hold
    
    env.step(0)  # sell
    
    assert env.portfolio_value >= 0, f"Portfolio should not go negative after sell, got {env.portfolio_value}"
    assert env.cash >= 0, f"Cash should not go negative, got {env.cash}"
```

- [ ] **Step 3: 运行测试**

```bash
cd python && python -m pytest tests/test_rl_env.py -v
```

Expected: All PASS (3 tests)

- [ ] **Step 4: Commit**

```bash
git add python/src/ml/rl/env.py python/tests/test_rl_env.py
git commit -m "fix(P0): rewrite RL TradingEnv portfolio calculation"
```

---

## Phase 1: P1 关键修复

### Task 10: 修复滑点模型 + PnL 覆盖 + 过户费

**Files:**
- Modify: `internal/backtest/engine_cn.go:42` (SquareRootSlippage)
- Modify: `internal/trading/oms.go:239,242` (PnL 覆盖)
- Modify: `internal/backtest/engine_cn.go:214,279` (过户费)
- Modify: `internal/trading/oms.go:191,197` (OMS 过户费)

- [ ] **Step 1: 修复 SquareRootSlippage**

```go
func (s *SquareRootSlippage) Apply(order trading.Order, bar trading.OHLCVBar) float64 {
    if bar.Volume <= 0 {
        return s.Base
    }
    partRate := float64(order.Quantity) / bar.Volume
    return s.Base + s.VolRatio * math.Sqrt(partRate)
}
```

- [ ] **Step 2: 修复 FillOrder PnL 覆盖**

```go
// oms.go:238-248 — 卖出后
realizedPnl := (fillPrice-pos.AvgPrice)*fillQty - commission - stampTax
pos.RealizedPnl += realizedPnl
pos.Quantity -= fillQty
if pos.Quantity == 0 {
    pos.AvgPrice = 0
    pos.PnL = 0
} else {
    pos.PnL = pos.RealizedPnl + (pos.MarketPrice - pos.AvgPrice) * pos.Quantity
}
pos.PnLPct = (pos.MarketPrice - pos.AvgPrice) / pos.AvgPrice * 100
```

- [ ] **Step 3: 添加过户费**

engine_cn.go buy: `transferFee := effectivePrice * qty * 0.00001`
engine_cn.go sell: `transferFee := effectivePrice * qty * 0.00001`

- [ ] **Step 4: 运行测试 + Commit**

```bash
go test ./internal/backtest/ -v -timeout 30s && go test ./internal/trading/ -v -timeout 30s
git add internal/backtest/engine_cn.go internal/trading/oms.go
git commit -m "fix(P1): fix slippage sqrt, PnL overwrite, add A-share transfer fee"
```

### Task 11-14: 剩余 P1 修复(架构补丁)

Task 11: 统一 Sharpe/Calmar 公式 (metrics.go, risk.go)
Task 12: 涨跌停四舍五入 (price_limit.go, oms.go)
Task 13: SQLite WAL 连接池 (db.go)  
Task 14: errgroup recover + 回测错误日志 (workflow/engine.go, engine_cn.go)

每任务同样按 TDD 流程: 修改 → 测试 → 验证 → 提交

---

## 执行摘要

| Task | 描述 | 预计时间 |
|------|------|---------|
| 1 | OMS 自死锁 + go.mod | 5m |
| 2 | DrawingPanel 修复 | 10m |
| 3 | ECharts CSS 变量 | 30m |
| 4 | 版本号统一 | 10m |
| 5 | 实盘涨跌停 | 30m |
| 6 | OMS T+1 可用份额 | 45m |
| 7 | 回测前视偏差 | 30m |
| 8 | 多标的权益 | 20m |
| 9 | RL 组合计算 | 30m |
| 10-14 | P1 修复 | 各 15-30m |

**Phase 0 总计**: ~3.5h | **Phase 1 总计**: ~2h

---

*Plan complete. Ready for execution.*

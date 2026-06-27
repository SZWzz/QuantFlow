# QuantFlow 评审问题修复 Spec

> **版本**: 0.2.0 | **日期**: 2026-06-27 | **状态**: Draft
> **基于**: docs/reviews/2026-06-27-comprehensive-review.md (9 P0 + 30+ P1)

## 概述

本 Spec 定义对全面评审中发现的 9 个 P0 致命问题和关键 P1 问题的修复方案。修复分两阶段：P0 立即修复（5 分钟~半天/个），P1 本轮迭代修复。

---

## Phase 0 — P0 致命修复

### P0-1: OMS 自死锁

**问题**: `getName()` 被写锁持有者调用时内部请求读锁 → `sync.RWMutex` 不可重入 → 死锁
**位置**: `internal/trading/oms.go:389-393` (getName), 调用点 119/224/269/280
**修复**: `getName` 不加锁，直接读 `quoteCache`（调用方已持锁）
**测试**: `go test ./internal/trading/ -run TestOMS -v -timeout 30s` 必须通过

```go
// Before (Line 389-393)
func (o *OMS) getName(symbol string) string {
    o.mu.RLock()
    defer o.mu.RUnlock()
    return o.quoteCache[symbol]
}

// After
func (o *OMS) getName(symbol string) string {
    return o.quoteCache[symbol]
}
```

---

### P0-2: 回测前视偏差（CNEngine + Runner）

**问题**: Signal 看到当日 Close 并以当日 Close 成交
**位置**: `internal/backtest/engine_cn.go:110,119,128,151,213,238`; `runner.go:65,73,104,140,166,198`
**修复**: 
  1. 信号基于 `bar.Close`（可接受，代表当日已知信息），但**成交价改用 `bar.Open`**（模拟开盘集合竞价成交）
  2. 止损检查用日内最差价格触发，成交价用 `bar.Open`
**权衡**: 理想方案是信号基于昨日信息、成交用次日 Open。当前复权未统一、多市场 bar 结构不同时 Open 可能是当天第一笔。先用 `bar.Open` 成交作为最低限度修复。

```go
// engine_cn.go:213 — 成交价改 bar.Open
effectivePrice := bar.Open * (1 + slippage)  // was bar.Close
```

**全文件搜索**：engine_cn.go 和 runner.go 中所有 `bar.Close` 作为成交价的点，按语义区分：「信号/止损判断可用当日信息」「成交价必须用 bar.Open」。
- 信号输入保持 `bar` 完整（可访问 OHLC），但文档标注"策略不得依赖未来信息"
- processCNBuySignal: line 213 `effectivePrice := bar.Open * (1 + slippage)`
- processCNSellSignal: line 278 `effectivePrice := bar.Open * (1 - slippage)`
- runner.go processBuySignal: line 140 `effectivePrice := bar.Open * (1 + r.config.Slippage)`
- runner.go processSellSignal: line 198 `effectivePrice := bar.Open * (1 - r.config.Slippage)`
- 止损/止盈 FillOrder: line 128,73,88 `bar.Open` 而非 `bar.Close`
- 止损入口 `UpdateMarketPrice` 保持用 `bar.Close`（更新市价）

---

### P0-3: 多标的权益按 0 估值

**问题**: 回测逐 bar 处理，每个 bar 只传当前 symbol 价格给 Equity()，其他持仓跳过按 0
**位置**: `internal/backtest/engine_cn.go:174`; `runner.go:116`; `config.go:48-56`
**修复**: 维护 `latestPrices map[string]float64`，每处理一个 bar 更新对应 symbol 价格，计算权益时用全量价格 map

```go
// CNEngine.Run() 增加
latestPrices := make(map[string]float64)
// 在每次更新行情后
latestPrices[bar.Symbol] = bar.Close
// 计算权益
equity := portfolio.Equity(latestPrices)
```

---

### P0-4/P0-5: DrawingPanel 翻译损坏 + Canvas CSS 变量

**问题**: `fillText` 被翻成 `fill文字`（不是 Canvas API）；Canvas 用 CSS 变量作颜色无效
**位置**: `frontend/src/terminal/panels/DrawingPanel.vue:29,142,146,201,218,224`
**修复**:
  1. `fill文字` → `fillText` (3 处: line 201, 218, 224)
  2. `active颜色` → `activeColor` (line 29, 173)
  3. Canvas 背景/网格颜色改为具体颜色值或从 computed style 读取

```typescript
// Line 29
const activeColor = ref('#58a6ff')
// Line 142 — 从 CSS 变量读取实际颜色
ctx.fillStyle = getComputedStyle(document.documentElement).getPropertyValue('--color-bg-default').trim() || '#ffffff'
// Line 146
ctx.strokeStyle = getComputedStyle(document.documentElement).getPropertyValue('--color-border-subtle').trim() || '#e0e0e0'
```

---

### P0-6: ECharts 75 处 CSS 变量不可用

**问题**: ECharts Canvas 渲染器不解析 CSS 变量
**位置**: 70+ 处（所有图表面板如 MonteCarloPanel, CandlestickPanel 等）
**修复**:
  1. 在 `frontend/src/lib/composables/` 新增 `useChartTheme.ts`
  2. 从 CSS variables 读取计算值返回 ECharts 纯色 theme 对象
  3. 在各图表面板调用 `useChartTheme()` 并应用到 echarts option
  4. 监听主题切换，对活动图表调用 `setOption` 重绘

```typescript
// useChartTheme.ts
import { computed } from 'vue'

export function useChartTheme() {
  const styles = getComputedStyle(document.documentElement)
  return {
    textColor: styles.getPropertyValue('--color-text-primary').trim() || '#333333',
    axisColor: styles.getPropertyValue('--color-text-tertiary').trim() || '#888888',
    splitColor: styles.getPropertyValue('--color-border-subtle').trim() || '#e8e8e8',
    bgColor: styles.getPropertyValue('--color-bg-elevated').trim() || '#ffffff',
  }
}
```

---

### P0-7: wailsjs 类型绑定未生成

**问题**: 无 wails3.yml 配置、无生成绑定，71 处 `(window as any).go.main.App.XXX` 失去类型安全
**位置**: `main.go:18`; `frontend/src/lib/wails.ts:42-46`; 71 处调用点
**修复**: 
  1. 生成 wails3.yml 配置文件（Wails v3 需要）
  2. 运行 `wails3 generate bindings` 生成类型绑定
  3. 将 71 处 `(window as any).go.main.App.XXX` 替换为类型安全的调用
  4. 在 `src/lib/wails.ts` 的 Proxy shim 中保留向后兼容

**注意**: Wails v3 处于 alpha 阶段，`wails3 generate` 命令可能不稳定。作为最小化修复：
  - 在 `src/types/wails-runtime.d.ts` 中声明 `App` 的所有 77 个方法的完整类型
  - 在 wails.ts shim 中用类型化 interface 替换 `any`

---

### P0-8: RL TradingEnv 组合账户计算错误

**问题**: 全仓后 cash=0，平仓时 cash 变负，portfolio_value 变负，reward≈-1
**位置**: `python/src/ml/rl/env.py:66-69`
**修复**: 重写 step() 中的组合计算逻辑为正确的基于份额的模型

```python
# 正确模型（基于份额而非全仓比例）
def step(self, action):
    prev_position = self.position
    self.position = np.clip(action_val, -1.0, 1.0)
    
    price = self.ohlcv[self.current_step, 0]
    price_return = ...
    
    # 计算仓位变化带来的现金变化
    shares_traded = (self.position - prev_position) * self.portfolio_value / price
    trade_cost = abs(shares_traded) * price * 0.001
    self.cash -= shares_traded * price + trade_cost
    
    # 更新组合价值（cash + 持仓市值）
    position_value = self.position * self.portfolio_value
    self.portfolio_value = self.cash + position_value * (1 + price_return)
    
    reward = (self.portfolio_value - self.prev_value) / self.prev_value
```

**关键变更**:
  1. 交易操作修改 cash，不直接修改 portfolio_value
  2. `self.position` 表示仓位比例（-1到1），持仓市值 = position * portfolio_value * price_return
  3. reward 为 portfolio value 相对变化率

---

### P0-9: 版本号四处不一致

**问题**: go.mod(1.26.4)/README(2026.6.19)/server.py(2026.6.26)/pyproject.toml(2026.6.17) 互不相同
**位置**:
- `go.mod:3` → `go 1.26.4` 不存在
- `python/src/server.py:55` → `2026.6.26`
- `python/pyproject.toml:3` → `2026.6.17`
- `python/tests/test_factor_engine.py:69` → 断言 `2026.6.17`
- `internal/python/sidecar.go:25` → `ExpectedSidecarVersion = "2026.6.26"`

**修复**:
  1. go.mod 改为 `go 1.22`
  2. 统一版本号 source-of-truth: `python/pyproject.toml` 的 version 字段
  3. server.py 从 `importlib.metadata.version("quantflow")` 读取版本（pyproject.toml 安装后）
  4. 或：在工程根目录 `resources/version.yaml` 放单一版本，Makefile target 同步所有引用
  5. sidecar.go 改为匹配新版本

---

### P0-10: OMS T+1 锁整个持仓（拒绝合法交易）

**问题**: A股 T+1 正确语义是"今日买入的份额不可卖"，当前实现锁整个持仓
**位置**: `internal/trading/oms.go:37, 175-177, 235`
**修复**: 改为「可用份额」模型

```go
// OMS struct: t1Lock 改类型
t1Lock map[string]float64 // symbol → today's locked quantity (was time.Time)

// FillOrder 卖出校验 (line 175-177)
if order.Side == SideSell {
    heldQty := pos.Quantity
    lockedQty := o.t1Lock[order.Symbol]
    availableQty := heldQty - lockedQty
    if availableQty <= 0 {
        return nil, fmt.Errorf("T+1 lock: all %s shares locked today", order.Symbol)
    }
    if fillQty > availableQty {
        fillQty = availableQty
    }
}

// FillOrder 买入后 (line 235)
o.t1Lock[order.Symbol] += fillQty  // was: time.Now()

// 日期切换时清零 T+1 锁
func (o *OMS) ClearT1Lock() {
    o.mu.Lock()
    defer o.mu.Unlock()
    o.t1Lock = make(map[string]float64)
}
```

---

### P0-11: 实盘涨跌停校验失效

**问题**: 生产代码 0 处调用 SetPriceLimit
**位置**: `app_trading.go:12-17`
**修复**: 在 PlaceOrder 前调用 SetPriceLimit

```go
// app_trading.go — 在 PlaceOrder 前根据 symbol 配置涨跌停
func (a *App) PlaceOrder(symbol, side, orderType string, qty, price float64) (*trading.Order, error) {
    if a.oms == nil {
        return nil, fmt.Errorf("OMS not initialized")
    }
    
    // 从行情数据或缓存获取昨收价
    if prevClose, ok := a.lastClose[symbol]; ok {
        a.oms.SetPriceLimit(symbol, prevClose, PriceLimitRatio(symbol))
    }
    
    return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), qty, price)
}
```

额外需要:
- App struct 增加 `lastClose map[string]float64`
- `PriceLimitRatio(symbol)` 函数（已有 `PriceLimitFor` 逻辑，复用）

---

## Phase 1 — P1 严重修复（精选）

### P1-1: 回测 T+1 每 bar 清空 map

**位置**: `engine_cn.go:171`
**问题**: `e.t1Lock.locked = make(map[string]float64)` 每 bar 清空，应在日期切换时清空
**修复**: 比较当前 bar.Date 与上一 bar.Date，不同时清空

```go
// engine_cn.go — 替换 line 171
// 仅在日期切换时清空 T+1 锁
if lastDate != "" && bar.Date != lastDate {
    e.t1Lock.locked = make(map[string]float64)
}
lastDate = bar.Date
```

### P1-2: FillOrder 卖出后 pos.PnL 被覆盖

**位置**: `oms.go:239, 242`
**问题**: 卖出后 PnL 覆盖为单笔实现盈亏，失去未实现盈亏
**修复**:

```go
// oms.go:238-248 — 卖出后不覆盖 pos.PnL
// 删除 pos.PnL = realizedPnl (line 239)
pos.RealizedPnl += realizedPnl
pos.Quantity -= fillQty
if pos.Quantity == 0 {
    pos.AvgPrice = 0
    pos.PnL = 0
} else {
    // 使用 marketPrice 重算
    pos.PnL = pos.RealizedPnl + (pos.MarketPrice - pos.AvgPrice) * pos.Quantity
}
pos.PnLPct = (pos.MarketPrice - pos.AvgPrice) / pos.AvgPrice * 100  // 用 MarketPrice
```

### P1-3: SquareRootSlippage 实现平方根错误

**位置**: `engine_cn.go:42`
**修复**:

```go
func (s *SquareRootSlippage) Apply(order trading.Order, bar trading.OHLCVBar) float64 {
    if bar.Volume <= 0 {
        return s.Base
    }
    // 标准平方根市场冲击: base + volRatio * sqrt(orderQty / barVolume)
    partRate := float64(order.Quantity) / bar.Volume
    return s.Base + s.VolRatio * math.Sqrt(partRate)
}
```

### P1-4: 各 adapter 复权方式不一致

**位置**: `tencent.go:80`, `eastmoney.go:128`, `akshare.go:85`
**修复**:
  1. 所有 CN adapter 尊重传入 `fqfactor` 参数
  2. tencent.go: `qfq` 不再写死，按 fqfactor 映射
  3. 回测统一用后复权（hfq）避免 look-ahead

### P1-5: A股过户费缺失

**位置**: `engine_cn.go:214,279`, `oms.go:189-198`
**修复**: 买卖双向加过户费 0.001%（万分之0.1）

```go
// engine_cn.go — 买入加过户费
transferFee := effectivePrice * qty * 0.00001  // 0.001%
cost := effectivePrice*qty + effectivePrice*qty*e.config.Commission + transferFee

// 卖出加过户费（双向收取）
transferFee := effectivePrice * qty * 0.00001
revenue := effectivePrice*qty - stampDuty(...) - ...*e.config.Commission - transferFee
```

### P1-6: 港股引擎缺失

**修复**: 新增 `internal/backtest/engine_hk.go` (HKEngine)，含：
- 港股印花税 0.13%（买卖双向各 0.1%+0.0027%+0.00015% trading fee etc.）
- 最小手数（因股而异的多手校验）
- T+2 结算
- 已实现 `backtest.go:142-144` 的 HK 分支路由

### P1-7: 幸存者偏差（退市股）

**位置**: `tushare.go:43, 88-92`
**修复**: 拉取数据时传 `list_status='L+D'`（含退市股），回测对退市日后的持仓强制清仓

### P1-8: PDT 规则实现错误

**位置**: `engine_us.go:42-67`
**修复**: 
  1. 实现 5 日滑动窗口 day trade 计数器
  2. 账户 < $25k 时阻断日内开仓
  3. 同标的不同日内方向转换才计为 day trade

### P1-9: 组合回测权益计算（同P0-3，已覆盖）

### P1-10: Sharpe/Calmar 公式不统一

**位置**: `metrics.go:64,90` vs `risk.go:74,96`
**修复**: 统一为标准定义
```go
// metrics.go
sharpe = (mean * 252 - 0.02) / annualVol  // CAGR → mean*252
calmar = cagr / (-maxDD)                   // 保持 CAGR

// risk.go
sharpe = ((mean * 252) - riskFreeRate) / annualVol  // 保持
calmar = cagr / maxDD  // mean*252 → cagr
```

### P1-11: 涨跌停价未四舍五入

**位置**: `price_limit.go:47`, `oms.go:360-361`
**修复**: `math.Round(raw*100)/100`

### P1-12: 双现金账本（回测 portfolio.Cash vs OMS cashLedger）

**位置**: `engine_cn.go:242` vs `oms.go:252`
**修复**: 
  1. 将 `portfolio.Cash` 作为单一源
  2. `FillOrder` 不再通过 `cashLedger.RecordTrade` 修改现金
  3. 或：取消回测中的 portfolio.Cash，统一用 OMS cashLedger

---

## 修复优先级总览

| # | 问题 | 成本 | 阶段 |
|---|------|------|------|
| P0-1 | OMS 自死锁 | 5m | Phase 0 |
| P0-4/P0-5 | DrawingPanel 翻译损坏 + Canvas | 10m | Phase 0 |
| P0-9 | 版本号统一 | 10m | Phase 0 |
| P0-11 | 实盘涨跌停 | 30m | Phase 0 |
| P0-6 | ECharts 75处 CSS 变量 | 2h | Phase 0 |
| P0-10 | OMS T+1 改可用份额 | 1h | Phase 0 |
| P0-2 | 回测前视偏差 | 2h | Phase 0 |
| P0-3 | 多标的权益全量价格 | 1h | Phase 0 |
| P0-8 | RL 组合计算重写 | 2h | Phase 0 |
| P1-1~P1-12 | P1 关键修复 | 各1-3h | Phase 1 |

---

*Spec 完。Execution Plan 见 docs/superpowers/plans/2026-06-27-fix-review-p0-p1.md*

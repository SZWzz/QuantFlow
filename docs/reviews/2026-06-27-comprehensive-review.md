# QuantFlow Terminal 全面评审报告

> **评审日期**：2026-06-27
> **评审范围**：全项目（Go 后端 / Vue 3 前端 / Python sidecar / 金融业务规则 / 产品定位）
> **评审角色**：代码专家 + 前后端专家 + 金融专家 + 产品专家
> **评审方式**：采样 60+ 关键文件 + 实际运行测试验证（已复现死锁）
> **结论**：**工程结构清晰、覆盖面广，但存在 9 个 P0 致命缺陷，当前回测结果不可作为实盘决策依据。** 建议修复全部 P0 + 关键 P1 后再上实盘。

---

## 一、执行摘要

QuantFlow 是一个雄心勃勃的双模式量化金融终端，Phase 1–11 声称完成，54+ 节点、46 面板、25+ 数据适配器。架构选型合理（Go 单二进制 + Wails v3 + Vue 3 + Python gRPC sidecar + SQLite WAL），DockView 自研面板系统、symbolContext 联动组、因子注册表、LLM provider 插件化等设计有亮点。

**但深入审查后发现，"完成"的定义过于宽松**：

- **主干测试无法通过**：OMS 存在自死锁（`sync.RWMutex` 不可重入），`go test ./internal/trading/` 与 `go test ./internal/backtest/` 全部超时挂死。
- **回测结果系统性失真**：前视偏差（信号与成交同价）+ 多标的权益按 0 估值 + 双现金账本发散，回测收益不可信。
- **实盘风险裸奔**：从未调用 `SetPriceLimit`，A 股涨跌停校验完全不生效；OMS T+1 锁的是"整个持仓"而非"今日买入份额"，会拒绝合法交易。
- **前端系统性视觉缺陷**：ECharts/Canvas 共 75+ 处使用 CSS 变量作颜色（Canvas 不解析 CSS 变量），深色主题下图表近乎不可见；DrawingPanel 出现 `ctx.fill文字`（翻译工具损坏代码）。
- **Python 训练无效**：RL TradingEnv 组合账户计算错误，reward 信号完全失真，所有 RL 训练（PPO/DQN/SAC）基于错误 reward；DeepEngine 时序训练 `shuffle=True` 数据泄漏。

### 问题统计

| 严重程度 | 数量 | 含义 |
|---------|:----:|------|
| **P0 致命** | 9 | 功能不可用 / 结果不可信 / 阻断测试 / 实盘风险 |
| **P1 严重** | 30+ | 逻辑错误 / 数据不一致 / 并发隐患 / 市场规则缺失 |
| **P2 一般** | 40+ | 边界 / 规范 / 可维护性 / 性能 |

### 维度问题分布

| 维度 | P0 | P1 | P2 | 评价 |
|------|:--:|:--:|:--:|------|
| 后端 Go | 3 | 12 | 10+ | 死锁阻断测试 + 前视偏差 + 双账本 |
| 前端 Vue/TS | 4 | 9 | 14 | CSS 变量误用 + 翻译损坏 + 类型失守 |
| Python sidecar | 2 | 12 | 20+ | RL 计算错误 + 数据泄漏 + 依赖分叉 |
| 金融正确性 | 3 | 9 | 6 | T+1/涨跌停/前视三大致命 |

---

## 二、架构与项目结构

### 2.1 技术栈选型（合理）

| 层 | 选择 | 评价 |
|----|------|------|
| 后端 | Go 1.22+ | ✅ goroutine DAG 并行，单二进制适合桌面 |
| 桌面 | Wails v3 (alpha) | ⚠️ 用 alpha 版（`v3.0.0-alpha2.103`）做生产，API 不稳定 |
| 前端 | Vue 3 + TS + vue-flow | ✅ vue-flow 是工作流编排的合理选择 |
| 数据库 | SQLite WAL | ✅ 桌面单用户，零配置 |
| ML | Python 3.12 gRPC sidecar | ✅ 隔离 GIL，pandas 生态 |
| 图表 | ECharts | ⚠️ 集成方式有问题（见前端评审） |

### 2.2 元数据一致性问题

- `go.mod:3` 写 `go 1.26.4` —— **该版本不存在**（Go 最新稳定版 1.24，1.25 尚未发布）。应改 `go 1.22`。
- `README.md:12` 版本 `2026.6.19`，`config.yaml` 版本 `0.0.1`，`python/server.py:55` `2026.6.26`，`pyproject.toml:3` `2026.6.17` —— **四处版本号互不相同**，sidecar 版本校验会触发重启循环。

### 2.3 架构亮点

- **双模式联动**：Terminal → Workflow（面板 `[⊕]` 生成节点）+ Workflow → Terminal（结果固定为监控面板），概念清晰。
- **DockView 自研面板系统**：symbolContext 联动组（同一标的的多面板联动）、拖拽分屏、布局持久化，设计完整。
- **因子注册表**：25 个因子 + 横截面/时序分类，`cross_sectional.py` 已正确用全 panel 计算，并有回归测试。
- **LLM provider 插件化**：注册表 + 自动注册，新增 provider 成本低。
- **FallbackChain 容灾**：10 源容灾的 A 股数据链路思路正确（但复权方式不一致，见金融评审）。

---

## 三、后端 Go 代码评审

### 🔴 P0-1 OMS 自死锁（阻断全包测试，已实测确认）

**位置**：`internal/trading/oms.go:119, 142, 224, 269, 280, 389-393`

```go
// oms.go:106  PlaceOrder 持写锁
o.mu.Lock()
defer o.mu.Unlock()
order.Name = o.getName(symbol)  // line 119 调用 getName

// oms.go:389-393  getName 又请求读锁
func (o *OMS) getName(symbol string) string {
    o.mu.RLock()        // ← 写锁持有者再请求读锁 → 自死锁
    defer o.mu.RUnlock()
    return o.quoteCache[symbol]
}
```

**实测**：`go test ./internal/trading/ -run TestOMS` 与 `go test ./internal/backtest/ -run TestCNEngine_AllowsNormalBuy` **30s/15s 超时挂死**，goroutine 栈定格在 `oms.go:390 RLock ← oms.go:119 PlaceOrder`。Go 的 `sync.RWMutex` 不可重入。受影响方法：`PlaceOrder`、`FillOrder`、`GetPosition`、`GetAllPositions`。

**修复**：`getName` 不再获取锁（调用方已持锁），直接读 `o.quoteCache[symbol]`。

### 🔴 P0-2 回测前视偏差（收益系统性高估）

**位置**：`internal/backtest/engine_cn.go:110, 119, 128, 151, 213, 238`；`internal/backtest/runner.go:65, 73, 104, 140, 166, 198`

```go
// engine_cn.go:110  用当日 Close 更新行情
e.oms.UpdateMarketPrice(bar.Symbol, bar.Close)
// engine_cn.go:151  策略基于 bar（含 Close）生成信号
signal := strategy.SignalFunc(bar, portfolio)
// engine_cn.go:213  又在当日 Close 成交
effectivePrice := bar.Close * (1 + slippage)
```

策略看到当日收盘价并在当日收盘价成交——教科书级前视偏差。`order_matcher.go:37` 本来正确（市价单按 `bar.Open` 成交），但回测引擎绕过 matcher。

**修复**：信号基于 T 日收盘，成交在 T+1 日开盘（`bar.Open`）；最低限度市价单用 `bar.Open`。

### 🔴 P0-3 多标的权益计算错误（指标全错）

**位置**：`internal/backtest/engine_cn.go:174`；`internal/backtest/runner.go:116`；`internal/backtest/config.go:48-56`

```go
// engine_cn.go:174  只传当前 bar 的标的价格
prices := map[string]float64{bar.Symbol: bar.Close}
equityCurve = append(equityCurve, EquityPoint{
    Equity: portfolio.Equity(prices),  // config.go:50 跳过不在 map 中的持仓 → 按 0 估值
})
```

多标的回测时，处理 A 的 bar 时 B 的持仓市值被忽略 → 权益曲线错误 → Sharpe/MaxDD/Calmar 全错。

**修复**：维护 `latestPrices map[string]float64`，每个 bar 更新后用全量价格算权益。

### P1 关键问题（精选）

| # | 问题 | 位置 |
|---|------|------|
| 1 | T+1 锁在回测中是死代码（同 bar 内设置又清空） | `engine_cn.go:171, 252` |
| 2 | OMS T+1 用 wall clock，回测中阻断所有卖出 | `oms.go:175, 235` |
| 3 | 资金账本双轨（回测 portfolio.Cash vs OMS cashLedger），佣金率不同 | `engine_cn.go:214` vs `oms.go:191` |
| 4 | CacheKey 值无定界，哈希碰撞（`{"a":"1\|b:2"}` 与 `{"a":"1","b":"2"}` 同 hash） | `cache.go:36-51` |
| 5 | 回测与 OMS 涨跌停边界语义不一致（`<` vs `<=`） | `price_limit.go:66` vs `oms.go:362` |
| 6 | Sharpe/Calmar 两文件公式不一致（CAGR vs 算术年化） | `metrics.go:64,90` vs `risk.go:74,96` |
| 7 | SQLite 无 `SetMaxOpenConns(1)`，WAL 下写会 `database is locked` | `db.go:15-32` |
| 8 | MinuteCache 锁升降级（RLock→RUnlock→Lock→Unlock→RLock）极脆弱 | `minute_cache.go:133-156` |
| 9 | errgroup 内无 recover，节点 panic 崩进程 | `workflow/engine.go:88-91` |
| 10 | 回测静默吞错误（资金不足/下单失败/风控拒绝无日志） | `engine_cn.go:217, 222, 238` |

### 值得肯定

- **SQL 注入防护到位**：全部查询用 `?` 参数化占位，无字符串拼接。
- **OMS 部分成交裁剪正确**：`oms.go:153-178` 裁剪到订单剩余量与持仓量，有测试守护。
- **涨跌停规则模块化**：`price_limit.go` 按市场区分 ratio，结构清晰（只是没被调用）。

---

## 四、前端 Vue/TypeScript 评审

### 🔴 P0-4 DrawingPanel.vue 翻译工具损坏代码

**位置**：`frontend/src/terminal/panels/DrawingPanel.vue:201, 218, 224`

```typescript
ctx.fill文字(b.y.toFixed(0), 6, b.y - 4)        // line 201
ctx.fill文字((ratios[i] * 100).toFixed(1) + '%', 6, y - 4)  // line 218
ctx.fill文字(d.text || '', p.x, p.y)             // line 224
```

`fill文字` 不是 Canvas API（正确为 `fillText`）。某批量翻译工具把 `fillText` 中的 "Text" 替换成了中文"文字"。**水平线/斐波那契/文字工具运行时全部抛 `TypeError`**。同文件还有中文标识符 `active颜色`、`stepsPer年`、`compute中位数Path`，严重影响可维护性。

### 🔴 P0-5 Canvas 用 CSS 变量作颜色（无效）

**位置**：`frontend/src/terminal/panels/DrawingPanel.vue:142, 146`

```typescript
ctx.fillStyle = 'var(--color-bg-elevated)'    // Canvas 不解析 CSS 变量
ctx.strokeStyle = 'var(--color-border-strong)' // 非法颜色，静默忽略
```

### 🔴 P0-6 ECharts 75 处用 CSS 变量作颜色（全局性视觉缺陷）

**位置**：几乎所有图表面板，如 `MonteCarloPanel.vue:141,145,146`、`CandlestickPanel.vue:308-409`

```typescript
axisLabel: { color: 'var(--color-text-tertiary)', fontSize: 10 },  // 无效
splitLine: { lineStyle: { color: 'var(--color-bg-elevated)' } },   // 无效
```

ECharts Canvas 渲染器不解析 CSS 变量，颜色回退到默认黑色。**深色主题下坐标轴标签、网格线近乎不可见**。

**修复**：封装 `useChartTheme()` composable，通过 `getComputedStyle` 读 CSS 变量计算值返回纯色；主题切换时遍历活动图表 `setOption` 重绘。

### 🔴 P0-7 未生成 wailsjs 类型绑定

**位置**：`frontend/src/lib/wails.ts:42-46`、`main.go:18`

Go 侧 `App` 有 77 个绑定方法，但无 `wails3.yml`、无生成的 `wailsjs/` 目录，全部靠手写字符串 `Call.ByName('main.App.GetVersion', ...)`。方法名拼写错误只能运行时暴露；71 处 `(window as any).go.main.App.XXX` 返回 `any`，类型安全归零。

### P1 关键问题

| # | 问题 | 位置 |
|---|------|------|
| 1 | i18n 双源不同步（session store vs i18n module，两个 localStorage key） | `session.ts:47` vs `i18n/index.ts:7` |
| 2 | `ref<Map>` + 普通函数访问器，响应式依赖丢失风险 | `data.ts:57-71`, `workflow.ts:32` |
| 3 | workflow 连接无校验（重复/自环/类型不匹配） | `WorkflowCanvas.vue:49-61` |
| 4 | 键盘快捷键依赖 div 焦点，实际几乎不可用 | `WorkflowCanvas.vue:99-106` |
| 5 | `import * as echarts` 全量导入，破坏 tree-shaking | `EquityCurvePanel.vue:17` 等 |
| 6 | DockSplitter 用 `e.target.parentElement` 定位容器（不可靠） | `DockSplitter.vue:29-34` |
| 7 | store↔vue-flow 单向同步，拖拽位置不回写 store | `WorkflowCanvas.vue:29-36` |
| 8 | 面板测试仅"能 mount"，无业务逻辑覆盖 | `__tests__/CandlestickPanel.test.ts` |
| 9 | vite.config 与 vitest.config 重复冲突 | `vite.config.ts` vs `vitest.config.ts` |

### 值得肯定

- **DockView 面板系统设计完整**：拖拽分屏、布局持久化、symbolContext 联动。
- **themes.css 设计令牌体系完善**：双主题 + 三密度的 CSS variables 定义清晰（只是用错了地方）。
- **workflow store 序列化/撤销重做**：`workflow.ts` 的 pushHistory/fromWorkflowJSON 实现完整。

---

## 五、Python sidecar 评审

### 🔴 P0-8 RL TradingEnv 组合账户计算完全错误

**位置**：`python/src/ml/rl/env.py:66-69`

```python
trade_cost = abs(self.position - prev_position) * self.portfolio_value * 0.001
self.cash -= trade_cost
self.portfolio_value = self.cash * (1 + self.position * price_return)
self.cash = self.portfolio_value * (1 - abs(self.position))
```

全仓（position=1）后 `cash=0`；下一步平仓时 `cash -= trade_cost` 变负，`portfolio_value` 变负，`reward ≈ -1`。**所有 RL 算法基于错误 reward 训练**。追踪验证（初始 10000，return=0.01）：step1 buy → cash=0, portfolio=10089；step2 hold → cash=-10, portfolio=-10, reward≈-1。

### 🔴 P0-9 版本号三方不一致（sidecar 重启循环 + 测试必失败）

- `python/src/server.py:55` 硬编码 `2026.6.26`
- `python/pyproject.toml:3` `2026.6.17`
- `python/tests/test_factor_engine.py:69` 断言 `2026.6.17`
- `internal/python/sidecar.go:25` `ExpectedSidecarVersion = "2026.6.26"`

Go 端版本校验依赖 server.py 硬编码字符串，任何一次只改一处都触发 sidecar 被 kill 重启循环。

### P1 关键问题

| # | 问题 | 位置 |
|---|------|------|
| 1 | gRPC 方法不设状态码，错误转 `codes.UNKNOWN`，重试策略失效 | `ml/engine.py:52,82` |
| 2 | 无优雅关闭，sidecar 重启切断在途 LLM/训练请求 | `server.py:94` |
| 3 | `ComputeFactorBatch` 假并发（async def 内无 await，gather 退化为串行） | `factor/engine.py:104-151` |
| 4 | DeepEngine 时序训练 `shuffle=True` 数据泄漏 | `deep_engine.py:93` |
| 5 | AlphaMining 无 train/test split，全量训练全量评估 | `alpha_mining/genetic.py:89,96` |
| 6 | RLPredict 是固定桩函数，永远返回 hold（忽略 model_id） | `ml/engine.py:208-220` |
| 7 | DQN sharpe 返回天文数字（单元素 std=0） | `rl/algorithms/dqn.py:94` |
| 8 | Anthropic `finish_reason` 语义未映射，前端解析错乱 | `anthropic_provider.py:157` |
| 9 | LLM provider 全局单例，运行时无法刷新 API key | `openai_provider.py:155` |
| 10 | 无数据缓存，重复请求全量重拉 TDX | `data/fetcher.py` |
| 11 | 无依赖 lock 文件，全 `>=` 无上限，模型跨版本可能无法 load | `pyproject.toml:6-23` |
| 12 | `requirements.txt` 与 `pyproject.toml` 严重分叉 | 两个文件 |

### 值得肯定

- **横截面因子已正确修复**：`engine.py:44-60` 全 panel 计算，有专门回归测试。
- **TreeEngine 时序 split**：`tree_engine.py:105-109` 已用 `shuffle=False`，有回归守护。
- **25 个因子无 look-ahead**：逐一检查 rolling/ewm/shift 均为 trailing。
- **evaluator.py AST 安全求值**：白名单 AST 而非 eval，安全性好。

---

## 六、金融业务正确性评审

### 🔴 P0-10 OMS T+1 锁整个持仓而非份额（拒绝合法交易）

**位置**：`internal/trading/oms.go:37, 175-177, 235`

A股 T+1 正确语义："今日买入的份额不可卖，昨日及之前的份额可卖"。当前实现把"今日是否有过买入"作为"整个持仓是否可卖"判据。实盘场景：用户今天先买入 100 股（`t1Lock=now`），再卖出昨日持仓 100 股 → **被错误拒绝**。

### 🔴 P0-11 实盘涨跌停校验形同虚设

**位置**：`app_trading.go:12-17`；`internal/trading/oms.go:344-366`

`PlaceOrder` 直接调 `oms.PlaceOrder`，**生产代码 0 处调用 `SetPriceLimit`**。`OMS.priceLimits` 永远为空，`CheckPriceLimit` 永远返回 `nil`。实盘不会拦截超涨跌停价的订单。

### 🔴 P0-12 回测前视偏差（同 P0-2，金融视角）

### P1 金融规则缺失

| # | 问题 | 位置 |
|---|------|------|
| 1 | 回测 T+1 每 bar 清空 map，多标的隔离脆弱 | `engine_cn.go:171` |
| 2 | FillOrder 卖出后 pos.PnL 被覆盖为单笔实现 P&L | `oms.go:239, 242` |
| 3 | SquareRootSlippage 实现为平方而非平方根 | `engine_cn.go:42` |
| 4 | 复权方式各 adapter 不一致，tencent 强制 qfq 有前视风险 | `tencent.go:80`; `eastmoney.go:128` |
| 5 | A 股过户费 0.001% 完全缺失 | `engine_cn.go`; `oms.go` |
| 6 | 港股引擎/费用（印花税 0.13% 等）完全缺失 | `backtest.go:142-144` |
| 7 | 退市股票未处理，存在幸存者偏差 | `tushare.go:43` |
| 8 | PDT 仅 informational 且统计逻辑错误 | `engine_us.go:42-67` |
| 9 | 组合回测权益按单 symbol 估值（同 P0-3） | `engine_cn.go:174` |

### 值得肯定

- **A 股印花税 0.05% 已实现**（数值正确，但写死不可配）。
- **历史模拟法 VaR 实现基本正确**（小样本有局限但逻辑对）。
- **部分成交裁剪正确**（有测试守护）。

---

## 七、产品与定位评审

### 7.1 产品定位（合理但野心过大）

- **双模式（彭博式面板 + 可视化工作流）概念有差异化**：竞品（聚宽/米筐/RiceQuant）多为 Web 端策略平台，QuantFlow 的桌面双模式 + 工作流编排有独特价值。
- **目标市场 A 股 > 港股 > 美股 > 加密 优先级清晰**，符合中国用户习惯。
- **AGPL-3.0 开源**：有利于社区，但商业化和 AGPL 兼容性需注意（云服务场景需双授权）。

### 7.2 竞品对比

| 维度 | QuantFlow | 聚宽 | 米筐 | TradingView |
|------|-----------|------|------|-------------|
| 形态 | 桌面终端 | Web 平台 | Web 平台 | Web/桌面 |
| 工作流编排 | ✅ 可视化 | ❌ 代码 | ❌ 代码 | ❌ Pine |
| 实盘交易 | ✅ 多券商 | ✅ | ✅ | 部分 |
| 本地数据隐私 | ✅ | ❌ | ❌ | ❌ |
| 回测可信度 | ❌ P0 缺陷 | ✅ | ✅ | ✅ |
| 成熟度 | 早期 | 成熟 | 成熟 | 成熟 |

**核心矛盾**：QuantFlow 的差异化在于"桌面 + 可视化 + 实盘"，但**回测可信度是量化工具的生命线**，当前 P0 缺陷使其核心价值无法兑现。

### 7.3 功能完整度

README 声称 "Phase 1–11 完成"，但评审发现大量"完成"实为半成品：

- "港股回测引擎 ✅" → 实际无 HKEngine，走 default 分支无市场规则。
- "实盘交易 ✅" → 涨跌停校验不生效，T+1 拒绝合法交易。
- "RL 监控面板 ✅" → RL 训练 reward 完全错误，监控的是垃圾信号。
- "加密 ✅" → 仅现货，无永续合约/资金费率。
- "测试 476 ✅" → 主干测试因死锁无法运行；面板测试仅 mount 级。

**建议**：将 README 的 "完成" 改为更诚实的状态标注（如"基础实现/部分实现/TODO"），避免误导用户。

### 7.4 用户路径与可用性

- **新手路径缺失**：无引导式策略创建向导，用户面对 54 节点 + 46 面板会无所适从。
- **错误反馈缺失**：回测静默吞错误，用户无法诊断"策略为何不交易"。
- **密钥管理**：LLM provider 改 key 需重启 sidecar，桌面终端可用性差。

---

## 八、修复优先级矩阵

### 第一优先：立即修复（P0，阻断核心价值）

| # | 问题 | 修复成本 | 影响 |
|---|------|---------|------|
| 1 | OMS 自死锁（`getName` 去 RLock） | 5 分钟 | 解除全包测试阻塞 |
| 2 | 回测前视偏差（成交改 `bar.Open`/T+1 开盘） | 中 | 回测收益可信 |
| 3 | 多标的权益用全量价格 map | 中 | 指标正确 |
| 4 | OMS T+1 改"可用份额"模型 | 中 | 实盘不拒绝合法交易 |
| 5 | 实盘下单前调用 `SetPriceLimit` | 小 | 涨跌停校验生效 |
| 6 | DrawingPanel `fill文字`→`fillText` + Canvas 颜色 | 10 分钟 | 绘图工具可用 |
| 7 | ECharts CSS 变量改 `useChartTheme()` | 中 | 全图表可读 |
| 8 | RL TradingEnv 组合计算重写 | 中 | RL 训练有效 |
| 9 | 版本号统一从 pyproject.toml 读取 | 小 | sidecar 不重启循环 |

### 第二优先：本轮迭代修复（P1）

10. 统一复权策略为后复权（所有 adapter 尊重传入 fqfactor）
11. 补齐 A 股过户费 + 港股费用模型与 HKEngine
12. 修复 FillOrder pos.PnL 覆盖 + 滑点平方根公式
13. 资金账本单一源 + 佣金配置统一
14. Sharpe/Calmar 公式统一为标准定义
15. SQLite `SetMaxOpenConns(1)` + `synchronous=NORMAL`
16. CacheKey 改 JSON 序列化
17. errgroup 加 recover + 回测错误加日志
18. 生成 wailsjs 绑定，消除 71 处 `as any`
19. 统一语言状态源
20. workflow 连接校验 + 快捷键改 window 监听
21. DeepEngine `shuffle=False` + AlphaMining 加 holdout
22. gRPC 方法设状态码 + 优雅关闭
23. 生成依赖 lock 文件
24. 退市股票处理（消除幸存者偏差）

### 第三优先：后续优化（P2）

涨跌停四舍五入、ST 股 ±5%、订单 ID、VaR 索引、方差样本估计、加密合约、PDT 阻断、依赖体积优化、tsconfig 严格选项等。

---

## 九、总体评价与建议

### 9.1 优点

1. **架构选型合理**：Go + Wails + Vue + Python gRPC + SQLite 的组合适合桌面量化终端，单二进制部署、进程隔离得当。
2. **工程结构清晰**：internal/ 按领域分包、frontend 面板/工作流分离、python 按服务分层，可读性好。
3. **覆盖面广**：54 节点、46 面板、25 适配器、4 LLM、4 AgentProfile，工作量巨大。
4. **部分模块质量高**：横截面因子修复、TreeEngine 时序 split、SQL 注入防护、AST 安全求值等。
5. **测试意识强**：476 测试（虽部分失效），OMS/涨跌停/T+1 都有测试（虽未覆盖关键路径）。

### 9.2 致命问题

1. **"完成"定义过宽**：多个 P0 表明核心功能未真正可用（回测不可信、实盘涨跌停失效、RL 训练无效）。
2. **金融正确性是量化工具生命线，当前不达标**：前视偏差 + 多标的权益错误 + T+1 语义错误，回测结果不可作为实盘依据。
3. **工程纪律不足**：go.mod 版本号错误、版本号四处不一致、依赖无 lock、wailsjs 未生成、翻译工具损坏代码未被 CI 拦截。
4. **CI gate 缺失**：`go test` 因死锁本应红但未被发现；`vue-tsc --noEmit` 未纳入 test 脚本；Python 覆盖率无门禁。

### 9.3 核心建议

1. **立即修复 9 个 P0**，否则项目无法用于任何真实场景。
2. **建立 CI gate**：`go vet && go test` + `vue-tsc --noEmit && vitest` + `pytest --cov`，三者全绿才能合并。
3. **统一版本号管理**：单一源（pyproject.toml）+ Makefile target 同步 Go/Python/README。
4. **诚实标注功能状态**：README 改用"基础实现/部分实现/TODO"，避免误导。
5. **金融正确性专项**：建议引入一个"金融规则一致性测试套件"，覆盖 T+1/涨跌停/前视/复权/费用的端到端验证。
6. **冻结 Phase 12 新功能**，集中 1–2 个迭代修复 P0/P1，再推进新功能。

---

## 附录：评审方法说明

- 后端 Go：采样 20+ 文件 + 实际 `go test` 运行复现死锁
- 前端：采样 55 面板/18 store/65 测试 + grep 统计 `as any`(80)/`window as any`(71)/CSS 变量(75)
- Python：采样 gRPC/factor/ml/llm/data/research 全链路 + 依赖对比
- 金融：逐条核对 12 项市场规则，含 T+1/涨跌停/费用/前视/幸存者/复权/PDT/资金费率

*报告完。如需针对任一 P0/P1 提供补丁级修复代码，请指明。*

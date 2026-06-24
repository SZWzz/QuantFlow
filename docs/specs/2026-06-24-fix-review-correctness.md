# Fix Review Findings — Correctness & Finance Bugs from 2026-06-24 Audit

> 评审报告 ref: [docs/QuantFlow_项目评审报告.html](../QuantFlow_项目评审报告.html)
> 审查方法：代码专家 + 金融专家双视角，后端 Go / Python sidecar / 前端 Vue 三部分走查
> 本 spec 已逐条核对实际代码行号，并修正了初版评审中两处过激判断（见下方"严重性修正"）

## Motivation

2026-06-24 评审给出综合 7.4/10。工程骨架（架构 8.5 / 规范 8.5 / 测试 8.0）属上乘，但金融正确性与若干并发/一致性问题会直接导致**回测结果失真**或**账本对不上**。量化产品的核心价值是回测可信度，这些问题不修，项目停留在"演示级"。

本 spec 收录经代码核对的 4 个 P0 + 7 个 P1 + 6 个 P2，共 17 项修复，按"先正确性、再防护、后工程打磨"排序。每项给出根因、修复方案、影响文件、验收标准。

### 严重性修正（相对初版评审报告）

| 项 | 初版判断 | 核对后 | 修正理由 |
|----|---------|--------|---------|
| T+1 锁定 | P0 多标的失效 | **P1 非日频失效** | 日频下"读取在 bar 开头、清空在 bar 末尾"巧合正确；分钟频/日内频下每个 bar 清空致 T+1 失效。多标的日频实际成立。 |
| CacheKey map 序列化 | P0 可能错误命中 | **P1 cache miss** | map %v 顺序随机产生**不同 key**，最坏是缓存永不命中（性能），不会碰撞出错误结果。 |

核对后仍为 P0 的：OMS FillOrder 裁剪顺序、涨跌停缺失。

## Design

### 组件交互总览

```
回测可信度链路（修复重点）：
  DataLoader → [Backtest Engine] → OMS.FillOrder → Portfolio/Positions → Metrics
       │              │                    │              │
       │              ├─ T+1 锁定(t1Tracker)  ├─ 订单账本(FilledQty) ─┐
       │              ├─ 涨跌停校验(缺失)     └─ 持仓账本(Quantity) ──┘ 须一致
       │              ├─ look-ahead 防护(缺失)
       │              └─ goto/魔法数字
       │
  Workflow: Engine → CacheKey → LRU   (map 序列化非确定 → cache miss)
  Python:   FactorEngine → 单标的过滤 → 横截面因子失效
            TreeEngine → 无验证集 → 过拟合/泄漏
  Frontend: data.ts ref<Map> → 响应性失效; wailsjs 绑定缺失 → IPC 全 any
```

---

### P0-1: OMS FillOrder 卖出裁剪顺序导致账本不一致

**根因**：[`internal/trading/oms.go:122-124`](internal/trading/oms.go#L122) 先用**未裁剪**的 `fillQty` 更新订单账本（`FilledQty` / `FilledAvgPrice`），随后 [`oms.go:160-161`](internal/trading/oms.go#L160) 才把 `fillQty` 裁剪到 `pos.Quantity`。结果：订单记录的成交量 > 实际持仓变动，P&L 与持仓账本对不上。当卖出量超过持仓（回测/风控漏判时）尤为严重。

**修复**：在更新订单账本**之前**完成 fillQty 裁剪。卖出分支需要持仓信息来裁剪，而当前裁剪发生在 position 更新段——需把裁剪上移到 `line 120` 之后、`line 122` 之前。买入无需裁剪（已由 remainingQty 保证），仅卖出路径需改。

```go
// oms.go FillOrder 内，remainingQty 裁剪后、订单账本更新前：
if order.Side == SideSell {
    pos, ok := o.positions[order.Symbol]
    if !ok || pos.Quantity <= 0 {
        return nil, fmt.Errorf("fill %s: no position to sell for %s", order.ID, order.Symbol)
    }
    if fillQty > pos.Quantity {
        fillQty = pos.Quantity   // 先裁剪
    }
}
// 此处 fillQty 已是最终值，再用它更新 order.FilledQty / FilledAvgPrice
totalValue := order.FilledAvgPrice*order.FilledQty + fillPrice*fillQty
order.FilledQty += fillQty
order.FilledAvgPrice = totalValue / order.FilledQty
...
// position 更新段移除重复裁剪（line 160-165），直接用已裁剪 fillQty
```

**影响文件**：`internal/trading/oms.go`（约 15 行调整）

**验收**：
- [ ] 卖出量 > 持仓时，`order.FilledQty == pos 实际减少量`
- [ ] 新增测试：卖出 200 股但持仓仅 150，断言 order.FilledQty=150、pos.Quantity=0、Trade.Quantity=150
- [ ] 现有 OMS 测试全绿

---

### P0-2: A 股回测缺失涨跌停限制

**根因**：[`internal/backtest/engine_cn.go:12`](internal/backtest/engine_cn.go#L12) 注释宣称"±10% (main board) or ±20% (ChiNext/STAR)"，但全文**无任何实现**。回测可在涨停价买入、跌停价卖出，产生现实不可能的交易，系统性高估策略收益。README 同样宣称支持该规则——属于虚假声明。

**修复**：在 CN 引擎撮合前（`processCNBuySignal` / `processCNSellSignal` 调用前）增加涨跌停校验。需前一日收盘价作为基准（首日无基准则跳过）。

```go
// 新增 priceLimitChecker（internal/backtest/price_limit.go）
type PriceLimitRule struct {
    Ratio   float64 // 0.10 主板, 0.20 创业板/科创板, 0.05 ST
    Prefix  string  // 板块识别：300/688/689 → 0.20；*ST → 0.05
}

// 引擎维护 prevClose map[string]float64（上一交易日收盘）
// 撮合前：
limitUp   := prevClose[symbol] * (1 + rule.Ratio)
limitDown := prevClose[symbol] * (1 - rule.Ratio)
if bar.Close >= limitUp   && side == buy  { reject / 部分成交 }
if bar.Close <= limitDown && side == sell { reject / 部分成交 }
```

板块识别按 symbol 代码前缀：`300/301`(创业板)、`688/689`(科创板) → ±20%；`*ST`/`ST` → ±5%；其余主板 ±10%。首日（无 prevClose）不限制。

**影响文件**：
- 新增 `internal/backtest/price_limit.go`（校验器 + 板块识别）
- `internal/backtest/engine_cn.go`（维护 prevClose、撮合前调用校验）
- `internal/backtest/engine_cn_test.go`（新增涨停买入被拒、跌停卖出被拒、创业板 ±20% 用例）

**验收**：
- [ ] 主板涨停价买入被拒，回测记录中无该笔成交
- [ ] 创业板（300xxx）±20% 生效
- [ ] ST 股 ±5% 生效
- [ ] 首日无 prevClose 时不限制（不 panic）

---

### P0-3: 横截面因子经标准 RPC 路径失效

**根因**：[`python/src/factor/engine.py:44`](python/src/factor/engine.py#L44) 先按 `df[df["symbol"]==symbol]` 过滤到**单标的**再调用因子；而 [`cross_sectional.py:33,57`](python/src/factor/cross_sectional.py#L33) 的 `zscore` / `rank` 在单标的上 `x.std()==0` 退化为 `0` / `0.5`。横截面因子是 Alpha 核心类别，经标准路径根本无法工作——功能性失效。

**修复**：因子引擎按因子**计算维度**（时序 vs 横截面）分发不同路径。横截面因子需在**完整截面**（多标的同日）上计算，不应先过滤单标的。

```python
# engine.py ComputeFactors：按 factor.kind 分发
CROSS_SECTIONAL = {"zscore", "rank", "cross_sectional_rank", ...}
for fname in request.factors:
    spec = self.registry[fname]
    if spec.kind in CROSS_SECTIONAL:
        # 在完整 panel（多 symbol × 多 date）上计算，再按 symbol 切片返回
        out[fname] = self._compute_cross_sectional(df, spec)
    else:
        # 时序因子：按 symbol 分组（现有逻辑）
        for sym, sub in df.groupby("symbol"):
            out[fname][sym] = spec.fn(sub)
```

proto 层 `FactorRequest` 已支持多 symbol（`repeated string symbols`），无需改 schema；只需 engine 不再逐 symbol 过滤后调用横截面因子。

**影响文件**：
- `python/src/factor/engine.py`（分发逻辑）
- `python/src/factor/cross_sectional.py`（确认在完整 panel 上行为）
- `python/tests/test_factor_engine.py`（新增多标的 zscore 非零、rank 分布合理用例）

**验收**：
- [ ] 3+ 标的同日 zscore 不再全为 0，且跨标的之和为 0（标准化特性）
- [ ] rank 在多标的上产生 1..N 的合理分布
- [ ] 时序因子（pct_change 等）不受影响

---

### P0-4: ML 训练无验证集划分（过拟合/数据泄漏）

**根因**：[`python/src/ml/tree_engine.py:110-122`](python/src/ml/tree_engine.py#L110) 训练后用**同一份 X** 计算 `train_accuracy` / `train_rmse` 作为评估指标，无 train/val split、无 TimeSeriesSplit、无早停。时序数据若做随机 shuffle 会泄漏未来信息。模型注册表仅内存 `self._models={}`，重启即丢失，无版本/特征追踪。

**修复**：
1. 强制 TimeSeriesSplit（前向滚动）或按时间切分 train/val（默认 80/20，不可 shuffle）
2. 评估指标在**验证集**上计算
3. 模型元数据（版本、特征列、指标、训练时间）落 SQLite（Go 侧 `models` 表 + gRPC 返回 model_id）

```python
# tree_engine.py train
tscv = TimeSeriesSplit(n_splits=5) if len(X) > 200 else None
if tscv:
    scores = cross_val_score(model, X, y, cv=tscv, scoring="neg_mean_squared_error")
X_train, X_val, y_train, y_val = train_test_split(X, y, test_size=0.2, shuffle=False)
model.fit(X_train, y_train)
val_pred = model.predict(X_val)
val_rmse = mean_squared_error(y_val, val_pred, squared=False)  # 报 val，不报 train
```

**影响文件**：
- `python/src/ml/tree_engine.py`（切分 + 验证集评估）
- `python/src/ml/registry.py`（新增：落库 + 版本管理，或 Go 侧 `internal/storage/migrations` 加 `models` 表）
- `internal/python/proto/ml.proto`（TrainResponse 增加 `model_id` / `val_metrics` 字段）
- `python/tests/test_tree_engine.py`（验证集指标非空、shuffle=False 断言）

**验收**：
- [ ] TrainResponse 返回 val_rmse（非 train_rmse）
- [ ] 切分 `shuffle=False`（时序不泄漏）
- [ ] 模型重启后可通过 model_id 从 SQLite 取回元数据
- [ ] 现有预测测试不受影响

---

### P1-1: T+1 锁定在非日频数据下失效

**根因**：[`internal/backtest/engine_cn.go:121`](internal/backtest/engine_cn.go#L121) 在**每个 bar 末尾** `e.t1Lock.locked = make(map[string]float64)` 清空全部锁定。日频下因"读取在 bar 开头(line 73)、清空在 bar 末尾"而巧合正确；但分钟频/日内频下，当天买入后下一个分钟 bar 即解锁，T+1 失效（当天买当天可卖）。实现脆弱依赖 bar 频率假设。

**修复**：按**交易日切换**清空锁定，而非每个 bar 清空。引擎维护 `lastTradeDate`，仅当 `bar.Date` 的交易日部分变化时才清空当日锁定。

```go
// engine_cn.go 循环内
if isSameTradeDay(bar.Date, e.lastDate) {
    // 同一交易日，保留锁定
} else {
    e.t1Lock.locked = make(map[string]float64) // 新交易日，清空昨日锁定
    e.lastDate = bar.Date
}
```

`isSameTradeDay` 比较 `bar.Date` 的 date 部分（OHLCVBar.Date 若为字符串 "2026-06-24" 则直接比较；若含时间则截断到日）。多标的同日 bar 共享同一 `lastDate`，锁定正确累积。

**影响文件**：`internal/backtest/engine_cn.go`（约 8 行）

**验收**：
- [ ] 日频回测结果与修复前一致（回归不破坏）
- [ ] 分钟频回测：当日买入当日不可卖（构造 09:30 买、10:00 卖场景，卖出被拒）
- [ ] 次日开盘可卖

---

### P1-2: 无 look-ahead / survivorship bias 防护

**根因**：回测引擎、因子计算未见显式未来函数检测、数据对齐校验、退市股票剔除处理。CLAUDE.md 把"防 look-ahead bias"列为关键文档要求，但实现层缺失。这是量化回测可信度的根基。

**修复**：
1. **因子层**：横截面/时序因子计算强制 `shift(1)` 防当日未来引用（信号用昨日因子值）
2. **回测层**：新增 `BiasGuard`——加载股票池时保留退市标的（防 survivorship）；提供 point-in-time 数据访问接口（`GetAsOf(date)` 只返回 ≤ date 的数据）
3. **校验**：回测启动前扫描因子矩阵，若某日因子值依赖了该日之后的 OHLCV，报错中止

**影响文件**：
- `internal/backtest/bias_guard.go`（新增）
- `internal/backtest/engine_cn.go` / `engine_us.go`（启动前调用 BiasGuard）
- `python/src/factor/base.py`（shift 强制）
- 对应测试

**验收**：
- [ ] 因子值引用未来数据时回测报错中止
- [ ] 退市标的保留在回测股票池中（不剔除）
- [ ] 文档：在 `docs/specs/` 补 bias 防护说明

---

### P1-3: Workflow CacheKey map 序列化非确定

**根因**：[`internal/workflow/cache.go:33`](internal/workflow/cache.go#L33) `fmt.Sprintf("%s:%v", nodeID, inputs)` 对 `map[string]any` 做 `%v`，Go map 迭代顺序随机，相同输入产生不同 key → 缓存永不命中（性能损失），非错误命中。

**修复**：对 inputs 按 key 排序后确定性序列化。

```go
func CacheKey(nodeID string, inputs map[string]any) string {
    keys := make([]string, 0, len(inputs))
    for k := range inputs { keys = append(keys, k) }
    sort.Strings(keys)
    var b strings.Builder
    b.WriteString(nodeID)
    for _, k := range keys {
        b.WriteByte('|'); b.WriteString(k); b.WriteByte('=')
        // 嵌套 map/struct 需递归确定性序列化，最简方案用 json.Marshal（map key 已排序）
        enc, _ := json.Marshal(inputs[k])
        b.Write(enc)
    }
    h := sha256.Sum256([]byte(b.String()))
    return fmt.Sprintf("%x", h[:16])
}
```

**影响文件**：`internal/workflow/cache.go`（约 12 行）+ `cache_test.go`

**验收**：
- [ ] 相同 inputs 两次调用 CacheKey 结果一致
- [ ] 1000 次随机 map 迭代下 key 稳定
- [ ] 缓存命中率测试：同节点同输入第二次执行命中缓存

---

### P1-4: Calmar Ratio 量纲不匹配

**根因**：[`internal/portfolio/risk.go:94-97`](internal/portfolio/risk.go#L94) `CalmarRatio = mean*252 / maxDD`，分子是绝对收益率（如 0.15），分母 maxDD 是比例值（如 0.2），量纲不一致。标准 Calmar = CAGR / MaxDD，两者同量纲。

**修复**：分子改用 CAGR（与 MaxDD 同为比例），或分子分母都用绝对金额。推荐前者：

```go
// risk.go
cagr := math.Pow(finalEquity/initialEquity, 1/years) - 1  // 比例
calmar := cagr / maxDD
```

**影响文件**：`internal/portfolio/risk.go`（约 6 行）+ 测试

**验收**：
- [ ] 单元测试：已知 equity curve 的 Calmar 与公式手算一致
- [ ] maxDD=0 时返回 0（除零保护）

---

### P1-5: 前端 ref&lt;Map&gt; 响应性失效

**根因**：[`frontend/src/stores/data.ts:57-58`](frontend/src/stores/data.ts#L57) 用 `ref(new Map())` 后调 `.set()` 更新，Vue 3 对 Map 的 `.set()` 不触发响应式更新，quotes/ohlcvCache 变更面板感知不到。[`workflow.ts:27`](frontend/src/stores/workflow.ts#L27) `nodeStatuses` 同样问题。

**修复**：改用 `reactive(new Map())`（Vue 3.4+ 支持 Map/Set 响应式），或用普通对象 + 触发 `.value` 重新赋值。

```ts
// data.ts
const quotes = reactive(new Map<string, Quote>())  // 替代 ref(new Map())
quotes.set(sym, q)  // reactive Map 的 set 触发更新
```

**影响文件**：`frontend/src/stores/data.ts`、`workflow.ts`（约 4 处）+ 组件测试

**验收**：
- [ ] 实时行情面板 quote 更新后 UI 刷新（vitest 断言渲染）
- [ ] workflow 节点状态变更后画布节点颜色更新

---

### P1-6: IPC 双通道未统一，wailsjs 绑定缺失

**根因**：[`frontend/src/lib/wails.ts`](frontend/src/lib/wails.ts) 仅 2 处引用，9 个文件直接用 `(window as any).go.main.App.*`。`wailsjs/` 目录不存在，Go 绑定类型未生成，IPC 入口 `Call(...): Promise<any>` 全程无类型。

**修复**：
1. 运行 `wails generate module` 生成 `frontend/wailsjs/` 绑定
2. 统一所有 IPC 调用走 `wailsjs/main/App.*`（有类型）
3. 移除手写 `lib/wails.ts` 和 `window.go` 直调

**影响文件**：9 个 panel/store 文件（批量替换）+ 新增 `wailsjs/`

**验收**：
- [ ] `grep -r "window as any).go" frontend/src` 无结果
- [ ] `grep -r "lib/wails" frontend/src` 无结果
- [ ] vue-tsc --noEmit 通过（IPC 调用有类型）

---

### P1-7: 收益率 simple/log 混用隐患

**根因**：因子全用简单收益 `pct_change`，而 `garch.py` docstring 声称 "log returns" 却不做变换，依赖调用方。跨模块累计会出错。

**修复**：统一收益率为对数收益（量化标准），或显式标注每个模块用哪种并提供转换函数。推荐：因子层统一 `log_return`，提供 `to_simple()` 转换。

**影响文件**：`python/src/factor/*.py`、`python/src/ml/garch.py` + 测试

**验收**：
- [ ] 所有因子 docstring 标注收益类型
- [ ] garch 实际使用 log returns（与 docstring 一致）

---

### P2 项（6 条，简列）

| # | 问题 | 位置 | 修复 |
|---|------|------|------|
| P2-1 | RL 模块不完整：RLPredict 硬编码 hold、sac NotImplementedError、DQN Sharpe std=0 退化 | `python/src/ml/engine.py:211`、`sac.py:19`、`dqn.py:94` | 补齐或从 README 移除"RL×3"声明 |
| P2-2 | 魔法数字散落：252/0.0005/0.0003/256/5000 | backtest/各处 | 提为命名常量 `TradingDaysPerYear` 等 |
| P2-3 | backtest 3 处 goto | `engine_cn.go:98,119` | 结构化控制流替代 |
| P2-4 | 数据适配器缺统一限流/超时 | 30 个 adapter 仅 EastMoney 有限流 | AdapterRegistry 层注入统一 limiter+timeout |
| P2-5 | 版本号不一致 | `go.mod` go 1.26.4(不存在)、config.yaml 0.0.1 vs README 2026.6.19 | 统一为 go 1.22.x、版本对齐 |
| P2-6 | HK/CRYPTO 回测引擎缺失 | README 宣称四市场，仅 CN/US 实现 | 实现 HK(T+2/港股通)、CRYPTO(资金费率) 或更新 README |

## 数据流变更

- **OMS**：卖出路径裁剪上移，订单账本与持仓账本统一以裁剪后 fillQty 计算
- **回测**：新增 `prevClose`（涨跌停）、`lastDate`（T+1 按日清空）、`BiasGuard`（启动前校验）
- **因子**：横截面因子不再逐 symbol 过滤，在完整 panel 上计算
- **ML**：训练强制时序切分，模型元数据落 SQLite `models` 表
- **前端**：`ref<Map>` → `reactive(Map)`；IPC 走 wailsjs 绑定

## Schema 迁移

新增 `models` 表（ML 模型元数据），作为新 migration 追加（不修改历史 migration）：

```sql
-- migration N+1
CREATE TABLE IF NOT EXISTS models (
    id           TEXT PRIMARY KEY,          -- uuid
    name         TEXT NOT NULL,
    kind         TEXT NOT NULL,             -- tree / linear / rl
    features     TEXT NOT NULL,             -- JSON array
    val_metrics  TEXT NOT NULL,             -- JSON {rmse, accuracy, ...}
    trained_at   INTEGER NOT NULL,          -- unix ts
    artifact_path TEXT                       -- 模型文件路径（可选）
);
CREATE INDEX IF NOT EXISTS idx_models_name ON models(name);
```

## 验收标准（整体）

- [ ] P0 四项全部修复，新增测试全绿
- [ ] `go test ./...` / `vitest run` / `pytest` 三套测试全绿，无回归
- [ ] 已知多标的日频回测结果与修复前一致（P1-1 T+1 回归保护）
- [ ] `grep -r "window as any).go" frontend/src` 无结果
- [ ] README 中"涨跌停""RL×3""HK/CRYPTO 回测"等声明与实现一致（不一致则更新 README）
- [ ] CHANGELOG.md 记录所有变更，scope 标签齐全
- [ ] 版本号三处（package.json / README / CHANGELOG）同步

## 风险 / 权衡

| 风险 | 缓解 |
|------|------|
| OMS 裁剪顺序改动影响实盘路径 | 实盘 broker 回调也走 FillOrder，需同步验证 Alpaca/Binance 路径；先在 Paper 模式回归 |
| 涨跌停板块识别按代码前缀，可能漏配新股/特殊标识 | 提供 override 配置；首版覆盖主板/创业板/科创板/ST，北交所(8/4开头)±30% 作为 P2 后续 |
| 横截面因子改动 proto 行为，旧前端调用可能不兼容 | proto 字段不变（已支持 repeated symbols），仅 engine 内部分发逻辑变；保持返回结构兼容 |
| BiasGuard 误报导致正常回测中止 | 提供 `skip_bias_check` 配置项；默认开启但可关 |
| ML 时序切分对小样本(<200 条)不稳定 | 小样本 fallback 到简单 80/20 时间切分，不做 TimeSeriesSplit |
| 前端 reactive(Map) 需 Vue 3.4+ | 确认 package.json vue 版本 ≥ 3.4；否则用 ref + 整体替换策略 |

## 实施顺序（建议分 sub-phase）

1. **Phase A（P0 正确性）**：P0-1 OMS → P0-2 涨跌停 → P0-3 横截面因子 → P0-4 ML 验证集。每项独立 plan + TDD。
2. **Phase B（P1 防护与一致性）**：P1-1 T+1 → P1-2 BiasGuard → P1-3 CacheKey → P1-4 Calmar → P1-5 前端响应性 → P1-6 IPC 统一 → P1-7 收益率统一。
3. **Phase C（P2 工程打磨）**：按需排期，可与功能开发并行。

每个 sub-phase 一个 plan 文件（`docs/superpowers/plans/2026-06-24-fix-review-<phase>.md`），subagent-driven 执行，phase 间设 review gate。

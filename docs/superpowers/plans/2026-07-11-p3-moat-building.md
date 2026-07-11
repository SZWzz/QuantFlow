# P3 护城河建设 — 实施计划

> 日期: 2026-07-11 | 参考 Spec: `docs/specs/2026-07-11-p3-moat-building.md`

## 阶段一: 可转债分析模块 (1 周) — 先做，投入产出比最高

### Task 1: EastMoney 可转债数据适配器

**新建文件**: `internal/market/adapters/eastmoney_cb.go`

**步骤**:
1. 对接东方财富可转债接口: `https://datacenter.eastmoney.com/api/data/v1/get`
   - 可转债列表 + 实时行情
   - 转股价、转股溢价率、纯债价值（YTM）、到期日
   - 回售触发价格、强赎触发价格、下修触发价格
2. 实现 `CBQuote` 数据结构:
```go
type CBQuote struct {
    Code         string  // 转债代码 (123xxx/113xxx/128xxx)
    Name         string  // 转债名称
    StockCode    string  // 正股代码
    Price        float64 // 转债价格
    StockPrice   float64 // 正股价格
    PremiumRate  float64 // 转股溢价率 (%)
    ConversionPrice float64 // 转股价
    YTM          float64 // 到期收益率
    BondValue    float64 // 纯债价值
    PutPrice     float64 // 回售触发价
    CallPrice    float64 // 强赎触发价
    IssueDate    string  // 发行日
    MaturityDate string  // 到期日
    StockChange  float64 // 正股涨跌幅
    CBChange     float64 // 转债涨跌幅
}
```
3. 编写 `eastmoney_cb_test.go`

**提交**: `feat(market): EastMoney convertible bond data adapter`

---

### Task 2: 可转债分析器

**新建文件**: `internal/research/cb_analyzer.go`

**步骤**:
1. 实现双低排名: `score = price + premiumRate`
2. 实现估值模型: `fairValue = bondValue + optionValue`
   - bondValue: 按 YTM 折现未来现金流
   - optionValue: 简化 BS 模型（正股为标的，转股价为行权价）
3. 实现下修概率估算: 基于财务压力指标
4. 编写 `cb_analyzer_test.go`

**提交**: `feat(research): CB analyzer with dual-low ranking and fair value estimation`

---

### Task 3: 可转债面板 + Workflow 节点

**新建文件**:
- `frontend/src/terminal/panels/ConvertibleBondPanel.vue`
- `internal/workflow/nodes/cb_scanner.go`

**步骤**:
1. ConvertibleBondPanel: 双低排名表（sortable table）+ 单券分析（条款时间线）
2. `cb_scanner` 节点: 参数（maxPrice, maxPremium, minYears）→ 输出转债列表
3. 编写测试

**提交**: `feat(frontend,workflow): CB panel and scanner workflow node`

---

## 阶段二: AI Agent + Workflow (2 周) — 核心护城河

### Task 4: 策略生成 Agent

**新建文件**: `internal/ai/strategy_agent.go`

**步骤**:
1. 构建节点 catalog prompt:
   - 遍历 NodeRegistry，提取每个节点的: type, category, params schema, input/output ports
   - 格式化为 LLM-friendly 的 JSON schema
2. 策略生成 prompt 模板:
```
你是一个量化策略专家。根据用户描述，使用以下可用节点生成一个 workflow。
只输出合法的 JSON，不要解释。

可用节点: [{catalog}]

用户需求: {userInput}

输出格式:
{
  "name": "策略名称",
  "nodes": [{id, type, params}],
  "edges": [{source, sourcePort, target, targetPort}]
}

规则:
1. edges 必须形成 DAG，不能有环
2. 每个 edge 的 sourcePort 类型必须与 targetPort 兼容
3. 只使用上面列出的节点类型
```
3. 验证生成的 Workflow: TopoSort + ValidateEdgeTypes
4. 不合法 → 注入错误反馈 → 重新生成（最多 3 轮）
5. 编写测试 mock LLM 响应

**提交**: `feat(ai): strategy generation agent — NL to workflow DAG`

---

### Task 5: 策略迭代 Agent

**新建文件**: `internal/ai/strategy_iteration.go`

**步骤**:
1. 读取回测结果 (Sharpe, MaxDD, WinRate, ProfitFactor)
2. 分析 prompt:
```
回测结果: {metrics}
当前参数: {params}
用户目标: {userGoal}

分析失败原因并建议参数调整。输出:
{
  "analysis": "一句话分析",
  "changes": [{param: "参数名", from: 旧值, to: 新值, reason: "原因"}]
}
```
3. 应用参数变更 → 重新执行 workflow → 比较结果
4. 最多 5 轮，保留 Sharpe 最高的参数组合
5. 编写测试

**提交**: `feat(ai): strategy iteration agent — auto param tuning via backtest feedback`

---

### Task 6: AI Strategy 前端面板

**新建文件**: `frontend/src/terminal/panels/AIStrategyPanel.vue`

**步骤**:
1. 输入区: 自然语言 textarea + 示例 prompts
2. 预览区: 复用 vue-flow 组件渲染生成的 DAG
3. 操作区: 回测 / 修改 / 保存 / 导出 Python 按钮
4. 编写测试

**提交**: `feat(frontend): AI Strategy panel — NL input, DAG preview, one-click backtest`

---

## 阶段三: 期权定价升级 (1 周)

### Task 7: Binomial Tree + Greeks

**修改文件**: `python/src/analytics/options.py`

**步骤**:
1. 实现 `binomial_price()` — CRR 树，支持美式/欧式期权
2. 实现 `compute_greeks()` — Delta/Gamma/Theta/Vega/Rho
3. 实现 `implied_vol()` — Newton-Raphson 寻根
4. Go 端 gRPC 调用封装
5. 编写测试:
   - 验证 BS 与 Binomial (steps=500) 结果偏差 < 0.1%
   - 验证 Put-Call Parity
   - 验证 Greeks 数值偏导与解析解一致

**提交**: `feat(python): Binomial Tree pricing, Greeks, and implied volatility`

---

### Task 8: IV 曲面面板

**新建文件**: `frontend/src/terminal/panels/VolatilitySurfacePanel.vue`

**步骤**:
1. 从期权链数据构建 IV 矩阵 (strike × expiry)
2. ECharts 3D surface 图渲染
3. 可旋转/缩放/点击查看详情
4. 增强 OptionChainPanel 增加 Greeks 列

**提交**: `feat(frontend): volatility surface 3D chart and option chain Greeks`

---

## 阶段四: 富途 OpenD 实盘接入 (1 周)

### Task 9: FutuAdapter 实现

**修改文件**: `internal/trading/brokers/futu.go`

**步骤**:
1. 实现 Connect: TCP 连接 `localhost:11111` → 握手 → 解锁交易
2. 实现 SubmitOrder: 港股/美股/A 股下单
3. 实现 GetPositions: 持仓查询
4. 实现 GetAccount: 账户信息
5. 实现 CancelOrder: 撤单
6. 编写集成测试 (`//go:build integration`)

**提交**: `feat(broker): Futu OpenD broker adapter — full Broker interface`

---

### Task 10: 富途配置与文档

**修改文件**: `frontend/src/terminal/panels/SettingsPanel.vue`

**步骤**:
1. Settings 面板新增富途配置: OpenD 地址、端口、交易密码
2. 编写 README 中的富途配置指南

**提交**: `feat(broker): Futu OpenD configuration UI and setup guide`

---

## 阶段五: CHANGELOG (Task 11)

**提交**: `chore: update CHANGELOG for P3 moat building`

---

## 执行顺序

```
阶段一: 可转债 (1 周)
  Task 1 → Task 2 → Task 3

阶段二: AI Agent (2 周，可在阶段一之后开始)
  Task 4 → Task 5 → Task 6

阶段三: 期权 (1 周，可与阶段二并行)
  Task 7 → Task 8

阶段四: 富途 (1 周，可与阶段二/三并行)
  Task 9 → Task 10

若人力允许，阶段二+三+四可交叉并行推进。
建议优先级: 可转债 > AI Agent > 期权 > 富途
```

## 总时间线

```
Week 1: 可转债模块
Week 2-3: AI Agent 策略生成与迭代
Week 4: 期权定价升级
Week 4-5: 富途 OpenD + 收尾
```

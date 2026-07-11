# P3 — 护城河建设：AI Agent、可转债、期权定价与富途接入

> 日期: 2026-07-11 | 优先级: P3 | 预计工作量: 4-6 周

## Motivation

P0-P2 解决正确性和可用性问题后，P3 建设长期护城河——这些功能不能靠短期冲刺完成，需要持续投入：

1. **AI Agent + Workflow Engine = 自然语言生成策略** — 这是量化终端产品的终极形态，竞争对手难以复制
2. **可转债分析** — A 股可转债市场规模超 8000 亿，策略类型丰富（打新、双低、轮动、套利），目前完全缺失
3. **期权定价升级** — 当前仅有基础实现，缺少 Binomial Tree、Greeks 曲面、隐含波动率计算
4. **富途 OpenD 实盘接入** — 华语用户最重要的实盘交易渠道，`futu.go` 目前全 stub

## Design

### Part A: AI Agent → Workflow DAG（核心护城河）

**用户流程**:
```
用户输入: "找出过去20天涨幅超过10%但RSI低于30的A股，买入前3只，止损5%"
    ↓
LLM 解析意图 → 生成 Workflow JSON
    ↓
Workflow: [DataLoader] → [PctChange>0.1] → [RSI<30] → [Rank] → [TopK=3] → [OrderEntry] → [StopLoss=0.05]
    ↓
自动执行 → 回测 → 展示结果
```

**架构设计**:

1. **新增 `internal/ai/strategy_agent.go`** — 策略生成 Agent
   - Prompt 注入所有可用节点类型（名称 + 参数 schema + 端口定义）
   - LLM 输出结构化 JSON（Workflow 定义格式）
   - 验证生成的 Workflow 是否合法（DAG 无环、端口类型匹配）
   - 不合法 → 反馈错误给 LLM → 重新生成（最多 3 轮）

2. **新增 `internal/ai/strategy_iteration.go`** — 策略迭代 Agent
   - 读取回测结果（Sharpe/MaxDD/WinRate）
   - 分析失败原因: "止损太紧" / "持仓时间太长" / "选股池太小"
   - 自动调整参数 → 重新回测 → 比较结果
   - 最多迭代 5 轮，保留最佳结果

3. **前端新增 `AIStrategyPanel.vue`** — AI 策略面板
   - 输入框: 自然语言描述策略意图
   - 预览区: 生成的 Workflow DAG 可视化预览（复用 vue-flow）
   - 操作区: 回测 / 修改 / 保存 / 导出 Python 代码

### Part B: 可转债分析模块

**市场背景**: A 股可转债（Convertible Bond）市场特色：
- T+0 交易（与股票 T+1 不同！）
- 无涨跌停限制（理论上有熔断机制）
- 打新（CB IPO）= 几乎无风险套利
- 双低策略（低价格 + 低转股溢价率）是最受欢迎的转债策略
- 回售/强赎/下修条款复杂

**新增模块**:

1. **数据适配器**: `internal/market/adapters/eastmoney_cb.go`
   - 可转债列表 + 实时行情
   - 转股溢价率、纯债价值、到期收益率
   - 回售/强赎/下修触发状态

2. **分析模型**: `internal/research/cb_analyzer.go`
   - CB 估值: 纯债价值 + 期权价值（BS 模型）
   - 双低排名: `price + premium_rate * 100` 排序
   - 下修概率估算: 基于财务压力 + 历史行为

3. **前端面板**: `ConvertibleBondPanel.vue`
   - 双低排名表
   - 单券分析: 条款时间线 + 估值分解
   - 打新日历

4. **Workflow 节点**: `cb_scanner` 节点
   - 参数: 价格上限、溢价率上限、剩余年限范围
   - 输出: 符合条件的转债列表

### Part C: 期权定价升级

**当前状态**: `python/src/analytics/options.py` 有 BS 模型基础实现。

**升级内容**:

1. **Binomial Tree (CRR) 定价** — 支持美式期权（可提前行权）
   ```python
   def binomial_price(S, K, T, r, sigma, steps=100, option_type='call', style='american'):
       # Cox-Ross-Rubinstein tree
   ```

2. **Greeks 计算** — Delta/Gamma/Theta/Vega/Rho
   ```python
   def compute_greeks(S, K, T, r, sigma, option_type='call'):
       # Black-Scholes Greeks
   ```

3. **隐含波动率曲面** — 从市场期权价格反推 IV
   ```python
   def implied_vol(market_price, S, K, T, r, option_type='call'):
       # Newton-Raphson root finding
   ```

4. **前端新增**:
   - `VolatilitySurfacePanel.vue` — 3D IV 曲面图
   - `OptionChainPanel.vue` — 期权链 + Greeks 表（已有 `us-option-chain` 面板，增强即可）

### Part D: 富途 OpenD 实盘接入

**当前状态**: `internal/trading/brokers/futu.go` 全部返回 `"not yet implemented"`。

**实现方案**:

1. **富途 OpenD 协议**: 本地 TCP 连接（默认 `localhost:11111`），protobuf 协议
2. **引入 `github.com/futuopen/ftapi4go`** 或自实现核心协议
3. **实现 `Broker` 接口**:
   - `Connect` → 连接 OpenD → 解锁交易密码
   - `SubmitOrder` → 港股/美股/A 股下单
   - `GetPositions` → 持仓查询
   - `GetAccount` → 账户信息（资金、购买力）

4. **注意事项**:
   - 富途 OpenD 需要本地运行 FutuOpenD 客户端
   - A 股交易需要连接华泰等券商（富途是通道）
   - 港股/美股直接走富途通道

## Acceptance Criteria

### Part A: AI Agent
- [ ] 自然语言输入 "找出RSI<30的A股" → 自动生成包含 DataLoader + RSI + Filter 的合法 Workflow JSON
- [ ] 生成的 Workflow 可正常执行并输出结果
- [ ] 策略迭代 Agent 可根据回测结果自动调整参数并在 3 轮内找到更优 Sharpe
- [ ] AIStrategyPanel 交互流畅，预览 DAG 可视化

### Part B: 可转债
- [ ] 可转债列表加载 + 实时行情正常
- [ ] 双低排名正确（price + premium_rate * 100）
- [ ] CB 估值 = 纯债价值 + 期权价值，与市场价偏差 < 20%
- [ ] `cb_scanner` workflow 节点可用

### Part C: 期权
- [ ] BS、Binomial Tree 两种定价模型可选
- [ ] Delta/Gamma/Theta/Vega/Rho 全部可计算
- [ ] IV 曲面图可交互旋转/缩放
- [ ] 期权链面板展示 Greeks

### Part D: 富途
- [ ] FutuAdapter 实现完整 `Broker` 接口
- [ ] 港股/美股下单→成交→持仓更新全流程通过
- [ ] 集成测试标记 `//go:build integration`

## Risks / Trade-offs

- **AI Agent 幻觉风险**: LLM 可能生成不合法的 Workflow（循环依赖、不存在的节点类型、错误的端口映射）。通过多层验证（类型检查 + DAG 合法性 + 端口兼容性）兜底，不合法的自动要求 LLM 重生成。
- **可转债数据源**: EastMoney 的 CB 数据可用但字段可能不完整。备选: 集思录（付费 API）。
- **富途 OpenD 依赖**: 需要本地运行 FutuOpenD，增加了部署复杂度。适合专业用户，不适合一键启动场景。
- **期权数据成本**: 美股实时期权链数据需要付费（OPRA 数据费），免费数据源有 15 分钟延迟。建议先用延迟数据实现功能，实时数据标记为 Premium 功能。

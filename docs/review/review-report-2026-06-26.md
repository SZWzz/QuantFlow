# QuantFlow Terminal 综合评审报告

**评审日期**: 2026-06-26  
**项目版本**: 2026.6.26  
**评审性质**: 全量代码 + 架构 + 量化金融 + 前端综合评审

---

## 目录

1. [执行摘要](#1-执行摘要)
2. [Go 后端评审](#2-go-后端评审)
3. [Vue 前端评审](#3-vue-前端评审)
4. [Python Sidecar 评审](#4-python-sidecar-评审)
5. [量化金融专项评审](#5-量化金融专项评审)
6. [安全审计](#6-安全审计)
7. [测试覆盖评审](#7-测试覆盖评审)
8. [综合评分与优先级](#8-综合评分与优先级)

---

## 1. 执行摘要

QuantFlow Terminal 是一个由单人开发的、scope 宏大的双模式量化金融终端。项目在 101+ commits 内跨越了从"概念验证"到"功能丰富但尚未打磨"的阶段。

### 核心量化指标

| 指标 | 数值 | 评价 |
|------|------|------|
| Go internal packages | 14 个 | 合理分层 |
| 工作流节点 | ~90 个文件 | 极其丰富，部分仅供接口占位 |
| 前端面板 | 52 个 Vue 组件 | scope 明显超纲 |
| 数据适配器 | 30+ 个 | 广度惊人，深度参差 |
| SQLite 迁移 | 10+ 个版本 | 良好设计 |
| 测试文件 | Go ~20 + Vue ~15 + Python ~5 | 覆盖不足，深度不够 |
| 文档 | 54+ specs + CLAUDE.md | 业界罕见的规范程度 |
| 开发周期 | ~72 小时 | 令人惊叹但也带来风险 |

### 现状判断

项目处于 **功能堆叠中期 → 工程收敛前夜**。最大的优势在于架构设计的远见（dual-mode、fallback 链、workflow DAG 引擎）和文档规范的严谨。最大的风险在于单人维护 52 个面板 + 90 个节点 + 30+ 适配器的长尾技术债。

---

## 2. Go 后端评审

### 2.1 架构设计

**评级: A-**

分层清晰：`app.go` (Wails 绑定) → `internal/{workflow,market,trading,...}` (领域逻辑) → `internal/storage` (SQLite 持久化)。依赖注入通过 `App` 结构体字段 + setter 函数 (`nodes.SetXxx()`) 实现，在单体桌面应用中合理。

**优点**:
- 接口设计干净：`Adapter`、`Broker`、`BaseNode` 都有明确的契约
- 容灾链机制 (`FetchQuoteWithFallback`) 应对中国数据源不稳定现状，非常有实用价值
- SQLite 迁移框架设计良好：embed + 版本号 + 事务包裹

**问题**:

1. **`nodes.SetXxx()` 全局 setter 模式（中风险）**  
   `nodes` 包通过全局变量注入依赖（`research_deps.go`），破坏了编译期依赖可见性。例如 `SetAgentDependencies`、`SetTradingOMS`、`SetNewsAdapter` 等赋值给包级变量。在 `app.go:ServiceStartup()` 中调用顺序成为隐式契约，误调用顺序会导致 nil pointer dereference。  
   *建议*: 改为 `NodeContext` 结构体作为执行上下文传给 `Execute()`。

2. **`app.go` 臃肿（1343 行）（高风险）**  
   作为 Wails 绑定层，混合了初始化编排、所有 IPC 方法、市场数据逻辑、研究逻辑。`ServiceStartup()` 函数长达 ~220 行，是所有依赖的硬编码编织。  
   *建议*: 拆分为 `app_init.go`（启动编排）、`app_trading.go`、`app_research.go`、`app_market.go`。

3. **`commodity.go` 死代码（低风险）**  
   `parseSinaCommodities` 末尾有 `_ = json.NewDecoder` 和 `var _ = fmt.Sprintf`，仅为了满足 import 而不编译报错。这表明代码拆分后遗留了残留 import。

4. **`MarketDataHub` 空转（中风险）**  
   `app.go:171` 创建 `NewHub()` 但返回值直接赋给 `_`，且 `hub.go` 中的 `Publish()` 从未被调用。整个 pub/sub 基础设施已搭建但未接入任何数据源。桌面应用中这种"预留"模式可以接受，但文档需注明。

### 2.2 错误处理

**评级: B**

- 良好的 wrapped error 风格 (`fmt.Errorf("...: %w", err)`)
- `ServiceStartup()` 中部分错误仅 `slog.Warn` 而不中止启动（如 sidecar 失败、migration 失败），用户友好但有时隐藏了真正的故障

**问题**:

1. **`GetStockResearch` 错误静默吞咽（中风险）**  
   `financials`、`peers`、`estimates`、`insider` 等子查询错误被 `_, _ = svc.GetXxx()` 静默吞噬。用户看到部分空白的页面却不知道数据加载失败。  
   *建议*: 返回 partial error 结构体，或在结果中嵌入 error 字段让前端可以展示。

2. **`ServiceStartup` 单点故障（高风险）**  
   整个启动流程是线性顺序的，任何一个步骤失败（哪怕是可选依赖）都会让 `return err` 阻止后续初始化。sidecar 启动失败后，bridge 为 nil，但后续 `capabilities.RegisterFactorCapabilities(a.capRegistry, a.bridge)` 传参为 nil 不会报错——因为它接受 nil bridge 然后默默跳过——这种隐式契约靠的是读代码才能理解。

### 2.3 并发安全

**评级: A-**

`sync.RWMutex` 在 `OMS`、`AdapterRegistry`、`NodeRegistry`、`topicBroker` 中正确使用。`topicBroker.unsubscribe()` 故意不关闭 channel（文档解释了原因），属于有意识的设计取舍。

**问题**:

1. **`OMS.FillOrder` 持有锁期间调用回调（中风险）**  
   `o.notifyTrade(trade)` 和 `o.notifyOrder(order)` 在 `o.mu.Lock()` 持有的情况下同步调用。文档注释 `"Callbacks are called synchronously under the OMS lock; keep them fast"`。如果某个回调阻塞（哪怕是意外的网络调用），整个 OMS 停摆。  
   *建议*: 在锁外调用回调（收集回调列表，释放锁，再逐个调用）。

---

## 3. Vue 前端评审

### 3.1 架构设计

**评级: B+**

Pinia stores 分层合理：`data.ts`（行情）、`session.ts`（UI 状态）、`workflow.ts`（画布）、`terminal.ts`（面板布局）、`settings.ts`（配置）、`portfolio.ts`（持仓）、`notify.ts`（通知）。

**优点**:
- TypeScript 全覆盖，类型定义清晰
- Panel registry 使用 `defineAsyncComponent` 实现懒加载
- 主题系统通过 CSS 变量 + body class 实现，组件只需引用 `var(--color-xxx)`
- `workflow.ts` 中的 undo/redo 历史栈实现得当

**问题**:

1. **`(window as any).go.main.App.Xxx()` 直接调用模式（高风险）**  
   所有 IPC 调用都通过 `(window as any).go.main.App` 进行，没有封装层。这意味着：
   - 没有类型安全：方法名拼写错误在运行时才暴露
   - 没有 mock 层：测试时必须模拟全局 `window.go` 对象
   - 没有错误统一处理：每个组件各自 try/catch  
   *建议*: 创建 `src/lib/wails.ts` 封装所有 IPC 调用，导出类型安全的 async 函数。

2. **Pinia store 空值处理不一致（中风险）**  
   `marketOverview` 初始化为 `null`，`quotes` 初始化为 `Map()`，两者混用判空逻辑 (`if (!app)` vs `try/catch`)。  
   *建议*: 统一定义 store 状态的初始值（空对象而非 null），减少前端判空分支。

3. **`WatchlistPanel.vue` 数据流不完整（中风险）**  
   首次加载 + 添加自选时调用 `refreshQuote`，但缺少自动刷新定时器和 WebSocket 推送。用户需要手动刷新或增加自选才能看到更新。

4. **`CandlestickPanel.vue` 硬编码 API 路径（中风险）**  
   `watch(symbol, ...)` 和 `loadOHLCV` 通过 `(window as any).go.main.App.FetchOHLCV(...)` 调用，没有使用 wailsjs 生成的类型绑定。随着 Wails v3 升级，这些调用路径可能变化。

5. **50+ 面板维护负担（高风险）**  
   `registry.ts` 注册了 52 个面板，每个面板是一个独立的 Vue SFC。大部分面板与后端"单查询-单渲染"一一对应。这种粒度在单人项目中难以长期维护，尤其是当后端 API 签名变更时需要逐个面板排查。  
   *建议*: 将数据结构相似的面板（如 FinancialsPanel / PeerComparisonPanel / AnalystEstimatesPanel）合并为"ResearchPanel"，通过 prop 切换显示模式。

### 3.2 i18n

**评级: B+**

i18n 架构完整，zh.ts ~350 key，50 个面板全部从硬编码字符串迁移到 `$t()`。

**问题**:
- 部分新面板（如 `SatellitePanel`、`GeopoliticsPanel`）可能还未完成国际化的全覆盖检查
- `session.ts` 中的 UI 标签（welcome tab label）未走 i18n

### 3.3 路由与状态同步

**评级: A**

`App.vue` 中的 `watch` 双向同步 `session.ui.mode` 与 `route.path` 是一个非常干净的模式，确保 URL 与状态一致。这个细节处理得当。

---

## 4. Python Sidecar 评审

### 4.1 架构设计

**评级: B**

gRPC 作为 Go/Python 边界的选择正确。proto 定义清晰，server.py 使用 `grpc.aio` 异步风格。

**问题**:

1. **单点故障影响整个应用（高风险）**  
   Python sidecar 是可选依赖（Go 侧优雅降级），但 `src/server.py` 本身没有进程管理器。如果 sidecar crash，Go 侧无法自动重启（`app.go` 中 sidecar 启动后不再监控其健康状态）。  
   *建议*: 在 Go 侧实现一个 watchdog goroutine，定期 Ping health endpoint，crash 时自动重启。

2. **并发限制（中风险）**  
   `ThreadPoolExecutor(max_workers=10)` 在 ML/因子计算等高耗时场景下可能导致所有 worker 被占满，后续请求排队。  
   *建议*: 对长时间运行的任务使用独立进程或 asyncio 子任务。

3. **`resource.getrusage` macOS 特定（低风险）**  
   `health.py` 中的内存统计使用 `resource.RUSAGE_SELF`，在 Linux 上行为不同（`ru_maxrss` 在 Linux 是 KB，macOS 是 bytes）。

---

## 5. 量化金融专项评审

### 5.1 关键发现

**评级: C+**

项目在数据源广度（30+ 适配器）和回测基础设施上投入了大量精力，但金融核心逻辑存在几个需要关注的问题。

**已修复**: 上次评审发现的回测前瞻偏差（`strategy.go`/`backtest.go` 在同一根 bar 的 Close 上生成和执行信号）——需要登录证实现有代码已更正。

**当前问题**:

1. **`Position.PnL` 计算公式不完整（高风险）**  
   `oms.go:248` 中 `UnrealizedPnL = (marketPrice - avgPrice) * quantity` 仅在多头有效。公式未考虑交易成本（佣金、印花税、滑点）。对于 A 股，印花税单边 0.05%、佣金 ~0.025% 对高频/短线策略有实质影响。  
   *建议*: 在 `FillOrder` 中记录交易成本字段，P&L 计算中扣除。

2. **现金账本缺失（高风险）**  
   `GetPortfolioSummary` 中现金是从交易历史推导的累加值 `cash -= t.Price * t.Quantity`，没有独立的现金账本。这意味着：
   - 现金可以为负（没有保证金检查）
   - 多账户或多币种场景无法支持
   - 从外部转入/转出资金无法记录  
   *建议*: 实现 `CashLedger` 结构体，记录每笔资金变动。

3. **`RiskPipeline.CheckDrawdown` 修改 receiver（中风险）**  
   `r.config.PeakEquity` 在 `CheckDrawdown` 中被修改（`r.config.PeakEquity = currentEquity`），而该方法签名为值接收者 `(r *RiskPipeline)`。虽然 Go 允许，但违背了"Checker 不改变状态"的直觉。并且当多个策略共享同一个 `RiskPipeline` 实例时，PeakEquity 会被不正确地共享。  
   *建议*: 将 PeakEquity 改为外部传入或使用互斥锁保护。

4. **买卖价差/滑点模型缺失（中风险）**  
   所有回测和模拟执行都假设以指定价格成交。没有滑点模型意味着回测结果会系统性偏乐观。  
   *建议*: 在 `PaperEngine` 中加入可配置的滑点模型（固定百分比或基于成交量的动态滑点）。

5. **`OHLCVBar` 类型重复定义（低风险）**  
   `internal/market/types.go` 和 `internal/trading/types.go` 各自定义了完全相同的 `OHLCVBar` 结构体。虽然在不同包中不冲突，但语义上表明市场数据和交易引擎之间的模型未统一。

### 5.2 中国市场规则合规

**评级: C**

- **T+1 未实现**: A 股 T+1 交收规则未在 `PaperEngine` 或 `OMS` 中实现。当日买入的股票可以立即卖出。锁仓机制缺失。
- **涨跌停板未实现**: ±10% / ±20%（科创/创业）的日内限价未在订单校验中实现。
- **最小交易单位**: 100 股（主板）/ 200 股（科创板）的整数倍检查缺失。
- *建议*: 这些是 A 股交易的基础约束，应在 `OrderMatcher` 或 `RiskPipeline` 中加入。没有涨跌停限制的回测对 A 股不可靠。

### 5.3 数据源广度

**评级: A**

30+ 个数据适配器覆盖了 A 股（EastMoney、Sina、Tencent、mootdx、TuShare、AKShare、Baidu）、美股（Yahoo、Finnhub、Polygon、Alpaca）、港股（Yahoo、Tencent）、加密（Binance、OKX、GateIO、CoinGecko）以及另类数据（GDELT、Polymarket、FRED、NASA POWER/FIRMS、Congress Trades）。

容灾链机制 (`FallbackChains`) 是在中国数据源可靠性环境下非常有价值的设计。

**问题**:
- 多数适配器只有 "fetch" 能力，没有 "subscribe"（WebSocket/stream）。实时行情推送缺失
- EastMoney 系列适配器依赖 HTTP/1.1 fallback 来解决 CDN 兼容问题（已修复），但部分适配器可能在生产环境中周期性超时
- mootdx 依赖 Python sidecar + TDX TCP 协议，延迟 ~4s，在行情节点中实用性有限

---

## 6. 安全审计

### 6.1 关键发现

**评级: A-**

| 风险 | 状态 | 备注 |
|------|------|------|
| API Key 暴露到前端 | ✅ 已修复 | `GetConfig()` 不再返回 `api_keys` |
| gRPC 绑定到 localhost | ✅ 已修复 | 从 `[::]` 改为 `localhost` |
| SQL 注入 | ✅ 安全 | 使用参数化查询 (`?`) |
| JWT/认证 | ✅ N/A | 桌面应用，无需 |
| config.yaml 权限 | ⚠️ 低风险 | 0644 权限，同机器其他用户可读 API keys |

**问题**:

1. **config.yaml 权限过于宽松（低风险）**  
   `config.go:Save()` 使用 `0644` 权限保存包含 `api_keys` 的配置文件。在多用户 Mac/Linux 系统上，其他用户可读取 API keys。  
   *建议*: 使用 `0600`。

2. **Sidecar 无认证（中风险）**  
   Python gRPC 绑定到 localhost，但无任何认证机制。如果攻击者能在 localhost 上运行进程（如恶意 JS 扩展），可以访问 sidecar。这在一个桌面安全边界内是可接受的，但文档应注明。

3. **日志可能泄漏敏感信息（低风险）**  
   `slog.Warn` 和 `slog.Info` 在多处记录适配器错误详情，如果 API key 错误被包含在错误消息中，可能泄露到日志。`config.go` 中使用 `os.Getenv("FRED_API_KEY")`，环境变量也可能被 `/proc` 泄漏。

---

## 7. 测试覆盖评审

### 7.1 Go 测试

**评级: C**

| 包 | 测试文件 | 覆盖深度 | 评审 |
|-----|---------|---------|------|
| app | 1 | 浅 | 仅验证注册计数 + nil registry guard |
| workflow | 5 | 中 | DAG、Engine、Node、Registry 有覆盖 |
| storage | 4 | 中 | DB、migrations、workflow_repo |
| market | 5 | 中 | Hub、Registry、Retry、SymbolSearch |
| trading | 3 | 浅 | OMS 测试缺失，Broker、Engine 测试 |
| portfolio | 2 | 浅 | Service、Analytics |
| python | 3 | 浅 | Bridge、LLM、ML clients |
| ai | 1 | 浅 | 基本 agent 测试 |
| research | 1 | 浅 | SentimentEngine |

**关键缺失**:
- `OMS` 核心逻辑（`PlaceOrder`、`FillOrder`、`CancelOrder`）没有独立测试
- `AdapterRegistry` 的 fallback 链未做集成测试
- `MarketDataHub` 的 pub/sub 并发安全没有压力测试
- `RiskPipeline` 的 `CheckDrawdown` 并发修改 race 没有检测

### 7.2 前端测试

**评级: C+**

18 个 store 测试文件，部分覆盖 session、terminal、workflow 等核心 store。面板组件缺少渲染测试和交互测试。

### 7.3 Python 测试

**评级: D**

`tests/` 目录存在但内容极少。`server.py` 的 gRPC handler 单元测试缺失。

---

## 8. 综合评分与优先级

### 评分汇总

| 维度 | 分数 | 关键短板 |
|------|------|---------|
| Go 架构设计 | A- | `app.go` 臃肿，全局 setter |
| Go 工程质量 | B+ | 错误处理可改进，锁内回调有隐患 |
| Vue 架构设计 | B+ | IPC 无封装层，面板 scope 超纲 |
| Vue 工程质量 | B | 类型安全不足，mock 测试不易 |
| Python 工程质量 | B | 进程管理缺失，并发限制 |
| 量化金融正确性 | C+ | 交易成本缺失，T+1/涨跌停未实现 |
| 测试覆盖 | C | 核心 OMS 无测试，回测集成测试缺失 |
| 安全 | A- | Sidecar 无认证，配置文件权限可收紧 |
| 文档 | A | 业界罕见的规范程度 |
| 可维护性 | C+ | 单人 50+ 面板 + 90 节点的长尾维护 |

### 优先级建议

**P0 — 立即处理**:
1. 量化核心: 实现 A 股 T+1 锁仓 + 涨跌停校验（否则回测不可信）
2. 量化核心: 引入交易成本模型（印花税 + 佣金）
3. 量化核心: 实现独立 `CashLedger` 而非从交易记录反推
4. 工程: 封装 `lib/wails.ts` 使得前端 IPC 调用有类型安全

**P1 — 本周内**:
1. 测试: 补 `OMS` 核心 test suite（Place/Fill/Cancel)
2. 工程: 拆分 `app.go` 为多文件
3. 工程: `nodes.SetXxx()` → `NodeContext` 结构体传递
4. 测试: AdapterRegistry fallback 集成测试
5. 量化: `Position.PnL` 公式补充交易成本

**P2 — 本月内**:
1. 量化: `RiskPipeline.CheckDrawdown` 并发修复 + 市值状态分离
2. 工程: `nodes/research_deps.go` 全局变量重构
3. 前端: 合并相似面板减少维护量（如合并 3 个 Research 面板）
4. Python: sidecar watchdog goroutine
5. 测试: 前端面板组件测试框架搭建

**P3 — 长远规划**:
1. Python: sidecar worker pool 隔离 ML 任务
2. 前端: 从 `(window as any)` 迁移到 Wails v3 官方 TypeScript 绑定生成
3. 量化: 滑点模型 + 部分成交模型
4. Python: Linux `ru_maxrss` 单位适配
5. 工程: `OHLCVBar` 类型归一化

---

*报告生成: Claude Code · 2026-06-26*

# QuantFlow Terminal — 新一代量化金融终端项目方案

> **目标**: 融合 FinceptTerminal 全功能 + AstockPursue 工作流引擎，全新架构，全新实现
> **核心设计理念**: Workflow-First Architecture — 一切功能皆为工作流节点

---

## 目录

1. [项目定位与技术选型](#1-项目定位与技术选型)
2. [架构全景](#2-架构全景)
3. [工作流引擎——核心骨骼](#3-工作流引擎核心骨骼)
4. [功能模块融合映射](#4-功能模块融合映射)
5. [数据架构](#5-数据架构)
6. [AI 智能体系统](#6-ai-智能体系统)
7. [前端设计](#7-前端设计)
8. [项目目录结构](#8-项目目录结构)
9. [实现路线图](#9-实现路线图)
10. [关键设计决策](#10-关键设计决策)
11. [风险与缓解](#11-风险与缓解)

---

## 1. 项目定位与技术选型

### 1.1 为什么需要新项目

| 维度 | FinceptTerminal | AstockPursue | QuantFlow 目标 |
|------|----------------|-------------|---------------|
| 技术栈 | C++20/Qt6 + Python | Python/React | **Go + Vue 3 + Wails** |
| UI 架构 | 56 个独立屏幕，Qt Widget | 独立页面，@xyflow 画布 | **双模式: 彭博终端面板 + 工作流画布** |
| 工作流 | 节点编辑器（12 类，附属工具） | 可视化工作流（58 节点，核心引擎） | **200+ 节点，16 类别，一切皆节点** |
| 交易引擎 | C++ 原生 OrderMatcher | Python TradingEngine | **Go 协程驱动，统一 Bar-by-Bar 管线** |
| AI 智能体 | MCP + finagent_core | 89 Skill + ReAct Agent | **Go 原生 Agent 循环 + Python gRPC LLM** |
| 目标市场 | 印度18+全球3+加密2 | 中国A股为主 | **A 股 + 港股 + 美股 + 加密** |
| 部署 | Qt 桌面应用 + Python 子进程 | Docker 容器化 Web | **Wails 桌面 + 可选 Docker 后端** |
| 性能模型 | 主线程单例 + QtConcurrent | asyncio + ProcessPool | **Go goroutine + channel 零拷贝 + SQLite WAL** |

### 1.2 技术栈选型详解

#### 后端核心: Go 1.22+

```
选型理由:
├── 并发模型: goroutine + channel 天然适合工作流 DAG 并行执行
├── 编译产物: 单二进制，无运行时依赖 → 部署极简
├── 性能: 接近 C++ 的吞吐，远低于 C++ 的维护成本
├── 生态: 丰富的金融库（go-talib, go-quant, marketstore）
└── 与 AstockPursue 的 Python 生态互补:
    ├── Go 负责: 交易引擎、工作流引擎、数据管道、API 网关
    └── Python 负责: ML 模型、因子计算（pandas/numpy）、LLM 推理
```

#### Python Sidecar: Python 3.12+ (gRPC 桥接)

```
选型理由:
├── 保留量化生态: pandas, numpy, scikit-learn, statsmodels, qlib
├── 保留 LLM 生态: langchain, openai, anthropic SDK
├── gRPC 协议: 强类型、高性能、流式传输
└── 独立部署: 可独立扩缩容，Go 进程崩溃不影响 Python ML 任务
```

#### 前端: SolidJS + TypeScript (Tauri 桌面壳)

```
选型理由:
├── SolidJS: 无虚拟 DOM，编译时响应式，性能接近原生 JS
│   └── 比 React 快 3-5x，适合金融数据高频更新场景
├── @xyflow/solid: 工作流画布（React 版的 Solid 移植）
├── ECharts: 金融图表（保持与 AstockPursue 一致）
├── Monaco Editor: 代码编辑
├── Tauri: 轻量桌面壳（~10MB vs Electron ~150MB）
│   ├── Rust 内核提供系统级 API
│   └── Go 后端通过 HTTP/WebSocket 与前端通信
└── Tailwind CSS: 实用优先的样式系统
```

#### 存储层 — 一个 SQLite 就够了

```
SQLite (WAL 模式) → 唯一数据库。承载所有持久化需求。
    ├── 用户配置、偏好设置
    ├── 工作流定义 + 版本历史 (JSON 字段)
    ├── 交易记录 / 订单 / 持仓
    ├── K 线数据缓存 (按 symbol+interval 分区，TTL 自动清理)
    ├── 因子计算结果缓存
    └── 审计日志

为什么不需要 PostgreSQL/Redis/DuckDB？
    ├── 这是桌面应用，不是 SaaS。SQLite 单文件、零配置、嵌入进程内。
    ├── WAL 模式下读写并发足够（FinceptTerminal 证明了这一点）。
    ├── Go channel 替代 Redis PubSub（进程内零拷贝）。
    ├── 内存 map + TTL 替代 Redis 缓存（无需网络往返）。
    └── 只有极端场景（10 年分钟级 K 线、TB 级因子回测）才考虑
        可选启用 DuckDB 作为分析加速器——但这是 opt-in，不是默认。
```

#### 桌面壳选择: Wails v3 替代 Tauri

```
之前选 Tauri (Rust 壳 + Go 后端)，但这引入了不必要的复杂度:
    ├── 需要维护 Rust 工具链（仅为了系统壳）
    ├── Go 和 Rust 两套编译流水线
    └── Tauri 的 IPC 序列化开销

Wails v3 是更合理的选择:
    ├── 壳层用 Go（与后端同一语言、同一进程）
    ├── 前端 ↔ 后端直接函数调用，无 HTTP 序列化开销
    ├── 单 Go 工具链编译整个桌面应用
    └── 原生菜单、对话框、系统托盘、文件关联
```

#### 前端框架重新审视 — Vue 3 + vue-flow

```
SolidJS + @xyflow/solid 的风险:
    ├── @xyflow/solid 社区远小于 React 版，成熟度存疑
    ├── SolidJS 生态（UI 组件库、devtools）不如 Vue/React 丰富
    └── 招聘市场上 SolidJS 开发者极少

Vue 3 + vue-flow 的理由:
    ├── vue-flow (@vue-flow/core) 是 xyflow 的官方 Vue 移植，成熟度仅次于 React 版
    ├── Vue 3 Composition API + TypeScript 支持与 React Hooks 同级别
    ├── 响应式系统原生适合金融数据流（computed/watch 自动依赖追踪）
    ├── 生态丰富: Naive UI/Element Plus 组件库、Pinia 状态管理、Vite 构建
    ├── 学习曲线低于 React（模板语法 vs JSX 心智模型）
    └── 与 Go 后端无任何绑定——纯前端，Wails WebView 加载
```

#### 技术栈对比总览

| 层 | FinceptTerminal | AstockPursue | QuantFlow |
|----|----------------|-------------|-----------|
| 后端语言 | C++20 | Python | **Go 1.22+** |
| ML/AI 语言 | Python (QProcess 子进程) | Python (同进程) | **Python (gRPC sidecar)** |
| 桌面壳 | 原生 Qt | 无（浏览器） | **Wails v3 (Go 原生)** |
| 前端框架 | Qt6 Widgets | React 19 | **Vue 3 + TypeScript** |
| 流程图 | Qt 自定义绘制 | @xyflow/react | **@vue-flow/core** |
| 图表 | Qt Charts | ECharts | **ECharts (vue-echarts)** |
| 数据库 | SQLite | PostgreSQL | **SQLite (WAL)** |
| 消息总线 | DataHub (C++ 单例) | asyncio Queue | **Go channel (类型安全)** |
| 状态管理 | 信号/槽 | Zustand | **Pinia** |
| 构建 | CMake | Vite + pip | **Go build + Vite** |
| 部署 | 单 Qt 二进制 | Docker Compose | **单 Go 二进制（Python sidecar 可选嵌入）** |

---

## 2. 架构全景

### 2.1 核心原则: 双模式 — 彭博终端 + 工作流画布

**FinceptTerminal 的核心价值**: 类彭博终端体验——输入代码即刻获取一切、密集信息面板自由停靠、实时流式数据、键盘驱动操作。

**AstockPursue 的核心价值**: 可视化工作流编排——拖拽节点构建量化研究管线、因子→策略→回测→归因一站式。

**QuantFlow 不做二选一——两者合一**:

```
┌──────────────────────────────────────────────────────────────────┐
│                      QuantFlow Desktop                            │
│                                                                   │
│  ┌───────────────────────────┐    ┌────────────────────────────┐  │
│  │    TERMINAL MODE (默认)    │    │    WORKFLOW MODE (Tab切换)  │  │
│  │    彭博式面板终端           │◄──►│    可视化工作流画布          │  │
│  │                           │    │                            │  │
│  │  ┌─────┐┌─────┐┌─────┐   │    │  [数据]→[因子]→[策略]→[回测] │  │
│  │  │AAPL ││Port ││News │   │    │     │                │       │  │
│  │  │Quote││Risk ││Feed │   │    │     └→[AI]──→[信号]→[下单]   │  │
│  │  └─────┘└─────┘└─────┘   │    │                            │  │
│  │  ┌──────────┐┌──────┐   │    │  节点=面板功能                  │  │
│  │  │K线图      ││深度数据│   │    │  面板有"添加到工作流"按钮      │  │
│  │  └──────────┘└──────┘   │    │  工作流结果可固定到终端面板       │  │
│  └───────────────────────────┘    └────────────────────────────┘  │
│                                                                   │
│  共享底层: 同一个 Go 后端 / 同一个 SQLite / 同一条数据总线          │
└──────────────────────────────────────────────────────────────────┘
```

**两种模式的交互关系**（不是二选一，是双向流动）:

```
Terminal Mode → Workflow Mode:
    任何面板右上角 [⊕] 按钮 → 将该面板的数据/参数生成为一个工作流节点
    例如: AAPL 股票研究面板 → 创建 StockResearchNode( symbol="AAPL" )

Workflow Mode → Terminal Mode:
    工作流执行结果 → [固定到终端] → 在 Terminal Mode 中创建一个实时监控面板
    例如: 回测结果 → 创建 EquityCurvePanel + MetricsPanel，实时更新
```

### 2.2 系统架构图

```
┌──────────────────────────────────────────────────────────────┐
│                    Frontend (Vue 3 + Wails)                    │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Workflow     │  │ Dashboard    │  │ AI Chat Panel        │ │
│  │ Canvas       │  │ Views        │  │ (流式 SSE)           │ │
│  │ vue-flow     │  │ ECharts/Monaco│  │ Markdown + CodeBlock │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘ │
│         │                 │                      │             │
│         └─────────────────┼──────────────────────┘             │
│                           │ Wails IPC (Go 函数直接调用)          │
└───────────────────────────┼──────────────────────────────────┘
                            │
┌───────────────────────────┼──────────────────────────────────┐
│              Go Backend (单二进制)                             │
│                           │                                    │
│  ┌────────────────────────┼────────────────────────────┐      │
│  │              Wails App (HTTP/WS/SSE + IPC)           │      │
│  └────────┬───────────┬───────────┬───────────────────┘      │
│           │            │            │                          │
│  ┌────────┴──┐  ┌─────┴─────┐  ┌──┴──────────────┐          │
│  │ Workflow   │  │ Trading   │  │ Market Data     │          │
│  │ Engine     │  │ Engine    │  │ Hub             │          │
│  │            │  │           │  │                 │          │
│  │ · Kahn 调度 │  │ · Bar-by- │  │ · Go channel    │          │
│  │ · goroutine│  │   Bar 管线 │  │   发布/订阅      │          │
│  │   并行执行  │  │ · 9 市场   │  │ · TTL 内存管理   │          │
│  │ · 断点调试  │  │ · 风控管线 │  │ · 请求合并       │          │
│  │ · LRU 缓存  │  │ · 模拟/实盘│  │ · 离线回退       │          │
│  └─────┬──────┘  └─────┬─────┘  └──────┬──────────┘          │
│        │                │                │                     │
│  ┌─────┴────────────────┴────────────────┴──────────┐         │
│  │              Service Layer (领域服务)               │         │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │         │
│  │  │ AI Agent │ │ Research │ │ Portfolio/Risk   │  │         │
│  │  │ Orchestr.│ │ Service  │ │ Service          │  │         │
│  │  └────┬─────┘ └────┬─────┘ └───────┬──────────┘  │         │
│  └───────┼─────────────┼───────────────┼─────────────┘         │
│          │              │                │                      │
│  ┌───────┴──────────────┴────────────────┴──────────┐         │
│  │            Data Layer (全部进程内)                  │         │
│  │  ┌──────────┐  ┌─────────────────────────────┐   │         │
│  │  │ SQLite   │  │ In-Memory (sync.Map + chan)  │   │         │
│  │  │ (WAL)    │  │ · 行情缓存 · 去重窗口        │   │         │
│  │  │ 单文件   │  │ · LRU 节点缓存 · TTL 惰性淘汰 │   │         │
│  │  └──────────┘  └─────────────────────────────┘   │         │
│  └──────────────────────────────────────────────────┘         │
│                                                                │
│  ┌──────────────────────────────────────────────────┐         │
│  │          gRPC Bridge → Python Sidecar             │         │
│  │  · Factor Service (pandas)                       │         │
│  │  · ML Service (PyTorch/qlib)                     │         │
│  │  · LLM Service (langchain/ReAct)                 │         │
│  └──────────────────────────────────────────────────┘         │
└────────────────────────────────────────────────────────────────┘
```

---

## 3. 工作流引擎——核心骨骼

### 3.1 设计哲学

**AstockPursue 的工作流引擎是核心差异优势**。QuantFlow 将其从「量化研究管道」扩展为「金融终端统一操作层」：

- **FinceptTerminal 的 56 个屏幕** → QuantFlow 的 **200+ 个工作流节点**
- **每个屏幕的操作** → QuantFlow 中可组合、可编排、可自动化的节点
- **用户的工作习惯** → 可保存、可分享、可定时执行的工作流模板

### 3.2 节点类型体系 (16 类, 200+ 节点)

**来自 AstockPursue 的 58 节点 + 来自 FinceptTerminal 的功能节点化**

```
类别                      节点数    来源                         说明
──────────────────────────────────────────────────────────────────────────
1. 数据加载 (Data)          ~25     AS+FT     股票池、OHLCV、基本面、财报、另类数据
2. Alpha 因子 (Alpha)       ~20     AS         AlphaZoo、因子挖掘、GP 进化
3. 技术指标 (Indicator)     ~15     AS+FT      60+ 技术指标、自定义指标
4. 信号生成 (Signal)        ~10     AS+FT      Tick/Batch 信号、多因子合并
5. 策略构建 (Strategy)      ~12     AS         SignalEngine、权重优化
6. 回测执行 (Backtest)      ~10     AS+FT      9 市场引擎、参数扫描
7. 归因分析 (Attribution)    ~8     AS         Brinson、因子、行业分解
8. 风控分析 (Risk)           ~8     AS+FT      VaR、CVaR、压力测试、因子衰减
9. 交易执行 (Trading)       ~20     FT         A股/港股/美股/加密 券商、订单管理、仓位管理
10. 市场数据 (Market)       ~18     FT         实时行情、K 线、搜索、深度数据
11. 投资组合 (Portfolio)    ~12     FT         组合分析、优化、蒙特卡洛
12. 研究分析 (Research)     ~15     FT         股票研究、财务分析、情绪分析
13. AI 智能体 (Agent)       ~10     AS+FT      AgentNode、ReAct 循环、LLM 调用
14. 通知输出 (Notify)        ~8     AS         Telegram/Discord/Email/Feishu
15. 控制流 (Control)         ~8     AS+FT      If/Switch/Loop/Merge/Wait/Schedule
16. 工具节点 (Utility)      ~15     FT+新      代码编辑、Excel、报表、文件、HTTP
──────────────────────────────────────────────────────────────────────────
合计                        ~214
```

### 3.3 核心节点详解 (关键新增/融合节点)

#### 3.3.1 数据节点 — 融合 FinceptTerminal 的 100+ 数据连接器

```go
// 节点定义示例 (Go 结构体)
type StockUniverseNode struct {
    BaseNode
    // 从 AstockPursue 继承: 股票池
    UniverseType string   // "index" | "custom" | "screener_result" | "watchlist"
    IndexCode    string   // "CSI300" | "SP500" | ...
    CustomCodes  []string
}

type MarketDataNode struct {
    BaseNode
    // 从 FinceptTerminal 新增加: 100+ 连接器的统一接口
    Source       string   // "yahoo" | "fmp" | "polygon" | "futu" | "eastmoney" | ...
    DataType     string   // "quote" | "history" | "fundamentals" | "news" | ...
    Symbols      []string // 支持动态输入 from StockUniverse
    Interval     string   // "1m" | "5m" | "1h" | "1d" | ...
    Range        string   // "1mo" | "6mo" | "1y" | "5y" | "max"
}

type AlternativeDataNode struct {
    BaseNode
    // 从 FinceptTerminal 的另类数据
    DataCategory string  // "maritime" | "satellite" | "geopolitics" | "gov" | "prediction" | "congress"
    SubType      string  // 子类型: "vessel_track" | "ndvi" | "conflict_event" | ...
    Filters      map[string]any
}

type DataNormalizationNode struct {
    BaseNode
    // 从 FinceptTerminal 的数据归一化
    Schema       string  // "OHLCV" | "QUOTE" | "TICK" | "ORDER" | ...
    FieldMapping []FieldMap
    Validators   []string
}
```

#### 3.3.2 交易节点 — 多市场券商体系（A 股/港股/美股/加密）

```go
type BrokerNode struct {
    BaseNode
    // 多市场券商统一接口 (聚焦 A 股/港股/美股/加密，去掉印度市场)
    BrokerID     string  // "futu" | "ibkr" | "alpaca" | "binance" | "okx" | "longport" | ...
    Mode         string  // "paper" | "live"
    AccountUUID  string
}

type OrderNode struct {
    BaseNode
    // 融合 AstockPursue 信号 + FinceptTerminal 下单
    OrderType    string  // "market" | "limit" | "stop" | "stop_limit" | "bracket"
    ProductType  string  // "MIS" | "CNC" | "NRML" | "MTF"
    Quantity     int
    Price        float64 // 0 = market
    TriggerPrice float64
}

type SmartOrderNode struct {
    BaseNode
    // 量化仓位下单 (FinceptTerminal SmartOrderEngine)
    TargetWeight float64
    PositionSizing string // "equal" | "kelly" | "risk_parity" | "custom"
    MaxSlippageBps float64
}
```

#### 3.3.3 AI 智能体节点 — 融合 MCP + Skills

```go
type AgentNode struct {
    BaseNode
    // 从 AstockPursue 的 AgentNode 升级
    AgentProfile string  // "analysis_agent" | "trading_agent" | "buffett_agent" | ...
    LLMProvider  string  // "openai" | "anthropic" | "deepseek" | "ollama" | ...
    Skills       []string // 激活的 Skill 包
    MCPTools     []string // 激活的 MCP 工具
    Prompt       string  // 系统提示词
    MaxSteps     int     // ReAct 循环最大步数
}

type MCPProviderNode struct {
    BaseNode
    // 从 FinceptTerminal 的 MCP 系统
    MCPTools     []string  // 81+ 个工具
    AuthLevel    string    // "none" | "authenticated" | "verified" | "subscribed"
    TimeoutMs    int
}
```

### 3.4 工作流执行引擎

**从 AstockPursue 的 asyncio 引擎迁移到 Go goroutine 引擎**:

```go
// Go 实现的并发 DAG 执行引擎
type WorkflowEngine struct {
    globalSem     chan struct{}     // 全局并发控制
    profileSems   map[string]chan struct{} // 按资源类型限流
    nodeResults   sync.Map          // nodeID → results (线程安全)
    nodeStatus    sync.Map          // nodeID → NodeStatus
    edgeMap       map[EdgeKey]PortConnection
    
    // 高级功能
    debugMode     bool              // 断点调试
    nodeCache     *lru.Cache        // 节点级缓存
    continueOnErr bool              // 错误继续
    cancelCtx     context.Context   // 取消传播
    
    // Python gRPC 调用
    pythonClient  pb.FactorServiceClient
    pythonClient  pb.MLServiceClient
    pythonClient  pb.LLMServiceClient
}

func (e *WorkflowEngine) Execute(ctx context.Context, dag *WorkflowDAG) (*RunResult, error) {
    // 1. 拓扑排序 (Kahn 算法)
    layers := topologicalSort(dag)
    
    // 2. 逐层并行执行
    for _, layer := range layers {
        var wg sync.WaitGroup
        for _, nodeID := range layer {
            wg.Add(1)
            go func(nid string) {
                defer wg.Done()
                e.executeNode(ctx, dag, nid)
            }(nodeID)
        }
        wg.Wait()
        
        // 3. 检查取消信号
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
    }
    
    // 4. 聚合结果
    return e.aggregateResults(dag)
}

func (e *WorkflowEngine) executeNode(ctx context.Context, dag *WorkflowDAG, nodeID string) {
    // 获取并发槽位
    profile := dag.Nodes[nodeID].ResourceProfile
    sem := e.getProfileSem(profile)
    sem <- struct{}{}
    defer func() { <-sem }()
    
    // 检查缓存
    if cacheKey := e.computeCacheKey(dag, nodeID); cacheKey != "" {
        if cached, ok := e.nodeCache.Get(cacheKey); ok {
            e.nodeStatus.Store(nodeID, NodeStatusCached)
            e.nodeResults.Store(nodeID, cached)
            return
        }
    }
    
    // 收集输入
    inputs := e.collectInputs(dag, nodeID)
    
    // 根据节点类型分派执行
    node := dag.Nodes[nodeID]
    switch {
    case node.IsPythonBound():
        result = e.executeViaPython(ctx, node, inputs)  // gRPC → Python
    case node.IsCPUBound():
        result = e.executeCPU(ctx, node, inputs)        // goroutine pool
    case node.IsIOBound():
        result = e.executeIO(ctx, node, inputs)          // async I/O
    default:
        result = e.executeInline(ctx, node, inputs)
    }
}
```

### 3.5 工作流模板系统

**从 AstockPursue 的 15 个预置模板扩展为 50+ 模板**，覆盖 FinceptTerminal 的所有典型工作流：

```
模板分类:
├── 量化研究 (10 模板)
│   ├── 单因子 IC 分析
│   ├── 多因子组合回测
│   ├── 因子挖掘 → 回测 → 归因
│   ├── 市场状态检测 → 策略选择
│   └── 策略进化 (GP + WalkForward)
│
├── 交易执行 (8 模板)
│   ├── 信号 → 风控 → 下单
│   ├── 篮子交易 (Basket Order)
│   ├── TWAP/VWAP 执行算法
│   └── Webhook 信号 → 自动交易
│
├── 投资组合管理 (6 模板)
│   ├── 组合再平衡
│   ├── Black-Litterman 优化
│   ├── 风险归因
│   └── 蒙特卡洛模拟
│
├── AI 驱动 (8 模板)
│   ├── LLM 因子挖掘
│   ├── AI 策略生成 → 回测验证
│   ├── 多智能体辩论 (Bull vs Bear)
│   └── 新闻情绪 → 交易信号
│
├── 另类数据 (6 模板)
│   ├── 卫星图像 → 供应链分析
│   ├── 海事 AIS → 商品供需
│   ├── 预测市场 → 事件交易
│   └── 地缘政治 → 风险对冲
│
├── 监控告警 (6 模板)
│   ├── 价格突破 → 通知
│   ├── 组合回撤 → 减仓
│   ├── 因子衰减 → 策略暂停
│   └── 定时报告生成
│
└── 定时任务 (6 模板)
    ├── 每日因子刷新
    ├── 每周组合再平衡
    ├── 月度绩效报告
    └── 数据健康检查
```

---

## 4. 功能模块融合映射

### 4.1 FinceptTerminal 功能 → QuantFlow 节点/模块映射

#### 4.1.1 市场数据模块

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| MarketDataService (批量报价) | Go Market Data Hub + gRPC Python | MarketQuoteNode, MarketHistoryNode |
| DataHub (发布/订阅) | Go Channel (进程内类型安全总线) | MarketDataHub (Go 包) |
| MarketSearchService | Go REST 客户端 | MarketSearchNode |
| 100+ 数据连接器 | Go adapter 接口 + Python 脚本 | DataSourceNode (通用) |
| 数据归一化 (7 Schema) | Go struct tag + JSONPath | DataNormalizeNode |
| 凭据管理 (AES-256-GCM) | Go crypto/aes + macOS Keychain | CredentialManager (Go 包) |
| PythonWorker 守护进程 | gRPC 长连接 Python sidecar | PythonBridge (Go 包) |

#### 4.1.2 交易执行模块

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| UnifiedTrading (订单中枢) | Go TradingHub (channel-based) | OrderNode, TradeHub |
| IBroker 接口 (多市场) | Go Broker 接口 (struct embed) | BrokerNode, 多市场 broker adapters |
| PaperTrading (模拟交易) | Go PaperEngine (goroutine) | PaperTradingNode |
| OrderMatcher (撮合引擎) | Go OrderMatcher (channel + mutex) | 内嵌于 PaperEngine |
| SmartOrderEngine | Go PositionSizer | SmartOrderNode |
| ActionCenter (审批) | Go ApprovalQueue (channel) | ApprovalNode |
| WebhookListener | Go net/http server | WebhookListenerNode |
| AlgoEngine (算法引擎) | 合并入 WorkflowEngine | StrategyNode → Backtest/Live |
| AccountManager | Go AccountService + SQLite | AccountManager (Go 包) |
| ExchangeService (加密货币) | Go CCXT 封装 + Python WS | CryptoExchangeNode |

#### 4.1.3 AI 智能体系统

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| finagent_core (Python) | Go AgentOrchestrator + Python LLM sidecar | AgentNode |
| MCP (81 工具) | Go MCP Provider + Tool Registry | MCPProviderNode |
| 37+ 智能体角色 | Agent Profile 系统 (YAML 配置) | AgentNode.profile |
| TerminalMcpBridge | Go HTTP MCP Bridge | mcpbridge (Go 包) |
| AgentService | Go AgentService | agentservice (Go 包) |
| AI Chat Screen | AI Chat Panel (SolidJS, SSE 流式) | ChatPanel 组件 |
| 89 Skills (AstockPursue) | Skill Registry (YAML + Markdown) | skills/ 目录 |

#### 4.1.4 研究分析模块

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| EquityResearchScreen (7 tabs) | StockResearchNode (全屏编辑) | StockResearchNode |
| 股票情绪分析 | SentimentNode (融合 AS+FT) | SentimentNode |
| 投资组合屏幕 (34 files) | PortfolioWorkspace (节点组) | 5+ Portfolio 节点 |
| QuantLib (18 模块, 590 端点) | Python gRPC QuantLib 服务 | QuantLibNode |
| 曲面分析 (35 曲面) | SurfaceAnalytics 面板 | SurfaceNode |
| 回测 (6 提供商) | 合并入 Workflow BacktestNode | BacktestNode |
| Vision Quant | Python gRPC Vision 服务 | VisionQuantNode |

#### 4.1.5 AI Quant Lab

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| Qlib (15 模型) | Python gRPC Qlib 服务 | QlibTrainNode, QlibBacktestNode |
| 强化学习 (5 算法) | Python RL 服务 | RLTrainingNode, RLEvalNode |
| 高频交易 | Python HFT 服务 | HFTOrderBookNode, MarketMakingNode |
| 特征工程 (16 指标) | Go/Python 混合 | FeatureEngineeringNode |
| 高级回测 (执行算法) | Go ExecutionSimulator | AdvancedBacktestNode |
| 在线学习/元学习 | Python OnlineLearning 服务 | OnlineLearningNode |

#### 4.1.6 另类数据模块

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| Polymarket/Kalshi | PredictionMarketNode | PredictionNode |
| 海事智能 | MaritimeService (Go + Python) | MaritimeNode |
| 地缘政治 | GeopoliticsNode | GeopoliticsNode |
| 政府数据 | GovDataNode | GovDataNode |
| 卫星数据 (NASA/Sentinel) | SatelliteDataNode (Python) | SatelliteNode |
| 新闻聚合 & NLP | NewsNode (融合 AS+FT) | NewsNode, NLPPipelineNode |

#### 4.1.7 工具与实用功能

| FinceptTerminal 功能 | QuantFlow 实现 | 节点/模块 |
|---------------------|---------------|----------|
| 节点编辑器 (12 类) | **升级为 Workflow Canvas 本身** | 整个主界面 |
| 代码编辑器 | Monaco Editor 面板 | CodeEditorNode |
| Excel/Spreadsheet | 嵌入式电子表格 | SpreadsheetNode |
| Report Builder | 报表生成节点 | ReportNode |
| 笔记 | 工作流注释 + Markdown 节点 | NoteNode, MarkdownNode |
| 文件管理器 | FileManager 面板 | FileNode |
| 数据映射 | 包含在 DataNormalizeNode | 同上 |
| 语音控制 | Go 语音服务 (Whisper API) | VoiceNode |
| 设置 (18 分区) | Settings 面板 (SolidJS) | Settings 页面 |
| 翻译 (12 语言) | i18n (SolidJS i18n) | i18n 系统 |

### 4.2 AstockPursue 核心功能融入方案

| AstockPursue 功能 | 在 QuantFlow 中的位置 | 增强/变化 |
|-------------------|---------------------|----------|
| **工作流引擎** (58 节点) | **核心骨骼** → 扩展为 200+ 节点 | 从 Python asyncio → Go goroutine |
| **TradingEngine** (bar-by-bar) | 合并入 Go Trading Engine | 增加多市场券商接口 (A股/港股/美股/加密) |
| **Alpha Factory** (450 因子) | Alpha Zoo 节点组 | 保持 Python ML，Go 调度 |
| **GP 进化引擎** | FactorEvolutionNode | Python → Go (性能改进) |
| **AI Agent** (89 Skills) | AgentNode + Skill Registry | 融合 MCP 81 工具 |
| **实验管道** | ExperimentNode 节点组 | 保持设计，增强可视化 |
| **市场状态检测** | RegimeNode | 增加 A 股特有状态 |
| **策略进化** | StrategyEvolutionNode | 保持 GP 流水线 |
| **选股器** | ScreenerNode | 增强条件组合 |
| **归因分析** | AttributionNode | 保持 Brinson + 因子分解 |
| **通知系统** | NotifyNode 系列 | 增加更多渠道 |
| **模拟交易** | 合并入 Go PaperEngine | 增强多市场 |
| **策略市场** | StrategyMarketplace 页面 | 保持概念 |
| **定时任务** | ScheduleNode + Cron 引擎 | 增强触发条件 |

---

## 5. 数据架构

### 5.1 核心原则: 进程内优先、零网络往返

QuantFlow 是桌面应用，不是分布式系统。数据架构遵循:

1. **能进程内解决的不走网络** — 发布/订阅用 Go channel，缓存用 `sync.Map`
2. **能用文件的不起服务** — SQLite 是库调用，不是独立进程
3. **Python 只做计算，不做存储** — gRPC 传的是计算请求，不是数据查询

### 5.2 MarketDataHub — 类型安全的 Go Channel 总线

FinceptTerminal 的 DataHub 是 C++ 单例，用 `QMetaObject::invokeMethod` 跨线程调用。
AstockPursue 用 `asyncio.Queue` 在协程间传递数据。

QuantFlow 的答案: **Go channel 天然就是类型安全的发布/订阅**。

```go
// MarketDataHub — 编译期类型检查的发布/订阅核心
// 不需要 Redis、不需要序列化、不需要网络。
type MarketDataHub struct {
    // 每个 topic 一个广播器
    topics   map[string]*topicBroker
    mu       sync.RWMutex
    producers map[string]Producer

    // 全局反压控制
    maxSubscriberBuffer int  // 默认 64
}

// 每个 topic 独立的 goroutine 负责广播
type topicBroker struct {
    subscribers map[string]chan<- MarketMessage  // 订阅者通道
    cache       *CachedMessage                   // 最新值 + TTL
    producer    Producer
    ttl         time.Duration
    stopCh      chan struct{}
}

func (h *MarketDataHub) Subscribe(topic string, subID string) (<-chan MarketMessage, func()) {
    broker := h.getOrCreateBroker(topic)

    ch := make(chan MarketMessage, h.maxSubscriberBuffer)
    broker.subscribers[subID] = ch

    // 有缓存立即发送
    if cached := broker.cache; cached != nil && !cached.Expired() {
        select {
        case ch <- cached.msg:
        default:
        }
    }

    // 返回退订函数
    unsubscribe := func() {
        broker.unsubscribe(subID)
        close(ch)
    }
    return ch, unsubscribe
}

// 每个 broker 一个广播 goroutine，避免锁竞争
func (b *topicBroker) broadcastLoop() {
    ticker := time.NewTicker(b.ttl / 2)
    defer ticker.Stop()
    for {
        select {
        case msg := <-b.incoming:
            b.cache = &CachedMessage{msg: msg, at: time.Now()}
            for _, ch := range b.subscribers {
                select {
                case ch <- msg:
                default: // 慢消费者丢帧，记指标
                }
            }
        case <-ticker.C:
            b.refreshIfStale()
        case <-b.stopCh:
            return
        }
    }
}
```

**关键设计差异**:
- **FinceptTerminal**: `DataHub::subscribe()` 返回 void，通过 Qt 信号/槽回调 → 类型不安全，运行时崩溃风险
- **AstockPursue**: `asyncio.Queue` 传递 Python dict → 无类型保证，运行时 KeyError
- **QuantFlow**: `<-chan MarketMessage` → 编译期类型检查，IDE 自动补全，重构安全

### 5.3 存储设计: SQLite 单文件，分区表

```sql
-- 核心表设计（全部在一个 SQLite 文件中）

-- 工作流（JSON 字段存储 DAG，SQLite 5.45+ 支持 JSONB 二进制存储）
CREATE TABLE workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    dag_json BLOB NOT NULL,          -- JSONB 压缩存储 DAG 定义
    viewport_json BLOB,              -- 画布视口状态
    created_at INTEGER NOT NULL,     -- unix 毫秒
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1,
    parent_version_id TEXT            -- 版本链
);
CREATE INDEX idx_workflows_updated ON workflows(updated_at);

-- 工作流执行快照（可复现）
CREATE TABLE workflow_runs (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id),
    dag_snapshot BLOB NOT NULL,      -- 执行时的完整 DAG 副本
    status TEXT NOT NULL,            -- running|completed|failed|cancelled
    started_at INTEGER NOT NULL,
    finished_at INTEGER,
    node_results_json BLOB           -- {nodeID: outputs} 嵌套 JSON
);

-- K 线缓存（分区表，按 symbol+interval 组织）
CREATE TABLE ohlcv_cache (
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,          -- 1m|5m|1h|1d|1w|1M
    ts INTEGER NOT NULL,             -- bar 时间戳
    open REAL, high REAL, low REAL, close REAL, volume REAL,
    fetched_at INTEGER NOT NULL,     -- 抓取时间，用于 TTL 淘汰
    PRIMARY KEY (symbol, interval, ts)
) WITHOUT ROWID;                    -- 无 rowid 表，减少存储 30%+

-- 交易记录（不可变追加）
CREATE TABLE trades (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL,
    broker TEXT NOT NULL,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,              -- buy|sell
    order_type TEXT NOT NULL,        -- market|limit|stop|stop_limit
    qty REAL NOT NULL,
    price REAL,                      -- NULL for market orders
    filled_qty REAL,
    filled_avg_price REAL,
    status TEXT NOT NULL,
    placed_at INTEGER NOT NULL,
    filled_at INTEGER,
    error_reason TEXT,               -- 失败原因
    broker_order_id TEXT             -- 券商返回的订单 ID
);
CREATE INDEX idx_trades_account ON trades(account_id, placed_at);

-- 用户配置（单行 JSON）
CREATE TABLE user_config (
    key TEXT PRIMARY KEY,
    value_json BLOB NOT NULL,        -- JSONB
    updated_at INTEGER NOT NULL
);
```

**为什么不需要多数据库**:

| 需求 | 错误方案 | 正确方案 |
|------|---------|---------|
| 实时行情推送 | Redis PubSub | Go channel goroutine 广播（零拷贝） |
| 行情缓存 | Redis KV | `sync.Map` + TTL 惰性淘汰 |
| K 线查询 | DuckDB 列存 | SQLite WITH COVERING INDEX，100 万行查询 <3ms |
| 因子计算 | PostgreSQL | Python sidecar 内存 DataFrame |
| 向量相似搜索 | pgvector | 本地 FAISS/Bolt 嵌入式索引 |
| 工作流持久化 | PostgreSQL JSONB | SQLite JSONB（5.45+ 原生支持） |
| 多用户隔离 | PostgreSQL schema | **桌面应用不存在多用户问题** |
| 冷归档 | Parquet 文件 | SQLite 离线备份（单文件复制即可） |

### 5.4 缓存分层（全部进程内）

```
请求到达 → L0 检查 → L1 检查 → Python 抓取 → 回填

L0: 内存 sync.Map
    ├── 实时报价 (TTL 5s，goroutine 定时清扫)
    ├── 最近查询的去重 (100ms 窗口合并，对标 FT 的 coalesce)
    └── 显示名称缓存 (symbol→name，永久，对标 FT SettingsRepository)

L1: SQLite ohlcv_cache 表
    ├── K 线按 (symbol, interval) 分区
    ├── TTL 策略: 1m bars=2h, 5m=1d, 1h=7d, 1d=永久
    ├── 自动清理: INSERT 时检查同 (symbol,interval) 行数，超出阈值删最旧
    └── 无网络时完全可用（离线模式）

L2: Python 数据脚本 (gRPC)
    └── 只在 L0 miss 且 L1 miss/stale 时触发
```

### 5.5 Go ↔ Python 通信: 精准的 gRPC 边界

```
Go 负责:                                Python 负责:
┌──────────────────────────┐            ┌──────────────────────┐
│ 工作流 DAG 调度           │            │ 因子计算 (pandas)     │
│ 交易引擎 (bar-by-bar)     │  gRPC     │ ML 训练/推理         │
│ 市场数据 Hub (实时行情)   │◄────────►│ LLM 推理 (ReAct)     │
│ 券商 REST/WS 连接        │  protobuf │ Qlib/QuantLib 调用   │
│ SQLite 读写               │            │ 数据抓取脚本         │
│ 前端 API/WS/SSE 服务      │            │ GP 进化引擎          │
└──────────────────────────┘            └──────────────────────┘

关键决策: Python 不直接访问 SQLite。
    ├── Python 只做计算: 接收 DataFrame → 计算 → 返回 DataFrame
    ├── 所有存储操作由 Go 统一管理，保证单写者
    └── gRPC 请求/响应用 Arrow Flight 序列化 DataFrame（零拷贝）
```

### 5.6 离线模式设计

桌面终端的关键优势 — 没有网络也能工作:

```
联网模式:                          离线模式:
用户操作 → L0(miss) → L1(miss)   用户操作 → L0(miss) → L1(hit)
           → gRPC Python            → 直接返回
           → 外部 API               → UI 显示 "离线 | 数据截止 XX:XX"
           → 写入 L0+L1

切换检测: Go 后台 goroutine 每 30s ping 外部 API
         → 网络恢复自动切回联网模式
         → 积压的数据请求批量补充
```

---

## 6. AI 智能体系统

### 6.1 原创设计: 不是 Agent+Tools，而是 "工作流节点作为 LLM 函数"

两个源项目各自有一个 AI 系统:
- FinceptTerminal: MCP 协议 + 81 工具 + 37 角色 → LLM 通过工具调用访问终端功能
- AstockPursue: 89 Skill + ReAct Agent → LLM 通过 prompt 注入领域知识

两者的问题相同: **AI 是「外挂」，不是「原生」**。用户需要在聊天框和功能界面之间切换。

**QuantFlow 的方案: 工作流节点本身就是 LLM 可调用的 function**。

```
传统模式 (FT + AS):
    用户 → 聊天框 → LLM → 调用工具 → 返回结果 → 用户复制到功能界面

QuantFlow 模式:
    用户 → 拖入 AgentNode → 连接上游数据节点 → 连接下游策略节点
          → AgentNode 的输出是类型化的 port
          → 下游节点直接消费，无需手动搬运
```

### 6.2 统一 Tool/Skill 注册 — 从 81+89 到 统一适配

不维护两套工具系统，而是所有能力统一注册:

```go
// 统一的能力注册表 — 既可以被 AgentNode 调用，也可以被工作流节点调用
type Capability struct {
    Name        string
    Description string           // LLM function description
    Parameters  jsonschema.Schema
    Handler     func(ctx context.Context, params json.RawMessage) (any, error)
    
    // 同时注册为工作流节点
    NodeType    string           // 对应的节点类型
}

// 注册来源:
// 1. 内置 Go 函数 (获取行情、下单、读取数据库...)
// 2. Python gRPC 函数 (计算因子、训练模型、LLM 推理...)
// 3. Skill 知识库 (领域特定的 prompt 模板，注入到 LLM system prompt)
```

### 6.3 智能体角色 → Agent Profile

FinceptTerminal 的 37 个角色 + AstockPursue 的 10 个投资人角色 → 统一为 **Agent Profile YAML**:

```yaml
# profiles/warren_buffett.yaml
name: warren_buffett
display: "Warren Buffett — 价值投资"
system_prompt: |
  你采用巴菲特的价值投资方法论。你关注:
  - 经济护城河: 品牌、转换成本、网络效应、成本优势
  - 所有者收益: FCF 调整、维持性资本支出
  - 管理层质量: 资本配置记录、股东友好度
  - 估值纪律: 安全边际至少 30%
  
  你的回答应包括: 护城河评级(1-5) + 内在价值估算 + 建议安全边际买入价

tools: [quote_lookup, financials, news_search, dcf_calculator]
default_llm: anthropic/claude-opus-4-8
```

### 6.4 AgentNode 在工作流中的定位

AgentNode 不是聊天窗口——它是工作流中的一个**类型化转换节点**:

```
输入 ports:                    输出 ports:
┌──────────────┐              ┌──────────────┐
│ prompt: str  │              │ code: str    │ → 连接 CodeEvalNode
│ context: any │→ AgentNode →│ analysis:str │ → 连接 ReportNode
│ data: df     │              │ signal: df   │ → 连接 StrategyNode
│ constraints  │              │ factors:list │ → 连接 AlphaZooNode
└──────────────┘              └──────────────┘

具体场景:
1. [StockUniverse] → [OHLCVLoader] → [AgentNode(llm=gpt-4o)] → [StrategyNode]
   Agent 根据 OHLCV 数据生成 SignalEngine 代码，直接注入 StrategyNode 执行

2. [NewsNode] → [AgentNode(llm=claude)] → [SentimentScore] → [TradingNode]
   Agent 分析新闻并输出情绪信号，直接驱动交易决策

3. [FactorZooNode] → [AgentNode(llm=deepseek)] → [FactorSelection] → [BacktestNode]
   Agent 分析因子 IC 报告，选择最优因子组合，进入回测
```

### 6.5 为什么不需要 LangChain/LlamaIndex

两个源项目都深度依赖 LangChain。QuantFlow 刻意不引入:

- LangChain 的 Agent 循环在 Python 进程中运行 → 与 Go 工作流引擎割裂
- LangChain 的 Tool 抽象与工作流节点的类型系统冲突
- ReAct 循环的每一步都应该是一个工作流节点执行事件，前端可观测

**替代方案**: Go 侧实现轻量 Agent 循环，LLM 推理通过 gRPC 调用 Python:

```go
// Go 侧的 Agent 循环 — 每一步都向工作流引擎报告状态
func (a *AgentExecutor) Run(ctx context.Context, node *AgentNode, inputs map[string]any) (*AgentResult, error) {
    messages := []Message{{Role: "system", Content: node.SystemPrompt}}
    messages = append(messages, Message{Role: "user", Content: formatInputs(inputs)})
    
    for step := 0; step < node.MaxSteps; step++ {
        // 1. LLM 推理 (gRPC → Python)
        resp := a.llmClient.Chat(ctx, messages, node.Tools)
        
        // 2. 如果要调用工具 → 在工作流引擎上下文中执行
        if resp.ToolCalls != nil {
            for _, tc := range resp.ToolCalls {
                result := a.toolRegistry.Execute(ctx, tc.Name, tc.Args)
                messages = append(messages, Message{Role: "tool", Content: result})
                // 发射事件: 前端可看到 Agent 调用了哪个工具
                a.emitEvent(node.ID, "tool_call", tc.Name, result)
            }
            continue
        }
        
        // 3. 最终响应 → 解析为类型化输出端口
        return a.parseOutput(resp.Content, node.OutputPorts), nil
    }
    return nil, ErrMaxStepsExceeded
}
```

---


---

## 7. 前端设计: 双模式终端

### 7.1 设计理念: 彭博终端体验 + 工作流编排

FinceptTerminal 的灵魂是 **彭博式面板终端**——不是「应用里有几个页面」，而是「用户自由组合几十个面板，键盘驱动，数据实时流动」。这个体验必须保留和强化。

AstockPursue 的灵魂是 **可视化工作流**——量化研究的最佳交互方式是拖拽 DAG。

QuantFlow 把两者放在同一个应用里，**不是两个模式二选一，而是同一个底层的两种视图，可双向流通**。

### 7.2 Terminal Mode — 默认启动模式

这是用户打开应用看到的第一界面。对标 FinceptTerminal 的 ADS 停靠系统 + 彭博终端的键盘驱动。

```
┌──────────────────────────────────────────────────────────────────────┐
│  Menu Bar                                                             │
├──────────────────────────────────────────────────────────────────────┤
│  Command Bar:  AAPL US EQUITY  [实时报价 $195.32 △2.1%]  [Ctrl+K]   │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────────┐ ┌──────────────────────┐ ┌──────────────────┐  │
│  │ ★ Watchlist     │ │ AAPL — K 线图         │ │ AAPL — 订单簿     │  │
│  │ AAPL  195.32 △  │ │ ┌──────────────────┐ │ │ Bid    195.30 500│  │
│  │ GOOGL 142.15 ▽  │ │ │   /\    /\       │ │ │ Ask    195.35 200│  │
│  │ MSFT  378.91 △  │ │ │  /  \  /  \  /\  │ │ │                  │  │
│  │ TSLA  245.30 ▽  │ │ │ /    \/    \/  \ │ │ │ 最近成交          │  │
│  │                  │ │ │─────────────────│ │ │ 195.30  100      │  │
│  │ [添加自选]       │ │ └──────────────────┘ │ │ 195.32  500      │  │
│  └─────────────────┘ └──────────────────────┘ └──────────────────┘  │
│                                                                       │
│  ┌───────────────────────────────┐ ┌──────────────────────────────┐  │
│  │ AAPL — 股票研究 (7 Tab)        │ │ 新闻流 — AAPL                │  │
│  │ [概览][财务][技术][同行][情绪]  │ │ • Apple Q2 财报超预期         │  │
│  │ 市值: $3.02T   PE: 32.5       │ │ • iPhone 出货量增长 8%        │  │
│  │ EPS: $6.43     股息率: 0.48%  │ │ • 分析师上调目标价             │  │
│  └───────────────────────────────┘ └──────────────────────────────┘  │
│                                                                       │
├──────────────────────────────────────────────────────────────────────┤
│  StatusBar: ●connected | 2券商在线 | 12数据流 | 内存 1.8G | 离线模式  │
│  [PushPin: AAPL] [PushPin: BTCUSD] [PushPin: MyStrategy]             │
└──────────────────────────────────────────────────────────────────────┘
```

**彭博终端关键交互**:

```
1. 命令栏 (Command Bar) — 对标彭博的键盘驱动
   ├── Ctrl+K 激活命令栏
   ├── 输入股票代码 → 打开股票研究面板
   ├── 输入 "opt AAPL" → 打开期权链面板
   ├── 输入 "/wf my-strat" → 切换到 Workflow Mode 打开工作流
   ├── 输入 "/heatmap" → 打开市场热力图
   └── 历史记录 + 模糊匹配 + 自动补全

2. 停靠面板系统 (对标 ADS CDockManager)
   ├── 面板可拖拽到任何位置（停靠/标签/浮动）
   ├── 自动网格: 1=全屏, 2=左右分, 3=上二下一, 4=2x2
   ├── 撕下面板到独立窗口（对标 ADS tear_off_to_new_frame）
   ├── Focus Mode: Ctrl+Shift+F 隐藏所有 chrome
   ├── 布局保存/恢复: 命名布局 ("trading", "research", "morning-brief")
   └── 多显示器支持

3. PushPin Bar (对标 FT PushpinBar)
   └── 将面板或符号固定在底部，跨布局持久存在

4. 实时数据 (对标 FT DataHub 订阅)
   └── 每个面板独立订阅数据 topic，Go channel → Wails IPC → Vue 响应式
```

### 7.3 面板目录 (50+ Terminal Panels)

对标 FinceptTerminal 的 56 屏幕 + Component Catalog 的 46 组件。这些不是工作流节点——它们是彭博式即开即用的面板:

```
── 市场 (8)
│   WatchlistPanel | MarketOverviewPanel | QuoteDetailPanel
│   SparklinePanel | MarketDepthPanel | TickerTapePanel
│   HeatmapPanel | CryptoOverviewPanel
│
├── 图表 (6)
│   CandlestickPanel | EquityCurvePanel | SurfaceChartPanel
│   CorrelationPanel | DistributionPanel | DrawingPanel
│
├── 交易 (8)
│   OrderEntryPanel | PositionPanel | OrderBlotterPanel
│   ExecutionPanel | BasketOrderPanel | BrokerStatusPanel
│   ActionCenterPanel | WebhookMonitorPanel
│
├── 研究 (8)
│   StockResearchPanel (7 tab) | FinancialsPanel | AnalystEstimatesPanel
│   PeerComparisonPanel | NewsPanel | SentimentPanel
│   InsiderTradingPanel | CongressTradingPanel
│
├── 投资组合 (6)
│   PortfolioSummaryPanel | AllocationPanel | PerformancePanel
│   RiskPanel (VaR/CVaR) | MonteCarloPanel | RebalancePanel
│
├── 另类数据 (5)
│   PredictionMarketPanel | MaritimePanel | GeopoliticsPanel
│   SatellitePanel | GovDataPanel
│
├── AI & 量化 (5)
│   AIChatPanel | QuantLibPanel | BacktestResultPanel
│   FactorAnalysisPanel | ExperimentPanel
│
├── 工具 (5)
│   SpreadsheetPanel | CodeEditorPanel | ReportPanel
│   NotesPanel | FileManagerPanel
│
└── 监控 (3)
    AlertPanel | SchedulerPanel | SystemMonitorPanel
```

### 7.4 Workflow Mode — Ctrl+W 切换

```
┌──────────────────────────────────────────────────────────────────┐
│  [★ Terminal Mode] [Workflow Mode]                    [Ctrl+W]   │
├──────────┬───────────────────────────────────────┬───────────────┤
│ 节点面板  │       Workflow Canvas (vue-flow)      │ 属性/日志     │
│          │                                        │               │
│ 搜索框    │  [自选列表] → [OHLCV] → [AlphaZoo]    │ ┌───────────┐ │
│ ┌──────┐ │       │            │          │        │ │ 节点属性    │ │
│ │Data  │ │       │            │          ▼        │ │ params...  │ │
│ │Alpha │ │       │         [策略节点] → [回测]    │ └───────────┘ │
│ │Signal│ │       │            │            │       │               │
│ │Strat │ │       │            ▼            ▼       │ ┌───────────┐ │
│ │Backt │ │       │       [AI分析]  [净值曲线]      │ │ 执行日志    │ │
│ │Trade │ │       │            │                    │ │ Node1: ok   │ │
│ │...   │ │       │            ▼                    │ │ Node2: ...  │ │
│ └──────┘ │       │       [下单] → [通知]           │ └───────────┘ │
│          │       │                                        │               │
│ 模板列表  │       │  [MiniMap]              [缩放 50%]   │ 节点右键 →    │
│ ┌──────┐ │       │                                        │ "固定到终端"   │
│ │因子IC │ │       │                                        │               │
│ │选股   │ │       │                                        │               │
│ └──────┘ │       │                                        │               │
└──────────┴───────────────────────────────────────┴───────────────┘
```

**Terminal ↔ Workflow 双向流动**:

```
Terminal → Workflow:
    任意面板右上角 [⊕ 添加到工作流]
    → 将该面板的当前状态（symbol, 参数, 数据源）创建为工作流节点
    → 自动切换到 Workflow Mode，新节点已放置在画布中央

Workflow → Terminal:
    节点右键 → [固定到终端]
    → 在 Terminal Mode 中创建对应的实时面板
    → 面板标题标注 "WF: 工作流名称"
    → 每次工作流执行结束，面板数据自动更新
```

### 7.5 布局系统 — 对标 ADS 的 DockView

FinceptTerminal 用 Qt ADS (Advanced Docking System) 实现停靠面板。QuantFlow 在 Vue 中自研轻量 DockView:

```
DockView 组件架构:
├── DockContainer.vue (递归容器)
│   ├── 方向: horizontal | vertical
│   ├── 子节点: [DockTab[], ...]
│   └── 分割比例: [0.3, 0.4, 0.3]
│
├── DockTab.vue (面板容器)
│   ├── 标签: 面板名称 + 实时指示器 (绿点=数据活跃)
│   ├── 内容: <component :is="panel.component" />
│   └── 操作: 关闭 / 浮动 / 撕下 / 固定
│
├── 预置布局 (对标 ADS auto grid):
│   single | split-h | split-v | 2x2 | classic | trading
│
├── 浮动窗口 (对标 ADS tear_off):
│   └── Wails 多窗口 API，独立渲染
│
└── 布局持久化: SQLite user_config key="layout.{name}"
```

### 7.6 前端组件树

```
App.vue
├── TerminalMode.vue (默认)
│   ├── CommandBar.vue             # 对标彭博命令栏
│   ├── DockView.vue               # 停靠面板系统
│   │   ├── DockContainer.vue      #   递归分割容器
│   │   ├── DockTab.vue            #   标签页 + 面板渲染
│   │   └── DockSplitter.vue       #   分割条拖拽
│   ├── panels/                    # 50+ 彭博式面板
│   │   ├── WatchlistPanel.vue
│   │   ├── CandlestickPanel.vue
│   │   ├── OrderEntryPanel.vue
│   │   ├── StockResearchPanel.vue
│   │   └── ... (50+)
│   ├── PushPinBar.vue             # 对标 FT
│   ├── StatusBar.vue
│   └── PanelCatalog.vue           # 面板目录 (Ctrl+B)
│
├── WorkflowMode.vue (Ctrl+W)
│   ├── WorkflowCanvas.vue         # vue-flow
│   ├── NodePalette.vue            # 16 类节点
│   ├── TemplatesPanel.vue         # 预置模板
│   ├── PropertyPanel.vue          # 节点配置
│   └── ExecutionLog.vue           # 实时日志
│
└── shared/
    ├── charts/                    # ECharts 封装
    ├── MonacoEditor.vue
    ├── Spreadsheet.vue
    └── theme/
```

### 7.7 Pinia 状态 (4 Store, 双模式共享)

```typescript
// 1. terminalStore — Terminal Mode 状态
stores/terminal.ts:
    layout: DockLayoutTree          // 面板布局树
    activePanels: Map<id, PanelState>  // 活跃面板及其参数
    commandHistory: string[]        // 命令历史
    pushPins: PushPinItem[]         // 固定项
    focusMode: boolean

// 2. workflowStore — Workflow Mode 状态
stores/workflow.ts:
    canvas: { nodes, edges, viewport, selectedId }
    runner: { status, nodeStatuses, progress, runId }
    templates: WorkflowTemplate[]
    clipboard: { nodes }

// 3. dataStore — 统一数据层 (两种模式共享)
stores/data.ts:
    subscriptions: Map<topic, Subscriber[]>
    quotes: Map<symbol, QuoteSnapshot>     // 实时行情
    ohlcv: Map<key, CachedOHLCV>           // K 线
    sourceStatus: DataSourceStatus[]       // 数据源状态

// 4. sessionStore — 用户 + UI 偏好
stores/session.ts:
    user: { token, profile }
    ui: { theme, density, language, mode }  // mode: 'terminal'|'workflow'
    brokers: BrokerConnection[]
```

### 7.8 键盘快捷键

```
全局:
    Ctrl+K          → 激活命令栏 (任何模式下)
    Ctrl+B          → 打开面板目录
    Ctrl+W          → 切换 Terminal/Workflow Mode
    Ctrl+Shift+F    → Focus Mode
    Ctrl+数字        → 切换到保存的布局 1-9

Terminal Mode:
    Esc             → 关闭当前活动面板
    Ctrl+T          → 撕下当前面板到独立窗口
    Ctrl+D          → 复制当前面板
    Space           → 暂停/恢复实时数据

Workflow Mode:
    Delete          → 删除选中节点/连线
    Ctrl+Z / Ctrl+Shift+Z → 撤销/重做
    F5              → 执行整个工作流
    F9              → 执行选中节点
    F10             → 从选中节点执行到末尾
```

---

## 8. 项目目录结构

```
quantflow/
├── README.md
├── LICENSE (AGPL-3.0)
├── CHANGELOG.md
├── Makefile
├── wails.json                      # Wails 项目配置
│
├── app/                            # Go 后端 (Wails 应用主体)
│   ├── go.mod / go.sum
│   ├── main.go                     # Wails 入口 + 窗口创建
│   ├── app.go                      # App struct — 暴露给前端的 Go 函数
│   ├── internal/
│   │   ├── config/                 # 配置管理 (Viper + YAML)
│   │   ├── workflow/               # ★ 工作流引擎
│   │   │   ├── engine.go           #   DAG 执行引擎 (Kahn + goroutine)
│   │   │   ├── schema.go           #   节点/端口定义、连接验证
│   │   │   ├── node.go             #   BaseNode 接口
│   │   │   ├── registry.go         #   节点注册表
│   │   │   ├── cache.go            #   LRU 节点缓存
│   │   │   ├── debugger.go         #   断点调试
│   │   │   ├── store.go            #   SQLite 持久化 + 版本管理
│   │   │   ├── templates/          #   预置工作流模板
│   │   │   └── nodes/              #   16 类节点实现
│   │   │       ├── data.go         #     数据加载节点
│   │   │       ├── alpha.go        #     Alpha 因子节点
│   │   │       ├── indicator.go    #     技术指标节点
│   │   │       ├── signal.go       #     信号生成节点
│   │   │       ├── strategy.go     #     策略构建节点
│   │   │       ├── backtest.go     #     回测执行节点
│   │   │       ├── attribution.go  #     归因分析节点
│   │   │       ├── risk.go         #     风控分析节点
│   │   │       ├── trading.go      #     交易执行节点
│   │   │       ├── market.go       #     市场数据节点
│   │   │       ├── portfolio.go    #     投资组合节点
│   │   │       ├── research.go     #     研究分析节点
│   │   │       ├── agent.go        #     AI 智能体节点
│   │   │       ├── notify.go       #     通知输出节点
│   │   │       ├── control.go      #     控制流节点
│   │   │       └── utility.go      #     工具节点
│   │   ├── trading/                # 交易引擎
│   │   │   ├── engine.go           #   统一 bar-by-bar 管线
│   │   │   ├── risk_pipeline.go    #   风控管线
│   │   │   ├── signal_adapter.go   #   信号适配器 (Tick/Batch)
│   │   │   ├── oms.go              #   订单管理系统
│   │   │   ├── paper_engine.go     #   模拟交易引擎
│   │   │   ├── order_matcher.go    #   撮合引擎
│   │   │   ├── position.go         #   仓位计算
│   │   │   ├── backtest.go         #   回测驱动器
│   │   │   ├── live.go             #   实盘驱动器
│   │   │   ├── ws_feed.go          #   WebSocket 行情接收
│   │   │   └── brokers/            #   券商适配器 (聚焦 A 股/港股/美股/加密)
│   │   │       ├── broker.go       #     Broker 接口
│   │   │       ├── registry.go     #     券商注册
│   │   │       ├── futu/           #     Futu (A/HK/US — 核心)
│   │   │       ├── longport/       #     LongPort (A/HK/US)
│   │   │       ├── ibkr/           #     IBKR (全球)
│   │   │       ├── alpaca/         #     Alpaca (美股)
│   │   │       ├── binance/        #     Binance (加密)
│   │   │       ├── okx/            #     OKX (加密)
│   │   │       └── ...
│   │   ├── market/                 # 市场数据中枢
│   │   │   ├── hub.go              #   Go channel 发布/订阅
│   │   │   ├── adapters/           #   数据源适配器 (100+)
│   │   │   │   ├── adapter.go      #     Adapter 接口
│   │   │   │   ├── yahoo.go
│   │   │   │   ├── polygon.go
│   │   │   │   ├── eastmoney.go
│   │   │   │   └── ...
│   │   │   └── normalize.go        #   数据归一化
│   │   ├── ai/                     # AI 智能体
│   │   │   ├── agent.go            #   Agent 循环 (Go 实现)
│   │   │   ├── capability.go       #   统一工具/Skill 注册
│   │   │   ├── profiles/           #   智能体角色 YAML
│   │   │   └── mcp.go              #   MCP Provider
│   │   ├── portfolio/              # 投资组合 + 风险分析
│   │   ├── research/               # 股票研究 + 情绪 + 新闻 NLP
│   │   ├── notify/                 # 通知引擎 (Telegram/Discord/...)
│   │   ├── schedule/               # 定时任务 (cron)
│   │   ├── auth/                   # JWT + 本地锁屏
│   │   ├── storage/                # SQLite 封装
│   │   │   ├── db.go               #   连接管理 (WAL 模式)
│   │   │   ├── migrate.go          #   迁移系统
│   │   │   └── queries/            #   SQL 查询
│   │   ├── python/                 # Python gRPC 桥接
│   │   │   ├── bridge.go           #   gRPC 客户端管理
│   │   │   └── proto/              #   protobuf 生成代码
│   │   └── crypto/                 # AES-256-GCM 密钥存储
│   └── proto/                      # proto 定义
│       ├── factor.proto
│       ├── ml.proto
│       └── llm.proto
│
├── python/                         # Python gRPC Sidecar
│   ├── pyproject.toml
│   ├── src/
│   │   ├── server.py               #   gRPC 服务入口
│   │   ├── factor/                 #   因子计算 (450+)
│   │   ├── ml/                     #   ML 服务 (Qlib/PyTorch/RL)
│   │   ├── llm/                    #   LLM 推理
│   │   ├── data/                   #   数据抓取脚本
│   │   ├── analytics/              #   分析脚本
│   │   └── quantlib/               #   QuantLib 服务
│   └── tests/
│
├── frontend/                       # Vue 3 前端 (Wails WebView)
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   ├── src/
│   │   ├── main.ts                 # Vue 入口 + Pinia + Wails 绑定
│   │   ├── App.vue
│   │   ├── terminal/               # ★ Terminal Mode
│   │   │   ├── TerminalMode.vue    #   终端模式容器
│   │   │   ├── CommandBar.vue      #   命令栏
│   │   │   ├── DockView.vue        #   停靠系统
│   │   │   ├── PushPinBar.vue      #   固定栏
│   │   │   ├── StatusBar.vue
│   │   │   └── panels/             #   50+ 彭博式面板
│   │   │       ├── WatchlistPanel.vue
│   │   │       ├── CandlestickPanel.vue
│   │   │       ├── OrderEntryPanel.vue
│   │   │       ├── StockResearchPanel.vue
│   │   │       └── ... (50+)
│   │   ├── workflow/               # ★ Workflow Mode
│   │   │   ├── WorkflowMode.vue    #   工作流模式容器
│   │   │   ├── WorkflowCanvas.vue  #   vue-flow 画布
│   │   │   ├── NodePalette.vue     #   节点面板
│   │   │   ├── PropertyPanel.vue   #   属性编辑
│   │   │   └── ExecutionLog.vue    #   执行日志
│   │   ├── stores/                 # Pinia (4 stores)
│   │   │   ├── terminal.ts
│   │   │   ├── workflow.ts
│   │   │   ├── data.ts
│   │   │   └── session.ts
│   │   ├── lib/                    # 工具库
│   │   │   ├── i18n/               #   12 语言
│   │   │   ├── theme.ts            #   主题/密度
│   │   │   └── format.ts           #   数据格式化
│   │   └── types/
│   └── tests/
│
├── resources/                      # 静态资源
│   ├── app-icon.png
│   ├── templates/                  # 工作流模板 JSON
│   ├── agent-profiles/             # 智能体角色 YAML
│   └── skills/                     # AI 知识库 (Markdown)
│
├── scripts/                        # 构建脚本
│   ├── install.sh
│   └── build.sh
│
└── docs/
    ├── architecture.md
    ├── development.md
    └── user-guide.md
```

---

## 9. 实现路线图

### 阶段 1: 核心骨架 (第 1-3 月)

```
目标: Terminal Mode 可运行 + Workflow Mode 可拖拽连接

1.1 Go 项目骨架 (2 周)
    ├── Wails v3 项目初始化、Go module
    ├── 配置管理 (Viper + YAML)、日志 (slog)
    ├── SQLite 连接 (WAL 模式) + 迁移框架
    └── App struct 前置函数注册

1.2 工作流引擎核心 (4 周)
    ├── BaseNode 接口 + Schema 系统 (端口类型、连接验证)
    ├── NodeRegistry 注册机制
    ├── DAG 拓扑排序 + goroutine 逐层并行执行
    ├── SQLite 工作流持久化 + 版本历史
    └── LRU 节点缓存 + 断点调试

1.3 Terminal Mode + 首批面板 (3 周)
    ├── DockView 停靠系统 (DockContainer + DockTab + DockSplitter)
    ├── CommandBar 命令栏 (Ctrl+K 搜索面板/符号/命令)
    ├── 首批 10 面板: Watchlist, QuoteDetail, Candlestick, NewsPanel,
    │   OrderEntry, Position, StockResearch, AIChat, Notes, SystemMonitor
    ├── PushPinBar + StatusBar
    └── 布局保存/恢复

1.4 Workflow Mode + 前端集成 (3 周)
    ├── vue-flow 画布 (VueFlow + CustomNode + ConnectionLine)
    ├── NodePalette (16 类可搜索节点面板)
    ├── PropertyPanel (动态属性编辑表单)
    ├── Terminal ↔ Workflow 双向切换
    └── [⊕ 添加到工作流] + [固定到终端]
```

### 阶段 2: 交易引擎 + 市场数据 (第 4-6 月)

```
2.1 Go 交易引擎 (4 周)
    ├── TradingEngine.on_bar() 统一管线
    ├── 信号适配器 (Tick/Batch 模式)
    ├── 风控管线 (止损/止盈/追踪止损)
    ├── 订单管理系统 (OMS)
    └── 模拟交易引擎

2.2 市场数据中枢 (3 周)
    ├── MarketDataHub (Go channel 发布/订阅)
    ├── TTL 管理、冷启动预取、请求合并
    ├── 数据源适配器接口 + 首批 8 数据源
    │   (A股: EastMoney/AKShare/TuShare → 港股: Futu/新浪 →
    │    美股: Yahoo/Polygon → 加密: Binance/OKX)
    └── 数据归一化 + SQLite 缓存

2.3 券商集成 v1 (3 周)
    ├── Broker 接口定义
    ├── 首批券商: 富途Futu (A+H+美, 核心) + Binance (加密) + Alpaca (美股)
    ├── 交易面板完善: OrderEntry, Position, OrderBlotter,
    │   BasketOrder, BrokerStatus
    └── Webhook 监听器面板
    ├── 交易面板完善: OrderEntry, Position, OrderBlotter,
    │   BasketOrder, BrokerStatus
    └── Webhook 监听器面板

2.4 终端体验完善 (2 周)
    ├── 面板目录扩充到 25+
    ├── 多窗口支持 (Wails 多窗口 API)
    ├── Focus Mode (Ctrl+Shift+F)
    └── 布局模板 (trading, research, morning-brief)
```

### 阶段 3: 因子 + 策略 + 回测 (第 7-9 月)

```
3.1 Alpha 因子系统 (3 周)
    ├── Python gRPC Factor 服务
    ├── 450+ 因子迁移
    ├── Go 端因子节点 + GP 进化引擎
    └── FactorAnalysis 面板

3.2 策略 + 回测 (4 周)
    ├── StrategyNode + BacktestNode
    ├── 7 市场引擎 (按优先级):
    │   A股 (T+1/涨跌停/印花税) → 港股 (T+2/港股通) → 美股 (T+2)
    │   → 加密永续 (资金费率/强平) → A股期货 (CFFEX/SHFE/DCE)
    │   → 美股期权 → 外汇
    ├── BacktestResult 面板 (净值曲线+指标+交易明细)
    └── 参数扫描/优化

3.3 归因 + 风控 + 实验 (2 周)
    ├── 归因/风控节点 + 对应面板
    ├── 实验管道节点 (Regime→Variant→Backtest→Score)
    └── SurfaceChart 面板 (35 曲面)

3.4 Qlib 集成 + 研究面板 (2 周)
    ├── Python gRPC Qlib 服务
    ├── StockResearch 面板完善 (7 tab 对标 FT)
    ├── Financials 面板 + PeerComparison 面板
    └── QuantLib 面板
```

### 阶段 4: AI 智能体 + 全功能 (第 10-12 月)

```
4.1 AI 智能体系统 (4 周)
    ├── Go Agent 循环 (ReAct)
    ├── Python gRPC LLM 服务
    ├── 统一 Capability 注册 (MCP + Skill + Workflow Nodes)
    ├── AIChat 面板完善 (流式 SSE, Markdown, CodeBlock)
    └── AgentNode 工作流集成

4.2 投资组合 + 另类数据 (3 周)
    ├── 投资组合面板组 (6 面板)
    ├── 另类数据面板 (5 面板: 预测市场/海事/卫星/地缘政治/政府)
    ├── 新闻 NLP 管道
    └── 情绪分析面板

4.3 工具面板 + 全市场券商 (3 周)
    ├── 工具面板: 电子表格, 代码编辑器, 报表, 文件管理
    ├── A股券商: 华泰/中信/国泰君安 → 港股券商: 富途/长桥/华盛
    │   美股券商: Alpaca/Tradier → 加密: Binance/OKX/Bybit
    ├── 定时任务 + 告警面板
    └── 通知渠道 (Telegram/Discord/Email/Feishu)

4.4 节点扩充到 200+ (2 周)
    └── 全部 16 类节点实现 + 50+ 工作流模板
```

### 阶段 5: 打磨 + 发布 (第 13-15 月)

```
5.1 国际化 + 主题 + 设置 (2 周)
    ├── 12 语言翻译 (对标 FT)
    ├── 主题系统 (亮色/暗色) + 密度 (紧凑/默认/舒适)
    ├── 设置面板 (对标 FT 18 分区)
    └── 字体/排版

5.2 桌面打包 (2 周)
    ├── Wails 构建 (macOS/Windows/Linux)
    ├── Go 交叉编译
    ├── Python sidecar 可选内嵌 (PyInstaller)
    ├── 自动更新
    └── 安装程序

5.3 测试 + 文档 (3 周)
    ├── Go 单元测试 (80%+)
    ├── Python 测试
    ├── 前端组件测试 + E2E
    ├── 压力测试 (100 并发工作流)
    └── 用户指南 + 开发者文档

5.4 策略市场 + v1.0 (3 周)
    ├── 策略/工作流模板发布/浏览
    ├── 性能优化 (Go profiling + 前端优化)
    ├── 安全审计
    └── v1.0 发布
```

---

## 10. 关键设计决策

### 10.1 为什么 SQLite 单数据库而不是 PostgreSQL + Redis + DuckDB？

这是一个桌面应用，不是云服务。核心事实:

| 需求 | 桌面终端现实 | 结论 |
|------|------------|------|
| 并发用户 | **1 个**（本地桌面） | PostgreSQL 的多用户隔离毫无意义 |
| 网络往返 | **0**（全部本地） | Redis 的 TCP 开销是纯损耗 |
| 数据量级 | 百万行 K 线，非十亿 | SQLite WAL 模式下千万行查询 <10ms |
| 运维成本 | **用户不是 DBA** | 零配置是硬需求，SQLite 是唯一解 |
| 实时推送 | 进程内 Go channel | 比 Redis PubSub 快 100x（零拷贝 vs TCP） |

FinceptTerminal 证明了这一点: 用 SQLite 承载了全部 50 个迁移表、K 线缓存、交易记录的完整需求。

### 10.2 为什么要双模式而不是纯工作流画布？

彭博终端用户的核心操作: 输入代码 → 看数据 → 做决策 → 下单。这不是「编排工作流」的场景，而是「快速获取信息」的场景。

如果强制用户通过工作流画布完成这些操作，用户体验会很差:
- 查看 AAPL 报价需要"拖 Data 节点 → 连 Quote 节点 → 执行" → 太慢
- 彭博做法: 输入 `AAPL US EQUITY` → 所有数据即刻展示 → 1 秒

量化研究者用 Workflow Mode 编排自动化管道，交易员/分析师用 Terminal Mode 做日常操作。**同一底层，两种交互**。

### 10.3 为什么 Wails v3 而不是 Tauri？

| 维度 | Wails v3 | Tauri v2 |
|------|---------|---------|
| 壳语言 | Go（与后端同语言、同进程） | Rust（独立工具链） |
| 前后端通信 | Go 函数直接调用，无序列化 | IPC 序列化/反序列化 |
| 编译 | `go build` 一次搞定 | Go+Rust 两套编译 |
| 包体积 | ~8MB 壳 | ~5MB 壳 |

后端是 Go，用 Go 壳消除了 Rust 工具链依赖。如果 Wails v3 不成熟，退回 Wails v2（稳定 3 年+）。

### 10.4 为什么保留 Python？

- pandas/numpy/qlib/quantstats 只有 Python 绑定
- 1000+ FinceptTerminal Python 数据分析脚本不宜重写
- gRPC sidecar 边界清晰: Python 负责计算，Go 负责编排
- 不想用 Python 的用户可以不启动 sidecar —— 核心功能纯 Go

### 10.5 为什么 Vue 3 而不是 React？

- **vue-flow 是 @xyflow 官方 Vue 版** — 成熟度仅次于 React 版
- **Pinia** 结构化程度优于 Zustand（Zustand 太自由，容易长成 14 个 store）
- Vue 的 `computed`/`watch` 响应式系统比 React 的 `useMemo`/`useEffect` 更少出错
- Naive UI/Element Plus 提供大量开箱即用的金融 UI 组件
- 如果 vue-flow 不够用，切回 React 的成本可控（API 风格相似）

---

## 11. 风险与缓解

| 风险 | 概率 | 缓解措施 |
|------|------|----------|
| Wails v3 不稳定 | 中 | 退回 Wails v2 (已稳定 3 年+) |
| vue-flow 大工作流性能 | 低 | vue-flow 与 React 版共享核心算法，已验证 500+ 节点 |
| 自研 DockView 复杂度高 | 中 | 渐进实现: v1 固定 2x2 布局 → v2 拖拽 → v3 撕下窗口 |
| Go 金融计算库不完善 | 中 | 计算密集走 Python gRPC，逐步补全 Go 原生库 |
| Python sidecar 打包体积大 | 高 | 精简依赖；可选安装（核心功能纯 Go） |
| 多市场券商接口迁移 | 高 | 分 3 批: ①富途(打通 A+H+美) + Binance(加密) ②IBKR+Alpaca(美股) ③OKX+长桥+更多 |
| 100+ 数据连接器迁移 | 高 | 首批 10 个覆盖 90% 场景；统一 Adapter 接口 |
| SQLite 性能上限 | 低 | WAL 模式下千万行 <10ms；极端场景可选 DuckDB |

---

> **本文档是 QuantFlow Terminal 的顶层设计方案。各模块的详细设计将在实现阶段展开。**

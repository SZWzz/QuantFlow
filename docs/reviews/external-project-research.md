# 外部项目调研报告：QuantFlow 可复用资产

> **日期**: 2026-06-27 | **调研对象**: 某终端项目 / 某 A 股项目 / 某 TDX 项目

---

## 一、调研摘要

| 项目 | 语言 | 定位 | 核心资产 | 协议 |
|------|------|------|---------|:--:|
| **某终端项目** | C++20/Qt6+Python3.11 | 彭博式机构金融工作站 | 100+ 数据源、期权定价、组合优化、37+ AI Agent、DataHub 架构 | AGPL-3.0 |
| **某 A 股项目** | Go+Python+Next.js | AI 驱动 A 股量化工作流 | A股引擎、ST 风险预测、北向资金、450+ 因子 Zoo、因子 GP 进化 | MIT |
| **某 TDX 项目** | Python 3.10+ | 通达信协议完整逆向实现 | 4 套 TDX 协议、codec 层 700 行可翻译 Go、离线数据解析 | MIT |

## 二、QuantFlow 现状对比

### 已有能力（不重复引入）

| 能力 | QuantFlow 现有 | 某终端项目 | 某 A 股项目 | 某 TDX 项目 |
|------|:--:|:--:|:--:|:--:|
| A股 K 线 10 源回退链 | ✅ registry.go | ✅ 20+ akshare 脚本 | ✅ 7 源 FALLBACK_CHAINS | ✅ TDX 私有协议 |
| 因子系统 | ✅ 25 因子 | ✅ | ✅ 450+ Alpha Zoo | ✅ 34 技术指标 |
| OMS/T+1/涨跌停 | ✅ 已修复 | ✅ 印度券商 | ✅ 完整 A 股引擎 | — |
| 回测引擎 | ✅ CN/US/Runner | ✅ 6 引擎 | ✅ Pipeline 统一 | ✅ 内置回测 |
| gRPC Python sidecar | ✅ | ❌ QProcess 桥 | ✅ gRPC | — |
| 工作流编排 | ✅ vue-flow | ✅ 节点编辑器+MCP | ✅ 58 节点 | — |
| LLM 多 provider | ✅ 4 个 | ✅ 37+ AI Agent | ✅ ReAct Agent | — |
| 港股引擎 | ✅ 刚创建 stub | ✅ | ✅ | ✅ |
| 美股引擎 | ✅ engine_us.go | ✅ | ✅ | ✅ |
| 加密 | ✅ 现货 (Binance) | ✅ 现货+合约 | ✅ | — |
| 期权定价 | ❌ | ✅ BSM+二叉树+Greeks | ❌ | — |
| 组合优化 | ❌ | ✅ 6 策略+BL | ❌ | — |
| ST 风险预测 | ❌ 无 | ❌ 无 | ✅ 4R+3E 框架 | ❌ 无 |
| 北向资金 | ❌ 无 | ❌ 无 | ✅ Go adapter | ❌ 无 |
| iWenCai 选股 | ❌ 无 | ❌ 无 | ✅ Go adapter | ❌ 无 |
| TDX 协议直连 | ❌ mootdx Python | ❌ 无 | ✅ mootdx | ✅ 4 套协议 |
| 板块/资金流 | ❌ 无 | ✅ akshare | ✅ MacClient 协议 | ✅ 完整 |
| 财务数据 | ❌ 基础 | ✅ 完整 | ✅ | ✅ gpcw.dat |

### 关键结论

> **三个项目中，某 A 股项目 与 QuantFlow 重叠度最高**（同为 Go+Python+gRPC），可直接复用 Go 适配器和 A 股特定模块。**某 TDX 项目 提供 QuantFlow 最缺失的 TDX 协议直连能力**，其 codec 层可直接翻译为 Go。**某终端项目 提供期权/组合优化/DataHub 架构等高级功能**，适合中长期纳入。

---

## 三、可直接复用资产（按优先级）

### 🔴 P0 — 立即移植（填补关键缺口）

#### 1. A 股数据源增强：新增 mootdx + northbound + iWenCai 适配器

**来源**: 某 A 股项目 + 某 TDX 项目

| 适配器 | 来源 | 文件 | 移植成本 |
|--------|------|------|:--:|
| **mootdx (Go)** | 某 TDX 项目 codec 翻译 | `internal/market/adapters/tdx.go` | 3h（700 行 Python→Go） |
| **北向资金** | 某 A 股项目 | `internal/market/adapters/northbound.go` | 30m（Python→Go 翻译） |
| **iWenCai 选股** | 某 A 股项目 | `internal/market/adapters/iwencai.go` | 30m（直接复用） |

**优先级理由**: 北向资金是 A 股独有的核心信号，iWenCai 提供自然语言选股，TDX 协议直连是替代 mootdx Python 的最优方案。

#### 2. ST/*ST 风险预测模块

**来源**: 某 A 股项目 `services/python/src/skills/ashare-pre-st-filter/`

**移植方案**: 完整复用 SKILL.md 中的规则知识图谱，用 Python sidecar 实现为 `python/src/research/st_risk.py`。

**核心规则**:
- R1: 营收+净利润红线（主板 3 亿 / 科创创业 1 亿 / 北交所 5000 万）
- R2: 年末净资产红线
- R3: 分红达标前瞻（主板 5000 万 / 科创创业 3000 万）
- R4: 连续亏损/扣非亏损链
- E1: 审计意见（无法表示/否定/保留）
- E2: 监管处罚（双窗口+主体加权）
- E3: 交易类临界（1 元退市预警）

**移植成本**: 4h（规则引擎 ~300 行 Python + Go gRPC 接口）

#### 3. A 股新闻聚合

**来源**: 某 A 股项目 `backtest/loaders/news_sources/`

| 新闻源 | 文件 | 用途 |
|--------|------|------|
| EastMoney 新闻 | `eastmoney_stock.py` | 个股新闻 |
| 巨潮资讯公告 | `cninfo.py` | 上市公司公告 |
| 财联社电报 | `cls_telegraph.py` | 实时快讯 |

**移植方案**: 这三个文件总计不到 800 行 Python，可直接放入 `python/src/data/news/` 目录，通过 gRPC 对接前端 NewsPanel。

**移植成本**: 1h

---

### 🟡 P1 — 本轮移植（增强功能和数据丰富度）

#### 4. Alpha Zoo 因子库补充（450+ 因子）

**来源**: 某 A 股项目 `services/python/src/factors/zoo/`

QuantFlow 现有 25 个因子，某 A 股项目 有 450+ 个。可选择性引入 A 股特定因子：

| 因子集 | 数量 | 来源 | 移植价值 |
|--------|:--:|------|---------|
| **Alpha101** | 101 个 | WorldQuant 2015 论文 | 全球通用，非 A 股特定 |
| **GTJA191** | 191 个 | 国泰君安 2017 研报 | **A 股专项**，含 A 股常用因子 |
| **Qlib158** | 158 个 | 微软 Qlib | 全球通用 |

**移植方案**: GTJA191（A 股最相关）+ 精选 Alpha101/Qlib158 中前 50 个高 IC 因子。总计引入 ~150 个因子。

**移植成本**: 4h（复制 Python 文件 + 注册 + 测试；无新代码，纯搬运）

#### 5. 期权定价模块

**来源**: 某终端项目 `scripts/Analytics/derivatives/options.py`

QuantFlow 目前完全没有期权定价能力。这是最直接的填补。

| 功能 | 来源文件 | 方法 |
|------|---------|------|
| Black-Scholes-Merton | options.py | `black_scholes_merton()` |
| 二叉树（欧式+美式） | options.py | `binomial_tree_european()` / `binomial_tree_american()` |
| 一阶 Greeks | options.py | `delta()`, `gamma()`, `theta()`, `vega()`, `rho()` |
| 二阶 Greeks | options.py | `vanna()`, `volga()`, `charm()`, `vomma()` |
| 隐含波动率 | options.py | `implied_volatility()` (Brent 求根法) |
| 看涨看跌平价 | options.py | `put_call_parity()` |

**移植方案**: 完整搬运 `options.py` (~268 行) 到 `python/src/analytics/options.py`。纯 numpy/scipy，无额外依赖。

**移植成本**: 1h

#### 6. DataHub 架构模式

**来源**: 某终端项目 `src/datahub/`

概念而非代码的可移植性。QuantFlow 目前无进程内 pub/sub 机制。架构灵感：

- **TopicPolicy**: 按主题配置 TTL / 最小刷新间隔 / 请求合并窗口
- **Producer**: 声明主题、实现 refresh()
- **请求合并**: 100ms 窗口内同主题请求只发一次

**移植方案**: 用 Go 的 `sync.Map` + `time.Ticker` 实现轻量 DataHub（~200 行）。供面板间行情共享使用。

**移植成本**: 2h

---

### 🟢 P2 — 后续纳入（中长期功能）

#### 7. 组合优化模块

**来源**: 某终端项目 `scripts/Analytics/portfolioManagement/portfolio_optimization.py`

6 种优化策略: Max Sharpe / Min Volatility / Efficient Risk / Max Quadratic Utility / Risk Parity / Black-Litterman

**移植方案**: 搬运到 `python/src/analytics/portfolio_opt.py`

#### 8. 量化分析工具集

**来源**: 某终端项目 `scripts/Analytics/quant/quant_modules_3042.py` (1278 行)

时间序列（ADF/KPSS/ARIMA）、ML（Ridge/Lasso/ElasticNet）、采样（Bootstrap/jackknife）

#### 9. C++ 交易类型系统参考

**来源**: 某终端项目 `src/trading/TradingTypes.h` (889 行)

完整的交易域模型结构体定义，可作为 QuantFlow Go 类型系统的设计参考（不搬运代码，参考概念）。

---

## 四、复用决策矩阵

| 资产 | 来源 | 类型 | 成本 | 优先级 | 理由 |
|------|------|------|:--:|:--:|------|
| TDX 协议 Go 适配器 | 某 TDX 项目 | 代码翻译 | 3h | **P0** | 填补 A 股最低延迟数据源缺口 |
| 北向资金适配器 | 某 A 股项目 | 代码复用 | 30m | **P0** | A 股独有信号，Go adapter 直接可用 |
| iWenCai 适配器 | 某 A 股项目 | 代码复用 | 30m | **P0** | 自然语言选股，Go adapter 直接可用 |
| ST 风险预测 | 某 A 股项目 | 规则翻译 | 4h | **P0** | A 股退市预警关键功能 |
| A 股新闻源 | 某 A 股项目 | 代码搬运 | 1h | **P0** | 补齐信息流 |
| Alpha Zoo GTJA191 | 某 A 股项目 | 代码搬运 | 4h | **P1** | 因子库扩充 10 倍 |
| 期权定价 | 某终端项目 | 代码搬运 | 1h | **P1** | 从零补全新品类 |
| DataHub 架构 | 某终端项目 | 概念复用 | 2h | **P1** | 进程内行情总线 |
| Qlib 集成 | 某终端项目 | 代码搬运 | 8h | **P2** | ML 训练框架 |
| 组合优化 | 某终端项目 | 代码搬运 | 2h | **P2** | 补全分析能力 |
| 量化工具集 | 某终端项目 | 代码搬运 | 3h | **P2** | 增厚分析工具箱 |
| 某 TDX 项目 技术指标 | 某 TDX 项目 | 代码搬运 | 1h | **P2** | 34 个指标可选引入 |

---

## 五、推荐执行路线

```
Phase A (本周): P0 移植
├── 某 TDX 项目 codec 层翻译为 Go → tdx.go (3h)
├── northbound.go + iwencai.go 复用 (1h)
├── ST 风险预测模块 (4h)
└── 新闻聚合 (1h)
    ── 总计: 9h, 填补 5 个 A 股关键缺口

Phase B (下周): P1 移植  
├── Alpha Zoo GTJA191 扩展 (4h)
├── 期权定价模块 (1h)
└── DataHub 行情总线 (2h)
    ── 总计: 7h

Phase C (后续): P2 移植
├── Qlib 集成 (8h)
├── 组合优化 (2h)
└── 量化工具集 (3h)
    ── 总计: 13h
```

---

## 六、风险与注意事项

1. **协议合法性**: TDX 协议为私有二进制协议，反向工程用于个人学习合法，但商业化使用有法律风险。AGPL 开源项目慎用。
2. **依赖膨胀**: GTJA191 因子集引入大量 numpy 计算，需评估 Python sidecar 内存占用。
3. **维护成本**: 引入外部代码 = 承担维护责任。某 A 股项目 的 Go 适配器代码质量过硬（含测试），某终端项目 的 Python 脚本较成熟。
4. **AGPL 兼容性**: 某终端项目 也是 AGPL-3.0，代码搬运可直接复用。某 A 股项目/某 TDX 项目 都是 MIT，兼容 AGPL。

*报告完*

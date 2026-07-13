# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。
格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [2026.7.13] - 2026-07-13

### 新增

- [Market] 港股分时双通道 — AKShare Python sidecar 通道（免费，`ak.stock_hk_hist_min_em` 东财数据源）和 QOS API 通道（付费，需配置 key）；`MinuteChains["HK"]` 改为 `["akshare_hk_minute", "qos", "yahoo"]`，Yahoo 作最后回退；两个适配器均已注册并实现 3s cooldown
- [Python] AKShare 港股分时数据 — 新增 `_fetch_akshare_hk_minute` 函数，通过 `stock_hk_hist_min_em` 获取当日分钟线并缓存 60s；gRPC `data_type="hk_minute"` 分支
- [Frontend] QOS API Key 配置 — SettingsPanel API 密钥区新增 QOS 输入框，通过 CredentialManager 加密存储

### 变更

- [Market] HK 分钟链从 `["yahoo"]` 改为 `["akshare_hk_minute", "qos", "yahoo"]` — AKShare 免费通道优先，QOS 付费备用，Yahoo 兜底

---

## [2026.7.12] - 2026-07-12

### 新增

- [Frontend] 港股美股搜索联想 — 嵌入 50 只港股蓝筹 + 50 只美股龙头作为兜底股票列表，EastMoney API 失败时自动回退；`SymbolSearchService` 启动时始终合并嵌入列表，不受 SQLite 缓存影响
- [Market] 港股美股分时数据 — 新增 `MinuteChains["HK"]={"stock":["yahoo"]}` 和 `MinuteChains["US"]={"stock":["yahoo"]}`；Yahoo 适配器新增 `FetchMinuteLine` 方法，通过 `interval=1m&range=1d` 获取当日分钟线；非交易时段自动回退到 SQLite 缓存的最近交易日分时
- [Market] 港股美股 K 线数据 — `MarketForSymbol` 和前端 `detectMarket` 均支持 1-5 位纯数字识别为港股；Yahoo 适配器修复 HK 代码转换（`TrimPrefix` 替代 `TrimLeft`，00700→0700.HK 而非 700.HK）；跳过无效的 crumb 认证请求
- [Frontend] 财务报表面板全面重做 — 新增 ECharts 趋势折线图（营业收入柱状+净利润折线）；KPI 摘要卡片（最新值+同比变化）；自动计算关键财务比率（毛利率、净利率、资产负债率、权益乘数）；年报/季报切换；修复科目顺序（Sina API 返回正确顺序，Go 侧 `formatFinPeriodsRaw` 改用有序数组替代无序 map）；过滤全零科目；固定 170px 卡片宽度防止拉伸
- [Frontend] 港股财务报表接入 — `GetHKFinancialStatements` 通过 AKShare EastMoney 端点获取利润表/资产负债表/现金流量表；Python 侧 `get_stock_financial_hk_report_em` 直接返回标准 `[{report_date, items}]` 格式；自动检测市场（A股/港股/美股）
- [Frontend] MCP 服务器 — 独立 `cmd/mcp` 二进制，通过 stdio JSON-RPC 暴露 QuantFlow 能力；`internal/mcp/` 包实现传输层+处理器+服务循环；节点→能力适配器（25 个精选工作流节点注册为 LLM 可调用工具）；Python 技能注册表（15 个技能支持搜索和 LLM 工具格式导出）
- [Frontend] E2E 测试基础设施 — Playwright + Vite 集成；`e2e/fixtures/mock-app.ts` 注入完整 Go 后端 mock（30+ 方法覆盖行情/交易/回测）；15 个测试用例覆盖看盘/下单/复盘三条核心流程；`data-testid` 属性加入 4 个核心面板；CI 新增 e2e job
- [Frontend] 前端性能优化 — `vendor-markdown` 手动分块（marked+highlight.js 从 AIChatPanel 分离，993KB→5KB）；面板异步加载时显示 SkeletonPanel 骨架屏；`requestIdleCallback` 预加载 echarts；`build:analyze` 脚本集成 rollup-plugin-visualizer

### 变更

- [Frontend] 面板合并 — 10 个重叠集群合并，面板从 91 减至 72（-21%）：订单/成交（5→2：ExecutionPanel+OrderBlotter→TradeHistory，ActionCenter→TradingJournal）；异动扫描（3→1：LimitUpDown+AbnormalStocks+DragonTiger→MarketScannerPanel）；研究子面板（4→删除：Sentiment/AnalystEst./InsiderTrading/PeerComparison 已嵌入 StockResearchPanel）；持仓/组合（5→3：Risk+Equity→PortfolioSummary 标签页，PositionDetail→PositionPanel 内联详情）；市场统一（IPO 日历/相关性/期权链/财务报表/估值预测 各合并一对）；ModelRegistryPanel 重定向到 SettingsPanel；WelcomePanel 过滤 12 个隐藏面板 ID
- [Frontend] 测试文件统一 — stores 测试从紧邻源文件移入 `stores/__tests__/`，与项目其他目录一致
- [Trading] 下单流程修复 — `PlaceOrder` 新增 `brokerName` 参数支持真实券商路由；`OrderEntryPanel` 自动检测市场（CN/HK/US/Crypto）；`BrokerStatusPanel` 动态探测 4 个券商连接状态；新增 IBKR/Alpaca 券商选项；删除死胡同 `RunBacktest` API
- [Frontend] 删除顶部 CN/HK/US 市场切换按钮 — 全自动检测替代手动选择
- [Frontend] 3 个 Mock 面板接线 — BrokerConfig 接入凭证存储；BasketOrderPanel 接入下单 API；FactorAnalysisPanel 接入因子列表（ChanlunPanel 和 StockScannerPanel 此前已接线）
- [Python] `stock_financial_hk_report_em` wrapper 适配新版 AKShare API（`indicator` 参数替代 `date`）；数据直接在 Python 侧转换为标准格式
- [Market] 港股分时链从 Tencent（不支持，返回 302）改为 Yahoo

### 修复

- [Frontend] `FinancialsPanel` Rollup 生产构建 TDZ 错误 — 消除 `nonUSStatements` 中间 computed，CN/港股共用同一 `statements` ref
- [Frontend] `FetchData` 缓存键缺失参数 — 加入 `params` 避免不同语句类型共享缓存
- [Python] 修复 pytest 测试收集（0→162 可发现，56 通过）— 新增 `conftest.py` 添加 `src/` 到 `sys.path`
- [Market] GDELT 测试 HTTP 429 限流 — 新增 `skipIfRateLimited` 辅助函数自动跳过
- [Market] `NormalizeInterval` 产出大写 `1D` 与 Yahoo API 要求小写 `1d` 冲突 — Yahoo 适配器内部转小写
- [Frontend] 实现状态文档更新 — 面板数 22→72（合并后），节点数 54→196
- [Frontend] 行情适配器修复 — `toTencentCode` 裸码识别为深圳（而非港股），`detectMarket` 裸码识别为美股（而非港股）

### 移除

- [Frontend] 顶部 CN/HK/US 市场切换按钮
- [Frontend] 19 个面板文件（合并删除）+ ModelRegistryPanel 孤立文件
- [Trading] 死胡同 `RunBacktest` API（已统一到工作流回测路径）

---

## [2026.7.11] - 2026-07-11

### 变更
- [Python] Fincept 模块调度重构：将 19 种 AKShare 数据类型和 3 个宏观数据源
  (BIS/WTO/EIA) 的 subprocess.run() 替换为直接 importlib.import_module()。
  每次请求节省约 200ms，启用原生 async/await。
  统一的 _call_fincept_module() 调度同时支持 ENDPOINTS-dict 和 Wrapper-class
  两种模块模式。
- [Python] 为 macro_bis.py、macro_wto.py 添加 call_endpoint_async()；
  为 macro_eia.py 添加 call_endpoint() 作为编程式直接导入入口。
- [Python] 从 _handle_macro() 和 _handle_akshare() gRPC 处理器中移除
  subprocess 回退代码——清除 120 行无用的环境变量/工作目录/超时胶水代码。

### 修复
- [Python] Macro BIS 获取命令 SDMX 展平：将 main() CLI 中的内联 JSON 解析逻辑
  移植到 call_endpoint_async()，使 BIS 数据流无需 subprocess 即可工作。
- [Backtest] SquareRootSlippage 重命名为 QuadraticSlippage——实际公式为
  二次冲击 (Base × (1+impact²))，原名与实现不符，误导回测滑点预期
- [Backtest] PDT 日交易窗口改为交易日计数：原 AddDate(0,0,-5) 使用日历日，
  周末/假期会错误触发或漏过 PDT 限制，改为基于回测日期数组的 5 个交易日窗口
- [Workflow] 注销 19 个 stub 指标节点 (KDJ/DMI/ATR/WR/CCI/BIAS/OBV/MFI
  /SAR/VWAP/AROON/ASI/BRAR/MASS/PSY/ROC/BBI)——节点代码保留，接通 Python
  gRPC 后恢复注册
- [Frontend] NodePalette 测试修复：createI18n 添加 legacy: false 匹配项目
  Composition API 模式，消除 $t proxy 冲突
- [Trading] Wash sale 亏损计算修复：原公式用卖出价-回购价，
  改为成本价-卖出价 (IRS Rule 1091 正确语义)，新增成本基础跟踪
- [Backtest] 印花税四舍五入到分 (0.01 CNY)，使用 math.Round 精确计算
- [Backtest] 美股默认交易量改为 1 股 (原 100 股)，匹配零股交易现实
- [Backtest] Sharpe/Sortino 无风险利率可配置：Config 新增 RiskFreeRate
  字段 (默认 0.02)，ComputeMetrics 接受参数化利率
- [Market] Hub subscriber 重构：新增 subscriber struct 封装 channel +
  atomic closed flag，消除 unsubscribe 后的 send-on-closed-channel 竞态
- [Workflow] ExecutionQueue 使用 sync.Cond 替换轮询：新增
  WaitForCompletion 方法，状态变更时 Broadcast 唤醒等待者
- [Workflow] 工具函数集中化：创建 nodes/utils.go 统一收纳
  extractFloatSlice/extractFloat64Slice/getStringParam/getFloatParam/
  getIntParam，消除 macd.go 的 56 处跨文件依赖和 floatutil.go 重复定义
- [Market] QuotePoller WS 降级集成：新增 WSCoverageChecker 接口 +
  SetWSCoverageChecker 方法，WS 活跃时自动跳过 HTTP 轮询（仅加密市场），
  WS 断开后 QuotePoller 自动恢复轮询
- [Broker] Alpaca 增强：AlpacaConfig 新增 Environment 字段 (paper/live)；
  NewAlpacaBroker 根据 Environment 自动切换 BaseURL；新增 IsPaper 方法；
- [CI] 更新 go-version 至 1.25，新增 --bail=5
- [Market] EastMoney IPv6 修复，OHLCV 渐进加载，指数分钟线 prevClose 修复，VADER 离线打包
- [Market] OHLCV 数据范围扩展、缓存接线、CN 适配器链优化
- [Frontend] CandlestickPanel 使用 useMinuteChart composable
- [Frontend] MarketOverviewPanel 使用 useMinuteChart composable
- [Frontend] MarketOverview 交易时段每 30s 自动刷新
- [Frontend] isTradingHours 提取为共享工具函数
- [Storage] 回测结果历史持久化
- [Docs] 性能优化设计方案和实施计划

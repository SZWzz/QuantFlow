# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [2026.7.1] - 2026-07-01

### Added
- [Backend] **FetchData TTL 缓存** — Go 侧 `fetchDataCache` 缓存 Python sidecar 数据请求结果，macro_cn_summary 30min 过期，其他 akshare/ccxt/sec 端点 10min，mootdx 实时行情不缓存
- [Frontend] **K 线增量轮询** — 30s 定时器仅拉取增量数据而非全量 250-450 根 K 线，合并去重（loadSeq 竞态守卫）
- [Frontend] **数据加载错误提示** — CandlestickPanel 内联 err-toast（8s 自动消失）
- [Frontend] **typed Wails bridge** — `useWailsApp()` 类型化桥接，覆盖 FetchOHLCV/GetMinuteLine/GetMultiDayMinute/GetAuditFindings/GetFinancialAnalysis/GetDelistingRisk
- [Frontend] **createIndicatorCache** — 指标计算 memoization 工厂（getCached/clear/delete），2 单元测试通过

### Changed
- [Frontend] **CandlestickPanel 3 阶段重构** — 756 行 → ~383 行脚本，拆分为 KlineChart 组件(稳定 key)、buildChartOption 纯函数 3 个、useWailsApp 类型化桥接
- [Frontend] **useChartTheme 缓存化** — MutationObserver 单次订阅 + 模块级 globalTheme 缓存，避免每次 computed 评估触发 7 次 getComputedStyle

### Fixed
- [Python] **宏观经济硬编码路径** — `fetcher.py:_handle_akshare` 中 `cwd` 写死了旧项目路径 `/Volumes/etx/coding/rebuild/quantflow/python`，改为从 `__file__` 动态推导，移动项目或打包后路径变化不再报错
- [Backend] **多日分时 TDX MAC 服务器单点故障** — MacAdapter 从单地址改为 7 个已知 TDX 服务器轮询（119.147.212.81/168, 115.238.56.58, 123.125.104.230, 180.153.18.170, 61.152.107.247, 124.74.236.65），连接超时自动 fallback 下一地址
- [Frontend] **多日分时数据解析错误** — `GetMultiDayMinute` 返回 `{symbol, days[]}` 结构，前端之前直接当数组处理导致 `days` 字段未解出，数据始终为空
- [Frontend] **KDJ 改为递归加权平均（首值种子 50）** — `kdj()` 之前用 `sma()` 导致第一个 RSV 后还需等 6 个周期才有 K/D/J（分时 09:42 才出图），现改为通达信/同花顺风格递归平滑 `K[i]=2/3×K[i-1]+1/3×RSV[i]` 且 K/D 首值=50，09:30 RSV 出值即绘制三线
- [Frontend] **AuditPanel 营收增长率改为同比（同周期类型）** — `growthRate()` 之前固定取 `periods[0]` vs `periods[3]`（混合不同周期类型如 Q1 vs 半年报），现在改为提取最新期的 month-day（如 `"03-31"`）在历史中匹配相同 month-day 的期进行对比，实现真正同比
- [Python] **analyzer 营收评分改为同比比较** — `analyze_report()` 同期比较逻辑从固定 `revs[0] vs revs[3]` 改为按 month-day 匹配同周期类型，修正之前混合 Q1/半年报/年报带来的评分偏差
- [Frontend] **CandlestickPanel 分时图刷新闪烁** — 轮询刷新时因 `minuteLoading` 翻转导致 loading 层替换 VChart（DOM 卸载→重建），改为仅在首次加载（`!minuteTicks.length`）时显示 loading，后续轮询保持 VChart 持续渲染。同时移除 `minuteBottomMode` 在 `:key` 中的依赖，避免切换副图指标时全量销毁/重建 ECharts 实例
- [Frontend] **MACD 改用首值种子 EMA（同花顺 MACDFS 风格）** — `ema()` 从 SMA 种子（等 N 个值才有第一个值）改为首值种子（第 1 个值就初始化 EMA），分时图 MACD 从 09:30 开盘即可绘制 DIF/DEA/HIST，不再需要等到 10:03。同步加回 `BAR=2×(DIFF-DEA)` 倍数
- [Frontend] **分时图轮询 10s → 5s** — 提高实时性，配合后端 3s 冷却锁避免 mootdx 空转
- [Backend] **MootdxAdapter per-symbol 冷却锁** — `FetchMinuteLine` 新增 `sync.Mutex` + `map[string]time.Time` 冷却锁，同一标的两次调用间隔 < 3s 时直接返回空，避免交易时段轮询空转时重复请求 TDX 服务器
- [Python] **缠论分析输出格式** — 重写 `chanlun.py` CLI 数据序列化，fractals/bi_list/zs_list 输出格式现在匹配前端 `Fractal`/`BiSegment`/`ZSBlock` TypeScript 接口。之前因字段名不匹配（如 `fx_type` vs `type`/`start_date` vs `from_date`）导致前端渲染为空
- [Backend] **pythonDir 路径传递** — `BridgeOptions` 新增 `PythonDir` 字段，`app.go` 启动时将正确的 `python/` 目录路径传递给 `PythonBridge`，修复 `AnalyzeChanlun` 子进程因路径错误找不到 Python 模块的问题

### Changed
- [Frontend] **FinancialsPanel** — 完全重写：移除评分/异常列（移至 AuditPanel），改为三 Tab（利润表/资产负债表/现金流量表）展示 Sina 原始财务数据，行=科目名，列=报告期，数值自动格式化亿/万
- [Backend] **GetFinancialStatements** — 新增 Wails 方法，直接返回 Sina 原始三表数据（不经过 Python analyzer 预处理）
- [Frontend] **AuditPanel** — 完全重写：新增风险仪表盘 + 财务健康双 gauge、KPI 指标卡片(ROE/负债率/净利率/毛利率/营收增长)、评分明细展开、异常发现分类卡片、12 期财务历史表格；并行调用 GetAuditFindings + GetFinancialAnalysis 合并显示
- [Frontend] **MarketOverviewPanel** — 全面性能优化：Promise.all 并行化 GetMarketOverview 和 GetIndustryRanks；IndustryRanks 前端缓存 5 分钟；骨架屏替代全屏 loading 遮罩；使用 PanelHeader/PanelTable 共享组件
- [Frontend] **GovDataPanel** — 迁移至 PanelHeader/PanelTable/PanelToolbar 共享组件，统一加载/空态/错误显示
- [Frontend] **IndicatorPanel** — 迁移至 PanelHeader/PanelTable 共享组件，硬编码颜色替换为 CSS 变量
- [Frontend] **StockScannerPanel** — 迁移至 PanelHeader/PanelTable 共享组件，CSS 变量颜色统一
- [Frontend] **LimitUpDownPanel** — 迁移至 PanelHeader/PanelTable 共享组件，并行数据加载
- [Frontend] **WatchlistPanel** — 迁移至 PanelHeader/PanelTable 共享组件，CSS 变量颜色统一
- [Frontend] **~50 面板** — 硬编码颜色（`#ef4444`/`#22c55e`/`#3b82f6` 等）全部替换为 CSS 变量 `var(--color-up)`/`var(--color-down)`/`var(--color-accent)`，主题切换后全面板生效
- [Frontend] **ModelRegistryPanel** — 硬编码颜色替换为 CSS 变量
- [Backend] **GetIndustryRanks** — 移除重试逻辑（原最多 3 次），一次失败立即返回空切片，避免阻塞面板 5-15 秒
- [Backend] **GetMarketOverview** — 缓存 TTL 30s → 60s，减少重复请求
- [Backend] **MacAdapter** — 新增 TCP 连接池复用（sync.Mutex + 持久连接），断线自动重连，消除每次 GetBlockRank 新建 TCP 的开销
- [Frontend] **symbolSearch** — 搜索结果上限 20 条，Go 后端失败时优雅回退到前端 mock 搜索
- [Build] **Makefile** — 新增 `build-full` 目标（frontend build → Go 编译 → Python sidecar rsync）
- [Build] **Taskfile** — Python venv 符号链接在 rsync 后创建，避免 --delete 误删

### Added
- [Frontend] **AuditPanel 退市风险 tab** — 新增 A 股/港股/美股退市风险检测，覆盖财务类(营收+净利润组合/净资产)、交易类(面值/总市值)、规范类、重大违法类，纯 Go 规则引擎无 Python 依赖
- [Backend] **GetDelistingRisk** — 新增 Wails 方法按市场路由(A/HK/US)返回退市风险结构化数据，含板块检测、ST 状态、逐项指标状态灯
- [Backend] **internal/trading/delisting_risk** — 退市规则引擎包：ExtractFinancialMetrics 解析 Sina 财务 JSON，AssessCN/HK/US 三市场规则实现
- [Frontend] **usePanelCache** — 新增通用面板缓存 composable，基于 dataStore 的 TTL 缓存，统一面板数据缓存逻辑
- [Frontend] **TickerBar** — 新增全局滚动行情栏组件，展示 CN/HK/US 市场实时报价
- [Frontend] **ForecastPanel** — 新增 ECharts 分组柱状图：X 轴三情景(保守/基准/乐观)，每组基准年/Y1/Y2 营收对比
- [Frontend] **ModelRegistryPanel** — 重写为大模型配置面板：Provider 卡片(OpenAI/Anthropic/DeepSeek/Ollama)、API Key + Base URL 编辑、连接测试、模型列表浏览
- [Backend] **ListLLMModels** — 新增 Wails 方法调用 Python gRPC `ListModels`，返回可用 LLM 模型列表
- [Frontend] **Panel 共享组件库** — 新增 10 个标准化组件：PanelHeader（标题/标签/操作栏）、PanelCard（信息卡片）、PanelTable（可排序表格）、PanelTabs、PanelToolbar、LoadingState、EmptyState、SignalBadge、TrendIndicator，统一 50+ 面板的 UI 模式
- [Frontend] **Panel tokens CSS** — themes.css 新增面板设计令牌：panel-padding/title-size/table-row-height/tab-height/toolbar-gap 等，支持 density（紧凑/舒适）预设

## [2026.6.29] - 2026-06-29

### Changed
- [Frontend] **28 panels** — 全部接入 usePanelCache 缓存(5分钟TTL，系统监控5秒)，切换 symbol 时命中缓存避免重复 Go IPC 调用
- [Frontend] **ForecastPanel** — 重写完整面板：代码解压缩、i18n 全覆盖、增加年均利润率/CAGR/基准增长率指标栏; 修复季度累计营收误作年化基数的显示问题
- [Python] **forecast_financials** — 修复：区分年度(12-31)与季度累计(03-31/06-30/09-30)数据，年度数据不足时年化处理，返回 period_type/latest_period 供前端准确展示
- [Frontend] **i18n** — 新增 forecast_* 系列翻译键(zh/en)
- [Frontend] **registry.ts** — 移除 ticker-tape 面板注册，改为全局 TickerBar

### 修复 — 行情详细面板数据缺失
- [行情] **QuoteSnapshot** — 扩展 struct 新增 `PrevClose`/`Turnover`/`MarketCap`/`Pe`/`Exchange` 字段，解决前端显示大量 `--` 的问题
- [行情] **EastMoney 适配器** — 映射 F48(成交额)→Turnover、F116(总市值)→MarketCap、F162(市盈率)→Pe，从 secid 前缀推断 Exchange(SH/SZ)
- [行情] **Sina CN 解析器** — 映射 field[2](昨收)→PrevClose、field[9](成交额 万元→元)→Turnover
- [行情] **Sina HK 解析器** — 映射 field[3](昨收)→PrevClose、field[11](成交额)→Turnover
- [行情] **Tencent/AKShare 解析器** — 映射 fields[4](昨收)→PrevClose、fields[37](成交额)→Turnover
- [行情] **Yahoo 适配器** — 从 prev 日线提取 PrevClose
- [前端] **QuoteDetailPanel** — 新增成交额/总市值/市盈率展示，总市值用万亿/亿/万格式化

### 修复 — 分红除权面板数据全零
- [行情] **ExDividendCalendar** — `CLOSE_PRICE` 不存在于 EastMoney RPT_SHAREBONUS_DET 接口，改用 `DIVIDENT_RATIO * 100` 计算股息率；`TRANSFER_RATIO` 改用 `IT_RATIO`（转增比例），`BONUS_RATIO` 保留原字段名（送股比例）

### 修复 — 分析师预测面板目标价全零/分析师名重复
- [行情] **ResearchReport** — 新增 `Analyst`/`TargetPriceLow`/`TargetPriceHigh` 字段，映射 API 的 `researcher`/`indvAimPriceL`/`indvAimPriceT`
- [行情] **reportItem** — 新增 `Researcher`/`IndvAimPriceL`/`IndvAimPriceT` 字段以捕获 API 完整数据
- [行情] **AnalystEstimatesService** — 修正字段映射：`Analyst` 改用分析师姓名（`r.Analyst`），`TargetLow`/`TargetHigh` 改用目标价（`r.TargetPriceLow`/`r.TargetPriceHigh`），不再使用机构名和EPS预测冒充目标价
- [行情] **AnalystEstimatesService** — 添加 `(org+analyst+rating)` 去重，消除同一机构同评级重复行

### 修复 — 市场概括加载卡顿
- [行情] **GetMarketOverview** — 并行化指数行情请求（goroutine + WaitGroup），5 个指数从串行 ~1.5s 降至 ~300ms
- [行情] **marketOverviewCache** — 新增 30s TTL 结果级缓存，避免 15s 自动刷新周期内重复请求
- [行情] **idxDef** — 提取为包级类型，供 getIndexName 查找函数复用

### 修复 — 市场深度面板无数据
- [行情] **DepthLevel/DepthSnapshot** — 新增 5 档盘口行情类型（`market/types.go`），Bids/Asks 各 5 档含 price/size
- [行情] **Tencent 适配器** — 新增 `FetchDepth` 方法，从 Tencent API 响应中解析 fields[7]~[26] 的 5 档买卖盘数据（字段格式: bid1_price~ask1_price~bid1_vol~ask1_vol~bid2_price~…），vol 统一从 手→股
- [行情] **GetDepth** — 新增 Wails 方法（`app_market.go`），按市场路由: CN/HK → Tencent, CRYPTO → Python CCXT, US → 暂不支持
- [前端] **MarketDepthPanel** — `refresh()` 改为并行调用 `GetQuote` + `GetDepth`，有真实 5 档盘口数据时用实时数据、移除 "模拟盘口" 标签，失败时回退到模拟显示

### 修复 — 情绪分析面板分数异常 (POSITIVE +100.0)
- [Python] **nlp_pipeline** — SnowNLP 置信度不再硬编码 0.7：极端分数 (raw≤0.05或≥0.95) → confidence=0.35，边界 (≤0.15或≥0.85) → 0.5，正常范围 → 0.65；短文本 (<50字) 再打 5 折
- [Python] **nlp_pipeline** — SnowNLP 极端分数向中性拉回：raw≤0.05或≥0.95 时映射到 `0.5 + (raw-0.5)*0.3`，避免 +100 或 -100 的假象
- [Python] **nlp_pipeline** — 关键词过滤数值 token（`_is_numeric_token`），不再显示 "2.78"、"515.32" 等数字垃圾
- [Python] **nlp_pipeline** — 短文本置信度衰减：<50字 *0.5、<100字 *0.8，过滤噪声文本的过度自信

### 修复 — 终端模式股票名称乱码
- [行情] **Sina 适配器** — `decodeGBK` 失败时不再直接返回原始 GBK 字节导致 JSON 序列化产生 U+FFFD 乱码：检查是否已是合法 UTF-8，否则返回错误让 fallback 链切换到下一适配器
- [行情] **Sina CN/HK 解析器** — 新增 `cleanName()` 函数，验证 UTF-8 合法性，非法字节替换为 `?`，避免非 UTF-8 字符串传到前端显示乱码
- [行情] **Sina HK 解析器** — 修复 `Name` 误用 `fields[0]`（英文名）→ 改回 `fields[1]`（中文名）

### 修复 — Tencent 适配器 GBK 编码乱码（经 Sina fix 后 fallback 链切到 Tencent 仍乱码）
- [行情] **Tencent 适配器** — `FetchQuote`/`FetchDepth` 响应体添加 GBK→UTF-8 解码（与 Sina 共用 `decodeGBK`），中文股票名称不再显示乱码
- [行情] **parseTencentQuote (akshare.go)** — Name 字段添加 `cleanName()` UTF-8 验证，双重保障

### 修复 — 分时图不刷新
- [行情] **GetMinuteLine** — 当 `sinceTimestamp > 0`（增量轮询或切换代码时），缓存无新数据后不再返回空，改为执行 mootdx 实时抓取并合并到缓存，使分时图在交易时段内持续更新
- [存储] **minute_cache_test.go** — 修复因硬编码日期 `"2026-06-26"` 与 `time.Now()` 不匹配导致的测试失败（`SaveAndGet`、`LRUFull`）

### 修复 — K线指标预热数据不足
- [前端] **CandlestickPanel** — 日K回溯从 90 天改为 365 天（~250 交易日），周K从 180 天改为 450 天（~64 周），确保 MA60 及所有附图指标有足够预热数据，消除前段空白

### 变更 — README 文档重构
- [Docs] **README.md/README.en.md** — 全面重构：ASCII 架构图 → Mermaid 图、节点/面板/适配器列表转为 `<details>` 可折叠、修复 shields.io badge、精简过时内容、统一中英文排版

### 新增 — 财务报表面板毛利率指标
- [Python] **analyzer.py** — 从利润表提取 `营业成本` 计算 `gross_margin`（毛利率 = (营收 - 营业成本) / 营收 × 100%）
- [前端] **FinancialsPanel** — 新增毛利率列（带 YoY），营收/净利润/归母净利 cell 内嵌 YoY 标签（绿涨红跌），右栏评分区锁定 180px 宽度
- [Go] **sinaToAnalyzer** — 添加 `营业成本` 映射条目

### 修复 — Python sidecar 进程残留导致新代码不生效
- [Engine] **StartSidecar** — 取消版本号匹配复用逻辑，改为每次启动无条件 kill 端口 50051 残留进程后全新启动，确保 Python 代码每次加载最新版本
- [Python] **pyproject.toml** — version 0.2.2 → 0.2.3（配合版本管理，防止残留进程匹配预期版本号）

## [2026.6.28] - 2026-06-28

### 新增 — 财务深度分析面板（Financial Analyzer 集成）
- [Python] **analyzer.py** — 4 个纯计算分析函数（`report_analysis`/`valuation`/`audit`/`forecast`），输入财报 JSON → 输出结构化分析结果
- [Python] **fetcher.py 扩展** — 新增 `analyzer` source 直接 import 调用（避免 gRPC fork 冲突）
- [行情] **财务报表面板重写** — FinancialsPanel 裸 JSON → 三年对比表 + 财务健康评分(0-100) + 异常标记（商誉/负债率/应收）
- [终端] **DCF 估值面板** — ValuationPanel 三情景 DCF + 自由现金流 + 买卖建议
- [终端] **财务审计面板** — AuditPanel 按风险等级分组审计发现（应收/商誉/现金流/负债）
- [终端] **财务预测面板** — ForecastPanel 线性回归两年三情景预测表
- [行情] **Sina 字段映射** — 20+ 字段名从新浪长名称 → 标准化短名称（`归属于母公司所有者的净利润` → `归母净利润`）
- [行情] **股本查询** — `fetchQuoteJSON` 集成 EastMoney `FetchStockInfo` 获取总股本/市值

### 修复 — 面板质量与数据管道
- [前端] **MarginPanel/SECFinancialsPanel/SEC13FPanel** — 裸 JSON → 结构化表格
- [前端] **RiskDashboard** — 移除占位横幅 + 接入 portfolio store 真实持仓/净值计算 6 项风险指标
- [前端] **BasketOrderPanel** — 移除硬编码 mock 行 + `setTimeout` 模拟，改为 Go `PlaceOrder` 真实下单
- [前端] **TradingJournalPanel** — 移除 `buildMockData()` 假数据 fallback
- [前端] **3 个 JSON 面板 UI 错误修复** — ChanlunPanel/IndicatorPanel/StockScannerPanel CSS 使用不存在的 `--term-*` 变量 → 替换为标准 `--color-*` 变量
- [前端] **SectorRotationPanel** — 数据源从 `GetMarketOverview`（无 sectors 字段）改为 `GetIndustryRanks`
- [前端] **WelcomePanel** — 修复 `delete groups['系统']` 导致系统分类 6 面板消失；新增港股/美股/加密货币配色
- [前端] **CandlestickPanel** — VChart key 缺失 indicator state 导致 MA/BB/副图合并渲染残缺；yAxis `scale` 裁剪 BB 轨线；dataZoom 默认只显示 50% 数据
- [前端] **useStockName debounce** — 400ms 防抖 + 最小符号长度校验，终止逐字输入时的 API 风暴

### 新增 — 工作流模式增强（P1-P3）
- [工作流] **Go ListNodes 扩展** — `NodeMeta` 新增 `InputPorts`/`OutputPorts`/`Params` 字段，`ListAll` 创建临时节点实例读取真实端口
- [工作流] **前端端口映射** — 删除 `pmap` 硬编码，改用 Go `ListNodes` 真实端口；连线时类型校验阻止不兼容连接
- [工作流] **7 个预置量化模板** — 金叉选股/MACD底背离/多因子打分/布林带突破/RSI超卖/配对交易/MACD+RSI共振
- [工作流] **节点视觉差异化** — 按类别显示颜色边框 + emoji 图标
- [工作流] **多工作流管理** — localStorage 多文件存储 + WorkflowList 侧边抽屉（新建/加载/重命名/删除）
- [工作流] **Walk-Forward 回测** — `ExecuteBacktest` 滚动窗口训练/测试分离
- [工作流] **参数优化** — `OptimizeParams` 网格搜索 + 按 Sharpe/Return 排序
- [工作流] **Workflow.Clone()** — 深拷贝用于独立窗口执行

### 修复 — Go 后端
- [研究] **InsiderTradingService** — 3 条硬编码 mock → Python SEC EDGAR Form 4 真实数据
- [研究] **PeerComparisonService** — 真实同行名但 MarketCap=0 → EastMoney 填充市值
- [机器学习] **ML 服务层** — `ListMLModels` 文件系统扫描 + `GetPredictions`/`RunAlphaMining`/`AssessRisk` Python gRPC 代理
- [Python] **bridge.go** — 新增 `PythonDir()` getter

### i18n 完善
- [前端] **工作流面板全 i18n** — PropertyPanel/CustomNode/WorkflowMode/NodePalette/WorkflowList 所有硬编码英文 → `$t('workflow.*')`
- [前端] **25 个 P1 面板 SVG 图标** — WelcomePanel 卡片不再缺失图标
- [前端] **SettingsPanel 版本号** — 硬编码 `2026.6.17` → `src/version.ts` 统一管理

### 新增

#### P1 — 终端面板增强（7 个新面板）
- [终端] **龙虎榜面板** (`DragonTigerPanel.vue`) — 日榜单+个股历史，买卖 TOP5 营业部展开，symbol 点击联动
- [终端] **涨跌停监控面板** (`LimitUpDownPanel.vue`) — 从 `GetAbnormalStocks` 过滤涨停/跌停，统计计数，30s 自动刷新
- [终端] **港股通面板** (`HKConnectPanel.vue`) — 北向资金实时流入分时图+历史表格+额度概览，复用 `GetNorthboundFlow`
- [终端] **资金费率面板** (`FundingRatePanel.vue`) — 加密永续合约资金费率+标记价+指数价+下期结算，30s 自动刷新
- [终端] **爆仓追踪面板** (`LiquidationPanel.vue`) — 24h 爆仓统计+历史列表，多/空方向标识
- [终端] **板块轮动面板** (`SectorRotationPanel.vue`) — RRG 散点图 4 象限+板块强度表，CN/HK/US 市场切换
- [终端] **经济日历面板** (`EconomicCalendarPanel.vue`) — 全球宏观事件时间线，前值/预期/实际三级展示，CN/US 过滤

#### P2 — 用户体验增强
- [终端] **SkeletonPanel 骨架屏组件** — table/card/chart 三种类型，统一加载态
- [终端] **ErrorBoundary 面板级错误边界** — 捕获渲染异常，显示重试 UI
- [终端] **全局快捷键** — `Ctrl+Shift+D`龙虎榜 `Ctrl+Shift+L`涨跌停 `Ctrl+Shift+H`港股通 `Ctrl+Shift+F`资金费率 `Ctrl+Shift+Q`板块轮动 `Ctrl+Shift+E`经济日历
- [终端] **WelcomePanel 动态化** — 最近面板列表+市场快照（上证/恒指实时行情）
- [终端] **recentPanels 面板历史追踪** — 最近 20 个面板记录，localStorage 持久化

#### 后端
- [行情] **BinanceFuturesAdapter 扩展** — 新增 `FetchFundingRates` (费率) 和 `FetchLiquidations` (爆仓) 方法，注册到 adapter 注册表
- [行情] **新 App 方法** — `GetCryptoFundingRates` / `GetCryptoLiquidations` 暴露给前端
- [行情] **Finnhub 适配器扩展** — 新增 `FetchShortInterest`（做空数据）和 `FetchEarningsCalendar`（财报日历）
- [行情] **新 App 方法** — `GetShortInterest` / `GetEarningsCalendar` 暴露给前端

### 新增 — 加密补齐 (深度对比 + DeFi TVL + 巨鲸追踪 + Gas)
- [终端] **多交易所深度对比面板** (`DepthComparisonPanel.vue`) — 跨 Binance/OKX/Gate.io 买卖盘深度并排对比
- [终端] **DeFi TVL 排行面板** (`DefiTVLPanel.vue`) — DeFi Llama 协议锁仓量排行榜
- [终端] **巨鲸追踪面板** (`WhaleTrackingPanel.vue`) — Etherscan 大额链上转账监控
- [终端] **Gas 追踪面板** (`GasFeePanel.vue`) — Etherscan Gas Tracker 实时 Gas 价格三档显示
- [Python] **新增 `crypto_extras.py`** — DeFi Llama/Etherscan 数据源封装模块
- [Python] **`fetcher.py` 扩展** — 新增 `crypto_extras` 路由
- [行情] **新 App 方法** — `GetCryptoDepth` / `GetDeFiTVL` / `GetWhaleTransactions` / `GetGasFees`

### 新增 — 美股补缺 (期权链 + Wash Sale + 机构交易)
- [终端] **期权链面板** (`USOptionsPanel.vue`) — 看涨/看跌矩阵，行权价×到期日，Finnhub option-chain API
- [终端] **Wash Sale 面板** (`WashSalePanel.vue`) — IRS 1091 洗售亏损检测，纯 Go 逻辑
- [终端] **机构交易面板** (`DarkPoolPanel.vue`) — SEC 13F/4 文件中的机构/内部人交易活动
- [行情] **Finnhub 扩展** — `FetchOptionChain` + `FetchSECFilings` 方法
- [交易] **Wash Sale 引擎** — `internal/trading/wash_sale.go` 扫描交易历史识别洗售模式
- [行情] **新 App 方法** — `GetUSOptionChain` / `GetSECFilings` / `CheckWashSale`

### 新增 — Panel Cache Layer (已实现, 验收通过)
- [终端] **dataStore 缓存层** — `CacheEntry` 类型 + `setCached`/`getCached`/`clearCached` TTL 方法
- [终端] **HeatmapPanel** — 30s TTL 缓存, 市场切换先读缓存再回退到 fetch
- [终端] **GovDataPanel** — 信号列表 5min 缓存, 指标详情 10min 缓存, 切换源重新获取

### 新增 — 港股补缺 (香港IPO + 牛熊证/涡轮 + 交收规则)
- [终端] **香港IPO日历面板** (`HKIPOPanel.vue`) — 新股认购/即将上市/近期表现，通过 Python AKShare
- [终端] **牛熊证/涡轮面板** (`HKDerivativesPanel.vue`) — 牛熊证+认股证数据，代码点击跳转
- [终端] **港股交收面板** (`HKSettlementPanel.vue`) — T+2 时间线+费用计算器+交易日历
- [行情] **Python HK 模块** — `fincept/hk.py` 封装 `stock_hk_ipo_subscription/record`, `stock_hk_cbbc`, `stock_hk_warrants`, `tool_trade_date_hist`
- [行情] **Go 后端扩展** — `GetHKIPOCalendar`, `GetHKDerivatives`, `GetHKTradingCalendar`, `GetHKSettlementInfo`

### 新增 — A股残局 (IPO 日历 + 分红除权 + 可转债套利)
- [终端] **新股日历面板** (`IPOCalendarPanel.vue`) — 新股发行/申购/上市日历，今日申购/即将上市/近期上市三 tab
- [终端] **分红除权面板** (`ExDividendPanel.vue`) — A 股全市场除权除息日历，今日/本周/本月切换，自动计算股息率
- [终端] **可转债套利面板** (`CBArbitragePanel.vue`) — 集思录溢价率排序+强赎预警+回售机会，负溢价绿色高亮
- [行情] **Go 后端扩展** — `FetchIPOCalendar` (eastmoney_signals)、`FetchExDividendCalendar` (eastmoney_capital 日期范围版)、`GetCBArbitrageData` (app_market 聚合)
- [行情] **Python 扩展** — fetcher.py 新增 `cb_arbitrage` / `cb_redeem` 路由到 bonds.py 集思录端点
- [行情] **新 App 方法** — `GetIPOCalendar` / `GetExDividendCalendar` / `GetCBArbitrageData` 暴露给前端

### 新增 — 第 2 批 P1 面板
- [终端] **交易日志面板** (`TradingJournalPanel.vue`) — 逐日 P&L 归因+盈亏统计+品种归因
- [终端] **做空数据面板** (`ShortInterestPanel.vue`) — 美股做空比例/覆盖天数/趋势判断
- [终端] **跨资产相关性面板** (`CrossAssetCorrelationPanel.vue`) — 6 组预设+热力图矩阵（股票/债券/商品/加密）
- [终端] **财报日历面板** (`EarningsCalendarPanel.vue`) — 美股财报预期/实际/超预期，周/月/季切换
- [终端] **情景分析面板** (`ScenarioAnalysisPanel.vue`) — 5 种压力场景+组合影响模拟

### 修复
- [前端] **SectorRotationPanel 数据源修正** — 板块轮动面板从 `GetMarketOverview`（仅返回指数行情）切换为 `GetIndustryRanks`（返回行业涨跌幅排行），修复 RRG 散点图始终为空的问题
- [前端] **WelcomePanel 分类修复** — 修复 `delete groups['系统']` 导致整个系统分类（6 个面板）不显示的 bug，改为只过滤 welcome 面板；新增 港股/美股/加密货币 分类专属配色

### 审计 — 全面板质量审查 (90 面板)

#### 真实数据 (28 面板)
交易：PositionPanel, OrderBlotterPanel, ExecutionPanel, TradeHistory, PositionDetail, OrderEntryPanel, ActionCenterPanel（均通过 portfolio store → Go backend）
行情：AbnormalStocks, Candlestick, CryptoOverview, DragonTiger, HKConnect, LimitUpDown, FundingRate, Liquidation, MarketDepth, MarketOverview, QuoteDetail, ShortInterest, TickerTape, Watchlist, FuturesPanel
研究：CongressTrading（telep.io 国会交易 API）
系统：AIChat, News, SystemMonitor, Settings
图表：EquityCurvePanel

#### 部分真实 (Go 层 mock fallback) (10 面板)
StockResearch, AnalystEstimates（API 失败 → 5 条硬编码）, PeerComparison（真实同行名但 PE=0）, Sentiment（需 Python sidecar）, InsiderTrading（Go 层纯 stub 3 条假数据）, FinancialsPanel, FundFlowPanel（需 Python mootdx）, OptionsPanel（BSM 前端计算）, SectorRotation（刚修复数据源）, TradingJournal（mock fallback）

#### 纯 Mock/空壳 (7 面板)
RiskDashboard（显式占位横幅）, BasketOrderPanel（setTimeout 模拟执行）, AlphaMiningWorkspace（ML store stub）, ModelRegistry（ML store stub）, PredictionDashboard（ML store stub）, RLMonitor（占位面板）, BacktestResultPanel（仅 empty state）

#### UI 异常 (3 面板)
MarginPanel, SECFinancialsPanel, SEC13FPanel — 使用 `<pre>{{ JSON.stringify(data) }}</pre>` 裸 JSON 输出，无结构化表格/图表

### 修复 — Go 后端 stub 服务真实数据接入
- [后端] **InsiderTradingService 真实化** — 从 3 条硬编码 mock → Python sidecar SEC EDGAR Form 4 真实数据（edgartools 库），构造器新增 `*PythonBridge` 参数
- [后端] **PeerComparisonService MarketCap 填充** — 从概念块取同行后，通过 EastMoney `FetchStockInfo` 批量获取每只股市值，构造器新增 `*EastMoneyAdapter` 参数
- [后端] **ML 服务层实现** — 新增 `app_ml.go`，实现 `ListMLModels`（文件系统扫描 models/ 目录）、`GetPredictions`/`RunAlphaMining`/`AssessRisk`（Python gRPC 代理），前端 ml store 3 个 stub 方法全部接入真实后端
- [前端] **SECFinancialsPanel UI 重构** — 裸 JSON → SEC 财务报表分区展示（BS/IS/CF），数字格式化（B/M/K），负值红色高亮
- [前端] **SEC13FPanel UI 重构** — 裸 JSON → 可排序列表格（13F 持仓），自适应列宽，支持 100+ 行分页
- [前端] **RiskDashboard 真实数据接入** — 移除占位横幅和所有硬编码 mock 值，接入 portfolio store 真实净值曲线+持仓，计算 VaR/CVaR/最大回撤/Sharpe/Sortino/年化波动，回撤图表使用真实 NAV
- [前端] **BasketOrderPanel 真实下单** — 移除 setTimeout 模拟执行和硬编码 mock 行，改为调用 Go PlaceOrder 逐笔下单，支持失败状态显示

- [行情] MAC 协议适配器：通达信 TCP 二进制协议直连，5 数据通道（金钻 VIP 快照、主力大单、板块成分、通用行情、Level-2 十档），零中间件零依赖
- [行情] StockSymbolCache：SQLite 持久化股票名称列表（7 天 TTL），启动跳过 API 加载，闭市时从缓存预填自选股名称
- [前端] 64 面板股票名称展示：Trade/Order/Position 新增 `Name` 字段，quoteCache 自动回填，21 个面板代码旁显示中文名称
- [前端] `useDataFetch` composable：loading/error/data 三态标准模式，覆盖 SystemMonitorPanel 及后续所有面板
- [前端] 分时图增量渲染：MinuteCache SQLite 持久化 + LRU，跨面板共享缓存，增量加载禁动画，filterSince 时区 bug 修复
- [前端] 3 个新面板：FQFactor（因子分析面板）、ChanlunBi/ChanlunDuan/ChanlunZhongshu（缠论可视化面板）、MACProtocol（MAC 数据面板）
- [前端] 周末假日分时图回退：mootdx 周末返回空 → 回退到最近交易日分时数据
- [工作流] 20 个技术指标节点：OBV, MFI, PSY, Aroon, ASI, WR, CCI, ROC, BIAS, Chaikin, Keltner, Donchian, TRIX, MassIndex, Vortex + 原有 5 个
- [工作流] 5 个缠论节点：ChanlunBi（笔）、ChanlunDuan（段）、ChanlunZhongshu（中枢）、ChanlunMaiDian（买卖点）、ChanlunLeixing（走势类型）
- [工作流] 3 个滑点模型节点：FixedSlippage、PercentageSlippage、VolumeSlippage
- [工作流] fqfactor 节点：易量化因子集成
- [引擎] P0 金融正确性：T+1 锁仓、涨跌停价格限制、交易成本扣除、CashLedger 现金流账簿
- [引擎] app.go 按领域拆分：app_market.go、app_trading.go、app_research.go、app_system.go

### 修复

- [前端] 全面消除静默 catch：22 个面板 `catch (_) {}` → 统一 error 处理 + 用户可见提示（Tasks 3-5）
- [前端] 加载状态 & 空态修复：10+ 面板 loading skeleton / empty state（Tasks 6-8）
- [前端] Store 错误状态 + IPC 一致性：5 个 store 错误传播修复（Tasks 9-10）
- [前端] 分时图 `notMerge: false` 强制全量重建 bug → 禁用全量重建防止闪烁
- [前端] 分时图 `filterSince` 时区 bug：时间戳比较未考虑 UTC+8 偏移 → 增量加载断裂
- [后端] NodeContext 集成修复：所有节点构造函数签名对齐 `node.NewBaseNode(id, nodeType)`
- [行情] Mootdx 分钟线周末空数据：判断 `len == 0` 不再 panic，回退最近交易日

- [Docs] 新增 5 篇规范文档：TDX 协议集成、分时图实时渲染、前端质量重塑、股票名称展示、前端质量实施计划

### 优化

- [前端] Pinia `dataStore` 通用 TTL 缓存层：`setCached`/`getCached`/`clearCached`，带自动过期
- [前端] HeatmapPanel 改用 TTL 缓存（30s），切换面板免重复加载；切换市场/手动刷新立即回源
- [前端] GovDataPanel 信号列表缓存 5min、详情数据缓存 10min，切换面板/切换数据源回源

### 修复

- [Python] BIS 子进程 `rc=64`：`_handle_macro` 改用 `sys.executable` 避免 build 目录下 venv 路径解析歧义；显式传入 `PYTHONPATH` 确保模块可寻址

### 新增

- [Python] BIS `get_summary` 命令：并行拉取所有 BIS 数据流的最新值（`lastNObservations=1`）+ 日期（`reportingEnd`），一次性返回用于面板卡片展示
- [前端] GovDataPanel BIS 标签页改用 `get_summary`：卡片直接显示最新数值和日期，替代原来全部显示"点击查看数据"

### 修复

- [Python] 个股资金流向 `get_stock_individual_fund_flow`：增加 A 股代码校验（6位数字），美股/港股直接返回友好错误提示而非 `NoneType` 崩溃
- [Python] `get_fund_report_stock_cninfo` 错误传递 `symbol` 参数至底层 akshare 函数（该函数仅接受 `date`），改为先拉全量再按股票代码过滤
- [前端] OptionsPanel: 路由从 `get_all_endpoints`（端点列表）改为 `option_current_day_sse`（实时 SSE 期权数据），去除 `symbol` 多余传参
- [前端] IndexPanel: 路由从不存在的 `stock_board_index_cons_em` 改为 `index_stock_cons`（指数成分股），移除 `"index"` 符号排除限制
- [前端] FundsPanel: 路由从 `get_all_endpoints` 改为 `get_fund_report_stock_cninfo`（基金持仓覆盖数据），移除 `"funds"` 符号排除限制
- [前端] BondsPanel: 路由从 `get_all_endpoints` 改为 `get_bond_zh_cov_spot`（可转债实时行情）
- [前端] SentimentPanel: `isBridgeAvailable` 因 `app.GetVersion` 不存在而永远为 `null`，改为直接标记桥接可用

### 新增

- [前端] BondsPanel: 搜索过滤 + 可转债行情表（代码/名称/最新价/涨跌幅/成交量/成交额/正股代码/时间）
- [前端] OptionsPanel: 期权链表格（合约代码/名称/标的/认购认沽/行权价/到期日），替代裸 JSON
- [前端] IndexPanel: 指数成分股表格（代码/名称/纳入日期），输入框可切换指数代码
- [前端] FundsPanel: 基金覆盖信息卡片（报告期/基金覆盖家数/持股总数/持股总市值），替代裸 JSON
- [前端] MarketOverviewPanel: 指数卡片点击联动，通过 SymbolContext 设置当前指数代码，IndexPanel 自动加载对应成分股

---

## [2026.6.26] - 2026-06-26

### 新增

- [研究] 真实国会交易数据：接入 `trades.telep.io` Capitol Trades API（免费无需 Key），覆盖 House + Senate 两院 ~10000+ 条 STOCK Act 披露交易。新增 `CongressTradesAdapter`，过滤仅保留 Stock 类型
- [情感] 中文 NLP 情感分析：`detectLanguage()` 自动识别 A 股代码走 `zh` 通道，启用 SnowNLP + jieba 分词。安装 `nltk`/`snownlp`/`jieba` 到 Python venv

### 修复

- [工作流] Go 状态字符串 `"success"` → `"completed"` 对齐前端 `ExecutionStatus` 类型，`ExecutionLog` Done 标签正常显示
- [工作流] `CacheKey` map 迭代顺序不确定导致 2+ 输入节点缓存命中率 ~50%：改为排序 map 键后再 SHA256
- [工作流] `executeNode` 中 `nodeInstance == nil` 时 `nr.Status` 未赋值残留零值：补 `nr.Status = "failed"`
- [工作流] `fromWorkflowJSON` 边重连按 `node_type` 匹配，同类型多节点时边全连错：改为 ID 映射表 `oldID → newID` 精准重建
- [前端] `NodePalette.vue` 使用 `(window as any).go.main.App.ListNodes()` → 改用 `@/lib/wails` 类型化 `ListNodes` 封装
- [前端] Pin to Terminal 符号写死 `'AAPL'` → 改为读取 `node.data.params.symbol`，兜底 `'600519'`
- [前端] 端口映射仅覆盖 5 种节点类型 → 新增 `GetNodePorts` Go API，`WorkflowCanvas.onDrop` 调用动态获取端口定义
- [工作流] 调度器 `WorkflowExecutor` 接口定义了未实现 → 新增 `ExecuteWorkflowByID` + `workflowExecutorAdapter` 接入 cron scheduler
- [前端] 白色主题全面板文字不可见：根因 `--color-text`/`--color-bg` 变量未定义 → 始终回退 `#e5e7eb`（暗色近白）。`themes.css` 补全变量定义，另 CongressTradingPanel/SentimentPanel/PeerComparisonPanel 等面板的标签徽章对比度提升
- [行情] 全部 EastMoney 适配器 HTTP/2 EOF：Go `net/http` 默认 ALPN 协商 HTTP/2，东方财富 `push2.eastmoney.com` CDN 不兼容 → 连接被 EOF 断开。创建 `newEastMoneyHTTPClient()` 共用工厂，`TLSNextProto` 空 map + TLS 1.2+ 强制 HTTP/1.1。涵盖 `eastmoney.go`、`eastmoney_concept.go`、`eastmoney_signals.go`、`eastmoney_fundflow.go`、`eastmoney_capital.go`
- [财务] PE/PB 始终为 0：Sina 财报不提供 MarketCap → `ComputeRatios` 跳过计算。`GetStockResearch` 从 EastMoney `StockInfo` 回填 `fd.MarketCap`，`TotalDebt` 用 `TotalAssets - TotalEquity` 兜底
- [财务] EPS 为 0（负利润场景）：mootdx `fin.Profit > 0` 跳过负利润 EPS 推导 → 改为 `fin.Profit != 0`
- [财务] 自由现金流为 0：`fetchFromMootdx` 未提取负债/现金流字段 → 添加 `zongfuzhai`/`经营活动现金流量净额` 映射
- [财务] 财务比率标签硬编码英文：`FinancialsPanel.vue` 使用 `k.replace(/_/g, ' ')` → 改为 `$t('research.' + k)`
- [前端] `eastmoneyAdpt` 未初始化：App struct 仅声明字段无赋值 → 所有 EastMoney 数据源（总市值/行业/概念等）全部失效
- [同业] 同业对比显示假美股数据：conceptAdapter HTTP/2 EOF → `mockPeerData` 回退返回 MSFT/GOOGL/AMZN → 删除 mock，`ConceptBlock` 新增 `LeadStockCode` 字段（f140），按龙头股代码去重
- [情感] `SentimentOutput` 序列化丢失：Wails v3 自定义序列化器对 `json:"sentiment,omitempty"` 处理异常 → 移除 `omitempty`
- [情感] 语言参数固定 `en`：中文新闻走 VADER → 永远中性 → `detectLanguage()` 自动识别 zh/en
- [情感] Python sidecar 版本号更新 → 强制重启加载新 NLP 包
- [行情] `GetMarketSnapshot` 硬编码 `"CN"` → 使用 `MarketForSymbol()` 逐 symbol 动态路由，传入 `"AAPL"` 时自动走 US 链而非 CN
- [行情] `GetCorrelationMatrix` / `GetReturnDistribution` / `GetVolatilitySurface` 同理：每个 symbol 独立推断市场，修复这些 API 对港股/美股/加密返回 CN 数据的错误
- [行情] CRYPTO 回退链缺少 `gateio`：国内用户无法获取加密行情 → 加入链尾 (`gateio` 在国内可访问)
- [行情] Polygon 适配器是完全的 stub（总是返回 "not implemented"）→ 从 US 回退链移除，US 链变为 `yahoo → sina → finnhub`
- [前端] `OrderEntryPanel` 调用 `GetQuote` 时市场参数写死 `'CN'`，输入 AAPL 也走 CN 链 → 改为 `detectMarket(symbol)` 动态路由
- [行情] `GetMinuteLine` 仅支持 CN（依赖 mootdx）→ 对非 CN 市场返回明确错误，提示前端切到 1d 间隔
- [前端] `TickerTapePanel` 多市场：新增 CN/HK/US 选项卡切换，HK 预设 8 只港股、US 预设 8 只美股，调用 `detectMarket` 动态路由
- [前端] `PositionDetail` mock 数据替换为真实 `GetPositions` API，按 symbol 过滤显示持仓，空状态显示"暂无持仓"
- [前端] `CandlestickPanel` 非 A 股市场交易时段：HKEX 09:30-12:00/13:00-16:00、NYSE 13:30-21:00 UTC、加密全天候
- [前端] `MarketOverviewPanel` + `HeatmapPanel` 多市场：CN/HK/US 选项卡切换，`GetMarketOverview` 传入 market 参数返回对应指数
- [行情] `GetMarketOverview` 新增 `mkt` 参数：CN 返回 5 大 A 股指数，HK 返回恒生/国企/科技，US 返回 SPX/NASDAQ/DJI
- [前端] `useMarketColors` 新增 composable：按 symbol 市场自动切换颜色方案（CN 红涨绿跌 / 其他绿涨红跌），覆盖 CandlestickPanel/TickerTapePanel/MarketDepthPanel 硬编码颜色
- [前端] `SymbolSearch` 搜索结果增加市场标签筛选：顶部 All/沪/深/港/美/京 过滤 tabs，前端侧过滤
- [前端] 全局市场选择器：TerminalMode header 新增 CN/HK/US 按钮，`session.ui.activeMarket` 持久化到 localStorage，面板可 watch 刷新

### 变更

- [前端] FinancialPanel/StockResearchPanel/CongressTradingPanel 面板级 color 从 `var(--color-text, #e5e7eb)` → `var(--color-text-primary)`
- [前端] StockResearchPanel overview 的 `symbol` 标签从硬编码英文 → i18n `common.symbol`
- [前端] `PeerComparisonPanel` 列简化：代码+名称+市值+市盈率（去除无数据的营收增长/利润率/ROE），添加数据来源提示
- [Go] `eastmoneyAdpt` 初始化 hook 到 App 构造函数，`GetStockResearch` 增加 eastmoney 初始化/失败日志
- [Go] `fetchFromMootdx` 额外提取 `zongfuzhai`/`经营活动现金流量净额` 字段
- [Python] NLP pipeline：`jieba` 中文分词 + 停用词过滤；`NLPPipeline.analyze()` 结果包含 `engine` 诊断字段

### 新增

- [设置] API Key 持久化链路：SettingsPanel「API 密钥」section → settings store → Go UpdateConfig() → config.yaml。支持 FRED、Finnhub、爱问财三种 Key，保存后重启生效
- [行情] GetCommodityQuotes：大商品实时报价 API（新浪期货 `hf_CL` / `hf_NG`），WTI 原油 + 天然气从 FRED 延迟数据迁到实时源
- [前端] 涨跌颜色切换：设置→外观→A股（红涨绿跌）/ 美股（绿涨红跌），CSS 变量 `--color-up` / `--color-down` 动态注入，自选股/市场概览/K线图全面板自动响应
- [前端] 自选股 localStorage 持久化：加/删自选股票自动保存到 `quantflow-watchlist`，重启不丢失
- [Go] 行情快照内存缓存：AdapterRegistry 新增 5s TTL 的 quoteCache，同一股票 5 秒内不重复走容灾链

### 变更

- [宏观] GovDataPanel 能源分类重构：原油/天然气指标从 FRED 移除（延迟 1-3 天），改为新浪期货实时数据，与 FRED 卡片风格统一（可点击、信号徽章）
- [前端] 主题系统重写：CSS 选择器从 `:root` 迁移到 `body`，使用 `body.classList` 而非 `documentElement`（Wails v3 管理 `<html>` class），暗色/亮色主题切换 + 3 级密度均生效

### 修复

- [行情] FRED 天然气系列 ID 错误：`NGDPRPI`（GDP 平减指数）→ `DHHNGSP`（Henry Hub 天然气现货）
- [行情] 财报数据为空：mootdx `client.finance()` 返回 pinyin 字段名（`jinglirun` / `zhuyingshouru`），Go 侧原用中文字段名查询全部 miss。新增 pinyin→中文映射 + EPS/ROE 自动推算
- [行情] 分时图无数据：mootdx `client.minute()` 返回 `price/vol/volume` 三列，rename `vol`→`volume` 产生重复列名导致 `float(Series)` 异常。修复：先 drop 原 `volume` 再 rename
- [行情] 分时图时间轴缺失午休跳空：之前线性映射 index→time 未跳过 11:30-13:00 午休段，导致下午标签错位。修复：`idx ≥ 120` 自动补 90 分钟偏移
- [前端] 5 个假数据面板接入真实后端：PositionPanel（→ GetPositions）、NotifyPanel（→ useNotifyStore）、NewsPanel（→ 新增 GetNews 绑定）、BrokerStatusPanel（→ 新增 GetBrokerStatuses 绑定）、SchedulePanel（→ 新增 CRUD 绑定），移除全部硬编码 mock 数据
- [前端] i18n 国际化全覆盖：zh.ts 扩展至 ~350 key（18 domain），50 个面板全部从硬编码字符串迁移到 `$t()`，en.ts 补齐完整英文翻译，语言切换后全局生效
- [Python] 分时图 stale sidecar：打包运行后 `StartSidecar` 静默复用旧版 Python 进程（代码已更新但 sidecar 未重启），导致 `unsupported data_type 'minute'`。修复：(1) 新增版本号检查 `ExpectedSidecarVersion`，版本不匹配时自动 SIGTERM 旧进程并启动新 sidecar；(2) PID 文件持久化便于跨次重启追踪；(3) `darwin:build` 任务新增 `rsync python/ → build/python/` + venv symlink，确保构建输出的 sidecar 可独立启动
- [主题] 30+ 面板硬编码暗色（`#1a1a2e`/`#16213e`/`#0f2137` 等）→ CSS 变量（`--color-bg-panel`/`--color-bg-subtle`/`--color-bg-input`），亮色主题切换全局面板生效
- [主题] themes.css 新增 `--color-success`/`--color-danger` 语义变量，独立于 CN 涨跌色方案，StatusBar 连接状态修复（已连接=绿、未连接=红）
- [终端] Tab 标签硬编码英文修复：`openPanel()` 从 panel ID 自动生成英文标签（`candlestick`→`Candlestick`）→ 从 registry 读取中文 label（→`K 线图`）
- [终端] 分屏工具栏恢复（`□ ◫ ⊞ ⊟` 预设布局 + Ctrl+1~4 快捷键）
- [终端] Tab 跨 pane 拖拽支持 + `activeLeafId` 追踪（右侧欢迎页打开面板不再默认跳左侧）
- [工作流] NodePalette：5 个硬编码假节点 → 调用 `ListNodes()` 加载 75 个真实节点（18 分类 + 中文标签 + 颜色映射）
- [工作流] CustomNode 卡片中文化：75 个节点类型 + 15 个端口名 → 中文映射（`data_loader`→数据加载、`bollinger`→布林带…）
- [工作流] 工作流组件硬编码暗色 + 英文标签 → CSS 变量 + `$t()` 国际化
- [P&L] Position 新增 `RealizedPnl` 累加器：卖单成交后已实现盈亏不再被 mark-to-market 覆盖，`TotalPnl = RealizedPnl + UnrealizedPnl`
- [风控] `PlaceOrderLive` 新增风险检查（`CheckDrawdown` 断路器 + `CheckOrder` 仓位上限）
- [安全] `GetConfig()` 不再向前端暴露 `api_keys`
- [安全] Python gRPC sidecar 绑定从 `[::]` 改为 `localhost` only
- [风控] 止损/止盈成交失败 → `slog.Error` 报警（原为 `_, _ =` 静默吞咽）
- [后端] MarketDataHub + Scheduler 启动初始化
- [后端] `RunBacktest` Wails 绑定

---

## [2026.6.24] - 2026-06-24

### 修复

- [Engine] OMS FillOrder 卖出裁剪顺序：fillQty 在更新订单账本（FilledQty/FilledAvgPrice）前裁剪到持仓量，保证 order.FilledQty == Trade.Quantity == 持仓变动 (P0-1)
- [Engine] CNEngine 补齐 A 股涨跌停限制：主板 ±10%、创业板/科创板 ±20%、北交所 ±30%，涨停封板买不进、跌停封板卖不出；覆盖信号卖出、止损/止盈、买入三条路径 (P0-2)
- [Python] 横截面因子经标准 RPC 路径失效：zscore/rank 现在在完整多标的 panel 上计算后按 symbol 切片，不再逐 symbol 过滤后退化为 0/0.5 (P0-3)
- [Python] ML 树模型训练无验证集：XGBoost/LightGBM 加入 80/20 时序安全切分（shuffle=False 防未来泄漏），同时返回 train_* 与 val_* 指标；小样本(<50) fallback 到 train-set 评估并 warning (P0-4)

### 已知偏差（延后至 Phase B）

- [Engine] ST/*ST ±5% 涨跌停：无法从 symbol 代码识别 ST 状态，需 Config.PriceLimitOverrides，Phase B 实现
- [Python] ML 模型注册表 SQLite 持久化 + model_id 返回：spec P0-4 要求，Phase A 仅实现 train/val split，模型元数据落库延后
- [Engine] 分钟频涨跌停的 prevClose 语义：当前每个 bar 更新 prevClose，日频正确，分钟频需配合 P1-1 (T+1 非日频) 在 Phase B 统一处理
- [Python] 横截面因子 RPC 路径集成测试：Phase A 以 compute() 直调测试验证因子数学，gRPC 端到端测试 Phase B 补齐

---

## [2026.6.19] - 2026-06-19

### 新增

- [Terminal] Symbol Context 联动系统：Bloomberg 式 Link Group 架构，4 个颜色编码 Group（红/绿/黄/蓝），跨面板 symbol 自动联动
- [Terminal] symbolContextStore：Pinia store 管理 4 个 LinkGroup + panel-to-group binding + symbol history（最近 10 个）+ Linked/Unlinked 状态
- [Terminal] SymbolBar：快速 symbol 输入栏 + 4 Group 切换标签，对标 Bloomberg 命令行
- [Terminal] StatusBar 增强：显示所有活跃 Group 的 symbol + 颜色标识
- [Terminal] DockTab 增强：面板标签页显示 Group 颜色圆点指示器

### 变更

- [Frontend] 15 个面板迁移到 symbolContext 联动系统：
  - **5 个 Publisher**（可发布 symbol 到 Group）：WatchlistPanel、QuoteDetailPanel、StockResearchPanel、OrderEntryPanel、FinancialsPanel
  - **10 个 Subscriber**（自动跟随 Group symbol）：CandlestickPanel、SentimentPanel、PeerComparisonPanel、AnalystEstimatesPanel、InsiderTradingPanel、MarketDepthPanel、PositionDetail、DrawingPanel、DistributionPanel、SurfaceChartPanel、PredictionDashboardPanel
- [Frontend] terminalStore：移除 deprecated activeSymbol / lastSymbolUpdate / setActiveSymbol（已迁移到 symbolContextStore）

- [MarketData] SatelliteAdapter：卫星替代数据适配器，集成 NASA POWER API（太阳能 GHI / 风速）和 NASA FIRMS（野火数据），免费无需 API Key，5 个预定义能源区域（德州风能走廊、北海风电场、戈壁太阳能基地、撒哈拉太阳能带、美国中西部农业带）
- [Research] SatelliteService：卫星能源数据服务，支持区域快照/详情/30 天时间序列/异常检测信号，5 分钟 TTL 缓存，完整的 mock 数据回退
- [Workflow] SatelliteNode：卫星数据工作流节点，区域输入 → energy_signal/solar_ghi/wind_speed 输出，alternative_data 分类
- [Frontend] SatellitePanel Vue 组件：卫星能源面板，5 个区域卡片（太阳辐射 + 风速仪表盘 + 趋势箭头 + 野火计数），ECharts 30 天太阳/风速双轴折线图，Wails IPC 回退模式
- [Frontend] GeopoliticsPanel Vue 组件：地缘政治风险面板，2 列卡片网格展示 10 个地缘政治主题，支持风险等级筛选（全部/高/中/低），ECharts 情绪趋势图，Wails IPC 模拟回退模式

- [券商] AlpacaBroker（美股）：完整 trading.Broker 实现 — Connect/GetAccount/GetOrders/GetPositions/SubmitOrder/CancelOrder，默认 paper trading，环境变量配置
- [行情] YahooAdapter 修复：Cookie jar + crumb 机制 + HTML 检测 + query1/query2 容灾 + 完整浏览器 UA + 港股/美股代码归一化
- [行情] AkShareAdapter 扩展：腾讯 K 线 OHLCV 支持（日/周/月），CN 和 HK 双市场，toTencentCode 支持港股代码
- [行情] GateIOAdapter：加密实时报价 + OHLCV，Gate.io 免 key 免翻墙（国内唯一可用），BTC/USDT 实测 62740.10
- [行情] SinaAdapter 港股支持：扩展 toSinaCode + toSinaHKCode，parseSinaHKQuote 港股字段映射，实测腾讯 440.20 港元
- [行情] AkShare/Tencent K 线修复：迁移到 proxy.finance.qq.com/newkline 端点（CN/HK 通用）
- [行情] FinnhubAdapter：美股实时报价 + OHLCV，免费 API Key，60 次/分钟
- [行情] 完整 Phase 1-3 适配器体系：新闻/资金流/龙虎榜/研报/巨潮/爱问财 等 15+ 新适配器
- [研究] 9 个 Research Service：Financials/AnalystEstimates/PeerComparison/InsiderTrading/CongressTrading/Capital/FundFlow/Northbound/Announcement
- [前端] 17 个新面板：
  - **市场** (5)：MarketOverview（7 大指数 + 涨跌比 + 板块排名）、MarketDepth（五档盘口 + 逐笔成交）、Heatmap（板块热力图）、TickerTape（滚动报价条）、CryptoOverview（Top 20 加密）
  - **图表** (7)：EquityCurve（净值曲线 + 回撤）、SurfaceChart（波动率曲面）、Correlation（相关性矩阵）、Distribution（收益分布）、Drawing（画线工具）、MonteCarlo（蒙特卡洛模拟）、Rebalance（组合再平衡）
  - **交易** (5)：OrderBlotter（订单流水）、Execution（成交明细）、BasketOrder（篮子交易）、BrokerStatus（券商状态）、ActionCenter（操作中心）
- [前端] stats.ts 统计库：pearsonMatrix/histogramBins/simulateGBM/computeDrawdowns/sharpeRatio（纯 TS，零依赖）
- [前端] Store 扩展：dataStore (+marketOverview)、portfolioStore (+orders/trades/equityCurve)
- [前端] SymbolIdentity + NormalizeCN：统一 8 种股票代码格式 → 9 个转换方法
- [前端] SymbolSearchService：全 A 股 5534 + 港股 2584 + 美股 13462 内存索引，拼音/代码/名称搜索
- [前端] SymbolSearch 组件：实时联想输入框，200ms 防抖，键盘导航
- [前端] vitest.config.ts + setup：全局 ResizeObserver mock
- [研究] SentimentEngine：NLP 情绪分析 + 新闻自动拉取
- [Python] NLPPipeline：三层回退（VADER→TextBlob→关键词）+ SnowNLP 中文，依赖可选
- [工作流] NewsFetcherNode：输入代码→自动拉取新闻→输出给 SentimentNode
- [行情] EastMoneyRateLimiter：全局限流器（QPS≤2，500ms 间隔+抖动）
- [Terminal] 9 个新 Wails 导出方法：GetCapitalData/FundFlow/NorthboundFlow/Announcements 等
- [另类数据] Polymarket 预测市场适配器：Gamma API 免费接入，5 类事件（经济/加密/政治/体育/科技），概率走势图
- [另类数据] PredictionMarketService：5 分钟 TTL 缓存 + 概率突破信号提取（默认阈值 5%）
- [前端] PredictionMarketPanel：类别过滤 + 概率走势 ECharts + 信号徽标（第 47 个面板）
- [工作流] prediction_market 节点：类别/阈值输入 → 事件列表 + 交易信号输出（类别：alternative_data）
- [另类数据] GDELT 地缘政治适配器：DOC 2.0 API 免费接入，10 个预定义话题查询（中东/台海/俄乌/关税/朝鲜/美联储/欧洲能源/恐怖主义/中国经济/半导体）
- [另类数据] GeopoliticsService：5 分钟 TTL 缓存 + 风险评分引擎（覆盖量+情绪双重异常检测）+ 10 话题 Mock 数据
- [工作流] geopolitics 节点：topic/region 输入 → risk_signal + risk_score + tone 输出（类别：alternative_data）
- [另类数据] GovDataAdapter（FRED + SEC EDGAR）：15 个美国经济指标 + SEC 公司申报文件查询，FRED_API_KEY 环境变量配置，无 key 自动降级
- [另类数据] GovDataService：5 分钟 TTL 缓存 + 宏观信号提取（15 指标→bullish/bearish/neutral 信号），Mock 数据全覆盖
- [工作流] gov_data 节点：indicator/country 输入 → macro_signal + latest_value + change 输出（类别：alternative_data）
- [前端] GovDataPanel 组件：15 指标 3 列卡片网格，6 类过滤标签（全部/GDP/通胀/就业/利率/能源/房地产），ECharts 时间序列图，信号摘要统计
- [文档] 数据源整合/前后端修复/剩余缺口修复/爱问财/代码归一化/预测市场等 8 篇 Spec

### 修复

- [行情] BaiduAdapter：ResultCode int/string 类型不稳定
- [行情] THSConsensusAdapter：GBK 编码未解码 + 表格解析重构
- [行情] queryDatacenter filter URL 编码缺失 → HTML 误返回
- [行情] EastMoneyNewsAdapter innerParams JSON URL 编码 → 400 错误
- [行情] THSHotAdapter Market 字段 interface{} → JSON 解析
- [Python] requirements.txt 补充 nltk/snownlp
- [Python] deep_engine.py PyTorch 未安装时引用错误
- [Python] nlp_pipeline.py vader 下载超时（线程 join 3s 硬超时）
- [Python] fetcher.py mootdx 三级容灾服务器 + pandas truthiness
- [研究] FinancialsBundle 嵌套结构 + InsiderTransaction Value 字段 + insider_trades→insider 键名
- [研究] PeerComparisonData net_margin→margin 字段名
- [Terminal] GetCongressTrades 导出 → 面板不再走 mock
- [工作流] SentimentNode 信号阈值 ±0.3→±0.15

---

## [2026.6.18] - 2026-06-18

### 新增

- [研究] 情绪分析模块：NLP pipeline (Python) + SentimentEngine (Go) + SentimentNode (工作流)
- [研究] 6 个工作流节点：sentiment/stock_research/financials/peer_compare/analyst_estimates/insider_trades
- [研究] ResearchRepo (SQLite, migration 011)，无 Python sidecar 时优雅降级
- [前端] researchStore：研究分析 Pinia store，Wails 桥接 + 前端 mock
- [前端] 7 个研究面板：SentimentPanel/StockResearchPanel/FinancialsPanel/PeerComparisonPanel/AnalystEstimatesPanel/InsiderTradingPanel/CongressTradingPanel
- [Python] SentimentService gRPC（单文本 + 批量并发 N 路扇出）
- [文档] proposal 实现状态图谱 — 全部模块 ✅/🔶/📋 标注
- [文档] 7 篇待开发 Spec：研究分析/另类数据/缺失面板/工作流节点/Broker/AI/杂项增强
- [行情] Mootdx 适配器：真实通达信 TCP 协议（Python gRPC 桥接），免注册免 Key 无封 IP 风险
- [Go] DataClient：gRPC 客户端包装，超时/重试
- [行情] 12 适配器全部注册到 AdapterRegistry，CN 容灾链生效

#### Phase 11A — 前端测试
- [前端] 8 个 Pinia store 测试套件：38 个 store 测试
- [前端] 22 个面板冒烟测试 + 8 个工作流/DockView 组件测试
- [前端] 合计 76 测试全通过

#### Phase 11B — Python 测试
- [Python] 波动率/成交量/横截面因子测试 + LLM provider 测试
- [Python] 合计 120 测试全通过 (+38)

#### Phase 11C — Go 深度测试
- [Go] 13 个行情适配器测试 + AI capability 测试 + storage/config/schedule/notify 扩展
- [Go] 合计 251 测试全通过 (+75)

#### Phase 10.1 — 收益预测引擎
- [Python] TreeEngine (XGBoost/LightGBM) + DeepEngine (LSTM/Transformer)
- [Go] MLClient + ModelRegistry + FeatureEngineer/Train/Predict/Evaluate 节点
- [前端] ModelRegistry/PredictionDashboard 面板 + mlStore

#### Phase 10.2 — Alpha 挖掘引擎
- [Python] AlphaMiningEngine：遗传规划因子发现 (gplearn)

#### Phase 10.3 — RL 交易引擎
- [Python] TradingEnv + PPO/DQN/SAC Trainer + ReplayBuffer
- [前端] RLMonitorPanel + mlStore 扩展

#### Phase 10.4 — 风险建模
- [Python] GARCH/GJR-GARCH/EGARCH + Ledoit-Wolf 协方差
- [前端] RiskDashboard 扩展：GARCH 波动率图表

### 修复

- [行情] Mootdx OHLCV 区间转发（之前硬编码 1D，1W/1m/5m/15m 被静默改回日线）
- [行情] EastMoney OHLCV：URL 修正 + 丢弃 HTTP 响应修复
- [行情] Tencent 适配器：接入真实 K 线 API（web.ifzq.gtimg.cn，2000 根）
- [行情] Baidu 适配器：报价解析器修复 + 真实 K 线 API
- [行情] 删除旧 mootdx 适配器（包装新浪 HTTP，不是通达信）
- [行情] Sina/Tencent/AKShare/Baidu OHLCV：不再静默返回假数据
- [工作流] Engine 传递节点参数到 Execute（修复配置参数丢失）
- [工作流] 三种 OHLCVBar 类型统一 → data_loader→backtest 管道修复
- [App] PythonBridge 接入 ML 节点 + ML 节点注册
- [行情] TuShare 解析修复（data.fields+data.items 格式）
- [Python] Factor engine NaN 保留（不再转为 0，防前视偏差）
- [存储] 迁移执行包裹事务（原子性）
- [回测] PnL 扣除交易成本（之前用毛价，系统高估胜率和夏普）
- [交易] OMS 卖空验证/已实现盈亏/空头止损止盈/市价单风控
- [App] GetPortfolioSummary 现金从真实交易历史推导（不再硬编码 10 万）
- [AI] Quote capability 接入 AdapterRegistry（不再返回 $100 占位符）
- [前端] CandlestickPanel OHLCV 索引修正 + DockView 内存泄漏修复
- [Python] isTransient 用 gRPC status.Code 替代字符串匹配
- [Python] evaluator.py eval() 替换为 AST 白名单解析器（消除 RCE）
- 更多修复共 30+ 项

---

## [2026.6.17] - 2026-06-17

### 新增

#### Phase 1 — 核心骨架
- [引擎] Go 模块初始化 + config/logging/Makefile
- [工作流] BaseNode 接口 + NodeRegistry + 5 内置节点 + DAG + TopoSort + Engine
- [存储] SQLite WAL + 嵌入式迁移框架 + WorkflowRepo
- [前端] qf CLI + 示例工作流 + 样本数据

#### Phase 2 — 前端 + 交易引擎
- [前端] Wails v3 桌面壳 + Vue 3 前端嵌入
- [前端] Terminal 模式：CommandBar (Ctrl+K) + DockView 停靠系统 + 8 面板
- [前端] Workflow 模式：vue-flow 画布 + CustomNode/NodePalette/PropertyPanel/ExecutionLog
- [前端] 4 Pinia stores：terminal/workflow/data/session（含 undo/redo）
- [引擎] 交易引擎：OMS + PaperEngine + OrderMatcher + RiskPipeline
- [引擎] MarketDataHub：Go channel pub/sub + L0 TTL 缓存 + 3 适配器
- [存储] Migration 004 (orders/trades/positions) + 005 (ohlcv_cache)

#### Phase 2.5 — 数据源加固
- [行情] 14 个真实数据适配器 + AdapterRegistry + FallbackChain 容灾
- [行情] A 股 7 源容灾链 + 加密 3 源链，RetryWithBudget + TransientError

#### Phase 3 — Python Sidecar + 因子 + 回测
- [Python] gRPC sidecar 项目 + 25 Alpha 因子（动量/趋势/波动/量/横截面）
- [Python] Arrow IPC 零拷贝传输 + 19 测试
- [Go] PythonBridge + FactorNode/StrategyNode/BacktestNode
- [引擎] 回测引擎：CN/US 市场规则（T+1/涨跌停/印花税）+ 7 指标
- [前端] BacktestResultPanel + FactorAnalysisPanel

#### Phase 4 — AI Agent 系统
- [AI] AgentOrchestrator (ReAct 循环) + CapabilityRegistry (10 能力) + EventEmitter (SSE)
- [AI] 4 AgentProfile (YAML) + AgentNode (工作流集成)
- [Python] LLM Service：4 provider (OpenAI/Anthropic/DeepSeek/Ollama) + PromptTemplate
- [Python] Skill KB：15 技能 Markdown 文件
- [前端] AIChatPanel：SSE 流式 + Markdown 渲染 + 工具调用可视化

#### Phase 5 — 券商 + 风控 + 通知 + 调度
- [券商] BinanceBroker（REST 实盘）+ FutuBroker（存根）
- [通知] Telegram/InApp 通知 + Notify/Alert 节点
- [调度] robfig/cron 引擎 + Schedule/Wait 节点
- [组合] PortfolioService + RiskMetrics (VaR/CVaR/MaxDD/Sharpe/Sortino/Calmar)
- [存储] Migration 006-009

#### Phase 6 — 前端面板 + SSE + Pinia Store 扩展
- [前端] 7 新面板 + portfolioStore/notifyStore + ECharts 集成

#### Phase 7 — 主题 + i18n + 设置
- [前端] CSS Variables 双主题 + 3 密度 + vue-i18n 中英文 + SettingsPanel (9 配置区)

#### Phase 8 — 节点扩展 (20→34)
- [工作流] 4 指标 + 3 数据 + 2 信号 + 2 风控 + 3 工具 = 14 新节点

#### Phase 9 — 因子原子 + 信号工程 (34→54)
- [工作流] 12 因子原子 + 5 信号工程 + 3 控制/输出 = 20 新节点

### 变更
- [引擎] Go module 从 `app/` 重构到项目根目录（Wails v3 标准布局）
- [前端] 5 核心面板从硬编码色迁移到 CSS 自定义属性

---

## 模板

```markdown
## [YYYY.M.D] - YYYY-MM-DD

### 新增
- [范围] 新功能描述

### 变更
- [范围] 变更内容描述

### 修复
- [范围] Bug 描述和根因

### 移除
- [范围] 移除内容及原因
```

**范围标签**：`[终端]` `[工作流]` `[引擎]` `[券商]` `[行情]` `[AI]` `[前端]` `[存储]` `[Python]` `[文档]`

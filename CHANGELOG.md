# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [2026.6.28] - 2026-06-28

### 新增

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

- [Docs] 新增 5 篇规范文档：easy-tdx 集成、分时图实时渲染、前端质量重塑、股票名称展示、前端质量实施计划

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

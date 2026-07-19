# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。
格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [2026.7.19] - 2026-07-19

### 变更

- [Frontend] Phase 2 骨架屏迁移完成 — 所有面板组（chart-group-a/b、prediction-valuation、research-tools、system-aux、funding/hk、arbitrage/calendar、bonds/funds/futures、toolbar-variant）全面迁移至 `SkeletonPanel` / `LoadingState` / `ErrorState` 共享组件，消除约 8,183 行重复架子代码
- [Frontend] 剩余面板批量 Token 化 — font-size 全部改用 `--font-*` CSS 变量（最小字号 11px），padding/margin 改用 `--space-*` token，消除面板层级硬编码色值
- [Frontend] stylelint 规则提升至 error 级别 — Phase 2 所有 CSS 合规检查通过
- [Frontend] PanelTable 增强 — 粘性表头（sticky header）、`colorize` 守卫、全局 `btn-sm` 类，恢复 Whale Tracking 地址 tooltip 和 CB Arbitrage 阈值高亮
- [Frontend] 测试覆盖率补充 — WatchlistPanel 新增 49 条测试用例，更新 SurfaceChartPanel/TickerTapePanel 测试适配新组件

## [2026.7.17] - 2026-07-17

### 新增

- [Terminal] 崩溃报告系统 — `internal/crash/` 包捕获 goroutine panic（`CapturePanic` deferred recovery）和 OS 崩溃信号（SIGABRT/SIGSEGV/SIGILL/SIGBUS）；结构化崩溃报告（panic 消息、堆栈、最近 100 条日志、应用状态）保存为本地 JSON，零 PII
- [Terminal] 崩溃报告存储 — 平台专属目录（macOS `~/Library/Logs/QuantFlow/crashes/`、Linux `~/.local/share/QuantFlow/crashes/`、Windows `%LOCALAPPDATA%\QuantFlow\crashes\`），30 天自动清理，opt-in 上传（默认关闭）
- [Frontend] CrashDialog 崩溃恢复对话框 — 下次启动时展示崩溃时间、panic 消息、可折叠堆栈和日志；CrashHistoryPanel 崩溃历史管理（设置 → 崩溃报告），支持查看/删除/上传
- [Storage] `RingBuffer.LastN(n)` 导出最近 N 条日志供崩溃报告嵌入
- [CI] GitHub Actions 自动发版流水线 — `git tag v*` push 触发 6 平台并行构建矩阵（macOS arm64/amd64、Linux amd64/arm64、Windows amd64/arm64），自动生成 SHA256 checksums，上传至 GitHub Releases
- [Build] 跨平台打包脚本 — `scripts/darwin-package.sh`（.app + .dmg）、`scripts/linux-package.sh`（.tar.gz）、`scripts/windows-package.ps1`（.zip）
- [Build] Makefile 发版目标 — `make release` 一站式构建 macOS + Linux + checksums；Homebrew formula 模板
- [Frontend] Toast 通知系统 — `useToast` composable（4 类型 + 30s 去重合并 + 自动消失），`ToastContainer` 浮动容器（fixed top-right），错误类永久展示需手动关闭
- [Frontend] 状态栏增强 — 底部 StatusBar 新增行情源/券商/Python 连接状态行（彩色状态点 + 点击弹详情对话框），版本号显示
- [Logging] RingBuffer WS 实时广播 — `SetHub(hub)` 注入 WebSocket Hub，新 Push 的日志条目通过 `system:notification` topic 推送到前端
- [Backend] `GetConnectionStatus()` IPC — 返回实时行情适配器、券商连接、Python sidecar 三组状态
- [Trading] 日结报告系统 — `GenerateDailyReport()` 自动汇总当日成交、持仓市值、盈亏、佣金税费、最大回撤、最佳/最差交易；报告持久化到 SQLite `daily_reports` 表（migration 019）
- [Trading] Paper→Live 实盘切换 — `TradingMode` 类型（paper/live）+ `SafetyCheck` 安全检查清单 + `EngineMode` 模式管理器；`SwitchToLive()` 执行安全检查后方可切换
- [Trading] 紧急关停 — Live 模式下红色`紧急平仓`按钮 → 确认后并行撤销所有委托 + 切换回 Paper 模式
- [Frontend] LiveModeBanner — Paper 模式绿色标签（`切换实盘`按钮）+ Live 模式红色闪烁横幅（`紧急平仓` / `切换模拟`按钮）+ 安全检查对话框
- [Trading] 持仓对账系统 — `ReconcileAll()` 对比 OMS 持仓与券商持仓，生成差异报告（数量/均价偏差、OMS 独有/券商独有）；报告持久化到 SQLite `reconciliation_reports` 表（migration 020）
- [CI] 覆盖率门禁 — 后端 60% / 前端 40% 阈值，CI 自动检查 + `make coverage-gate`
- [CI] Goroutine 泄漏检测 — `go test -race` 关键路径（ws/market/workflow），`make leak-check`
- [CI] 错误处理审计 — `scripts/error-audit.sh` 扫描未检查错误、裸 panic、log.Fatal、%v 错误包装
- [Frontend] API 密钥管理面板 — 13 个数据源/券商/AI 的密钥集中管理（保存/删除/验证），按市场分组
- [Frontend] 首次启动向导 — 5 步指引（欢迎 → 终端模式 → 工作流模式 → 数据配置 → 准备就绪），localStorage 记录完成状态
- [Workflow] 6 套官方策略模板 — 双均线交叉、布林带突破、RSI 均值回归、期现套利、组合再平衡、AI 情绪驱动
- [Frontend] DailyReportPanel — 日结报告展示面板（盈亏汇总、持仓明细、最佳/最差交易高亮、导出 CSV、历史报告列表），注册到面板系统
- [Analysis] 分析维度补齐 P0 — SectorDashboard（申万行业热力图+PE/PB估值水位），ValuationBand（PE/PB Band 均值±2σ通道+历史分位），DupontAnalysis（ROE拆解树+同行雷达图）
- [Analysis] 分析维度补齐 P1 — MacroDashboard（GDP/CPI/PMI/M2四象限+中美对比），MarketStyle（大小盘×价值成长象限+情绪温度计），EventStudy（CAR事件窗口曲线+超额收益）
- [Analysis] 分析维度补齐 P2 — ShareholderPanel（十大流通股东+机构持仓），UnlockCalendar（限售解禁日历+冲击预警），FactorAttribution（组合收益因子拆解瀑布图）
- [Frontend] 分析面板联动 — Dupont/Shareholder/EventStudy 与 Watchlist 通过 SymbolContext 联动，切换自选股自动刷新，标题栏显示红点链接指示器
- [Frontend] Watchlist 右键分析入口 — 杜邦分析/股东分析/事件分析
- [Frontend] 交易模式指示器重构 — 从独立横条改为 Header 内嵌胶囊标签（模拟=绿色/实盘=红色脉冲+紧急平仓按钮）
- [Frontend] 状态栏紧凑化 — 连接状态合并为统一胶囊样式，正常项收入"已连接"徽章，异常项单独红色展示
- [Frontend] Welcome 面板图标补全 — 日结报告/API密钥/帮助中心/存储管理/布局模板 + 9 个分析面板 SVG 图标

### 变更

- [Frontend] 设计系统重构（亮色优先）— `themes.css` 整体重写：亮色主题为默认（暗色改为 `body.theme-dark`，两主题同等维护）；全新亮色调色板，文字对比度全部 ≥ WCAG AA 4.5:1；brand 粉色并入 accent 蓝色，色彩角色统一；`session.ui.theme` 默认值改为 light
- [Frontend] 去装饰化 — 全局删除约 30 处辉光阴影、渐变分割线、网格背景、渐变文字标题（`background-clip: text`）与逐卡 stagger 入场动画；`0 0 0 3px` 焦点环全部保留；动画支持 `prefers-reduced-motion` 降级
- [Frontend] z-index 收敛为 6 层语义 token（`--z-sticky/dropdown/overlay/modal/toast/tooltip`），替换 14 处魔法数（9999/10002 等）
- [Frontend] 字体栈修正 — 正文 fallback 移除 JetBrains Mono，新增 `--font-mono` 仅限数字/行情场景，全局 `tabular-nums`；紧凑档字号下限 10→11px，表格表头 11→12px
- [Workflow] 工作流模式亮主题适配 — 新增 20+ `--wf-*` token（画布/节点/边/端口/minimap），硬编码 GitHub-dark 色值全部 token 化；暗色视觉零变化，亮色正常渲染；新增 `cssVar()` helper 供 JS/SVG 读取 token
- [Frontend] 欢迎页重构 — 新增面板搜索框 + 分类筛选 chips（带数量），82 个面板可检索；搜索匹配空格归一化（"K线"可命中"K 线 / 分时"）；命令栏搜索同步修复
- [Frontend] 命令栏空态改进 — 显示最近使用面板（可直接打开）+ 历史命令，无内容时显示搜索提示；分类标签中文化（新增 5 个 i18n key）
- [Frontend] PRODUCT.md 设计上下文 — 记录产品定位、亮色优先决策、反参考与设计原则，供后续设计工作对齐

### 修复

- [CI] 修复存量测试失败 — 适配器注册计数断言从魔法数字改为显式名单（17 个，与 `registerMarketAdapters` 对齐，新增适配器需显式登记）;mootdx 实盘集成测试加 `integration` build tag 移出默认测试套件（需 Python sidecar + 外网，且其历史 K 线断言与 mootdx 已知限制冲突）
- [Market] K 线缓存永不过期 — OHLCV 缓存命中只要求"范围内有数据"且不检查新鲜度，日 K 停留在上次抓取日期不再更新（如 7/10 后不更新到 7/17）；现缓存命中前校验末根 bar 是否达到最新交易日（跳过周末），过期则重新抓取（10 分钟冷却防止节假日空打），重抓失败回退旧数据而不是报错
- [Frontend] 命令栏弹层永远撑满 520px 高 — overlay flex 容器默认 `align-items: stretch` 拉伸所致，改为 `align-items: flex-start` 按内容收缩
- [Frontend] 设置页 3 个 i18n key 缺失（API 密钥/日志/布局模板）导致导航直接显示原始 key
- [Frontend] CrashDialog/UpdatePrompt 引用从未定义的 `--bg-surface`，弹窗背景透明
- [Frontend] 美股涨跌配色切换从不生效 — `color-us` 类缺少对应 CSS 规则，补齐亮/暗两套
- [Frontend] ContextMenu/SymbolSearch/TickerBar 等硬编码暗色背景与边框（`#1c2333`/`#374151`/`#60a5fa`），亮主题下破版，全部 token 化
- [Frontend] FinancialsPanel 主题检测读取 `html.dark`（从未被设置）改为 `body.theme-dark`
- [CI] vitest 误收集 e2e/ 目录 Playwright 用例导致 `npm run test` 必失败，`vite.config.ts` 增加 exclude
- [Frontend] 布局模板面板 null 引用 — `savedLayouts` IPC 返回 null 时 Array.isArray 守卫
- [Frontend] 版本号显示 v0.0.1 → 2026.7.17 — `GetVersion()` 优先使用 ldflags 注入的构建版本
- [Trading] Paper/Live 模式管理器初始化 — `tradingMode` 在 ServiceStartup 中正确创建
- [StatusBar] 连接状态首次不刷新 — `onMounted` 立即调用 `GetConnectionStatus()` + 30s 定时刷新
- [Sector] 行业仪表盘 adapter chain 修复 — 复用 `App.GetIndustryRanks` 已有的完整 fallback + 缓存逻辑

## [2026.7.16] - 2026-07-16

### 新增

- [Terminal] 自动更新系统 — GitHub Releases 客户端（`internal/updater/release.go`），支持平台资产匹配、SHA256 校验和验证、版本号比较（`YYYY.M.D` 格式）；macOS/Linux/Windows 三平台 Replace + Restart 逻辑
- [Terminal] 更新检查 IPC — `CheckUpdate()`、`ApplyUpdate()`、`GetUpdateInterval()`、`SetUpdateInterval()` 暴露给前端；配置 `UpdateCheckInterval`（always/daily/never）
- [Frontend] UpdatePrompt 更新提示对话框 — 显示当前/最新版本、更新日志、下载进度条、手动检查按钮、检查频率设置
- [Frontend] 启动 30 秒后自动检查更新，SettingsPanel 关于区新增「检查更新」按钮和频率下拉框

### 变更

- [Storage] Config 结构体新增 `UpdateCheckInterval` 字段，默认 `"daily"`

## [2026.7.14] - 2026-07-14

### 变更

- [Market] Tencent K线解析全面重构 — 支持三种响应格式（map / array of maps / flat array）和混合类型数值解析（`rawKline` 替代 `[][]string`）；AKShare 适配器改用 `json.RawMessage` 动态分发，兼容新旧 API 返回格式
- [Market] `toTencentCode` 支持裸港股代码 — 1-5 位纯数字自动加 `hk` 前缀（`00700` → `hk00700`），不再要求 `.HK` 后缀

### 修复

- [Market] Tencent OHLCV 解析 panic — 当 API 返回 `[["2026-07-14", 450.0, 455.0, ...]]` 而非常规 map 格式时，`ql.([]any)` type assertion 不会 panic，改用多格式回退解析

---

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

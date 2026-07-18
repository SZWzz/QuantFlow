# 前端视觉统一与精修设计

日期：2026-07-18
状态：已获用户批准（brainstorming 三节设计均确认）
范围：Terminal 模式全部 82 个面板 + 终端外壳；不含 workflow 画布

## 1. 背景与问题

2026-07-17 已完成亮色优先主题重设计（commit `06366ef`），token 体系（`frontend/src/assets/themes.css`）与共享组件库（`frontend/src/terminal/components/panel/`）均已建好且质量合格。但审计发现 ~80% 面板未接入，仍处于"自绘三件套"状态：

| 问题 | 数据 |
|---|---|
| 面板头部 3+ 套写法 | 共享 `PanelHeader` 仅 14 个面板用；43 个手写 `.panel-header`；~15 个用 `.toolbar`/`.tab-bar`/`.filter-bar` 等各自命名，padding/字号/边框互相漂移 |
| font-size 硬编码 | 824 处 px，覆盖 84/84 面板文件；9px–18px 共 9 档；`--font-*` token 几乎无人引用，密度切换对字体失效 |
| padding/margin 硬编码 | 913 处 px，~85 文件；`--space-*` 密度体系被绕过 |
| 空/错态 N 份拷贝 | 40 个面板各自重定义 `.empty-state`，17 个各自定义 `.panel-error`；共享 `EmptyState`/`LoadingState` 合计仅 ~10 面板采用 |
| 表格 3 种实现 | 原生 `<table>` ×21、div 模拟 ×28、共享 `PanelTable` ×5 |
| ECharts 主题双轨 | ~14 文件走 `useChartTheme`（读 CSS 变量）；~9 文件硬编码暗色 hex，亮色主题下坐标轴/分割线颜色错误 |
| 圆角/状态色散落 | 44 处硬编码 `border-radius`（~20 文件）；style 块内 113 处 `rgba()`（~51 文件） |

结论：视觉不统一的根源不是 token 缺失，而是 token 与共享组件"建好了没人搬"。

## 2. 目标与原则

- 全部面板统一成一种骨架长相，风格遵循 PRODUCT.md 定位（TradingView 亮色系 × Linear：克制专业）
- 在克制基础上允许适度表现力：KPI 排版、图表质感、状态性微动效、面板层次
- 精修只发生在共享组件层一次，82 个面板通过迁移自动继承，不逐面板抠细节
- 遵守 PRODUCT.md 设计原则：对比度 ≥ WCAG AA、密度可选、组件词汇一致、动效只表达状态、亮暗双主题平等维护

## 3. 视觉语言精修规范

### 3.1 面板骨架

- 外壳由 DockTab 统一提供（`--color-bg-panel` 底、无面板自绘外框）；规则：**面板组件禁止自绘外框/阴影/圆角**
- 统一 `PanelHeader`：高 40px；padding `var(--space-sm) var(--panel-padding)`；标题 13px/600 `--color-text-primary`；副标题 12px tertiary；右侧控件区 gap 8px；底部 1px `--color-border-subtle`
- Tab 型面板：下划线式 tab，选中 = 文字 primary + 2px accent 下划线；切换时下划线 200ms 滑动
- 内容区 padding `var(--panel-padding)`（随密度缩放）
- 面板内分区只用"1px 分割线 + 间距"，**禁止嵌套卡片**

### 3.2 排版层级（9 档字号收敛为 4 档）

- 面板标题 13/600 · 区块标题 12/600 secondary · 正文 13 · 辅助 12 tertiary
- 数字/价格/代码一律 `var(--font-mono)` + `tabular-nums`
- 硬规则：最小 11px（仅 compact 档辅助文字）；清除全部 9px/10px 字号

### 3.3 表格（一种表格语言）

- 全部收敛到精修版 `PanelTable`：行高 `var(--table-row-height)`（密度联动）、表头 12px tertiary、数值列右对齐等宽、文本列左对齐
- 涨跌 = 颜色 + `+/-` 号（色盲友好）
- 斑马纹 `--table-row-odd` + hover `--table-row-hover`；选中行只用 `--color-bg-selected`，无装饰色条

### 3.4 KPI 统计块

- 统一 `StatItem` 组件：label 12px tertiary + 数值 20px/600 等宽 + 可选涨跌 badge，横向排列，gap 16–24px
- 不用 SaaS 式渐变 hero 大数字模板

### 3.5 状态三件套（各只有一份实现）

- 空态 `EmptyState`：32px 图标 tertiary + 13px/500 标题 + 12px 描述 + 可选动作按钮，垂直居中
- 加载 `LoadingState`：骨架屏优先，不用 spinner
- 错误 `ErrorState`（新增）：danger 图标 + 描述 + 重试按钮

### 3.6 图表

- 全部面板收口到 `useChartTheme`（读 CSS 变量，MutationObserver 随主题切换），色板 `--chart-1..6`，网格 `--chart-grid`，背景透明融入面板
- 清除 ~9 个面板硬编码暗色 hex（`#2a2a3a`/`#8b8ba0` 等）
- 删除 `BacktestPanel` 内联复制的 K线 option，回归 `buildKlineOption`

### 3.7 微动效（只表达状态，150–250ms ease-out）

- hover/选中：底色/边框色过渡（`--transition-fast`）
- Tab 下划线滑动 200ms
- **数据刷新反馈**：数值变化时背景以涨/跌色短暂闪烁后淡出（TradingView 式信息性动效）
- `prefers-reduced-motion` 全部降级为即时切换

### 3.8 质感

- 阴影只给浮层（dropdown/modal/toast），面板平面化；玻璃效果仅顶栏浮层
- 圆角统一：控件 `--radius-md`（6px）、卡片 `--radius-lg`（12px）；清除 44 处硬编码圆角
- 状态色只保留语义角色，不做装饰

## 4. 工程结构与迁移机制

### 4.1 两步走：全局类先行，组件深化随后

- **第一步（零模板改动）**：在 `themes.css` 定义全局 `.panel-header`/`.empty-state`/`.panel-error` 等基础类（参数与第 3 节规范一致），批量删除 43+40+17 份 scoped 拷贝。机械删除，删完全局面板立刻统一为精修长相，风险极低
- **第二步（按批深化）**：迁移时把手写结构换成共享组件（支持 tabs/controls 插槽），并完成 token 化清理

### 4.2 共享组件层（`frontend/src/terminal/components/panel/`）

- 精修：`PanelHeader`、`PanelTabs`（下划线滑动）、`PanelTable`（行高 token/斑马纹/数值列等宽右对齐）、`EmptyState`、`LoadingState`（骨架屏）
- 新增：`StatItem`、`ErrorState`、`useFlashOnUpdate`（数据刷新涨跌闪烁 composable）
- ECharts：`useChartTheme` 补全所有在用图表类型后全量替换

### 4.3 单面板迁移 checklist（7 步，每面板同一把尺）

1. header → `PanelHeader`（含 `.toolbar`/`.tab-bar`/`.filter-bar` 变体）
2. 表格 → `PanelTable`
3. 空/错/加载态 → `EmptyState`/`ErrorState`/`LoadingState`
4. `font-size`/`padding`/`margin` px → `--font-*`/`--space-*` token（密度切换随之生效）
5. hex/`rgba()` 颜色 → 语义 token
6. ECharts → `useChartTheme`
7. 删面板自绘外壳（border/shadow/radius）

### 4.4 防退化

- stylelint 规则：scoped style 内禁 hex 颜色与 px 字号（Phase 0 上线为 warn，Phase 2 完成后转 error）
- 约定写进 `frontend/AGENTS.md`：新面板必须从共享组件搭建

## 5. 批次划分

### Phase 0：地基（一次性做完）

- 精修 + 新增共享组件（见 4.2）
- `themes.css` 补全局骨架类；批量删除 scoped 拷贝（43 header + 40 empty-state + 17 panel-error）
- `useChartTheme` 补全 + 替换 ~9 个硬编码暗色图表面板
- stylelint warn 规则上线
- 产出：不改任何面板模板，全 App 视觉一致性已肉眼可见提升

### Phase 1：打样验收（5 个面板，覆盖所有形态）

- `WatchlistPanel`（header + 表格 + 刷新闪烁）
- `CandlestickPanel`（图表 + 工具栏）
- `PositionPanel`（KPI + 表格）
- `NewsPanel`（列表 + 空态）
- `MarketOverviewPanel`（多区块）
- 作为"标准长相"验收点，用户确认后才动其余面板

### Phase 2：分批迁移（每批独立可交付）

1. 行情图表类（~20 个：Heatmap、DepthComparison、Correlation、FundFlow…）
2. 交易持仓类（~15 个：BasketOrder、TradeHistory、BrokerStatus…）
3. 研究分析类（~25 个：Dupont、EventStudy、FactorAttribution、MacroDashboard…）
4. 系统数据类（~20 个：Settings、Storage、Log、HelpCenter…）

全部完成后 stylelint 规则从 warn 转 error。

## 6. 验收方式（每个 Phase 结束都跑）

- `npm run typecheck`（vue-tsc）+ `npm run test`（vitest）全绿
- 视觉走查：`wails dev` 启动截图，亮/暗双主题 × 三档密度抽查
- 对比度抽查：正文 ≥ 4.5:1
- 每批产出迁移前/后截图对比，用户确认再进下一批

## 7. 非目标（YAGNI）

- 不动 workflow 画布（已有独立 `--wf-*` token 体系，风格自洽）
- 不改信息架构、不增删面板功能、不动后端
- 不引入 UI 框架（Element Plus 之类），共享组件层已够用
- 不做编排式入场动画、不做暗色优先、不做玻璃拟态装饰

# QuantFlow 前端美化计划 Spec

> Version: 2026-06-30 | 阶段: Spec → Plan → Execute
> 目标: 统一 60+ 面板的视觉语言，消除硬编码样式，建立共享组件库，提升专业金融终端质感。

---

## 1. 现状诊断（Audit）

### 1.1 已具备的良好基础 ✅

| 模块 | 评价 |
|------|------|
| `themes.css` | 完整的 Design Token 系统（Dark/Light + 3 Density + CN 红涨绿跌），覆盖颜色/间距/字体/圆角/阴影/动画 |
| `DockView` | 成熟的布局系统（分割、拖拽、Tab 切换、4 种预设布局） |
| `WelcomePanel` | 视觉设计较成熟（渐变标题、卡片网格、分类色、hover 动效） |
| `StatusBar` | 状态指示清晰（连接状态、分组、时间） |
| `SymbolBar` / `TickerBar` | 设计统一，交互流畅 |
| `SettingsPanel` | 左侧导航 + 表单布局规范，复用性强 |
| `SkeletonPanel` | 已有三种骨架屏类型（table/card/chart） |

### 1.2 核心问题 🔴

#### A. Design Token 使用严重不一致（P0）

大量面板**完全绕过** `themes.css` 的 token，使用硬编码颜色：

```css
/* 硬编码颜色分布（grep 统计） */
#ef4444, #dc2626, #f87171  →  红色系（涨/危险）约 20+ 处
#22c55e, #16a34a, #4ade80  →  绿色系（跌/成功）约 20+ 处
#60a5fa, #3b82f6, #2563eb  →  蓝色系（强调）约 15+ 处
#e5e7eb, #6b7280, #9ca3af  →  灰色系（文字）约 30+ 处
#1a1a2e, #2a2a3e          →  旧暗色背景（已废弃）约 10+ 处
#4a90d9                    →  AlphaMining 旧主题色  多处
#534ab7                    →  IndicatorPanel 旧主题色  多处
```

**后果**：
- 切换 Light Theme 后面板仍显示暗色硬编码，严重破坏视觉一致性
- 修改主题色需要改 N 个文件，无法单点维护
- 部分面板使用已废弃的 CSS 变量名（如 `--border-color`, `--term-bg-dim`, `--term-accent-dim`）

#### B. 面板布局/间距/字体不统一（P0）

| 面板 | padding | 标题大小 | 表格字体 | Tab 样式 |
|------|---------|----------|----------|----------|
| MarketOverview | `16px` | `14px` | `11px` / `12px` | 自定义 `.mkt-tab` |
| LimitUpDown | `12px` | `14px` | `10px` / `12px` | 自定义 `.mkt-tab` / `.f-tab` |
| Watchlist | `6px 8px` | 无标题 | `11px` / `var(--font-base)` | 无 |
| Indicator | `12px` | `15px` | `12px` | 无（chip 网格） |
| AlphaMining | `12px` | 默认 `h3` | `0.85em` / `0.9em` | 无 |
| GovData | `8px 12px` | `14px` | `11px` / `13px` | `.source-tab` + `.tab` |
| BacktestResult | `16px` | `16px` | 无 | 无 |

**后果**：相邻面板并排时视觉跳跃明显，缺乏"同一个应用"的质感。

#### C. 重复造轮子：每个面板自研 Tab/Toolbar/Table/Empty（P1）

- `panel-header` + `market-tabs` + `header-controls` 组合在 **10+ 面板**中各自实现
- 表格样式（`.table-header`, `.table-row`, `.br-header-row`, `.data-table`）在 **8+ 面板**中各自实现
- Empty State / Loading State 样式不统一
- Refresh/AutoRefresh 按钮组合在 **5+ 面板**中各自实现

#### D. 响应式与容器适配缺陷（P1）

- `GovDataPanel` 的 `indicator-grid` 使用 `grid-template-columns: repeat(3, 1fr)`，在窄面板（如侧边栏 300px）中卡片被严重压缩
- `MarketOverviewPanel` 的 `indices-row` 使用 `overflow-x: auto`，但无滚动指示器
- 表格列宽使用固定像素（如 `width: 64px`），在 compact density 下显得过大
- 骨架屏 overlay 使用 `position: absolute`，但父容器未设 `position: relative`，布局可能错位

#### E. 视觉层次与交互细节粗糙（P1）

- `GovDataPanel` 使用 emoji（🟢🔴⚪🔄📈📉）作为状态指示，缺乏专业金融终端质感
- 部分按钮无 hover/active/focus 状态，或过渡动画生硬
- 输入框聚焦 ring 在某些面板中被 border 覆盖
- 部分面板缺少 `overflow: hidden` 导致内容溢出破坏 Dock 布局

#### F. 字体栈与排版不一致（P2）

- 数字字体混用：有的用 `font-variant-numeric: tabular-nums`（好），有的用默认比例字体
- 等宽字体引用不统一：`monospace` vs `'JetBrains Mono', monospace`
- 中文内容使用默认 `font-size: 13px`，在 comfortable density 下可读性不佳

---

## 2. 目标（Goals）

### 2.1 核心目标

1. **单点主题控制**：所有面板 100% 通过 `themes.css` token 渲染，零硬编码颜色
2. **视觉一致性**：任意两个面板并排时，看起来像"同一个应用"的有机组成部分
3. **专业质感**：消除 emoji/粗糙边框/生硬过渡，达到 Bloomberg Terminal / 同花顺 iFinD 级别的视觉标准
4. **响应式适配**：面板在 280px ~ 全屏宽度范围内均有良好的信息密度与可读性

### 2.2 量化指标

| 指标 | 当前 | 目标 |
|------|------|------|
| 硬编码颜色值（#hex）在 .vue 文件中 | ~100+ 处 | 0 处（除特殊图像/图表配置外） |
| 面板使用共享组件（PanelHeader/PanelTable/EmptyState） | 0% | 100% |
| 面板 padding 标准差 | 较大 | ≤ 2px（统一为 token） |
| 表格/列表字体大小标准差 | 较大 | 统一为 `var(--font-xs)` ~ `var(--font-base)` 阶梯 |
| 响应式断点覆盖 | 0 | 至少 3 个（compact/normal/wide） |

---

## 3. 方案设计

### 3.1 架构图：三层设计体系

```
┌─────────────────────────────────────────────────────────────┐
│  Layer 3: 面板层 (Panel) — 只关心业务逻辑和数据               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │Watchlist │  │MarketOv  │  │GovData   │  ... 60+ panels │
│  │  Panel   │  │  Panel   │  │  Panel   │                 │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                 │
├───────┼─────────────┼─────────────┼─────────────────────────┤
│  Layer 2: 共享组件库 (Panel Components)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │PanelHeader│  │PanelTable│  │PanelTabs │  │EmptyState│   │
│  │PanelToolbar│ │PanelCard │  │PanelChart│  │Skeleton  │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│  Layer 1: 设计 Token (themes.css)                           │
│  颜色 · 间距 · 字体 · 圆角 · 阴影 · 动画 · 渐变             │
│  Dark/Light · Compact/Default/Comfortable · CN/US 配色     │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Layer 1: 设计 Token 增强（themes.css）

新增/完善以下 token，确保覆盖所有面板需求：

#### 新增 Token（现有缺失的）

```css
/* 面板容器 */
--panel-padding: var(--padding-panel);        /* 默认 10px */
--panel-padding-lg: 16px;                      /* 宽面板 */
--panel-padding-sm: 8px;                       /* 窄面板 */

/* 面板标题 */
--panel-title-size: var(--font-lg);           /* 16px */
--panel-title-weight: 600;
--panel-subtitle-size: var(--font-sm);        /* 12px */
--panel-subtitle-color: var(--color-text-tertiary);

/* 表格 */
--table-header-size: var(--font-xs);          /* 10px */
--table-header-color: var(--color-text-tertiary);
--table-header-weight: 600;
--table-row-size: var(--font-sm);             /* 12px */
--table-row-height: var(--row-height);        /* 36px */
--table-row-height-compact: 28px;
--table-border: var(--color-border-subtle);
--table-row-hover: var(--color-bg-hover);
--table-row-odd: rgba(255,255,255,0.02);     /* dark */
--table-row-odd-light: rgba(0,0,0,0.02);    /* light */

/* Tab（面板内） */
--tab-height: 28px;
--tab-padding: 4px 12px;
--tab-font-size: var(--font-xs);
--tab-active-bg: var(--color-bg-panel);
--tab-active-border: var(--color-accent);
--tab-inactive-color: var(--color-text-tertiary);

/* Toolbar */
--toolbar-height: 36px;
--toolbar-gap: var(--space-sm);
--toolbar-padding: var(--space-sm) var(--space-md);

/* 卡片（指标卡片/数据卡片） */
--card-padding: var(--space-md);
--card-gap: var(--space-sm);
--card-min-width: 200px;

/* 专业图标（替代 emoji） */
--icon-bullish: var(--color-up);      /* 使用 CSS 伪元素或 SVG */
--icon-bearish: var(--color-down);
--icon-neutral: var(--color-text-tertiary);
--icon-trend-up: var(--color-up);
--icon-trend-down: var(--color-down);
```

#### 废弃旧变量（不再使用）

- `--border-color` → 使用 `--color-border`
- `--term-bg-dim` → 使用 `--color-bg-subtle`
- `--term-accent-dim` → 使用 `--color-accent-soft`
- `--color-text` → 使用 `--color-text-primary`
- `--color-bg` → 使用 `--color-bg-panel`
- 直接 `#1a1a2e`, `#2a2a3e` → 全部替换为 token

### 3.3 Layer 2: 共享组件库

新建 `frontend/src/terminal/components/panel/` 目录，包含以下组件：

#### 3.3.1 PanelHeader

```vue
<!-- 统一面板头部：标题 + 可选 Tabs + 可选 Controls -->
<PanelHeader
  :title="$t('misc.market_overview')"
  :subtitle="formatTime(updatedAt)"
  :tabs="[{ key: 'CN', label: 'CN' }, { key: 'HK', label: 'HK' }]"
  :active-tab="activeMarket"
  :controls="[{ icon: 'refresh', action: refresh, loading }]"
  @tab-change="switchMarket"
/>
```

**样式标准**：
- 高度：自适应（flex wrap）
- 标题：`var(--panel-title-size)` + `var(--panel-title-weight)`
- 副标题：`var(--panel-subtitle-size)` + `var(--panel-subtitle-color)`
- Tabs：使用统一的 `--tab-*` token，样式与 TerminalMode 的 market-tabs 一致但使用 token
- Controls 区：按钮使用 `.btn` + `.btn-ghost` 全局类

#### 3.3.2 PanelToolbar

```vue
<!-- 工具栏：搜索 + 筛选 + 操作按钮 -->
<PanelToolbar
  :search-placeholder="$t('common.search')"
  :filters="[{ key: 'all', label: '全部' }, ...]"
  :actions="[{ label: '导出', icon: 'export', action: onExport }]"
  @search="onSearch"
  @filter-change="onFilter"
/>
```

#### 3.3.3 PanelTable

```vue
<!-- 统一表格：支持固定列、排序、hover、斑马纹、密度适配 -->
<PanelTable
  :columns="[
    { key: 'symbol', label: '代码', width: 80, align: 'left' },
    { key: 'price', label: '价格', align: 'right', format: 'price' },
    { key: 'changePct', label: '涨跌幅', align: 'right', colorize: true },
  ]"
  :data="stocks"
  :row-height="density === 'compact' ? 'compact' : 'default'"
  :striped="true"
  :loading="loading"
  @row-click="onSymbolClick"
/>
```

**样式标准**：
- 表头：`--table-header-size`, 大写/小写标签统一, `border-bottom: 1px solid var(--color-border-strong)`
- 行高：`--table-row-height`（default 36px, compact 28px, comfortable 44px）
- 斑马纹：使用 `--table-row-odd`（主题适配）
- Hover：`--table-row-hover`
- 数字列：右对齐 + `font-variant-numeric: tabular-nums`
- 颜色化列：根据值自动使用 `--color-up` / `--color-down`
- 响应式：列宽支持 `flex` 比例，最小宽度保护

#### 3.3.4 PanelTabs

```vue
<!-- 面板内 Tab 切换（如 GovData 的 source-tabs） -->
<PanelTabs
  :tabs="[{ key: 'fred', label: 'FRED 美国' }, ...]"
  :active="activeSource"
  :variant="'pill' | 'underline' | 'button'"
  @change="onChange"
/>
```

三种变体：
- `pill`：圆角胶囊，适合少量选项（如 source switch）
- `underline`：底部指示线，适合大量选项（如 category filter）
- `button`：方框按钮，适合操作型 tab

#### 3.3.5 PanelCard

```vue
<!-- 指标卡片/数据卡片（如 MarketOverview 的 index-card / GovData 的 indicator-card） -->
<PanelCard
  :title="idx.name"
  :value="idx.last"
  :change="idx.changePct"
  :format="'price' | 'percent' | 'volume'"
  :sparkline="idx.sparkline"
  :clickable="true"
  @click="onClick"
/>
```

**样式标准**：
- 背景：`var(--gradient-card)` + `border: 1px solid var(--color-border)`
- Hover：`border-color: var(--color-accent)` + `box-shadow: var(--shadow-md)` + 微上移
- 数值：`font-size: var(--font-lg)` + `font-weight: 600` + `tabular-nums`
- 涨跌幅：`badge-up` / `badge-down` 样式
- Sparkline：SVG polyline，颜色与涨跌一致

#### 3.3.6 EmptyState

```vue
<!-- 空状态 -->
<EmptyState
  :icon="'search' | 'chart' | 'data' | 'settings'"
  :title="$t('common.no_data')"
  :description="$t('workflow.backtest_empty')"
  :action="{ label: '去设置', handler: openSettings }"
/>
```

**样式标准**：
- 居中布局，flex column
- 图标：`48px` + `color: var(--color-text-tertiary)` + `opacity: 0.5`
- 标题：`var(--font-lg)` + `color: var(--color-text-secondary)`
- 描述：`var(--font-sm)` + `color: var(--color-text-tertiary)`
- 操作按钮：`.btn-primary` 或 `.btn`（如有）

#### 3.3.7 LoadingState

```vue
<!-- 加载状态（替代 SkeletonPanel） -->
<LoadingState
  :type="'table' | 'card' | 'chart' | 'inline'"
  :rows="5"
/>
```

**样式标准**：
- 使用 `themes.css` 中已有的 shimmer 动画
- 背景色使用 token（`--color-bg-elevated`, `--color-bg-hover`）
- 在 Light Theme 下自动适配（骨架色为浅色）

#### 3.3.8 SignalBadge（新增）

```vue
<!-- 替代 GovDataPanel 的 emoji 信号指示 -->
<SignalBadge
  :signal="'bullish' | 'bearish' | 'neutral'"
  :show-label="true | false"
  :size="'sm' | 'md' | 'lg'"
/>
```

**样式标准**：
- bullish：`badge-up` 样式（红底/红字）
- bearish：`badge-down` 样式（绿底/绿字）
- neutral：`badge` 基础样式（灰底/灰字）
- 无 emoji，使用圆点 + 文字，专业简洁

#### 3.3.9 TrendIndicator（新增）

```vue
<!-- 趋势方向指示（替代 emoji） -->
<TrendIndicator
  :direction="'up' | 'down' | 'flat'"
  :change="0.5"
/>
```

**样式标准**：
- up：▲ 图标 + `--color-up`
- down：▼ 图标 + `--color-down`
- flat：▬ 图标 + `--color-text-tertiary`

### 3.4 Layer 3: 面板迁移计划

#### 3.4.1 迁移优先级

| 优先级 | 面板 | 原因 |
|--------|------|------|
| P0 | `AlphaMiningWorkspacePanel` | 硬编码 `#4a90d9` + 废弃变量 `--border-color`，视觉最突兀 |
| P0 | `IndicatorPanel` | 硬编码 `#534ab7` + `#1a1a2e` + `#2a2a3e`，与主题完全不兼容 |
| P0 | `MarketOverviewPanel` | 硬编码颜色多，骨架屏布局问题 |
| P0 | `LimitUpDownPanel` | 硬编码 `#dc2626`/`#16a34a`/`#60a5fa`，表格样式独立 |
| P1 | `GovDataPanel` | emoji 问题 + 响应式问题 + 部分硬编码 |
| P1 | `WatchlistPanel` | 样式较规范但可统一使用 PanelTable |
| P1 | `BacktestResultPanel` | 极简，可丰富为 EmptyState + 操作引导 |
| P2 | 其他 50+ 面板 | 逐个检查并迁移到共享组件 |

#### 3.4.2 迁移模式

每个面板的迁移遵循以下模式：

```
1. 识别面板中的重复结构（header/table/toolbar/empty/loading）
2. 用对应的共享组件替换
3. 删除硬编码颜色，替换为 CSS 变量
4. 删除废弃的 CSS 变量引用
5. 检查响应式（缩小面板到 300px 测试）
6. 检查 Light Theme 兼容性
7. 检查 Compact/Comfortable Density 兼容性
```

#### 3.4.3 具体迁移清单

**P0 批次（立即执行）**

| 面板 | 变更 |
|------|------|
| `AlphaMiningWorkspacePanel` | 1. `.chip.active` 的 `#4a90d9` → `var(--color-accent)` 2. `.btn-run` 的 `#4a90d9` → `var(--color-accent)` 3. `var(--border-color)` → `var(--color-border)` 4. 使用 `PanelTable` 替换原生 table 5. 使用 `PanelHeader` 替换自定义 header 6. 使用 `EmptyState`（如结果为空） |
| `IndicatorPanel` | 1. `#1a1a2e` → `var(--color-bg-panel)` 2. `#2a2a3e` → `var(--color-border)` 3. `#534ab7` → `var(--color-accent)` 4. `#e5e7eb` → `var(--color-text-primary)` 5. `#6b7280` → `var(--color-text-tertiary)` 6. 使用 `PanelHeader` 替换 `.panel-header` 7. 使用 `PanelTable` 替换 `.data-table` 8. `.indicator-chip` → 使用 `PanelCard` 或统一 chip 样式 |
| `MarketOverviewPanel` | 1. `#ef4444`/`#22c55e` → `var(--color-up)`/`var(--color-down)` 2. `#60a5fa` → `var(--color-accent)` 3. `#4b5563` → `var(--color-text-tertiary)` 4. 使用 `PanelHeader` 替换 `.panel-header` 5. 使用 `PanelCard` 替换 `.index-card` 6. 使用 `PanelTable` 替换 `.block-rank-table` 7. 修复骨架屏 `position: absolute` 问题 |
| `LimitUpDownPanel` | 1. `#dc2626`/`#16a34a` → `var(--color-up)`/`var(--color-down)` 2. `#60a5fa` → `var(--color-accent)` 3. 使用 `PanelHeader` 替换 `.panel-header` 4. 使用 `PanelTable` 替换 `.table-wrapper` 5. 使用 `EmptyState` 替换 `.empty-state` |

**P1 批次（次优先）**

| 面板 | 变更 |
|------|------|
| `GovDataPanel` | 1. `#16a34a`/`#dc2626` → `var(--color-up)`/`var(--color-down)` 2. emoji → `SignalBadge` + `TrendIndicator` 3. `indicator-grid` 添加响应式：`@media (max-width: 400px) { grid-template-columns: 1fr; }` 4. 使用 `PanelTabs` 替换 `.source-tab` / `.tab` 5. 使用 `PanelHeader` 替换 `.panel-header` |
| `WatchlistPanel` | 1. 使用 `PanelTable` 替换 `.symbol-list` 2. 使用 `PanelToolbar` 替换 `.panel-toolbar` |
| `BacktestResultPanel` | 1. 使用 `EmptyState` 替换 `.empty-state` 2. 添加操作引导按钮（"去 Workflow 创建回测"） |
| `WelcomePanel` | 1. `#dc2626`/`#16a34a` 在 `.snap-pct` → `var(--color-up)`/`var(--color-down)` 2. 检查其他硬编码 |

---

## 4. 执行计划（Phase 规划）

### Phase 1: 基础设施（1-2 天）

**目标**：建立共享组件库 + 增强 themes.css

| 任务 | 输出 |
|------|------|
| 1.1 增强 `themes.css` | 新增 token（panel/table/toolbar/card/signal 等），废弃旧变量列表 |
| 1.2 创建 `PanelHeader.vue` | 支持 title/subtitle/tabs/controls 的组合 |
| 1.3 创建 `PanelToolbar.vue` | 支持 search/filter/actions |
| 1.4 创建 `PanelTable.vue` | 支持 columns/data/loading/striped/row-click |
| 1.5 创建 `PanelTabs.vue` | 支持 pill/underline/button 三种变体 |
| 1.6 创建 `PanelCard.vue` | 支持 title/value/change/sparkline/clickable |
| 1.7 创建 `EmptyState.vue` | 支持 icon/title/desc/action |
| 1.8 创建 `LoadingState.vue` | 支持 table/card/chart/inline 四种骨架 |
| 1.9 创建 `SignalBadge.vue` | 替代 emoji 信号指示 |
| 1.10 创建 `TrendIndicator.vue` | 替代 emoji 趋势指示 |

### Phase 2: P0 面板迁移（2-3 天）

| 任务 | 面板 | 变更 |
|------|------|------|
| 2.1 | `AlphaMiningWorkspacePanel` | 全量迁移到共享组件 |
| 2.2 | `IndicatorPanel` | 全量迁移到共享组件 |
| 2.3 | `MarketOverviewPanel` | 全量迁移到共享组件 |
| 2.4 | `LimitUpDownPanel` | 全量迁移到共享组件 |

### Phase 3: P1 面板迁移（2-3 天）

| 任务 | 面板 | 变更 |
|------|------|------|
| 3.1 | `GovDataPanel` | 响应式 + emoji 替换 + 组件迁移 |
| 3.2 | `WatchlistPanel` | PanelTable + PanelToolbar 迁移 |
| 3.3 | `BacktestResultPanel` | EmptyState + 操作引导 |
| 3.4 | `WelcomePanel` | 硬编码颜色清理 |
| 3.5 | 其他高频面板 | 逐个检查硬编码和组件迁移 |

### Phase 4: 全局清理与验证（1-2 天）

| 任务 | 方法 |
|------|------|
| 4.1 | 全局 grep `#` 颜色值，确认 .vue 文件中零硬编码（图表 config 除外） |
| 4.2 | 全局 grep 废弃变量名（`--border-color`, `--term-bg-dim` 等），确认清零 |
| 4.3 | 在 Dark/Light/Compact/Comfortable 四种组合下截图对比 |
| 4.4 | 将面板缩小到 280px 宽度测试响应式 |
| 4.5 | 运行前端 build，确保无类型/样式错误 |

---

## 5. 验收标准（Acceptance Criteria）

### 5.1 代码层面

- [ ] `grep -r "#ef4444\|#dc2626\|#22c55e\|#16a34a\|#60a5fa\|#3b82f6\|#e5e7eb\|#6b7280\|#1a1a2e\|#2a2a3e\|#4a90d9\|#534ab7" frontend/src/terminal/panels/` 返回空（除注释和图表配置外）
- [ ] `grep -r "--border-color\|--term-bg-dim\|--term-accent-dim\|--color-text\|--color-bg" frontend/src/terminal/panels/` 返回空
- [ ] 所有 P0/P1 面板使用 `PanelHeader`/`PanelTable`/`PanelTabs`/`PanelCard`/`EmptyState`/`LoadingState` 之一

### 5.2 视觉层面

- [ ] 在 Dark Theme 下，所有面板背景/边框/文字颜色协调一致
- [ ] 在 Light Theme 下，所有面板正确显示浅色主题（无暗色残留）
- [ ] Compact Density 下信息密度提升但不拥挤
- [ ] Comfortable Density 下留白充足、可读性良好
- [ ] 面板在 280px ~ 全屏范围内均有合理布局（无截断/重叠/溢出）
- [ ] GovDataPanel 不再使用 emoji，改为专业图标/徽章

### 5.3 交互层面

- [ ] 所有按钮有 hover/active/focus 状态
- [ ] 所有 Tab 切换有视觉反馈
- [ ] 表格行有 hover 高亮
- [ ] 加载状态和空状态在所有面板中统一呈现

---

## 6. 风险与回退方案

| 风险 | 影响 | 缓解 |
|------|------|------|
| 共享组件引入 regression | 中 | 每个面板迁移后，在对应模式（Dark/Light/Compact）下截图对比 |
| 图表库（ECharts）样式与主题冲突 | 低 | 图表颜色使用 `useChartTheme` composable，已在宏观面板中验证 |
| 工作量超出预期 | 中 | 按 Phase 切分，每 Phase 可独立合并；P0 完成即可显著改善体验 |
| VueFlow 工作流画布与主题冲突 | 低 | 画布使用独立 CSS，不受影响 |

---

## 7. 参考与灵感

- **Bloomberg Terminal**: 信息密度、专业配色、无 emoji、表格为主
- **同花顺 iFinD**: 中文金融终端的表格/卡片布局、红涨绿跌
- **TradingView**: 暗色主题质感、卡片 hover 效果、边框层次
- **Linear**: 简洁的按钮/Tab/输入框样式、微交互动效
- **本项目的 `WelcomePanel`**: 作为现有最佳实践的参考基准

---

*Spec 作者: AI Assistant | 评审后进入 Plan 阶段*

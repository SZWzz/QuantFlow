# QuantFlow 前端视觉设计改进 Spec

- **日期:** 2026-07-02
- **范围:** 前端整体视觉设计升级（CSS 变量系统、面板组件、交互反馈、品牌化细节）
- **风格方向:** 平衡方案（兼具数据密度与美感）+ 克制优雅（3-4 个微妙品牌细节）
- **实施策略:** 混合式（分四阶段，每阶段可独立运行和验证）

---

## 1. 设计原则

1. **专业感优先:** 作为量化金融终端，视觉应传递冷静、精确、可信赖的气质。
2. **层级清晰:** 通过颜色对比、边框、阴影建立明确的信息层级（App → Dock → Panel → Card → Element）。
3. **克制优雅:** 动画和装饰元素必须服务于信息传达，避免花哨。
4. **一致性:** 所有视觉元素必须使用设计令牌（CSS 变量），禁止硬编码。

---

## 2. Phase 1：修复缺陷（P0）

### 2.1 定义缺失的 CSS 变量

在 `frontend/src/assets/styles/themes.css` 的 `[data-theme="dark"]` 和 `:root`（light）部分添加以下变量：

**语义颜色（柔和背景）:**

| 变量名 | Light | Dark |
|--------|-------|------|
| `--color-success-soft` | `#ecfdf5` | `rgba(34, 197, 94, 0.15)` |
| `--color-danger-soft` | `#fef2f2` | `rgba(239, 68, 68, 0.15)` |
| `--color-warning-soft` | `#fffbeb` | `rgba(245, 158, 11, 0.15)` |
| `--color-info-soft` | `#eff6ff` | `rgba(59, 130, 246, 0.15)` |

**分类颜色（WelcomePanel 分类图标）:**

| 变量名 | Light | Dark |
|--------|-------|------|
| `--cat-market` | `#3b82f6` | `#60a5fa` |
| `--cat-trading` | `#10b981` | `#34d399` |
| `--cat-research` | `#8b5cf6` | `#a78bfa` |
| `--cat-ml` | `#f59e0b` | `#fbbf24` |
| `--cat-risk` | `#ef4444` | `#f87171` |
| `--cat-data` | `#06b6d4` | `#22d3ee` |
| `--cat-workflow` | `#ec4899` | `#f472b6` |
| `--cat-portfolio` | `#6366f1` | `#818cf8` |
| `--cat-settings` | `#64748b` | `#94a3b8` |
| `--cat-help` | `#84cc16` | `#a3e635` |
| `--cat-govdata` | `#d946ef` | `#e879f9` |

### 2.2 修复 App.vue class 重复挂载

**文件:** `frontend/src/App.vue`

**问题:** `div.app` 上同时绑定了 `theme-${session.ui.theme}` 和 `density-${session.ui.density}`，但 theme store 已在 `body` 上挂载这些类。类名重复，可能导致样式冲突。

**修复:** 从 `div.app` 移除 `:class` 中的 theme 和 density 类绑定，只保留 `app` 类。确保 theme 和 density 类仅由 theme store 在 body 上管理。

**验证:** 检查所有依赖 `div.app` 上的 theme/density 类的选择器，确认它们通过 `body.theme-xxx` 或 `body.density-xxx` 可以正确命中。

---

## 3. Phase 2：提升视觉层级（P1）

### 3.1 深色模式层级重建

**目标:** 解决 app 背景和面板背景对比度不足（0.03）的问题，建立清晰的层级。

**颜色层级表（Dark 模式）:**

| 层级 | 变量名 | 当前值 | 新值 | 说明 |
|------|--------|--------|------|------|
| App 背景 | `--color-bg-app` | `#0b0f19` | `#090d16` | 再压暗，作为画布 |
| 面板背景 | `--color-bg-panel` | `#121929` | `#161f2e` | 提亮 0.04，与 app 背景建立对比 |
| 卡片/输入框 | `--color-bg-elevated` | `#162236` | `#1a2638` | 比面板再亮，用于嵌套卡片 |
| 悬浮态 | `--color-bg-hover` | `#1a2a40` | `#1e3248` | hover 态比卡片更亮 |
| 选中态 | `--color-bg-active` | 新增 | `#223450` | 选中/激活态 |

**Light 模式同步调整:**

| 层级 | 变量名 | 当前值 | 新值 | 说明 |
|------|--------|--------|------|------|
| App 背景 | `--color-bg-app` | `#f8fafc` | `#f1f5f9` | 稍深，避免太白 |
| 面板背景 | `--color-bg-panel` | `#ffffff` | `#ffffff` | 保持不变 |
| 卡片/输入框 | `--color-bg-elevated` | `#f1f5f9` | `#f8fafc` | 比面板稍暗，建立层级 |
| 悬浮态 | `--color-bg-hover` | `#e2e8f0` | `#e2e8f0` | 保持不变 |
| 选中态 | `--color-bg-active` | 新增 | `#dbeafe` | 淡蓝色选中背景 |

### 3.2 边框和分隔线增强

**目标:** 当前边框 `#1e2d4d` 在 `#121929` 上几乎不可见，失去分隔作用。

| 变量名 | 当前值 (Dark) | 新值 (Dark) | 说明 |
|--------|--------------|------------|------|
| `--color-border` | `#1e2d4d` | `#2a3e5f` | 基础边框，提升可见度 |
| `--color-border-strong` | `#243250` | `#334b73` | 强边框，面板分隔、表头下划线 |
| `--color-border-hover` | 新增 | `#3d5a85` | 悬浮态边框高亮 |

**Light 模式同步:**

| 变量名 | 当前值 (Light) | 新值 (Light) |
|--------|---------------|-------------|
| `--color-border` | `#e2e8f0` | `#cbd5e1` |
| `--color-border-strong` | `#cbd5e1` | `#94a3b8` |
| `--color-border-hover` | 新增 | `#64748b` |

### 3.3 字号系统调整（桌面端）

**目标:** 基础字号 13px 在桌面端偏小，表格表头 10px 难以阅读。

| 变量名 | 当前值 | 新值 | 说明 |
|--------|--------|------|------|
| `--font-base` | 13px | 14px | 基础字号 |
| `--font-sm` | 12px | 13px | 次要文本 |
| `--font-xs` | 11px | 12px | 辅助信息 |
| `--font-lg` | 15px | 16px | 面板标题、大数字 |
| `--font-xl` | 18px | 20px | 大标题 |
| `--table-header-size` | 10px | 11px | 表头字号（取消 uppercase） |
| `--table-header-weight` | 600 | 500 | 表头字重（medium 即可） |
| `--table-row-size` | 12px | 13px | 表格行内容 |
| `--table-row-height` | 28px | 32px | 表格行高（增加呼吸感） |
| `--badge-font-size` | 10px | 11px | 徽章文字 |
| `--panel-title-size` | 15px | 16px | 面板标题 |
| `--panel-subtitle-size` | 12px | 13px | 面板副标题 |

### 3.4 文字颜色层级增强

**目标:** `--color-text-secondary` 和 `--color-text-tertiary` 差异太小，难以区分。

**Dark 模式:**

| 变量名 | 当前值 | 新值 | 说明 |
|--------|--------|------|------|
| `--color-text-primary` | `#e2e8f0` | `#f1f5f9` | 接近纯白，最醒目 |
| `--color-text-secondary` | `#94a3b8` | `#cbd5e1` | 次要信息，比 primary 稍暗 |
| `--color-text-tertiary` | `#64748b` | `#94a3b8` | 辅助信息，拉开与 secondary 差距 |
| `--color-text-disabled` | 新增 | `#64748b` | 禁用态文字 |

**Light 模式:**

| 变量名 | 当前值 | 新值 | 说明 |
|--------|--------|------|------|
| `--color-text-primary` | `#1e293b` | `#0f172a` | 接近纯黑 |
| `--color-text-secondary` | `#64748b` | `#475569` | 次要信息 |
| `--color-text-tertiary` | `#94a3b8` | `#64748b` | 辅助信息 |
| `--color-text-disabled` | 新增 | `#94a3b8` | 禁用态文字 |

### 3.5 面板聚焦态（Dock 激活面板）

**目标:** 当前 Dock 内多个面板同时显示时，用户无法知道哪个是激活面板。

**设计:**

```css
/* Dock 内当前激活的面板 */
.dock-panel.active {
  box-shadow: 0 0 0 1.5px var(--color-accent),
              0 0 12px rgba(59, 130, 246, 0.2);
  z-index: 5;
}

/* 底部微妙的 glow 条 */
.dock-panel.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--color-accent), transparent);
  opacity: 0.6;
}
```

**实现:** 在 `DockPanel.vue` 或 `DockLayout.vue` 中，监听当前聚焦的面板（通过点击或 tab 切换），给对应面板添加 `active` class。当面板失焦时移除。

### 3.6 间距系统优化

**目标:** 增加面板内部呼吸感，避免信息拥挤。

| 变量名 | 当前值 | 新值 | 说明 |
|--------|--------|------|------|
| `--panel-padding` | 10px | 12px | 面板内边距 |
| `--space-xs` | 4px | 4px | 保持不变 |
| `--space-sm` | 6px | 8px | 小间距 |
| `--space-md` | 10px | 12px | 中间距 |
| `--space-lg` | 16px | 20px | 大间距 |
| `--space-xl` | 24px | 32px | 超大间距 |
| `--card-padding` | 12px | 14px | 卡片内边距 |
| `--radius-sm` | 4px | 4px | 保持不变 |
| `--radius-md` | 6px | 6px | 保持不变 |
| `--radius-lg` | 10px | 12px | 大圆角，卡片等 |

---

## 4. Phase 3：交互优化（P1）

### 4.1 Tab 拖拽视觉反馈

**目标:** 当前拖拽仅 `opacity: 0.4`，缺少明确的放置区域指示。

**设计:**

```css
/* 被拖拽的 tab */
.dock-tab.dragging {
  opacity: 0.3;
  transform: scale(0.95);
  cursor: grabbing;
}

/* 放置指示线（出现在拖拽经过的 tab 之间） */
.dock-tab-drop-indicator {
  width: 2px;
  height: 24px;
  background: var(--color-accent);
  border-radius: 1px;
  box-shadow: 0 0 6px var(--color-accent);
  animation: drop-pulse 1.5s ease infinite;
  flex-shrink: 0;
}

@keyframes drop-pulse {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}
```

**实现:** 在 `DockTab.vue` 的拖拽逻辑中，当拖拽元素经过其他 tab 时，计算插入位置，动态插入 `.dock-tab-drop-indicator` 元素。放置时移除指示器。

### 4.2 EmptyState 增强

**目标:** 当前空状态只有图标+标题，缺少引导性操作。

**设计:**

```css
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  animation: empty-enter 0.4s ease-out;
}

@keyframes empty-enter {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.empty-state-icon {
  width: 48px;
  height: 48px;
  opacity: 0.4;
  margin-bottom: 16px;
}

.empty-state-title {
  font-size: var(--font-sm);
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.empty-state-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin-bottom: 20px;
  text-align: center;
  max-width: 240px;
}

.empty-state-actions {
  display: flex;
  gap: 8px;
}

.empty-state-actions .btn {
  padding: 6px 14px;
  font-size: var(--font-xs);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.empty-state-actions .btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
  border-color: var(--color-border-hover);
}

.empty-state-actions .btn-primary {
  background: var(--color-accent-soft);
  border-color: var(--color-accent);
  color: var(--color-accent);
}
```

**组件 API 变更:**

```typescript
interface EmptyStateProps {
  icon: string        // 图标名称
  title: string      // 标题
  description?: string // 说明文字（可选）
  actions?: {         // 操作按钮（可选）
    label: string
    primary?: boolean
    onClick: () => void
  }[]
}
```

**实现:** 修改 `EmptyState.vue` 组件，接受新的 props。更新所有使用 EmptyState 的面板，传入合适的 actions（如「刷新数据」、「打开面板」）。

### 4.3 面板切换微动画

**目标:** 当前面板切换内容直接「闪现」，增加 fade 过渡。

**设计:**

```css
/* 面板内容容器 */
.panel-content {
  transition: opacity 0.15s ease;
}

/* 切换中的状态（由 JS 在切换开始时添加，切换结束后移除） */
.panel-content.transitioning {
  opacity: 0.5;
}

/* 新内容入场 */
.panel-content.entering {
  animation: panel-enter 0.2s ease-out;
}

@keyframes panel-enter {
  from { opacity: 0; }
  to { opacity: 1; }
}
```

**实现:** 在 `DockPanel.vue` 或各面板组件中，当面板数据/状态变化时，添加 `transitioning` class 0.1s，然后移除并添加 `entering` class 0.2s。注意：面板内部数据刷新（如自动刷新）不应触发此动画，只在面板切换（如 tab 切换、市场切换）时触发。

### 4.4 PanelTable 表头优化

**目标:** 取消无意义的 uppercase，增加表头与内容的分隔。

**变更:**

1. 移除 `frontend/src/terminal/components/panel/PanelTable.vue` 中 `.table-header-row` 的 `text-transform: uppercase`
2. 增加表头上下 padding：`padding: 6px 0`（从 4px 增加）
3. 增加表头与表格体的分隔：`border-bottom: 1.5px solid var(--color-border-strong)`（从 1px 增加）
4. 表头文字颜色使用 `--color-text-secondary`（不是 primary），通过字重（500）和大小（11px）建立层级

---

## 5. Phase 4：品牌化（P2）

### 5.1 细网格背景（仅 Dark 模式）

**目标:** 在纯黑背景上增加 subtle 的网格线，传递专业终端感。

**设计:**

```css
.app.dark {
  background-image: 
    linear-gradient(rgba(255, 255, 255, 0.015) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.015) 1px, transparent 1px);
  background-size: 40px 40px;
  background-position: center top;
}
```

**说明:** 透明度仅 1.5%，不仔细看几乎不可见，但打破了纯黑的单调，增加空间纵深感。网格大小 40px，与 Bloomberg Terminal 的网格密度类似。

**Light 模式:** 不添加网格背景（不需要）。

### 5.2 Header 和 StatusBar 精致分隔线

**目标:** 当前 StatusBar 的分隔线已不错，但可以让 Header 和 StatusBar 的分隔线更精致，形成视觉呼应。

**StatusBar 分隔线（优化）:**

```css
.status-bar::before {
  content: '';
  position: absolute;
  top: -1px;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(59, 130, 246, 0.2) 15%,
    rgba(59, 130, 246, 0.5) 50%,
    rgba(59, 130, 246, 0.2) 85%,
    transparent 100%
  );
  opacity: 0.8;
}
```

**Header 分隔线（新增）:**

```css
.main-header::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(59, 130, 246, 0.15) 20%,
    rgba(59, 130, 246, 0.35) 50%,
    rgba(59, 130, 246, 0.15) 80%,
    transparent 100%
  );
  opacity: 0.6;
}
```

**说明:** 使用 accent 色（蓝色）的渐变分隔线，中间亮两边暗，形成微妙的「光带」效果。透明度控制在 0.3-0.5，不喧宾夺主。

### 5.3 KPI 数据变化动画

**目标:** 当数据更新时，数字不是直接跳变，而是有短暂的闪光效果，让用户感知到「数据更新了」。

**设计:**

```css
/* 数字变化时的短暂高亮 */
.number-changed {
  animation: number-flash 0.6s ease-out;
}

@keyframes number-flash {
  0% { 
    color: var(--color-accent); 
    text-shadow: 0 0 8px rgba(59, 130, 246, 0.3); 
  }
  100% { 
    color: inherit; 
    text-shadow: none; 
  }
}
```

**实现:**

1. 在 `PanelCard.vue` 中，监听 `value` 和 `change` props 的变化
2. 当数值变化时，给数字元素添加 `number-changed` class
3. 动画结束后（0.6s）移除 class
4. 涨跌颜色（红/绿）在动画结束后恢复正常

**触发条件:**
- 自动刷新时数据变化
- 手动刷新后数据变化
- 市场切换时数据变化

**不触发:**
- 面板首次加载（使用骨架屏，不需要闪光）
- 相同数值（通过对比前后值判断）

---

## 6. 一致性规范（全局）

### 6.1 border-radius 使用规范

所有组件必须使用以下设计令牌，禁止硬编码：

| 场景 | 变量 | 值 |
|------|------|-----|
| 按钮、标签、小元素 | `--radius-sm` | 4px |
| 输入框、卡片、tab | `--radius-md` | 6px |
| 大卡片、面板、弹窗 | `--radius-lg` | 12px |
| 圆形元素（头像、状态点） | `--radius-full` | 9999px |

**需要修复的硬编码位置:**
- `GovDataPanel.vue` 中 `indicator-card` 的 `border-radius: 8px` → 改为 `var(--radius-md)` 或 `var(--radius-lg)`
- `WelcomePanel.vue` 中 `border-radius: 12px` → 改为 `var(--radius-lg)`
- 所有其他面板中硬编码的 `border-radius: 6px/8px/10px` → 统一使用变量

### 6.2 面板卡片统一使用 PanelCard

**目标:** 当前 MarketOverview 使用 PanelCard，GovData 自己写卡片，Watchlist 使用 flex row，风格不一致。

**规范:** 所有面板中的数据卡片（如指数卡片、指标卡片）必须使用 `PanelCard` 组件，或基于 `PanelCard` 的样式类。禁止每个面板自己写卡片样式。

**PanelCard 组件 API:**

```typescript
interface PanelCardProps {
  title: string           // 卡片标题
  value: number | string  // 主数值
  change?: number         // 变化值（可选，用于显示涨跌颜色）
  format?: 'price' | 'percent' | 'number' // 格式化类型
  sparkline?: number[]    // 迷你走势图（可选）
  subtitle?: string       // 副标题（可选）
  size?: 'sm' | 'md' | 'lg' // 卡片大小
  clickable?: boolean     // 是否可点击
  active?: boolean        // 是否激活状态
}
```

**需要统一的面板:**
- `GovDataPanel.vue` 的 indicator-card → 使用 PanelCard
- `WatchlistPanel.vue` 的 watchlist-row → 使用 PanelCard 或提取为 ListItem 组件
- `MarketOverviewPanel.vue` 已使用 PanelCard，保持不变

### 6.3 阴影系统

**目标:** 当前阴影使用较少，需要建立一套微妙的阴影系统用于层级区分。

**新增变量:**

```css
:root {
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  --shadow-focus: 0 0 0 2px rgba(59, 130, 246, 0.2);
  --shadow-glow: 0 0 12px rgba(59, 130, 246, 0.15);
}
```

**使用场景:**
- `--shadow-sm`: 按钮、小卡片悬浮
- `--shadow-md`: 下拉菜单、tooltip、悬浮面板
- `--shadow-lg`: 弹窗、modal
- `--shadow-focus`: 输入框、按钮聚焦态
- `--shadow-glow`: 面板激活态（DockPanel.active）

---

## 7. 验收标准

### 7.1 Phase 1 验收

- [ ] `--color-success-soft`、`--color-danger-soft`、`--color-warning-soft` 在 light 和 dark 模式下都有定义
- [ ] 所有 11 个 `--cat-*` 变量在 light 和 dark 模式下都有定义
- [ ] WelcomePanel 的分类图标颜色正确显示
- [ ] StatusBar 的在线/离线徽章背景色正确显示
- [ ] App.vue 的 div.app 上不再挂载 theme 和 density 类
- [ ] 所有主题切换功能正常

### 7.2 Phase 2 验收

- [ ] Dark 模式下 app 背景 `#090d16` 和面板背景 `#161f2e` 对比度可感知
- [ ] 边框在 dark 模式下可见，分隔作用明确
- [ ] 基础字号 14px，表头 11px，无 uppercase
- [ ] 文字颜色层级清晰：primary → secondary → tertiary → disabled
- [ ] Dock 内点击面板后，该面板有 accent 色边框和底部 glow 条
- [ ] 切换面板时，聚焦态正确转移
- [ ] Light 模式同步调整，无回归问题

### 7.3 Phase 3 验收

- [ ] Tab 拖拽时有 drop indicator 指示放置位置
- [ ] EmptyState 组件显示图标、标题、说明、操作按钮
- [ ] 面板切换时有 0.15s fade 过渡动画
- [ ] PanelTable 表头无 uppercase，上下 padding 增加，border-bottom 更明显
- [ ] 动画不卡顿，不影响数据刷新性能

### 7.4 Phase 4 验收

- [ ] Dark 模式下 app 背景有 subtle 网格线（40px 间隔，1.5% 透明度）
- [ ] Header 和 StatusBar 有渐变分隔线
- [ ] KPI 数据变化时有 0.6s 闪光动画
- [ ] 所有动画结束后恢复原始状态，无残留样式
- [ ] Light 模式下无网格背景，无多余效果

### 7.5 全局验收

- [ ] 所有 border-radius 使用设计令牌，无硬编码
- [ ] 所有面板卡片使用 PanelCard 或统一样式
- [ ] 主题切换（light/dark）无闪烁
- [ ] 密度切换（compact/normal/comfortable）正常
- [ ] 红涨绿跌（CN）和绿涨红跌（US）颜色方案正常
- [ ] 无明显性能下降（动画使用 CSS transform/opacity，不触发 layout）
- [ ] 无视觉回归（所有面板正常显示，无错位、无颜色异常）

---

## 8. 风险与回滚

### 8.1 风险

1. **CSS 变量大规模变更可能导致回归:** 修改 `--color-bg-panel` 等基础变量会影响所有面板。
   - **缓解:** 每阶段结束后运行完整的前端构建和视觉检查。

2. **字号增大可能导致布局溢出:** 14px 基础字号可能使某些紧凑布局溢出。
   - **缓解:** 同步增加面板 padding 和表格行高，给内容更多空间。

3. **动画可能触发性能问题:** 在数据密集面板中，动画可能卡顿。
   - **缓解:** 所有动画使用 CSS `transform` 和 `opacity`（GPU 加速），避免触发 layout。在 low-end 设备上测试。

### 8.2 回滚策略

每阶段结束后提交一个独立的 commit。如果发现问题，可以单独回滚该阶段的 commit，不影响其他阶段。

---

## 9. 附录

### 9.1 相关文件清单

**主题系统:**
- `frontend/src/assets/styles/themes.css` — 主主题文件
- `frontend/src/assets/styles/variables.css` — 变量定义（如果存在）
- `frontend/src/App.vue` — 根组件，theme class 绑定

**布局组件:**
- `frontend/src/terminal/StatusBar.vue` — 状态栏
- `frontend/src/terminal/Header.vue` — 顶部栏（如果存在）
- `frontend/src/terminal/DockLayout.vue` — Dock 布局
- `frontend/src/terminal/DockPanel.vue` — Dock 面板容器
- `frontend/src/terminal/DockTab.vue` — Dock Tab

**通用组件:**
- `frontend/src/terminal/components/panel/PanelCard.vue` — 卡片组件
- `frontend/src/terminal/components/panel/PanelTable.vue` — 表格组件
- `frontend/src/terminal/components/panel/PanelHeader.vue` — 面板头部
- `frontend/src/terminal/components/panel/EmptyState.vue` — 空状态
- `frontend/src/terminal/components/panel/LoadingState.vue` — 加载状态
- `frontend/src/terminal/components/panel/SignalBadge.vue` — 信号徽章

**面板组件:**
- `frontend/src/terminal/panels/MarketOverviewPanel.vue`
- `frontend/src/terminal/panels/GovDataPanel.vue`
- `frontend/src/terminal/panels/WatchlistPanel.vue`
- `frontend/src/terminal/panels/WelcomePanel.vue`
- 其他所有面板组件

**Store:**
- `frontend/src/stores/theme.ts` — 主题 store（如果存在）
- `frontend/src/stores/session.ts` — 会话 store

### 9.2 颜色参考

**Dark 模式完整色板:**

| 用途 | 颜色值 |
|------|--------|
| App 背景 | `#090d16` |
| 面板背景 | `#161f2e` |
| 卡片背景 | `#1a2638` |
| 悬浮背景 | `#1e3248` |
| 选中背景 | `#223450` |
| 边框 | `#2a3e5f` |
| 强边框 | `#334b73` |
| 悬浮边框 | `#3d5a85` |
| 主文字 | `#f1f5f9` |
| 次文字 | `#cbd5e1` |
| 辅助文字 | `#94a3b8` |
| 禁用文字 | `#64748b` |
| 上涨（CN） | `#ef4444` |
| 下跌（CN） | `#22c55e` |
| 上涨（US） | `#22c55e` |
| 下跌（US） | `#ef4444` |
| Accent | `#3b82f6` |
| Success | `#22c55e` |
| Danger | `#ef4444` |
| Warning | `#f59e0b` |

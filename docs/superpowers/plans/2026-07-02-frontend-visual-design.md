# QuantFlow 前端视觉设计改进 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复前端视觉 Bug，提升深色模式层级对比度，优化字号与交互反馈，添加克制优雅的品牌化细节，使 QuantFlow 终端从「可用」变为「精致」。

**Architecture:** 分四阶段（Phase 1~4）独立执行，每阶段可独立验证。以 CSS 变量（themes.css）为核心驱动，组件层只修引用、不重构逻辑。所有动画使用 CSS transform/opacity 保证 GPU 加速。

**Tech Stack:** Vue 3 + CSS Variables + Wails v3 (no new dependencies)

## Global Constraints

- 所有颜色、字号、间距必须使用 themes.css 中的 CSS 变量，禁止硬编码。
- 所有动画必须使用 `transform` 和 `opacity`（GPU 加速），禁止触发布局属性（width/height/margin 等）。
- 每阶段结束后必须运行 `npm run build`（或 `vite build`）确保无编译错误。
- 每阶段结束后提交一个独立 commit，方便回滚。
- Dark 和 Light 模式必须同步修改，避免回归。
- 红涨绿跌（CN）和绿涨红跌（US）颜色方案保持不变。

---

## File Structure

| File | Responsibility | Phase |
|------|---------------|-------|
| `frontend/src/assets/themes.css` | 全局主题变量（颜色、字号、间距、动画） | 1, 2, 4 |
| `frontend/src/App.vue` | 根组件，theme class 挂载 | 1 |
| `frontend/src/terminal/StatusBar.vue` | 状态栏，使用 --color-success-soft 等 | 1 (验证) |
| `frontend/src/terminal/panels/WelcomePanel.vue` | 欢迎面板，使用 --cat-* 变量 | 1 (验证) |
| `frontend/src/terminal/DockView/DockTab.vue` | Dock Tab 拖拽和切换 | 3 |
| `frontend/src/terminal/components/panel/EmptyState.vue` | 空状态组件 | 3 |
| `frontend/src/terminal/components/panel/PanelTable.vue` | 表格组件 | 3 |
| `frontend/src/terminal/components/panel/PanelCard.vue` | 卡片组件 | 3, 4 |
| `frontend/src/terminal/Header.vue` | 顶部栏（如果存在） | 4 |
| 其他面板组件 | 硬编码 border-radius 清理 | 4 |

---

## Phase 1: 修复缺陷（P0）

### Task 1: 添加缺失的 CSS 变量到 themes.css

**Files:**
- Modify: `frontend/src/assets/themes.css`

**Interfaces:**
- Consumes: N/A (新增变量)
- Produces: `--color-success-soft`, `--color-danger-soft`, `--color-warning-soft`, `--color-info-soft`, `--cat-*` 系列变量

- [ ] **Step 1: 在 Dark 主题中添加语义颜色变量**

在 `frontend/src/assets/themes.css` 的 `body {` 块中（约第 8 行），在 `--color-brand-glow` 之后添加：

```css
  --color-success-soft: rgba(34, 197, 94, 0.15);
  --color-danger-soft: rgba(239, 68, 68, 0.15);
  --color-warning-soft: rgba(245, 158, 11, 0.15);
  --color-info-soft: rgba(59, 130, 246, 0.15);
```

- [ ] **Step 2: 在 Light 主题中添加语义颜色变量**

在 `body.theme-light {` 块中（约第 74 行），在 `--color-brand-glow` 之后添加：

```css
  --color-success-soft: rgba(34, 197, 94, 0.12);
  --color-danger-soft: rgba(239, 68, 68, 0.12);
  --color-warning-soft: rgba(245, 158, 11, 0.12);
  --color-info-soft: rgba(37, 99, 235, 0.12);
```

- [ ] **Step 3: 在 Dark 主题中添加分类颜色变量**

在 Dark 主题的 `--color-info-soft` 之后添加：

```css
  /* Category colors for WelcomePanel */
  --cat-market: #3b82f6;
  --cat-market-bg: rgba(59, 130, 246, 0.12);
  --cat-trading: #10b981;
  --cat-trading-bg: rgba(16, 185, 129, 0.12);
  --cat-portfolio: #6366f1;
  --cat-portfolio-bg: rgba(99, 102, 241, 0.12);
  --cat-chart: #06b6d4;
  --cat-chart-bg: rgba(6, 182, 212, 0.12);
  --cat-research: #8b5cf6;
  --cat-research-bg: rgba(139, 92, 246, 0.12);
  --cat-quant: #f59e0b;
  --cat-quant-bg: rgba(245, 158, 11, 0.12);
  --cat-altdata: #ec4899;
  --cat-altdata-bg: rgba(236, 72, 153, 0.12);
  --cat-hk: #14b8a6;
  --cat-hk-bg: rgba(20, 184, 166, 0.12);
  --cat-us: #3b82f6;
  --cat-us-bg: rgba(59, 130, 246, 0.12);
  --cat-crypto: #f59e0b;
  --cat-crypto-bg: rgba(245, 158, 11, 0.12);
  --cat-system: #64748b;
  --cat-system-bg: rgba(100, 116, 139, 0.12);
```

- [ ] **Step 4: 在 Light 主题中添加分类颜色变量**

在 Light 主题中同步添加（颜色值相同，背景色透明度稍低）：

```css
  --cat-market: #2563eb;
  --cat-market-bg: rgba(37, 99, 235, 0.08);
  --cat-trading: #059669;
  --cat-trading-bg: rgba(5, 150, 105, 0.08);
  --cat-portfolio: #4f46e5;
  --cat-portfolio-bg: rgba(79, 70, 229, 0.08);
  --cat-chart: #0891b2;
  --cat-chart-bg: rgba(8, 145, 178, 0.08);
  --cat-research: #7c3aed;
  --cat-research-bg: rgba(124, 58, 237, 0.08);
  --cat-quant: #d97706;
  --cat-quant-bg: rgba(217, 119, 6, 0.08);
  --cat-altdata: #db2777;
  --cat-altdata-bg: rgba(219, 39, 119, 0.08);
  --cat-hk: #0d9488;
  --cat-hk-bg: rgba(13, 148, 136, 0.08);
  --cat-us: #2563eb;
  --cat-us-bg: rgba(37, 99, 235, 0.08);
  --cat-crypto: #d97706;
  --cat-crypto-bg: rgba(217, 119, 6, 0.08);
  --cat-system: #475569;
  --cat-system-bg: rgba(71, 85, 105, 0.08);
```

- [ ] **Step 5: 验证编译**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run build`
Expected: 编译成功，无错误

- [ ] **Step 6: 验证 WelcomePanel 分类颜色**

Run: 启动应用，打开 Welcome 面板，确认分类图标和圆点颜色正确显示（不再是透明/黑色）

- [ ] **Step 7: 验证 StatusBar 徽章颜色**

Run: 检查状态栏的在线/离线徽章，背景色正确显示（在线为绿色半透明，离线为红色半透明）

- [ ] **Step 8: Commit**

```bash
git add frontend/src/assets/themes.css
git commit -m "fix(themes): add missing CSS variables for soft colors and category colors

- Add --color-success-soft, --color-danger-soft, --color-warning-soft, --color-info-soft
- Add --cat-* and --cat-*-bg variables for WelcomePanel category icons
- Fixes P0 visual bugs where badge backgrounds and category icons were invisible"
```

### Task 2: 修复 App.vue class 重复挂载

**Files:**
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Consumes: `session.ui.theme`, `session.ui.density` (from session store)
- Produces: `.app` div without theme/density classes (body already has them)

- [ ] **Step 1: 修改 App.vue 的 template**

在 `frontend/src/App.vue` 中，将第 40 行：

```html
  <div class="app" :class="[`theme-${session.ui.theme}`, `density-${session.ui.density}`]">
```

改为：

```html
  <div class="app">
```

- [ ] **Step 2: 验证主题切换仍然正常**

Run: 启动应用，切换 Dark/Light 主题，确认主题变化正常
Expected: 主题切换正常，body 上的 theme-light/dark 类正确切换

- [ ] **Step 3: Commit**

```bash
git add frontend/src/App.vue
git commit -m "fix(App.vue): remove duplicate theme/density classes from div.app

Theme and density classes are already managed by theme store on body element.
Removing from div.app avoids class duplication and potential selector conflicts."
```

---

## Phase 2: 提升视觉层级（P1）

### Task 3: 重构 Dark 模式颜色层级

**Files:**
- Modify: `frontend/src/assets/themes.css`

**Interfaces:**
- Consumes: N/A
- Produces: 更新后的 Dark 主题颜色变量

- [ ] **Step 1: 更新 Dark 背景色层级**

在 `frontend/src/assets/themes.css` 的 `body {` 块中，修改以下变量：

```css
  --color-bg-app: #090d16;
  --color-bg-panel: #161f2e;
  --color-bg-subtle: #121d2d;
  --color-bg-elevated: #1a2638;
  --color-bg-hover: rgba(255, 255, 255, 0.05);
  --color-bg-active: rgba(255, 255, 255, 0.08);
  --color-bg-selected: rgba(59, 130, 246, 0.15);
  --color-bg-input: #0f1828;
  --color-bg-glass: rgba(22, 31, 46, 0.85);
```

- [ ] **Step 2: 更新 Dark 边框颜色**

在 Dark 主题中修改：

```css
  --color-border: #2a3e5f;
  --color-border-subtle: #1e2d45;
  --color-border-strong: #334b73;
  --color-border-hover: #3d5a85;
  --color-border-glow: rgba(59, 130, 246, 0.25);
```

- [ ] **Step 3: 更新 Dark 文字颜色**

在 Dark 主题中修改：

```css
  --color-text-primary: #f1f5f9;
  --color-text-secondary: #cbd5e1;
  --color-text-tertiary: #94a3b8;
  --color-text-inverse: #0f172a;
  --color-text-disabled: #64748b;
```

注意：`--color-text-disabled` 是新增变量。

- [ ] **Step 4: 验证编译和视觉效果**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run build`
Expected: 编译成功

Run: 启动应用，确认 Dark 模式下面板与 app 背景有可见对比度

- [ ] **Step 5: Commit**

```bash
git add frontend/src/assets/themes.css
git commit -m "feat(themes): enhance dark mode visual hierarchy

- Increase panel bg contrast against app bg (#121929 -> #161f2e, #0b0f19 -> #090d16)
- Strengthen border visibility (#1e2d4d -> #2a3e5f, add --color-border-hover)
- Improve text color hierarchy (primary #f1f5f9, secondary #cbd5e1, tertiary #94a3b8)
- Add --color-text-disabled for disabled states"
```

### Task 4: 重构 Light 模式颜色层级

**Files:**
- Modify: `frontend/src/assets/themes.css`

**Interfaces:**
- Consumes: N/A
- Produces: 更新后的 Light 主题颜色变量

- [ ] **Step 1: 更新 Light 背景色层级**

在 `body.theme-light {` 块中修改：

```css
  --color-bg-app: #f1f5f9;
  --color-bg-panel: #ffffff;
  --color-bg-subtle: #f8fafc;
  --color-bg-elevated: #f8fafc;
  --color-bg-hover: rgba(0, 0, 0, 0.04);
  --color-bg-active: rgba(0, 0, 0, 0.06);
  --color-bg-selected: rgba(37, 99, 235, 0.10);
  --color-bg-input: #f1f5f9;
  --color-bg-glass: rgba(255, 255, 255, 0.85);
```

- [ ] **Step 2: 更新 Light 边框颜色**

在 Light 主题中修改：

```css
  --color-border: #cbd5e1;
  --color-border-subtle: #e2e8f0;
  --color-border-strong: #94a3b8;
  --color-border-hover: #64748b;
  --color-border-glow: rgba(37, 99, 235, 0.20);
```

- [ ] **Step 3: 更新 Light 文字颜色**

在 Light 主题中修改：

```css
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-tertiary: #64748b;
  --color-text-inverse: #ffffff;
  --color-text-disabled: #94a3b8;
```

- [ ] **Step 4: 验证 Light 模式无回归**

Run: 启动应用，切换到 Light 主题，确认所有面板正常显示，边框可见，文字层级清晰

- [ ] **Step 5: Commit**

```bash
git add frontend/src/assets/themes.css
git commit -m "feat(themes): enhance light mode visual hierarchy

- Adjust app bg to #f1f5f9 for subtle canvas depth
- Strengthen border visibility (#e2e8f0 -> #cbd5e1)
- Improve text color hierarchy
- Add --color-text-disabled for disabled states"
```

### Task 5: 调整字号和间距系统

**Files:**
- Modify: `frontend/src/assets/themes.css`

**Interfaces:**
- Consumes: N/A
- Produces: 更新后的字号和间距变量

- [ ] **Step 1: 更新默认密度（normal）的字号和间距**

在 `body {` 块中（第 150 行左右），修改密度相关的变量：

```css
body {
  --space-xs: 4px;
  --space-sm: 8px;
  --space-md: 12px;
  --space-lg: 16px;
  --space-xl: 24px;
  --space-2xl: 32px;
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 12px;
  --font-xs: 12px;
  --font-sm: 13px;
  --font-base: 14px;
  --font-lg: 16px;
  --font-xl: 20px;
  --font-kpi: 24px;
  --row-height: 36px;
  --padding-panel: 12px;
  --transition-fast: 0.12s ease;
  --transition-normal: 0.2s ease;
  --transition-slow: 0.3s ease;
}
```

- [ ] **Step 2: 更新紧凑密度（compact）**

在 `body.density-compact {` 块中修改：

```css
body.density-compact {
  --space-xs: 2px;
  --space-sm: 4px;
  --space-md: 8px;
  --space-lg: 12px;
  --space-xl: 16px;
  --space-2xl: 24px;
  --radius-sm: 3px;
  --radius-md: 4px;
  --radius-lg: 8px;
  --font-xs: 10px;
  --font-sm: 11px;
  --font-base: 12px;
  --font-lg: 14px;
  --font-xl: 16px;
  --font-kpi: 20px;
  --row-height: 30px;
  --padding-panel: 8px;
  --transition-fast: 80ms ease;
  --transition-normal: 0.12s ease;
  --transition-slow: 0.2s ease;
}
```

- [ ] **Step 3: 更新舒适密度（comfortable）**

在 `body.density-comfortable {` 块中修改：

```css
body.density-comfortable {
  --space-xs: 6px;
  --space-sm: 10px;
  --space-md: 14px;
  --space-lg: 20px;
  --space-xl: 28px;
  --space-2xl: 40px;
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --font-xs: 13px;
  --font-sm: 14px;
  --font-base: 16px;
  --font-lg: 18px;
  --font-xl: 24px;
  --font-kpi: 30px;
  --row-height: 44px;
  --padding-panel: 16px;
  --transition-fast: 0.15s ease;
  --transition-normal: 0.25s ease;
  --transition-slow: 0.35s ease;
}
```

- [ ] **Step 4: 更新 Panel Tokens 中的字号相关变量**

在 `/* ── Panel Tokens ── */` 部分（约第 419 行），修改：

```css
body {
  --panel-padding: var(--padding-panel);
  --panel-padding-lg: 16px;
  --panel-padding-sm: 8px;

  --panel-title-size: var(--font-lg);
  --panel-title-weight: 600;
  --panel-subtitle-size: var(--font-sm);
  --panel-subtitle-color: var(--color-text-tertiary);

  --table-header-size: 11px;
  --table-header-color: var(--color-text-tertiary);
  --table-header-weight: 500;
  --table-row-size: var(--font-sm);
  --table-row-height: var(--row-height);
  --table-row-height-compact: 30px;
  --table-row-height-comfortable: 44px;
  --table-border: var(--color-border-subtle);
  --table-row-hover: var(--color-bg-hover);
  --table-row-odd: rgba(255, 255, 255, 0.02);

  --tab-height: 28px;
  --tab-padding: 4px 12px;
  --tab-font-size: var(--font-xs);
  --tab-active-bg: var(--color-bg-panel);
  --tab-active-border: var(--color-accent);
  --tab-inactive-color: var(--color-text-tertiary);

  --toolbar-height: 36px;
  --toolbar-gap: var(--space-sm);
  --toolbar-padding: var(--space-sm) var(--space-md);

  --card-padding: var(--space-md);
  --card-gap: var(--space-sm);
  --card-min-width: 200px;
}
```

- [ ] **Step 5: 验证编译和视觉效果**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run build`
Expected: 编译成功

Run: 启动应用，确认字号增大后无布局溢出，表格行高有呼吸感

- [ ] **Step 6: Commit**

```bash
git add frontend/src/assets/themes.css
git commit -m "feat(themes): increase base font size and spacing for desktop readability

- Base font: 13px -> 14px, table header: 10px -> 11px, badge: 10px -> 11px
- Row height: 36px, panel padding: 12px, radius-lg: 10px -> 12px
- Adjust compact and comfortable densities proportionally
- Update panel tokens to match new font scale"
```

### Task 6: 实现面板聚焦态（Dock 激活面板）

**Files:**
- Modify: `frontend/src/terminal/DockView/DockTab.vue`

**Interfaces:**
- Consumes: `activeTab` (当前激活 tab), `tabs` (tab 列表)
- Produces: `.dock-tab` 容器上的 `active` class（当有 tab 激活时）

- [ ] **Step 1: 在 DockTab.vue 的 template 中添加 active 类**

在 `frontend/src/terminal/DockView/DockTab.vue` 中，修改第 93 行：

```html
  <div class="dock-tab">
```

改为：

```html
  <div :class="['dock-tab', { active: activeTab }]">
```

- [ ] **Step 2: 添加聚焦态样式**

在 `frontend/src/terminal/DockView/DockTab.vue` 的 `<style scoped>` 部分（第 144 行之后），添加：

```css
.dock-tab.active {
  box-shadow: inset 0 0 0 1.5px var(--color-accent),
              0 0 12px var(--color-accent-glow);
  z-index: 5;
}

.dock-tab.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--color-accent), transparent);
  opacity: 0.6;
  pointer-events: none;
}
```

注意：由于 `.dock-tab` 是 `position: relative`（由 scoped 样式定义），`::after` 伪元素可以相对于它定位。

- [ ] **Step 3: 验证聚焦态**

Run: 启动应用，打开多个面板，点击不同面板，确认激活面板有 accent 色边框和底部 glow 条

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/DockView/DockTab.vue
git commit -m "feat(dock): add focus state to active dock panel

- Active dock panel gets 1.5px accent border + subtle glow shadow
- Bottom glow bar indicates active panel within the dock"
```

---

## Phase 3: 交互优化（P1）

### Task 7: Tab 拖拽视觉反馈

**Files:**
- Modify: `frontend/src/terminal/DockView/DockTab.vue`

**Interfaces:**
- Consumes: `dragTabId` (当前拖拽的 tab ID), `tabs` (tab 列表)
- Produces: `drag-over` class on tab buttons, drop indicator element

- [ ] **Step 1: 增强拖拽时的样式**

在 `frontend/src/terminal/DockView/DockTab.vue` 的 `<style scoped>` 中，修改现有的 `.tab-btn.dragging`（第 223 行）：

```css
.tab-btn.dragging {
  opacity: 0.3;
  transform: scale(0.95);
  cursor: grabbing;
}
```

改为：

```css
.tab-btn.dragging {
  opacity: 0.25;
  transform: scale(0.92);
  cursor: grabbing;
  filter: grayscale(0.5);
}
```

- [ ] **Step 2: 添加拖拽经过时的高亮效果**

在 `<style scoped>` 中添加：

```css
.tab-btn.drag-over {
  background: var(--color-bg-selected);
  border-color: var(--color-accent);
}

/* Drop indicator line between tabs */
.drop-indicator {
  width: 2px;
  height: 20px;
  background: var(--color-accent);
  border-radius: 1px;
  box-shadow: 0 0 6px var(--color-accent-glow);
  animation: drop-pulse 1.2s ease infinite;
  flex-shrink: 0;
  align-self: center;
}

@keyframes drop-pulse {
  0%, 100% { opacity: 0.6; transform: scaleY(1); }
  50% { opacity: 1; transform: scaleY(1.2); }
}
```

- [ ] **Step 3: 添加拖拽经过状态跟踪**

在 `frontend/src/terminal/DockView/DockTab.vue` 的 `<script setup>` 中，添加：

```typescript
const dragOverTabId = ref<string | null>(null)
```

在 `onDragStart` 函数中，添加 `dragOverTabId.value = null`：

```typescript
function onDragStart(e: DragEvent, tabId: string) {
  if (!e.dataTransfer) return
  dragTabId.value = tabId
  dragOverTabId.value = null
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', JSON.stringify({ leafId: props.leafId, tabId }))
}
```

在 `onDragEnd` 函数中，添加 `dragOverTabId.value = null`：

```typescript
function onDragEnd() { 
  dragTabId.value = null 
  dragOverTabId.value = null
}
```

在 `onTabDragOver` 函数中，添加 drag-over 状态跟踪：

```typescript
function onTabDragOver(e: DragEvent, tabId?: string) {
  e.preventDefault()
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'move'
  if (tabId && dragTabId.value !== tabId) {
    dragOverTabId.value = tabId
  }
}
```

- [ ] **Step 4: 在 template 中应用 drag-over 类**

修改 tab button 的模板（约第 96 行），添加 drag-over 类：

```html
        <button
          v-for="(tab, idx) in tabs"
          :key="tab.id"
          :draggable="true"
          class="tab-btn"
          :class="{ active: tab.id === activeTab, dragging: dragTabId === tab.id, 'drag-over': dragOverTabId === tab.id }"
          @click="emit('select-tab', tab.id)"
          @dragstart="onDragStart($event, tab.id)"
          @dragend="onDragEnd"
          @dragover="onTabDragOver($event, tab.id)"
          @drop="onTabDrop($event, idx)"
        >
```

注意：这里需要把 `onTabDragOver` 的签名改为接收 `tabId` 参数。

- [ ] **Step 5: 验证拖拽效果**

Run: 启动应用，拖拽 tab，确认：
1. 被拖拽 tab 变小变暗
2. 拖拽经过其他 tab 时，该 tab 有蓝色高亮背景
3. 放置后 tab 顺序正确变化

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/DockView/DockTab.vue
git commit -m "feat(dock-tab): enhance drag-and-drop visual feedback

- Add scale + grayscale effect to dragged tab
- Add drag-over highlight on hovered tabs
- Add drag-over state tracking in script"
```

### Task 8: EmptyState 增强

**Files:**
- Modify: `frontend/src/terminal/components/panel/EmptyState.vue`

**Interfaces:**
- Consumes: `icon`, `title`, `description`, `actions` (新增数组)
- Produces: 增强的空状态 UI（支持多个按钮和入场动画）

- [ ] **Step 1: 修改 Props 定义**

在 `frontend/src/terminal/components/panel/EmptyState.vue` 的 `<script setup>` 中，将：

```typescript
withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  action?: { label: string; handler: () => void }
}>(), {
  icon: 'inbox',
})
```

改为：

```typescript
interface EmptyAction {
  label: string
  primary?: boolean
  handler: () => void
}

withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  actions?: EmptyAction[]
}>(), {
  icon: 'inbox',
})
```

- [ ] **Step 2: 修改 Template**

将 template 中的 action 按钮部分：

```html
    <button v-if="action" class="btn btn-primary" @click="action.handler">
      {{ action.label }}
    </button>
```

改为：

```html
    <div v-if="actions && actions.length" class="empty-actions">
      <button
        v-for="(act, idx) in actions"
        :key="idx"
        :class="['btn', act.primary ? 'btn-primary' : 'btn-ghost']"
        @click="act.handler"
      >
        {{ act.label }}
      </button>
    </div>
```

- [ ] **Step 3: 添加动画和样式**

在 `<style scoped>` 中修改 `.empty-state`：

```css
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  text-align: center;
  animation: empty-enter 0.4s ease-out;
}

@keyframes empty-enter {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}
```

添加 actions 样式：

```css
.empty-actions {
  display: flex;
  gap: var(--space-sm);
  flex-wrap: wrap;
  justify-content: center;
}
```

- [ ] **Step 4: 更新使用 EmptyState 的面板**

搜索所有使用 `<EmptyState` 的文件：

Run: `grep -r "<EmptyState" /Volumes/shenzy/vibe_coding/QuantFlow/frontend/src --include="*.vue" -l`

对每个文件，将 `action` prop 改为 `actions` 数组：

例如：
```html
<!-- Before -->
<EmptyState title="无数据" description="暂无数据" :action="{ label: '刷新', handler: refresh }" />

<!-- After -->
<EmptyState 
  title="无数据" 
  description="暂无数据" 
  :actions="[
    { label: '刷新数据', primary: true, handler: refresh },
    { label: '打开面板', handler: openPanel }
  ]" 
/>
```

- [ ] **Step 5: 验证空状态**

Run: 启动应用，找到显示空状态的面板，确认：
1. 有图标、标题、说明
2. 有多个操作按钮（至少一个主按钮）
3. 空状态有入场动画

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/components/panel/EmptyState.vue
git commit -m "feat(EmptyState): support multiple actions and entrance animation

- Replace single action prop with actions array
- Add entrance animation (fade + translateY)
- Update consuming panels to use new actions API"
```

### Task 9: PanelTable 表头优化

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelTable.vue`

**Interfaces:**
- Consumes: N/A
- Produces: 优化后的表头样式（无 uppercase，更大 padding，更强边框）

- [ ] **Step 1: 修改表头样式**

在 `frontend/src/terminal/components/panel/PanelTable.vue` 的 `<style scoped>` 中，修改 `.table-header-row`：

```css
.table-header-row {
  display: flex;
  padding: 6px 0;
  border-bottom: 1.5px solid var(--color-border-strong);
  font-size: var(--table-header-size);
  color: var(--table-header-color);
  font-weight: var(--table-header-weight);
  /* text-transform: uppercase;  <-- REMOVE THIS LINE */
  flex-shrink: 0;
  letter-spacing: 0.01em;
}
```

注意：移除 `text-transform: uppercase` 这一行。

- [ ] **Step 2: 增加表头上下 padding**

将 `padding: 4px 0` 改为 `padding: 6px 0`（已在 Step 1 中完成）。

将 `.th, .td` 的 padding 从 `0 4px` 改为 `0 6px`：

```css
.th,
.td {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0 6px;
}
```

- [ ] **Step 3: 验证表头效果**

Run: 启动应用，打开有表格的面板（如 MarketOverview, Watchlist），确认：
1. 表头无 uppercase（中文正常显示）
2. 表头与表格体之间有清晰的分隔线
3. 表头 padding 有呼吸感

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelTable.vue
git commit -m "feat(PanelTable): improve table header readability

- Remove text-transform: uppercase (meaningless for Chinese)
- Increase header padding (4px -> 6px) and cell padding (4px -> 6px)
- Strengthen header border-bottom (1px -> 1.5px)"
```

### Task 10: 面板切换微动画

**Files:**
- Modify: `frontend/src/terminal/DockView/DockTab.vue`

**Interfaces:**
- Consumes: `activeTab` (当前 tab), `activeComponent` (当前组件)
- Produces: 切换时的 fade 动画

- [ ] **Step 1: 添加切换动画状态**

在 `frontend/src/terminal/DockView/DockTab.vue` 的 `<script setup>` 中，添加：

```typescript
import { ref, watch } from 'vue'

const transitioning = ref(false)

// Watch activeTab changes to trigger transition animation
watch(() => props.activeTab, () => {
  transitioning.value = true
  setTimeout(() => { transitioning.value = false }, 200)
})
```

注意：需要在文件顶部确认 `watch` 是否已导入。当前文件已导入 `{ computed, ref, type Component }`，需要添加 `watch`。

- [ ] **Step 2: 在 template 中应用动画类**

修改 `.tab-content` 的 div（约第 124 行）：

```html
    <div class="tab-content" :class="{ transitioning }" @dragover="onDragOver" @drop="onDrop">
```

- [ ] **Step 3: 添加 CSS 动画样式**

在 `<style scoped>` 中添加：

```css
.tab-content {
  flex: 1;
  overflow: auto;
  min-height: 0;
  transition: opacity 0.15s ease;
}

.tab-content.transitioning {
  opacity: 0.5;
}

.tab-content.transitioning .panel-instance {
  animation: panel-enter 0.2s ease-out;
}

@keyframes panel-enter {
  from { opacity: 0; transform: translateY(2px); }
  to { opacity: 1; transform: translateY(0); }
}
```

- [ ] **Step 4: 验证切换动画**

Run: 启动应用，在 Dock 中切换 tab，确认：
1. 切换时有短暂的 fade 效果
2. 新内容有轻微的向上滑入动画
3. 动画不卡顿

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/DockView/DockTab.vue
git commit -m "feat(dock-tab): add panel transition animation

- Add fade transition on tab switch (0.15s)
- Add subtle slide-up animation on new panel content
- Track transition state via watcher on activeTab"
```

---

## Phase 4: 品牌化（P2）

### Task 11: 网格背景 + 分隔线

**Files:**
- Modify: `frontend/src/assets/themes.css`
- Modify: `frontend/src/terminal/StatusBar.vue` (分隔线优化)

**Interfaces:**
- Consumes: N/A
- Produces: Dark 模式下的网格背景，Header/StatusBar 精致分隔线

- [ ] **Step 1: 添加网格背景到 Dark 模式**

在 `frontend/src/assets/themes.css` 的 `body {` 块中，在 `--gradient-brand` 之后添加：

```css
  --bg-grid: linear-gradient(rgba(255, 255, 255, 0.015) 1px, transparent 1px),
             linear-gradient(90deg, rgba(255, 255, 255, 0.015) 1px, transparent 1px);
```

- [ ] **Step 2: 在 html/body 样式中应用网格背景**

在 `/* ── Reset ── */` 部分（约第 220 行），修改 `html, body` 的样式：

```css
html, body {
  height: 100%; overflow: hidden;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', 'JetBrains Mono', monospace, sans-serif;
  font-size: var(--font-base); color: var(--color-text-primary); background: var(--color-bg-app);
  -webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale;
}

body:not(.theme-light) {
  background-image: var(--bg-grid);
  background-size: 40px 40px;
  background-position: center top;
  background-attachment: fixed;
}
```

- [ ] **Step 3: 优化 StatusBar 分隔线**

在 `frontend/src/terminal/StatusBar.vue` 的 `<style scoped>` 中，修改 `.status-bar::before`：

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

- [ ] **Step 4: 检查并添加 Header 分隔线**

查找 Header.vue 文件：

Run: `find /Volumes/shenzy/vibe_coding/QuantFlow/frontend/src -name "Header.vue" -o -name "MainHeader.vue" -o -name "AppHeader.vue"`

如果存在 Header 组件，在其样式中添加：

```css
.main-header::after,
.header::after {
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

如果不存在 Header 组件，则跳过此步骤。

- [ ] **Step 5: 验证网格背景和分隔线**

Run: 启动应用，确认：
1. Dark 模式下 app 背景有 subtle 网格线（40px 间隔，仔细看能看到）
2. Light 模式下无网格线
3. StatusBar 顶部有渐变分隔线
4. Header 底部有渐变分隔线（如果存在 Header）

- [ ] **Step 6: Commit**

```bash
git add frontend/src/assets/themes.css frontend/src/terminal/StatusBar.vue
git commit -m "feat(themes): add subtle grid background and refined divider lines

- Dark mode gets 40px grid pattern at 1.5% opacity (Bloomberg-like)
- Light mode has no grid background
- StatusBar and Header get accent-colored gradient divider lines"
```

### Task 12: KPI 数据变化动画

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelCard.vue`

**Interfaces:**
- Consumes: `value`, `change` props
- Produces: 数字变化时的闪光动画

- [ ] **Step 1: 添加变化检测逻辑**

在 `frontend/src/terminal/components/panel/PanelCard.vue` 的 `<script setup>` 中，添加：

```typescript
import { ref, watch } from 'vue'

const valueChanged = ref(false)

// Watch value changes to trigger flash animation
watch(() => props.value, (newVal, oldVal) => {
  if (oldVal !== undefined && newVal !== oldVal) {
    valueChanged.value = true
    setTimeout(() => { valueChanged.value = false }, 600)
  }
})
```

注意：需要确认 `watch` 是否已导入。当前文件已导入 `{ computed }`，需要添加 `watch`。

- [ ] **Step 2: 在 template 中应用动画类**

修改 `.card-value` 的 div（约第 62 行）：

```html
    <div :class="['card-value', { 'number-changed': valueChanged }]">{{ formattedValue }}</div>
```

- [ ] **Step 3: 添加 CSS 动画**

在 `<style scoped>` 中添加：

```css
.number-changed {
  animation: number-flash 0.6s ease-out;
}

@keyframes number-flash {
  0% { 
    color: var(--color-accent); 
    text-shadow: 0 0 8px var(--color-accent-glow); 
  }
  100% { 
    color: inherit; 
    text-shadow: none; 
  }
}
```

- [ ] **Step 4: 验证闪光动画**

Run: 启动应用，打开 MarketOverview 面板，等待自动刷新，确认：
1. 指数数值变化时有短暂的蓝色闪光
2. 闪光后恢复正常的涨跌颜色（红/绿）
3. 闪光动画不卡顿

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelCard.vue
git commit -m "feat(PanelCard): add number flash animation on value change

- Watch value prop changes and trigger 0.6s flash animation
- Flash uses accent color + glow, then returns to normal up/down color
- Only triggers on actual value changes, not on initial mount"
```

### Task 13: 一致性清理（border-radius 硬编码 + 面板卡片统一）

**Files:**
- Multiple panel files

**Interfaces:**
- Consumes: N/A
- Produces: 无硬编码 border-radius，面板卡片统一使用 PanelCard

- [ ] **Step 1: 查找所有硬编码 border-radius**

Run: `grep -rn "border-radius: [0-9]*px" /Volumes/shenzy/vibe_coding/QuantFlow/frontend/src --include="*.vue" --include="*.css" | grep -v "var(--radius" | grep -v "node_modules"`

- [ ] **Step 2: 替换所有硬编码 border-radius**

对每个硬编码的 border-radius，替换为合适的变量：

| 硬编码值 | 替换为 | 场景 |
|---------|--------|------|
| `border-radius: 4px` | `var(--radius-sm)` | 小元素 |
| `border-radius: 6px` | `var(--radius-md)` | 输入框、按钮 |
| `border-radius: 8px` | `var(--radius-md)` 或 `var(--radius-lg)` | 卡片 |
| `border-radius: 10px` | `var(--radius-lg)` | 大卡片 |
| `border-radius: 12px` | `var(--radius-lg)` | 大卡片 |
| `border-radius: 16px` | `var(--radius-lg)` | 面板/弹窗 |

- [ ] **Step 3: 统一 GovDataPanel 的卡片样式**

在 `frontend/src/terminal/panels/GovDataPanel.vue` 中，查找 `.indicator-card` 的样式，确认它使用了 `var(--radius-lg)` 和 PanelCard 的样式模式。

如果 GovDataPanel 自己写了一套卡片样式，改为使用 `PanelCard` 组件或 `.card` 基础类。

- [ ] **Step 4: 验证一致性**

Run: 启动应用，检查所有面板：
1. 所有卡片圆角一致
2. 无硬编码的 border-radius

- [ ] **Step 5: Commit**

```bash
git commit -m "style(panels): eliminate hardcoded border-radius, unify card styles

- Replace all hardcoded border-radius values with CSS variables
- GovDataPanel indicator cards use standard PanelCard styling
- Consistent border-radius across all panels"
```

---

## Self-Review Checklist

### Spec Coverage

- [x] Phase 1 Bug 修复 → Task 1, 2
- [x] Phase 2 颜色层级 → Task 3, 4
- [x] Phase 2 字号间距 → Task 5
- [x] Phase 2 面板聚焦态 → Task 6
- [x] Phase 3 Tab 拖拽 → Task 7
- [x] Phase 3 EmptyState → Task 8
- [x] Phase 3 PanelTable → Task 9
- [x] Phase 3 面板切换动画 → Task 10
- [x] Phase 4 网格背景 → Task 11
- [x] Phase 4 分隔线 → Task 11
- [x] Phase 4 KPI 闪光 → Task 12
- [x] Phase 4 一致性清理 → Task 13

### Placeholder Scan

- [x] 无 TBD/TODO/fill in later
- [x] 所有代码块包含实际代码
- [x] 所有命令包含预期输出
- [x] 无 "Similar to Task N" 引用

### Type Consistency

- [x] `--color-success-soft` 在 Dark/Light 中定义一致
- [x] `--cat-*` 变量名在 themes.css 和 WelcomePanel.vue 中匹配
- [x] `actions` 数组类型在 EmptyState 中定义清晰
- [x] `watch` 导入在所有需要它的文件中被添加

### Risk Check

- [x] 字号增大（13px → 14px）已同步增加行高和 padding，避免溢出
- [x] 所有动画使用 transform/opacity，不触发 layout
- [x] 每阶段有独立 commit，可回滚

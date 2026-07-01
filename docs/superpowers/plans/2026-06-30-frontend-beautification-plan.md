# QuantFlow 前端美化执行计划

> 基于 Spec: `docs/specs/2026-06-30-frontend-beautification-plan.md`
> 阶段: Plan → Execute
> 目标: 将 Spec 转化为可逐条执行的 checklist，含文件路径、验收标准、依赖关系

---

## 全局约定

- **所有输出文件**以 `frontend/src/terminal/` 为根目录
- **共享组件**目录：`frontend/src/terminal/components/panel/`
- **themes.css** 路径：`frontend/src/assets/themes.css`
- **验收命令**：每个任务完成后运行指定 grep 命令验证
- **回滚策略**：Git 分支 `feat/ui-unification`，每 Phase 完成一个 commit

---

## Phase 1: 基础设施（共享组件库 + Token 增强）

> **目标**：建立所有面板共享的组件库，增强 themes.css 的 token 覆盖。所有后续 Phase 依赖此 Phase。
> **时间**：1.5 天 | **Commit**：`feat: add panel shared components & theme tokens`

---

### Task 1.1: 增强 themes.css — 新增 Design Token

**文件**：`frontend/src/assets/themes.css`

**具体工作**：
在现有文件末尾（`/* ── Animation utilities ── */` 之后）追加以下 token 段：

```css
/* ── Panel Tokens ─────────────────────────────────────────────── */
body {
  --panel-padding: var(--padding-panel);
  --panel-padding-lg: 16px;
  --panel-padding-sm: 8px;

  --panel-title-size: var(--font-lg);
  --panel-title-weight: 600;
  --panel-subtitle-size: var(--font-sm);
  --panel-subtitle-color: var(--color-text-tertiary);

  --table-header-size: var(--font-xs);
  --table-header-color: var(--color-text-tertiary);
  --table-header-weight: 600;
  --table-row-size: var(--font-sm);
  --table-row-height: var(--row-height);
  --table-row-height-compact: 28px;
  --table-row-height-comfortable: 44px;
  --table-border: var(--color-border-subtle);
  --table-row-hover: var(--color-bg-hover);
  --table-row-odd: rgba(255,255,255,0.02);

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

body.theme-light {
  --table-row-odd: rgba(0,0,0,0.02);
}
```

**注意**：不要修改现有 token，仅追加新段，避免 regression。

**验收标准**：
```bash
# 验证新 token 已追加
grep -n "table-row-odd" frontend/src/assets/themes.css | head -2
# 预期输出包含 dark 和 light 两种定义

grep -n "panel-title-size" frontend/src/assets/themes.css
# 预期输出包含定义

grep -n "tab-height" frontend/src/assets/themes.css
# 预期输出包含定义
```

**依赖**：无

---

### Task 1.2: 创建 `PanelHeader.vue` — 统一面板头部

**文件**：`frontend/src/terminal/components/panel/PanelHeader.vue`

**接口设计**：

```vue
<template>
  <div class="panel-header">
    <div class="header-left">
      <h3 v-if="title" class="panel-title">{{ title }}</h3>
      <span v-if="subtitle" class="panel-subtitle">{{ subtitle }}</span>
    </div>
    <div v-if="tabs?.length" class="header-tabs">
      <button
        v-for="tab in tabs" :key="tab.key"
        :class="['tab-btn', { active: activeTab === tab.key }]"
        @click="$emit('tabChange', tab.key)"
      >{{ tab.label }}</button>
    </div>
    <div v-if="controls?.length" class="header-controls">
      <button
        v-for="ctrl in controls" :key="ctrl.icon"
        :class="['btn btn-ghost', { loading: ctrl.loading }]"
        @click="ctrl.action"
        :title="ctrl.title"
      >
        <span v-if="ctrl.icon" class="icon" v-html="getIcon(ctrl.icon)" />
        <span v-if="ctrl.label">{{ ctrl.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { getIcon } from '@/lib/icons'
interface Tab { key: string; label: string }
interface Control { icon?: string; label?: string; title?: string; action: () => void; loading?: boolean }
withDefaults(defineProps<{
  title?: string
  subtitle?: string
  tabs?: Tab[]
  activeTab?: string
  controls?: Control[]
}>(), { tabs: () => [], controls: () => [] })
defineEmits<{ (e: 'tabChange', key: string): void }>()
</script>

<style scoped>
.panel-header {
  display: flex; align-items: center; gap: var(--space-md);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border);
  min-height: var(--toolbar-height);
  flex-wrap: wrap;
}
.header-left { display: flex; align-items: baseline; gap: var(--space-sm); flex: 1; min-width: 0; }
.panel-title { margin: 0; font-size: var(--panel-title-size); font-weight: var(--panel-title-weight); color: var(--color-text-primary); }
.panel-subtitle { font-size: var(--panel-subtitle-size); color: var(--panel-subtitle-color); }
.header-tabs { display: flex; gap: var(--space-xs); }
.tab-btn {
  padding: var(--tab-padding); height: var(--tab-height); font-size: var(--tab-font-size);
  border: 1px solid transparent; border-radius: var(--radius-md); background: transparent;
  color: var(--tab-inactive-color); cursor: pointer; transition: all var(--transition-fast);
}
.tab-btn:hover { color: var(--color-text-primary); background: var(--color-bg-hover); }
.tab-btn.active { color: var(--color-accent); border-color: var(--tab-active-border); background: var(--tab-active-bg); }
.header-controls { display: flex; gap: var(--space-xs); align-items: center; }
</style>
```

**验收标准**：
1. 文件存在且可编译
2. `PanelHeader` 在 DevTools 中渲染时，所有样式使用 `var(--*)` 而非硬编码
3. 在 WelcomePanel 中临时替换 header 测试，视觉正常

**依赖**：Task 1.1（token 必须先定义）

---

### Task 1.3: 创建 `PanelToolbar.vue` — 统一工具栏

**文件**：`frontend/src/terminal/components/panel/PanelToolbar.vue`

**接口设计**：

```vue
<template>
  <div class="panel-toolbar">
    <div v-if="searchPlaceholder" class="toolbar-search">
      <input
        type="text"
        :placeholder="searchPlaceholder"
        v-model="searchVal"
        @input="$emit('search', searchVal)"
        class="toolbar-input"
      />
    </div>
    <div v-if="filters?.length" class="toolbar-filters">
      <button
        v-for="f in filters" :key="f.key"
        :class="['filter-btn', { active: activeFilter === f.key }]"
        @click="$emit('filterChange', f.key)"
      >{{ f.label }}</button>
    </div>
    <div v-if="actions?.length" class="toolbar-actions">
      <button
        v-for="a in actions" :key="a.label"
        class="btn btn-ghost"
        @click="a.handler"
      >
        <span v-if="a.icon" class="icon" v-html="getIcon(a.icon)" />
        {{ a.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getIcon } from '@/lib/icons'
interface Filter { key: string; label: string }
interface Action { label: string; icon?: string; handler: () => void }
withDefaults(defineProps<{
  searchPlaceholder?: string
  filters?: Filter[]
  activeFilter?: string
  actions?: Action[]
}>(), { filters: () => [], actions: () => [] })
defineEmits<{ (e: 'search', val: string): void; (e: 'filterChange', key: string): void }>()
const searchVal = ref('')
</script>

<style scoped>
.panel-toolbar {
  display: flex; align-items: center; gap: var(--toolbar-gap);
  padding: var(--toolbar-padding); min-height: var(--toolbar-height);
  border-bottom: 1px solid var(--color-border);
  flex-wrap: wrap;
}
.toolbar-input {
  width: 180px; padding: var(--space-xs) var(--space-sm); font-size: var(--font-sm);
}
.toolbar-filters { display: flex; gap: var(--space-xs); }
.filter-btn {
  padding: 2px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-md);
  background: transparent; color: var(--color-text-tertiary); cursor: pointer; font-size: var(--font-xs);
  transition: all var(--transition-fast);
}
.filter-btn:hover { border-color: var(--color-border-strong); color: var(--color-text-primary); }
.filter-btn.active { color: var(--color-accent); border-color: var(--color-accent); background: var(--color-accent-soft); }
.toolbar-actions { display: flex; gap: var(--space-xs); margin-left: auto; }
</style>
```

**验收标准**：
1. 文件存在且可编译
2. 所有样式使用 `var(--*)`

**依赖**：Task 1.1

---

### Task 1.4: 创建 `PanelTable.vue` — 统一表格

**文件**：`frontend/src/terminal/components/panel/PanelTable.vue`

**接口设计**：

```vue
<template>
  <div class="panel-table-wrapper">
    <div class="table-header-row">
      <span
        v-for="col in columns" :key="col.key"
        :class="['th', col.align || 'left']"
        :style="colStyle(col)"
      >{{ col.label }}</span>
    </div>
    <div class="table-body">
      <div
        v-for="(row, idx) in data" :key="rowKey(row, idx)"
        :class="['table-row', { striped: striped && idx % 2 === 1, clickable: !!$attrs.onRowClick }]"
        @click="$emit('rowClick', row)"
      >
        <span
          v-for="col in columns" :key="col.key"
          :class="['td', col.align || 'left', { colorize: col.colorize }]"
          :style="[{ color: col.colorize ? colorize(row[col.key]) : undefined }, colStyle(col)]"
        >{{ formatCell(row, col) }}</span>
      </div>
    </div>
    <LoadingState v-if="loading" type="table" :rows="3" />
  </div>
</template>

<script setup lang="ts">
import LoadingState from './LoadingState.vue'
interface Column {
  key: string; label: string; width?: number; flex?: number;
  align?: 'left' | 'right' | 'center'; format?: 'price' | 'percent' | 'volume' | 'number';
  colorize?: boolean; formatter?: (val: any) => string;
}
withDefaults(defineProps<{
  columns: Column[]; data: any[]; loading?: boolean; striped?: boolean; rowKey?: (row: any, idx: number) => string | number;
}>(), { striped: true, loading: false })
defineEmits<{ (e: 'rowClick', row: any): void }>()
function colStyle(col: Column) {
  const s: Record<string, string> = {}
  if (col.width) s.width = col.width + 'px'; else if (col.flex) s.flex = String(col.flex); else s.flex = '1'
  return s
}
function formatCell(row: any, col: Column) {
  const v = row[col.key]; if (v == null) return '--'
  if (col.formatter) return col.formatter(v)
  switch (col.format) {
    case 'price': return typeof v === 'number' ? v.toFixed(2) : v
    case 'percent': return typeof v === 'number' ? (v >= 0 ? '+' : '') + v.toFixed(2) + '%' : v
    case 'volume': return typeof v === 'number' ? (v >= 1e8 ? (v/1e8).toFixed(2)+'亿' : v >= 1e4 ? (v/1e4).toFixed(1)+'万' : String(v)) : v
    case 'number': return typeof v === 'number' ? v.toFixed(4) : v
    default: return v
  }
}
function colorize(v: number) { return v >= 0 ? 'var(--color-up)' : 'var(--color-down)' }
</script>

<style scoped>
.panel-table-wrapper { flex: 1; overflow: hidden; display: flex; flex-direction: column; }
.table-header-row {
  display: flex; padding: 4px 0; border-bottom: 1px solid var(--color-border-strong);
  font-size: var(--table-header-size); color: var(--table-header-color);
  font-weight: var(--table-header-weight); text-transform: uppercase; flex-shrink: 0;
}
.th, .td { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 0 4px; }
.th.left, .td.left { text-align: left; }
.th.right, .td.right { text-align: right; }
.th.center, .td.center { text-align: center; }
.table-body { flex: 1; overflow-y: auto; font-size: var(--table-row-size); }
.table-row {
  display: flex; align-items: center; min-height: var(--table-row-height);
  border-bottom: 1px solid var(--table-border); transition: background var(--transition-fast); cursor: default;
}
.table-row:hover { background: var(--table-row-hover); }
.table-row.clickable { cursor: pointer; }
.table-row.striped { background: var(--table-row-odd); }
.table-row.striped:hover { background: var(--table-row-hover); }
.td { font-variant-numeric: tabular-nums; }
.td.colorize { font-weight: 500; }
</style>
```

**验收标准**：
1. 文件存在且可编译
2. 在 `LimitUpDownPanel` 中临时替换表格测试，显示正常
3. 斑马纹在 Dark/Light 下正确显示

**依赖**：Task 1.1, Task 1.8（LoadingState）

---

### Task 1.5: 创建 `PanelTabs.vue` — 统一 Tab 切换

**文件**：`frontend/src/terminal/components/panel/PanelTabs.vue`

**接口设计**：

```vue
<template>
  <div :class="['panel-tabs', `variant-${variant}`]">
    <button
      v-for="tab in tabs" :key="tab.key"
      :class="['tab', { active: active === tab.key }]"
      @click="$emit('change', tab.key)"
    >{{ tab.label }}</button>
  </div>
</template>

<script setup lang="ts">
interface Tab { key: string; label: string }
withDefaults(defineProps<{
  tabs: Tab[]; active: string; variant?: 'pill' | 'underline' | 'button'
}>(), { variant: 'pill' })
defineEmits<{ (e: 'change', key: string): void }>()
</script>

<style scoped>
.panel-tabs { display: flex; gap: var(--space-xs); }

/* pill */
.panel-tabs.variant-pill .tab {
  padding: var(--tab-padding); height: var(--tab-height); font-size: var(--tab-font-size);
  border: 1px solid transparent; border-radius: var(--radius-md); background: transparent;
  color: var(--tab-inactive-color); cursor: pointer; transition: all var(--transition-fast);
}
.panel-tabs.variant-pill .tab:hover { color: var(--color-text-primary); background: var(--color-bg-hover); }
.panel-tabs.variant-pill .tab.active { color: var(--color-accent); border-color: var(--tab-active-border); background: var(--tab-active-bg); }

/* underline */
.panel-tabs.variant-underline .tab {
  padding: 4px 0; font-size: var(--tab-font-size); border: none; border-bottom: 2px solid transparent;
  background: transparent; color: var(--tab-inactive-color); cursor: pointer; transition: all var(--transition-fast);
  margin-right: var(--space-md);
}
.panel-tabs.variant-underline .tab:hover { color: var(--color-text-primary); }
.panel-tabs.variant-underline .tab.active { color: var(--color-accent); border-bottom-color: var(--color-accent); }

/* button */
.panel-tabs.variant-button .tab {
  padding: var(--tab-padding); height: var(--tab-height); font-size: var(--tab-font-size);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm); background: var(--color-bg-subtle);
  color: var(--color-text-secondary); cursor: pointer; transition: all var(--transition-fast);
}
.panel-tabs.variant-button .tab:hover { border-color: var(--color-border-strong); }
.panel-tabs.variant-button .tab.active { background: var(--color-accent); border-color: var(--color-accent); color: var(--color-text-inverse); }
</style>
```

**验收标准**：
1. 三种变体在 DevTools 中切换 class 均正常显示

**依赖**：Task 1.1

---

### Task 1.6: 创建 `PanelCard.vue` — 统一数据卡片

**文件**：`frontend/src/terminal/components/panel/PanelCard.vue`

**接口设计**：

```vue
<template>
  <div :class="['panel-card', { clickable: !!$attrs.onClick }]" @click="$emit('click')">
    <div class="card-header">
      <span class="card-title">{{ title }}</span>
      <span v-if="change != null" :class="['badge', change >= 0 ? 'badge-up' : 'badge-down']">
        {{ change >= 0 ? '+' : '' }}{{ (change * 100).toFixed(2) }}%
      </span>
    </div>
    <div class="card-value">{{ formattedValue }}</div>
    <svg v-if="sparkline?.length" class="sparkline" viewBox="0 0 100 30" preserveAspectRatio="none">
      <polyline :points="sparkPoints" fill="none" :stroke="change >= 0 ? 'var(--color-up)' : 'var(--color-down)'" stroke-width="1.5" />
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = withDefaults(defineProps<{
  title: string; value?: number; change?: number; format?: 'price' | 'percent' | 'volume' | 'number'; sparkline?: number[];
}>(), { format: 'price' })
const formattedValue = computed(() => {
  const v = props.value; if (v == null) return '--'
  switch (props.format) {
    case 'price': return v.toFixed(2)
    case 'percent': return (v >= 0 ? '+' : '') + (v * 100).toFixed(2) + '%'
    case 'volume': return v >= 1e8 ? (v/1e8).toFixed(2)+'亿' : v >= 1e4 ? (v/1e4).toFixed(1)+'万' : String(v)
    case 'number': return v.toFixed(4)
    default: return String(v)
  }
})
const sparkPoints = computed(() => {
  const d = props.sparkline; if (!d?.length) return ''
  const min = Math.min(...d), max = Math.max(...d), range = max - min || 1
  return d.map((v, i) => `${(i / (d.length - 1)) * 100},${30 - ((v - min) / range) * 30}`).join(' ')
})
</script>

<style scoped>
.panel-card {
  display: flex; flex-direction: column; gap: var(--space-sm); padding: var(--card-padding);
  background: var(--gradient-card); border: 1px solid var(--color-border); border-radius: var(--radius-lg);
  transition: all var(--transition-normal); min-width: var(--card-min-width); position: relative; overflow: hidden;
}
.panel-card:hover { border-color: var(--color-accent); box-shadow: var(--shadow-md); transform: translateY(-1px); }
.panel-card.clickable { cursor: pointer; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-title { font-size: var(--font-sm); color: var(--color-text-secondary); }
.card-value { font-size: var(--font-kpi); font-weight: 600; color: var(--color-text-primary); font-variant-numeric: tabular-nums; }
.sparkline { width: 100%; height: 24px; opacity: 0.6; }
</style>
```

**验收标准**：
1. 卡片在 Dark/Light 下背景/边框正确
2. Hover 时有 border-color 变化和微上移
3. Sparkline 颜色与涨跌一致

**依赖**：Task 1.1

---

### Task 1.7: 创建 `EmptyState.vue` — 统一空状态

**文件**：`frontend/src/terminal/components/panel/EmptyState.vue`

**接口设计**：

```vue
<template>
  <div class="empty-state">
    <span class="empty-icon" v-html="getIcon(icon || 'inbox')" />
    <h4 class="empty-title">{{ title }}</h4>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <button v-if="action" class="btn btn-primary" @click="action.handler">{{ action.label }}</button>
  </div>
</template>

<script setup lang="ts">
import { getIcon } from '@/lib/icons'
withDefaults(defineProps<{
  icon?: string; title: string; description?: string; action?: { label: string; handler: () => void }
}>(), { icon: 'inbox' })
</script>

<style scoped>
.empty-state {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: var(--space-md); padding: var(--space-xl); text-align: center;
}
.empty-icon { display: inline-flex; width: 48px; height: 48px; color: var(--color-text-tertiary); opacity: 0.5; }
.empty-icon :deep(svg) { width: 100%; height: 100%; }
.empty-title { font-size: var(--font-lg); font-weight: 600; color: var(--color-text-secondary); margin: 0; }
.empty-desc { font-size: var(--font-sm); color: var(--color-text-tertiary); margin: 0; max-width: 280px; }
</style>
```

**验收标准**：
1. 在 `BacktestResultPanel` 中临时替换测试，视觉正常

**依赖**：无

---

### Task 1.8: 创建 `LoadingState.vue` — 统一加载骨架

**文件**：`frontend/src/terminal/components/panel/LoadingState.vue`

**接口设计**：

```vue
<template>
  <div :class="['loading-state', `type-${type}`]">
    <template v-if="type === 'table'">
      <div v-for="i in rows" :key="i" class="skeleton-row">
        <div v-for="j in cols" :key="j" class="skeleton-cell" />
      </div>
    </template>
    <template v-else-if="type === 'card'">
      <div v-for="i in rows" :key="i" class="skeleton-card">
        <div class="skeleton-line w-60" /><div class="skeleton-line w-40" /><div class="skeleton-line w-80" />
      </div>
    </template>
    <template v-else-if="type === 'chart'">
      <div class="skeleton-chart" />
    </template>
    <template v-else>
      <div class="skeleton-inline" :style="{ width: inlineWidth }" />
    </template>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'table' | 'card' | 'chart' | 'inline'; rows?: number; cols?: number; inlineWidth?: string;
}>(), { type: 'table', rows: 5, cols: 4, inlineWidth: '120px' })
</script>

<style scoped>
.loading-state { width: 100%; }
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.skeleton-base {
  background: linear-gradient(90deg, var(--color-bg-elevated) 25%, var(--color-bg-hover) 50%, var(--color-bg-elevated) 75%);
  background-size: 200% 100%; animation: shimmer 1.5s ease-in-out infinite; border-radius: var(--radius-sm);
}
.skeleton-row { display: flex; gap: var(--space-sm); padding: var(--space-sm) 0; }
.skeleton-cell { flex: 1; height: 16px; composes: skeleton-base; }
.skeleton-card { display: flex; flex-direction: column; gap: var(--space-sm); padding: var(--space-md); border: 1px solid var(--color-border); border-radius: var(--radius-lg); margin-bottom: var(--space-sm); }
.skeleton-line { height: 12px; composes: skeleton-base; }
.w-60 { width: 60%; } .w-40 { width: 40%; } .w-80 { width: 80%; }
.skeleton-chart { height: 200px; composes: skeleton-base; border-radius: var(--radius-lg); }
.skeleton-inline { height: 16px; composes: skeleton-base; }
</style>
```

**注意**：`composes` 在 Vue scoped CSS 中不支持，需用 `:global()` 或 mixin。实际实现时改为直接复制 `background` 属性到每个类。

**验收标准**：
1. 在 Dark/Light 下骨架色自动适配（Dark 下深灰，Light 下浅灰）

**依赖**：Task 1.1

---

### Task 1.9: 创建 `SignalBadge.vue` — 替代 emoji 信号

**文件**：`frontend/src/terminal/components/panel/SignalBadge.vue`

**接口设计**：

```vue
<template>
  <span :class="['signal-badge', `signal-${signal}`, `size-${size}`]">
    <span class="signal-dot" />
    <span v-if="showLabel" class="signal-label">{{ label }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = withDefaults(defineProps<{
  signal: 'bullish' | 'bearish' | 'neutral'; showLabel?: boolean; size?: 'sm' | 'md' | 'lg';
}>(), { showLabel: true, size: 'md' })
const label = computed(() => ({ bullish: '偏多', bearish: '偏空', neutral: '中性' }[props.signal]))
</script>

<style scoped>
.signal-badge { display: inline-flex; align-items: center; gap: 4px; }
.signal-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.signal-bullish .signal-dot { background: var(--color-up); box-shadow: 0 0 4px var(--color-up-glow); }
.signal-bearish .signal-dot { background: var(--color-down); box-shadow: 0 0 4px var(--color-down-glow); }
.signal-neutral .signal-dot { background: var(--color-text-tertiary); }
.signal-bullish { color: var(--color-up); }
.signal-bearish { color: var(--color-down); }
.signal-neutral { color: var(--color-text-tertiary); }
.signal-label { font-weight: 600; }
.size-sm .signal-label { font-size: var(--font-xs); }
.size-md .signal-label { font-size: var(--font-sm); }
.size-lg .signal-label { font-size: var(--font-base); }
.size-lg .signal-dot { width: 8px; height: 8px; }
</style>
```

**验收标准**：
1. 在 `GovDataPanel` 中替换 emoji 测试，无 emoji 残留

**依赖**：Task 1.1

---

### Task 1.10: 创建 `TrendIndicator.vue` — 替代 emoji 趋势

**文件**：`frontend/src/terminal/components/panel/TrendIndicator.vue`

**接口设计**：

```vue
<template>
  <span :class="['trend-indicator', `trend-${direction}`]">
    <span class="trend-arrow">{{ arrow }}</span>
    <span v-if="change != null" class="trend-value">{{ formattedChange }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
const props = defineProps<{
  direction: 'up' | 'down' | 'flat'; change?: number;
}>()
const arrow = computed(() => ({ up: '▲', down: '▼', flat: '▬' }[props.direction]))
const formattedChange = computed(() => {
  if (props.change == null) return ''
  return (props.change >= 0 ? '+' : '') + props.change.toFixed(2) + '%'
})
</script>

<style scoped>
.trend-indicator { display: inline-flex; align-items: center; gap: 4px; font-size: var(--font-sm); font-weight: 500; }
.trend-up { color: var(--color-up); }
.trend-down { color: var(--color-down); }
.trend-flat { color: var(--color-text-tertiary); }
.trend-arrow { font-size: 0.8em; }
.trend-value { font-variant-numeric: tabular-nums; }
</style>
```

**验收标准**：
1. 三种方向在 Dark/Light 下颜色正确

**依赖**：Task 1.1

---

### Task 1.11: 创建 `index.ts` 组件导出文件

**文件**：`frontend/src/terminal/components/panel/index.ts`

```typescript
export { default as PanelHeader } from './PanelHeader.vue'
export { default as PanelToolbar } from './PanelToolbar.vue'
export { default as PanelTable } from './PanelTable.vue'
export { default as PanelTabs } from './PanelTabs.vue'
export { default as PanelCard } from './PanelCard.vue'
export { default as EmptyState } from './EmptyState.vue'
export { default as LoadingState } from './LoadingState.vue'
export { default as SignalBadge } from './SignalBadge.vue'
export { default as TrendIndicator } from './TrendIndicator.vue'
```

**验收标准**：
1. 在任意面板中 `import { PanelHeader, PanelTable } from '@/terminal/components/panel'` 能正确编译

**依赖**：Task 1.2 ~ 1.10

---

### Phase 1 总验收

运行以下命令，全部通过：

```bash
# 1. 确认所有组件文件存在
ls frontend/src/terminal/components/panel/PanelHeader.vue
ls frontend/src/terminal/components/panel/PanelToolbar.vue
ls frontend/src/terminal/components/panel/PanelTable.vue
ls frontend/src/terminal/components/panel/PanelTabs.vue
ls frontend/src/terminal/components/panel/PanelCard.vue
ls frontend/src/terminal/components/panel/EmptyState.vue
ls frontend/src/terminal/components/panel/LoadingState.vue
ls frontend/src/terminal/components/panel/SignalBadge.vue
ls frontend/src/terminal/components/panel/TrendIndicator.vue
ls frontend/src/terminal/components/panel/index.ts

# 2. 确认 themes.css 新 token 已追加
grep -c "panel-title-size" frontend/src/assets/themes.css   # 应 >= 1
grep -c "table-row-odd" frontend/src/assets/themes.css       # 应 >= 2 (dark+light)
grep -c "tab-height" frontend/src/assets/themes.css         # 应 >= 1

# 3. 确认无硬编码颜色在共享组件中
grep -rn "#[0-9a-f]\{6\}" frontend/src/terminal/components/panel/ || echo "PASS: no hardcoded colors"
# 预期输出 "PASS: no hardcoded colors"（或仅注释/图表配置中的匹配）

# 4. 前端构建通过
npm run build
# 或
vite build
```

---

## Phase 2: P0 面板迁移（4 个高频面板）

> **目标**：将 4 个视觉最突兀的面板迁移到共享组件，消除硬编码颜色。
> **时间**：2 天 | **Commit**：`feat: migrate P0 panels to shared components`
> **前提**：Phase 1 全部完成并通过验收

---

### Task 2.1: 迁移 `AlphaMiningWorkspacePanel.vue`

**文件**：`frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue`

**变更清单**：
1. 导入 `PanelHeader`, `PanelTable`, `EmptyState`
2. 替换自定义 `.panel-header` → `<PanelHeader>`
3. 替换原生 `<table>` → `<PanelTable>`
4. 替换 `#4a90d9` → `var(--color-accent)`（在 `.chip.active`, `.btn-run`）
5. 替换 `var(--border-color)` → `var(--color-border)`
6. 添加 `EmptyState` 用于无结果场景

**验收标准**：
```bash
grep "#4a90d9" frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue || echo "PASS"
grep "--border-color" frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue || echo "PASS"
grep "PanelHeader\|PanelTable\|EmptyState" frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue
# 预期包含导入行
```

**依赖**：Phase 1

---

### Task 2.2: 迁移 `IndicatorPanel.vue`

**文件**：`frontend/src/terminal/panels/IndicatorPanel.vue`

**变更清单**：
1. 导入 `PanelHeader`, `PanelTable`
2. 替换 `#1a1a2e` → `var(--color-bg-panel)`
3. 替换 `#2a2a3e` → `var(--color-border)`
4. 替换 `#534ab7` → `var(--color-accent)`
5. 替换 `#e5e7eb` → `var(--color-text-primary)`
6. 替换 `#6b7280` → `var(--color-text-tertiary)`
7. 替换 `.panel-header` → `<PanelHeader>`
8. 替换 `.data-table` → `<PanelTable>`
9. 保留 `.indicator-chip` 样式，但使用 `var(--*)` 替代硬编码

**验收标准**：
```bash
grep "#1a1a2e\|#2a2a3e\|#534ab7\|#e5e7eb\|#6b7280" frontend/src/terminal/panels/IndicatorPanel.vue || echo "PASS"
```

**依赖**：Phase 1

---

### Task 2.3: 迁移 `MarketOverviewPanel.vue`

**文件**：`frontend/src/terminal/panels/MarketOverviewPanel.vue`

**变更清单**：
1. 导入 `PanelHeader`, `PanelCard`, `PanelTable`
2. 替换 `#ef4444`/`#22c55e` → `var(--color-up)`/`var(--color-down)`
3. 替换 `#60a5fa` → `var(--color-accent)`
4. 替换 `#4b5563` → `var(--color-text-tertiary)`
5. 替换 `.panel-header` → `<PanelHeader>`
6. 替换 `.index-card` → `<PanelCard>`
7. 替换 `.block-rank-table` → `<PanelTable>`
8. 检查骨架屏 `.loading-overlay` 的 `position: absolute` → 确认父容器有 `position: relative`

**验收标准**：
```bash
grep "#ef4444\|#22c55e\|#60a5fa\|#4b5563" frontend/src/terminal/panels/MarketOverviewPanel.vue || echo "PASS"
```

**依赖**：Phase 1

---

### Task 2.4: 迁移 `LimitUpDownPanel.vue`

**文件**：`frontend/src/terminal/panels/LimitUpDownPanel.vue`

**变更清单**：
1. 导入 `PanelHeader`, `PanelTable`, `EmptyState`
2. 替换 `#dc2626`/`#16a34a` → `var(--color-up)`/`var(--color-down)`
3. 替换 `#60a5fa` → `var(--color-accent)`
4. 替换 `.panel-header` → `<PanelHeader>`
5. 替换 `.table-wrapper` → `<PanelTable>`
6. 替换 `.empty-state` → `<EmptyState>`
7. 删除 `.market-tabs`, `.filter-tabs` 样式（由 PanelHeader/PanellTabs 接管）

**验收标准**：
```bash
grep "#dc2626\|#16a34a\|#60a5fa" frontend/src/terminal/panels/LimitUpDownPanel.vue || echo "PASS"
```

**依赖**：Phase 1

---

### Phase 2 总验收

```bash
# 确认 P0 面板使用共享组件
for panel in AlphaMiningWorkspacePanel IndicatorPanel MarketOverviewPanel LimitUpDownPanel; do
  echo "=== $panel ==="
  grep "PanelHeader\|PanelTable\|PanelCard\|EmptyState" frontend/src/terminal/panels/${panel}.vue || echo "MISSING"
done

# 确认零硬编码（常见颜色）
grep -rn "#ef4444\|#dc2626\|#22c55e\|#16a34a\|#60a5fa\|#3b82f6\|#4a90d9\|#534ab7\|#1a1a2e\|#2a2a3e\|#e5e7eb\|#6b7280" frontend/src/terminal/panels/ \
  | grep -v "//" | grep -v "chartConfig" | grep -v "echarts" || echo "PASS: no hardcoded colors in panels"
```

---

## Phase 3: P1 面板迁移（高频 + 特殊问题面板）

> **目标**：迁移 5 个面板，修复 emoji、响应式、布局等专项问题。
> **时间**：2 天 | **Commit**：`feat: migrate P1 panels & fix responsive/emoji issues`
> **前提**：Phase 2 完成

---

### Task 3.1: 迁移 `GovDataPanel.vue` — 重点修复 emoji + 响应式

**文件**：`frontend/src/terminal/panels/GovDataPanel.vue`

**变更清单**：
1. 导入 `PanelHeader`, `PanelTabs`, `SignalBadge`, `TrendIndicator`
2. 替换 `#16a34a`/`#dc2626` → `var(--color-up)`/`var(--color-down)`
3. 替换所有 emoji（🟢🔴⚪🔄📈📉）→ `SignalBadge` / `TrendIndicator` / 图标 SVG
4. `.indicator-grid` 添加响应式：
   ```css
   .indicator-grid {
     display: grid;
     grid-template-columns: repeat(3, 1fr);
     gap: var(--card-gap);
   }
   @media (max-width: 400px) { .indicator-grid { grid-template-columns: 1fr; } }
   @media (min-width: 401px) and (max-width: 600px) { .indicator-grid { grid-template-columns: repeat(2, 1fr); } }
   ```
5. 替换 `.source-tab` / `.tab` → `<PanelTabs variant="pill">`
6. 替换 `.panel-header` → `<PanelHeader>`

**验收标准**：
```bash
# 确认无 emoji
grep -oP '[\x{1F300}-\x{1F9FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]' frontend/src/terminal/panels/GovDataPanel.vue || echo "PASS: no emoji"
# 确认使用共享组件
grep "SignalBadge\|TrendIndicator\|PanelTabs\|PanelHeader" frontend/src/terminal/panels/GovDataPanel.vue
# 确认响应式代码存在
grep "grid-template-columns" frontend/src/terminal/panels/GovDataPanel.vue | wc -l
# 预期 >= 3（3 个断点）
```

**依赖**：Phase 2

---

### Task 3.2: 迁移 `WatchlistPanel.vue`

**文件**：`frontend/src/terminal/panels/WatchlistPanel.vue`

**变更清单**：
1. 导入 `PanelToolbar`, `PanelTable`
2. 替换 `.symbol-list` → `<PanelTable>`（columns: symbol, name, last, changePct）
3. 替换 `.panel-toolbar` → `<PanelToolbar>`（搜索框）
4. 保留行内点击选中逻辑（`@rowClick` → `selectSymbol`）
5. 保留删除按钮（作为表格最后一列的 action slot）

**注意**：Watchlist 的交互较特殊（点击选中行、删除按钮），`PanelTable` 需要支持 `action` slot 或最后一列自定义渲染。

**PanelTable 扩展**：在 Task 1.4 中预留 `action` slot：
```vue
<!-- 在 PanelTable.vue 的 td 循环末尾添加 slot -->
<span v-if="$slots.action" class="td action">
  <slot name="action" :row="row" />
</span>
```

**验收标准**：
```bash
grep "PanelToolbar\|PanelTable" frontend/src/terminal/panels/WatchlistPanel.vue
```

**依赖**：Phase 2（需先扩展 PanelTable 支持 action slot）

---

### Task 3.3: 迁移 `BacktestResultPanel.vue`

**文件**：`frontend/src/terminal/panels/BacktestResultPanel.vue`

**变更清单**：
1. 导入 `EmptyState`
2. 替换 `.empty-state` → `<EmptyState>`
3. 丰富操作引导：添加 "去 Workflow 创建回测" 按钮（点击打开 `workflow` 面板）

**验收标准**：
```bash
grep "EmptyState" frontend/src/terminal/panels/BacktestResultPanel.vue
```

**依赖**：Phase 2

---

### Task 3.4: 迁移 `WelcomePanel.vue`

**文件**：`frontend/src/terminal/panels/WelcomePanel.vue`

**变更清单**：
1. 替换 `.snap-pct` 中的 `#dc2626`/`#16a34a` → `var(--color-up)`/`var(--color-down)`
2. 检查是否还有其他硬编码（如 categoryColors 的 rgba 值是设计意图，可保留）
3. 检查是否使用了废弃变量名

**验收标准**：
```bash
grep "#dc2626\|#16a34a" frontend/src/terminal/panels/WelcomePanel.vue || echo "PASS"
```

**依赖**：Phase 2

---

### Task 3.5: 批量检查其他 50+ 面板

**方法**：运行自动化扫描脚本，逐个面板处理：

```bash
# 扫描所有面板中的硬编码颜色和废弃变量
for f in frontend/src/terminal/panels/*.vue; do
  echo "=== $(basename $f) ==="
  grep -n "#ef4444\|#dc2626\|#22c55e\|#16a34a\|#60a5fa\|#3b82f6\|#2563eb\|#e5e7eb\|#6b7280\|#9ca3af\|#1a1a2e\|#2a2a3e\|#4a90d9\|#534ab7" "$f" || echo "  OK"
  grep -n "--border-color\|--term-bg-dim\|--term-accent-dim\|--color-text\|--color-bg" "$f" || echo "  OK"
done > /tmp/panel_audit.txt
```

**处理策略**：
- 对于只有少量硬编码的面板，直接在当前 Task 中修复
- 对于问题较多的面板，创建 Phase 4 的后续任务

**验收标准**：
```bash
# 确认所有面板中常见硬编码颜色清零
grep -rln "#ef4444\|#dc2626\|#22c55e\|#16a34a\|#60a5fa\|#3b82f6\|#e5e7eb\|#6b7280\|#1a1a2e\|#2a2a3e\|#4a90d9\|#534ab7" frontend/src/terminal/panels/ || echo "PASS"
# 确认废弃变量清零
grep -rln "--border-color\|--term-bg-dim\|--term-accent-dim" frontend/src/terminal/panels/ || echo "PASS"
```

**依赖**：Phase 2

---

### Phase 3 总验收

```bash
# 1. 所有面板零硬编码（常见颜色）
# 2. 所有面板零废弃变量
# 3. P1 面板使用共享组件
# 4. GovDataPanel 无 emoji
# 5. GovDataPanel 有响应式代码
```

---

## Phase 4: 全局验证与收尾

> **目标**：在四种主题组合下全面验证，确保无 regression。
> **时间**：1 天 | **Commit**：`chore: verify theme consistency across all panels`
> **前提**：Phase 3 完成

---

### Task 4.1: 硬编码颜色清零验证

**方法**：运行全面扫描

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
# 扫描所有 .vue 面板文件中的 6 位 hex 颜色（排除注释和图表配置）
echo "=== Hardcoded colors in panels ==="
grep -rn "#[0-9a-fA-F]\{6\}" frontend/src/terminal/panels/ \
  | grep -v "//" \
  | grep -v "chartConfig" \
  | grep -v "echarts" \
  | grep -v "color:" \
  | grep -v "background:" \
  || echo "PASS: no hardcoded colors"

# 扫描废弃变量
echo "=== Deprecated variables in panels ==="
grep -rln "--border-color\|--term-bg-dim\|--term-accent-dim\|--color-text\b\|--color-bg\b" frontend/src/terminal/panels/ || echo "PASS"
```

**验收标准**：
- 所有命令输出 `PASS`（或仅有图表 config 中的合法匹配）

**依赖**：Phase 3

---

### Task 4.2: 四模式截图对比

**方法**：在应用中切换以下 4 种组合，对关键面板截图：

| 组合 | 设置方法 |
|------|----------|
| Dark + Default Density | 默认状态 |
| Dark + Compact Density | Settings → Density → Compact |
| Dark + Comfortable Density | Settings → Density → Comfortable |
| Light + Default Density | Settings → Theme → Light |

**截图面板清单**：
- WelcomePanel（整体质感）
- MarketOverviewPanel（卡片 + 表格）
- LimitUpDownPanel（表格 + Tab）
- WatchlistPanel（列表 + 搜索）
- GovDataPanel（卡片 + 响应式 + 无 emoji）
- IndicatorPanel（表格 + chip）
- BacktestResultPanel（空状态）

**验收标准**：
- 所有面板在 4 种模式下均显示正确，无暗色残留（Light 模式）
- Compact 模式下信息密度提升但无拥挤
- Comfortable 模式下留白充足

**依赖**：Phase 3

---

### Task 4.3: 响应式断点测试

**方法**：在浏览器 DevTools 中调整面板宽度：

| 宽度 | 测试面板 | 预期 |
|------|----------|------|
| 280px | GovDataPanel | 单列卡片，无截断 |
| 280px | WatchlistPanel | 表格可横向滚动，列宽不压缩到不可读 |
| 400px | GovDataPanel | 双列卡片 |
| 600px | GovDataPanel | 三列卡片 |
| 800px | MarketOverviewPanel | 卡片网格 3-4 列，表格正常 |

**验收标准**：
- 所有测试宽度下无内容溢出、无截断、无重叠
- 表格在窄宽度下可横向滚动（scrollable）

**依赖**：Phase 3

---

### Task 4.4: 前端 Build 验证

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend
npm run build
# 或
npx vite build
```

**验收标准**：
- Build 零错误、零警告（TypeScript + Vue 编译）

**依赖**：Phase 3

---

### Task 4.5: 交互状态检查

**方法**：在浏览器中逐个检查：

| 交互 | 检查方式 | 预期 |
|------|----------|------|
| 按钮 hover | 鼠标悬停所有 `.btn` 类按钮 | border-color 变 accent，背景变 soft，有 transition |
| 按钮 active | 点击按钮 | 有 `translateY(1px)` 或背景加深 |
| 按钮 focus | Tab 键聚焦 | 有 `outline: 2px solid var(--color-accent)` |
| Tab 切换 | 点击 PanelTabs | active 状态有颜色变化和 border 指示 |
| 表格行 hover | 鼠标悬停表格行 | 背景变 `--table-row-hover` |
| 卡片 hover | 鼠标悬停 PanelCard | border 变 accent，微上移，有 shadow |
| 输入框 focus | 点击输入框 | border 变 accent，有 glow ring |
| 空状态 | 打开无数据面板 | 显示 EmptyState 组件，图标 + 标题 + 描述居中 |
| 加载状态 | 打开正在加载的面板 | 显示 LoadingState 骨架，shimmer 动画正常 |

**验收标准**：
- 所有交互在 Dark/Light 两种主题下均正常

**依赖**：Phase 3

---

### Phase 4 总验收

```bash
# 1. Build 通过
npm run build

# 2. 硬编码颜色清零
grep -rln "#ef4444\|#dc2626\|#22c55e\|#16a34a\|#60a5fa\|#3b82f6\|#e5e7eb\|#6b7280\|#1a1a2e\|#2a2a3e\|#4a90d9\|#534ab7" frontend/src/terminal/panels/ || echo "PASS"

# 3. 废弃变量清零
grep -rln "--border-color\|--term-bg-dim\|--term-accent-dim" frontend/src/terminal/panels/ || echo "PASS"

# 4. 共享组件使用覆盖
grep -rln "PanelHeader\|PanelTable\|PanelTabs\|PanelCard\|EmptyState\|LoadingState\|SignalBadge\|TrendIndicator" frontend/src/terminal/panels/ | wc -l
# 预期 >= 8（至少 8 个面板使用了共享组件）
```

---

## 附录 A：迁移检查清单（每个面板通用）

每个面板迁移时，按以下 checklist 执行：

```markdown
- [ ] 识别面板中的 header 结构 → 替换为 PanelHeader
- [ ] 识别面板中的 table 结构 → 替换为 PanelTable
- [ ] 识别面板中的 toolbar 结构 → 替换为 PanelToolbar
- [ ] 识别面板中的 tabs 结构 → 替换为 PanelTabs
- [ ] 识别面板中的卡片/指标卡片 → 替换为 PanelCard
- [ ] 识别空状态 → 替换为 EmptyState
- [ ] 识别加载状态 → 替换为 LoadingState
- [ ] 替换所有 `#hex` 硬编码颜色为 `var(--*)`
- [ ] 替换所有废弃变量名
- [ ] 检查 `overflow: hidden`（Dock 容器需要）
- [ ] 检查输入框 focus ring 是否被覆盖
- [ ] 检查按钮 hover/active/focus 状态
- [ ] 检查 Light Theme 兼容性
- [ ] 检查 Compact Density 兼容性
- [ ] 检查 Comfortable Density 兼容性
- [ ] 在 280px/400px/600px/800px 宽度下测试响应式
- [ ] 运行前端 build，确保无错误
```

---

## 附录 B：文件路径速查

| 类型 | 路径 |
|------|------|
| themes.css | `frontend/src/assets/themes.css` |
| 共享组件目录 | `frontend/src/terminal/components/panel/` |
| 共享组件导出 | `frontend/src/terminal/components/panel/index.ts` |
| P0 面板 | `frontend/src/terminal/panels/AlphaMiningWorkspacePanel.vue` |
|  | `frontend/src/terminal/panels/IndicatorPanel.vue` |
|  | `frontend/src/terminal/panels/MarketOverviewPanel.vue` |
|  | `frontend/src/terminal/panels/LimitUpDownPanel.vue` |
| P1 面板 | `frontend/src/terminal/panels/GovDataPanel.vue` |
|  | `frontend/src/terminal/panels/WatchlistPanel.vue` |
|  | `frontend/src/terminal/panels/BacktestResultPanel.vue` |
|  | `frontend/src/terminal/panels/WelcomePanel.vue` |
| 所有面板 | `frontend/src/terminal/panels/*.vue` (60+ 个) |

---

*Plan 版本: 2026-06-30 | 基于 Spec v1.0 | 执行时按 Phase 顺序推进*

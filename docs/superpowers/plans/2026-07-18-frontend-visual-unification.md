# 前端视觉统一与精修 Implementation Plan（Phase 0 + Phase 1）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立精修后的共享组件层与全局骨架类，批量消除面板自绘样式，完成 5 个打样面板迁移，作为 Phase 2 全量迁移的验收标准。

**Architecture:** 设计 token（`themes.css`）为唯一样式来源；共享组件（`frontend/src/terminal/components/panel/`）承载全部精修；先上全局类 + 机械删除 scoped 拷贝获得即时一致性，再逐面板迁移到组件。规格文档：`docs/superpowers/specs/2026-07-18-frontend-visual-unification-design.md`。

**Tech Stack:** Vue 3 (`<script setup>`), TypeScript, Pinia, Vite, vitest + @vue/test-utils, ECharts, Wails v3。

## Global Constraints

- 样式只允许引用 CSS 变量；scoped style 内禁止 hex 颜色、禁止 px 字号（`font-size`），padding/margin 用 `--space-*`
- 排版四档：面板标题 13px/600（`--font-sm`）、区块标题 12px/600 secondary、正文 13px、辅助 12px tertiary；数字/价格用 `var(--font-mono)` + `tabular-nums`
- 最小字号 11px（仅 compact 档辅助文字）；禁止出现 9px/10px
- 正文对比度 ≥ 4.5:1（WCAG AA）
- 涨跌表达 = 颜色 + `+/-` 号，不得只靠颜色
- 动效 150–250ms ease-out，只表达状态；`prefers-reduced-motion` 下全部降级
- 面板组件禁止自绘外框/box-shadow/border-radius 外壳（DockTab 已提供容器）
- 面板内禁止嵌套卡片；分区用 1px `--color-border-subtle` 分割线 + 间距
- 所有命令在 `frontend/` 目录下执行；提交信息用 conventional commits（`feat(frontend):` / `refactor(frontend):` / `fix(frontend):` / `style(frontend):` / `test(frontend):` / `chore(frontend):`）
- 每个 Task 结束必须 `npm run typecheck` 无错误、`npm run test` 全绿后才 commit

## 迁移模式库（Tasks 13–17 共用，每个迁移任务都要完整应用）

**P1 · header 统一**：把面板顶部手写块（`.panel-header` / `.toolbar` / `.tab-bar` / `.filter-bar` / `.chat-header`）替换为：

```vue
<PanelHeader
  title="面板名"
  :subtitle="可选副标题"
  :tabs="可选 [{ key, label }]"
  :active-tab="当前 tab key"
  :controls="可选 [{ icon: 'refresh', title: '刷新', action: load, loading }]"
  @tab-change="onTabChange"
/>
```

import 来自 `@/terminal/components/panel`（ barrel `index.ts`）。无标题面板（如纯 KPI+表格）也要补一个 PanelHeader 保持骨架一致。

**P2 · 表格统一**：原生 `<table>` 或 div 模拟表格替换为：

```vue
<PanelTable :columns="cols" :data="rows" :loading="loading" clickable @row-click="onRow" />
```

`cols: Column[]`（`import type { Column } from '@/terminal/components/panel'`）；数值列 `align: 'right'` + `format: 'price'|'percent'|'volume'|'number'`（自动等宽字体）；涨跌列 `colorize: true`。

**P3 · 状态三件套**：手写 `.empty-state` / `.panel-error` / `v-if="loading"` 文案块替换为 `<EmptyState title="..." description="..." />`、`<ErrorState :description="err" @retry="load" />`、`<LoadingState type="table" />`。

**P4 · 尺寸 token 化映射表**（scoped style 逐条替换）：

| 硬编码 | 替换为 |
|---|---|
| `font-size: 9/10/11px` | `var(--font-xs)`（并检查语义，9/10px 内容提升可读性） |
| `font-size: 12px` | `var(--font-xs)` |
| `font-size: 13/14px` | `var(--font-sm)` / `var(--font-base)` |
| `font-size: 16px+`（标题） | `var(--font-lg)`；面板标题一律走 PanelHeader |
| `padding/margin: Npx` | `var(--space-xs/sm/md/lg/xl)`（4/8/12/16/24） |
| `border-radius: Npx` | `var(--radius-sm/md/lg)` |
| `transition: all 0.xxs` | `var(--transition-fast)` / `var(--transition-normal)` |

**P5 · 颜色 token 化**：hex/rgba → 语义 token（`--color-text-*`、`--color-border*`、`--color-accent*`、`--color-up/down*`、`--color-*-soft`）。图表 option 里的 hex → Task 10 扩展后的 `useChartTheme()` 字段。

**P6 · ECharts 收口**：`import { useChartTheme } from '@/lib/composables/useChartTheme'`，option 中 `textStyle.color/axisLabel/axisLine/splitLine/tooltip` 全部取自 theme 对象；K 线用 `buildKlineOption`（`@/lib/buildChartOption`）。

**P7 · 外壳清理**：删除面板根元素上的 `border`/`box-shadow`/`border-radius`/`background: var(--color-bg-panel)`（DockTab 已提供）；根类名统一为 `xxx-panel`，样式 `height: 100%; display: flex; flex-direction: column; overflow: hidden`。

**迁移自检**（每个面板任务必跑）：

```bash
grep -nE 'font-size:\s*[0-9]+px|#[0-9a-fA-F]{3,8}\b|rgba?\(' src/terminal/panels/<File>.vue
# 期望：无输出（模板中 getIcon 调用除外）
grep -nE '\.panel-header|\.empty-state|\.panel-error|\.toolbar\b|\.tab-bar' src/terminal/panels/<File>.vue
# 期望：无输出
```

---

### Task 1: themes.css token 增补与全局骨架类

**Files:**
- Modify: `frontend/src/assets/themes.css`

**Interfaces:**
- Produces: `--panel-header-height`（40/36/44 随密度）；`--panel-title-size` 改为 `var(--font-sm)`；全局类 `.panel-header`、`.panel-header h3`、`.panel-header .header-actions`、`.empty-state`、`.panel-error`、`.section-title`、`.flash-up`、`.flash-down`。Task 2 依赖这些全局类；Task 9 依赖 `.flash-*`。

- [ ] **Step 1: 修改 panel token 区（`--panel-title-size` 行附近，约第 596–632 行）**

把 `--panel-title-size: var(--font-lg);` 改为：

```css
  --panel-title-size: var(--font-sm);
  --panel-header-height: 40px;
```

在 `body.density-compact` 块中追加 `--panel-header-height: 36px;`，在 `body.density-comfortable` 块中追加 `--panel-header-height: 44px;`。

- [ ] **Step 2: 在文件末尾追加全局骨架类**

```css
/* ── 全局面板骨架类（手写面板过渡期统一长相；共享组件优先） ─────────── */
.panel-header {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  flex-wrap: wrap;
  min-height: var(--panel-header-height);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}
.panel-header h3 {
  margin: 0;
  font-size: var(--panel-title-size);
  font-weight: var(--panel-title-weight);
  color: var(--color-text-primary);
  white-space: nowrap;
}
.panel-header .header-actions,
.panel-header .header-controls {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  margin-left: auto;
  flex-shrink: 0;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  text-align: center;
  color: var(--color-text-tertiary);
  font-size: var(--font-sm);
}

.panel-error {
  padding: var(--space-md) var(--panel-padding);
  color: var(--color-danger);
  font-size: var(--font-sm);
}

.section-title {
  font-size: var(--font-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0;
}

/* ── 数据刷新涨跌闪烁（信息性动效） ──────────────────────────────── */
@keyframes flash-up-kf {
  from { background-color: var(--color-up-soft); }
  to { background-color: transparent; }
}
@keyframes flash-down-kf {
  from { background-color: var(--color-down-soft); }
  to { background-color: transparent; }
}
.flash-up { animation: flash-up-kf 0.6s ease-out; }
.flash-down { animation: flash-down-kf 0.6s ease-out; }

@media (prefers-reduced-motion: reduce) {
  .flash-up, .flash-down { animation: none; }
}
```

- [ ] **Step 3: 验证**

Run: `cd frontend && npm run typecheck`
Expected: 无错误（纯 CSS 改动，确认没有破坏构建）

- [ ] **Step 4: Commit**

```bash
git add frontend/src/assets/themes.css
git commit -m "style(frontend): add panel-header-height token, global skeleton classes and flash animations"
```

---

### Task 2: 批量删除面板 scoped 样式拷贝

**Files:**
- Modify: `frontend/src/terminal/panels/*.vue`（约 60 个文件，清单见 Step 1）

**Interfaces:**
- Consumes: Task 1 的全局类（`.panel-header`/`.empty-state`/`.panel-error`）
- Produces: 删除后这些选择器在 panels 目录 scoped 中零残留，Task 13–17 在此基础上面板模板换组件。

- [ ] **Step 1: 生成待处理文件清单**

```bash
cd frontend
grep -rlE '^[[:space:]]*\.(panel-header|empty-state|panel-error)[[:space:]]*[,{]' src/terminal/panels --include='*.vue' | sort
```

（当前约 58 个文件命中。）对清单中每个文件执行 Step 2 规则。

- [ ] **Step 2: 删除规则（逐文件应用）**

在 `<style scoped>` 中：
1. 删除选择器**恰好**为 `.panel-header`、`.empty-state`、`.panel-error` 的整条规则块（含逗号分组命中时只移除对应选择器）。这些文件的手写 `.panel-header` 常见附带 `flex-wrap: wrap`、`gap`，全局类已是超集，直接删。
2. 删除 `.panel-header h3` / `.panel-header .header-actions` / `.panel-header .header-controls` 规则块（全局类已覆盖）。
3. **保留**后代/子元素特有规则（如 `.empty-state .empty-icon`、`.panel-error a`），以及 `.empty-state.compact` 这类带额外类的变体 —— 但若其属性与全局类重复（display/flex/padding/color）则删除重复声明。
4. 删除后检查该 `<style scoped>` 内不再引用被删类名造成的空规则。

注意：`NewsPanel.vue`、`WatchlistPanel.vue` 等使用共享组件的面板也可能有自己的 `.empty-state` 覆盖块，同样适用上述规则。

- [ ] **Step 3: 验证残留为零**

```bash
cd frontend
grep -rE '^[[:space:]]*\.(panel-header|empty-state|panel-error)[[:space:]]*[,{]' src/terminal/panels --include='*.vue' | wc -l
# 期望：0（或仅剩带额外限定符的变体，逐一人工确认合理）
npm run typecheck && npm run test
```

Expected: typecheck 无错误；测试全绿

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "refactor(frontend): remove duplicated panel-header/empty-state/panel-error scoped styles in favor of global classes"
```

---

### Task 3: PanelTabs 下划线滑动指示器

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelTabs.vue`
- Test: `frontend/src/terminal/__tests__/PanelTabs.test.ts`

**Interfaces:**
- Produces: `<PanelTabs :tabs="Tab[]" :active="string" variant?: 'pill'|'underline'|'button' @change="(key: string) => void">`。underline 变体带 `.tab-indicator` 滑动元素。Task 4 的 PanelHeader 以 `variant="underline"` 复用它。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/PanelTabs.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelTabs from '../components/panel/PanelTabs.vue'

const tabs = [
  { key: 'a', label: 'Tab A' },
  { key: 'b', label: 'Tab B' },
]

describe('PanelTabs', () => {
  it('renders all tabs and marks active', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a' } })
    expect(w.findAll('.tab')).toHaveLength(2)
    expect(w.find('.tab.active').text()).toBe('Tab A')
  })

  it('emits change on click', async () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a' } })
    await w.findAll('.tab')[1].trigger('click')
    expect(w.emitted('change')).toEqual([['b']])
  })

  it('underline variant renders a sliding indicator', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'b', variant: 'underline' } })
    expect(w.find('.tab-indicator').exists()).toBe(true)
  })

  it('pill variant has no indicator', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a', variant: 'pill' } })
    expect(w.find('.tab-indicator').exists()).toBe(false)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelTabs.test.ts`
Expected: FAIL（`.tab-indicator` 不存在）

- [ ] **Step 3: 实现滑动指示器**

`<script setup>` 替换为：

```typescript
import { ref, watch, onMounted, nextTick } from 'vue'

interface Tab {
  key: string
  label: string
}

const props = withDefaults(defineProps<{
  tabs: Tab[]
  active: string
  variant?: 'pill' | 'underline' | 'button'
}>(), {
  variant: 'pill',
})

defineEmits<{
  (e: 'change', key: string): void
}>()

const root = ref<HTMLElement | null>(null)
const indicatorStyle = ref<{ left: string; width: string }>({ left: '0px', width: '0px' })

async function updateIndicator() {
  if (props.variant !== 'underline') return
  await nextTick()
  const el = root.value?.querySelector<HTMLElement>('.tab.active')
  if (!el) return
  indicatorStyle.value = { left: `${el.offsetLeft}px`, width: `${el.offsetWidth}px` }
}

onMounted(updateIndicator)
watch(() => [props.active, props.tabs], updateIndicator, { deep: true })
```

模板根元素改为：

```vue
<div ref="root" :class="['panel-tabs', `variant-${variant}`]">
  <button
    v-for="tab in tabs"
    :key="tab.key"
    :class="['tab', { active: active === tab.key }]"
    @click="$emit('change', tab.key)"
  >
    {{ tab.label }}
  </button>
  <span v-if="variant === 'underline'" class="tab-indicator" :style="indicatorStyle" />
</div>
```

样式：`.panel-tabs` 加 `position: relative;`；underline 变体的 `.tab` 移除 `border-bottom` 相关两行，追加：

```css
.panel-tabs.variant-underline .tab-indicator {
  position: absolute;
  bottom: 0;
  height: 2px;
  background: var(--color-accent);
  border-radius: 1px;
  transition: left var(--transition-normal), width var(--transition-normal);
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .panel-tabs.variant-underline .tab-indicator { transition: none; }
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelTabs.test.ts`
Expected: 4 passed

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelTabs.vue frontend/src/terminal/__tests__/PanelTabs.test.ts
git commit -m "feat(frontend): add sliding underline indicator to PanelTabs underline variant"
```

---

### Task 4: PanelHeader 精修

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelHeader.vue`
- Test: `frontend/src/terminal/__tests__/PanelHeader.test.ts`

**Interfaces:**
- Consumes: Task 3 的 PanelTabs（underline 变体）
- Produces: `<PanelHeader title? subtitle? :tabs? :active-tab? :controls? @tab-change>`。Control 类型：`{ icon?: string; label?: string; title?: string; action: () => void; loading?: boolean }`。Task 13–17 全部依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/PanelHeader.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelHeader from '../components/panel/PanelHeader.vue'

describe('PanelHeader', () => {
  it('renders title and subtitle', () => {
    const w = mount(PanelHeader, { props: { title: '自选股', subtitle: '12 只' } })
    expect(w.find('.panel-title').text()).toBe('自选股')
    expect(w.find('.panel-subtitle').text()).toBe('12 只')
  })

  it('renders underline tabs and forwards tabChange', async () => {
    const w = mount(PanelHeader, {
      props: { title: 'T', tabs: [{ key: 'a', label: 'A' }, { key: 'b', label: 'B' }], activeTab: 'a' },
    })
    const tabs = w.findComponent({ name: 'PanelTabs' })
    expect(tabs.exists()).toBe(true)
    expect(tabs.props('variant')).toBe('underline')
    await tabs.vm.$emit('change', 'b')
    expect(w.emitted('tabChange')).toEqual([['b']])
  })

  it('renders controls and triggers action', async () => {
    let called = 0
    const w = mount(PanelHeader, {
      props: { title: 'T', controls: [{ icon: 'refresh', title: '刷新', action: () => { called++ } }] },
    })
    await w.find('.header-controls button').trigger('click')
    expect(called).toBe(1)
  })

  it('omits sections when props absent', () => {
    const w = mount(PanelHeader, { props: { title: 'T' } })
    expect(w.find('.header-controls').exists()).toBe(false)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelHeader.test.ts`
Expected: FAIL（当前实现 tabs 是内联 button 组，不存在 PanelTabs 子组件调用）

- [ ] **Step 3: 改造 PanelHeader**

模板中 `.header-tabs` 块替换为：

```vue
    <PanelTabs
      v-if="tabs?.length"
      :tabs="tabs"
      :active="activeTab ?? ''"
      variant="underline"
      @change="(key: string) => $emit('tabChange', key)"
    />
```

script 顶部追加 `import PanelTabs from './PanelTabs.vue'`；删除模板里旧的 `.header-tabs` button 循环和样式中 `.header-tabs`、`.tab-btn` 相关块。

样式精修（`.panel-header` scoped 块）：

```css
.panel-header {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-sm) var(--panel-padding);
  border-bottom: 1px solid var(--color-border-subtle);
  min-height: var(--panel-header-height);
  flex-wrap: wrap;
  flex-shrink: 0;
}
```

`.panel-title` 的 `font-size` 保持 `var(--panel-title-size)`（Task 1 已把 token 改为 13px 档）。其余样式不变。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelHeader.test.ts src/terminal/__tests__/PanelTabs.test.ts`
Expected: 全部通过

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelHeader.vue frontend/src/terminal/__tests__/PanelHeader.test.ts
git commit -m "feat(frontend): refine PanelHeader with underline tabs and subtle border"
```

---

### Task 5: PanelTable 精修（等宽数字列 + token 化）

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelTable.vue`
- Modify: `frontend/src/terminal/components/panel/types.ts`
- Test: `frontend/src/terminal/__tests__/PanelTable.test.ts`

**Interfaces:**
- Produces: `Column` 增加 `mono?: boolean`（缺省时 `format` 为 price/percent/volume/number 的列自动等宽右对齐）。Task 13–17 依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/PanelTable.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelTable from '../components/panel/PanelTable.vue'
import type { Column } from '../components/panel/types'

const cols: Column[] = [
  { key: 'name', label: '名称' },
  { key: 'price', label: '现价', align: 'right', format: 'price' },
  { key: 'chg', label: '涨跌', align: 'right', format: 'percent', colorize: true },
]
const data = [
  { name: '平安银行', price: 12.345, chg: 1.234 },
  { name: '万科A', price: 8.5, chg: -0.876 },
]

describe('PanelTable', () => {
  it('formats price/percent with sign', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const text = w.text()
    expect(text).toContain('12.35')
    expect(text).toContain('+1.23%')
    expect(text).toContain('-0.88%')
  })

  it('applies mono class automatically to numeric formats', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const firstRowTds = w.findAll('.table-row')[0].findAll('.td')
    expect(firstRowTds[0].classes()).not.toContain('mono')
    expect(firstRowTds[1].classes()).toContain('mono')
  })

  it('colorize paints up/down by sign', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const chgCells = w.findAll('.td.colorize')
    expect(chgCells[0].attributes('style')).toContain('var(--color-up)')
    expect(chgCells[1].attributes('style')).toContain('var(--color-down)')
  })

  it('shows loading state when loading and empty', () => {
    const w = mount(PanelTable, { props: { columns: cols, data: [], loading: true } })
    expect(w.find('.loading-state').exists()).toBe(true)
  })

  it('emits rowClick', async () => {
    const w = mount(PanelTable, { props: { columns: cols, data, clickable: true } })
    await w.findAll('.table-row')[0].trigger('click')
    expect(w.emitted('rowClick')).toHaveLength(1)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelTable.test.ts`
Expected: FAIL（`mono` class 不存在）

- [ ] **Step 3: 实现**

`types.ts` 的 `Column` 接口追加：

```typescript
  /** 等宽数字列；缺省时 format 为 price/percent/volume/number 自动为 true */
  mono?: boolean
```

`PanelTable.vue` script 追加：

```typescript
const NUMERIC_FORMATS = new Set(['price', 'percent', 'volume', 'number'])
function isMono(col: Column): boolean {
  return col.mono ?? (col.format ? NUMERIC_FORMATS.has(col.format) : false)
}
```

td 的 class 绑定改为：

```vue
:class="['td', col.align || 'left', { colorize: col.colorize, mono: isMono(col) }]"
```

样式追加/修改：

```css
.td.mono {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}
```

并把现有硬编码 token 化：`.table-header-row` 的 `padding: 6px 0` → `padding: var(--space-xs) 0`、`border-bottom: 1.5px solid var(--color-border-strong)` → `border-bottom: 1px solid var(--color-border)`；`.th, .td` 的 `padding: 0 6px` → `padding: 0 var(--space-xs)`。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd frontend && npx vitest run src/terminal/__tests__/PanelTable.test.ts`
Expected: 5 passed

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelTable.vue frontend/src/terminal/components/panel/types.ts frontend/src/terminal/__tests__/PanelTable.test.ts
git commit -m "feat(frontend): auto mono numeric columns and token cleanup in PanelTable"
```

---

### Task 6: EmptyState / LoadingState 精修

**Files:**
- Modify: `frontend/src/terminal/components/panel/EmptyState.vue`
- Modify: `frontend/src/terminal/components/panel/LoadingState.vue`
- Test: `frontend/src/terminal/__tests__/EmptyState.test.ts`

**Interfaces:**
- Produces: `<EmptyState icon?='inbox' title :description? :actions?>` 视觉参数改为：图标 32px、标题 13px/500、描述 12px；移除入场动画。`<LoadingState type rows cols inlineWidth>` 增加 reduced-motion 降级。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/EmptyState.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from '../components/panel/EmptyState.vue'

describe('EmptyState', () => {
  it('renders title/description and no entrance animation class', () => {
    const w = mount(EmptyState, { props: { title: '暂无数据', description: '稍后再试' } })
    expect(w.find('.empty-title').text()).toBe('暂无数据')
    expect(w.find('.empty-desc').text()).toBe('稍后再试')
    expect(w.html()).not.toContain('empty-enter')
  })

  it('renders actions and triggers handler', async () => {
    let n = 0
    const w = mount(EmptyState, {
      props: { title: '空', actions: [{ label: '去添加', primary: true, handler: () => { n++ } }] },
    })
    await w.find('.empty-actions .btn-primary').trigger('click')
    expect(n).toBe(1)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/EmptyState.test.ts`
Expected: FAIL（当前含 `empty-enter` keyframes 引用）

- [ ] **Step 3: 精修 EmptyState**

删除 `@keyframes empty-enter` 与 `.empty-state` 上的 `animation` 声明。样式参数改为：

```css
.empty-icon {
  display: inline-flex;
  width: 32px;
  height: 32px;
  color: var(--color-text-tertiary);
}

.empty-title {
  font-size: var(--font-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  margin: 0;
}

.empty-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin: 0;
  max-width: 280px;
  line-height: 1.6;
}
```

- [ ] **Step 4: LoadingState 降级**

`LoadingState.vue` 样式末尾追加：

```css
@media (prefers-reduced-motion: reduce) {
  .skeleton-cell, .skeleton-line, .skeleton-chart, .skeleton-inline {
    animation: none;
  }
}
```

- [ ] **Step 5: 跑测试确认通过并提交**

Run: `cd frontend && npx vitest run src/terminal/__tests__/EmptyState.test.ts`
Expected: 2 passed

```bash
git add frontend/src/terminal/components/panel/EmptyState.vue frontend/src/terminal/components/panel/LoadingState.vue frontend/src/terminal/__tests__/EmptyState.test.ts
git commit -m "feat(frontend): refine EmptyState typography, drop entrance animation, reduce-motion for skeletons"
```

---

### Task 7: StatItem 新组件（KPI 统计块）

**Files:**
- Create: `frontend/src/terminal/components/panel/StatItem.vue`
- Modify: `frontend/src/terminal/components/panel/index.ts`
- Test: `frontend/src/terminal/__tests__/StatItem.test.ts`

**Interfaces:**
- Produces: `<StatItem label :value :delta?>`，delta 为百分数（如 `1.23` 表示 +1.23%）。barrel 导出 `StatItem`。Task 14/15 依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/StatItem.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatItem from '../components/panel/StatItem.vue'

describe('StatItem', () => {
  it('renders label and value', () => {
    const w = mount(StatItem, { props: { label: '总市值', value: '1,234.5亿' } })
    expect(w.find('.stat-label').text()).toBe('总市值')
    expect(w.find('.stat-value').text()).toBe('1,234.5亿')
  })

  it('renders positive delta with badge-up and plus sign', () => {
    const w = mount(StatItem, { props: { label: '盈亏', value: '100', delta: 1.234 } })
    const badge = w.find('.stat-delta')
    expect(badge.text()).toBe('+1.23%')
    expect(badge.classes()).toContain('badge-up')
  })

  it('renders negative delta with badge-down', () => {
    const w = mount(StatItem, { props: { label: '盈亏', value: '-50', delta: -0.5 } })
    const badge = w.find('.stat-delta')
    expect(badge.text()).toBe('-0.50%')
    expect(badge.classes()).toContain('badge-down')
  })

  it('omits delta badge when not provided', () => {
    const w = mount(StatItem, { props: { label: 'A', value: '1' } })
    expect(w.find('.stat-delta').exists()).toBe(false)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/StatItem.test.ts`
Expected: FAIL（组件不存在）

- [ ] **Step 3: 实现 StatItem.vue**

```vue
<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: string | number
  /** 涨跌百分数，如 1.23 表示 +1.23%；不传则不显示 */
  delta?: number
}>()

const deltaText = computed(() => {
  if (props.delta == null) return ''
  return (props.delta >= 0 ? '+' : '') + props.delta.toFixed(2) + '%'
})
const deltaClass = computed(() => (props.delta != null && props.delta >= 0 ? 'badge-up' : 'badge-down'))
</script>

<template>
  <div class="stat-item">
    <span class="stat-label">{{ label }}</span>
    <span class="stat-value-row">
      <span class="stat-value">{{ value }}</span>
      <span v-if="delta != null" :class="['stat-delta', 'badge', deltaClass]">{{ deltaText }}</span>
    </span>
  </div>
</template>

<style scoped>
.stat-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
  min-width: 0;
}

.stat-label {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  white-space: nowrap;
}

.stat-value-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-sm);
}

.stat-value {
  font-family: var(--font-mono);
  font-size: var(--font-xl);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
  white-space: nowrap;
}
</style>
```

`index.ts` 追加：`export { default as StatItem } from './StatItem.vue'`

- [ ] **Step 4: 跑测试确认通过并提交**

Run: `cd frontend && npx vitest run src/terminal/__tests__/StatItem.test.ts`
Expected: 4 passed

```bash
git add frontend/src/terminal/components/panel/StatItem.vue frontend/src/terminal/components/panel/index.ts frontend/src/terminal/__tests__/StatItem.test.ts
git commit -m "feat(frontend): add StatItem KPI component"
```

---

### Task 8: ErrorState 新组件

**Files:**
- Create: `frontend/src/terminal/components/panel/ErrorState.vue`
- Modify: `frontend/src/terminal/components/panel/index.ts`
- Test: `frontend/src/terminal/__tests__/ErrorState.test.ts`

**Interfaces:**
- Produces: `<ErrorState title?='加载失败' :description? retry-label?='重试' @retry>`。barrel 导出 `ErrorState`。Task 13–17 依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/ErrorState.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ErrorState from '../components/panel/ErrorState.vue'

describe('ErrorState', () => {
  it('renders default title and description', () => {
    const w = mount(ErrorState, { props: { description: '网络超时' } })
    expect(w.find('.error-title').text()).toBe('加载失败')
    expect(w.find('.error-desc').text()).toBe('网络超时')
  })

  it('emits retry on button click', async () => {
    const w = mount(ErrorState, { props: {} })
    await w.find('.error-retry').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/ErrorState.test.ts`
Expected: FAIL（组件不存在）

- [ ] **Step 3: 实现 ErrorState.vue**

```vue
<script setup lang="ts">
import { getIcon } from '@/lib/icons'

withDefaults(defineProps<{
  title?: string
  description?: string
  retryLabel?: string
}>(), {
  title: '加载失败',
  retryLabel: '重试',
})

defineEmits<{
  (e: 'retry'): void
}>()
</script>

<template>
  <div class="error-state">
    <span class="error-icon" v-html="getIcon('warning')" />
    <h4 class="error-title">{{ title }}</h4>
    <p v-if="description" class="error-desc">{{ description }}</p>
    <button class="btn error-retry" @click="$emit('retry')">{{ retryLabel }}</button>
  </div>
</template>

<style scoped>
.error-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-md);
  padding: var(--space-xl);
  text-align: center;
}

.error-icon {
  display: inline-flex;
  width: 32px;
  height: 32px;
  color: var(--color-danger);
}

.error-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.error-title {
  font-size: var(--font-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  margin: 0;
}

.error-desc {
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  margin: 0;
  max-width: 280px;
  line-height: 1.6;
}
</style>
```

`index.ts` 追加：`export { default as ErrorState } from './ErrorState.vue'`

- [ ] **Step 4: 跑测试确认通过并提交**

Run: `cd frontend && npx vitest run src/terminal/__tests__/ErrorState.test.ts`
Expected: 2 passed

```bash
git add frontend/src/terminal/components/panel/ErrorState.vue frontend/src/terminal/components/panel/index.ts frontend/src/terminal/__tests__/ErrorState.test.ts
git commit -m "feat(frontend): add ErrorState component with retry"
```

---

### Task 9: useFlashOnUpdate composable（数据刷新涨跌闪烁）

**Files:**
- Create: `frontend/src/lib/composables/useFlashOnUpdate.ts`
- Test: `frontend/src/terminal/__tests__/useFlashOnUpdate.test.ts`

**Interfaces:**
- Consumes: Task 1 的全局类 `.flash-up`/`.flash-down`
- Produces: `useFlashOnUpdate(source: Ref<number | null | undefined>, duration?: number): { flashClass: Ref<'' | 'flash-up' | 'flash-down'> }`。Task 15 依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/useFlashOnUpdate.test.ts
import { describe, it, expect, vi, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useFlashOnUpdate } from '@/lib/composables/useFlashOnUpdate'

describe('useFlashOnUpdate', () => {
  afterEach(() => { vi.useRealTimers() })

  it('sets flash-up when value rises, then clears after duration', async () => {
    vi.useFakeTimers()
    const v = ref(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 11
    await nextTick()
    expect(flashClass.value).toBe('flash-up')
    vi.advanceTimersByTime(650)
    await nextTick()
    expect(flashClass.value).toBe('')
  })

  it('sets flash-down when value falls', async () => {
    vi.useFakeTimers()
    const v = ref(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 9.5
    await nextTick()
    expect(flashClass.value).toBe('flash-down')
  })

  it('does not flash on equal or null transitions', async () => {
    vi.useFakeTimers()
    const v = ref<number | null>(10)
    const { flashClass } = useFlashOnUpdate(v)
    v.value = 10
    await nextTick()
    expect(flashClass.value).toBe('')
    v.value = null
    await nextTick()
    expect(flashClass.value).toBe('')
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/useFlashOnUpdate.test.ts`
Expected: FAIL（模块不存在）

- [ ] **Step 3: 实现**

```typescript
// frontend/src/lib/composables/useFlashOnUpdate.ts
import { ref, watch, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export type FlashClass = '' | 'flash-up' | 'flash-down'

/**
 * 监听数值变化，返回一个短暂设置的 CSS class（配合全局 .flash-up/.flash-down 动画）。
 * 仅在前后值均为有限数值且发生变化时闪烁。
 */
export function useFlashOnUpdate(
  source: Ref<number | null | undefined>,
  duration = 600,
): { flashClass: Ref<FlashClass> } {
  const flashClass = ref<FlashClass>('')
  let timer: ReturnType<typeof setTimeout> | undefined

  const stop = watch(source, (next, prev) => {
    if (typeof next !== 'number' || typeof prev !== 'number') return
    if (!Number.isFinite(next) || !Number.isFinite(prev) || next === prev) return
    flashClass.value = next > prev ? 'flash-up' : 'flash-down'
    clearTimeout(timer)
    timer = setTimeout(() => { flashClass.value = '' }, duration)
  })

  onUnmounted(() => {
    stop()
    clearTimeout(timer)
  })

  return { flashClass }
}
```

注意：此 composable 调用 `onUnmounted`，必须在组件 `setup()` 上下文中使用；测试中直接调用时需包一层组件（若测试报 "onUnmounted is called when there is no active component instance" 警告，可接受；若要消除，用 `mount` 一个内联测试组件包装）。

- [ ] **Step 4: 跑测试确认通过并提交**

Run: `cd frontend && npx vitest run src/terminal/__tests__/useFlashOnUpdate.test.ts`
Expected: 3 passed

```bash
git add frontend/src/lib/composables/useFlashOnUpdate.ts frontend/src/terminal/__tests__/useFlashOnUpdate.test.ts
git commit -m "feat(frontend): add useFlashOnUpdate composable for price-flash feedback"
```

---

### Task 10: useChartTheme 修复与扩展

**Files:**
- Modify: `frontend/src/lib/composables/useChartTheme.ts`
- Test: `frontend/src/terminal/__tests__/useChartTheme.test.ts`

**Interfaces:**
- Produces: `ChartThemeColors` 扩展为 `{ textColor, axisColor, splitColor, bgColor, crossColor, tooltipBg, tooltipText, upColor, downColor, gridColor, palette: string[] }`。修复 `--color-error` → `--color-danger` 的 bug。Task 11 依赖。

- [ ] **Step 1: 写失败测试**

```typescript
// frontend/src/terminal/__tests__/useChartTheme.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { useChartTheme } from '@/lib/composables/useChartTheme'

function mountTheme() {
  let theme: ReturnType<typeof useChartTheme>
  const Comp = defineComponent({
    setup() {
      theme = useChartTheme()
      return () => null
    },
  })
  mount(Comp)
  return theme!
}

describe('useChartTheme', () => {
  beforeEach(() => {
    document.body.style.cssText = `
      --color-text-primary: #111111;
      --color-text-tertiary: #555555;
      --color-border-subtle: #eeeeee;
      --color-bg-elevated: #ffffff;
      --color-danger: #cc0000;
      --color-bg-glass: rgba(255,255,255,0.9);
      --color-up: #c62828;
      --color-down: #2e7d32;
      --chart-grid: #dddddd;
      --chart-1: #1d64d8; --chart-2: #2e7d32; --chart-3: #b45309;
      --chart-4: #c62828; --chart-5: #6d28d9; --chart-6: #0e7490;
    `
  })

  it('reads colors from CSS variables', () => {
    const theme = mountTheme()
    expect(theme.textColor).toBe('#111111')
    expect(theme.crossColor).toBe('#cc0000')
  })

  it('exposes up/down colors, grid and 6-color palette', () => {
    const theme = mountTheme()
    expect(theme.upColor).toBe('#c62828')
    expect(theme.downColor).toBe('#2e7d32')
    expect(theme.gridColor).toBe('#dddddd')
    expect(theme.palette).toHaveLength(6)
    expect(theme.palette[0]).toBe('#1d64d8')
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/terminal/__tests__/useChartTheme.test.ts`
Expected: FAIL（crossColor 读到 fallback `#e24b4a`，且缺 upColor/downColor/gridColor/palette 字段）

- [ ] **Step 3: 修改 useChartTheme.ts**

`ChartThemeColors` 接口追加字段：

```typescript
export interface ChartThemeColors {
  textColor: string
  axisColor: string
  splitColor: string
  bgColor: string
  crossColor: string
  tooltipBg: string
  tooltipText: string
  upColor: string
  downColor: string
  gridColor: string
  palette: string[]
}
```

`readTheme()` 中 `crossColor: s('--color-error')` 改为 `crossColor: s('--color-danger')`，并追加读取（fallback 用 themes.css 亮主题值）：

```typescript
      upColor: s('--color-up') || '#c62828',
      downColor: s('--color-down') || '#2e7d32',
      gridColor: s('--chart-grid') || 'rgba(16, 24, 40, 0.06)',
      palette: [1, 2, 3, 4, 5, 6].map(i => s(`--chart-${i}`) || ['#1d64d8', '#2e7d32', '#b45309', '#c62828', '#6d28d9', '#0e7490'][i - 1]),
```

两处 fallback 对象（`!root` 分支与 `catch` 分支）同步补全这 4 个字段。

- [ ] **Step 4: 跑测试确认通过并提交**

Run: `cd frontend && npx vitest run src/terminal/__tests__/useChartTheme.test.ts`
Expected: 2 passed

```bash
git add frontend/src/lib/composables/useChartTheme.ts frontend/src/terminal/__tests__/useChartTheme.test.ts
git commit -m "fix(frontend): correct chart theme danger token and expose up/down/grid/palette"
```

---

### Task 11: 图表面板硬编码颜色清理

**Files:**
- Modify: `frontend/src/terminal/panels/BacktestPanel.vue`（内联 K线 option + 暗色 hex）
- Modify: `frontend/src/terminal/panels/DupontPanel.vue`、`EventStudyPanel.vue`、`SectorDashboard.vue`、`MarketStylePanel.vue`、`HKConnectPanel.vue`、`FinancialsPanel.vue`（暗色 hex）
- Modify: Step 1 grep 新发现的其它面板

**Interfaces:**
- Consumes: Task 10 的 `useChartTheme()`（含 upColor/downColor/gridColor/palette）与 `@/lib/buildChartOption` 的 `buildKlineOption`

- [ ] **Step 1: 全量定位**

```bash
cd frontend
grep -rlnE "#2a2a3a|#8b8ba0|#9ca3af|rgba\(255,\s*255,\s*255|#4caf50|#ff9800|#ef5350|#66bb6a" src/terminal/panels --include='*.vue'
```

- [ ] **Step 2: 逐文件替换规则**

对每个命中文件：
1. script 中 `import { useChartTheme } from '@/lib/composables/useChartTheme'`，`setup` 内 `const chartTheme = useChartTheme()`（若已存在则复用）。
2. ECharts option 里的轴/文字/网格硬编码色替换：`axisLabel.color`/`axisLine.lineStyle.color` → `chartTheme.axisColor`，`splitLine.lineStyle.color` → `chartTheme.gridColor`，`textStyle.color` → `chartTheme.textColor`，tooltip 背景/文字 → `chartTheme.tooltipBg`/`tooltipText`，涨跌色 → `chartTheme.upColor`/`downColor`，系列色 → `chartTheme.palette[n]`。option 若是 computed/method 返回值，直接引用 reactive theme 即可随主题切换。
3. `BacktestPanel.vue` 特有：删除内联复制的 K线 option 逻辑（约 181–197 行），改为调用 `buildKlineOption`（`import { buildKlineOption } from '@/lib/buildChartOption'`），保持传参语义不变。

- [ ] **Step 3: 验证**

```bash
cd frontend
grep -rnE "#2a2a3a|#8b8ba0|#9ca3af|rgba\(255,\s*255,\s*255" src/terminal/panels --include='*.vue' | wc -l
# 期望：0
npm run typecheck && npm run test
```

Expected: 0 残留；typecheck 无错误；测试全绿

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "fix(frontend): route chart panels through useChartTheme, drop hardcoded dark colors"
```

---

### Task 12: stylelint 防退化守卫

**Files:**
- Create: `frontend/.stylelintrc.json`
- Modify: `frontend/package.json`（scripts + devDependencies）

**Interfaces:**
- Produces: `npm run lint:styles` 命令（warn 级，exit 0）；Phase 2 结束后转 error。

- [ ] **Step 1: 安装依赖**

```bash
cd frontend && npm i -D stylelint stylelint-config-standard-vue
```

- [ ] **Step 2: 配置 `.stylelintrc.json`**

```json
{
  "extends": "stylelint-config-standard-vue",
  "rules": {
    "color-no-hex": [true, { "severity": "warning" }],
    "declaration-property-unit-disallowed-list": [
      { "font-size": ["px"] },
      { "severity": "warning" }
    ],
    "declaration-property-value-disallowed-list": [
      { "/^border$/": ["none"] },
      { "severity": "warning" }
    ],
    "selector-class-pattern": null,
    "no-descending-specificity": null,
    "declaration-block-no-duplicate-properties": null,
    "no-empty-source": null,
    "font-family-no-duplicate-names": null,
    "property-no-deprecated": null,
    "media-feature-name-no-vendor-prefix": null,
    "value-no-vendor-prefix": null,
    "alpha-value-notation": null,
    "color-function-notation": null,
    "length-zero-no-unit": null,
    "shorthand-property-no-redundant-values": null,
    "selector-not-notation": null,
    "keyframes-name-pattern": null,
    "hue-degree-notation": null,
    "import-notation": null,
    "media-feature-range-notation": null
  },
  "overrides": [
    {
      "files": ["src/assets/**/*.css"],
      "rules": {
        "color-no-hex": null,
        "declaration-property-unit-disallowed-list": null,
        "declaration-property-value-disallowed-list": null
      }
    }
  ]
}
```

（`src/assets/themes.css` 是 token 定义源，豁免三条守卫规则；其余规则为降噪关闭，守卫只聚焦 hex/px 字号/border:none 三类退化。）

`package.json` scripts 追加：

```json
"lint:styles": "stylelint \"src/**/*.{vue,css}\""
```

- [ ] **Step 3: 运行验证**

Run: `cd frontend && npm run lint:styles; echo "exit=$?"`
Expected: 输出大量 warning（迁移前存量），`exit=0`

- [ ] **Step 4: Commit**

```bash
git add frontend/.stylelintrc.json frontend/package.json frontend/package-lock.json
git commit -m "chore(frontend): add stylelint guard against hex colors and px font-sizes (warn level)"
```

---

### Task 13: NewsPanel 迁移（打样 1/5 · 列表 + 空态）

**Files:**
- Modify: `frontend/src/terminal/panels/NewsPanel.vue`（100 行，已用 PanelHeader）

**Interfaces:**
- Consumes: 迁移模式库 P1–P7；`EmptyState`（Task 6 精修版）

- [ ] **Step 1: 应用迁移模式库**

通读文件后应用：P3（手写空态块 → `<EmptyState title="暂无资讯" description="..." />`）、P4（scoped 内 px 字号/padding → token）、P5（颜色 → token）、P7（根元素外壳清理为 `height:100%; display:flex; flex-direction:column; overflow:hidden`）。列表项结构不变，仅样式 token 化；新闻条目时间/来源用 12px tertiary（`--font-xs` + `--color-text-tertiary`）。

- [ ] **Step 2: 自检 + 验证**

```bash
cd frontend
grep -nE 'font-size:\s*[0-9]+px|#[0-9a-fA-F]{3,8}\b|rgba?\(' src/terminal/panels/NewsPanel.vue
npm run typecheck && npm run test
```

Expected: grep 无输出；typecheck/测试通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/NewsPanel.vue
git commit -m "refactor(frontend): migrate NewsPanel to shared EmptyState and tokens"
```

---

### Task 14: PositionPanel 迁移（打样 2/5 · KPI + 表格）

**Files:**
- Modify: `frontend/src/terminal/panels/PositionPanel.vue`（160 行，无 header，顶部 `.summary-row`）

**Interfaces:**
- Consumes: `PanelHeader`（Task 4）、`StatItem`（Task 7）、`PanelTable`（Task 5）、迁移模式库

- [ ] **Step 1: 应用迁移模式库**

1. P1：顶部补 `<PanelHeader title="持仓" :controls="[{ icon: 'refresh', title: '刷新', action: load, loading }]" />`（action/loading 用该面板现有的刷新函数与状态）。
2. KPI：`.summary-row` 内的各统计块换成 `<StatItem label="..." :value="..." :delta="...">`，外层容器保留但样式改为 `display: flex; gap: var(--space-xl); padding: var(--space-sm) var(--panel-padding); border-bottom: 1px solid var(--color-border-subtle);`。
3. P2：持仓表格换 `<PanelTable>`，金额/盈亏列 `align: 'right'` + 合适 `format`，盈亏列 `colorize: true`。
4. P3：空态/错误态换 `EmptyState`/`ErrorState`。
5. P4/P5/P7：样式 token 化 + 外壳清理。

- [ ] **Step 2: 自检 + 验证**

```bash
cd frontend
grep -nE 'font-size:\s*[0-9]+px|#[0-9a-fA-F]{3,8}\b|rgba?\(' src/terminal/panels/PositionPanel.vue
npm run typecheck && npm run test
```

Expected: grep 无输出；全部通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/PositionPanel.vue
git commit -m "refactor(frontend): migrate PositionPanel to PanelHeader/StatItem/PanelTable"
```

---

### Task 15: WatchlistPanel 迁移（打样 3/5 · 表格 + 刷新闪烁）

**Files:**
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`（576 行，已用 PanelHeader）

**Interfaces:**
- Consumes: `PanelTable`、`EmptyState`、`useFlashOnUpdate`（Task 9）

- [ ] **Step 1: 应用迁移模式库**

1. P2：自选股表格换 `PanelTable`（代码/名称/现价/涨跌幅/成交额列；涨跌幅 `format: 'percent'` + `colorize: true`；保留现有的右键菜单与行点击跳转逻辑——用 `PanelTable` 的 `@row-click` 与 `rowTestId`，右键菜单通过 `#action` 插槽或保留行级监听实现，行为不得回退）。
2. 涨跌闪烁：对现价/涨跌幅单元格应用 Task 9 的 `useFlashOnUpdate`。由于 PanelTable 单元格是格式化渲染，闪烁通过行级 class 实现：在 panel 中为每个 symbol 维护 flash 状态（watch 数据源的 price 字段，变化时在该行根元素加 `.flash-up`/`.flash-down`，600ms 后移除）。实现方式：利用 `rowKey` + 一个 `flashMap: Record<string, FlashClass>`，配合 PanelTable 的 `:row-class` 若不存在则给 PanelTable 加一个可选 prop `rowClass?: (row: any) => string`（同步补一个 PanelTable 测试：rowClass 返回的 class 出现在行元素上）。
3. P3：空态换 `EmptyState`（该面板自带的 `.empty-icon` 32px 定义删除，用组件默认）。
4. P7：删除 scoped 内的 `box-shadow`（本文件是审计发现的 7 个自绘阴影面板之一）。
5. P4/P5：样式 token 化。

- [ ] **Step 2: 自检 + 验证**

```bash
cd frontend
grep -nE 'font-size:\s*[0-9]+px|#[0-9a-fA-F]{3,8}\b|rgba?\(|box-shadow' src/terminal/panels/WatchlistPanel.vue
npm run typecheck && npm run test
```

Expected: grep 无输出；全部通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/WatchlistPanel.vue frontend/src/terminal/components/panel/PanelTable.vue frontend/src/terminal/__tests__/PanelTable.test.ts
git commit -m "refactor(frontend): migrate WatchlistPanel to PanelTable with price-flash rows"
```

---

### Task 16: MarketOverviewPanel 迁移（打样 4/5 · 多区块）

**Files:**
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`（734 行，已用 PanelHeader）

**Interfaces:**
- Consumes: 迁移模式库；`.section-title` 全局类（Task 1）

- [ ] **Step 1: 应用迁移模式库**

1. 各区块标题统一为 `<h4 class="section-title">`（全局类已定义），区块间用 `border-top: 1px solid var(--color-border-subtle)` + `padding: var(--space-sm) var(--panel-padding)` 分区，删除嵌套卡片样式。
2. 指数/统计数据块换成 `StatItem`；板块表格换 `PanelTable`。
3. P3：空/错/加载态换三件套。
4. P4/P5/P7：token 化 + 外壳清理；ECharts 若存在则按 P6 收口。

- [ ] **Step 2: 自检 + 验证**

```bash
cd frontend
grep -nE 'font-size:\s*[0-9]+px|#[0-9a-fA-F]{3,8}\b|rgba?\(' src/terminal/panels/MarketOverviewPanel.vue
npm run typecheck && npm run test
```

Expected: grep 无输出；全部通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/MarketOverviewPanel.vue
git commit -m "refactor(frontend): migrate MarketOverviewPanel to unified sections and StatItem"
```

---

### Task 17: CandlestickPanel 迁移（打样 5/5 · 图表 + 工具栏）

**Files:**
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`（1107 行，工具栏自绘）
- 可能涉及: `frontend/src/terminal/panels/candlestick/` 子目录组件

**Interfaces:**
- Consumes: `PanelHeader`（controls 插槽）、`useChartTheme`、`buildKlineOption`、迁移模式库

- [ ] **Step 1: 应用迁移模式库**

1. P1：顶部工具栏（周期切换、指标开关、画线工具等）整合为 `PanelHeader`：`title` 为当前合约名，`tabs` 放周期切换（`activeTab` 绑定现有周期状态），其余做成 `controls`。画线/指标等自绘按钮组若超出 controls 能力，保留在 header 下方一行作为次级工具条，但样式必须 token 化、按钮统一用全局 `.btn`。
2. P6：图表 option 走 `useChartTheme`/`buildKlineOption`（本面板若已是该路径则仅清理残留 hex）。
3. P7：删除 scoped 内 `box-shadow`（本文件在 7 个自绘阴影面板清单中）。
4. P4/P5：样式 token 化。
5. 不改图表交互逻辑（十字线、画线、指标计算），纯样式与骨架迁移。

- [ ] **Step 2: 自检 + 验证**

```bash
cd frontend
grep -nE 'font-size:\s*[0-9]+px|box-shadow' src/terminal/panels/CandlestickPanel.vue
grep -nE '#[0-9a-fA-F]{3,8}\b|rgba?\(' src/terminal/panels/CandlestickPanel.vue | grep -v 'getIcon'
npm run typecheck && npm run test
```

Expected: grep 无输出（或仅剩 getIcon 内 SVG 固有内容）；全部通过

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/CandlestickPanel.vue frontend/src/terminal/panels/candlestick/
git commit -m "refactor(frontend): migrate CandlestickPanel toolbar to PanelHeader and token cleanup"
```

---

### Task 18: Phase 1 验收门

**Files:**
- 无新增；产出验收记录

- [ ] **Step 1: 全量自动检查**

```bash
cd frontend
npm run typecheck && npm run test && npm run lint:styles && npm run build
```

Expected: typecheck 无错误；全部测试通过；stylelint 仅 warning 且 exit 0；构建成功

- [ ] **Step 2: 走查对比（与用户一起）**

`wails dev` 启动应用，逐项走查并截图记录：

- [ ] 5 个打样面板（自选股/K线/持仓/资讯/市场概览）在**亮主题 + 默认密度**下的长相：header 高度/字号一致、表格行高一致、空态一致、KPI 为 StatItem 样式
- [ ] 切换**暗主题**：图表面板坐标轴/网格颜色正确（Task 11 修复点）
- [ ] 切换**三档密度**：已迁移面板的字号/行高/间距随密度变化（token 化生效）
- [ ] 自选股面板价格刷新时有涨跌闪烁（Task 15）
- [ ] 抽查对比度：正文 ≥ 4.5:1

- [ ] **Step 3: 用户确认**

用户确认 5 个打样面板的"标准长相"后，Phase 1 关闭，进入 Phase 2 四批迁移（另行出计划）。

- [ ] **Step 4: Commit 验收记录（如有修正）**

若走查发现问题，修复后提交；无问题则本任务无 commit。

---

## Self-Review 记录

- **Spec coverage**：规格 §3.1→Task 1/2/4；§3.2→Task 1/4/P4；§3.3→Task 5；§3.4→Task 7；§3.5→Task 6/8；§3.6→Task 10/11；§3.7→Task 3/9/1(flash)；§3.8→Task 1/2/P7；§4.1→Task 1/2；§4.2→Task 3–10；§4.4→Task 12；§5 Phase 0→Task 1–12、Phase 1→Task 13–18。Phase 2 不在本计划（打样验收后另行出计划）。
- **Placeholder scan**：面板迁移任务（13–17）以"迁移模式库"完整代码块 + 逐面板差异说明承载，无 TBD/TODO。
- **Type consistency**：`PanelTabs` emit `change`、`PanelHeader` emit `tabChange`（与现有代码一致）；`Column.mono`、`FlashClass`、`ChartThemeColors` 扩展字段在消费任务前定义。

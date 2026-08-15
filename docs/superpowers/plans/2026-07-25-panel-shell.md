# PanelShell — Unified Loading/Error/Empty State Component

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a reusable `<PanelShell>` component that standardizes loading, error, empty, and loaded state rendering across all 84 terminal panels, replacing ad-hoc `v-if` / `v-else` / try-catch patterns.

**Architecture:** A thin Vue wrapper component that takes a `state` prop (enum) and optional `error` message + `@retry` event. Panels wrap their content in `<PanelShell>` slots. Existing panels are migrated incrementally — start with 10 highest-traffic panels, then batch the rest.

**Tech Stack:** Vue 3 Composition API, `<script setup lang="ts">`, scoped CSS with CSS variables

**Design:**

```
<PanelShell :state="state" :error="error" @retry="loadData">
  <template #loaded>
    <!-- actual panel content -->
  </template>
  <template #empty>
    <!-- optional custom empty state, default: "暂无数据" -->
  </template>
</PanelShell>
```

`state` values: `'loading' | 'loaded' | 'error' | 'empty'`

---

### Task 1: Create PanelShell component

**Files:**
- Create: `frontend/src/terminal/components/panel/PanelShell.vue`
- Test: `frontend/src/terminal/components/panel/__tests__/PanelShell.spec.ts`

**Interfaces:**
- Consumes: none (standalone component)
- Produces: `<PanelShell>` component with 4 states, 2 slots (`#loaded`, `#empty`)

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/terminal/components/panel/__tests__/PanelShell.spec.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelShell from '../PanelShell.vue'

describe('PanelShell', () => {
  it('renders spinner when state is loading', () => {
    const wrapper = mount(PanelShell, { props: { state: 'loading' } })
    expect(wrapper.find('[data-testid="panel-loading"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="panel-loaded"]').exists()).toBe(false)
  })

  it('renders loaded slot content when state is loaded', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'loaded' },
      slots: { loaded: '<div data-testid="content">Hello</div>' },
    })
    expect(wrapper.find('[data-testid="content"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="panel-loading"]').exists()).toBe(false)
  })

  it('renders error message and retry button when state is error', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'error', error: 'API failed' },
    })
    expect(wrapper.find('[data-testid="panel-error"]').text()).toContain('API failed')
    expect(wrapper.find('[data-testid="panel-retry-btn"]').exists()).toBe(true)
  })

  it('emits retry event when retry button clicked', async () => {
    const wrapper = mount(PanelShell, { props: { state: 'error', error: 'err' } })
    await wrapper.find('[data-testid="panel-retry-btn"]').trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
    expect(wrapper.emitted('retry')!.length).toBe(1)
  })

  it('renders default empty state when state is empty', () => {
    const wrapper = mount(PanelShell, { props: { state: 'empty' } })
    expect(wrapper.text()).toContain('暂无数据')
  })

  it('renders custom empty slot when provided', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'empty' },
      slots: { empty: '<div data-testid="custom-empty">No items</div>' },
    })
    expect(wrapper.find('[data-testid="custom-empty"]').exists()).toBe(true)
  })

  it('emits retry on Command+K / Escape focus trigger', () => {
    const wrapper = mount(PanelShell, { props: { state: 'error', error: 'x' } })
    // Retry button should be focusable
    expect(wrapper.find('[data-testid="panel-retry-btn"]').attributes('tabindex')).toBe('0')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/terminal/components/panel/__tests__/PanelShell.spec.ts
```
Expected: FAIL (module not found)

- [ ] **Step 3: Write minimal implementation**

Create `frontend/src/terminal/components/panel/PanelShell.vue`:

```vue
<script setup lang="ts">
defineProps<{
  state: 'loading' | 'loaded' | 'error' | 'empty'
  error?: string
}>()

defineEmits<{
  retry: []
}>()
</script>

<template>
  <div class="panel-shell" role="region" :aria-busy="state === 'loading'">
    <!-- Loading -->
    <div v-if="state === 'loading'" class="panel-shell-loading" data-testid="panel-loading">
      <div class="panel-shell-spinner" aria-label="加载中" />
      <span class="panel-shell-text">加载中…</span>
    </div>

    <!-- Error -->
    <div v-else-if="state === 'error'" class="panel-shell-error" data-testid="panel-error">
      <span class="panel-shell-error-icon" aria-hidden="true">⚠</span>
      <p class="panel-shell-error-message">{{ error }}</p>
      <button
        class="panel-shell-retry-btn"
        data-testid="panel-retry-btn"
        tabindex="0"
        @click="$emit('retry')"
      >
        重试
      </button>
    </div>

    <!-- Empty -->
    <div v-else-if="state === 'empty'" class="panel-shell-empty" data-testid="panel-empty">
      <slot name="empty">
        <span class="panel-shell-empty-default">暂无数据</span>
      </slot>
    </div>

    <!-- Loaded content -->
    <div v-else class="panel-shell-loaded" data-testid="panel-loaded">
      <slot name="loaded" />
    </div>
  </div>
</template>

<style scoped>
.panel-shell {
  display: flex;
  flex-direction: column;
  min-height: 120px;
}

.panel-shell-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--muted, #888);
  font-size: var(--font-sm, 13px);
}

.panel-shell-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border, #e0e0e0);
  border-top-color: var(--accent, #4a90d9);
  border-radius: 50%;
  animation: panel-shell-spin 0.7s linear infinite;
}

@keyframes panel-shell-spin {
  to { transform: rotate(360deg); }
}

.panel-shell-text { user-select: none; }

.panel-shell-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--danger, #d32f2f);
  font-size: var(--font-sm, 13px);
}

.panel-shell-error-icon { font-size: 20px; }

.panel-shell-error-message {
  margin: 0;
  text-align: center;
  word-break: break-word;
  color: var(--muted, #888);
}

.panel-shell-retry-btn {
  padding: 4px 14px;
  border: 1px solid var(--border, #e0e0e0);
  border-radius: var(--radius-sm, 4px);
  background: var(--surface, #f5f5f5);
  color: var(--text, #222);
  cursor: pointer;
  font-size: var(--font-sm, 13px);
  outline: none;
}

.panel-shell-retry-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--accent, #4a90d9);
}

.panel-shell-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  color: var(--muted, #888);
  font-size: var(--font-sm, 13px);
}
</style>
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend && npx vitest run src/terminal/components/panel/__tests__/PanelShell.spec.ts
```
Expected: PASS (all 7 tests)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelShell.vue frontend/src/terminal/components/panel/__tests__/PanelShell.spec.ts
git commit -m "feat(panel): add PanelShell unified loading/error/empty component"
```

---

### Task 2: Migrate first 10 high-traffic panels to use PanelShell

**Files:**
- Modify: `frontend/src/terminal/panels/WelcomePanel.vue`
- Modify: `frontend/src/terminal/panels/MarketOverviewPanel.vue`
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`
- Modify: `frontend/src/terminal/panels/PortfolioSummary.vue`
- Modify: `frontend/src/terminal/panels/TradeHistory.vue`
- Modify: `frontend/src/terminal/panels/FinancialsPanel.vue`
- Modify: `frontend/src/terminal/panels/GovDataPanel.vue`
- Modify: `frontend/src/terminal/panels/MarketScannerPanel.vue`
- Modify: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Modify: `frontend/src/terminal/panels/IndicatorPanel.vue`

**Pattern for each panel:**

Replace:
```vue
<script setup lang="ts">
const loading = ref(true)
const error = ref('')
// ...
try {
  loading.value = true
  // fetch data
  loading.value = false
} catch (e: any) {
  error.value = e.message || '加载失败'
  loading.value = false
}
</script>

<template>
  <div v-if="loading" class="loading">加载中…</div>
  <div v-else-if="error" class="error">{{ error }} <button @click="load">重试</button></div>
  <div v-else>
    <!-- content -->
  </div>
</template>
```

With:
```vue
<script setup lang="ts">
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')
const error = ref('')

async function loadData() {
  state.value = 'loading'
  error.value = ''
  try {
    // fetch data
    state.value = hasData ? 'loaded' : 'empty'
  } catch (e: any) {
    error.value = e?.message || String(e)
    state.value = 'error'
  }
}

onMounted(loadData)
</script>

<template>
  <PanelShell :state="state" :error="error" @retry="loadData">
    <template #loaded>
      <!-- content -->
    </template>
  </PanelShell>
</template>
```

For each panel, the migration steps are identical — show only the first as a complete example:

**Step 1a — Migrate WelcomePanel:**

Read `frontend/src/terminal/panels/WelcomePanel.vue`, then:

1. Add `import PanelShell from '@/terminal/components/panel/PanelShell.vue'`
2. Replace loading/error refs with a single `const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loading')` and `const loadError = ref('')`
3. Wrap template content in `<PanelShell :state="state" :error="loadError" @retry="load"><template #loaded>…</template></PanelShell>`
4. Remove the standalone loading/error divs

- [ ] **Step 2a: Run tests to verify**

```bash
cd frontend && npx vitest run
```
Expected: PASS (all tests, no regression)

- [ ] **Step 3a: Repeat for MarketOverviewPanel, WatchlistPanel, PortfolioSummary, TradeHistory, FinancialsPanel, GovDataPanel, MarketScannerPanel, CandlestickPanel, IndicatorPanel**

Each follows the same pattern above. After each panel, run `npx vitest run src/terminal/panels/__tests__/{panel}.spec.ts` to confirm.

After all 10:
- [ ] **Step 4a: Commit**

```bash
git add frontend/src/terminal/panels/
git commit -m "refactor(panel): migrate 10 high-traffic panels to PanelShell"
```

---

### Task 3: Batch-migrate remaining panels

**Files:**
- Modify: All remaining 74 panel `.vue` files in `frontend/src/terminal/panels/`

**Same pattern as Task 2** — mechanical replacement of ad-hoc loading/error divs with `<PanelShell>`.

Group into 5 sub-commits to keep diffs reviewable:
- Batch A (15 panels): `AuditPanel` through `CBArbitragePanel` alphabetical
- Batch B (15 panels): `CoinBase*` through `DarkPoolPanel`
- Batch C (15 panels): `DepthComparisonPanel` through `LiquidationPanel`
- Batch D (15 panels): `MacroEconomicsPanel` through `RebalancePanel`
- Batch E (14 panels): `ResearchPanel` through `WhaleTransactionsPanel`

- [ ] **Step 1: Process Batch A** — apply pattern, `vitest run`, commit
- [ ] **Step 2: Process Batch B** — apply pattern, `vitest run`, commit
- [ ] **Step 3: Process Batch C** — apply pattern, `vitest run`, commit
- [ ] **Step 4: Process Batch D** — apply pattern, `vitest run`, commit
- [ ] **Step 5: Process Batch E** — apply pattern, `vitest run`, commit

Each batch commit message:
```
git commit -m "refactor(panel): migrate Batch {letter} to PanelShell"
```

---

### Task 4: Verify full test suite

- [ ] **Step 1: Run all frontend tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS (253+ tests)

- [ ] **Step 2: Run type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No type errors

- [ ] **Step 3: Final commit**

```bash
git add frontend/src/
git commit -m "chore(panel): verify PanelShell migration — all tests pass"
```

---

### Task 5: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entry**

```markdown
### Added
- [Panel] New PanelShell component unifying loading/error/empty states across all 84 terminal panels
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for PanelShell"
```

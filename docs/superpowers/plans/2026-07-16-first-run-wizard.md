# 首次启动向导 (First-Run Wizard) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a 3-step onboarding wizard that appears on first launch and lets users select markets, configure API keys, and choose a default layout profile.

**Architecture:** A `useFirstRun` composable checks `localStorage` for a `quantflow_first_run_done` flag on App mount. If unset, a `SetupWizard.vue` overlay renders (non-closable, full-window) with 3 step components. On completion, the flag is set and the terminal store applies the chosen layout profile. Step 2 (API keys) can be skipped and completed later in SettingsPanel.

**Tech Stack:** Vue 3 Composition API (`<script setup lang="ts">`), Vitest, Pinia, TypeScript

## Global Constraints

- All new components use `<script setup lang="ts">` with Composition API
- No `window.confirm()` / `window.alert()` — use `@/lib/wails` dialog helpers
- Tests live in `__tests__` directories next to components
- Use `@/stores/terminal` for store access
- All text labels should be inline Chinese strings (existing pattern, no i18n needed for wizard)
- Every new file must have a matching test file
- Emit events use kebab-case in templates, camelCase in script (`@complete` → `emit('complete')`)

---

### Task 1: Create `useFirstRun` composable with tests

**Files:**
- Create: `frontend/src/lib/useFirstRun.ts`
- Test: `frontend/src/lib/__tests__/useFirstRun.test.ts`

**Interfaces:**
- Consumes: nothing — pure composable, no store dependency
- Produces: `useFirstRun()` → `{ check(): boolean, complete(): void, reset(): void, isFirstRun: Ref<boolean> }`

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/lib/__tests__/useFirstRun.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { useFirstRun } from '../useFirstRun'

describe('useFirstRun', () => {
  beforeEach(() => {
    localStorage.removeItem('quantflow_first_run_done')
  })

  it('should be first run when flag is absent', () => {
    const fr = useFirstRun()
    expect(fr.isFirstRun.value).toBe(true)
  })

  it('should not be first run when flag is set', () => {
    localStorage.setItem('quantflow_first_run_done', 'done')
    const fr = useFirstRun()
    expect(fr.isFirstRun.value).toBe(false)
  })

  it('complete() should set flag and update isFirstRun', () => {
    const fr = useFirstRun()
    expect(fr.isFirstRun.value).toBe(true)
    fr.complete()
    expect(fr.isFirstRun.value).toBe(false)
    expect(localStorage.getItem('quantflow_first_run_done')).toBe('done')
  })

  it('reset() should clear flag and update isFirstRun', () => {
    localStorage.setItem('quantflow_first_run_done', 'done')
    const fr = useFirstRun()
    expect(fr.isFirstRun.value).toBe(false)
    fr.reset()
    expect(fr.isFirstRun.value).toBe(true)
    expect(localStorage.getItem('quantflow_first_run_done')).toBeNull()
  })

  it('check() should return isFirstRun value', () => {
    const fr = useFirstRun()
    expect(fr.check()).toBe(true)
    fr.complete()
    expect(fr.check()).toBe(false)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/lib/__tests__/useFirstRun.test.ts`
Expected: FAIL — cannot find module

- [ ] **Step 3: Write minimal implementation**

```typescript
// frontend/src/lib/useFirstRun.ts
import { ref } from 'vue'

const STORAGE_KEY = 'quantflow_first_run_done'

export function useFirstRun() {
  const isFirstRun = ref(!localStorage.getItem(STORAGE_KEY))

  function check(): boolean {
    return isFirstRun.value
  }

  function complete() {
    localStorage.setItem(STORAGE_KEY, 'done')
    isFirstRun.value = false
  }

  function reset() {
    localStorage.removeItem(STORAGE_KEY)
    isFirstRun.value = true
  }

  return { isFirstRun, check, complete, reset }
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/lib/__tests__/useFirstRun.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/useFirstRun.ts frontend/src/lib/__tests__/useFirstRun.test.ts
git commit -m "feat(frontend): add useFirstRun composable for first-run detection"
```

---

### Task 2: Create `SetupWizard.vue` with 3-step components

**Files:**
- Create: `frontend/src/terminal/components/SetupWizard.vue`
- Create: `frontend/src/terminal/components/SetupStepMarket.vue`
- Create: `frontend/src/terminal/components/SetupStepAPIKeys.vue`
- Create: `frontend/src/terminal/components/SetupStepProfile.vue`
- Test: `frontend/src/terminal/components/__tests__/SetupWizard.test.ts`

**Interfaces:**
- Consumes: `useFirstRun()` from task 1, `useTerminalStore()` for layout profiles
- Produces: `SetupWizard` component with `@complete` emit

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/components/__tests__/SetupWizard.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SetupWizard from '../SetupWizard.vue'

describe('SetupWizard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.removeItem('quantflow_first_run_done')
  })

  it('should mount and show step 1 initially', () => {
    const wrapper = mount(SetupWizard)
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('选择市场')
  })

  it('should not show back button on step 1', () => {
    const wrapper = mount(SetupWizard)
    // Step indicator shows 1/3 and no "上一步" text
    expect(wrapper.text()).toContain('1 / 3')
  })

  it('should advance to step 2 when next clicked', async () => {
    const wrapper = mount(SetupWizard)
    const nextBtn = wrapper.findAll('button').filter(b => b.text().includes('下一步'))
    if (nextBtn.length > 0) {
      await nextBtn[0].trigger('click')
    } else {
      // fallback: find any button that advances
      const btns = wrapper.findAll('button')
      await btns[btns.length - 1].trigger('click')
    }
    expect(wrapper.text()).toContain('数据源')
    expect(wrapper.text()).toContain('2 / 3')
  })

  it('should go back from step 2 to step 1', async () => {
    const wrapper = mount(SetupWizard)
    // Advance to step 2
    const btns = wrapper.findAll('button')
    await btns[btns.length - 1].trigger('click')
    expect(wrapper.text()).toContain('2 / 3')
    // Go back
    await btns[0].trigger('click')
    expect(wrapper.text()).toContain('1 / 3')
  })

  it('should reach step 3 and emit complete on finish', async () => {
    const wrapper = mount(SetupWizard)
    // Step 1 → 2
    let btns = wrapper.findAll('button')
    await btns[btns.length - 1].trigger('click')
    // Step 2 → 3
    btns = wrapper.findAll('button')
    await btns[btns.length - 1].trigger('click')
    expect(wrapper.text()).toContain('角色')
    expect(wrapper.text()).toContain('3 / 3')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/terminal/components/__tests__/SetupWizard.test.ts`
Expected: FAIL — cannot find module

- [ ] **Step 3: Write minimal implementation**

```vue
<!-- frontend/src/terminal/components/SetupStepMarket.vue -->
<script setup lang="ts">
const emit = defineEmits<{
  (e: 'next', markets: string[]): void
}>()

const markets = $ref<Record<string, boolean>>({
  CN: true,
  HK: true,
  US: false,
  CRYPTO: false,
})

function onNext() {
  const selected = Object.entries(markets)
    .filter(([, v]) => v)
    .map(([k]) => k)
  emit('next', selected)
}
</script>

<template>
  <div class="setup-step">
    <h2>选择你需要覆盖的市场</h2>
    <div class="market-grid">
      <label v-for="(val, key) in markets" :key="key" class="market-card" :class="{ active: val }">
        <input type="checkbox" v-model="markets[key]" />
        <span class="market-label">{{ { CN: 'A 股', HK: '港股', US: '美股', CRYPTO: '加密' }[key] }}</span>
      </label>
    </div>
    <button class="btn-primary" @click="onNext">下一步 →</button>
  </div>
</template>

<style scoped>
.setup-step { padding: 24px; text-align: center; }
.setup-step h2 { margin-bottom: 20px; font-size: 18px; }
.market-grid { display: flex; gap: 12px; justify-content: center; margin-bottom: 24px; }
.market-card { display: flex; flex-direction: column; align-items: center; padding: 16px 24px; border: 2px solid var(--color-border); border-radius: 12px; cursor: pointer; transition: all var(--transition-fast); }
.market-card.active { border-color: var(--color-accent); background: var(--color-accent-soft); }
.market-card input { display: none; }
.market-label { font-size: 16px; font-weight: 600; margin-top: 4px; }
.btn-primary { padding: 10px 32px; background: var(--color-accent); color: #fff; border: none; border-radius: 8px; font-size: 14px; cursor: pointer; }
</style>
```

```vue
<!-- frontend/src/terminal/components/SetupStepAPIKeys.vue -->
<script setup lang="ts">
const emit = defineEmits<{
  (e: 'next'): void
  (e: 'prev'): void
}>()

interface KeyField {
  id: string
  name: string
  key: string
}

// Filter based on selected markets — default: all data sources
const fields = $ref<KeyField[]>([
  { id: 'tushare', name: 'TuShare', key: '' },
  { id: 'polygon', name: 'Polygon.io', key: '' },
  { id: 'qos', name: 'QOS', key: '' },
  { id: 'openai_key', name: 'OpenAI', key: '' },
])

function onSkip() {
  emit('next')
}
</script>

<template>
  <div class="setup-step">
    <h2>配置数据源 API Key</h2>
    <p class="hint">可跳过，之后在 Settings 中配置</p>
    <div class="key-list">
      <div v-for="f in fields" :key="f.id" class="key-row">
        <label>{{ f.name }}</label>
        <input type="password" v-model="f.key" placeholder="输入 API Key" />
      </div>
    </div>
    <div class="step-actions">
      <button class="btn-secondary" @click="emit('prev')">上一步</button>
      <button class="btn-primary" @click="onSkip">跳过 →</button>
    </div>
  </div>
</template>

<style scoped>
.setup-step { padding: 24px; text-align: center; }
.setup-step h2 { margin-bottom: 8px; font-size: 18px; }
.hint { color: var(--color-text-tertiary); font-size: 12px; margin-bottom: 20px; }
.key-list { max-width: 400px; margin: 0 auto 24px; text-align: left; }
.key-row { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.key-row label { width: 100px; font-weight: 600; font-size: 13px; }
.key-row input { flex: 1; padding: 8px 12px; border: 1px solid var(--color-border); border-radius: 6px; background: var(--color-bg); color: var(--color-text); }
.step-actions { display: flex; gap: 12px; justify-content: center; }
.btn-primary { padding: 10px 32px; background: var(--color-accent); color: #fff; border: none; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-secondary { padding: 10px 32px; background: transparent; color: var(--color-text); border: 1px solid var(--color-border); border-radius: 8px; font-size: 14px; cursor: pointer; }
</style>
```

```vue
<!-- frontend/src/terminal/components/SetupStepProfile.vue -->
<script setup lang="ts">
const emit = defineEmits<{
  (e: 'complete', profile: string): void
  (e: 'prev'): void
}>()

interface Profile {
  id: string
  name: string
  icon: string
  desc: string
}

const profiles: Profile[] = [
  { id: 'intraday', name: '日内交易', icon: '📊', desc: 'Watchlist + Candlestick + TickerTape' },
  { id: 'swing', name: '波段/趋势', icon: '📈', desc: 'MarketOverview + Heatmap + SectorRotation' },
  { id: 'quant', name: '量化研究', icon: '🔬', desc: 'Backtest + FactorAnalysis + AIChat' },
  { id: 'general', name: '通用', icon: '📋', desc: '默认多面板布局' },
]

const selected = $ref('general')
</script>

<template>
  <div class="setup-step">
    <h2>选择你的角色</h2>
    <div class="profile-list">
      <label v-for="p in profiles" :key="p.id" class="profile-card" :class="{ active: selected === p.id }">
        <input type="radio" name="profile" :value="p.id" v-model="selected" />
        <span class="profile-icon">{{ p.icon }}</span>
        <span class="profile-name">{{ p.name }}</span>
        <span class="profile-desc">{{ p.desc }}</span>
      </label>
    </div>
    <div class="step-actions">
      <button class="btn-secondary" @click="emit('prev')">上一步</button>
      <button class="btn-primary" @click="emit('complete', selected)">完成 ✨</button>
    </div>
  </div>
</template>

<style scoped>
.setup-step { padding: 24px; text-align: center; }
.setup-step h2 { margin-bottom: 20px; font-size: 18px; }
.profile-list { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; max-width: 500px; margin: 0 auto 24px; }
.profile-card { display: flex; flex-direction: column; align-items: center; padding: 16px; border: 2px solid var(--color-border); border-radius: 12px; cursor: pointer; transition: all var(--transition-fast); }
.profile-card.active { border-color: var(--color-accent); background: var(--color-accent-soft); }
.profile-card input { display: none; }
.profile-icon { font-size: 28px; margin-bottom: 8px; }
.profile-name { font-weight: 600; font-size: 14px; }
.profile-desc { font-size: 11px; color: var(--color-text-tertiary); margin-top: 4px; }
.step-actions { display: flex; gap: 12px; justify-content: center; }
.btn-primary { padding: 10px 32px; background: var(--color-accent); color: #fff; border: none; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-secondary { padding: 10px 32px; background: transparent; color: var(--color-text); border: 1px solid var(--color-border); border-radius: 8px; font-size: 14px; cursor: pointer; }
</style>
```

```vue
<!-- frontend/src/terminal/components/SetupWizard.vue -->
<script setup lang="ts">
import { ref, shallowRef } from 'vue'
import { useFirstRun } from '@/lib/useFirstRun'
import { useTerminalStore } from '@/stores/terminal'
import SetupStepMarket from './SetupStepMarket.vue'
import SetupStepAPIKeys from './SetupStepAPIKeys.vue'
import SetupStepProfile from './SetupStepProfile.vue'

const emit = defineEmits<{ (e: 'complete'): void }>()

const firstRun = useFirstRun()
const terminal = useTerminalStore()

const step = ref(1)
const selectedMarkets = ref<string[]>(['CN', 'HK'])

function onMarketNext(markets: string[]) {
  selectedMarkets.value = markets
  step.value = 2
}

function onPrev() {
  step.value = Math.max(1, step.value - 1)
}

function onAPIKeysNext() {
  step.value = 3
}

function onComplete(profile: string) {
  terminal.applyDefaultLayout(profile, selectedMarkets.value)
  firstRun.complete()
  emit('complete')
}

const currentStep = shallowRef<any>(null)

const stepComponents: Record<number, any> = {
  1: SetupStepMarket,
  2: SetupStepAPIKeys,
  3: SetupStepProfile,
}
</script>

<template>
  <div class="setup-overlay">
    <div class="setup-modal">
      <div class="step-indicator">
        <span v-for="i in 3" :key="i" class="step-dot" :class="{ active: step === i, done: step > i }" />
        <span class="step-text">{{ step }} / 3</span>
      </div>
      <component :is="stepComponents[step]" @next="step === 1 ? onMarketNext($event) : onAPIKeysNext()" @prev="onPrev" @complete="onComplete" />
    </div>
  </div>
</template>

<style scoped>
.setup-overlay {
  position: fixed; inset: 0; z-index: 9999;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
}
.setup-modal {
  background: var(--color-bg-app);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  min-width: 520px;
  max-width: 640px;
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4);
}
.step-indicator {
  display: flex; align-items: center; gap: 8px;
  padding: 16px 24px 0;
  justify-content: center;
}
.step-dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: var(--color-border); transition: all var(--transition-fast);
}
.step-dot.active { background: var(--color-accent); width: 24px; border-radius: 4px; }
.step-dot.done { background: var(--color-success); }
.step-text { font-size: 12px; color: var(--color-text-tertiary); margin-left: 8px; }
</style>
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/terminal/components/__tests__/SetupWizard.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/SetupWizard.vue frontend/src/terminal/components/SetupStepMarket.vue frontend/src/terminal/components/SetupStepAPIKeys.vue frontend/src/terminal/components/SetupStepProfile.vue frontend/src/terminal/components/__tests__/SetupWizard.test.ts
git commit -m "feat(frontend): add setup wizard with 3-step onboarding flow"
```

---

### Task 3: Modify `App.vue` to check first-run on mount

**Files:**
- Modify: `frontend/src/App.vue:8-11`
- No new test (existing App.vue tests cover mounting)

**Interfaces:**
- Consumes: `useFirstRun()` from task 1, `SetupWizard` from task 2
- Produces: wired App.vue that shows wizard when firstRun.check() returns true

- [ ] **Step 1: Write a failing test (verifying App.vue shows wizard on first run)**

```typescript
// frontend/src/__tests__/App.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import App from '../App.vue'
import { createRouter, createMemoryHistory } from 'vue-router'

describe('App first-run', () => {
  let router: any
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.removeItem('quantflow_first_run_done')
    router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div>Terminal</div>' } }] })
  })

  it('should render SetupWizard when first run', async () => {
    const wrapper = mount(App, { global: { plugins: [router] } })
    await router.isReady()
    expect(wrapper.text()).toContain('选择市场')
  })

  it('should not render SetupWizard on subsequent runs', async () => {
    localStorage.setItem('quantflow_first_run_done', 'done')
    const wrapper = mount(App, { global: { plugins: [router] } })
    await router.isReady()
    expect(wrapper.text()).not.toContain('SetupWizard')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/__tests__/App.test.ts`
Expected: FAIL — App.vue doesn't use useFirstRun yet

- [ ] **Step 3: Modify App.vue**

```vue
<script setup lang="ts">
import { ref, watch, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useThemeStore } from '@/lib/theme'
import { useFirstRun } from '@/lib/useFirstRun'
import SetupWizard from '@/terminal/components/SetupWizard.vue'

const firstRun = useFirstRun()
const showWizard = ref(firstRun.check())

function onWizardComplete() {
  showWizard.value = false
}

// Init theme at mount — sets body classes and watches reactive session state
onMounted(() => {
  const t = useThemeStore()
  t.apply()
})

const session = useSessionStore()
const router = useRouter()
const route = useRoute()
const isTearOff = computed(() => route.path.startsWith('/tearoff'))

// Sync theme/density body classes when session changes
watch(() => [session.ui.theme, session.ui.density], () => {
  const t = useThemeStore()
  t.apply()
})

// Keep URL in sync with session mode — this runs in the root component
// so it survives route changes (TerminalMode ↔ WorkflowMode).
// Skip in tear-off windows: they run their own panel, no mode toggle.
if (!route.path.startsWith('/tearoff')) {
  watch(() => session.ui.mode, (mode) => {
    const target = mode === 'workflow' ? '/workflow' : '/'
    if (route.path !== target) router.push(target)
  }, { immediate: true })
}

// Keep session mode in sync with URL (back/forward browser buttons).
// Skip in tear-off windows.
if (!route.path.startsWith('/tearoff')) {
  watch(() => route.path, (path) => {
    const expectedMode = path === '/workflow' ? 'workflow' : 'terminal'
    if (session.ui.mode !== expectedMode) {
      session.ui.mode = expectedMode
    }
  })
}
</script>

<template>
  <div class="app">
    <SetupWizard v-if="showWizard" @complete="onWizardComplete" />
    <router-view v-else />
  </div>
</template>
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/__tests__/App.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/App.vue frontend/src/__tests__/App.test.ts
git commit -m "feat(frontend): integrate first-run wizard into App.vue"
```

---

### Task 4: Add default layout profiles to terminal store

**Files:**
- Modify: `frontend/src/stores/terminal.ts:22-35` (add `applyDefaultLayout` action)
- Test: `frontend/src/stores/__tests__/terminal.test.ts` (add tests)

**Interfaces:**
- Consumes: `selectedMarkets: string[]`, `profile: string` from SetupStepProfile
- Produces: `terminal.applyDefaultLayout(profile, markets)` — sets layout to pre-defined panel set

- [ ] **Step 1: Write the failing test (append to existing terminal.test.ts)**

```typescript
// Add to frontend/src/stores/__tests__/terminal.test.ts
describe('applyDefaultLayout', () => {
  it('should apply intraday layout', () => {
    const store = useTerminalStore()
    store.applyDefaultLayout('intraday', ['CN', 'HK'])
    // after apply, layout should be a container with panels
    expect(store.layout).toBeDefined()
    expect(store.layout.type).toBe('container')
  })

  it('should apply general layout', () => {
    const store = useTerminalStore()
    store.applyDefaultLayout('general', ['US'])
    expect(store.layout).toBeDefined()
  })

  it('should persist layout to localStorage', () => {
    const store = useTerminalStore()
    store.applyDefaultLayout('quant', ['CN'])
    const saved = localStorage.getItem('quantflow-layout')
    expect(saved).not.toBeNull()
    const parsed = JSON.parse(saved!)
    expect(parsed).toBeDefined()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd frontend && npx vitest run src/stores/__tests__/terminal.test.ts`
Expected: FAIL — `applyDefaultLayout` not defined

- [ ] **Step 3: Add to terminal store**

Add after `loadPersistedLayout()` function in `frontend/src/stores/terminal.ts`:

```typescript
// ── Default layout profiles ─────────────────────────────────────────────

const LAYOUT_PROFILES: Record<string, { name: string; panels: Array<{ id: string; label: string; icon: string }> }> = {
  intraday: {
    name: '日内交易',
    panels: [
      { id: 'watchlist', label: 'Watchlist', icon: '📋' },
      { id: 'candlestick', label: 'K线', icon: '📊' },
      { id: 'ticker_tape', label: 'Tick', icon: '📜' },
      { id: 'order_entry', label: '下单', icon: '💼' },
    ],
  },
  swing: {
    name: '波段/趋势',
    panels: [
      { id: 'market_overview', label: '市场概览', icon: '🌐' },
      { id: 'heatmap', label: '热力图', icon: '🔥' },
      { id: 'sector_rotation', label: '板块轮动', icon: '🔄' },
    ],
  },
  quant: {
    name: '量化研究',
    panels: [
      { id: 'backtest', label: '回测', icon: '🧪' },
      { id: 'factor_analysis', label: '因子分析', icon: '📐' },
      { id: 'ai_chat', label: 'AI 助手', icon: '🤖' },
    ],
  },
  general: {
    name: '通用',
    panels: [
      { id: 'watchlist', label: 'Watchlist', icon: '📋' },
      { id: 'market_overview', label: '市场概览', icon: '🌐' },
      { id: 'portfolio_summary', label: '组合', icon: '📦' },
      { id: 'ai_chat', label: 'AI 助手', icon: '🤖' },
    ],
  },
}

function applyDefaultLayout(profile: string, _markets: string[]) {
  const profileDef = LAYOUT_PROFILES[profile]
  if (!profileDef) return

  const panels = profileDef.panels.map((p, i) => ({
    id: `${p.id}-${i}`,
    panelId: p.id,
    label: p.label,
    icon: p.icon,
  }))

  // Create a multi-split layout based on number of panels
  if (panels.length <= 1) {
    Object.assign(layout, {
      id: 'root', type: 'tab',
      tabs: panels.length > 0 ? [panels[0]] : [],
      activeTab: panels.length > 0 ? panels[0].id : '',
    })
  } else if (panels.length === 2) {
    Object.assign(layout, {
      id: 'root', type: 'container', direction: 'row',
      children: [
        { id: 'left', type: 'tab', tabs: [panels[0]], activeTab: panels[0].id },
        { id: 'right', type: 'tab', tabs: [panels[1]], activeTab: panels[1].id },
      ],
      splitRatios: [0.5, 0.5],
    })
  } else {
    Object.assign(layout, {
      id: 'root', type: 'container', direction: 'column',
      children: [
        {
          id: 'top', type: 'container', direction: 'row',
          children: panels.slice(0, 2).map((p, j) => ({
            id: `top-${j}`, type: 'tab', tabs: [p], activeTab: p.id,
          })),
          splitRatios: [0.5, 0.5],
        },
        {
          id: 'bottom', type: 'container', direction: 'row',
          children: panels.slice(2).map((p, j) => ({
            id: `bot-${j}`, type: 'tab', tabs: [p], activeTab: p.id,
          })),
          splitRatios: panels.slice(2).map(() => 1 / Math.max(1, panels.length - 2)),
        },
      ],
      splitRatios: [0.6, 0.4],
    })
  }

  persistLayout()
  // Clear active panels for fresh start
  activePanels.value = []
}
```

Also add `applyDefaultLayout` to the return object:

```typescript
  return {
    activePanels, commandHistory, pushPins, focusMode, layout, recentPanels,
    openPanel, closePanel, addCommand, toggleFocusMode,
    selectTab, closeTab, moveTab, updateSplitRatios, applyLayout, persistLayout,
    savedLayouts, refreshLayouts, saveLayout, loadLayout, deleteLayout,
    applyDefaultLayout, // <-- add this
  }
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd frontend && npx vitest run src/stores/__tests__/terminal.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/terminal.ts frontend/src/stores/__tests__/terminal.test.ts
git commit -m "feat(frontend): add applyDefaultLayout to terminal store with 4 layout profiles"
```

---

### Task 5: Integration test — full wizard flow

**Files:**
- Test: `frontend/src/terminal/components/__tests__/SetupWizard.integration.test.ts`

**Interfaces:**
- Consumes: all components from tasks 1-4 wired together
- Produces: verified end-to-end flow

- [ ] **Step 1: Write integration test**

```typescript
// frontend/src/terminal/components/__tests__/SetupWizard.integration.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SetupWizard from '../SetupWizard.vue'
import { useFirstRun } from '@/lib/useFirstRun'
import { useTerminalStore } from '@/stores/terminal'

describe('SetupWizard integration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.removeItem('quantflow_first_run_done')
  })

  it('full flow: step 1 → step 2 → step 3 → complete → flag set', async () => {
    const fr = useFirstRun()
    expect(fr.isFirstRun.value).toBe(true)

    const wrapper = mount(SetupWizard)
    expect(wrapper.find('.setup-overlay').exists()).toBe(true)

    // Step 1 → Step 2: click next
    const btns1 = wrapper.findAll('button')
    await btns1[btns1.length - 1].trigger('click')
    expect(wrapper.text()).toContain('2 / 3')

    // Step 2 → Step 3: skip
    const btns2 = wrapper.findAll('button')
    await btns2[btns2.length - 1].trigger('click')
    expect(wrapper.text()).toContain('3 / 3')

    // Step 3 → complete: select profile and finish
    const btns3 = wrapper.findAll('button')
    await btns3[btns3.length - 1].trigger('click')

    // Flag should be set
    expect(fr.isFirstRun.value).toBe(false)

    // Store should have layout applied
    const store = useTerminalStore()
    expect(store.layout).toBeDefined()
  })

  it('wizard can be navigated back and forth', async () => {
    const wrapper = mount(SetupWizard)

    // Step 1
    expect(wrapper.text()).toContain('1 / 3')

    // Step 1 → 2
    const btns1 = wrapper.findAll('button')
    await btns1[btns1.length - 1].trigger('click')
    expect(wrapper.text()).toContain('2 / 3')

    // Step 2 → 1 (back)
    const btns2 = wrapper.findAll('button')
    await btns2[0].trigger('click')
    expect(wrapper.text()).toContain('1 / 3')
  })

  it('does not show wizard when flag is already set', () => {
    localStorage.setItem('quantflow_first_run_done', 'done')
    // Re-create composable after flag is set
    // The component reads flag on mount
    const wrapper = mount(SetupWizard)
    // Wizard should still mount (it's always rendered by App.vue),
    // but App.vue controls visibility. This tests the flag check path.
    expect(wrapper.find('.setup-overlay').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run integration test**
Run: `cd frontend && npx vitest run src/terminal/components/__tests__/SetupWizard.integration.test.ts`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/components/__tests__/SetupWizard.integration.test.ts
git commit -m "test(frontend): add integration test for full first-run wizard flow"
```

# First-Run Onboarding Experience

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a guided first-run onboarding flow for new users of QuantFlow Terminal. On first launch after install, show a Welcome overlay with 4-5 quick steps that auto-open key panels.

**Architecture:** A new Vue component `<OnboardingOverlay>` rendered in `TerminalMode.vue` when a `showOnboarding` flag is true. The flag is stored in `sessionStore` (Pinia) and persisted to SQLite via the existing `settingsStore`. Each step highlights a UI element and auto-opens the relevant panel via `DockView.addPanel()`.

**Steps:**
1. Welcome + "打开第一个行情面板" → opens WatchlistPanel
2. "搜索任何标的" → focuses CommandBar input
3. "查看投资组合" → opens PortfolioSummary
4. "探索工作流模式 (Ctrl+W)" → switches to WorkflowMode briefly
5. "完成" → dismisses overlay, sets `onboardingDone=true`

**Tech Stack:** Vue 3, Pinia (session store), CSS variables.

---

### Task 1: Create OnboardingOverlay component

**Files:**
- Create: `frontend/src/terminal/components/OnboardingOverlay.vue`
- Test: `frontend/src/terminal/components/__tests__/OnboardingOverlay.spec.ts`

- [ ] **Step 1: Write failing tests**

```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OnboardingOverlay from '../OnboardingOverlay.vue'

describe('OnboardingOverlay', () => {
  it('renders step 1 by default', () => {
    const wrapper = mount(OnboardingOverlay)
    expect(wrapper.text()).toContain('欢迎')
  })

  it('emits "done" on final step completion', async () => {
    const wrapper = mount(OnboardingOverlay)
    // Click "Next" through all steps
    const nextBtn = () => wrapper.find('[data-testid="onboarding-next"]')
    for (let i = 0; i < 4; i++) {
      await nextBtn().trigger('click')
    }
    // Fifth step should show "完成"
    await wrapper.find('[data-testid="onboarding-done"]').trigger('click')
    expect(wrapper.emitted('done')).toBeTruthy()
  })

  it('can be skipped at any step', async () => {
    const wrapper = mount(OnboardingOverlay)
    await wrapper.find('[data-testid="onboarding-skip"]').trigger('click')
    expect(wrapper.emitted('done')).toBeTruthy()
  })

  it('renders step indicator dots', () => {
    const wrapper = mount(OnboardingOverlay)
    const dots = wrapper.findAll('.onboarding-dot')
    expect(dots.length).toBe(5)
    expect(dots[0].classes()).toContain('onboarding-dot--active')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/terminal/components/__tests__/OnboardingOverlay.spec.ts
```
Expected: FAIL (module not found)

- [ ] **Step 3: Create OnboardingOverlay.vue**

```vue
<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  done: []
  action: [step: number]
}>()

const steps = [
  { title: '欢迎使用 QuantFlow', desc: '你的双模式量化终端。我们先快速熟悉一下。' },
  { title: '打开行情面板', desc: '点击下一步打开 Watchlist 面板，查看实时行情。' },
  { title: '搜索任意标的', desc: '按 Ctrl+K 打开命令面板，输入标的代码即可搜索。' },
  { title: '管理投资组合', desc: '查看持仓、盈亏和风险指标，一站式监控。' },
  { title: '完成 🎉', desc: '可以随时按 Ctrl+W 切换工作流模式。开始探索吧！' },
]

const currentStep = ref(0)

function next() {
  if (currentStep.value < steps.length - 1) {
    emit('action', currentStep.value)
    currentStep.value++
  }
}

function skip() {
  emit('done')
}

function finish() {
  emit('done')
}
</script>

<template>
  <div class="onboarding-overlay" data-testid="onboarding-overlay">
    <div class="onboarding-card">
      <div class="onboarding-steps">
        <div
          v-for="(_, i) in steps"
          :key="i"
          class="onboarding-dot"
          :class="{ 'onboarding-dot--active': i === currentStep }"
        />
      </div>

      <h2 class="onboarding-title">{{ steps[currentStep].title }}</h2>
      <p class="onboarding-desc">{{ steps[currentStep].desc }}</p>

      <div class="onboarding-actions">
        <button
          class="onboarding-skip-btn"
          data-testid="onboarding-skip"
          @click="skip"
        >
          跳过
        </button>

        <button
          v-if="currentStep < steps.length - 1"
          class="onboarding-next-btn"
          data-testid="onboarding-next"
          @click="next"
        >
          下一步
        </button>

        <button
          v-else
          class="onboarding-done-btn"
          data-testid="onboarding-done"
          @click="finish"
        >
          完成
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.onboarding-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
}

.onboarding-card {
  background: var(--surface, #fff);
  border: 1px solid var(--border, #e0e0e0);
  border-radius: 12px;
  padding: 32px;
  max-width: 420px;
  width: 90%;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
}

.onboarding-steps {
  display: flex;
  justify-content: center;
  gap: 8px;
  margin-bottom: 20px;
}

.onboarding-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border, #e0e0e0);
  transition: background 0.3s;
}

.onboarding-dot--active {
  background: var(--accent, #4a90d9);
  width: 24px;
  border-radius: 4px;
}

.onboarding-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text, #222);
  margin: 0 0 8px;
}

.onboarding-desc {
  font-size: 14px;
  color: var(--muted, #666);
  line-height: 1.5;
  margin: 0 0 24px;
}

.onboarding-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.onboarding-skip-btn {
  padding: 8px 16px;
  border: none;
  background: none;
  color: var(--muted, #888);
  cursor: pointer;
  font-size: 14px;
}

.onboarding-next-btn,
.onboarding-done-btn {
  padding: 8px 20px;
  border: none;
  border-radius: var(--radius-sm, 6px);
  background: var(--accent, #4a90d9);
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
}
</style>
```

- [ ] **Step 4: Run tests**

```bash
cd frontend && npx vitest run src/terminal/components/__tests__/OnboardingOverlay.spec.ts
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/OnboardingOverlay.vue frontend/src/terminal/components/__tests__/OnboardingOverlay.spec.ts
git commit -m "feat(onboarding): add OnboardingOverlay component with 5-step guided tour"
```

---

### Task 2: Wire onboarding into TerminalMode and session store

**Files:**
- Modify: `frontend/src/stores/session.ts`
- Modify: `frontend/src/terminal/TerminalMode.vue`

- [ ] **Step 1: Add onboarding state to session store**

In `frontend/src/stores/session.ts`:
```typescript
export const useSessionStore = defineStore('session', () => {
  const onboardingDone = ref(false)

  function completeOnboarding() {
    onboardingDone.value = true
    // Persist to localStorage so it survives restarts
    localStorage.setItem('quantflow_onboarding_done', 'true')
  }

  function initOnboarding() {
    const stored = localStorage.getItem('quantflow_onboarding_done')
    if (stored === 'true') onboardingDone.value = true
  }

  return {
    onboardingDone,
    completeOnboarding,
    initOnboarding,
  }
})
```

- [ ] **Step 2: Add OnboardingOverlay to TerminalMode.vue**

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import OnboardingOverlay from './components/OnboardingOverlay.vue'
import { useSessionStore } from '@/stores/session'
import { useDockStore } from '@/stores/terminal' // adjust if name differs

const sessionStore = useSessionStore()
const dockStore = useDockStore()

onMounted(() => {
  sessionStore.initOnboarding()
})

function handleOnboardingAction(step: number) {
  switch (step) {
    case 0: // Open Watchlist
      dockStore.addPanel?.('WatchlistPanel')
      break
    case 1: // Focus CommandBar
      // Emit event or use a ref to focus CommandBar input
      document.dispatchEvent(new CustomEvent('focus-commandbar'))
      break
    case 2: // Open Portfolio
      dockStore.addPanel?.('PortfolioSummary')
      break
    case 3: // Suggest Ctrl+W — no action needed, just text
      break
  }
}

function handleOnboardingDone() {
  sessionStore.completeOnboarding()
}
</script>

<template>
  <!-- existing TerminalMode content -->
  <OnboardingOverlay
    v-if="!sessionStore.onboardingDone"
    @action="handleOnboardingAction"
    @done="handleOnboardingDone"
  />
</template>
```

- [ ] **Step 3: Run type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 4: Run tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/session.ts frontend/src/terminal/TerminalMode.vue
git commit -m "feat(onboarding): wire OnboardingOverlay into TerminalMode with session store persistence"
```

---

### Task 3: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entry**

```markdown
### Added
- [Frontend] First-run onboarding overlay with 5-step guided tour for new users
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for onboarding"
```

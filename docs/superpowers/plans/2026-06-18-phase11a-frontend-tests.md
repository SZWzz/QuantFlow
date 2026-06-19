# Phase 11A: Frontend Test Infrastructure + Coverage — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Build frontend testing from zero: 8 store test suites + 37 component smoke tests = ≥40 test files, `vitest run` all green.

**Architecture:** vitest + jsdom + @vue/test-utils. vitest config is inline in vite.config.ts (`test.globals: true, environment: 'jsdom'`). Stores are pure Pinia — testable without Wails bridge. Panels get shallow mount with stubs.

**Tech Stack:** Vitest 2.x, Vue 3.5, Pinia 2.x, @vue/test-utils 2.4, jsdom 25.

**Depends on:** Nothing — no prior tests exist.

## Global Constraints
- vitest + @vue/test-utils already installed as devDependencies
- vite.config.ts already has `test: { globals: true, environment: 'jsdom' }`
- Each test file follows: `describe('ComponentName', () => { it('should ...', () => { ... }) })`
- Store tests use `setActivePinia(createPinia())` in beforeEach
- Panel tests use `mount()` from @vue/test-utils with `shallow: true`
- Panels that call `(window as any).go.main.App.*` must mock before mount
- Test files live alongside source: `src/stores/ml.test.ts` next to `src/stores/ml.ts`
- No snapshot tests — explicit assertions only
- Every test file must have ≥2 test cases

---

### Task 1: Store Test — data.ts

**Files:**
- Create: `frontend/src/stores/data.test.ts`

- [ ] **Step 1: Write the test**

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDataStore } from './data'

describe('useDataStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty quotes map', () => {
    const store = useDataStore()
    expect(store.quotes.size).toBe(0)
  })

  it('should update and retrieve quote', () => {
    const store = useDataStore()
    const snap = { symbol: '000001', last: 10.5, bid: 10.4, ask: 10.6, volume: 1000, change: 0.1, changePct: 1.0, timestamp: Date.now() }
    store.updateQuote('000001', snap)
    expect(store.getQuote('000001')).toEqual(snap)
  })

  it('should return undefined for missing quote', () => {
    const store = useDataStore()
    expect(store.getQuote('nope')).toBeUndefined()
  })

  it('should set and get OHLCV cache', () => {
    const store = useDataStore()
    const bars = [{ date: '2024-01-01', open: 10, high: 11, low: 9, close: 10.5, volume: 5000 }]
    store.setOHLCV('key1', bars)
    expect(store.getOHLCV('key1')).toEqual(bars)
  })

  it('should toggle offline mode', () => {
    const store = useDataStore()
    expect(store.isOffline).toBe(false)
    store.toggleOffline()
    expect(store.isOffline).toBe(true)
  })
})
```

- [ ] **Step 2: Run test**

```bash
cd frontend && npx vitest run src/stores/data.test.ts
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/data.test.ts && git commit -m "test: add data store unit tests"
```

---

### Task 2: Store Test — settings.ts

**Files:**
- Create: `frontend/src/stores/settings.test.ts`

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSettingsStore } from './settings'

describe('useSettingsStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('should initialize with defaults', () => {
    const store = useSettingsStore()
    expect(store.settings.language).toBe('zh')
    expect(store.settings.defaultBroker).toBe('paper')
    expect(store.settings.defaultQty).toBe(100)
  })

  it('should update a setting and persist to localStorage', () => {
    const store = useSettingsStore()
    store.update('defaultQty', 200)
    expect(store.settings.defaultQty).toBe(200)
    const saved = JSON.parse(localStorage.getItem('quantflow-settings')!)
    expect(saved.defaultQty).toBe(200)
  })

  it('should reset to defaults', () => {
    const store = useSettingsStore()
    store.update('language', 'en')
    store.reset()
    expect(store.settings.language).toBe('zh')
  })

  it('should load persisted settings on init', () => {
    localStorage.setItem('quantflow-settings', JSON.stringify({ language: 'en', defaultQty: 50 }))
    setActivePinia(createPinia())
    const store = useSettingsStore()
    expect(store.settings.language).toBe('en')
    expect(store.settings.defaultQty).toBe(50)
  })

  it('should handle corrupted localStorage gracefully', () => {
    localStorage.setItem('quantflow-settings', 'not-json')
    setActivePinia(createPinia())
    const store = useSettingsStore()
    expect(store.settings.language).toBe('zh')
  })
})
```

Commit:

```bash
git add frontend/src/stores/settings.test.ts && git commit -m "test: add settings store unit tests"
```

---

### Task 3: Store Test — session.ts

**Files:**
- Create: `frontend/src/stores/session.test.ts`

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from './session'

describe('useSessionStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('should default to dark/zh/terminal', () => {
    const store = useSessionStore()
    expect(store.ui.theme).toBe('dark')
    expect(store.ui.language).toBe('zh')
    expect(store.ui.mode).toBe('terminal')
  })

  it('should toggle mode between terminal and workflow', () => {
    const store = useSessionStore()
    store.toggleMode()
    expect(store.ui.mode).toBe('workflow')
    store.toggleMode()
    expect(store.ui.mode).toBe('terminal')
  })

  it('should set theme to light', () => {
    const store = useSessionStore()
    store.setTheme('light')
    expect(store.ui.theme).toBe('light')
  })

  it('should persist changes to localStorage', () => {
    const store = useSessionStore()
    store.setTheme('light')
    const saved = JSON.parse(localStorage.getItem('quantflow-session')!)
    expect(saved.theme).toBe('light')
  })

  it('should load persisted session on init', () => {
    localStorage.setItem('quantflow-session', JSON.stringify({ theme: 'light', language: 'en', mode: 'workflow' }))
    setActivePinia(createPinia())
    const store = useSessionStore()
    expect(store.ui.theme).toBe('light')
    expect(store.ui.language).toBe('en')
    expect(store.ui.mode).toBe('workflow')
  })
})
```

Commit:

```bash
git add frontend/src/stores/session.test.ts && git commit -m "test: add session store unit tests"
```

---

### Task 4: Store Test — terminal.ts

**Files:**
- Create: `frontend/src/stores/terminal.test.ts`

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTerminalStore } from './terminal'

describe('useTerminalStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty panels', () => {
    const store = useTerminalStore()
    expect(store.activePanels).toHaveLength(0)
  })

  it('should open and close panel', () => {
    const store = useTerminalStore()
    const id = store.openPanel('watchlist')
    expect(store.activePanels).toHaveLength(1)
    expect(store.activePanels[0].panelId).toBe('watchlist')
    store.closePanel(id)
    expect(store.activePanels).toHaveLength(0)
  })

  it('should pass params when opening panel', () => {
    const store = useTerminalStore()
    store.openPanel('quote-detail', { symbol: '000001' })
    expect(store.activePanels[0].params).toEqual({ symbol: '000001' })
  })

  it('should manage command history with max 20 entries', () => {
    const store = useTerminalStore()
    store.addCommand('cmd1')
    expect(store.commandHistory[0]).toBe('cmd1')
    // Add 25 commands, history should cap at 20
    for (let i = 2; i <= 25; i++) store.addCommand(`cmd${i}`)
    expect(store.commandHistory).toHaveLength(20)
  })

  it('should toggle focus mode', () => {
    const store = useTerminalStore()
    expect(store.focusMode).toBe(false)
    store.toggleFocusMode()
    expect(store.focusMode).toBe(true)
  })
})
```

Commit:

```bash
git add frontend/src/stores/terminal.test.ts && git commit -m "test: add terminal store unit tests"
```

---

### Task 5: Store Test — workflow.ts

**Files:**
- Create: `frontend/src/stores/workflow.test.ts`

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWorkflowStore } from './workflow'

describe('useWorkflowStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty nodes and edges', () => {
    const store = useWorkflowStore()
    expect(store.nodes).toHaveLength(0)
    expect(store.edges).toHaveLength(0)
    expect(store.executionStatus).toBe('idle')
  })

  it('should add a node', () => {
    const store = useWorkflowStore()
    const id = store.addNode('sma', { x: 100, y: 200 })
    expect(store.nodes).toHaveLength(1)
    expect(store.nodes[0].data.nodeType).toBe('sma')
    expect(id).toContain('sma-')
  })

  it('should remove a node and its edges', () => {
    const store = useWorkflowStore()
    const id = store.addNode('sma', { x: 100, y: 200 })
    store.edges.push({ id: 'e1', source: id, target: 'other' })
    store.removeNode(id)
    expect(store.nodes).toHaveLength(0)
    expect(store.edges).toHaveLength(0)
  })

  it('should add and remove edges', () => {
    const store = useWorkflowStore()
    store.addEdge({ id: 'e1', source: 'a', target: 'b' })
    expect(store.edges).toHaveLength(1)
    store.removeEdge('e1')
    expect(store.edges).toHaveLength(0)
  })

  it('should select a node', () => {
    const store = useWorkflowStore()
    const id = store.addNode('sma', { x: 0, y: 0 })
    store.selectNode(id)
    expect(store.selectedNodeId).toBe(id)
    store.selectNode(null)
    expect(store.selectedNodeId).toBeNull()
  })

  it('should undo/redo', () => {
    const store = useWorkflowStore()
    store.addNode('sma', { x: 0, y: 0 })
    expect(store.nodes).toHaveLength(1)
    store.undo()
    expect(store.nodes).toHaveLength(0)
    store.redo()
    expect(store.nodes).toHaveLength(1)
  })

  it('should serialize to WorkflowJSON', () => {
    const store = useWorkflowStore()
    store.addNode('sma', { x: 100, y: 200 }, { period: 20 })
    store.addNode('data_loader', { x: 0, y: 0 }, { symbol: '000001' })
    const wf = store.toWorkflowJSON('Test WF')
    expect(wf.name).toBe('Test WF')
    expect(wf.nodes).toHaveLength(2)
    expect(wf.nodes[0].node_type).toBe('sma')
  })

  it('should reset execution state', () => {
    const store = useWorkflowStore()
    store.executionStatus = 'failed'
    store.resetExecution()
    expect(store.executionStatus).toBe('idle')
    expect(store.nodeStatuses.size).toBe(0)
  })
})
```

Commit:

```bash
git add frontend/src/stores/workflow.test.ts && git commit -m "test: add workflow store unit tests"
```

---

### Task 6: Store Test — notify.ts + portfolio.ts + ml.ts

**Files:**
- Create: `frontend/src/stores/notify.test.ts`
- Create: `frontend/src/stores/portfolio.test.ts`
- Create: `frontend/src/stores/ml.test.ts`

notify.test.ts:
```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNotifyStore } from './notify'

describe('useNotifyStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty notifications', () => {
    const store = useNotifyStore()
    expect(store.notifications).toHaveLength(0)
    expect(store.unreadCount).toBe(0)
  })

  it('should set filter level', () => {
    const store = useNotifyStore()
    store.setFilter('error')
    expect(store.levelFilter).toBe('error')
  })

  it('should filter notifications by level', () => {
    const store = useNotifyStore()
    store.notifications.push(
      { id: 1, level: 'info', title: 't1', body: 'b1', metadata: '{}', is_read: false, created_at: '2024-01-01' },
      { id: 2, level: 'error', title: 't2', body: 'b2', metadata: '{}', is_read: false, created_at: '2024-01-02' },
    )
    store.setFilter('error')
    expect(store.filteredNotifications).toHaveLength(1)
    expect(store.filteredNotifications[0].level).toBe('error')
  })
})
```

portfolio.test.ts:
```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePortfolioStore } from './portfolio'

describe('usePortfolioStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with null summary and empty positions', () => {
    const store = usePortfolioStore()
    expect(store.summary).toBeNull()
    expect(store.positions).toHaveLength(0)
  })

  it('should start and stop auto refresh', () => {
    const store = usePortfolioStore()
    store.startAutoRefresh()
    expect((store as any).timer).not.toBeNull()
    store.stopAutoRefresh()
    expect((store as any).timer).toBeNull()
  })
})
```

ml.test.ts:
```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMLStore } from './ml'

describe('useMLStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should start with empty models', () => {
    const store = useMLStore()
    expect(store.models).toHaveLength(0)
  })

  it('should compute readyModels from models', () => {
    const store = useMLStore()
    store.models.push(
      { id: '1', name: 'm1', model_type: 'xgboost', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'ready', created_at: '', updated_at: '' },
      { id: '2', name: 'm2', model_type: 'lstm', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'training', created_at: '', updated_at: '' },
    )
    expect(store.readyModels).toHaveLength(1)
    expect(store.readyModels[0].id).toBe('1')
  })

  it('should manage RL training state', () => {
    const store = useMLStore()
    expect(store.rlTrainingRunning).toBe(false)
    store.startRLTraining('ppo')
    expect(store.rlTrainingRunning).toBe(true)
    expect(store.rlAlgorithm).toBe('ppo')
    expect(store.rlTrainingEpisodes).toHaveLength(0)
    store.addRLUpdate({ episode: 1, reward: 0.05, sharpe: 1.2, steps: 100, epsilon: 0.3 })
    expect(store.rlTrainingEpisodes).toHaveLength(1)
    store.stopRLTraining()
    expect(store.rlTrainingRunning).toBe(false)
  })

  it('should manage risk model result', () => {
    const store = useMLStore()
    store.setRiskModelResult({ model_type: 'garch', volatility: [0.01, 0.02], aic: -500, bic: -490 })
    expect(store.riskModelResult?.model_type).toBe('garch')
    store.setRiskModelResult(null)
    expect(store.riskModelResult).toBeNull()
  })

  it('should select and deselect model', () => {
    const store = useMLStore()
    const model = { id: '1', name: 'm1', model_type: 'xgboost', category: 'prediction', hyperparams: {}, metrics: {}, file_path: '', status: 'ready' as const, created_at: '', updated_at: '' }
    store.selectModel(model)
    expect(store.selectedModel).toEqual(model)
    store.selectModel(null)
    expect(store.selectedModel).toBeNull()
  })
})
```

Commit:

```bash
git add frontend/src/stores/notify.test.ts frontend/src/stores/portfolio.test.ts frontend/src/stores/ml.test.ts && git commit -m "test: add notify, portfolio, ml store unit tests"
```

---

### Task 7: Panel Smoke Tests — Terminal Panels (Part 1)

**Files:**
- Create: `frontend/src/terminal/panels/__tests__/WatchlistPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/QuoteDetailPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/CandlestickPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/OrderEntryPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/PositionPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/NewsPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/AIChatPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/SystemMonitorPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/BacktestResultPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/FactorAnalysisPanel.test.ts`

Pattern for each smoke test:
```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import WatchlistPanel from '../WatchlistPanel.vue'

// Mock Wails bridge
;(window as any).go = { main: { App: {} } }

describe('WatchlistPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(WatchlistPanel, {
      global: { stubs: { echarts: true, v-chart: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})
```

Apply the same pattern to all 10 panels, adjusting the component name and import path. Use appropriate stubs (echarts, v-chart, Transition, etc.).

Commit:

```bash
git add frontend/src/terminal/panels/__tests__/ && git commit -m "test: add smoke tests for 10 terminal panels"
```

---

### Task 8: Panel Smoke Tests — Terminal Panels (Part 2) + Workflow Components

**Files:**
- Create: `frontend/src/terminal/panels/__tests__/PortfolioSummary.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/PositionDetail.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/RiskDashboard.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/TradeHistory.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/SchedulePanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/NotifyPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/BrokerConfig.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/SettingsPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/ModelRegistryPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/PredictionDashboardPanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/AlphaMiningWorkspacePanel.test.ts`
- Create: `frontend/src/terminal/panels/__tests__/RLMonitorPanel.test.ts`

Same smoke test pattern as Task 7.

Commit:

```bash
git add frontend/src/terminal/panels/__tests__/ && git commit -m "test: add smoke tests for remaining 12 terminal panels"
```

---

### Task 9: Panel Smoke Tests — Workflow Components + Terminal Shell

**Files:**
- Create: `frontend/src/workflow/__tests__/NodePalette.test.ts`
- Create: `frontend/src/workflow/__tests__/PropertyPanel.test.ts`
- Create: `frontend/src/workflow/__tests__/ExecutionLog.test.ts`
- Create: `frontend/src/workflow/canvas/__tests__/CustomNode.test.ts`
- Create: `frontend/src/workflow/canvas/__tests__/WorkflowCanvas.test.ts`
- Create: `frontend/src/terminal/__tests__/CommandBar.test.ts`
- Create: `frontend/src/terminal/__tests__/StatusBar.test.ts`
- Create: `frontend/src/terminal/__tests__/PushPinBar.test.ts`

WorkflowCanvas needs extra stubs for vue-flow:
```typescript
import { mount } from '@vue/test-utils'
import WorkflowCanvas from '../WorkflowCanvas.vue'

describe('WorkflowCanvas', () => {
  it('should mount', () => {
    const wrapper = mount(WorkflowCanvas, {
      global: {
        stubs: { VueFlow: true, MiniMap: true, Controls: true, Background: true },
      },
    })
    expect(wrapper.exists()).toBe(true)
  })
})
```

Commit:

```bash
git add frontend/src/workflow/__tests__/ frontend/src/workflow/canvas/__tests__/ frontend/src/terminal/__tests__/ && git commit -m "test: add smoke tests for workflow and terminal shell components"
```

---

### Task 10: Panel Smoke Tests — DockView Components

**Files:**
- Create: `frontend/src/terminal/DockView/__tests__/DockView.test.ts`
- Create: `frontend/src/terminal/DockView/__tests__/DockContainer.test.ts`
- Create: `frontend/src/terminal/DockView/__tests__/DockSplitter.test.ts`
- Create: `frontend/src/terminal/DockView/__tests__/DockTab.test.ts`

Same smoke test pattern with required props:
```typescript
import { mount } from '@vue/test-utils'
import DockView from '../DockView.vue'
import { createTabLeaf } from '../types'

describe('DockView', () => {
  it('should mount', () => {
    const layout = createTabLeaf('root', { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' })
    const wrapper = mount(DockView, {
      props: { layout },
      global: { stubs: { DockContainer: true, DockTab: true, DockSplitter: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })
})
```

Commit:

```bash
git add frontend/src/terminal/DockView/__tests__/ && git commit -m "test: add smoke tests for DockView components"
```

---

### Task 11: Final — run full vitest, fix failures

Run: `cd frontend && npx vitest run`
Expected: All tests pass. Fix any remaining import or stub issues.
Commit any fixes.

---

### Task 12: CHANGELOG + README update

Add Phase 11A entries to CHANGELOG:
```
### Added
- [Frontend] 8 Pinia store test suites (data, settings, session, terminal, workflow, notify, portfolio, ml)
- [Frontend] 37 component smoke tests (22 panels + 8 workflow + 4 dockview + 3 terminal shell)
```

Update README test counts.

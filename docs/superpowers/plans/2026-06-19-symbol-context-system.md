# Symbol Context 联动系统 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 Bloomberg 式 Link Group 联动系统——4 个颜色编码的 symbol 上下文组，面板绑定到 Group 后自动跟随 symbol 变化。

**Architecture:** `symbolContextStore` (Pinia) 管理 4 个 LinkGroup → 面板通过 `useSymbolContext()` 读/写 → Publisher 面板调用 `setGroupSymbol()` → Subscriber 面板 `watch(group.activeSymbol)` 自动更新。

**Tech Stack:** Vue 3 + Pinia + TypeScript，纯前端改动，Go 后端零修改。

## Global Constraints

- Vue 3 Composition API（`<script setup lang="ts">`）
- Pinia store：`defineStore` + `ref`/`reactive`
- 所有面板遵循现有 prop 接口（`panelId: string; params?: Record<string, any>`）
- CSS 使用 `var(--color-*)` 主题变量
- 向后兼容：`props.params?.symbol` 仍可用于初始化
- 现有 185 前端测试全通过

---

## Phase 1: Core Store + Infra (Tasks 1-3)

### Task 1: symbolContextStore

**Files:**
- Create: `frontend/src/stores/symbolContext.ts`

**Interfaces:**
- Produces: `useSymbolContext` store — `linkGroups`, `activeGroupId`, `setGroupSymbol()`, `getGroupSymbol()`, `getOrCreatePanelGroup()`, `setPanelLinked()`

```typescript
// frontend/src/stores/symbolContext.ts
import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

export interface LinkGroup {
  id: string
  color: string
  label: string
  activeSymbol: string | null
  symbolHistory: string[]
}

export const useSymbolContext = defineStore('symbolContext', () => {
  const linkGroups = reactive<Record<string, LinkGroup>>({
    'group-1': { id: 'group-1', color: '#ef4444', label: 'Red', activeSymbol: null, symbolHistory: [] },
    'group-2': { id: 'group-2', color: '#22c55e', label: 'Green', activeSymbol: null, symbolHistory: [] },
    'group-3': { id: 'group-3', color: '#f59e0b', label: 'Amber', activeSymbol: null, symbolHistory: [] },
    'group-4': { id: 'group-4', color: '#3b82f6', label: 'Blue', activeSymbol: null, symbolHistory: [] },
  })

  const activeGroupId = ref('group-1')

  // Panel → Group binding
  const panelGroups = reactive<Record<string, { groupId: string; linked: boolean }>>({})

  function setGroupSymbol(groupId: string, symbol: string) {
    const group = linkGroups[groupId]
    if (!group || !symbol) return
    const s = symbol.trim().toUpperCase()
    if (s !== group.activeSymbol) {
      group.activeSymbol = s
      group.symbolHistory = [s, ...group.symbolHistory.filter(h => h !== s)].slice(0, 10)
    }
  }

  function getGroupSymbol(groupId: string): string | null {
    return linkGroups[groupId]?.activeSymbol ?? null
  }

  function setActiveGroup(groupId: string) {
    if (linkGroups[groupId]) activeGroupId.value = groupId
  }

  function getOrCreatePanelGroup(panelId: string): { groupId: string; linked: boolean } {
    if (!panelGroups[panelId]) {
      panelGroups[panelId] = { groupId: activeGroupId.value, linked: true }
    }
    return panelGroups[panelId]
  }

  function setPanelGroup(panelId: string, groupId: string) {
    panelGroups[panelId] = { groupId, linked: true }
  }

  function setPanelLinked(panelId: string, linked: boolean) {
    if (panelGroups[panelId]) {
      panelGroups[panelId].linked = linked
    }
  }

  function getPanelGroupId(panelId: string): string {
    return panelGroups[panelId]?.groupId || 'group-1'
  }

  function getActiveSymbolForPanel(panelId: string): string | null {
    const pg = panelGroups[panelId]
    if (!pg || !pg.linked) return null
    return linkGroups[pg.groupId]?.activeSymbol ?? null
  }

  // Legacy compatibility: migrate from old terminalStore.activeSymbol
  function initFromLegacy(legacySymbol: string | null) {
    if (legacySymbol && !linkGroups['group-1'].activeSymbol) {
      linkGroups['group-1'].activeSymbol = legacySymbol
    }
  }

  return {
    linkGroups, activeGroupId, panelGroups,
    setGroupSymbol, getGroupSymbol, setActiveGroup,
    getOrCreatePanelGroup, setPanelGroup, setPanelLinked,
    getPanelGroupId, getActiveSymbolForPanel, initFromLegacy,
  }
})
```

- [ ] **Step 1:** Create the file with the full code above
- [ ] **Step 2:** Verify compilation: `cd frontend && npx vue-tsc --noEmit`
- [ ] **Step 3:** Commit: `git add frontend/src/stores/symbolContext.ts && git commit -m "feat: add symbolContext store with 4 link groups and panel binding"`

---

### Task 2: Clean up terminalStore + wire symbolContext

**Files:**
- Modify: `frontend/src/stores/terminal.ts` — remove old `activeSymbol`, `lastSymbolUpdate`, `setActiveSymbol`

**Changes in terminal.ts:**
- Remove `activeSymbol`, `lastSymbolUpdate`, `setActiveSymbol` from state/return
- Keep everything else unchanged

- [ ] **Step 1:** Remove the 3 lines from terminal.ts
- [ ] **Step 2:** Verify no remaining references: `cd frontend && grep -r "terminal\.activeSymbol\|terminal\.setActiveSymbol\|terminal\.lastSymbolUpdate" src/ --include="*.vue" --include="*.ts"` should return 0
- [ ] **Step 3:** If there are remaining references in panels, update them to use symbolContext instead (see Tasks 4-5)
- [ ] **Step 4:** Commit

---

### Task 3: StatusBar + SymbolBar + DockTab indicators

**Files:**
- Modify: `frontend/src/terminal/StatusBar.vue` — show all active group symbols
- Create: `frontend/src/terminal/SymbolBar.vue` — quick symbol input
- Modify: `frontend/src/terminal/DockView/DockTab.vue` — group color indicator on tabs

**StatusBar.vue** changes:
```vue
<script setup lang="ts">
// Add import
import { useSymbolContext } from '@/stores/symbolContext'

const ctx = useSymbolContext()

// Display active groups
const activeGroups = computed(() =>
  Object.values(ctx.linkGroups).filter(g => g.activeSymbol)
)
</script>

<!-- Add to template: between status-left and status-center -->
<div class="status-groups">
  <span v-for="g in activeGroups" :key="g.id" class="group-badge"
    :style="{ borderColor: g.color }">
    <span class="group-dot" :style="{ background: g.color }"></span>
    {{ g.activeSymbol }}
  </span>
</div>

<style scoped>
.status-groups { display: flex; gap: 8px; }
.group-badge {
  display: flex; align-items: center; gap: 4px;
  padding: 0 6px; border: 1px solid; border-radius: var(--radius-sm);
  font-size: var(--font-xs); font-weight: 600;
}
.group-dot { width: 6px; height: 6px; border-radius: 50%; }
</style>
```

**SymbolBar.vue** — quick symbol input bar:
```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useSymbolContext } from '@/stores/symbolContext'

const ctx = useSymbolContext()
const inputVal = ref('')

function submit() {
  if (inputVal.value.trim()) {
    ctx.setGroupSymbol(ctx.activeGroupId, inputVal.value.trim())
    inputVal.value = ''
  }
}

const groups = Object.entries(ctx.linkGroups)
</script>

<template>
  <div class="symbol-bar">
    <div class="group-tabs">
      <button v-for="[id, g] in groups" :key="id"
        :class="{ active: ctx.activeGroupId === id }"
        :style="{ '--gcolor': g.color }"
        @click="ctx.setActiveGroup(id)">
        <span class="dot" :style="{ background: g.color }"></span>
        {{ g.activeSymbol || '--' }}
      </button>
    </div>
    <form class="symbol-input-area" @submit.prevent="submit">
      <input v-model="inputVal" placeholder="Enter symbol..." />
    </form>
  </div>
</template>

<style scoped>
.symbol-bar {
  display: flex; align-items: center; padding: 4px 10px;
  background: var(--color-bg-subtle); border-bottom: 1px solid var(--color-border);
  min-height: 30px; gap: 8px;
}
.group-tabs { display: flex; gap: 4px; }
.group-tabs button {
  display: flex; align-items: center; gap: 4px;
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-secondary);
  border-radius: var(--radius-sm); font-size: var(--font-xs); cursor: pointer;
}
.group-tabs button.active { border-color: var(--gcolor); color: var(--color-text-primary); }
.dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.symbol-input-area { flex: 1; }
.symbol-input-area input {
  width: 100%; padding: 2px 8px; background: var(--color-bg-input);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  color: var(--color-text-primary); font-size: var(--font-xs); outline: none;
}
.symbol-input-area input:focus { border-color: var(--color-accent); }
</style>
```

**DockTab.vue** — add group indicator to tab buttons:
- After `tab-icon`, add a color dot if the panel belongs to a group with active symbol
- Import `useSymbolContext`, call `getPanelGroupId(tabId)`, show dot with group color

- [ ] **Step 1:** Update StatusBar.vue
- [ ] **Step 2:** Create SymbolBar.vue
- [ ] **Step 3:** Update DockTab.vue tab buttons with group dot
- [ ] **Step 4:** Add SymbolBar to TerminalMode.vue template (before PushPinBar)
- [ ] **Step 5:** Verify: `cd frontend && npx vitest run` — all pass
- [ ] **Step 6:** Commit

---

## Phase 2: Publisher Migration (Tasks 4-5)

### Task 4: Update WatchlistPanel as Publisher

**Files:**
- Modify: `frontend/src/terminal/panels/WatchlistPanel.vue`

**Changes:**
- Import `useSymbolContext` instead of `useTerminalStore` for symbol context
- `selectSymbol(sym)` → `ctx.setGroupSymbol(ctx.getPanelGroupId(props.panelId), sym)`
- Call `ctx.getOrCreatePanelGroup(props.panelId)` in setup to bind this panel to a group
- active row highlight = symbol matches group's activeSymbol

- [ ] **Step 1:** Make the changes
- [ ] **Step 2:** Verify: `cd frontend && npx vitest run` — all pass
- [ ] **Step 3:** Commit: `git commit -m "feat: migrate WatchlistPanel to symbolContext publisher"`

### Task 5: Update other Publishers (QuoteDetail, StockResearch, OrderEntry, Financials)

**Files:**
- Modify: `QuoteDetailPanel.vue`, `StockResearchPanel.vue`, `OrderEntryPanel.vue`, `FinancialsPanel.vue`

**Pattern for each:**
```typescript
import { useSymbolContext } from '@/stores/symbolContext'
const ctx = useSymbolContext()
const panelCtx = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(panelCtx.groupId) || 'AAPL')

// Watch group symbol
watch(() => ctx.linkGroups[panelCtx.groupId].activeSymbol, (newSym) => {
  if (panelCtx.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    // reload data...
  }
})

// When user changes symbol via input, publish to group
function onSymbolSubmit(newSym: string) {
  ctx.setGroupSymbol(panelCtx.groupId, newSym)
  symbol.value = newSym.toUpperCase()
}
```

- [ ] **Step 1:** Update all 4 panels with the pattern above
- [ ] **Step 2:** Verify: `cd frontend && npx vitest run` — all pass
- [ ] **Step 3:** Commit: `git commit -m "feat: migrate 4 publisher panels to symbolContext"`

---

## Phase 3: Subscriber Migration (Task 6)

### Task 6: Update 10 Subscriber Panels

**Files:**
- Modify: `CandlestickPanel.vue`, `SentimentPanel.vue`, `PeerComparisonPanel.vue`, `AnalystEstimatesPanel.vue`, `InsiderTradingPanel.vue`, `MarketDepthPanel.vue`, `PositionDetail.vue`, `DrawingPanel.vue`, `DistributionPanel.vue`, `SurfaceChartPanel.vue`

**Pattern for each subscriber:**
```typescript
import { useSymbolContext } from '@/stores/symbolContext'
const ctx = useSymbolContext()
const panelCtx = ctx.getOrCreatePanelGroup(props.panelId)
const symbol = ref(props.params?.symbol || ctx.getGroupSymbol(panelCtx.groupId) || 'AAPL')

// Subscribe to group symbol changes
watch(() => ctx.linkGroups[panelCtx.groupId]?.activeSymbol, (newSym) => {
  if (panelCtx.linked && newSym && newSym !== symbol.value) {
    symbol.value = newSym
    // reload/regenerate data specific to this panel
  }
})
```

Each panel has its own data reload logic:
- Candlestick: `ohlcvData.value = generateMockData(90)`
- Sentiment: fetch sentiment
- PeerComparison: fetch peers
- AnalystEstimates: fetch estimates
- InsiderTrading: fetch insider trades
- MarketDepth: regenerate depth data
- PositionDetail: filter position
- Drawing: load from localStorage
- Distribution: recompute distribution
- SurfaceChart: regenerate surface

- [ ] **Step 1:** Update all 10 panels
- [ ] **Step 2:** Verify: `cd frontend && npx vitest run` — all pass
- [ ] **Step 3:** Commit: `git commit -m "feat: migrate 10 subscriber panels to symbolContext"`

---

## Phase 4: Final Verification (Task 7)

### Task 7: CHANGELOG + full verification

- [ ] **Step 1:** Update CHANGELOG.md — 新增条目

```markdown
- [Terminal] Symbol Context 联动系统：4 个 Link Group（红/绿/黄/蓝），Bloomberg 式跨面板联动
- [Terminal] SymbolBar：快速 Symbol 输入 + Group 切换
- [前端] 5 个 Publisher 面板 + 10 个 Subscriber 面板迁移到 symbolContext
```

- [ ] **Step 2:** Full verification

```bash
cd e:/coding/quantflow/frontend && npx vue-tsc --noEmit && npx vitest run
cd e:/coding/quantflow && go vet ./...
```

Expected: 185+ tests PASS, 15 pre-existing TS errors, Go vet clean (1 pre-existing)

- [ ] **Step 3:** Commit: `git commit -m "docs: update CHANGELOG for Symbol Context system"`

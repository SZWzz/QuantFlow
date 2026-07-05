# Plan: Terminal → Workflow Linkage

> **Spec**: `docs/specs/2026-07-05-terminal-to-workflow-linkage.md`

## Task Breakdown

### Task 1: Create `panelToNode.ts` mapping

**File**: `frontend/src/terminal/panelToNode.ts` (CREATE)

Complete mapping from panelId → nodeType(s). For panels that map to multiple node types, the first entry is the default.

```typescript
export interface PanelToNodeEntry {
  nodeType: string      // default node type to spawn
  label: string         // label shown in tooltip
  multi?: string[]      // alternative node types (for future submenu)
}

// panelId → PanelToNodeEntry
export const PANEL_TO_NODE: Record<string, PanelToNodeEntry> = {
  'candlestick':    { nodeType: 'data_loader',   label: 'Data Loader' },
  'watchlist':      { nodeType: 'loop',          label: 'Loop' },
  'indicator':      { nodeType: 'sma',           label: 'SMA',
                       multi: ['macd', 'rsi', 'bollinger', 'ema'] },
  'stock-scanner':  { nodeType: 'rank_select',   label: 'Rank Select' },
  'factor-analysis':{ nodeType: 'factor',        label: 'Factor' },
  'backtest':       { nodeType: 'backtest',      label: 'Backtest' },
  'sentiment':      { nodeType: 'sentiment',     label: 'Sentiment' },
  'news':           { nodeType: 'news_fetcher',  label: 'News Fetcher' },
  'correlation':    { nodeType: 'math_op',       label: 'Math Op' },
  'distribution':   { nodeType: 'math_op',       label: 'Math Op' },
  'financials':     { nodeType: 'financials',    label: 'Financials' },
  'peer-comparison':{ nodeType: 'peer_compare',  label: 'Peer Compare' },
  'analyst-estimates':{ nodeType: 'analyst_estimates', label: 'Analyst Estimates' },
  'insider-trading':{ nodeType: 'insider_trades',label: 'Insider Trades' },
  'prediction-market':{ nodeType: 'prediction_market', label: 'Prediction Market' },
  'geopolitics':    { nodeType: 'geopolitics',   label: 'Geopolitics' },
  'satellite':      { nodeType: 'satellite',     label: 'Satellite' },
  'macro':          { nodeType: 'gov_data',      label: 'Gov Data' },
  'risk-dashboard': { nodeType: 'risk_metrics',  label: 'Risk Metrics' },
  'fundflow':       { nodeType: 'data_loader',   label: 'Data Loader' },
  'market-overview':{ nodeType: 'data_loader',   label: 'Data Loader' },
}

export function getPanelToNode(panelId: string): PanelToNodeEntry | undefined {
  return PANEL_TO_NODE[panelId]
}
```

### Task 2: Create `useAddToWorkflow` composable

**File**: `frontend/src/terminal/composables/useAddToWorkflow.ts` (CREATE)

Returns a `Control` object for PanelHeader, or null if the panel has no workflow mapping.

```typescript
import { type Control } from '@/terminal/components/panel/PanelHeader.vue'
import { getPanelToNode } from '@/terminal/panelToNode'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { useDataStore } from '@/stores/data'
import { useI18n } from '@/lib/i18n'
import { confirmDialog } from '@/lib/wails'

export function useAddToWorkflow(panelId: string, symbol?: string): Control | null {
  const entry = getPanelToNode(panelId)
  if (!entry) return null

  const workflow = useWorkflowStore()
  const session = useSessionStore()
  const data = useDataStore()
  const { t } = useI18n()

  return {
    icon: 'plus',
    label: t('workflow.add_to_workflow'),
    title: `${t('workflow.add_to_workflow')}: ${entry.label}`,
    action: async () => {
      // Use current symbol from dataStore or the provided symbol
      const sym = symbol || data.currentSymbol || '600519'
      // Add node at canvas center (viewport-aware positioning handled by store)
      workflow.addNode(entry.nodeType, { x: 200, y: 200 }, { symbol: sym })
      // Ask user if they want to switch to workflow mode
      const ok = await confirmDialog(
        t('workflow.switch_confirm_title'),
        t('workflow.switch_confirm_body')
      )
      if (ok) {
        session.toggleMode()
      }
    },
  }
}
```

### Task 3: Add i18n keys

**File**: `frontend/src/lib/i18n/en.ts`

Add under `workflow` section (the existing `ml.add_to_workflow` key is unused; add proper keys):

```typescript
workflow: {
  // ... existing keys ...
  add_to_workflow: '+ Workflow',
  switch_confirm_title: 'Switch to Workflow Mode?',
  switch_confirm_body: 'The node has been added to the workflow canvas. Switch to workflow mode to connect and configure it?',
}
```

**File**: `frontend/src/lib/i18n/zh.ts`

```typescript
workflow: {
  // ... existing keys ...
  add_to_workflow: '+ 工作流',
  switch_confirm_title: '切换到工作流模式？',
  switch_confirm_body: '节点已添加到工作流画布。是否切换到工作流模式进行连接和配置？',
}
```

Remove the unused `ml.add_to_workflow` key from both files.

### Task 4: Add `centerViewport` to session store

**File**: `frontend/src/stores/session.ts`

Add a function that computes the center of the workflow canvas viewport. This will be used in a follow-up enhancement; for now `addNode` uses a fixed position (200, 200) which is safe for the default viewport.

Actually – no change needed; the store already handles mode toggling. Skip this task for now; the fixed position (200, 200) is acceptable in Phase 1.

### Task 5: Wire [⊕] into top 5 panels

Wire `useAddToWorkflow` into the 5 most-used panels to validate the pattern before scaling to all 20.

**Files to modify** (add the control to each panel's PanelHeader):

1. **`CandlestickPanel.vue`** — controls prepend:
```typescript
const addToWorkflow = useAddToWorkflow(props.panelId)
const controls = computed(() => [
  ...(addToWorkflow.value ? [addToWorkflow.value] : []),
  { icon: 'refresh', action: refresh },
])
```

2. **`WatchlistPanel.vue`** — same pattern
3. **`IndicatorPanel.vue`** — same pattern
4. **`FactorAnalysisPanel.vue`** — same pattern
5. **`BacktestPanel.vue`** — same pattern

### Task 6: Wire [⊕] into remaining 15 priority panels

Apply the same pattern to panels 6–20 from the priority list in the spec.

### Task 7: Verify build + lint

```bash
cd /Volumes/etx/coding/quantflow
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

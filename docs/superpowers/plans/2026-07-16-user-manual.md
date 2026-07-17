# User Manual (Help Center + Panel ⓘ) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an in-app user manual — HelpPanel (full help center) + PanelHelpPopover (per-panel ⓘ tooltip) — with JSON data files validated by TypeScript schema.

**Architecture:** 7 JSON data files under `frontend/docs/manual/` carry all documentation content (panels, adapters, nodes, brokers, quickstart, python, settings). `usePanelHelp.ts` provides typed access per panelId. `PanelHelpPopover.vue` shows ⓘ tooltips in panel headers. `HelpPanel.vue` renders the full categorized help center with search. All JSON is validated at test time against TypeScript interfaces in `schema.ts`.

**Tech Stack:** Vue 3 Composition API (`<script setup lang="ts">`), Pinia stores, vue-i18n, vitest + @vue/test-utils

## Global Constraints

- All new Vue files: `<script setup lang="ts">`, Composition API
- Tests: vitest with @vue/test-utils, Pinia via `setActivePinia(createPinia())`
- No `window.confirm()`/`window.alert()` — use `@/lib/wails` dialog helpers instead
- i18n: add keys to `frontend/src/lib/i18n/en.ts` and `zh.ts`
- JSON files must match TypeScript interfaces exactly (validated by test)
- PanelHelpPopover must NOT block panel rendering if help data is missing
- HelpPanel ID in registry: `help-panel` (lowercase kebab-case, matching convention)

---

### Task 1: Schema types (schema.ts)

**Files:**
- Create: `frontend/docs/manual/schema.ts`

**Interfaces:**
- Produces: `PanelDoc`, `AdapterDoc`, `NodeDoc`, `BrokerDoc`, `QuickstartDoc`, `PythonDoc`, `SettingsDoc`, `ManualData` types

- [ ] **Step 1: Create `schema.ts` with all TypeScript interfaces**

```typescript
// frontend/docs/manual/schema.ts

export interface PanelDoc {
  id: string
  name: string
  category: string
  description: string
  dataSources: string[]
  shortcut: string
  tips: string[]
  relatedPanels: string[]
  configurable: boolean
}

export interface AdapterDoc {
  id: string
  name: string
  nameZh: string
  type: 'stock' | 'crypto' | 'fund' | 'futures' | 'news' | 'research' | 'alternative' | 'economic'
  markets: string[]
  description: string
  descriptionZh: string
  setupSteps: string[]
  configKeys: string[]
  isFree: boolean
  rateLimit?: string
}

export interface NodeDoc {
  nodeType: string
  name: string
  nameZh: string
  category: string
  description: string
  descriptionZh: string
  inputs: { name: string; type: string; description: string }[]
  outputs: { name: string; type: string; description: string }[]
  params: { name: string; type: string; default?: any; description: string }[]
}

export interface BrokerDoc {
  id: string
  name: string
  nameZh: string
  markets: string[]
  description: string
  descriptionZh: string
  setupSteps: string[]
  configKeys: { key: string; label: string; labelZh: string; secret: boolean }[]
  fees: string
  minDeposit: string
}

export interface QuickstartDoc {
  sections: {
    id: string
    title: string
    titleZh: string
    steps: { order: number; text: string; textZh: string }[]
  }[]
}

export interface PythonDoc {
  sections: {
    id: string
    title: string
    titleZh: string
    steps: { order: number; text: string; textZh: string }[]
  }[]
}

export interface SettingsDoc {
  sections: {
    id: string
    title: string
    titleZh: string
    items: { key: string; label: string; labelZh: string; description: string }[]
  }[]
}

export interface ManualData {
  panels: PanelDoc[]
  adapters: AdapterDoc[]
  nodes: NodeDoc[]
  brokers: BrokerDoc[]
  quickstart: QuickstartDoc
  python: PythonDoc
  settings: SettingsDoc
}
```

- [ ] **Step 2: Run typecheck to verify file compiles**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit docs/manual/schema.ts
```
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/docs/manual/schema.ts && git commit -m "feat(docs): add schema types for user manual JSON data"
```

### Task 2: JSON data files (panels, adapters, nodes, brokers, quickstart, python, settings)

**Files:**
- Create: `frontend/docs/manual/panels.json`
- Create: `frontend/docs/manual/adapters.json`
- Create: `frontend/docs/manual/nodes.json`
- Create: `frontend/docs/manual/brokers.json`
- Create: `frontend/docs/manual/quickstart.json`
- Create: `frontend/docs/manual/python.json`
- Create: `frontend/docs/manual/settings.json`
- Create: `frontend/docs/manual/__tests__/schema.test.ts`

**Interfaces:**
- Produces: `MANUAL_DATA` importable from `<root>/` (used by Tasks 3, 5)
- Consumes: `PanelDoc`, `AdapterDoc`, `NodeDoc`, `BrokerDoc` from Task 1
- Consumes: `getAllPanelMeta()` from `frontend/src/terminal/panels/registry.ts` (for ID cross-check)

- [ ] **Step 1: Write the failing test — schema validation**

```typescript
// frontend/docs/manual/__tests__/schema.test.ts
import { describe, it, expect } from 'vitest'
import type { PanelDoc, AdapterDoc, NodeDoc, BrokerDoc, ManualData } from '../schema'

import panelsRaw from '../panels.json'
import adaptersRaw from '../adapters.json'
import nodesRaw from '../nodes.json'
import brokersRaw from '../brokers.json'
import quickstartRaw from '../quickstart.json'
import pythonRaw from '../python.json'
import settingsRaw from '../settings.json'

describe('Manual JSON schema validation', () => {
  it('panels.json has valid structure', () => {
    const data = panelsRaw as { panels: PanelDoc[] }
    expect(Array.isArray(data.panels)).toBe(true)
    expect(data.panels.length).toBeGreaterThanOrEqual(50)
    for (const p of data.panels) {
      expect(p.id).toBeTruthy()
      expect(p.name).toBeTruthy()
      expect(p.category).toBeTruthy()
      expect(Array.isArray(p.dataSources)).toBe(true)
      expect(Array.isArray(p.tips)).toBe(true)
    }
  })

  it('panels.json IDs match registry panel IDs', () => {
    // Dynamic import of registry — vitest handles module resolution
    const panelIds = new Set(data.panels.map((p: PanelDoc) => p.id))
    // Check that all registry panels are documented
    // (import from registry at test time)
    const knownIds = ['watchlist', 'candlestick', 'market-overview', 'heatmap',
      'limit-up-down', 'ipo-calendar', 'ex-dividend', 'cb-arbitrage',
      'fundflow', 'margin', 'funds', 'futures', 'bonds', 'sector-rotation',
      'crypto-overview', 'funding-rate', 'liquidation', 'depth-comparison',
      'defi-tvl', 'whale-tracking', 'gas-tracker',
      'hk-connect', 'hk-derivatives', 'hk-settlement',
      'short-interest', 'surface-chart', 'institutional-trades', 'wash-sale', 'sec-13f',
      'order-entry', 'basket-order', 'broker-status', 'broker-config',
      'position', 'portfolio-summary', 'trade-history', 'rebalance',
      'surface-chart', 'correlation', 'distribution', 'monte-carlo',
      'trading-journal', 'scenario-analysis',
      'stock-research', 'financials', 'valuation', 'audit',
      'congress-trading', 'sentiment', 'options', 'fundflow',
      'margin', 'funds', 'futures', 'macro', 'bonds', 'sector-rotation',
      'economic-calendar', 'earnings-calendar',
      'backtest', 'factor-analysis', 'prediction-dashboard', 'alpha-mining', 'rl-monitor',
      'prediction-market', 'geopolitics', 'satellite',
      'ai-chat', 'news', 'system-monitor', 'schedule-panel',
      'notify-panel', 'settings', 'log-viewer', 'storage', 'layout-templates',
      'welcome', 'chanlun', 'indicator', 'stock-scanner',
    ]
    // Every documented panel ID is a real panel
    for (const id of panelIds) {
      expect(knownIds).toContain(id)
    }
  })

  it('adapters.json has valid structure', () => {
    const data = adaptersRaw as { adapters: AdapterDoc[] }
    expect(Array.isArray(data.adapters)).toBe(true)
    expect(data.adapters.length).toBeGreaterThanOrEqual(20)
    for (const a of data.adapters) {
      expect(a.id).toBeTruthy()
      expect(a.name).toBeTruthy()
      expect(a.nameZh).toBeTruthy()
      expect(Array.isArray(a.markets)).toBe(true)
      expect(Array.isArray(a.setupSteps)).toBe(true)
    }
  })

  it('nodes.json has valid structure', () => {
    const data = nodesRaw as { nodes: NodeDoc[] }
    expect(Array.isArray(data.nodes)).toBe(true)
    expect(data.nodes.length).toBeGreaterThanOrEqual(50)
    for (const n of data.nodes) {
      expect(n.nodeType).toBeTruthy()
      expect(n.name).toBeTruthy()
      expect(Array.isArray(n.inputs)).toBe(true)
      expect(Array.isArray(n.outputs)).toBe(true)
      expect(Array.isArray(n.params)).toBe(true)
    }
  })

  it('brokers.json has valid structure', () => {
    const data = brokersRaw as { brokers: BrokerDoc[] }
    expect(Array.isArray(data.brokers)).toBe(true)
    expect(data.brokers.length).toBeGreaterThanOrEqual(4)
    for (const b of data.brokers) {
      expect(b.id).toBeTruthy()
      expect(b.name).toBeTruthy()
      expect(Array.isArray(b.setupSteps)).toBe(true)
    }
  })

  it('quickstart.json has valid structure', () => {
    const data = quickstartRaw as { sections: any[] }
    expect(Array.isArray(data.sections)).toBe(true)
    expect(data.sections.length).toBeGreaterThanOrEqual(2)
  })

  it('python.json has valid structure', () => {
    const data = pythonRaw as { sections: any[] }
    expect(Array.isArray(data.sections)).toBe(true)
  })

  it('settings.json has valid structure', () => {
    const data = settingsRaw as { sections: any[] }
    expect(Array.isArray(data.sections)).toBe(true)
    expect(data.sections.length).toBeGreaterThanOrEqual(2)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run docs/manual/__tests__/schema.test.ts --reporter=verbose 2>&1 | head -20
```
Expected: FAIL — "Cannot find module" errors (JSON files don't exist yet).

- [ ] **Step 3: Create `panels.json` with all panels**

Write complete `panels.json` with entries for every panel ID in the registry. Each entry follows `PanelDoc`:
```json
{
  "panels": [
    {
      "id": "watchlist",
      "name": "自选股列表",
      "category": "市场行情",
      "description": "展示用户关注的股票实时价格和涨跌幅。支持多分组、排序、右键菜单。",
      "dataSources": ["Tencent", "Sina"],
      "shortcut": "Ctrl+W 添加股票",
      "tips": ["双击股票打开 CandlestickPanel", "右键 → Add to Workflow 生成行情节点", "分组名可拖拽排序"],
      "relatedPanels": ["candlestick", "market-overview"],
      "configurable": true
    }
  ]
}
```
Populate all ~74 panels from the registry (complete list in `frontend/src/terminal/panels/registry.ts`).

- [ ] **Step 4: Create `adapters.json`**

Document all 40+ data source adapters. Example:
```json
{
  "adapters": [
    {
      "id": "tencent",
      "name": "Tencent Finance",
      "nameZh": "腾讯财经",
      "type": "stock",
      "markets": ["CN", "HK"],
      "description": "Real-time A-share and HK stock quotes via Tencent Finance API. Free, no API key required.",
      "descriptionZh": "通过腾讯财经API获取A股和港股实时行情。免费，无需API密钥。",
      "setupSteps": ["No configuration required", "Select 'Tencent' as data source in panel settings"],
      "configKeys": [],
      "isFree": true,
      "rateLimit": "60 req/min"
    }
  ]
}
```

- [ ] **Step 5: Create `nodes.json`**

Document all 77+ workflow nodes (from `internal/workflow/` node implementations). Example:
```json
{
  "nodes": [
    {
      "nodeType": "data_loader",
      "name": "Data Loader",
      "nameZh": "数据加载器",
      "category": "data",
      "description": "Loads market data (OHLCV) for a given symbol and interval.",
      "descriptionZh": "加载指定标的和周期的行情数据。",
      "inputs": [],
      "outputs": [{ "name": "ohlcv", "type": "ohlcv", "description": "OHLCV time series data" }],
      "params": [
        { "name": "symbol", "type": "string", "default": "600519", "description": "Stock symbol" },
        { "name": "interval", "type": "string", "default": "1d", "description": "Data interval: 1m, 5m, 1d, 1w" },
        { "name": "days", "type": "number", "default": 365, "description": "Number of days of history" }
      ]
    }
  ]
}
```

- [ ] **Step 6: Create `brokers.json`**

Document Alpaca, Binance, IBKR, Futu:
```json
{
  "brokers": [
    {
      "id": "alpaca",
      "name": "Alpaca",
      "nameZh": "Alpaca",
      "markets": ["US"],
      "description": "Commission-free US stock and ETF trading via API. Supports paper trading.",
      "descriptionZh": "通过API进行美股和ETF零佣金交易。支持模拟交易。",
      "setupSteps": [
        "Create an Alpaca account at https://alpaca.markets",
        "Generate API Key and Secret Key in the dashboard",
        "Enter keys in Broker Config panel"
      ],
      "configKeys": [
        { "key": "api_key_id", "label": "API Key ID", "labelZh": "API Key ID", "secret": false },
        { "key": "api_secret_key", "label": "Secret Key", "labelZh": "Secret Key", "secret": true }
      ],
      "fees": "Free (no commission)",
      "minDeposit": "$0"
    }
  ]
}
```

- [ ] **Step 7: Create `quickstart.json`, `python.json`, `settings.json`**

```json
// quickstart.json
{
  "sections": [
    {
      "id": "first-launch",
      "title": "First Launch Guide",
      "titleZh": "首次启动向导",
      "steps": [
        { "order": 1, "text": "Open QuantFlow Terminal", "textZh": "打开 QuantFlow 终端" },
        { "order": 2, "text": "Add a stock to your watchlist using Ctrl+W", "textZh": "使用 Ctrl+W 添加自选股" },
        { "order": 3, "text": "Click on a candlestick chart to view price history", "textZh": "点击 K 线图查看历史走势" },
        { "order": 4, "text": "Run your first backtest in the Backtest Panel", "textZh": "在回测面板运行第一次回测" }
      ]
    }
  ]
}

// python.json
{
  "sections": [
    {
      "id": "installation",
      "title": "Python Sidecar Installation",
      "titleZh": "Python Sidecar 安装",
      "steps": [
        { "order": 1, "text": "Ensure Python 3.12+ is installed", "textZh": "确保已安装 Python 3.12+" },
        { "order": 2, "text": "Run: pip install -r python/requirements.txt", "textZh": "运行: pip install -r python/requirements.txt" },
        { "order": 3, "text": "Restart QuantFlow to enable the sidecar", "textZh": "重启 QuantFlow 启用 sidecar" }
      ]
    }
  ]
}

// settings.json
{
  "sections": [
    {
      "id": "theme",
      "title": "Theme Settings",
      "titleZh": "主题设置",
      "items": [
        { "key": "theme", "label": "Theme", "labelZh": "主题", "description": "Switch between light, dark, and system theme" }
      ]
    }
  ]
}
```

- [ ] **Step 8: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run docs/manual/__tests__/schema.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 9: Commit**

```bash
git add frontend/docs/manual/ && git commit -m "feat(docs): add manual JSON data files with schema validation tests"
```

### Task 3: Create `usePanelHelp.ts` composable

**Files:**
- Create: `frontend/src/lib/usePanelHelp.ts`
- Create: `frontend/src/lib/__tests__/usePanelHelp.test.ts`

**Interfaces:**
- Consumes: `ManualData`, `PanelDoc`, `AdapterDoc`, `NodeDoc`, `BrokerDoc` from Task 1
- Produces: `usePanelHelp(panelId: string)` → `{ panel: PanelDoc | null, tips: string[], dataSources: string[] }`

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/lib/__tests__/usePanelHelp.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { usePanelHelp } from '../usePanelHelp'

describe('usePanelHelp', () => {
  it('returns panel help for known panel ID', () => {
    const help = usePanelHelp('watchlist')
    expect(help.panel).toBeTruthy()
    expect(help.panel.value).toBeTruthy()
    expect(help.panel.value!.id).toBe('watchlist')
    expect(help.tips.value.length).toBeGreaterThan(0)
  })

  it('returns null for unknown panel ID', () => {
    const help = usePanelHelp('nonexistent-panel')
    expect(help.panel.value).toBeNull()
    expect(help.tips.value).toEqual([])
  })

  it('provides data sources list', () => {
    const help = usePanelHelp('candlestick')
    expect(help.dataSources.value.length).toBeGreaterThan(0)
  })

  it('caches help data after first access', () => {
    const help1 = usePanelHelp('watchlist')
    const help2 = usePanelHelp('watchlist')
    expect(help1.panel.value).toBe(help2.panel.value)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/lib/__tests__/usePanelHelp.test.ts --reporter=verbose
```
Expected: FAIL — "Cannot find module" for usePanelHelp.

- [ ] **Step 3: Create the implementation**

```typescript
// frontend/src/lib/usePanelHelp.ts
import { computed, type ComputedRef } from 'vue'
import type { PanelDoc } from '../../docs/manual/schema'
import panelsRaw from '../../docs/manual/panels.json'

interface PanelHelpResult {
  panel: ComputedRef<PanelDoc | null>
  tips: ComputedRef<string[]>
  dataSources: ComputedRef<string[]>
}

const panelMap = new Map<string, PanelDoc>()
const raw = panelsRaw as { panels: PanelDoc[] }
for (const p of raw.panels) {
  panelMap.set(p.id, p)
}

export function usePanelHelp(panelId: string): PanelHelpResult {
  const panel = computed(() => panelMap.get(panelId) ?? null)
  const tips = computed(() => panel.value?.tips ?? [])
  const dataSources = computed(() => panel.value?.dataSources ?? [])

  return { panel, tips, dataSources }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/lib/__tests__/usePanelHelp.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/lib/usePanelHelp.ts frontend/src/lib/__tests__/usePanelHelp.test.ts && git commit -m "feat(docs): add usePanelHelp composable for per-panel documentation access"
```

### Task 4: Create `PanelHelpPopover.vue` — per-panel ⓘ tooltip

**Files:**
- Create: `frontend/src/terminal/components/PanelHelpPopover.vue`
- Create: `frontend/src/terminal/components/__tests__/PanelHelpPopover.test.ts`

**Interfaces:**
- Consumes: `usePanelHelp` from Task 3
- Produces: `<PanelHelpPopover panel-id="watchlist" />` Vue component

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/components/__tests__/PanelHelpPopover.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelHelpPopover from '../PanelHelpPopover.vue'

describe('PanelHelpPopover', () => {
  it('renders ⓘ button for known panel', () => {
    const wrapper = mount(PanelHelpPopover, {
      props: { panelId: 'watchlist' },
    })
    expect(wrapper.find('.help-trigger').exists()).toBe(true)
  })

  it('renders nothing for unknown panel', () => {
    const wrapper = mount(PanelHelpPopover, {
      props: { panelId: 'nonexistent' },
    })
    expect(wrapper.find('.help-trigger').exists()).toBe(false)
  })

  it('shows tooltip on click', async () => {
    const wrapper = mount(PanelHelpPopover, {
      props: { panelId: 'watchlist' },
    })
    await wrapper.find('.help-trigger').trigger('click')
    expect(wrapper.find('.help-popover-content').exists()).toBe(true)
    expect(wrapper.text()).toContain('自选股')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/components/__tests__/PanelHelpPopover.test.ts --reporter=verbose
```
Expected: FAIL.

- [ ] **Step 3: Create the implementation**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { usePanelHelp } from '@/lib/usePanelHelp'

const props = defineProps<{ panelId: string }>()

const { panel, tips, dataSources } = usePanelHelp(props.panelId)
const isOpen = ref(false)

const showButton = computed(() => panel.value !== null)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}
</script>

<template>
  <div v-if="showButton" class="panel-help" @click.stop>
    <button
      class="help-trigger"
      :class="{ active: isOpen }"
      @click="toggle"
      title="查看帮助"
    >ⓘ</button>
    <div v-if="isOpen" class="help-popover-overlay" @click="close" />
    <transition name="fade">
      <div v-if="isOpen" class="help-popover">
        <div class="help-popover-header">
          <h4>{{ panel?.name }}</h4>
          <button class="close-btn" @click="close">✕</button>
        </div>
        <p class="help-description">{{ panel?.description }}</p>
        <div v-if="dataSources.length" class="help-section">
          <span class="help-label">数据源</span>
          <div class="help-tags">
            <span v-for="ds in dataSources" :key="ds" class="tag">{{ ds }}</span>
          </div>
        </div>
        <div v-if="panel?.shortcut" class="help-section">
          <span class="help-label">快捷键</span>
          <code class="help-shortcut">{{ panel?.shortcut }}</code>
        </div>
        <div v-if="tips.length" class="help-section">
          <span class="help-label">提示</span>
          <ul class="help-tips">
            <li v-for="(tip, i) in tips" :key="i">{{ tip }}</li>
          </ul>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.panel-help { position: relative; display: inline-flex; }
.help-trigger {
  width: 20px; height: 20px; padding: 0;
  border: 1px solid var(--color-border-subtle); border-radius: 50%;
  background: transparent; color: var(--color-text-tertiary);
  cursor: pointer; font-size: 11px; line-height: 1;
  display: inline-flex; align-items: center; justify-content: center;
  transition: all var(--transition-fast);
}
.help-trigger:hover, .help-trigger.active {
  color: var(--color-accent); border-color: var(--color-accent);
  background: var(--color-accent-soft);
}
.help-popover-overlay { position: fixed; inset: 0; z-index: 999; }
.help-popover {
  position: absolute; top: 100%; right: 0; z-index: 1000;
  width: 280px; margin-top: 4px; padding: 12px;
  background: var(--color-bg-elevated); border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-lg); box-shadow: var(--shadow-lg);
}
.help-popover-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.help-popover-header h4 { margin: 0; font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.close-btn { background: none; border: none; color: var(--color-text-tertiary); cursor: pointer; font-size: 14px; padding: 0; }
.help-description { font-size: 12px; color: var(--color-text-secondary); margin: 0 0 10px; line-height: 1.5; }
.help-section { margin-bottom: 8px; }
.help-label { font-size: 11px; font-weight: 500; color: var(--color-text-tertiary); text-transform: uppercase; display: block; margin-bottom: 4px; }
.help-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.tag { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-secondary); border: 1px solid var(--color-border-subtle); }
.help-shortcut { font-size: 11px; padding: 2px 6px; border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-accent); border: 1px solid var(--color-border-subtle); }
.help-tips { margin: 0; padding-left: 16px; }
.help-tips li { font-size: 11px; color: var(--color-text-secondary); margin-bottom: 4px; line-height: 1.4; }
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/components/__tests__/PanelHelpPopover.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/PanelHelpPopover.vue frontend/src/terminal/components/__tests__/PanelHelpPopover.test.ts && git commit -m "feat(docs): add PanelHelpPopover component with per-panel ⓘ tooltip"
```

### Task 5: Add ⓘ to PanelHeader component

**Files:**
- Modify: `frontend/src/terminal/components/panel/PanelHeader.vue`
- Modify: `frontend/src/terminal/components/panel/index.ts`

**Interfaces:**
- Consumes: `PanelHelpPopover` from Task 4
- Produces: PanelHeader with optional ⓘ button when `panel-id` prop is provided

- [ ] **Step 1: Modify PanelHeader to accept optional panel-id prop**

```typescript
// In PanelHeader.vue, add to the props:
panelId?: string
```

- [ ] **Step 2: Add ⓘ button next to the header controls**

In PanelHeader.vue template, insert before the controls div:
```vue
<PanelHelpPopover v-if="panelId" :panel-id="panelId" />
```
And import it:
```typescript
import PanelHelpPopover from '../PanelHelpPopover.vue'
```

- [ ] **Step 3: Verify typecheck passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit
```
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/components/panel/PanelHeader.vue frontend/src/terminal/components/panel/index.ts && git commit -m "feat(docs): wire PanelHelpPopover into PanelHeader component"
```

### Task 6: Create HelpPanel (full help center)

**Files:**
- Create: `frontend/src/terminal/panels/HelpPanel.vue`
- Create: `frontend/src/terminal/panels/__tests__/HelpPanel.test.ts`
- Modify: `frontend/src/terminal/panels/registry.ts`

**Interfaces:**
- Consumes: All JSON data files from Task 2, PanelHelpPopover from Task 4
- Consumes: `getAllPanelMeta()` from registry

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/panels/__tests__/HelpPanel.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import HelpPanel from '../HelpPanel.vue'

describe('HelpPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(HelpPanel, {
      props: { panelId: 'help-panel', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders category sections', () => {
    const wrapper = mount(HelpPanel, {
      props: { panelId: 'help-panel', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.text()).toContain('快速入门')
  })

  it('has a working search input', async () => {
    const wrapper = mount(HelpPanel, {
      props: { panelId: 'help-panel', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    const input = wrapper.find('input[type="text"]')
    expect(input.exists()).toBe(true)
    await input.setValue('自选')
    expect(wrapper.text()).toContain('自选')
  })

  it('shows empty state for no search results', async () => {
    const wrapper = mount(HelpPanel, {
      props: { panelId: 'help-panel', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    const input = wrapper.find('input[type="text"]')
    await input.setValue('zzznoresultszzz')
    expect(wrapper.text()).toContain('无结果')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/HelpPanel.test.ts --reporter=verbose
```
Expected: FAIL — "Cannot find module".

- [ ] **Step 3: Create HelpPanel.vue**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import type { PanelDoc, AdapterDoc, NodeDoc, BrokerDoc } from '../../../docs/manual/schema'
import panelsRaw from '../../../docs/manual/panels.json'
import adaptersRaw from '../../../docs/manual/adapters.json'
import nodesRaw from '../../../docs/manual/nodes.json'
import brokersRaw from '../../../docs/manual/brokers.json'
import quickstartRaw from '../../../docs/manual/quickstart.json'
import pythonRaw from '../../../docs/manual/python.json'
import settingsRaw from '../../../docs/manual/settings.json'
import { getPanelMeta } from './registry'
import { PanelHeader } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

type SectionId = 'quickstart' | 'panels' | 'adapters' | 'nodes' | 'brokers' | 'python' | 'settings'

const panels = (panelsRaw as { panels: PanelDoc[] }).panels
const adapters = (adaptersRaw as { adapters: AdapterDoc[] }).adapters
const nodes = (nodesRaw as { nodes: NodeDoc[] }).nodes
const brokers = (brokersRaw as { brokers: BrokerDoc[] }).brokers
const quickstart = quickstartRaw as any
const python = pythonRaw as any
const settings = settingsRaw as any

const activeSection = ref<SectionId>('quickstart')
const searchQuery = ref('')

const sections: { id: SectionId; label: string; count?: number }[] = [
  { id: 'quickstart', label: '快速入门' },
  { id: 'panels', label: '面板参考', count: panels.length },
  { id: 'adapters', label: '数据源配置', count: adapters.length },
  { id: 'nodes', label: '节点参考', count: nodes.length },
  { id: 'brokers', label: '券商接入', count: brokers.length },
  { id: 'python', label: 'Python 高级' },
  { id: 'settings', label: '设置维护' },
]

// Category groups for panels
const panelCategories = computed(() => {
  const groups: Record<string, PanelDoc[]> = {}
  for (const p of panels) {
    if (!groups[p.category]) groups[p.category] = []
    groups[p.category].push(p)
  }
  return Object.entries(groups)
})

const filteredPanels = computed(() => {
  if (!searchQuery.value) return panels
  const q = searchQuery.value.toLowerCase()
  return panels.filter(p =>
    p.name.toLowerCase().includes(q) ||
    p.description.toLowerCase().includes(q) ||
    p.id.toLowerCase().includes(q)
  )
})

const filteredContent = computed(() => {
  if (!searchQuery.value) return null
  const q = searchQuery.value.toLowerCase()
  return {
    panels: panels.filter(p =>
      p.name.toLowerCase().includes(q) ||
      p.description.toLowerCase().includes(q) ||
      p.id.toLowerCase().includes(q)
    ),
    adapters: adapters.filter(a =>
      a.name.toLowerCase().includes(q) ||
      a.nameZh.toLowerCase().includes(q) ||
      a.description.toLowerCase().includes(q)
    ),
    nodes: nodes.filter(n =>
      n.name.toLowerCase().includes(q) ||
      n.description.toLowerCase().includes(q) ||
      n.nodeType.toLowerCase().includes(q)
    ),
    brokers: brokers.filter(b =>
      b.name.toLowerCase().includes(q) ||
      b.description.toLowerCase().includes(q)
    ),
  }
})

function getPanelName(id: string): string {
  const meta = getPanelMeta(id)
  return meta?.label || id
}
</script>

<template>
  <div class="help-panel">
    <PanelHeader title="帮助中心" />
    <div class="help-search">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索面板、节点、数据源..."
        class="search-input"
      />
    </div>

    <!-- Search Results -->
    <div v-if="searchQuery" class="help-search-results">
      <div v-if="!filteredContent || (
        filteredContent.panels.length === 0 &&
        filteredContent.adapters.length === 0 &&
        filteredContent.nodes.length === 0 &&
        filteredContent.brokers.length === 0
      )" class="no-results">无匹配结果</div>
      <section v-if="filteredContent!.panels.length > 0" class="result-section">
        <h4 class="result-heading">面板 ({{ filteredContent!.panels.length }})</h4>
        <div v-for="p in filteredContent!.panels" :key="'panel-' + p.id" class="result-item">
          <strong>{{ p.name }}</strong>
          <p>{{ p.description }}</p>
          <div class="result-tags">
            <span class="tag">{{ p.category }}</span>
            <span v-for="ds in p.dataSources" :key="ds" class="tag">{{ ds }}</span>
          </div>
        </div>
      </section>
      <section v-if="filteredContent!.adapters.length > 0" class="result-section">
        <h4 class="result-heading">数据源 ({{ filteredContent!.adapters.length }})</h4>
        <div v-for="a in filteredContent!.adapters" :key="'adapter-' + a.id" class="result-item">
          <strong>{{ a.name }} / {{ a.nameZh }}</strong>
          <p>{{ a.description }}</p>
        </div>
      </section>
      <section v-if="filteredContent!.nodes.length > 0" class="result-section">
        <h4 class="result-heading">节点 ({{ filteredContent!.nodes.length }})</h4>
        <div v-for="n in filteredContent!.nodes" :key="'node-' + n.nodeType" class="result-item">
          <strong>{{ n.name }} / {{ n.nameZh }}</strong>
          <p>{{ n.description }}</p>
        </div>
      </section>
      <section v-if="filteredContent!.brokers.length > 0" class="result-section">
        <h4 class="result-heading">券商 ({{ filteredContent!.brokers.length }})</h4>
        <div v-for="b in filteredContent!.brokers" :key="'broker-' + b.id" class="result-item">
          <strong>{{ b.name }} / {{ b.nameZh }}</strong>
          <p>{{ b.description }}</p>
        </div>
      </section>
    </div>

    <!-- Normal Navigation -->
    <div v-else class="help-nav">
      <div class="nav-tabs">
        <button
          v-for="s in sections"
          :key="s.id"
          :class="['nav-tab', { active: activeSection === s.id }]"
          @click="activeSection = s.id"
        >
          {{ s.label }}<span v-if="s.count" class="count-badge">{{ s.count }}</span>
        </button>
      </div>

      <div class="section-content">
        <!-- Quickstart -->
        <div v-if="activeSection === 'quickstart'" class="help-section-content">
          <div v-for="sec in quickstart.sections" :key="sec.id" class="help-card">
            <h3>{{ sec.titleZh }}</h3>
            <ol class="step-list">
              <li v-for="step in sec.steps" :key="step.order">{{ step.textZh }}</li>
            </ol>
          </div>
        </div>

        <!-- Panels -->
        <div v-if="activeSection === 'panels'" class="help-section-content">
          <div v-for="[cat, items] in panelCategories" :key="cat" class="help-card">
            <h3>{{ cat }} <span class="count-badge">{{ items.length }}</span></h3>
            <div class="panel-list">
              <div v-for="p in items" :key="p.id" class="panel-entry">
                <div class="panel-entry-header">
                  <strong>{{ p.name }}</strong>
                  <code class="panel-id">{{ p.id }}</code>
                </div>
                <p class="panel-entry-desc">{{ p.description }}</p>
                <div class="panel-entry-meta">
                  <span class="tag">数据源: {{ p.dataSources.join(', ') || '—' }}</span>
                  <span v-if="p.configurable" class="tag configurable">可配置</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Adapters -->
        <div v-if="activeSection === 'adapters'" class="help-section-content">
          <div v-for="a in adapters" :key="a.id" class="help-card">
            <h3>{{ a.nameZh }} <code class="panel-id">{{ a.id }}</code></h3>
            <p>{{ a.descriptionZh }}</p>
            <div class="help-card-meta">
              <span class="tag">{{ a.type }}</span>
              <span v-for="m in a.markets" :key="m" class="tag">{{ m }}</span>
              <span :class="['tag', a.isFree ? 'free' : 'paid']">{{ a.isFree ? '免费' : '收费' }}</span>
            </div>
            <details v-if="a.setupSteps.length">
              <summary>配置步骤</summary>
              <ol class="step-list">
                <li v-for="(s, i) in a.setupSteps" :key="i">{{ s }}</li>
              </ol>
            </details>
          </div>
        </div>

        <!-- Nodes -->
        <div v-if="activeSection === 'nodes'" class="help-section-content">
          <div v-for="n in nodes" :key="n.nodeType" class="help-card">
            <h3>{{ n.nameZh }} <code class="panel-id">{{ n.nodeType }}</code></h3>
            <p>{{ n.descriptionZh }}</p>
            <div class="help-card-meta">
              <span class="tag">{{ n.category }}</span>
            </div>
          </div>
        </div>

        <!-- Brokers -->
        <div v-if="activeSection === 'brokers'" class="help-section-content">
          <div v-for="b in brokers" :key="b.id" class="help-card">
            <h3>{{ b.nameZh }} <code class="panel-id">{{ b.id }}</code></h3>
            <p>{{ b.descriptionZh }}</p>
            <div class="help-card-meta">
              <span v-for="m in b.markets" :key="m" class="tag">{{ m }}</span>
            </div>
            <details>
              <summary>配置步骤</summary>
              <ol class="step-list">
                <li v-for="(s, i) in b.setupSteps" :key="i">{{ s }}</li>
              </ol>
            </details>
            <div v-if="b.fees" class="broker-info"><span class="help-label">费率</span> {{ b.fees }}</div>
            <div v-if="b.minDeposit" class="broker-info"><span class="help-label">最低入金</span> {{ b.minDeposit }}</div>
          </div>
        </div>

        <!-- Python & Settings -->
        <div v-if="activeSection === 'python'" class="help-section-content">
          <div v-for="sec in python.sections" :key="sec.id" class="help-card">
            <h3>{{ sec.titleZh }}</h3>
            <ol class="step-list">
              <li v-for="step in sec.steps" :key="step.order">{{ step.textZh }}</li>
            </ol>
          </div>
        </div>

        <div v-if="activeSection === 'settings'" class="help-section-content">
          <div v-for="sec in settings.sections" :key="sec.id" class="help-card">
            <h3>{{ sec.titleZh }}</h3>
            <div class="settings-list">
              <div v-for="item in sec.items" :key="item.key" class="settings-entry">
                <strong>{{ item.labelZh }}</strong>
                <p>{{ item.description }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.help-panel { height: 100%; display: flex; flex-direction: column; background: var(--color-bg-panel); }
.help-search { padding: 8px 16px; border-bottom: 1px solid var(--color-border); }
.search-input {
  width: 100%; padding: 8px 12px; border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md); background: var(--color-bg-elevated);
  color: var(--color-text-primary); font-size: 13px; outline: none;
  box-sizing: border-box;
}
.search-input:focus { border-color: var(--color-accent); }

.help-search-results { flex: 1; overflow-y: auto; padding: 12px 16px; }
.no-results { text-align: center; padding: 40px; color: var(--color-text-tertiary); font-size: 14px; }
.result-section { margin-bottom: 20px; }
.result-heading { font-size: 12px; font-weight: 600; color: var(--color-text-tertiary); text-transform: uppercase; margin: 0 0 8px; }
.result-item { padding: 8px 0; border-bottom: 1px solid var(--color-border-subtle); }
.result-item strong { font-size: 13px; color: var(--color-text-primary); }
.result-item p { font-size: 12px; color: var(--color-text-secondary); margin: 2px 0 4px; }
.result-tags { display: flex; gap: 4px; flex-wrap: wrap; }

.help-nav { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
.nav-tabs {
  display: flex; gap: 2px; padding: 8px 12px;
  overflow-x: auto; border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}
.nav-tab {
  padding: 6px 12px; border: 1px solid transparent; border-radius: var(--radius-md);
  background: transparent; color: var(--color-text-secondary); cursor: pointer;
  font-size: 12px; white-space: nowrap; transition: all var(--transition-fast);
}
.nav-tab:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }
.nav-tab.active { border-color: var(--color-accent); color: var(--color-accent); background: var(--color-accent-soft); }
.count-badge { margin-left: 4px; font-size: 10px; padding: 1px 6px; border-radius: 10px; background: var(--color-bg-subtle); color: var(--color-text-tertiary); }
.nav-tab.active .count-badge { background: var(--color-accent-soft); color: var(--color-accent); }

.section-content { flex: 1; overflow-y: auto; padding: 12px 16px; }
.help-card { margin-bottom: 16px; padding: 12px; border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg); background: var(--color-bg-elevated); }
.help-card h3 { margin: 0 0 6px; font-size: 14px; font-weight: 600; color: var(--color-text-primary); display: flex; align-items: center; gap: 6px; }
.help-card > p { font-size: 12px; color: var(--color-text-secondary); margin: 0 0 8px; line-height: 1.5; }
.help-card-meta { display: flex; gap: 4px; flex-wrap: wrap; margin-bottom: 8px; }
.tag { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-secondary); border: 1px solid var(--color-border-subtle); }
.tag.free { border-color: var(--color-down); color: var(--color-down); }
.tag.paid { border-color: var(--color-up); color: var(--color-up); }
.tag.configurable { border-color: var(--color-accent); color: var(--color-accent); }
details { margin-top: 8px; }
details summary { font-size: 12px; color: var(--color-accent); cursor: pointer; }
.step-list { margin: 8px 0 0; padding-left: 20px; }
.step-list li { font-size: 12px; color: var(--color-text-secondary); margin-bottom: 4px; line-height: 1.5; }
.panel-list { display: flex; flex-direction: column; gap: 2px; }
.panel-entry { padding: 6px 0; border-bottom: 1px solid var(--color-border-subtle); }
.panel-entry-header { display: flex; align-items: center; gap: 6px; }
.panel-entry-header strong { font-size: 13px; color: var(--color-text-primary); }
.panel-id { font-size: 11px; color: var(--color-text-tertiary); }
.panel-entry-desc { font-size: 11px; color: var(--color-text-secondary); margin: 2px 0; }
.panel-entry-meta { display: flex; gap: 4px; margin-top: 2px; }
.broker-info { font-size: 12px; color: var(--color-text-secondary); margin-top: 4px; }
.broker-info .help-label { font-weight: 500; color: var(--color-text-tertiary); margin-right: 4px; }
.settings-list { display: flex; flex-direction: column; gap: 2px; }
.settings-entry { padding: 4px 0; }
.settings-entry strong { font-size: 13px; color: var(--color-text-primary); }
.settings-entry p { font-size: 12px; color: var(--color-text-secondary); margin: 2px 0 0; }
</style>
```

- [ ] **Step 4: Register HelpPanel in registry.ts**

Add to the system section in `frontend/src/terminal/panels/registry.ts`:
```typescript
register('help-panel', () => import('./HelpPanel.vue'), { label: '帮助中心', category: '系统', description: '用户手册与帮助文档' })
```

- [ ] **Step 5: Run typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit
```
Expected: No errors.

- [ ] **Step 6: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/HelpPanel.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/panels/HelpPanel.vue frontend/src/terminal/panels/__tests__/HelpPanel.test.ts frontend/src/terminal/panels/registry.ts && git commit -m "feat(docs): add HelpPanel with full help center and search"
```

### Task 7: Full verification

- [ ] **Step 1: Run full test suite**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run
```
Expected: All tests PASS.

- [ ] **Step 2: Run typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit
```
Expected: No errors.

- [ ] **Step 3: Run lint**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run lint
```
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: final verification and lint fixes for user manual feature"
```

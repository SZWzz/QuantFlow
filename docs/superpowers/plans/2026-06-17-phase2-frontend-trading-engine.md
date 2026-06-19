# Phase 2: Frontend + Trading Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Wails + Vue 3 desktop shell, Terminal Mode (彭博式面板), Workflow Mode (vue-flow 画布), Go Trading Engine (bar-by-bar pipeline), and Market Data Hub (Go channel pub/sub) on top of the Phase 1 workflow engine.

**Architecture:** Six serial milestones. M1 scaffolds the Wails + Vue 3 project with Go↔JS bridge. M2 builds Terminal Mode (CommandBar, DockView, 8 panels). M3 builds Workflow Mode (vue-flow canvas, NodePalette, PropertyPanel). M4 integrates both modes with Pinia stores. M5 implements the Trading Engine (OMS + PaperEngine + OrderMatcher). M6 builds the Market Data Hub (pub/sub + 8 adapters + 3-level cache).

**Tech Stack:** Wails v3, Vue 3 + TypeScript, Vite, Pinia, vue-flow, ECharts, Naive UI, SQLite WAL, Go 1.22+

**Spec:** [docs/superpowers/specs/2026-06-17-phase2-frontend-trading-engine.md](../specs/2026-06-17-phase2-frontend-trading-engine.md)

---

## Prerequisites

Before starting, verify Phase 1 is clean:

```bash
cd app && make build && make test && make lint
# All must pass. If not, fix Phase 1 first.
```

---

## Milestone 1: Wails 骨架 + 前端地基

### Task 1: Wails v3 project initialization

**Files:**
- Create: `wails.json`
- Modify: `app/main.go`
- Create: `app/app.go`
- Create: `frontend/` directory structure

- [ ] **Step 1: Install Wails v3 CLI**

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
wails3 version
```

- [ ] **Step 2: Create wails.json**

Create `wails.json` at project root:

```json
{
  "name": "quantflow",
  "outputfilename": "quantflow",
  "assetdir": "frontend/dist",
  "frontend:build": "npm run build",
  "frontend:install": "npm install",
  "frontend:dev": "npm run dev",
  "wailsjsdir": "frontend/src/lib/wailsjs",
  "version": "2",
  "author": {
    "name": "QuantFlow Contributors",
    "email": ""
  }
}
```

- [ ] **Step 3: Rewrite app/main.go as Wails entry point**

```go
package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	app := application.New(application.Options{
		Name:        "quantflow",
		Description: "QuantFlow Terminal",
		Services: []application.Service{
			application.NewService(&App{}),
		},
	})

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title:  "QuantFlow Terminal",
		Width:  1400,
		Height: 900,
		URL:    "/",
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}
```

- [ ] **Step 4: Create app/app.go — Wails-bound struct**

```go
package main

import (
	"encoding/json"
	"fmt"

	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/storage"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

type App struct {
	cfg      *config.Config
	db       *sql.DB   // will be initialized
	registry *workflow.NodeRegistry
	engine   *workflow.Engine
}

// startup is called by Wails at application start
func (a *App) startup() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	a.cfg = cfg
	logging.Setup(cfg.LogLevel)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	a.db = db

	migrations, err := storage.BuiltinMigrations()
	if err != nil {
		return err
	}
	if err := storage.Run(db, migrations); err != nil {
		return err
	}

	a.registry = workflow.NewRegistry()
	nodes.RegisterAll(a.registry)

	engine, err := workflow.NewEngine(a.registry, 256)
	if err != nil {
		return err
	}
	a.engine = engine

	return nil
}

// shutdown is called by Wails at application termination
func (a *App) shutdown() {
	if a.db != nil {
		a.db.Close()
	}
}

// GetVersion returns the application version
func (a *App) GetVersion() string {
	return a.cfg.Version
}

// ListNodes returns all registered node types
func (a *App) ListNodes() []workflow.NodeMeta {
	return a.registry.ListAll()
}

// ValidateWorkflow validates a workflow JSON definition
func (a *App) ValidateWorkflow(jsonDef string) (string, error) {
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	if err := workflow.Validate(&wf); err != nil {
		return "", err
	}
	return "valid", nil
}

// RunWorkflow executes a workflow JSON definition and returns results
func (a *App) RunWorkflow(jsonDef string) (*workflow.ExecutionResult, error) {
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return a.engine.Execute(context.Background(), &wf)
}
```

- [ ] **Step 5: Add RegisterAll to nodes package**

Add `app/internal/workflow/nodes/register.go`:

```go
package nodes

import "quantflow/internal/workflow"

func RegisterAll(r *workflow.NodeRegistry) {
	r.RegisterWithCategory("data_loader", NewDataLoaderNode, "data")
	r.RegisterWithCategory("sma", NewSMANode, "indicator")
	r.RegisterWithCategory("cross_signal", NewCrossSignalNode, "signal")
	r.RegisterWithCategory("log_output", NewLogOutputNode, "output")
	r.RegisterWithCategory("loop", NewLoopNode, "control")
}
```

- [ ] **Step 6: Verify Go build**

```bash
cd app && go build ./...
```
Expected: builds successfully (may need `go mod tidy`)

- [ ] **Step 7: Commit**

```bash
git add wails.json app/main.go app/app.go app/internal/workflow/nodes/register.go
git commit -m "feat(m1): add Wails v3 entry point and App struct with Go↔JS bridge"
```

---

### Task 2: Vue 3 + Vite frontend scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.ts`
- Create: `frontend/src/App.vue`

- [ ] **Step 1: Create package.json**

```bash
mkdir -p frontend/src
cd frontend && npm init -y
```

Then edit `frontend/package.json`:

```json
{
  "name": "quantflow-frontend",
  "private": true,
  "version": "2026.6.17",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc --noEmit && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "vue": "^3.5.0",
    "pinia": "^2.2.0",
    "vue-router": "^4.4.0",
    "@vue-flow/core": "^1.40.0",
    "@vue-flow/background": "^1.3.0",
    "@vue-flow/controls": "^1.1.0",
    "@vue-flow/minimap": "^1.5.0",
    "vue-echarts": "^7.0.0",
    "echarts": "^5.5.0",
    "monaco-editor": "^0.50.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.1.0",
    "typescript": "^5.5.0",
    "vite": "^6.0.0",
    "vitest": "^2.1.0",
    "vue-tsc": "^2.1.0",
    "@vue/test-utils": "^2.4.0",
    "jsdom": "^25.0.0"
  }
}
```

- [ ] **Step 2: Install dependencies**

```bash
cd frontend && npm install
```

- [ ] **Step 3: Create vite.config.ts**

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
```

- [ ] **Step 4: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "jsx": "preserve",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "noEmit": true,
    "paths": {
      "@/*": ["./src/*"]
    },
    "types": ["vitest/globals"]
  },
  "include": ["src/**/*.ts", "src/**/*.d.ts", "src/**/*.vue"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 5: Create index.html**

```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>QuantFlow Terminal</title>
</head>
<body>
  <div id="app"></div>
  <script type="module" src="/src/main.ts"></script>
</body>
</html>
```

- [ ] **Step 6: Create src/main.ts**

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'terminal',
      component: () => import('@/terminal/TerminalMode.vue'),
    },
    {
      path: '/workflow',
      name: 'workflow',
      component: () => import('@/workflow/WorkflowMode.vue'),
    },
  ],
})

const pinia = createPinia()
const app = createApp(App)

app.use(router)
app.use(pinia)
app.mount('#app')
```

- [ ] **Step 7: Create src/App.vue**

```vue
<script setup lang="ts">
import { useSessionStore } from '@/stores/session'

const session = useSessionStore()
</script>

<template>
  <div class="app" :class="[`theme-${session.ui.theme}`, `density-${session.ui.density}`]">
    <router-view />
  </div>
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
.app { width: 100vw; height: 100vh; overflow: hidden; }
</style>
```

- [ ] **Step 8: Create placeholder TerminalMode.vue and WorkflowMode.vue**

```vue
<!-- frontend/src/terminal/TerminalMode.vue -->
<script setup lang="ts">
</script>
<template>
  <div style="display:flex;align-items:center;justify-content:center;height:100%;background:#1a1a2e;color:#fff;">
    <h1>Terminal Mode — Coming Soon</h1>
  </div>
</template>
```

```vue
<!-- frontend/src/workflow/WorkflowMode.vue -->
<script setup lang="ts">
</script>
<template>
  <div style="display:flex;align-items:center;justify-content:center;height:100%;background:#0d1117;color:#fff;">
    <h1>Workflow Mode — Coming Soon</h1>
  </div>
</template>
```

- [ ] **Step 9: Create Pinia store skeletons**

Create all 4 store files as minimal stubs:

```typescript
// frontend/src/stores/session.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSessionStore = defineStore('session', () => {
  const ui = ref({
    theme: 'dark' as 'light' | 'dark',
    density: 'default' as 'compact' | 'default' | 'comfortable',
    language: 'zh' as 'zh' | 'en',
    mode: 'terminal' as 'terminal' | 'workflow',
  })

  function toggleMode() {
    ui.value.mode = ui.value.mode === 'terminal' ? 'workflow' : 'terminal'
  }

  return { ui, toggleMode }
})
```

```typescript
// frontend/src/stores/terminal.ts
import { defineStore } from 'pinia'

export const useTerminalStore = defineStore('terminal', () => {
  // Will be populated in M2
  return {}
})
```

```typescript
// frontend/src/stores/workflow.ts
import { defineStore } from 'pinia'

export const useWorkflowStore = defineStore('workflow', () => {
  // Will be populated in M3
  return {}
})
```

```typescript
// frontend/src/stores/data.ts
import { defineStore } from 'pinia'

export const useDataStore = defineStore('data', () => {
  // Will be populated in M2-M6
  return {}
})
```

- [ ] **Step 10: Verify frontend builds**

```bash
cd frontend && npm run build
```
Expected: builds successfully

- [ ] **Step 11: Commit**

```bash
git add frontend/
git commit -m "feat(m1): scaffold Vue 3 + Vite + Pinia frontend with router and store skeletons"
```

---

### Task 3: Wails dev mode integration

- [ ] **Step 1: Run wails dev**

```bash
wails3 dev
```

Expected: Window opens showing "Terminal Mode — Coming Soon"

- [ ] **Step 2: Verify Go↔JS bridge**

Add a test call in TerminalMode.vue:
```typescript
import { GetVersion } from '@/lib/wailsjs/main/app'
const version = await GetVersion()
console.log('QuantFlow version:', version)
```

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "feat(m1): verify Wails dev mode with Go↔JS bridge working"
```

---

## Milestone 2: Terminal Mode 核心

### Task 4: CommandBar component

**Files:**
- Create: `frontend/src/terminal/CommandBar.vue`
- Create: `frontend/src/terminal/CommandBar.test.ts`

- [ ] **Step 1: Write the test**

```typescript
// CommandBar.test.ts
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import CommandBar from './CommandBar.vue'

describe('CommandBar', () => {
  it('opens on Ctrl+K', async () => {
    const wrapper = mount(CommandBar, {
      props: { modelValue: false }
    })
    // Dispatch keyboard event
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))
    await wrapper.vm.$nextTick()
    // Should be visible
  })

  it('filters results based on query', () => {
    // ...
  })

  it('closes on Escape', () => {
    // ...
  })
})
```

- [ ] **Step 2: Implement CommandBar**

Key features:
- Full-screen overlay with centered search input
- Debounced fuzzy search (300ms)
- Results grouped by category: Panels, Commands, Symbols
- Keyboard navigation: ↑↓ arrows, Enter to select, Esc to close
- History: last 20 commands stored in terminalStore

- [ ] **Step 3: Commit**

---

### Task 5: DockView system

**Files:**
- Create: `frontend/src/terminal/DockView/DockView.vue`
- Create: `frontend/src/terminal/DockView/DockContainer.vue`
- Create: `frontend/src/terminal/DockView/DockTab.vue`
- Create: `frontend/src/terminal/DockView/DockSplitter.vue`
- Create: `frontend/src/terminal/DockView/types.ts`

- [ ] **Step 1: Define DockLayout types**

```typescript
// types.ts
export interface DockLayoutTree {
  id: string
  type: 'container' | 'tab'
  direction?: 'row' | 'column'       // container only
  splitRatios?: number[]              // container only
  children?: DockLayoutTree[]         // container only
  tabs?: DockTabState[]               // tab only
  activeTab?: string                  // tab only
}

export interface DockTabState {
  id: string
  panelId: string
  label: string
  icon: string
  params?: Record<string, any>
}
```

- [ ] **Step 2: Implement recursive DockContainer**

Key behavior:
- Renders children either as `<DockContainer>` (recursive) or `<DockTab>` (leaf)
- Splitter between children, draggable to resize
- Accepts drop events to reorder/reparent panels

- [ ] **Step 3: Implement DockTab**

Key behavior:
- Renders tab headers + active panel content
- `<component :is="panelComponent">` for dynamic panel rendering
- Close button, float button, pin button per tab
- Notifies DockView on tab close/float

- [ ] **Step 4: Implement DockView**

Key behavior:
- Wraps the root DockContainer
- Manages layout tree in terminalStore
- Provides preset layouts: single, split-h, split-v, 2x2
- Handles keyboard shortcuts: `Ctrl+数字` for layout switching

- [ ] **Step 5: Commit**

---

### Task 6: First 4 panels

**Files:**
- Create: `frontend/src/terminal/panels/WatchlistPanel.vue`
- Create: `frontend/src/terminal/panels/QuoteDetailPanel.vue`
- Create: `frontend/src/terminal/panels/CandlestickPanel.vue`
- Create: `frontend/src/terminal/panels/SystemMonitorPanel.vue`

- [ ] **Step 1: WatchlistPanel**

- Editable symbol list (add/remove/reorder)
- Live price row: symbol, last price, change%, volume
- Color coding: green↑ / red↓
- Click row → opens QuoteDetailPanel

- [ ] **Step 2: QuoteDetailPanel**

- Header: symbol + company name + exchange
- OHLCV summary: Open, High, Low, Close, Volume (big numbers)
- Mini sparkline (ECharts)
- Props: `symbol: string`

- [ ] **Step 3: CandlestickPanel**

- ECharts candlestick chart (vue-echarts)
- Date range selector (1M/3M/6M/1Y)
- MA overlays (5, 10, 20, 60) toggleable
- Volume bars at bottom
- Props: `symbol: string`, `interval: string`

- [ ] **Step 4: SystemMonitorPanel**

- CPU usage (Go runtime.NumGoroutine, runtime.MemStats)
- Active data streams count
- Broker connection statuses
- Recent workflow executions

- [ ] **Step 5: Commit**

---

### Task 7: Remaining 4 panels + PushPinBar + StatusBar

**Files:**
- Create: `frontend/src/terminal/panels/OrderEntryPanel.vue`
- Create: `frontend/src/terminal/panels/PositionPanel.vue`
- Create: `frontend/src/terminal/panels/NewsPanel.vue`
- Create: `frontend/src/terminal/panels/AIChatPanel.vue`
- Create: `frontend/src/terminal/PushPinBar.vue`
- Create: `frontend/src/terminal/StatusBar.vue`

- [ ] **Step 1: Implement OrderEntryPanel, PositionPanel, NewsPanel, AIChatPanel**

(Follow same pattern as Task 6)

- [ ] **Step 2: Implement PushPinBar**

- Horizontal bar at bottom of screen area
- Shows pinned items: symbols, panels, workflows
- Click to focus/activate

- [ ] **Step 3: Implement StatusBar**

- Bottom bar showing: connection status (● green/gray), broker count, data stream count, memory usage
- Click connection indicator → toggle offline mode

- [ ] **Step 4: Commit**

---

## Milestone 3: Workflow Mode 核心

### Task 8: WorkflowCanvas — vue-flow integration

**Files:**
- Create: `frontend/src/workflow/canvas/WorkflowCanvas.vue`
- Create: `frontend/src/workflow/canvas/CustomNode.vue`
- Create: `frontend/src/workflow/canvas/ConnectionLine.vue`

- [ ] **Step 1: Implement CustomNode**

Renders a workflow node with:
- Colored header by category (data=blue, indicator=green, signal=red, output=purple, control=orange)
- Node type label + instance ID
- Input port dots (left side)
- Output port dots (right side)
- Execution status indicator (idle/running/success/failed)
- Params summary (e.g., "period=20")

- [ ] **Step 2: Implement ConnectionLine**

- Bezier curve between ports
- Color by PortType
- Animated dash when execution running

- [ ] **Step 3: Implement WorkflowCanvas**

Wraps vue-flow `<VueFlow>`:
- Registers CustomNode type
- Binds nodes/edges to workflowStore
- Handles: onConnect, onNodeDragStop, onNodeClick, onPaneClick
- MiniMap, Controls (zoom in/out/fit), Background grid
- Keyboard: Delete=remove selected, Ctrl+Z=undo
- Drag-and-drop from NodePalette (accepts @dragenter/@dragover/@drop)

- [ ] **Step 4: Commit**

---

### Task 9: NodePalette + PropertyPanel + ExecutionLog

**Files:**
- Create: `frontend/src/workflow/NodePalette.vue`
- Create: `frontend/src/workflow/PropertyPanel.vue`
- Create: `frontend/src/workflow/ExecutionLog.vue`

- [ ] **Step 1: Implement NodePalette**

- Left sidebar, collapsible
- Search bar at top (filters by name/category)
- Categories grouped with accordion (from `ListNodes()` Go call)
- Each node item: icon, name, brief description
- Draggable → WorkflowCanvas onDrop creates node

- [ ] **Step 2: Implement PropertyPanel**

- Right sidebar, shows when node selected
- Node info: ID, NodeType, Category
- Dynamic form from ParamSchema (rendered by param type)
- Input port list + Output port list (readonly)
- Validate button (calls Go `ValidateWorkflow`)

- [ ] **Step 3: Implement ExecutionLog**

- Bottom panel or collapsible sidebar
- Terminal-style log output
- Each line: timestamp, node ID, status icon, duration, error (if any)
- Layer grouping: parallel nodes shown in same block
- Auto-scroll to bottom during execution
- Clear log button

- [ ] **Step 4: Commit**

---

### Task 10: WorkflowMode container

**Files:**
- Create/Modify: `frontend/src/workflow/WorkflowMode.vue`

- [ ] **Step 1: Layout assembly**

```vue
<script setup lang="ts">
import WorkflowCanvas from './canvas/WorkflowCanvas.vue'
import NodePalette from './NodePalette.vue'
import PropertyPanel from './PropertyPanel.vue'
import ExecutionLog from './ExecutionLog.vue'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { RunWorkflow, ListNodes } from '@/lib/wailsjs/main/app'

const workflow = useWorkflowStore()
const session = useSessionStore()

async function onRun() {
  const json = JSON.stringify(workflow.toWorkflowJSON())
  const result = await RunWorkflow(json)
  // Update nodeStatuses from result
}
</script>

<template>
  <div class="workflow-mode">
    <NodePalette />
    <div class="canvas-area">
      <div class="toolbar">
        <button @click="onRun">▶ Run (F5)</button>
        <button @click="session.toggleMode()">📊 Terminal</button>
      </div>
      <WorkflowCanvas />
    </div>
    <PropertyPanel />
    <ExecutionLog />
  </div>
</template>
```

- [ ] **Step 2: Wire keyboard shortcuts**

- `F5` → run workflow
- `Ctrl+W` → toggle to Terminal mode
- `Delete` → remove selected nodes/edges
- `Space` → fit view

- [ ] **Step 3: Commit**

---

## Milestone 4: 双模式集成

### Task 11: Pinia stores — complete implementation

**Files:**
- Modify: `frontend/src/stores/terminal.ts`
- Modify: `frontend/src/stores/workflow.ts`
- Modify: `frontend/src/stores/data.ts`
- Modify: `frontend/src/stores/session.ts`

- [ ] **Step 1: terminalStore full implementation**

- Layout tree serialization/deserialization
- Panel registry (panelId → component mapping)
- Command history management
- PushPin CRUD
- Layout presets (single, split-h, 2x2, classic, trading)

- [ ] **Step 2: workflowStore full implementation**

- Nodes/edges synced with vue-flow
- Workflow serialization (to/from Phase 1 JSON format)
- Execution state machine: idle → running → completed/failed
- Undo/redo stack (command pattern)
- Clipboard for copy/paste

- [ ] **Step 3: dataStore full implementation**

- Topic subscription management
- Quote/OHLCV cache with TTL
- Wails event listeners for real-time data push

- [ ] **Step 4: sessionStore full implementation**

- Theme/density persistence to localStorage
- Mode toggle (terminal ↔ workflow)
- Broker connection management

- [ ] **Step 5: Commit**

---

### Task 12: Terminal ↔ Workflow bidirectional flow

**Files:**
- Modify: various panel components
- Modify: `WorkflowMode.vue`

- [ ] **Step 1: Terminal → Workflow: "Add to Workflow" button**

Add to each panel component:
```vue
<button class="add-to-workflow" @click="addToWorkflow" title="Add to Workflow">
  ⊕
</button>
```

`addToWorkflow()` implementation:
1. Get panel's current state (symbol, params)
2. Map panel type → node type (WatchlistPanel → data_loader, etc.)
3. Switch to Workflow mode
4. Create node at canvas center with mapped params

- [ ] **Step 2: Workflow → Terminal: "Pin to Terminal"**

Add to node right-click context menu:
```
── Pin to Terminal
   ├── As Watchlist
   ├── As Candlestick Chart
   └── As Position Monitor
```

Implementation:
1. Create panel instance in terminalStore
2. Set panel params from node outputs
3. Switch to Terminal mode
4. Panel shows with label "WF: <workflow name>"

- [ ] **Step 3: Commit**

---

## Milestone 5: 交易引擎

### Task 13: Trading types + OMS

**Files:**
- Create: `app/internal/trading/types.go`
- Create: `app/internal/trading/oms.go`
- Create: `app/internal/trading/oms_test.go`

- [ ] **Step 1: Define core types**

```go
// types.go
package trading

type OrderSide string
const (
    SideBuy  OrderSide = "buy"
    SideSell OrderSide = "sell"
)

type OrderType string
const (
    TypeMarket OrderType = "market"
    TypeLimit  OrderType = "limit"
    TypeStop   OrderType = "stop"
)

type OrderStatus string
const (
    StatusPending   OrderStatus = "pending"
    StatusPartial   OrderStatus = "partial"
    StatusFilled    OrderStatus = "filled"
    StatusCancelled OrderStatus = "cancelled"
    StatusRejected  OrderStatus = "rejected"
)
```

- [ ] **Step 2: Implement OMS**

```go
type OMS struct {
    mu       sync.RWMutex
    orders   map[string]*Order
    trades   []*Trade
    positions map[string]*Position
}
func (o *OMS) PlaceOrder(order *Order) error
func (o *OMS) CancelOrder(orderID string) error
func (o *OMS) FillOrder(orderID string, qty, price float64) error
func (o *OMS) GetPosition(symbol string) *Position
func (o *OMS) GetAllPositions() []*Position
```

- [ ] **Step 3: Write tests (TDD)**

Test order lifecycle: Place → Partial Fill → Full Fill → Position update

- [ ] **Step 4: Commit**

---

### Task 14: OrderMatcher + PaperEngine

**Files:**
- Create: `app/internal/trading/order_matcher.go`
- Create: `app/internal/trading/order_matcher_test.go`
- Create: `app/internal/trading/paper_engine.go`
- Create: `app/internal/trading/paper_engine_test.go`

- [ ] **Step 1: Implement OrderMatcher**

```go
type OrderMatcher struct{}

func (m *OrderMatcher) Match(order Order, bar OHLCVBar) (filledQty, avgPrice float64) {
    // Market buy: fill at bar.Open
    // Limit buy: fill if bar.Low <= limitPrice, avg price = min(limitPrice, bar.Open)
    // Stop buy: trigger if bar.High >= stopPrice
    // (mirror for sell)
}
```

- [ ] **Step 2: Implement PaperEngine**

```go
type PaperEngine struct {
    matcher *OrderMatcher
    oms     *OMS
}

func (pe *PaperEngine) OnBar(bar OHLCVBar) error {
    // 1. Check pending stop orders → activate if triggered
    // 2. Match all pending orders against this bar
    // 3. Update positions with fill results
}
```

- [ ] **Step 3: Write tests**

- Market buy fills at Open
- Limit buy fills only if Low <= limit
- Stop loss triggers when High >= stopPrice
- Position P&L calculation: `(marketPrice - avgPrice) * qty`

- [ ] **Step 4: Commit**

---

### Task 15: RiskPipeline + TradingEngine

**Files:**
- Create: `app/internal/trading/risk_pipeline.go`
- Create: `app/internal/trading/risk_pipeline_test.go`
- Create: `app/internal/trading/engine.go`
- Create: `app/internal/trading/engine_test.go`

- [ ] **Step 1: Implement RiskPipeline**

```go
type RiskConfig struct {
    MaxPositionPct   float64  // max single position as % of portfolio
    StopLossPct      float64  // stop loss %
    TakeProfitPct    float64  // take profit %
    MaxDrawdownPct   float64  // max drawdown before suspend
}

type RiskPipeline struct {
    config RiskConfig
}

func (r *RiskPipeline) Check(order Order, position *Position, portfolio *Portfolio) error {
    // Returns error if order violates any risk rule
}
```

- [ ] **Step 2: Implement TradingEngine**

```go
type TradingEngine struct {
    oms          *OMS
    paperEngine  *PaperEngine
    riskPipeline *RiskPipeline
    signalCh     chan Signal
}

func (e *TradingEngine) Start(ctx context.Context)
func (e *TradingEngine) OnBar(ctx context.Context, bar OHLCVBar) error
func (e *TradingEngine) SubmitSignal(signal Signal) error
```

- [ ] **Step 3: Write integration test**

Full pipeline: Signal → Risk check → Place Order → OnBar → Match → Fill → Position P&L

Use CSV replay with known data → verify final P&L matches expected

- [ ] **Step 4: Commit**

---

### Task 16: SQLite migration 004 — trading tables

**Files:**
- Create: `app/internal/storage/migrations/004_trading.sql`

- [ ] **Step 1: Create migration SQL**

```sql
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK(side IN ('buy','sell')),
    order_type TEXT NOT NULL CHECK(order_type IN ('market','limit','stop')),
    quantity REAL NOT NULL CHECK(quantity > 0),
    price REAL,
    filled_qty REAL NOT NULL DEFAULT 0,
    filled_avg_price REAL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','partial','filled','cancelled','rejected')),
    placed_at INTEGER NOT NULL,
    filled_at INTEGER
);
CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS trades (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL NOT NULL,
    trade_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_trades_order ON trades(order_id);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol, trade_at);

CREATE TABLE IF NOT EXISTS positions (
    symbol TEXT PRIMARY KEY,
    quantity REAL NOT NULL,
    avg_price REAL NOT NULL,
    updated_at INTEGER NOT NULL
);
```

- [ ] **Step 2: Verify migration applies**

```bash
cd app && go test ./internal/storage/ -v -run TestRun
```

- [ ] **Step 3: Commit**

---

## Milestone 6: 市场数据中枢

### Task 17: MarketDataHub — Go channel pub/sub

**Files:**
- Create: `app/internal/market/hub.go`
- Create: `app/internal/market/hub_test.go`
- Create: `app/internal/market/types.go`

- [ ] **Step 1: Define market types**

```go
// types.go
type QuoteSnapshot struct {
    Symbol    string
    Last      float64
    Bid       float64
    Ask       float64
    Volume    float64
    Change    float64
    ChangePct float64
    Timestamp time.Time
}

type OHLCVBar struct {
    Date   string
    Open   float64
    High   float64
    Low    float64
    Close  float64
    Volume float64
}

type MarketMessage struct {
    Topic     string
    Data      any
    Timestamp time.Time
}
```

- [ ] **Step 2: Implement MarketDataHub**

```go
type MarketDataHub struct {
    topics    map[string]*topicBroker
    mu        sync.RWMutex
    l0Cache   *sync.Map        // topic → *CachedMessage (with TTL)
}

func NewHub() *MarketDataHub
func (h *MarketDataHub) Subscribe(topic, subID string) (<-chan MarketMessage, func())
func (h *MarketDataHub) Publish(topic string, msg MarketMessage)
func (h *MarketDataHub) GetLatest(topic string) (*MarketMessage, bool)
```

- [ ] **Step 3: Write tests**

- Subscribe → Publish → received on channel
- Multiple subscribers all receive
- Unsubscribe stops delivery
- Slow consumer doesn't block others (buffer full → drop)
- GetLatest returns cached value within TTL

- [ ] **Step 4: Commit**

---

### Task 18: Adapter interface + first 4 adapters

**Files:**
- Create: `app/internal/market/adapters/adapter.go`
- Create: `app/internal/market/adapters/yahoo.go`
- Create: `app/internal/market/adapters/yahoo_test.go`
- Create: `app/internal/market/adapters/eastmoney.go`
- Create: `app/internal/market/adapters/eastmoney_test.go`
- Create: `app/internal/market/adapters/binance.go`
- Create: `app/internal/market/adapters/binance_test.go`
- Create: `app/internal/market/adapters/polygon.go`
- Create: `app/internal/market/adapters/polygon_test.go`

- [ ] **Step 1: Define Adapter interface**

```go
type Adapter interface {
    Name() string
    Markets() []string
    FetchQuote(ctx context.Context, symbol string) (*QuoteSnapshot, error)
    FetchOHLCV(ctx context.Context, symbol, interval string, start, end time.Time) ([]OHLCVBar, error)
    HealthCheck(ctx context.Context) error
}
```

- [ ] **Step 2: Implement Yahoo adapter**

HTTP calls to Yahoo Finance API (free tier, no auth required for basic quotes)

- [ ] **Step 3: Implement EastMoney adapter**

HTTP calls to EastMoney API (A-share data, free)

- [ ] **Step 4: Implement Binance adapter**

REST + WebSocket for crypto market data

- [ ] **Step 5: Implement Polygon adapter**

HTTP calls to Polygon.io API (requires free API key,美股)

- [ ] **Step 6: Write tests with httptest mocks**

- [ ] **Step 7: Commit**

---

### Task 19: Remaining 4 adapters + data normalization

**Files:**
- Create: `app/internal/market/adapters/akshare.go`
- Create: `app/internal/market/adapters/tushare.go`
- Create: `app/internal/market/adapters/futu.go`
- Create: `app/internal/market/adapters/sina.go`
- Create: `app/internal/market/normalize.go`
- Create: `app/internal/market/normalize_test.go`

- [ ] **Step 1: Implement remaining adapters**

- AKShare (A股, Python script via gRPC or HTTP bridge)
- TuShare (A股, requires token)
- Futu (A/HK/US, requires FutuOpenD running locally)
- Sina (港股, free HTTP)

- [ ] **Step 2: Implement data normalization**

```go
func NormalizeQuote(raw map[string]any, source string) (*QuoteSnapshot, error)
func NormalizeOHLCV(raw []map[string]any, source string) ([]OHLCVBar, error)
```

Handle field name mapping, timezone conversion, volume unit conversion

- [ ] **Step 3: Commit**

---

### Task 20: L1 SQLite cache + integration

**Files:**
- Create: `app/internal/market/sqlite_cache.go`
- Create: `app/internal/market/sqlite_cache_test.go`
- Create: `app/internal/storage/migrations/005_ohlcv_cache.sql`

- [ ] **Step 1: Create migration 005**

```sql
CREATE TABLE IF NOT EXISTS ohlcv_cache (
    symbol TEXT NOT NULL,
    interval TEXT NOT NULL,
    ts INTEGER NOT NULL,
    open REAL, high REAL, low REAL, close REAL, volume REAL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (symbol, interval, ts)
) WITHOUT ROWID;
```

- [ ] **Step 2: Implement SQLite cache layer**

```go
type SQLiteOHLCVCache struct {
    db *sql.DB
}
func (c *SQLiteOHLCVCache) Get(symbol, interval string, start, end time.Time) ([]OHLCVBar, error)
func (c *SQLiteOHLCVCache) Put(symbol, interval string, bars []OHLCVBar) error
func (c *SQLiteOHLCVCache) Evict(symbol, interval string, maxRows int) error
```

- [ ] **Step 3: Wire Hub + Adapters + Cache**

```go
func (h *MarketDataHub) FetchOHLCV(ctx context.Context, symbol, interval string, start, end time.Time) ([]OHLCVBar, error) {
    // L0: check sync.Map
    // L1: check SQLite cache
    // L2: try each adapter until one succeeds
    // Backfill L1 + L0
}
```

- [ ] **Step 4: Commit**

---

### Task 21: Wire everything into App struct

**Files:**
- Modify: `app/app.go`

- [ ] **Step 1: Add market and trading components to App**

```go
type App struct {
    // ... existing
    hub      *market.MarketDataHub
    trader   *trading.TradingEngine
}

func (a *App) startup() error {
    // ... existing init
    a.hub = market.NewHub()
    a.hub.RegisterAdapter(adapters.NewYahoo())
    a.hub.RegisterAdapter(adapters.NewEastMoney())
    // ... etc

    a.trader = trading.NewTradingEngine(/* ... */)
    go a.trader.Start(context.Background())
    return nil
}

// Wails-bound functions for frontend
func (a *App) Subscribe(topic string) error { ... }
func (a *App) FetchQuote(symbol string) (*market.QuoteSnapshot, error) { ... }
func (a *App) PlaceOrder(orderJSON string) (string, error) { ... }
func (a *App) GetPositions() ([]trading.Position, error) { ... }
```

- [ ] **Step 2: Update frontend to use real data**

Wire dataStore to call Go functions via Wails bridge

- [ ] **Step 3: End-to-end smoke test**

```
1. Open QuantFlow
2. Terminal Mode: WatchlistPanel shows AAPL quote (from Yahoo adapter)
3. Switch to Workflow Mode
4. Create simple workflow: DataLoader → SMA → LogOutput
5. Run → see results in ExecutionLog
6. Pin SMA output to Terminal → see chart
```

- [ ] **Step 4: Final commit for Phase 2**

```bash
git add -A
git commit -m "feat(m6): complete MarketDataHub + TradingEngine + full integration"
```

---

## Final Verification Checklist

Before declaring Phase 2 complete:

- [ ] `wails3 dev` launches desktop app
- [ ] Terminal Mode: CommandBar opens with Ctrl+K, DockView shows 8 panels
- [ ] Workflow Mode: vue-flow canvas, drag nodes, connect, configure, run
- [ ] Workflow execution: Phase 1 nodes work via GUI (not just CLI)
- [ ] Terminal ↔ Workflow: panel → node, node → panel
- [ ] TradingEngine: signal → order → match → fill → position → P&L (tested)
- [ ] MarketDataHub: subscribe → receive quote → L0 cache hit → L1 fallback
- [ ] At least 4 adapters working (Yahoo + EastMoney + Binance + 1 more)
- [ ] `cd app && make test && make lint` all pass
- [ ] `cd frontend && npm run build` succeeds
- [ ] `cd frontend && npm run test` all pass
- [ ] CHANGELOG.md updated
- [ ] Version date updated (if different day)

---

## Phase 2.5: 数据源夯实 — Adapter 增强 + Fallback Chain + 真实数据接入

> **Status**: Added 2026-06-17.  
> **Why**: Phase 2 M6 只实现了 3 个 mock adapter（Yahoo/EastMoney/Binance），缺乏 fallback 机制和真实 HTTP 接入。A 股单一数据源极易因限流/停服导致完全不可用。参考 astockpursue 的 8+2 A 股 fallback 链设计，在 Phase 3 Python gRPC 启动前夯实 Go 端数据获取能力。

### 参考项目对比

| 维度 | QuantFlow M6 现状 | astockpursue | 差距 |
|------|:---:|:---:|:---:|
| A 股数据源数 | 1 (EastMoney, mock) | **8** (mootdx→tushare→eastmoney→tencent→futu→baidu→twelvedata→akshare) | 7 个缺口 |
| 美股数据源数 | 1 (Yahoo, mock) | 4 | 3 个缺口 |
| 港股数据源数 | 0 | 5 | 5 个缺口 |
| 加密数据源数 | 1 (Binance, mock) | 3 | 2 个缺口 |
| Fallback 机制 | ❌ | ✅ 自动按优先级切换 | 完全缺失 |
| 真实 HTTP 调用 | ❌ 全部 rand 模拟 | ✅ | 完全缺失 |
| 重试/熔断 | ❌ | ✅ retry_with_budget | 完全缺失 |
| 并发批量拉取 | ❌ | ✅ fetch_concurrent | 完全缺失 |

### 目标

1. **Adapter 接口增强**：对齐 astockpursue 的 `DataLoaderProtocol`（`IsAvailable`、`RequiresAuth`）
2. **FallbackChain + AdapterRegistry**：每个市场 3-8 个 adapter，自动按优先级降级
3. **重试/熔断**：借鉴 `retry_with_budget` + `check_budget`
4. **14 个真实 adapter**：全部对接真实 HTTP/WebSocket，零 mock
5. **缓存穿透保护**：L0 命中则跳过远程调用；adapter 级 TTL

### 新增 Adapter 全景

| 优先级 | Adapter | 市场 | 认证 | 数据 | 备注 |
|--------|---------|------|------|------|------|
| **P0** | mootdx | A 股 | 无 | OHLCV | 通达信协议，astockpursue A 股首选 |
| **P0** | akshare | A/HK/US | 无 | Quote+OHLCV+基本面 | 数据最全，HTTP API 无需 Python |
| **P0** | yfinance | US/HK | 无 | Quote+OHLCV | 已有 mock→改造为真实 HTTP |
| **P0** | tushare | A 股 | Token | OHLCV+财务 | A 股主力，财务数据不可替代 |
| **P1** | tencent | A/HK | 无 | Quote+OHLCV | 腾讯财经 HTTP 接口 |
| **P1** | baidu | A 股 | 无 | Quote | 百度财经 HTTP 接口 |
| **P1** | sina | A/HK | 无 | Quote+OHLCV | 新浪财经 HTTP 接口 |
| **P1** | binance | 加密 | 无 | Quote+OHLCV+WS | 已有 mock→改造为真实 REST+WS |
| **P1** | okx | 加密 | 无 | Quote+OHLCV | OKX 行情 |
| **P2** | coingecko | 加密 | 免费 API | Quote | 加密货币 |
| **P2** | polygon | 美股 | API Key | Quote+OHLCV | 数据质量高 |
| **P2** | twelvedata | US/HK | 免费 Key | Quote+OHLCV | 多市场 |
| **P2** | finnhub | 美股 | 免费 Key | Quote+新闻 | 新闻+基本面 |
| **P2** | futu | A/HK/US | FutuOpenD | Quote+OHLCV | 实时行情（需本地运行） |

### Fallback Chain 设计（对齐 astockpursue）

```go
var FallbackChains = map[string][]string{
    "CN":     {"mootdx", "tushare", "eastmoney", "tencent", "baidu", "sina", "akshare"},
    "US":     {"yfinance", "polygon", "twelvedata", "finnhub"},
    "HK":     {"yfinance", "tencent", "sina", "twelvedata", "akshare"},
    "CRYPTO": {"binance", "okx", "coingecko"},
}
```

### 实施里程碑

| M | 内容 | 预估 |
|---|------|------|
| **M7** | Adapter 接口增强 + AdapterRegistry + FallbackChain + 重试机制 | 0.5 天 |
| **M8** | P0 Adapter 真实接入 (mootdx, akshare, yfinance, tushare) | 1.5 天 |
| **M9** | P1 Adapter 真实接入 (tencent, baidu, sina, binance, okx) | 1.5 天 |
| **M10** | P2 Adapter 真实接入 (coingecko, polygon, twelvedata, finnhub, futu) | 1.5 天 |

### 测试策略

- 每个 adapter：`httptest` mock HTTP 响应 → 验证解析正确性
- Fallback chain：模拟第一个 adapter 不可用 → 验证自动降级到第二个
- 集成测试：`FetchWithFallback` → 验证返回正确数据 + source 元信息

### 文件变更地图

```
internal/market/
├── types.go                    # 不变
├── hub.go                      # 不变
├── hub_test.go                 # 不变
├── adapter.go                  # ★ 增强: IsAvailable, RequiresAuth, markets
├── registry.go                 # ★ NEW: AdapterRegistry + FallbackChain
├── registry_test.go            # ★ NEW
├── retry.go                    # ★ NEW: retry_with_budget, check_budget
├── retry_test.go               # ★ NEW
└── adapters/
    ├── adapter.go              # (接口移到上层)
    ├── yahoo.go                # 改造: mock → 真实 HTTP
    ├── yahoo_test.go           # ★ NEW
    ├── eastmoney.go            # 改造: mock → 真实 HTTP
    ├── eastmoney_test.go       # ★ NEW
    ├── binance.go              # 改造: mock → 真实 HTTP+WS
    ├── binance_test.go         # ★ NEW
    ├── mootdx.go               # ★ NEW: 通达信协议
    ├── mootdx_test.go          # ★ NEW
    ├── akshare.go              # ★ NEW: AKShare HTTP API
    ├── akshare_test.go         # ★ NEW
    ├── tushare.go              # ★ NEW
    ├── tushare_test.go         # ★ NEW
    ├── tencent.go              # ★ NEW: 腾讯财经
    ├── tencent_test.go         # ★ NEW
    ├── baidu.go                # ★ NEW: 百度财经
    ├── baidu_test.go           # ★ NEW
    ├── sina.go                 # ★ NEW: 新浪财经
    ├── sina_test.go            # ★ NEW
    ├── okx.go                  # ★ NEW
    ├── okx_test.go             # ★ NEW
    ├── coingecko.go            # ★ NEW
    ├── coingecko_test.go       # ★ NEW
    ├── polygon.go              # ★ NEW
    ├── polygon_test.go         # ★ NEW
    ├── twelvedata.go           # ★ NEW
    ├── twelvedata_test.go      # ★ NEW
    ├── finnhub.go              # ★ NEW
    ├── finnhub_test.go         # ★ NEW
    ├── futu.go                 # ★ NEW
    └── futu_test.go            # ★ NEW
```

### Verification Checklist

- [ ] `go test ./internal/market/...` 全部通过（含 fallback/retry/adapter 测试）
- [ ] AdapterRegistry.FetchWithFallback("CN", "000001.SZ") 返回真实数据
- [ ] 第一个 adapter 不可用时自动降级到下一个
- [ ] `retry_with_budget` 在瞬态错误时重试，永久错误时立即返回
- [ ] Hub + Adapter 集成：Subscribe → Publish 真实 Quote
- [ ] 无 mock 残留：`rand.Float64()` 不在 adapter 代码中出现

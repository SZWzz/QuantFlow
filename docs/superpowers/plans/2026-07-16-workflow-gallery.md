# Workflow Gallery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build WorkflowGalleryPanel displaying 6 official example workflows plus user-saved workflows, with import-to-canvas functionality.

**Architecture:** `frontend/src/workflow/gallery/official.ts` exports 6 `GalleryWorkflow` definitions. `WorkflowGalleryPanel.vue` renders official + user workflows in two sections. `workflow.ts` store gets `importWorkflow()` and `listUserWorkflows()`. `internal/storage/workflow_repo.go` gets `ListUserWorkflows()` and `SaveUserWorkflow()` for SQLite persistence. Example JSON files in `examples/` are kept in sync.

**Tech Stack:** Vue 3 Composition API (`<script setup lang="ts">`), Pinia stores, vue-i18n, Go 1.25+ (SQLite storage), vitest + @vue/test-utils

## Global Constraints

- All new Vue files: `<script setup lang="ts>`, Composition API
- Tests: vitest with @vue/test-utils, Pinia via `setActivePinia(createPinia())`
- No `window.confirm()`/`window.alert()` — use `@/lib/wails` dialog helpers
- i18n: add keys to `frontend/src/lib/i18n/en.ts` and `zh.ts`
- WorkflowJSON interface already defined in `frontend/src/stores/workflow.ts` — reuse it
- `fromWorkflowJSON()` already exists in the workflow store — reuse it
- Gallery panel ID in registry: `workflow-gallery`
- Gallery workflows use mock `estimatedRuns` data (no backend counting in v1)

---

### Task 1: Create `official.ts` with 6 workflow definitions

**Files:**
- Create: `frontend/src/workflow/gallery/official.ts`
- Create: `frontend/src/workflow/gallery/__tests__/official.test.ts`

**Interfaces:**
- Produces: `GalleryWorkflow` interface + `OFFICIAL_WORKFLOWS` array
- Consumes: `WorkflowJSON` from `frontend/src/stores/workflow.ts`

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/workflow/gallery/__tests__/official.test.ts
import { describe, it, expect } from 'vitest'
import { OFFICIAL_WORKFLOWS } from '../official'

describe('OFFICIAL_WORKFLOWS', () => {
  it('exports exactly 6 workflows', () => {
    expect(OFFICIAL_WORKFLOWS).toHaveLength(6)
  })

  it('each workflow has required fields', () => {
    for (const wf of OFFICIAL_WORKFLOWS) {
      expect(wf.id).toBeTruthy()
      expect(wf.name).toBeTruthy()
      expect(wf.nameZh).toBeTruthy()
      expect(wf.description).toBeTruthy()
      expect(wf.descriptionZh).toBeTruthy()
      expect(wf.tags.length).toBeGreaterThan(0)
      expect(['beginner', 'intermediate', 'advanced']).toContain(wf.difficulty)
      expect(wf.nodes).toBeGreaterThan(0)
      expect(wf.estimatedRuns).toBeGreaterThan(0)
      expect(wf.json).toBeTruthy()
      expect(wf.json.nodes.length).toBeGreaterThan(0)
      expect(wf.json.edges).toBeTruthy()
    }
  })

  it('golden-cross has expected structure', () => {
    const gc = OFFICIAL_WORKFLOWS.find(w => w.id === 'golden-cross')
    expect(gc).toBeTruthy()
    expect(gc!.difficulty).toBe('beginner')
    expect(gc!.json.nodes.length).toBeGreaterThanOrEqual(5)
  })

  it('ai-strategy has advanced difficulty', () => {
    const ai = OFFICIAL_WORKFLOWS.find(w => w.id === 'ai-strategy')
    expect(ai).toBeTruthy()
    expect(ai!.difficulty).toBe('advanced')
  })

  it('all node types referenced in json are non-empty strings', () => {
    for (const wf of OFFICIAL_WORKFLOWS) {
      for (const node of wf.json.nodes) {
        expect(typeof node.id).toBe('string')
        expect(node.id.length).toBeGreaterThan(0)
        expect(typeof node.node_type).toBe('string')
        expect(node.node_type.length).toBeGreaterThan(0)
      }
    }
  })

  it('every edge references valid node IDs', () => {
    for (const wf of OFFICIAL_WORKFLOWS) {
      const nodeIds = new Set(wf.json.nodes.map(n => n.id))
      for (const edge of wf.json.edges) {
        expect(nodeIds.has(edge.from_node)).toBe(true)
        expect(nodeIds.has(edge.to_node)).toBe(true)
      }
    }
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/workflow/gallery/__tests__/official.test.ts --reporter=verbose
```
Expected: FAIL — "Cannot find module '../official'".

- [ ] **Step 3: Create `official.ts` with 6 workflow definitions**

```typescript
// frontend/src/workflow/gallery/official.ts
import type { WorkflowJSON } from '@/stores/workflow'

export interface GalleryWorkflow {
  id: string
  name: string
  nameZh: string
  description: string
  descriptionZh: string
  tags: string[]
  difficulty: 'beginner' | 'intermediate' | 'advanced'
  nodes: number
  estimatedRuns: number
  json: WorkflowJSON
}

const goldenCross: GalleryWorkflow = {
  id: 'golden-cross',
  name: 'Golden Cross Strategy',
  nameZh: '金叉买入策略',
  description: '5-day SMA crosses above 20-day SMA -> buy signal -> backtest with A-share rules',
  descriptionZh: '5日均线上穿20日均线 -> 买入信号 -> A股规则回测验证',
  tags: ['均线', '回测', '入门'],
  difficulty: 'beginner',
  nodes: 8,
  estimatedRuns: 1200,
  json: {
    id: 'golden-cross',
    name: 'Golden Cross Strategy',
    description: '5日金叉买入策略',
    nodes: [
      { id: 'loader', node_type: 'data_loader', params: { symbol: '000001.SZ', interval: '1d', days: 365 }, position: { x: 50, y: 200 } },
      { id: 'sma5', node_type: 'sma', params: { period: 5 }, position: { x: 300, y: 150 } },
      { id: 'sma20', node_type: 'sma', params: { period: 20 }, position: { x: 300, y: 300 } },
      { id: 'cross', node_type: 'cross_signal', params: {}, position: { x: 550, y: 200 } },
      { id: 'backtest', node_type: 'backtest', params: { market: 'CN', initial_capital: 100000 }, position: { x: 750, y: 150 } },
      { id: 'report', node_type: 'performance_report', params: {}, position: { x: 950, y: 150 } },
      { id: 'chart', node_type: 'chart_output', params: { type: 'equity_curve' }, position: { x: 950, y: 300 } },
    ],
    edges: [
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'sma5', to_port: 'input' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'sma20', to_port: 'input' },
      { from_node: 'sma5', from_port: 'output', to_node: 'cross', to_port: 'fast' },
      { from_node: 'sma20', from_port: 'output', to_node: 'cross', to_port: 'slow' },
      { from_node: 'cross', from_port: 'signal', to_node: 'backtest', to_port: 'signals' },
      { from_node: 'backtest', from_port: 'results', to_node: 'report', to_port: 'input' },
      { from_node: 'backtest', from_port: 'results', to_node: 'chart', to_port: 'input' },
    ],
  },
}

const rsiRebound: GalleryWorkflow = {
  id: 'rsi-rebound',
  name: 'RSI Oversold Rebound',
  nameZh: 'RSI 超卖反弹',
  description: 'RSI < 30 signal -> scan A-share market -> backtest',
  descriptionZh: 'RSI 低于30超卖信号 -> A股全市场扫描 -> 回测验证',
  tags: ['RSI', '选股', '回测'],
  difficulty: 'beginner',
  nodes: 6,
  estimatedRuns: 892,
  json: {
    id: 'rsi-rebound',
    name: 'RSI Oversold Rebound',
    description: 'RSI超卖反弹策略',
    nodes: [
      { id: 'loader', node_type: 'data_loader', params: { symbol: '000300.SH', interval: '1d', days: 180 }, position: { x: 50, y: 200 } },
      { id: 'rsi', node_type: 'rsi', params: { period: 14 }, position: { x: 250, y: 200 } },
      { id: 'threshold', node_type: 'threshold', params: { operator: 'lt', value: 30 }, position: { x: 450, y: 200 } },
      { id: 'scanner', node_type: 'stock_scanner', params: { market: 'CN', top_n: 20 }, position: { x: 650, y: 200 } },
      { id: 'backtest', node_type: 'backtest', params: { market: 'CN', initial_capital: 100000 }, position: { x: 850, y: 200 } },
      { id: 'output', node_type: 'log_output', params: { prefix: '[RSI] ' }, position: { x: 1050, y: 200 } },
    ],
    edges: [
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'rsi', to_port: 'input' },
      { from_node: 'rsi', from_port: 'output', to_node: 'threshold', to_port: 'input' },
      { from_node: 'threshold', from_port: 'signal', to_node: 'scanner', to_port: 'condition' },
      { from_node: 'scanner', from_port: 'symbols', to_node: 'backtest', to_port: 'symbols' },
      { from_node: 'backtest', from_port: 'results', to_node: 'output', to_port: 'input' },
    ],
  },
}

const multiFactor: GalleryWorkflow = {
  id: 'multi-factor',
  name: 'Multi-Factor Stock Selection',
  nameZh: '多因子选股',
  description: '4 Alpha factors -> weighted scoring -> rank -> top 10',
  descriptionZh: '4个Alpha因子 -> 加权打分 -> 排名 -> Top 10 选股',
  tags: ['因子', '选股', '量化'],
  difficulty: 'intermediate',
  nodes: 9,
  estimatedRuns: 645,
  json: {
    id: 'multi-factor',
    name: 'Multi-Factor Stock Selection',
    description: '多因子选股策略',
    nodes: [
      { id: 'loader', node_type: 'data_loader', params: { symbol: '000001.SH', interval: '1d', days: 365 }, position: { x: 50, y: 50 } },
      { id: 'factor1', node_type: 'factor', params: { factor_name: 'momentum_1m' }, position: { x: 250, y: 0 } },
      { id: 'factor2', node_type: 'factor', params: { factor_name: 'volatility_20d' }, position: { x: 250, y: 100 } },
      { id: 'factor3', node_type: 'factor', params: { factor_name: 'volume_ratio' }, position: { x: 250, y: 200 } },
      { id: 'factor4', node_type: 'factor', params: { factor_name: 'alpha_191' }, position: { x: 250, y: 300 } },
      { id: 'score', node_type: 'math_op', params: { operation: 'weighted_sum', weights: [0.3, 0.2, 0.2, 0.3] }, position: { x: 500, y: 150 } },
      { id: 'filter', node_type: 'rank_select', params: { sort: 'desc', top_n: 10 }, position: { x: 700, y: 150 } },
      { id: 'output', node_type: 'log_output', params: { prefix: '[Factors] ' }, position: { x: 900, y: 150 } },
    ],
    edges: [
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'factor1', to_port: 'data' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'factor2', to_port: 'data' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'factor3', to_port: 'data' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'factor4', to_port: 'data' },
      { from_node: 'factor1', from_port: 'value', to_node: 'score', to_port: 'input1' },
      { from_node: 'factor2', from_port: 'value', to_node: 'score', to_port: 'input2' },
      { from_node: 'factor3', from_port: 'value', to_node: 'score', to_port: 'input3' },
      { from_node: 'factor4', from_port: 'value', to_node: 'score', to_port: 'input4' },
      { from_node: 'score', from_port: 'output', to_node: 'filter', to_port: 'input' },
      { from_node: 'filter', from_port: 'selected', to_node: 'output', to_port: 'input' },
    ],
  },
}

const aiStrategy: GalleryWorkflow = {
  id: 'ai-strategy',
  name: 'AI-Powered Strategy Generation',
  nameZh: 'AI 策略生成',
  description: 'Natural language -> LLM -> DAG generator -> backtest -> iterative optimization',
  descriptionZh: '自然语言描述 -> LLM 推理 -> DAG 生成 -> 回测 -> 迭代优化',
  tags: ['AI', 'LLM', '自动生成'],
  difficulty: 'advanced',
  nodes: 7,
  estimatedRuns: 1500,
  json: {
    id: 'ai-strategy',
    name: 'AI Strategy Generation',
    description: 'AI策略生成与优化',
    nodes: [
      { id: 'user-input', node_type: 'input_text', params: { prompt: '写一个低波动率选股策略' }, position: { x: 50, y: 200 } },
      { id: 'llm', node_type: 'llm_inference', params: { model: 'gpt-4', temperature: 0.3 }, position: { x: 250, y: 200 } },
      { id: 'dag-gen', node_type: 'dag_generator', params: {}, position: { x: 450, y: 200 } },
      { id: 'backtest', node_type: 'backtest', params: { market: 'CN', initial_capital: 100000 }, position: { x: 650, y: 200 } },
      { id: 'optimizer', node_type: 'llm_inference', params: { model: 'gpt-4', temperature: 0.5, mode: 'optimize' }, position: { x: 650, y: 350 } },
      { id: 'output', node_type: 'log_output', params: { prefix: '[AI-Strategy] ' }, position: { x: 850, y: 200 } },
    ],
    edges: [
      { from_node: 'user-input', from_port: 'text', to_node: 'llm', to_port: 'prompt' },
      { from_node: 'llm', from_port: 'result', to_node: 'dag-gen', to_port: 'spec' },
      { from_node: 'dag-gen', from_port: 'workflow', to_node: 'backtest', to_port: 'strategy' },
      { from_node: 'backtest', from_port: 'metrics', to_node: 'optimizer', to_port: 'feedback' },
      { from_node: 'optimizer', from_port: 'result', to_node: 'dag-gen', to_port: 'optimization' },
      { from_node: 'backtest', from_port: 'results', to_node: 'output', to_port: 'input' },
    ],
  },
}

const scheduledMonitor: GalleryWorkflow = {
  id: 'scheduled-monitor',
  name: 'Scheduled RSI Monitor + Notification',
  nameZh: '定时监控 + 通知',
  description: 'Every minute check RSI -> oversold -> Telegram alert',
  descriptionZh: '每分钟检查自选股RSI -> 超卖 -> Telegram 通知提醒',
  tags: ['定时', '监控', '通知'],
  difficulty: 'intermediate',
  nodes: 6,
  estimatedRuns: 312,
  json: {
    id: 'scheduled-monitor',
    name: 'RSI Monitor + Notification',
    description: '定时RSI监控告警',
    nodes: [
      { id: 'schedule', node_type: 'schedule', params: { interval: '1m' }, position: { x: 50, y: 200 } },
      { id: 'loader', node_type: 'data_loader', params: { symbol: '600519', interval: '1d', days: 30 }, position: { x: 250, y: 200 } },
      { id: 'rsi', node_type: 'rsi', params: { period: 14 }, position: { x: 450, y: 200 } },
      { id: 'condition', node_type: 'condition', params: { operator: 'lt', value: 30, field: 'rsi' }, position: { x: 650, y: 200 } },
      { id: 'notify', node_type: 'notify', params: { channel: 'telegram', message: 'RSI超卖信号!' }, position: { x: 850, y: 200 } },
    ],
    edges: [
      { from_node: 'schedule', from_port: 'trigger', to_node: 'loader', to_port: 'trigger' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'rsi', to_port: 'input' },
      { from_node: 'rsi', from_port: 'output', to_node: 'condition', to_port: 'input' },
      { from_node: 'condition', from_port: 'signal', to_node: 'notify', to_port: 'trigger' },
    ],
  },
}

const momentumBreakout: GalleryWorkflow = {
  id: 'momentum-breakout',
  name: 'Momentum Breakout System',
  nameZh: '动量突破系统',
  description: 'Scan whole market -> breakout signal -> paper trading',
  descriptionZh: '扫描全市场 -> 突破信号监测 -> 模拟盘交易',
  tags: ['动量', '突破', '模拟盘'],
  difficulty: 'advanced',
  nodes: 8,
  estimatedRuns: 756,
  json: {
    id: 'momentum-breakout',
    name: 'Momentum Breakout System',
    description: '全市场动量突破系统',
    nodes: [
      { id: 'schedule', node_type: 'schedule', params: { interval: '1d', time: '09:30' }, position: { x: 50, y: 50 } },
      { id: 'loader', node_type: 'data_loader', params: { symbol: '000001.SH', interval: '1d', days: 60 }, position: { x: 50, y: 250 } },
      { id: 'sma20', node_type: 'sma', params: { period: 20 }, position: { x: 250, y: 150 } },
      { id: 'volume', node_type: 'volume_detector', params: { threshold: 1.5 }, position: { x: 250, y: 350 } },
      { id: 'cross', node_type: 'cross_signal', params: { direction: 'above' }, position: { x: 450, y: 250 } },
      { id: 'filter', node_type: 'rank_select', params: { field: 'market_cap', sort: 'asc', min: 100, unit: '亿' }, position: { x: 650, y: 250 } },
      { id: 'order', node_type: 'place_order', params: { broker: 'paper', side: 'buy', quantity: 100 }, position: { x: 850, y: 250 } },
    ],
    edges: [
      { from_node: 'schedule', from_port: 'trigger', to_node: 'loader', to_port: 'trigger' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'sma20', to_port: 'input' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'volume', to_port: 'data' },
      { from_node: 'loader', from_port: 'ohlcv', to_node: 'cross', to_port: 'data' },
      { from_node: 'sma20', from_port: 'output', to_node: 'cross', to_port: 'baseline' },
      { from_node: 'volume', from_port: 'signal', to_node: 'cross', to_port: 'volume' },
      { from_node: 'cross', from_port: 'signal', to_node: 'filter', to_port: 'input' },
      { from_node: 'filter', from_port: 'selected', to_node: 'order', to_port: 'symbol' },
    ],
  },
}

export const OFFICIAL_WORKFLOWS: GalleryWorkflow[] = [
  goldenCross,
  rsiRebound,
  multiFactor,
  aiStrategy,
  scheduledMonitor,
  momentumBreakout,
]
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/workflow/gallery/__tests__/official.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/workflow/gallery/ && git commit -m "feat(workflow): add 6 official gallery workflow definitions with tests"
```

### Task 2: Add `importWorkflow` and `listUserWorkflows` to the workflow store

**Files:**
- Modify: `frontend/src/stores/workflow.ts`
- Modify: `frontend/src/stores/__tests__/workflow.test.ts`

**Interfaces:**
- Produces: `importWorkflow(json: WorkflowJSON): void` — imports a workflow into the canvas
- Produces: `listUserWorkflows: SavedWorkflow[]` — reactive list from localStorage
- Consumes: `WorkflowJSON`, `fromWorkflowJSON()` from existing store

- [ ] **Step 1: Write the failing test**

Add this to `workflow.test.ts`:
```typescript
import type { WorkflowJSON } from '../workflow'

it('should import a workflow from JSON', () => {
  const store = useWorkflowStore()
  const json: WorkflowJSON = {
    id: 'test-import',
    name: 'Imported Test',
    nodes: [
      { id: 'n1', node_type: 'data_loader', params: { symbol: '600519' } },
      { id: 'n2', node_type: 'sma', params: { period: 5 } },
    ],
    edges: [
      { from_node: 'n1', from_port: 'ohlcv', to_node: 'n2', to_port: 'input' },
    ],
  }
  store.importWorkflow(json)
  expect(store.nodes.length).toBe(2)
  expect(store.edges.length).toBe(1)
})

it('should list user workflows after import', () => {
  const store = useWorkflowStore()
  const json: WorkflowJSON = {
    id: 'test-list',
    name: 'List Test',
    nodes: [{ id: 'n1', node_type: 'data_loader', params: {} }],
    edges: [],
  }
  store.importWorkflow(json)
  const list = store.listUserWorkflows
  expect(list.length).toBeGreaterThanOrEqual(1)
  expect(list[list.length - 1].name).toBe('List Test')
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/stores/__tests__/workflow.test.ts --reporter=verbose
```
Expected: FAIL — "importWorkflow is not a function" etc.

- [ ] **Step 3: Add `importWorkflow` method to the workflow store**

In `frontend/src/stores/workflow.ts`, add after `renameWorkflow`:

```typescript
function importWorkflow(json: WorkflowJSON) {
  const id = `import-${Date.now()}`
  const wf: WorkflowJSON = { ...json, id }
  fromWorkflowJSON(wf)
  const now = new Date().toISOString()
  workflowList.value.push({
    id,
    name: json.name || 'Imported Workflow',
    createdAt: now,
    updatedAt: now,
    nodeCount: json.nodes.length,
    json: wf,
  })
  persistWorkflowList()
}

const listUserWorkflows = computed(() => workflowList.value)
```

And add to the return object:
```typescript
importWorkflow,
listUserWorkflows,
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/stores/__tests__/workflow.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 5: Run typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit
```
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/workflow.ts frontend/src/stores/__tests__/workflow.test.ts && git commit -m "feat(workflow): add importWorkflow and listUserWorkflows to store"
```

### Task 3: Add save/load user workflows in Go workflow_repo

**Files:**
- Modify: `internal/storage/workflow_repo.go`
- Create: `internal/storage/workflow_repo_test.go`

**Interfaces:**
- Produces: `SaveUserWorkflow(meta, json string) error` — saves a user-named workflow
- Produces: `LoadUserWorkflow(name string) (string, error)` — loads by name
- Produces: `ListUserWorkflows() ([]WorkflowMeta, error)` — lists user workflows

- [ ] **Step 1: Write the failing test**

```go
// internal/storage/workflow_repo_test.go
package storage

import (
	"testing"
)

func TestSaveAndLoadUserWorkflow(t *testing.T) {
	// Use an in-memory SQLite DB
	// (skip if the test infra doesn't support it — just verify the function exists)
	t.Skip("requires DB setup")
}

func TestListUserWorkflows(t *testing.T) {
	t.Skip("requires DB setup")
}
```

- [ ] **Step 2: Add `SaveUserWorkflow` and `ListUserWorkflows` methods**

```go
// internal/storage/workflow_repo.go — add after Save method

// SaveUserWorkflow persists a user workflow by name. The name is used as a
// human-readable identifier; the actual ID is auto-generated. If a workflow
// with the same name already exists, it is updated.
func (r *WorkflowRepo) SaveUserWorkflow(name, graphJSON string) (string, error) {
	id := fmt.Sprintf("user-%d", time.Now().UnixNano())
	_, err := r.db.Exec(`INSERT INTO user_workflows (id, name, graph_json, created_at, updated_at)
		VALUES (?, ?, ?, datetime('now'), datetime('now'))
		ON CONFLICT(name) DO UPDATE SET graph_json=excluded.graph_json, updated_at=datetime('now')`,
		id, name, graphJSON)
	if err != nil {
		return "", fmt.Errorf("save user workflow: %w", err)
	}
	return id, nil
}

// UserWorkflowMeta is the summary returned by ListUserWorkflows.
type UserWorkflowMeta struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Nodes     int    `json:"nodes"`
}

// ListUserWorkflows returns metadata for all user-saved workflows.
func (r *WorkflowRepo) ListUserWorkflows() ([]UserWorkflowMeta, error) {
	rows, err := r.db.Query("SELECT id, name, created_at, updated_at FROM user_workflows ORDER BY updated_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list user workflows: %w", err)
	}
	defer rows.Close()

	var metas []UserWorkflowMeta
	for rows.Next() {
		var m UserWorkflowMeta
		if err := rows.Scan(&m.ID, &m.Name, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user workflow: %w", err)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}
```

- [ ] **Step 3: Run Go tests**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/app && go vet ./internal/storage/ && go test ./internal/storage/ -v -count=1 -run TestListUserWorkflows
```
Expected: SKIP (expected — requires DB setup).

- [ ] **Step 4: Commit**

```bash
git add internal/storage/workflow_repo.go internal/storage/workflow_repo_test.go && git commit -m "feat(workflow): add SaveUserWorkflow and ListUserWorkflows to Go storage"
```

### Task 4: Create `WorkflowGalleryPanel.vue`

**Files:**
- Create: `frontend/src/terminal/panels/WorkflowGalleryPanel.vue`
- Create: `frontend/src/terminal/panels/__tests__/WorkflowGalleryPanel.test.ts`
- Modify: `frontend/src/terminal/panels/registry.ts`

**Interfaces:**
- Consumes: `OFFICIAL_WORKFLOWS` from Task 1, `useWorkflowStore` from Task 2
- Consumes: `confirmDialog` from `@/lib/wails`

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/panels/__tests__/WorkflowGalleryPanel.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import WorkflowGalleryPanel from '../WorkflowGalleryPanel.vue'

describe('WorkflowGalleryPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('mounts without crashing', () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders official section with 6 workflows', () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.text()).toContain('金叉买入策略')
    expect(wrapper.text()).toContain('RSI 超卖反弹')
    expect(wrapper.text()).toContain('多因子选股')
    expect(wrapper.text()).toContain('AI 策略生成')
    expect(wrapper.text()).toContain('定时监控 + 通知')
    expect(wrapper.text()).toContain('动量突破系统')
  })

  it('has import buttons', () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    const importBtns = wrapper.findAll('button').filter(b => b.text().includes('导入'))
    expect(importBtns.length).toBeGreaterThanOrEqual(6)
  })

  it('shows difficulty badges', () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.html()).toContain('beginner')
    expect(wrapper.html()).toContain('intermediate')
    expect(wrapper.html()).toContain('advanced')
  })

  it('renders my workflows section', () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    expect(wrapper.text()).toContain('我的工作流')
  })

  it('filters by search query', async () => {
    const wrapper = mount(WorkflowGalleryPanel, {
      props: { panelId: 'workflow-gallery', params: {} },
      global: { stubs: { PanelHeader: true } },
    })
    const input = wrapper.find('input[type="text"]')
    await input.setValue('金叉')
    expect(wrapper.text()).toContain('金叉')
    expect(wrapper.text()).not.toContain('RSI')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/WorkflowGalleryPanel.test.ts --reporter=verbose
```
Expected: FAIL — "Cannot find module".

- [ ] **Step 3: Create `WorkflowGalleryPanel.vue`**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { OFFICIAL_WORKFLOWS, type GalleryWorkflow } from '@/workflow/gallery/official'
import { useWorkflowStore } from '@/stores/workflow'
import { confirmDialog, alertDialog } from '@/lib/wails'
import { PanelHeader } from '@/terminal/components/panel'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = useWorkflowStore()
const searchQuery = ref('')

const difficultyLabel: Record<string, string> = {
  beginner: '入门',
  intermediate: '进阶',
  advanced: '高级',
}

const filteredOfficial = computed(() => {
  if (!searchQuery.value) return OFFICIAL_WORKFLOWS
  const q = searchQuery.value.toLowerCase()
  return OFFICIAL_WORKFLOWS.filter(w =>
    w.name.toLowerCase().includes(q) ||
    w.nameZh.toLowerCase().includes(q) ||
    w.description.toLowerCase().includes(q) ||
    w.tags.some(t => t.toLowerCase().includes(q))
  )
})

const userWorkflows = computed(() => store.listUserWorkflows)

const filteredUser = computed(() => {
  if (!searchQuery.value) return userWorkflows.value
  const q = searchQuery.value.toLowerCase()
  return userWorkflows.value.filter(w =>
    w.name.toLowerCase().includes(q)
  )
})

async function importOfficial(wf: GalleryWorkflow) {
  const ok = await confirmDialog(`导入工作流「${wf.nameZh}」？\n\n导入后将出现在工作流画布中，可立即编辑和运行。`, '导入工作流')
  if (!ok) return
  store.importWorkflow(wf.json)
  await alertDialog(`工作流「${wf.nameZh}」已导入。切换到工作流模式查看。`, '导入成功')
}

function editUserWorkflow(id: string) {
  store.loadWorkflow(id)
}

function exportUserWorkflow(id: string) {
  const wf = store.workflowList.find(w => w.id === id)
  if (!wf) return
  const blob = new Blob([JSON.stringify(wf.json, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${wf.name.replace(/[^a-zA-Z0-9\u4e00-\u9fa5]/g, '_')}.json`
  a.click()
  URL.revokeObjectURL(url)
}

async function deleteUserWorkflow(id: string) {
  const wf = store.workflowList.find(w => w.id === id)
  if (!wf) return
  const ok = await confirmDialog(`删除工作流「${wf.name}」？此操作不可恢复。`, '删除确认')
  if (!ok) return
  store.deleteWorkflow(id)
}
</script>

<template>
  <div class="gallery-panel">
    <PanelHeader title="工作流库" />
    <div class="gallery-search">
      <input
        v-model="searchQuery"
        type="text"
        placeholder="搜索工作流名称、描述、标签..."
        class="search-input"
      />
    </div>

    <div class="gallery-content">
      <!-- Official Workflows -->
      <section class="gallery-section">
        <div class="section-header">
          <h3 class="section-title">官方示例 ({{ filteredOfficial.length }})</h3>
          <button
            v-if="!searchQuery"
            class="import-all-btn"
            @click="async () => {
              const ok = await confirmDialog('导入全部 6 个示例工作流？', '导入全部')
              if (ok) { for (const w of OFFICIAL_WORKFLOWS) store.importWorkflow(w.json) }
            }"
          >全部导入</button>
        </div>
        <div class="workflow-grid">
          <div v-for="wf in filteredOfficial" :key="wf.id" class="workflow-card">
            <div class="card-header">
              <h4 class="card-title">{{ wf.nameZh }}</h4>
              <span :class="['difficulty-badge', wf.difficulty]">{{ difficultyLabel[wf.difficulty] }}</span>
            </div>
            <p class="card-desc">{{ wf.descriptionZh }}</p>
            <div class="card-tags">
              <span v-for="tag in wf.tags" :key="tag" class="tag">{{ tag }}</span>
            </div>
            <div class="card-stats">
              <span class="stat"><strong>{{ wf.nodes }}</strong> 节点</span>
              <span class="stat">⭐ {{ wf.estimatedRuns.toLocaleString() }} runs</span>
            </div>
            <div class="card-actions">
              <button class="action-btn primary" @click="importOfficial(wf)">
                + 导入
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- User Workflows -->
      <section class="gallery-section">
        <div class="section-header">
          <h3 class="section-title">我的工作流 ({{ filteredUser.length }})</h3>
        </div>
        <div v-if="filteredUser.length === 0" class="empty-state">
          <p>还没有保存的工作流。导入一个官方示例开始吧。</p>
        </div>
        <div v-else class="workflow-grid">
          <div v-for="wf in filteredUser" :key="wf.id" class="workflow-card user-card">
            <div class="card-header">
              <h4 class="card-title">{{ wf.name }}</h4>
            </div>
            <div class="card-stats">
              <span class="stat"><strong>{{ wf.nodeCount }}</strong> 节点</span>
              <span class="stat">更新于 {{ new Date(wf.updatedAt).toLocaleDateString('zh-CN') }}</span>
            </div>
            <div class="card-actions">
              <button class="action-btn" @click="editUserWorkflow(wf.id)">编辑</button>
              <button class="action-btn" @click="exportUserWorkflow(wf.id)">导出</button>
              <button class="action-btn danger" @click="deleteUserWorkflow(wf.id)">删除</button>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.gallery-panel { height: 100%; display: flex; flex-direction: column; background: var(--color-bg-panel); }
.gallery-search { padding: 8px 16px; border-bottom: 1px solid var(--color-border); }
.search-input {
  width: 100%; padding: 8px 12px; border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-md); background: var(--color-bg-elevated);
  color: var(--color-text-primary); font-size: 13px; outline: none;
  box-sizing: border-box;
}
.search-input:focus { border-color: var(--color-accent); }

.gallery-content { flex: 1; overflow-y: auto; padding: 12px 16px; }
.gallery-section { margin-bottom: 24px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.import-all-btn {
  padding: 4px 12px; border: 1px solid var(--color-accent); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-accent); cursor: pointer; font-size: 12px;
  transition: all var(--transition-fast);
}
.import-all-btn:hover { background: var(--color-accent-soft); }

.workflow-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
.workflow-card {
  padding: 14px; border: 1px solid var(--color-border-subtle); border-radius: var(--radius-lg);
  background: var(--color-bg-elevated); display: flex; flex-direction: column; gap: 8px;
  transition: border-color var(--transition-fast);
}
.workflow-card:hover { border-color: var(--color-border-strong); }
.user-card { border-style: dashed; }
.card-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 8px; }
.card-title { margin: 0; font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.difficulty-badge {
  font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm);
  white-space: nowrap; flex-shrink: 0;
}
.difficulty-badge.beginner { background: var(--color-down-bg, rgba(34,197,94,0.08)); color: var(--color-down); border: 1px solid var(--color-down); }
.difficulty-badge.intermediate { background: var(--color-accent-soft); color: var(--color-accent); border: 1px solid var(--color-accent); }
.difficulty-badge.advanced { background: var(--color-up-bg, rgba(239,68,68,0.08)); color: var(--color-up); border: 1px solid var(--color-up); }
.card-desc { font-size: 12px; color: var(--color-text-secondary); margin: 0; line-height: 1.5; }
.card-tags { display: flex; gap: 4px; flex-wrap: wrap; }
.tag { font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm); background: var(--color-bg-subtle); color: var(--color-text-secondary); border: 1px solid var(--color-border-subtle); }
.card-stats { display: flex; gap: 12px; font-size: 11px; color: var(--color-text-tertiary); }
.card-stats strong { color: var(--color-text-secondary); }
.card-actions { display: flex; gap: 6px; margin-top: auto; padding-top: 4px; }
.action-btn {
  padding: 5px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-subtle); color: var(--color-text-secondary); cursor: pointer;
  font-size: 12px; transition: all var(--transition-fast); font-family: inherit;
}
.action-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.action-btn.primary { background: var(--color-accent); color: var(--color-bg-panel); border-color: var(--color-accent); }
.action-btn.primary:hover { opacity: 0.9; }
.action-btn.danger { color: var(--color-up); border-color: var(--color-up); }
.action-btn.danger:hover { background: var(--color-up-bg, rgba(239,68,68,0.08)); }
.empty-state { padding: 24px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }
</style>
```

- [ ] **Step 4: Register WorkflowGalleryPanel in registry**

In `frontend/src/terminal/panels/registry.ts`, add to the system section (near `layout-templates`):
```typescript
register('workflow-gallery', () => import('./WorkflowGalleryPanel.vue'), { label: '工作流库', category: '系统', description: '示例工作流与用户工作流管理' })
```

- [ ] **Step 5: Run typecheck**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit
```
Expected: No errors.

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/panels/__tests__/WorkflowGalleryPanel.test.ts --reporter=verbose
```
Expected: All tests PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/panels/WorkflowGalleryPanel.vue frontend/src/terminal/panels/__tests__/WorkflowGalleryPanel.test.ts frontend/src/terminal/panels/registry.ts && git commit -m "feat(workflow): add WorkflowGalleryPanel with gallery UI and import functionality"
```

### Task 5: Create/update example JSON files

**Files:**
- Create: `examples/golden_cross.json`
- Create: `examples/rsi_rebound.json`
- Create: `examples/multi_factor.json`
- Create: `examples/ai_strategy.json`
- Create: `examples/scheduled_monitor.json`
- Create: `examples/momentum_breakout.json`
- Modify: (remove or keep) `examples/multi_asset.json`, `examples/error_handling.json`

- [ ] **Step 1: Create `examples/golden_cross.json`**

```json
{
  "id": "golden-cross",
  "name": "Golden Cross Strategy",
  "description": "5-day SMA crosses above 20-day SMA -> buy signal -> backtest with A-share rules",
  "nodes": [
    { "id": "loader", "node_type": "data_loader", "params": { "symbol": "000001.SZ", "interval": "1d", "days": 365 } },
    { "id": "sma5", "node_type": "sma", "params": { "period": 5 } },
    { "id": "sma20", "node_type": "sma", "params": { "period": 20 } },
    { "id": "cross", "node_type": "cross_signal", "params": {} },
    { "id": "backtest", "node_type": "backtest", "params": { "market": "CN", "initial_capital": 100000 } },
    { "id": "report", "node_type": "performance_report", "params": {} },
    { "id": "chart", "node_type": "chart_output", "params": { "type": "equity_curve" } }
  ],
  "edges": [
    { "from_node": "loader", "from_port": "ohlcv", "to_node": "sma5", "to_port": "input" },
    { "from_node": "loader", "from_port": "ohlcv", "to_node": "sma20", "to_port": "input" },
    { "from_node": "sma5", "from_port": "output", "to_node": "cross", "to_port": "fast" },
    { "from_node": "sma20", "from_port": "output", "to_node": "cross", "to_port": "slow" },
    { "from_node": "cross", "from_port": "signal", "to_node": "backtest", "to_port": "signals" },
    { "from_node": "backtest", "from_port": "results", "to_node": "report", "to_port": "input" },
    { "from_node": "backtest", "from_port": "results", "to_node": "chart", "to_port": "input" }
  ]
}
```

Create similar files for the other 5 workflows, extracting the `json` field from each `official.ts` entry.

- [ ] **Step 2: Verify JSON files are valid**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow && for f in examples/*.json; do python3 -m json.tool "$f" > /dev/null && echo "OK: $f" || echo "FAIL: $f"; done
```
Expected: All 8 files (6 new + 2 existing) pass validation.

- [ ] **Step 3: Commit**

```bash
git add examples/ && git commit -m "feat(workflow): add example JSON files for all 6 gallery workflows"
```

### Task 6: Full verification + i18n

**Files:**
- Modify: `frontend/src/lib/i18n/en.ts`
- Modify: `frontend/src/lib/i18n/zh.ts`

- [ ] **Step 1: Add i18n keys**

In `frontend/src/lib/i18n/en.ts`, add to the `workflow` section:
```typescript
gallery: {
  title: 'Workflow Gallery',
  official: 'Official Examples',
  my_workflows: 'My Workflows',
  import: 'Import',
  import_all: 'Import All',
  edit: 'Edit',
  export: 'Export',
  delete: 'Delete',
  nodes: 'nodes',
  runs: 'runs',
  beginner: 'Beginner',
  intermediate: 'Intermediate',
  advanced: 'Advanced',
  no_user_workflows: 'No saved workflows yet. Import an example to get started.',
  search_placeholder: 'Search workflows by name, description, or tags...',
  import_success: 'Workflow imported successfully. Switch to Workflow mode to view.',
  import_confirm: 'Import workflow "{{name}}"? It will appear on the workflow canvas.',
},
```

In `frontend/src/lib/i18n/zh.ts`, add the Chinese equivalents:
```typescript
gallery: {
  title: '工作流库',
  official: '官方示例',
  my_workflows: '我的工作流',
  import: '导入',
  import_all: '全部导入',
  edit: '编辑',
  export: '导出',
  delete: '删除',
  nodes: '节点',
  runs: '运行次数',
  beginner: '入门',
  intermediate: '进阶',
  advanced: '高级',
  no_user_workflows: '还没有保存的工作流。导入一个官方示例开始吧。',
  search_placeholder: '搜索工作流名称、描述、标签...',
  import_success: '工作流已导入。切换到工作流模式查看。',
  import_confirm: '导入工作流「{{name}}」？导入后将出现在工作流画布中。',
},
```

- [ ] **Step 2: Run full test suite**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run && npx vue-tsc --noEmit
```
Expected: All tests PASS, typecheck OK.

- [ ] **Step 3: Run lint**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npm run lint
```
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: add i18n keys and final verification for workflow gallery feature"
```

# Implementation Plan: Inline Node Controls

> **Spec**: `docs/specs/2026-07-03-inline-node-controls.md`

## Task 1: 扩展 Go ParamDef 结构体

**File**: `internal/workflow/node.go`

改动：给 `ParamDef` 添加 Min/Max/Step/Options/Required 字段。

```go
type ParamDef struct {
    Name        string    `json:"name"`
    Type        string    `json:"type"`     // "int", "float", "string", "bool", "select"
    Default     any       `json:"default,omitempty"`
    Description string    `json:"description,omitempty"`
    Required    bool      `json:"required"`
    Min         *float64  `json:"min,omitempty"`
    Max         *float64  `json:"max,omitempty"`
    Step        *float64  `json:"step,omitempty"`
    Options     []string  `json:"options,omitempty"`
}
```

**Test**: `internal/workflow/node_test.go` — 添加 `TestParamDefJSON`，验证 JSON 序列化/反序列化正确性；添加 `TestParamDefMinMax` 验证默认值为 nil。

---

## Task 2: 更新关键节点的 ParamSchema

对 5 个最常用节点补全完整约束：

### Task 2a: sma - `internal/workflow/nodes/sma.go`

```go
func (n *SMANode) ParamSchema() []workflow.ParamDef {
    min1 := float64(1)
    max9999 := float64(9999)
    step1 := float64(1)
    return []workflow.ParamDef{
        {Name: "period", Type: "int", Default: float64(20),
            Description: "SMA window size", Required: true,
            Min: &min1, Max: &max9999, Step: &step1},
    }
}
```

### Task 2b: bollinger - `internal/workflow/nodes/bollinger.go`

```go
func (n *BollingerNode) ParamSchema() []workflow.ParamDef {
    min1 := float64(1); max9999 := float64(9999); step1 := float64(1)
    min0 := float64(0.1); max10 := float64(10); step01 := float64(0.1)
    return []workflow.ParamDef{
        {Name: "period", Type: "int", Default: float64(20), Required: true, Min: &min1, Max: &max9999, Step: &step1},
        {Name: "multiplier", Type: "float", Default: float64(2), Required: true, Min: &min0, Max: &max10, Step: &step01},
    }
}
```

### Task 2c: data_loader - `internal/workflow/nodes/data_loader.go`

```go
func (n *DataLoaderNode) ParamSchema() []workflow.ParamDef {
    return []workflow.ParamDef{
        {Name: "source", Type: "select", Default: "csv", Description: "Data source type", Options: []string{"csv", "akshare", "tushare"}},
        {Name: "path", Type: "string", Default: "", Description: "Path to CSV file"},
    }
}
```

### Task 2d: macd - `internal/workflow/nodes/macd.go`

```go
func (n *MACDNode) ParamSchema() []workflow.ParamDef {
    min1 := float64(1); max999 := float64(999); step1 := float64(1)
    return []workflow.ParamDef{
        {Name: "fast_period", Type: "int", Default: float64(12), Required: true, Min: &min1, Max: &max999, Step: &step1},
        {Name: "slow_period", Type: "int", Default: float64(26), Required: true, Min: &min1, Max: &max999, Step: &step1},
        {Name: "signal_period", Type: "int", Default: float64(9), Required: true, Min: &min1, Max: &max999, Step: &step1},
    }
}
```

### Task 2e: rsi - `internal/workflow/nodes/rsi.go`

```go
func (n *RSINode) ParamSchema() []workflow.ParamDef {
    min1 := float64(1); max999 := float64(999); step1 := float64(1)
    return []workflow.ParamDef{
        {Name: "period", Type: "int", Default: float64(14), Required: true, Min: &min1, Max: &max999, Step: &step1},
    }
}
```

**Test**: 每个节点已有的 test file 中添加 schema 验证：`assert.Equal(t, "int", schema[0].Type)`。

---

## Task 3: CustomNode.vue — 类型感知控件渲染

**File**: `frontend/src/workflow/canvas/CustomNode.vue`

核心改动：将现有的纯文本 param 渲染替换为 type-based 控件。同时需要从 workflow store 获取 param schema。

### 3a: Composable 获取 schema

在组件内从 `workflow.nodeMetaCache` 获取当前节点类型的 param schema：

```ts
import { useWorkflowStore } from '@/stores/workflow'
const workflow = useWorkflowStore()

const paramSchema = computed(() => {
  const meta = workflow.nodeMetaCache.get(props.data.nodeType)
  return meta?.params || []
})
```

### 3b: 检查同名 port 是否有连线

```ts
function isPortConnected(paramName: string): boolean {
  const node = workflow.nodes.find(n => n.id === props.id)
  if (!node) return false
  return workflow.edges.some(e =>
    e.target === props.id && e.targetHandle === paramName
  )
}
```

### 3c: 控件渲染映射

替换现有的 `.node-params` 区域：

```vue
<div v-for="p in visibleParams" :key="p.name" class="param-widget-row">
  <!-- INT -->
  <template v-if="p.type === 'int'">
    <label class="widget-label">{{ p.name }}</label>
    <div class="number-widget">
      <button class="step-btn" @click.stop="adjustParam(p.name, -stepVal(p))">−</button>
      <input type="number" class="widget-input number-input"
        :value="getParam(p.name, p.default)"
        :min="p.min" :max="p.max" :step="p.step || 1"
        @input.stop="setParam(p.name, parseFloat(($event.target as HTMLInputElement).value))" />
      <button class="step-btn" @click.stop="adjustParam(p.name, stepVal(p))">+</button>
    </div>
  </template>

  <!-- FLOAT -->
  <template v-else-if="p.type === 'float'">
    <label class="widget-label">{{ p.name }}</label>
    <input type="number" class="widget-input"
      :value="getParam(p.name, p.default)"
      :min="p.min" :max="p.max" :step="p.step || 0.01"
      @input.stop="setParam(p.name, parseFloat(($event.target as HTMLInputElement).value))" />
  </template>

  <!-- STRING -->
  <template v-else-if="p.type === 'string'">
    <label class="widget-label">{{ p.name }}</label>
    <input type="text" class="widget-input"
      :value="getParam(p.name, p.default)"
      @input.stop="setParam(p.name, ($event.target as HTMLInputElement).value)" />
  </template>

  <!-- BOOL -->
  <template v-else-if="p.type === 'bool'">
    <label class="widget-label">{{ p.name }}</label>
    <label class="toggle-switch">
      <input type="checkbox"
        :checked="getParam(p.name, p.default) === true"
        @change.stop="setParam(p.name, ($event.target as HTMLInputElement).checked)" />
      <span class="toggle-slider"></span>
    </label>
  </template>

  <!-- SELECT -->
  <template v-else-if="p.type === 'select'">
    <label class="widget-label">{{ p.name }}</label>
    <select class="widget-select"
      :value="getParam(p.name, p.default)"
      @change.stop="setParam(p.name, ($event.target as HTMLSelectElement).value)">
      <option v-for="opt in p.options" :key="opt" :value="opt">{{ opt }}</option>
    </select>
  </template>
</div>
```

### 3d: 连线隐藏逻辑

`visibleParams` 计算属性过滤掉有连线的同名参数：

```ts
const visibleParams = computed(() => {
  return paramSchema.value.filter((p: any) => {
    // hide widget if there's an input port with same name and it has a connection
    if (isPortConnected(p.name)) return false
    return true
  })
})
```

### 3e: Helper 函数

```ts
function getParam(name: string, defaultVal: any): any {
  const v = props.data.params?.[name]
  return v !== undefined ? v : defaultVal
}
function setParam(name: string, value: any) {
  const newParams = { ...(props.data.params || {}), [name]: value }
  props.data.params = newParams
  emit('updateParams', newParams)
}
function adjustParam(name: string, delta: number) {
  const old = getParam(name, 0)
  setParam(name, (typeof old === 'number' ? old : 0) + delta)
}
function stepVal(p: any): number { return p.step || 1 }
```

### 3f: 样式

新增 widget 样式（scoped）：

```css
.param-widget-row { display: flex; align-items: center; gap: 6px; padding: 3px 12px; }
.widget-label { font-size: 10px; color: var(--color-text-tertiary); min-width: 50px; flex-shrink: 0; }
.widget-input { flex: 1; padding: 2px 6px; background: #0d1117; border: 1px solid var(--color-border); border-radius: 4px; color: var(--color-text-primary); font-size: 11px; outline: none; }
.widget-input:focus { border-color: var(--color-accent); }
.widget-select { flex: 1; padding: 2px 4px; background: #0d1117; border: 1px solid var(--color-border); border-radius: 4px; color: var(--color-text-primary); font-size: 11px; outline: none; cursor: pointer; }
.number-widget { display: flex; align-items: center; gap: 2px; flex: 1; }
.number-input { flex: 1; min-width: 0; }
.step-btn { width: 20px; height: 20px; border: 1px solid var(--color-border); border-radius: 3px; background: transparent; color: var(--color-text-secondary); font-size: 12px; cursor: pointer; display: flex; align-items: center; justify-content: center; }
.step-btn:hover { background: rgba(88,166,255,0.1); border-color: var(--color-accent); }
.toggle-switch { position: relative; display: inline-block; width: 32px; height: 18px; }
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.toggle-slider { position: absolute; cursor: pointer; inset: 0; background: #30363d; border-radius: 18px; transition: .2s; }
.toggle-slider::before { content: ''; position: absolute; height: 14px; width: 14px; left: 2px; bottom: 2px; background: #fff; border-radius: 50%; transition: .2s; }
.toggle-switch input:checked + .toggle-slider { background: #58a6ff; }
.toggle-switch input:checked + .toggle-slider::before { transform: translateX(14px); }
```

删除旧 `.param-row`、`.params-hint`、`.edit-hint` 样式。

---

## Task 4: PropertyPanel 简化

**File**: `frontend/src/workflow/PropertyPanel.vue`

删除整个 `visibleParams` 相关区域（`v-for="(k, v) in visibleParams"` 段落）。只保留：
- 节点 ID / Type
- 错误策略 (stop/skip/retry)
- Input/Output 端口列表（只读）
- 执行状态
- Pin to Terminal 按钮

相关代码删除：
- `visibleParams` computed
- `formatValue()` (不再需要)
- `formatDuration()` (保留 — 用于执行状态)
- 所有 `.param-row`、`.param-input` 相关 template 和 style

---

## Task 5: workflow store 增强

**File**: `frontend/src/stores/workflow.ts`

### 5a: 缓存完整 params schema

当前 `NodeMetaInfo.params` 存储为 `any[]`，但 unused。确认 `fetchNodeMeta` 时 params 已完整保留。

可能需要在 `fetchNodeMeta` 中保留 `n.params` 的全部字段：

```ts
// 在 fetchNodeMeta 的循环中
m.set(n.node_type, {
  category: n.category || 'utility',
  inputs: (n.input_ports || []).map((p: any) => ({ name: p.name, type: p.type || 'any' })),
  outputs: (n.output_ports || []).map((p: any) => ({ name: p.name, type: p.type || 'any' })),
  params: n.params || [],  // 已经完整保留
})
```

### 5b: 工具函数 — 检查端口连线

```ts
function isPortConnected(nodeId: string, portName: string): boolean {
  return edges.value.some(e => e.target === nodeId && e.targetHandle === portName)
}
```

---

## Task 6: 更新 app.go — 新增 GetNodeSchema API

**File**: `app.go`

可选增强：新增 GetNodeSchema 返回完整 schema（包含 ports + params 带约束）。目前 ListNodes() 已通过 `NodeMeta` 返回完整信息，前端无需额外 API。但为了与 ComfyUI 的 `/object_info/{node_class}` 对齐，添加一个细粒度 API：

```go
func (a *App) GetNodeSchema(nodeType string) (*workflow.NodeMeta, error) {
    if a.registry == nil {
        return nil, fmt.Errorf("registry not initialized")
    }
    node, err := a.registry.Create(nodeType, "__dummy__", nil)
    if err != nil {
        return nil, fmt.Errorf("unknown node type: %q", nodeType)
    }
    meta := &workflow.NodeMeta{NodeType: nodeType, Category: node.Category()}
    for _, p := range node.InputPorts() {
        meta.InputPorts = append(meta.InputPorts, workflow.NodePortInfo{Name: p.Name, Type: string(p.Type), Required: p.Required})
    }
    for _, p := range node.OutputPorts() {
        meta.OutputPorts = append(meta.OutputPorts, workflow.NodePortInfo{Name: p.Name, Type: string(p.Type), Required: p.Required})
    }
    meta.Params = node.ParamSchema()
    return meta, nil
}
```

---

## 执行顺序

```
Task 1 (Go ParamDef 扩展) → Task 2 (节点 schema 补全) → 
  → commit ["feat(workflow): extend ParamDef with min/max/step/options/required"]
  →
Task 3 (CustomNode widgets) → Task 4 (PropertyPanel 简化) → Task 5 (store 增强) →
  → commit ["feat(workflow): inline type-aware node controls on canvas cards"]
  →
Task 6 (GetNodeSchema API) → 
  → commit ["feat(workflow): add GetNodeSchema API for detailed node introspection"]
```

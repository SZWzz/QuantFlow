# Workflow Mode Bugfixes

> 父 spec: 无（独立专项）
> 前序: 评审报告 `docs/review/review-report-2026-06-26.md`
> 日期: 2026-06-26

## Motivation

工作流模式评审发现 **4 个 P0（阻塞级）** 和 **5 个 P1（功能缺陷）** 问题：

**P0:**
1. Go 返回 `"success"`，前端认 `"completed"` — 状态一直显示 Not Done
2. `fromWorkflowJSON` 边重连按 `node_type` 匹配，同类型多节点时边全连错
3. 缓存键 `%v` 不保证 map 顺序，2+ 输入节点缓存命中率 ~50%
4. `nodeInstance == nil` 时 `nr.Status` 未设置，残留零值

**P1:**
5. 前端端口映射只写死 5 种类型，~80 种节点回退到通用 `['input']/['output']`
6. Pin to Terminal 符号写死 `'AAPL'`
7. `NodePalette` 直接用 `(window as any)` 而非 `wails.ts` 类型化封装
8. 执行结果不持久化到 `execution_history`
9. 调度器 `WorkflowExecutor` 接口定义了但从未实现

## Design

### 1. 状态字符串对齐

**改动**: 将 `engine.go` 的 `"success"` 改为 `"completed"`，保持前后端一致。

```go
// engine.go
result.Status = "completed"   // was "success"
nr.Status   = "completed"     // was "success"
```

前端的 `NodeExecStatus.status` 类型和 `ExecutionLog.vue` 中的 `statusIcon`/`statusColor` 保持原样（已经支持 `'success'`）。只需修改 Go 端常量。

### 2. `fromWorkflowJSON` 边重连修复

**问题**: 第 158-163 行调用 `addNode` 生成新 ID `{type}-{Date.now()}`，然后 167-189 行通过 `node_type` 找节点匹配边，同类型时 `find()` 总返回第一个。

**修复**: 在 `wf.nodes` 和创建的 `nodes.value` 之间建立 ID 映射表，按原始 `from_node`/`to_node` 匹配。

```typescript
// 在 fromWorkflowJSON 中用 ID map 而非类型查找
const nodeIdMap = new Map<string, string>() // oldID → newID
for (const n of wf.nodes) {
  const newId = addNode(n.node_type, ..., n.params)
  nodeIdMap.set(n.id, newId)
}
for (const e of wf.edges) {
  const sourceId = nodeIdMap.get(e.from_node)
  const targetId = nodeIdMap.get(e.to_node)
  if (sourceId && targetId) { ... }
}
```

### 3. 缓存键确定性

**改动**: 在 `CacheKey` 中手动排序 map 键后再序列化。

```go
func CacheKey(nodeID string, inputs map[string]any) string {
    keys := make([]string, 0, len(inputs))
    for k := range inputs { keys = append(keys, k) }
    sort.Strings(keys)
    var b strings.Builder
    b.WriteString(nodeID)
    for _, k := range keys {
        fmt.Fprintf(&b, "|%s:%v", k, inputs[k])
    }
    hash := sha256.Sum256([]byte(b.String()))
    return fmt.Sprintf("%x", hash[:16])
}
```

### 4. `nr.Status` 缺失赋值

**改动**: `engine.go:112-113` 在 `nodeInstance == nil` 的 return 前补上 `nr.Status = "failed"`。

### 5. 前端端口映射动态化

**改动**: 在 `WorkflowCanvas.vue onDrop` 中，`addNode` 后调用 Go 端的 `GetNodePorts(nodeType)` 获取端口定义，替代本地 `portMap`。

**新增 Go API**: `GetNodePorts(nodeType string) → {inputs: PortDef[], outputs: PortDef[]}`

**Fallback**: API 不可用时回退到当前逻辑。

### 6. Pin to Terminal 符号修复

**改动**: `PropertyPanel.vue:53` 和 `WorkflowMode.vue:104` 中，从 `node.data.params.symbol` 读取符号，而不是写死 `'AAPL'`。

```typescript
terminal.openPanel(panelId, { 
    symbol: node.data.params?.symbol || '600519'  // fallback 改为通用默认值
})
```

### 7. NodePalette 使用类型化封装

**改动**: `NodePalette.vue:13` 的 `(window as any).go.main.App.ListNodes()` 改为从 `@/lib/wails` 导入 `ListNodes`。

### 8. 执行结果持久化

**改动**: `WorkflowMode.vue` 的 `onRun()` 中，执行完成后调用 `SaveExecution`（新增 Go API）。

**新增 Go API**: `SaveExecution(workflowID, status, resultJSON)` — 复用已有的 `WorkflowRepo.SaveExecution`。

### 9. 调度器接入

**改动**: 实现 `WorkflowExecutor` 接口，将 `schedule.New()` 的 nil 参数替换为真实实现。

### 文件变更清单

| 文件 | 改动 |
|------|------|
| `internal/workflow/engine.go` | `"success"` → `"completed"`；补 `nr.Status = "failed"` |
| `internal/workflow/cache.go` | `CacheKey` 排序 map 键 |
| `internal/workflow/nodes/schedule.go` | 接入真实 `WorkflowExecutor` |
| `internal/schedule/types.go` | 确认接口定义，新增实现 |
| `app.go` | 新增 `GetNodePorts`、`SaveExecution` 导出方法 |
| `frontend/src/stores/workflow.ts` | `fromWorkflowJSON` 用 ID map 重连；`addNode` 动态端口 |
| `frontend/src/workflow/canvas/WorkflowCanvas.vue` | `onDrop` 调用 `GetNodePorts` |
| `frontend/src/workflow/WorkflowMode.vue` | Pin 符号修复；持久化调用 |
| `frontend/src/workflow/PropertyPanel.vue` | Pin 符号修复 |
| `frontend/src/workflow/NodePalette.vue` | 使用 `ListNodes` 类型化封装 |
| `frontend/src/lib/wails.ts` | 新增 `GetNodePorts`、`SaveExecution` |

### 数据流

```
执行前: NodePalette 调用 ListNodes() → 类型化 wails.ts 封装
执行中: onDrop → addNode → GetNodePorts(type) → 动态端口
执行后: RunWorkflow → SaveExecution(workflowID, status, resultJSON)
缓存时: CacheKey 排序后 SHA256 → 确定性命中
加载时: fromWorkflowJSON → ID map → 准确边重连
```

## Acceptance Criteria

- [ ] Go 状态字符串改为 `"completed"`，ExecutionLog "Done" 标签可见
- [ ] 加载含 2 个同类型节点的工作流，边正确重连
- [ ] 同一节点 2+ 输入时缓存命中率 ≈ 100%
- [ ] `nodeInstance == nil` 时 `nr.Status` 为 `"failed"` 而非空
- [ ] 动态端口：添加未在 `portMap` 中的节点类型时端口正确
- [ ] Pin to Terminal 使用节点配置的 symbol
- [ ] NodePalette 通过 `@/lib/wails` 调用，无 `(window as any)`
- [ ] 执行完成后 `execution_history` 表有记录
- [ ] 所有 Go 测试通过，前端 `vue-tsc --noEmit` 通过

## Risks / Trade-offs

- **动态端口增加一次 RTT**: `onDrop` → `GetNodePorts` 增加一次 IPC 调用，但仅在添加节点时触发，非频繁操作，可接受
- **后端状态字修改影响集成测试**: 需要同步修改 `integration_test.go` 和 `engine_test.go` 中的 `"success"` 断言
- **`SaveExecution` 新增导出让前端多一次调用**: 非阻塞、可失败（不影响主流程）

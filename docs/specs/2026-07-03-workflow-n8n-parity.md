# Workflow 对标 n8n — 能力提升路线图

## Motivation

当前工作流引擎在节点种类（85个）、引擎性能（Kahn DAG 并行）、回测优化（walk-forward + grid search）方面已达专业级。但与 n8n 等成熟工作流平台相比，在**调试体验、错误处理、数据流可视化、模块化复用**方面存在明显差距。本 spec 基于 n8n 对标分析，制定 10 项优化方案。

## 对比基线

| 维度 | QuantFlow 当前 | n8n | 差距 |
|------|---------------|-----|------|
| 节点数 | 85 个（18 类） | 400+ | 专用性不输，覆盖面需扩 |
| 调试 | 全量执行，无断点 | Pin 数据、单步、断点 | 🔴 核心差距 |
| 执行历史 | 无持久化 | 完整执行记录+回放 | 🔴 |
| 子工作流 | stub（空壳） | 完整嵌套调用 | 🔴 |
| 错误处理 | fail-fast | per-node retry/skip/stop | 🟡 |
| 数据流可视化 | 静态状态色 | 动画高亮路径 | 🟡 |
| 表达式 | math_op + json_parse | 全节点引用 `$node.xxx` | 🟡 |
| 画布辅助 | 基础 drag-drop | 注释/便签/快捷键提示 | 🟡 |
| 凭证管理 | 散落各处 | 集中加密管理 | 🟡 |
| 触发器 | schedule + 无 webhook | cron / webhook / polling | 🟡 |

## Design

### 数据流概览

```
┌─────────────────────────────────────────────────────────────────────┐
│ Workflow Canvas (Vue Flow)                                          │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │
│ │ Node 1   │  │ Node 2   │  │ Node 3   │  │ Comments / Stickies   │ │
│ │ [params] │──│ [params] │──│ [params] │  │ "策略说明: 金叉买入"    │ │
│ │ ⚡=enter │  │ ⚡=run   │  │ ⚡=run   │  └──────────────────────┘ │
│ └──────────┘  └──────────┘  └──────────┘                          │
│       ▲              │              │                               │
│       └── colored ───┘              │  ← execution animation       │
│          edge flash                  │                               │
├─────────────────────────────────────────────────────────────────────┤
│ ExecutionLog: [1] data_loader ✓ 12ms  [2] sma ✓ 3ms  [3] ❌  "EOF" │
├─────────────────────────────────────────────────────────────────────┤
│ Execution History (SQLite): run-20260703-001 → 回放  run-001 → 对比 │
└─────────────────────────────────────────────────────────────────────┘
```

### 1. 执行历史持久化（Execution History）

**目标**：每次运行完整记录到 SQLite，支持回溯查看和结果对比。

**数据流**：
```
WorkflowEngine.Execute()
  ↓ 完成后
execution_records 表 INSERT
  ├── run_id: "run-20260703-001"
  ├── workflow_json: {...}      ← 当时的 workflow 快照
  ├── status: "completed"|"failed"
  ├── node_results: [{id:, output:, duration:, error:}, ...]
  ├── started_at / finished_at
  └── triggered_by: "manual"|"schedule"|"webhook"
```

**新文件/修改**：
- `internal/storage/execution_repo.go` — 新建，CRUD for execution records
- `internal/storage/migrations.go` — 加 `execution_records` 表
- `app.go` — `GetExecutionHistory(limit int)`, `GetExecution(runID string)`, `ReplayExecution(runID string)`
- `frontend/src/workflow/ExecutionHistory.vue` — 新建，历史列表+详情面板
- `frontend/src/stores/workflow.ts` — 加 `executionHistory`, `loadHistory()`, `replayWorkflow()`

**前端 API**：
```typescript
// 获取历史列表
const history = await app.GetExecutionHistory(20)
// 查看单次执行详情
const run = await app.GetExecution("run-20260703-001")
// 回放：加载 workflow + 展示结果
await app.ReplayExecution("run-20260703-001")
```

**Acceptance Criteria**：
- [ ] 每次运行后自动保存执行记录到 SQLite
- [ ] 历史列表展示 run_id、状态、耗时、触发方式、时间
- [ ] 点击某条记录可查看每个节点的输入/输出/耗时/错误
- [ ] "回放"按钮：加载该次运行的 workflow JSON 到画布，标注各节点结果
- [ ] 失败运行可查看完整错误栈

### 2. 子工作流实现（Sub-workflow）

**目标**：`sub_workflow` 节点真正执行嵌套 workflow，支持「策略复用」。

**数据流**：
```
Canvas: 主 Workflow
  ├── data_loader
  ├── sub_workflow (id: "factor_pipeline")  ← 引用已保存的子 workflow
  │     ├── input: ohlcv data
  │     └── output: factor scores
  └── rank_select

子 Workflow "factor_pipeline"
  ├── pct_change  →  scale
  ├── std_dev     →  scale
  └── merge → rank
```

**修改**：
- `internal/workflow/nodes/sub_workflow.go` — 实现 Execute：加载子 workflow JSON → 注入 inputs → Engine.Execute(subWf) → 返回 outputs
- `frontend/src/workflow/canvas/` — sub_workflow 节点双击展开/折叠子画布视图（可选项）
- `frontend/src/stores/workflow.ts` — 子 workflow 下拉选择（从已保存列表）

**Acceptance Criteria**：
- [ ] 子 workflow 节点可下拉选择「已保存的 workflow」
- [ ] 执行时真正调用嵌套 workflow engine
- [ ] 子 workflow 的输入/输出端口正确映射
- [ ] 执行日志中显示嵌套层级（缩进）
- [ ] 错误在子 workflow 中发生时，父级正确捕获

### 3. 节点级错误处理（Per-Node Error Strategy）

**目标**：每个节点可配置 `onError` 策略，替代全局 fail-fast。

**新增参数**（所有节点）：
```go
type ErrorStrategy string
const (
    ErrorStop  ErrorStrategy = "stop"   // 停止（默认，当前行为）
    ErrorSkip  ErrorStrategy = "skip"   // 跳过，输出空值，继续执行
    ErrorRetry ErrorStrategy = "retry"  // 重试 N 次
)
```

**修改**：
- `internal/workflow/node.go` — `BaseNode` 接口加 `ErrorStrategy() ErrorStrategy` 和 `RetryCount() int`
- `internal/workflow/engine.go` — `executeNode` 中：捕获 error → 检查 strategy → skip/retry
- `frontend/src/workflow/PropertyPanel.vue` — 加错误策略下拉菜单

**Acceptance Criteria**：
- [ ] 每个节点属性面板有「错误处理: stop / skip / retry」选择
- [ ] skip: 错误时输出空值，下游节点继续执行
- [ ] retry: 失败后自动重试 N 次，间隔递增
- [ ] 执行日志显示 skip/retry 事件

### 4. 数据流执行动画（Execution Animation）

**目标**：运行时高亮正在执行的节点 + 连线流动动画，直观展示数据流向。

**实现**：
- 执行前：所有节点 `status = 'idle'`
- 执行中：当前层节点 `status = 'running'`（橙色闪烁），已完成节点 `status = 'success'`（绿色）
- Edge 动画：已完成节点 → 下游节点的 edge 加 `animated: true` + 颜色渐变
- 所有节点完成后：全绿边框

**修改**：
- `frontend/src/stores/workflow.ts` — `updateNodeStatus(nodeId, status)`, `animateEdge(edgeId)`
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — 监听 `nodeStatuses` 变化，动态更新 edge style
- `app.go` — `RunWorkflow` 改为流式返回（WebSocket / SSE），逐层推送状态

**Acceptance Criteria**：
- [ ] 执行时当前层节点橙色脉冲动画
- [ ] 已完成节点→下游节点的连线流动绿色动画
- [ ] 失败节点红色边框，错误信息显示在卡片上
- [ ] 执行完成后所有正常节点恢复默认样式

### 5. Pin 数据 / 部分执行（Pin & Partial Run）

**目标**：固定上游节点的输出值，只运行选中节点，大幅提升调试效率。

**实现**：
- 右键节点 → "Pin Output" → 弹出输入框填写 mock 数据 → 节点标记为 📌
- Pin 后，引擎跳过该节点及所有上游节点
- "Run Selected Node"：只执行选中节点 + 下游
- "Run to Here"：执行到选中节点停止

**修改**：
- `internal/workflow/engine.go` — `ExecutePartial(ctx, wf, pinnedOutputs, targetNodes)`
- `frontend/src/workflow/canvas/WorkflowCanvas.vue` — 右键菜单
- `frontend/src/stores/workflow.ts` — `pinnedNodes: Set<string>`, `pinData: Map<string, any>`

**Acceptance Criteria**：
- [ ] 右键节点 → "Pin Output" 输入 mock 数据
- [ ] Pin 节点显示 📌 图标
- [ ] 运行只执行未 Pin 的节点（复用缓存）
- [ ] "Run Selected" / "Run to Here" 菜单项

### 6. 表达式引擎（Expression System）

**目标**：节点参数支持 `{{ $node.data_loader_1.data.close }}` 引用上游输出。

**实现**：
- 解析 `{{ $node.<node_id>.<output_port>.<field> }}` 语法
- 执行前替换：遍历所有节点参数，展开表达式引用
- 支持简单表达式：`{{ $math_op_1.result + 0.01 }}`

**修改**：
- `internal/workflow/expression.go` — 新建，解析器 + 求值器
- `internal/workflow/engine.go` — `resolveExpressions(wf, outputs)` before executing each layer
- `frontend/src/workflow/PropertyPanel.vue` — 输入框提示 "可用 {{ $... }}" + 自动补全

**Acceptance Criteria**：
- [ ] 参数中 `{{ $node_id.port.field }}` 正确解析为上游输出
- [ ] 支持简单算术 `{{ $a.result + $b.result }}`
- [ ] 属性面板中有自动补全提示
- [ ] 解析失败时有清晰错误提示

### 7. Webhook 触发器（Webhook Trigger）

**目标**：外部系统通过 HTTP POST 触发 workflow 执行（策略信号 → 自动下单）。

**实现**：
```go
// 启动内置 HTTP server
POST /api/webhook/:workflowId
Body: {"inputs": {...}}
Response: {"run_id": "...", "status": "started"}
```

**修改**：
- `internal/workflow/webhook.go` — 新建，HTTP handler，路由注册
- `app.go` — 启动时注册 webhook 路由
- `internal/workflow/nodes/webhook_trigger.go` — 新建节点类型
- `frontend/src/workflow/NodePalette.vue` — 加 webhook_trigger 节点

**Acceptance Criteria**：
- [ ] 工作流包含 webhook_trigger 节点时可生成唯一 URL
- [ ] POST 到该 URL 触发 workflow 执行
- [ ] 请求体中的 inputs 注入到 trigger 节点的输出
- [ ] 返回 run_id 供轮询结果

### 8. 画布注释 & 便签（Comments & Sticky Notes）

**目标**：在画布上添加策略说明、备注、分组标注。

**实现**：
- 新增两条工具栏按钮：「添加注释」「添加便签」
- 注释：文本框，可拖拽、调整大小、修改颜色
- 便签：彩色便签纸，支持 Markdown
- 数据保存为 workflow JSON 的 `annotations` 字段

**修改**：
- `frontend/src/workflow/canvas/CanvasAnnotation.vue` — 新建，注释组件
- `frontend/src/workflow/canvas/StickyNote.vue` — 新建，便签组件
- `frontend/src/stores/workflow.ts` — 扩展 WorkflowJSON 加 `annotations` 字段
- `internal/workflow/dag.go` — Workflow 结构体加 `Annotations []Annotation`

**Acceptance Criteria**：
- [ ] 工具栏有「+ 注释」「+ 便签」按钮
- [ ] 注释/便签可拖拽移动、调整大小
- [ ] 便签支持 4 种颜色
- [ ] 保存/加载 workflow 时保留注释/便签
- [ ] 缩放时注释/便签跟随画布缩放

### 9. 节点搜索增强（Node Search Enhancement）

**目标**：Ctrl+K 搜索面板支持实时搜索、最近使用、收藏节点。

**实现**：
- 搜索面板改进：实时过滤（不需要回车）、分类折叠、键盘导航
- 最近使用：localStorage 存最近 10 个使用的节点类型
- 收藏：右键节点 → "收藏"，下次优先展示

**修改**：
- `frontend/src/workflow/NodePalette.vue` — UI 增强
- `frontend/src/stores/workflow.ts` — `recentNodes: string[]`, `favoriteNodes: Set<string>`

**Acceptance Criteria**：
- [ ] 搜索框实时过滤，不需要回车
- [ ] 最近使用 10 个节点显示在顶部
- [ ] 右键收藏节点在列表中高亮
- [ ] 键盘 ↑↓ 导航 + Enter 选择

### 10. 凭证/密钥管理（Credential Manager）

**目标**：集中管理 broker/data API key，加密存储，节点引用凭证而不是硬编码。

**数据流**：
```
Settings → Credential Manager
  ├── credential_id: "binance_main"
  ├── type: "api_key"
  ├── keys: { api_key: "encrypted_xxx", secret: "encrypted_yyy" }
  └── used_by: [place_order_1, position_query_2]

Node Config: "credential" → dropdown → "binance_main"
```

**修改**：
- `internal/auth/credential.go` — 新建，加密存储（AES-256-GCM）+ CRUD
- `internal/storage/migrations.go` — `credentials` 表
- `app.go` — `ListCredentials()`, `CreateCredential()`, `DeleteCredential()`
- `frontend/src/workflow/PropertyPanel.vue` — 凭证下拉选择器

**Acceptance Criteria**：
- [ ] 系统设置中有「凭证管理」面板
- [ ] 支持添加/删除/编辑凭证（名称 + key-value pairs）
- [ ] 凭证加密存储到 SQLite
- [ ] 节点参数中的 API key 字段改为凭证下拉选择
- [ ] 导出 workflow 时不包含凭证内容

## Risks / Trade-offs

| 风险 | 缓解 |
|------|------|
| WebSocket/SSE 流式执行增加复杂度 | 先用轮询模式，后续升级 |
| 子工作流递归深度失控 | 限制最大嵌套 3 层 |
| 表达式引擎性能 | 编译缓存，每个 layer 只解析一次 |
| Pin 数据与缓存冲突 | Pin 状态下跳过缓存，直接用 mock 数据 |
| 凭证加密密钥管理 | 用机器指纹 + 用户主密码派生密钥 |

## Implementation Phases

| Phase | 内容 | 预估 |
|-------|------|------|
| P1 (Week 1) | 执行历史 + 错误处理策略 | 3 天 |
| P2 (Week 2) | 子工作流 + 执行动画 | 3 天 |
| P3 (Week 3) | Pin 数据 + 表达式引擎 | 4 天 |
| P4 (Week 4) | Webhook + 画布注释 + 搜索增强 | 3 天 |
| P5 (Week 5) | 凭证管理 + Edge Cases | 2 天 |

# Plan: Workflow Mode P1-P2 Enhancements

P1（可用性）和 P2（体验）纯前端实现，P3（Walk-Forward + 参数优化）需 Go 后端另行计划。

## Phase 1: 端口类型系统 (P1.2)

**Task 1.1** — `internal/workflow/registry.go`: `NodeMeta` 新增 `InputPorts`/`OutputPorts` 字段。`ListAll` 改为创建临时节点实例读取真实端口。

**Task 1.2** — `frontend/src/stores/workflow.ts`: 删除 `pmap` 硬编码。`addNode` 改用 `ListNodes` 返回的端口定义。

**Task 1.3** — `frontend/src/workflow/canvas/WorkflowCanvas.vue`: `onConnect` 中比对端口类型。不兼容 → 阻止连接 + toast 提示。

## Phase 2: 预置模板 (P1.1)

**Task 2.1** — `frontend/src/workflow/templates.ts` (新): 7 个模板的 nodes+edges JSON。

**Task 2.2** — `frontend/src/workflow/NodePalette.vue`: 底部折叠区 "Templates"，点击插入画布。

## Phase 3: 节点视觉差异化 (P2.2)

**Task 3.1** — `frontend/src/workflow/canvas/CustomNode.vue`: 按 `node.data.category` 显示颜色边框 + emoji 图标。

## Phase 4: 多工作流管理 (P2.1)

**Task 4.1** — `frontend/src/stores/workflow.ts`: 新增 `saveWorkflow(name)`/`loadWorkflow(id)`/`deleteWorkflow(id)`。存储键升级为 `quantflow-workflows`。

**Task 4.21** — `frontend/src/workflow/WorkflowList.vue` (新): 侧边抽屉列出已保存工作流，支持新建/打开/重命名/删除。

**Task 4.3** — `frontend/src/workflow/WorkflowMode.vue`: 工具栏加"工作流列表"按钮，保存改为"保存为..."命名对话框。

## Phase 5: CHANGELOG

## Files

| File | Action |
|------|--------|
| `internal/workflow/registry.go` | NodeMeta + 端口字段 |
| `frontend/src/stores/workflow.ts` | 端口映射 + 多文件存储 |
| `frontend/src/workflow/canvas/CustomNode.vue` | 类别颜色图标 |
| `frontend/src/workflow/canvas/WorkflowCanvas.vue` | 端口校验 |
| `frontend/src/workflow/NodePalette.vue` | 模板区 |
| `frontend/src/workflow/templates.ts` | 新 — 7 模板 |
| `frontend/src/workflow/WorkflowList.vue` | 新 — 工作流管理 |
| `frontend/src/workflow/WorkflowMode.vue` | 集成 |
| `CHANGELOG.md` | 更新 |

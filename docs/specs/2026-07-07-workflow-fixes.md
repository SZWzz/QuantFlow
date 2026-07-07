# Workflow Mode Production Bug Fixes

## Motivation

工作流模式检查发现 2 类直接影响用户操作的问题：

1. **`CredentialManager.vue` 使用 `confirm()`/`alert()`** — Wails v3 webview 禁用了同步原生对话框,`confirm()` 始终返回 `false` 且不弹窗,导致凭证删除操作被静默阻止;`alert()` 是 no-op,保存错误提示完全不可见。

2. **`NodePalette.vue` 模板 `_id` 突变** — `templates.ts` 导出的 `TEMPLATES` 是模块级常量数组,`NodePalette.vue:19` 用 `(n as any)._id = id` 将其就地突变,导致同一模板被第二次拖入画布时携带了上一次的 `_id` 值。

此外还有一些代码整洁问题（`as any` 过多、`engine.go:178` 的 `_retryCount` 残留）属于低优先级,不在本次 spec 范围内。

## Design

### 数据流

```
User clicks "Delete" on credential
  → CredentialManager.vue calls confirm("确定删除?")
    → [BUG] window.confirm returns false (Wails v3)
    → Delete is silently blocked
```

修复后:

```
User clicks "Delete" on credential
  → CredentialManager.vue calls await confirmDialog("确定删除凭证 xxx？")
    → Wails Dialogs.Question shows native macOS dialog
    → "确定" button → resolve to true → proceed with delete
    → "取消" button → resolve to false → abort
```

### Fix 1: CredentialManager.vue — confirm/alert → confirmDialog/alertDialog

**文件**: `frontend/src/workflow/CredentialManager.vue`

- 引入 `confirmDialog` / `alertDialog` 从 `@/lib/wails`
- 将 `alert('保存失败...')` 改为 `await alertDialog('保存失败: ' + ...)`
- 将 `if (!confirm('确定删除...')) return` 改为 `if (!await confirmDialog('确定删除凭证 "' + name + '"？')) return`

**确认**: Wails v3 的 `Dialogs.Question` API 通过 `@/lib/wails` 的 `confirmDialog` 包装为 `Promise<boolean>`。`confirmDialog` 的签名已经在 `frontend/src/lib/wails.ts` 中定义（见 CLAUDE.md 规则文档）。

需要在 `setup` 中把 `confirm` 函数改为 async,所有调用处加 `await`。

### Fix 2: NodePalette.vue — 模板 _id 突变

**文件**: `frontend/src/workflow/NodePalette.vue:19`

**根本原因**: `templates.ts` 导出 `TEMPLATES` 作为 `export const TEMPLATES = [...]`。当 `NodePalette.vue` 对其节点做 `(n as any)._id = id` 时,修改的是模块级共享对象。

**修复**: 在读取模板时做深拷贝再修改:

```ts
// before
;(n as any)._id = id
// after
template.nodes = template.nodes.map(n => ({ ...n, _id: n.id }))
```

或者更好的方式:在显示模板预览时计算 `_id`,不做就地突变。

## Acceptance Criteria

- [ ] CredentialManager 的删除凭证按钮弹出原生确认对话框,点击"确定"后执行删除
- [ ] CredentialManager 的保存失败提示弹出原生信息对话框
- [ ] 双击同一模板两次,第二次不会携带第一次的 `_id`
- [ ] `npx vue-tsc --noEmit` 无新增错误
- [ ] `npx vitest run` 工作流相关测试通过

## Risks / Trade-offs

- `confirmDialog` 是异步的,函数签名从同步变为 async,需要遍历所有调用链确保 `await`
- CredentialManager 中如果有多个嵌套的 confirm 调用,需要全部改为 async/await 模式
- 模板修复的 `.map()` 方案会增加浅拷贝开销,但模板节点数极少(<20),性能影响可忽略

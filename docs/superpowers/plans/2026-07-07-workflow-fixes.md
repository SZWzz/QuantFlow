# Workflow Mode Production Bug Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement task-by-task.

**Goal:** Fix 2 bugs in workflow mode — CredentialManager `confirm()`/`alert()` violations and NodePalette template `_id` mutation.

**Architecture:** Two independent single-file fixes. No new files, no schema changes.

**Tech Stack:** Vue 3 Composition API + TypeScript, Wails v3 `Dialogs.Question`/`Dialogs.Info` via `@/lib/wails`

## Global Constraints
- No `window.confirm()` or `window.alert()` — use `confirmDialog()`/`alertDialog()` from `@/lib/wails` with `await`
- All `confirm()`/`alert()` callers must be `async` functions
- Do NOT add comments (per code style in CLAUDE.md)

---

### Task 1: Fix CredentialManager.vue — confirm/alert → confirmDialog/alertDialog

**Files:**
- Modify: `frontend/src/workflow/CredentialManager.vue`

**Interfaces:**
- Consumes: `confirmDialog(msg)` → `Promise<boolean>`; `alertDialog(msg)` → `Promise<void>` from `@/lib/wails`
- Produces: Async credential save/delete with native macOS dialogs

- [ ] **Step 1: Read the current file**

Read `frontend/src/workflow/CredentialManager.vue` fully to understand current structure.

- [ ] **Step 2: Update import and function signatures**

Add import:
```ts
import { confirmDialog, alertDialog } from '@/lib/wails'
```

Change `saveCredential()` from:
```ts
function saveCredential() {
  // ...
  alert('保存失败: ' + (e?.message || String(e)))
```
To:
```ts
async function saveCredential() {
  // ...
  await alertDialog('保存失败: ' + (e?.message || String(e)))
```

Change `deleteCredential(name)` from:
```ts
function deleteCredential(name: string) {
  if (!confirm('确定删除凭证 "' + name + '"？')) return
  // ...
```
To:
```ts
async function deleteCredential(name: string) {
  if (!await confirmDialog('确定删除凭证 "' + name + '"？')) return
  // ...
```

- [ ] **Step 3: Trace the call chain to ensure all callers handle async**

Check if `deleteCredential` is called from template with `@click` — if so, Vue handles async functions in event handlers correctly (Vue 3 catches promise rejections).

Example template usage like `@click="deleteCredential(item.name)"` works with async functions — Vue 3 calls the function and ignores the returned promise (errors go to Vue's global handler).

- [ ] **Step 4: Run type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit`
Expected: No errors in CredentialManager.vue

- [ ] **Step 5: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/workflow/CredentialManager.vue
git commit -m "fix: CredentialManager confirm/alert → confirmDialog/alertDialog (Wails v3 compat)"
```

---

### Task 2: Fix NodePalette.vue — template _id mutation

**Files:**
- Modify: `frontend/src/workflow/NodePalette.vue`

**Interfaces:**
- Consumes: `TEMPLATES` from `./templates` (module-level constant array)
- Produces: Template nodes with `_id` assigned via copy, not mutation

- [ ] **Step 1: Read current code**

Read `frontend/src/workflow/NodePalette.vue` — focus on the template rendering section and the drag handler.

- [ ] **Step 2: Fix the mutation**

Find the line that does `;(n as any)._id = id` — it should be in a function that renders or prepares template nodes for drag.

Replace with a non-mutating approach. If the code iterates over `template.nodes`, change to:

```ts
// before — mutates module-level TEMPLATES objects:
;(n as any)._id = id

// after — copy without mutation:
template.nodes = template.nodes.map(n => ({ ...n, _id: n.id }))
```

Or if the `_id` is assigned in the drag start handler, compute it at render time instead:

```ts
// In the template rendering, use a computed or local variable
const templateNodes = computed(() =>
  template.value?.nodes.map(n => ({ ...n, _id: n.id })) ?? []
)
```

- [ ] **Step 3: Run type check**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add frontend/src/workflow/NodePalette.vue
git commit -m "fix: NodePalette template _id mutation on module-level TEMPLATES constant"
```

---

### Task 3: Update CHANGELOG

- [ ] **Step 1: Add entries**

```markdown
### Fixed
- [Workflow] CredentialManager confirm/alert → confirmDialog/alertDialog (Wails v3 compat)
- [Workflow] NodePalette template _id no longer mutates module-level TEMPLATES constant
```

- [ ] **Step 2: Commit**

```bash
cd /Volumes/shenzy/vibe_coding/QuantFlow
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG with workflow fixes"
```

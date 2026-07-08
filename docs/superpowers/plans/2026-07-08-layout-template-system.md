# Layout Template System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add named layout save/load/delete to QuantFlow — SQLite persistence + IPC + frontend UI.

**Architecture:** Migration 018 (user_config table) → IPC methods in app_data.go → Pinia store actions → LayoutTemplatePanel.vue

**Tech Stack:** Go 1.25+, SQLite (mattn/go-sqlite3), Vue 3 + TypeScript, vitest

## Global Constraints

- SQLite is the only database — no PostgreSQL, no Redis
- All IPC methods go through App struct in `app_*.go` files
- Frontend confirm/alert must use `@/lib/wails` (not `window.confirm`)
- Tests use in-memory SQLite (`:memory:`)
- Migration files go in `internal/storage/migrations/`
- Layout JSON format is `DockLayoutTree` (already defined in frontend)

---

### Task 1: Migration 018 + user_config table

**Files:**
- Create: `app/internal/storage/migrations/018_user_config.sql`
- Modify: `app/internal/storage/migrate.go`

- [ ] **Step 1: Create migration SQL**

```sql
-- 018_user_config: key-value config store for UI state (layouts, preferences)
CREATE TABLE IF NOT EXISTS user_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

- [ ] **Step 2: Register in migrate.go**

Read `internal/storage/migrate.go`, find `BuiltinMigrations`, add:
```go
{18, "018_user_config"},
```

- [ ] **Step 3: Run build to verify**

```bash
cd app && go build ./...
```

Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add app/internal/storage/migrations/018_user_config.sql app/internal/storage/migrate.go
git commit -m "feat(storage): add migration 018 user_config table for layout templates"
```

---

### Task 2: IPC methods — SaveLayout / LoadLayout / ListLayouts / DeleteLayout

**Files:**
- Modify: `app/app_data.go`
- Modify: `app/app_data_test.go`

- [ ] **Step 1: Add IPC methods to app_data.go**

Append after existing `CleanupData` function:

```go
// SaveLayout stores layout JSON under a named key.
func (a *App) SaveLayout(ctx context.Context, name string, layoutJSON string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if name == "" {
		return fmt.Errorf("layout name cannot be empty")
	}
	_, err := a.db.Exec(
		"INSERT OR REPLACE INTO user_config (key, value) VALUES (?, ?)",
		"layout:"+name, layoutJSON,
	)
	if err != nil {
		return fmt.Errorf("save layout: %w", err)
	}
	return nil
}

// LoadLayout retrieves layout JSON by name.
func (a *App) LoadLayout(ctx context.Context, name string) (string, error) {
	if a.db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	var value string
	err := a.db.QueryRow(
		"SELECT value FROM user_config WHERE key = ?", "layout:"+name,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("layout %q not found", name)
	}
	if err != nil {
		return "", fmt.Errorf("load layout: %w", err)
	}
	return value, nil
}

// ListLayouts returns all saved layout names.
func (a *App) ListLayouts(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := a.db.Query("SELECT key FROM user_config WHERE key LIKE 'layout:%' ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list layouts: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		names = append(names, strings.TrimPrefix(key, "layout:"))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

// DeleteLayout removes a saved layout.
func (a *App) DeleteLayout(ctx context.Context, name string) error {
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	res, err := a.db.Exec("DELETE FROM user_config WHERE key = ?", "layout:"+name)
	if err != nil {
		return fmt.Errorf("delete layout: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("layout %q not found", name)
	}
	return nil
}
```

Add import for `"strings"` if not already present.

- [ ] **Step 2: Write integration tests**

Append to `app_data_test.go`:

```go
func TestAppLayout_RoundTrip(t *testing.T) {
	app := &App{db: setupTestDB(t)}
	defer app.db.Close()

	ctx := context.Background()

	// Save
	layoutJSON := `{"id":"root","type":"tab","tabs":[{"id":"w1","panelId":"watchlist","label":"自选股","icon":"📊"}],"activeTab":"w1"}`
	err := app.SaveLayout(ctx, "trading", layoutJSON)
	require.NoError(t, err)

	// List
	names, err := app.ListLayouts(ctx)
	require.NoError(t, err)
	require.Contains(t, names, "trading")

	// Load
	loaded, err := app.LoadLayout(ctx, "trading")
	require.NoError(t, err)
	require.JSONEq(t, layoutJSON, loaded)

	// Delete
	err = app.DeleteLayout(ctx, "trading")
	require.NoError(t, err)

	// List after delete
	names, err = app.ListLayouts(ctx)
	require.NoError(t, err)
	require.NotContains(t, names, "trading")
}

func TestAppLayout_NotFound(t *testing.T) {
	app := &App{db: setupTestDB(t)}
	defer app.db.Close()

	_, err := app.LoadLayout(context.Background(), "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	err = app.DeleteLayout(context.Background(), "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestAppLayout_EmptyName(t *testing.T) {
	app := &App{db: setupTestDB(t)}
	defer app.db.Close()

	err := app.SaveLayout(context.Background(), "", "{}")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be empty")
}
```

- [ ] **Step 3: Run tests**

```bash
cd app && go test -run TestAppLayout -v
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add app/app_data.go app/app_data_test.go
git commit -m "feat(layout): add SaveLayout/LoadLayout/ListLayouts/DeleteLayout IPC methods"
```

---

### Task 3: Frontend wails.ts bindings + terminalStore actions

**Files:**
- Modify: `frontend/src/lib/wails.ts`
- Modify: `frontend/src/stores/terminal.ts`

- [ ] **Step 1: Add typed IPC bindings to wails.ts**

Read `frontend/src/lib/wails.ts` to find existing pattern, then add:

```typescript
// Layout template management
export const layout = {
  save: (name: string, layoutJSON: string) =>
    call('SaveLayout', name, layoutJSON) as Promise<void>,
  load: (name: string) =>
    call('LoadLayout', name) as Promise<string>,
  list: () =>
    call('ListLayouts') as Promise<string[]>,
  delete: (name: string) =>
    call('DeleteLayout', name) as Promise<void>,
}
```

- [ ] **Step 2: Add store actions to terminal.ts**

Read the full terminal.ts store, then add:

```typescript
// Inside useTerminalStore:

const savedLayouts = ref<string[]>([])

async function refreshLayouts() {
  try {
    savedLayouts.value = await layout.list()
  } catch {}
}

async function saveLayout(name: string) {
  await layout.save(name, JSON.stringify(layout))
  // Also persist to localStorage as fallback
  try {
    localStorage.setItem(`quantflow-layout:${name}`, JSON.stringify(layout))
  } catch {}
  await refreshLayouts()
}

async function loadLayout(name: string) {
  // Try localStorage first for speed, then IPC
  let json: string | null = null
  try {
    json = localStorage.getItem(`quantflow-layout:${name}`)
  } catch {}
  if (!json) {
    json = await layout.load(name)
    // Cache it
    if (json) {
      try { localStorage.setItem(`quantflow-layout:${name}`, json) } catch {}
    }
  }
  if (json) {
    applyLayout(JSON.parse(json))
  }
}

async function deleteLayout(name: string) {
  await layout.delete(name)
  try {
    localStorage.removeItem(`quantflow-layout:${name}`)
  } catch {}
  await refreshLayouts()
}
```

Add these to the store's return block:
```typescript
return {
  // ... existing
  savedLayouts,
  refreshLayouts,
  saveLayout,
  loadLayout,
  deleteLayout,
}
```

- [ ] **Step 3: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/wails.ts frontend/src/stores/terminal.ts
git commit -m "feat(layout): add IPC bindings and Pinia store actions for layout templates"
```

---

### Task 4: DockView keyboard shortcuts (Ctrl+Shift+1..9)

**Files:**
- Modify: `frontend/src/terminal/DockView/DockView.vue`

- [ ] **Step 1: Read DockView.vue around keyboard shortcuts**

Find the existing keydown handler for Ctrl+1..4 presets.

- [ ] **Step 2: Add Ctrl+Shift+1..9 hotkeys**

Extend the handler:

```typescript
// After existing Ctrl+1..4:
// Ctrl+Shift+1..9 → load saved layouts
if (e.ctrlKey && e.shiftKey && e.key >= '1' && e.key <= '9') {
  e.preventDefault()
  const idx = parseInt(e.key) - 1
  if (idx < terminal.savedLayouts.length) {
    terminal.loadLayout(terminal.savedLayouts[idx])
  }
}
```

- [ ] **Step 3: Run typecheck**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/DockView/DockView.vue
git commit -m "feat(layout): add Ctrl+Shift+1..9 shortcuts for saved layouts"
```

---

### Task 5: LayoutTemplatePanel.vue

**Files:**
- Create: `frontend/src/terminal/panels/LayoutTemplatePanel.vue`
- Modify: `frontend/src/terminal/panels/registry.ts`
- Modify: `frontend/src/lib/i18n/zh.ts`
- Modify: `frontend/src/lib/i18n/en.ts`
- Create: `frontend/src/terminal/panels/__tests__/LayoutTemplatePanel.test.ts`

- [ ] **Step 1: Write failing test**

```typescript
// frontend/src/terminal/panels/__tests__/LayoutTemplatePanel.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import LayoutTemplatePanel from '../LayoutTemplatePanel.vue'
import { createTestingPinia } from '@pinia/testing'

vi.mock('@/lib/wails', () => ({
  layout: {
    save: vi.fn(),
    load: vi.fn(),
    list: vi.fn().mockResolvedValue(['trading', 'research']),
    delete: vi.fn(),
  },
  confirmDialog: vi.fn().mockResolvedValue(true),
}))

describe('LayoutTemplatePanel', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('mounts and shows saved layouts', async () => {
    const wrapper = mount(LayoutTemplatePanel, {
      global: { plugins: [createTestingPinia()] },
    })
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('trading')
    expect(wrapper.text()).toContain('research')
  })

  it('save button triggers save dialog', async () => {
    const wrapper = mount(LayoutTemplatePanel, {
      global: { plugins: [createTestingPinia()] },
    })
    await wrapper.vm.$nextTick()
    const saveBtn = wrapper.find('[data-testid="save-layout"]')
    expect(saveBtn.exists()).toBe(true)
  })
})
```

Run to verify failure:
```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/LayoutTemplatePanel.test.ts
```
Expected: FAIL (module not found / component not defined)

- [ ] **Step 2: Create LayoutTemplatePanel.vue**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { layout, confirmDialog } from '@/lib/wails'
import { useTerminalStore } from '@/stores/terminal'
import PanelHeader from '@/terminal/components/panel/PanelHeader.vue'
import EmptyState from '@/terminal/components/panel/EmptyState.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const terminal = useTerminalStore()

const loading = ref(true)
const saving = ref(false)
const newName = ref('')
const showSaveInput = ref(false)

async function loadList() {
  loading.value = true
  try {
    await terminal.refreshLayouts()
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const name = newName.value.trim()
  if (!name) return
  saving.value = true
  try {
    await terminal.saveLayout(name)
    newName.value = ''
    showSaveInput.value = false
  } finally {
    saving.value = false
  }
}

async function handleLoad(name: string) {
  await terminal.loadLayout(name)
}

async function handleDelete(name: string) {
  const ok = await confirmDialog(t('layout.confirmDelete', { name }))
  if (!ok) return
  await terminal.deleteLayout(name)
}

onMounted(loadList)
</script>

<template>
  <div class="layout-template-panel">
    <PanelHeader
      :title="t('layout.title')"
      :controls="[
        { icon: 'plus', action: () => { showSaveInput = !showSaveInput; if (showSaveInput) newName = '' }, title: t('layout.saveNew'), loading: saving },
        { icon: 'refresh', action: loadList, title: t('common.refresh') },
      ]"
    />

    <div v-if="showSaveInput" class="save-input">
      <input
        v-model="newName"
        :placeholder="t('layout.namePlaceholder')"
        @keyup.enter="handleSave"
        data-testid="layout-name-input"
      />
      <button @click="handleSave" :disabled="!newName.trim()" data-testid="save-layout">
        {{ t('common.save') }}
      </button>
    </div>

    <div v-if="loading" class="loading">{{ t('common.loading') }}</div>

    <div v-else-if="terminal.savedLayouts.length === 0" class="empty">
      <EmptyState :message="t('layout.empty')" />
    </div>

    <div v-else class="layout-list">
      <div
        v-for="(name, idx) in terminal.savedLayouts"
        :key="name"
        class="layout-item"
      >
        <div class="layout-info" @click="handleLoad(name)">
          <span class="layout-hotkey">Ctrl+Shift+{{ idx + 1 }}</span>
          <span class="layout-name">{{ name }}</span>
        </div>
        <button class="delete-btn" @click="handleDelete(name)" :title="t('common.delete')">
          ✕
        </button>
      </div>
    </div>

    <div class="hint">
      {{ t('layout.hint') }}
    </div>
  </div>
</template>

<style scoped>
.layout-template-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.save-input {
  display: flex;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--border-color, #333);
}
.save-input input {
  flex: 1;
  background: transparent;
  border: 1px solid var(--border-color, #555);
  color: var(--text-color, #fff);
  padding: 4px 8px;
  border-radius: 4px;
}
.save-input button {
  background: var(--accent-color, #4a9eff);
  color: #fff;
  border: none;
  padding: 4px 12px;
  border-radius: 4px;
  cursor: pointer;
}
.save-input button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.layout-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.layout-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color, #222);
}
.layout-item:hover {
  background: var(--hover-bg, rgba(255,255,255,0.05));
}
.layout-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}
.layout-hotkey {
  font-size: 11px;
  color: var(--text-secondary, #888);
}
.layout-name {
  font-size: 14px;
  color: var(--text-color, #eee);
}
.delete-btn {
  background: none;
  border: none;
  color: var(--text-secondary, #666);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
}
.delete-btn:hover {
  background: rgba(255, 0, 0, 0.1);
  color: #ff4444;
}
.loading,
.empty {
  padding: 24px;
  text-align: center;
  color: var(--text-secondary, #888);
}
.hint {
  padding: 8px 12px;
  font-size: 11px;
  color: var(--text-secondary, #666);
  border-top: 1px solid var(--border-color, #222);
}
</style>
```

- [ ] **Step 3: Register panel**

Read `frontend/src/terminal/panels/registry.ts`, add:

```typescript
register('layout-templates', () => import('./LayoutTemplatePanel.vue'), {
  id: 'layout-templates',
  label: '布局模板',
  category: 'system',
  description: '管理已保存的布局模板',
})
```

- [ ] **Step 4: Add i18n**

In `zh.ts`:
```typescript
layout: {
  title: '布局模板',
  saveNew: '保存当前布局',
  namePlaceholder: '布局名称...',
  empty: '暂无已保存的布局',
  confirmDelete: '确认删除布局 "{name}"？',
  hint: 'Ctrl+Shift+1..9 快速切换已保存布局',
},
```

In `en.ts`:
```typescript
layout: {
  title: 'Layout Templates',
  saveNew: 'Save Current Layout',
  namePlaceholder: 'Layout name...',
  empty: 'No saved layouts yet',
  confirmDelete: 'Delete layout "{name}"?',
  hint: 'Ctrl+Shift+1..9 to switch saved layouts',
},
```

- [ ] **Step 5: Run tests**

```bash
cd frontend && npx vitest run src/terminal/panels/__tests__/LayoutTemplatePanel.test.ts
```

Expected: PASS

- [ ] **Step 6: Run full check**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

Expected: No errors, all tests pass

- [ ] **Step 7: Commit**

```bash
git add frontend/src/terminal/panels/LayoutTemplatePanel.vue frontend/src/terminal/panels/registry.ts frontend/src/lib/i18n/zh.ts frontend/src/lib/i18n/en.ts frontend/src/terminal/panels/__tests__/LayoutTemplatePanel.test.ts
git commit -m "feat(layout): add LayoutTemplatePanel with save/load/delete UI"
```

---

### Task 6: Wire it all together — CHANGELOG + version check

**Files:**
- Modify: `CHANGELOG.md`
- Check: `frontend/package.json`
- Check: `README.md`

- [ ] **Step 1: Update CHANGELOG.md**

Add entry under today's date:
```markdown
### Added
- [Terminal] Layout Template System — named layout save/load/delete via SQLite persistence,
  LayoutTemplatePanel UI, Ctrl+Shift+1..9 shortcuts for quick switching
```

- [ ] **Step 2: Check version date**

Verify `frontend/package.json` version is `2026.7.8` and `README.md` badge matches. Update if stale.

- [ ] **Step 3: Run full check suite**

```bash
cd app && go vet ./... && go test ./... -count=1
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

Expected: All clean

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md frontend/package.json README.md
git commit -m "chore: update CHANGELOG and version for layout template system"
```

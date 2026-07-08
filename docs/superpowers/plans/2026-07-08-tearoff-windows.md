# 浮窗面板 (Tear-off Windows) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to tear off DockView panels into independent macOS OS windows via Wails v3 multi-window API.

**Architecture:** Go `App.TearOffPanel()` → `application.Current().Window.NewWithOptions()` with URL `/#/tearoff/{instanceId}`. New window loads SPA, Vue router routes to `TearOffPanel.vue`, which fetches panel info via Go IPC and renders the panel component.

**Tech Stack:** Go 1.25+ (Wails v3 alpha2.111), Vue 3 + TypeScript, Vite

## Global Constraints

- All Go code in `main` package at project root
- Use `application.Current()` to access global Wails App
- `window.go.main.App` proxy auto-exposes new methods to frontend
- No new external dependencies
- Hash-based routing (`createWebHashHistory`)
- No theme/state sync between windows for MVP

---

### Task 1: Go backend — App struct fields + tear-off IPC methods

**Files:**
- Modify: `app.go` — add `tearOffWindows` field to App struct
- Create: `app_tearoff.go` — TearOffPanel/CloseTearOffWindow/GetTearOffPanelInfo/ListTearOffWindows
- Create: `app_tearoff_test.go` — map management tests

**Interfaces:**
- Produces: `App.TearOffPanel(panelId, instanceId, label, paramsJson string) error`
- Produces: `App.CloseTearOffWindow(instanceId string) error`
- Produces: `App.GetTearOffPanelInfo(instanceId string) (string, string, string, error)` — returns `(panelId, label, paramsJson)`
- Produces: `App.ListTearOffWindows() []string`

- [ ] **Step 1: Add `tearOffWindows` + `tearOffWindowsMu` to App struct**

In `app.go`, find the `type App struct {` block and add before the closing `}`:

```go

	// Tear-off window tracking.
	tearOffWindows   map[string]*tearOffEntry  // instanceId → entry
	tearOffWindowsMu sync.RWMutex
```

Also add `"sync"` to the import block:
```go
import (
	// ...
	"sync"
	// ...
)
```

- [ ] **Step 2: Create `app_tearoff.go`**

```go
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// tearOffEntry holds information about a tear-off panel window.
type tearOffEntry struct {
	Win        *application.WebviewWindow
	PanelID    string
	InstanceID string
	Label      string
	Params     string // JSON
}

// TearOffPanel creates a new OS window containing the specified panel.
func (a *App) TearOffPanel(panelId, instanceId, label, paramsJson string) error {
	app := application.Current()
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      fmt.Sprintf("tearoff-%s", instanceId),
		Title:     label,
		Width:     800,
		Height:    600,
		MinWidth:  400,
		MinHeight: 300,
		URL:       fmt.Sprintf("/#/tearoff/%s", instanceId),
	})
	entry := &tearOffEntry{
		Win: win, PanelID: panelId,
		InstanceID: instanceId, Label: label, Params: paramsJson,
	}
	a.tearOffWindowsMu.Lock()
	a.tearOffWindows[instanceId] = entry
	a.tearOffWindowsMu.Unlock()

	win.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		slog.Debug("tear-off window closing", "instanceId", instanceId, "panelId", panelId)
		a.tearOffWindowsMu.Lock()
		delete(a.tearOffWindows, instanceId)
		a.tearOffWindowsMu.Unlock()
	})
	slog.Info("tear-off panel opened", "instanceId", instanceId, "panelId", panelId, "label", label)
	return nil
}

// CloseTearOffWindow closes a specific tear-off window by instance ID.
func (a *App) CloseTearOffWindow(instanceId string) error {
	a.tearOffWindowsMu.RLock()
	entry, ok := a.tearOffWindows[instanceId]
	a.tearOffWindowsMu.RUnlock()
	if !ok {
		return fmt.Errorf("tear-off window not found: %s", instanceId)
	}
	entry.Win.Close()
	return nil
}

// GetTearOffPanelInfo returns panelId, label, and params JSON for a tear-off panel.
func (a *App) GetTearOffPanelInfo(instanceId string) (string, string, string, error) {
	a.tearOffWindowsMu.RLock()
	entry, ok := a.tearOffWindows[instanceId]
	a.tearOffWindowsMu.RUnlock()
	if !ok {
		return "", "", "", fmt.Errorf("tear-off panel not found: %s", instanceId)
	}
	return entry.PanelID, entry.Label, entry.Params, nil
}

// ListTearOffWindows returns instance IDs of all open tear-off windows.
func (a *App) ListTearOffWindows() []string {
	a.tearOffWindowsMu.RLock()
	defer a.tearOffWindowsMu.RUnlock()
	ids := make([]string, 0, len(a.tearOffWindows))
	for id := range a.tearOffWindows {
		ids = append(ids, id)
	}
	return ids
}
```

- [ ] **Step 3: Create `app_tearoff_test.go`**

```go
package main

import (
	"testing"
)

func TestTearOffWindows_MapManagement(t *testing.T) {
	a := &App{tearOffWindows: make(map[string]*tearOffEntry)}

	if ids := a.ListTearOffWindows(); len(ids) != 0 {
		t.Fatalf("expected empty, got %v", ids)
	}

	a.tearOffWindowsMu.Lock()
	a.tearOffWindows["id1"] = &tearOffEntry{
		PanelID: "watchlist", InstanceID: "id1", Label: "自选股", Params: "{}",
	}
	a.tearOffWindows["id2"] = &tearOffEntry{
		PanelID: "order-entry", InstanceID: "id2",
		Label: "交易", Params: `{"symbol":"000001"}`,
	}
	a.tearOffWindowsMu.Unlock()

	if ids := a.ListTearOffWindows(); len(ids) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(ids))
	}

	panelId, label, params, err := a.GetTearOffPanelInfo("id1")
	if err != nil {
		t.Fatalf("GetTearOffPanelInfo error: %v", err)
	}
	if panelId != "watchlist" {
		t.Errorf("panelId = %q, want %q", panelId, "watchlist")
	}
	if label != "自选股" {
		t.Errorf("label = %q, want %q", label, "自选股")
	}
	if params != "{}" {
		t.Errorf("params = %q, want %q", params, "{}")
	}

	if _, _, _, err = a.GetTearOffPanelInfo("nonexistent"); err == nil {
		t.Fatal("expected error for non-existent instanceId")
	}

	a.tearOffWindowsMu.Lock()
	delete(a.tearOffWindows, "id1")
	a.tearOffWindowsMu.Unlock()

	if ids := a.ListTearOffWindows(); len(ids) != 1 {
		t.Fatalf("expected 1 window after delete, got %d", len(ids))
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test -run TestTearOff -v -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app.go app_tearoff.go app_tearoff_test.go
git commit -m "feat(tearoff): add Go backend — TearOffPanel IPC methods and window management"
```

---

### Task 2: Frontend — TearOffPanel route component + route + App.vue mode guard

**Files:**
- Create: `frontend/src/terminal/TearOffPanel.vue`
- Modify: `frontend/src/main.ts` — add /tearoff/:instanceId route
- Modify: `frontend/src/App.vue` — skip mode sync in tear-off mode

- [ ] **Step 1: Create `frontend/src/terminal/TearOffPanel.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import { getPanelComponent } from '@/terminal/panels/registry'

const route = useRoute()
const instanceId = route.params.instanceId as string
const panelId = ref('')
const params = ref<Record<string, any> | undefined>()
const panelComponent = shallowRef<any>(null)

onMounted(async () => {
  try {
    const go = (window as any).go?.main?.App
    if (!go) {
      console.error('[TearOffPanel] Wails bridge not available')
      return
    }
    // GetTearOffPanelInfo returns [panelId, label, paramsJson]
    const info: [string, string, string] = await go.GetTearOffPanelInfo(instanceId)
    panelId.value = info[0]
    const paramsJson = info[2]
    if (paramsJson && paramsJson !== '{}' && paramsJson !== '""') {
      try { params.value = JSON.parse(paramsJson) } catch { params.value = undefined }
    }
    panelComponent.value = getPanelComponent(panelId.value)
  } catch (err) {
    console.error('[TearOffPanel] failed to get panel info:', err)
  }
})
</script>

<template>
  <component
    v-if="panelComponent"
    :is="panelComponent"
    :panel-id="panelId"
    :params="params"
    class="tearoff-panel"
  />
  <div v-else class="tearoff-loading">
    <p>Loading panel...</p>
  </div>
</template>

<style scoped>
.tearoff-panel {
  width: 100vw;
  height: 100vh;
  overflow: auto;
}
.tearoff-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  color: #888;
  font-family: system-ui, sans-serif;
}
</style>
```

- [ ] **Step 2: Add tear-off route in `frontend/src/main.ts`**

Find the routes array and add after the workflow route:

```typescript
    {
      path: '/tearoff/:instanceId',
      name: 'tearoff',
      component: () => import('@/terminal/TearOffPanel.vue'),
    },
```

- [ ] **Step 3: Modify `frontend/src/App.vue` to skip mode sync in tear-off mode**

Replace the two `watch()` calls (mode sync and back/forward sync) with conditional versions. Add `import { computed } from 'vue'` and:

```typescript
const isTearOff = computed(() => route.path.startsWith('/tearoff'))
```

Wrap the first watcher (mode→URL sync):
```typescript
// Keep URL in sync with session mode — skip in tear-off windows.
if (!route.path.startsWith('/tearoff')) {
  watch(() => session.ui.mode, (mode) => {
    const target = mode === 'workflow' ? '/workflow' : '/'
    if (route.path !== target) router.push(target)
  }, { immediate: true })
}
```

Wrap the second watcher (URL→mode sync):
```typescript
// Keep session mode in sync with URL — skip in tear-off windows.
if (!route.path.startsWith('/tearoff')) {
  watch(() => route.path, (path) => {
    const expectedMode = path === '/workflow' ? 'workflow' : 'terminal'
    if (session.ui.mode !== expectedMode) {
      session.ui.mode = expectedMode
    }
  })
}
```

- [ ] **Step 4: Build check**

Run: `go build ./... && cd frontend && npx vue-tsc --noEmit`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/TearOffPanel.vue frontend/src/main.ts frontend/src/App.vue
git commit -m "feat(tearoff): add TearOffPanel route component and tear-off mode detection"
```

---

### Task 3: Frontend — DockTab tear-off button

**Files:**
- Modify: `frontend/src/terminal/DockView/DockTab.vue`

- [ ] **Step 1: Add `tearOff` function to script section**

Add after the `closeTab` function (line 101):

```typescript
function tearOff(tab: DockTabState) {
  const instanceId = `${tab.panelId}_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
  const go = (window as any).go?.main?.App
  if (!go) return
  go.TearOffPanel(tab.panelId, instanceId, tab.label, JSON.stringify(tab.params || {}))
    .then(() => closeTab(tab.id))
    .catch((err: any) => console.error('[DockTab] tear-off failed:', err))
}
```

- [ ] **Step 2: Add tear-off button in template**

Add a `↗` button before the close button in the tab-btn template (after the tab-label span, before the tab-close span):

```vue
          <span
            class="tab-tearoff"
            @click.stop="tearOff(tab)"
            title="撕下为新窗口"
          >↗</span>
```

- [ ] **Step 3: Add CSS for the tear-off button**

Add after the `.tab-close:hover` CSS block (around line 330):

```css
.tab-tearoff {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  opacity: 0;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
  cursor: pointer;
  margin-left: 1px;
  font-size: 11px;
  line-height: 1;
}
.tab-btn:hover .tab-tearoff {
  opacity: 0.4;
}
.tab-tearoff:hover {
  opacity: 1 !important;
  background: var(--color-bg-hover);
  color: var(--color-text-primary) !important;
}
```

- [ ] **Step 4: Build check**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: No type errors

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/DockView/DockTab.vue
git commit -m "feat(tearoff): add tear-off button to DockTab"
```

---

### Task 4: Type definitions + CHANGELOG + final check

**Files:**
- Modify: `frontend/src/types/wails-runtime.d.ts`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add new IPC methods to type definitions**

In `frontend/src/types/wails-runtime.d.ts`, add to the `AppMethods` interface (after the System Monitor section):

```typescript
  // --- Tear-off Windows ---
  TearOffPanel(panelId: string, instanceId: string, label: string, paramsJson: string): Promise<void>
  CloseTearOffWindow(instanceId: string): Promise<void>
  GetTearOffPanelInfo(instanceId: string): Promise<[string, string, string]>
  ListTearOffWindows(): Promise<string[]>
```

- [ ] **Step 2: Update CHANGELOG.md**

Under `[2026.7.8]`:
```markdown
- [Terminal] Tear-off windows — DockTab panels can be detached into independent OS windows
  via Wails multi-window API (TearOffPanel.vue, app_tearoff.go)
```

- [ ] **Step 3: Run full check**

```bash
go vet ./... && go test ./... -count=1
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/wails-runtime.d.ts CHANGELOG.md
git commit -m "chore: update types and CHANGELOG for tear-off windows"
```

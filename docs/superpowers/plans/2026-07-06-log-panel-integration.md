# Log Panel Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 stderr 日志集成到前端 UI，用户不再需要单独开终端窗口看日志。

**Architecture:** Go 后端用 RingBuffer 捕获所有 `slog` 输出，前端通过轮询 `GetLogs(afterID)` 拉取增量日志，`LogPanel.vue` 以终端风格展示并支持级别筛选和搜索。

**Tech Stack:** Go `log/slog` + custom handler, Vue 3 Composition API, Wails v3 IPC (polling)

## Global Constraints

- Go 1.25+, `log/slog` only (no third-party logging libs)
- RingBuffer 容量固定在 5000 条
- 前端面板保留最多 2000 条日志（防内存泄漏）
- stderr 输出保持不变（双写，不替换）
- 所有新代码必须包含测试
- Panel 注册在 `registry.ts` 中 `register()` 调用
- 新增 composable 放在 `frontend/src/lib/composables/` 下

---

### Task 1: RingBuffer + LogEntry (Go 端日志数据结构)

**Files:**
- Create: `internal/logging/ring_buffer.go`
- Test: `internal/logging/ring_buffer_test.go`

**Interfaces:**
- Produces: `LogEntry` struct, `RingBuffer` struct with `Push()`, `Lines()` methods

- [ ] **Step 1: Write the failing test**

```go
// internal/logging/ring_buffer_test.go
package logging

import (
	"testing"
	"time"
)

func TestRingBufferPushAndLines(t *testing.T) {
	rb := NewRingBuffer(3)
	e1 := LogEntry{ID: 1, Time: time.Now(), Level: "info", Message: "msg1"}
	e2 := LogEntry{ID: 2, Time: time.Now(), Level: "warn", Message: "msg2"}
	e3 := LogEntry{ID: 3, Time: time.Now(), Level: "error", Message: "msg3"}

	rb.Push(e1)
	rb.Push(e2)
	rb.Push(e3)

	lines := rb.Lines(0, 10)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestRingBufferOverflow(t *testing.T) {
	rb := NewRingBuffer(2)
	rb.Push(LogEntry{ID: 1, Message: "a"})
	rb.Push(LogEntry{ID: 2, Message: "b"})
	rb.Push(LogEntry{ID: 3, Message: "c"})

	lines := rb.Lines(0, 10)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after overflow, got %d", len(lines))
	}
	if lines[0].Message != "b" || lines[1].Message != "c" {
		t.Fatalf("expected oldest dropped, got %+v", lines)
	}
}

func TestRingBufferLinesAfterID(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push(LogEntry{ID: 1, Message: "a"})
	rb.Push(LogEntry{ID: 2, Message: "b"})
	rb.Push(LogEntry{ID: 3, Message: "c"})

	lines := rb.Lines(1, 10)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after ID 1, got %d", len(lines))
	}
	if lines[0].Message != "b" || lines[1].Message != "c" {
		t.Fatalf("got %+v", lines)
	}
}

func TestRingBufferZeroCapacity(t *testing.T) {
	rb := NewRingBuffer(0)
	rb.Push(LogEntry{ID: 1, Message: "a"})
	lines := rb.Lines(0, 10)
	if len(lines) != 0 {
		t.Fatal("expected empty ring buffer with 0 capacity")
	}
}

func TestRingBufferLinesLimit(t *testing.T) {
	rb := NewRingBuffer(100)
	for i := 1; i <= 20; i++ {
		rb.Push(LogEntry{ID: int64(i), Message: "msg"})
	}
	lines := rb.Lines(0, 5)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -run TestRingBuffer -v`
Expected: Build fail — "undefined: NewRingBuffer"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/logging/ring_buffer.go
package logging

import (
	"sync"
	"time"
)

type LogEntry struct {
	ID      int64          `json:"id"`
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

type RingBuffer struct {
	mu     sync.Mutex
	buffer []LogEntry
	nextID int64
	head   int
	count  int
	max    int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 5000
	}
	return &RingBuffer{
		buffer: make([]LogEntry, capacity),
		max:    capacity,
		nextID: 1,
	}
}

func (rb *RingBuffer) Push(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.max == 0 {
		return
	}
	entry.ID = rb.nextID
	rb.nextID++
	if rb.count < rb.max {
		rb.buffer[rb.count] = entry
		rb.count++
	} else {
		rb.buffer[rb.head] = entry
		rb.head = (rb.head + 1) % rb.max
	}
}

func (rb *RingBuffer) Lines(afterID int64, limit int) []LogEntry {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.count == 0 || limit <= 0 {
		return nil
	}

	// Traverse from oldest to newest
	var result []LogEntry
	for i := 0; i < rb.count; i++ {
		idx := (rb.head + i) % rb.max
		entry := rb.buffer[idx]
		if entry.ID > afterID {
			result = append(result, entry)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -run TestRingBuffer -v`
Expected: All 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logging/ring_buffer.go internal/logging/ring_buffer_test.go
git commit -m "feat(logging): add RingBuffer and LogEntry types"
```

---

### Task 2: Dual handler (slog.Handler 双写 stderr + RingBuffer)

**Files:**
- Create: `internal/logging/dual_handler.go`
- Test: `internal/logging/dual_handler_test.go`

**Interfaces:**
- Produces: `dualHandler` type that implements `slog.Handler`; global var `Ring *RingBuffer` in logging package
- Consumes: `RingBuffer` from Task 1

- [ ] **Step 1: Write the failing test**

```go
// internal/logging/dual_handler_test.go
package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestDualHandlerWritesToInner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "hello dual", 0)
	if err := dh.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if !strings.Contains(buf.String(), "hello dual") {
		t.Errorf("expected inner handler to receive message, got %q", buf.String())
	}
}

func TestDualHandlerWritesToRingBuffer(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "ring test", 0)
	if err := dh.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line in ring buffer, got %d", len(lines))
	}
	if lines[0].Message != "ring test" {
		t.Errorf("expected 'ring test', got %q", lines[0].Message)
	}
}

func TestDualHandlerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	ctx := context.Background()
	infoRec := slog.NewRecord(time.Now(), slog.LevelInfo, "should skip", 0)
	_ = dh.Handle(ctx, infoRec)

	errRec := slog.NewRecord(time.Now(), slog.LevelError, "should capture", 0)
	_ = dh.Handle(ctx, errRec)

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Message != "should capture" {
		t.Errorf("expected 'should capture', got %q", lines[0].Message)
	}
}

func TestDualHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	rb := NewRingBuffer(100)
	dh := newDualHandler(inner, rb)

	dh2 := dh.WithAttrs([]slog.Attr{slog.String("key1", "val1")})
	ctx := context.Background()
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "with attrs", 0)
	if err := dh2.Handle(ctx, rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	lines := rb.Lines(0, 10)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Attrs["key1"] != "val1" {
		t.Errorf("expected attrs key1=val1, got %v", lines[0].Attrs)
	}
	if !strings.Contains(buf.String(), "key1=val1") {
		t.Errorf("expected attrs in inner handler output, got %q", buf.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -run TestDualHandler -v`
Expected: Build fail — "undefined: newDualHandler"

- [ ] **Step 3: Write minimal implementation**

```go
// internal/logging/dual_handler.go
package logging

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type dualHandler struct {
	inner slog.Handler
	rb    *RingBuffer
	mu    sync.Mutex
	attrs []slog.Attr
}

func newDualHandler(inner slog.Handler, rb *RingBuffer) *dualHandler {
	return &dualHandler{inner: inner, rb: rb}
}

func (h *dualHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *dualHandler) Handle(ctx context.Context, rec slog.Record) error {
	// Write to inner (stderr)
	if err := h.inner.Handle(ctx, rec); err != nil {
		return err
	}

	// Collect attrs
	attrs := make(map[string]any)
	h.mu.Lock()
	for _, a := range h.attrs {
		attrs[a.Key] = a.Value.Any()
	}
	h.mu.Unlock()
	rec.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	// Push to ring buffer
	entry := LogEntry{
		Time:    rec.Time,
		Level:   rec.Level.String(),
		Message: rec.Message,
		Attrs:   attrs,
	}
	h.rb.Push(entry)

	return nil
}

func (h *dualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	combined := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(combined, h.attrs)
	copy(combined[len(h.attrs):], attrs)
	h.mu.Unlock()

	return &dualHandler{
		inner: h.inner.WithAttrs(attrs),
		rb:    h.rb,
		attrs: combined,
	}
}

func (h *dualHandler) WithGroup(name string) slog.Handler {
	return &dualHandler{
		inner: h.inner.WithGroup(name),
		rb:    h.rb,
		attrs: h.attrs,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -run TestDualHandler -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Modify logging.Setup to use dualHandler**

Edit `internal/logging/logging.go`:

```go
package logging

import (
	"log/slog"
	"os"
)

var Ring = NewRingBuffer(5000)

func Setup(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	handler := newDualHandler(textHandler, Ring)
	slog.SetDefault(slog.New(handler))
}
```

- [ ] **Step 6: Run all logging tests**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/logging/ -v`
Expected: All tests PASS (existing `TestSetupDoesNotPanic`, `TestSetupOutput`, `TestSetupLevelFilter` + new ones)

- [ ] **Step 7: Commit**

```bash
git add internal/logging/dual_handler.go internal/logging/dual_handler_test.go internal/logging/logging.go
git commit -m "feat(logging): dual slog handler writes to stderr + RingBuffer"
```

---

### Task 3: 暴露 GetLogs API 给前端

**Files:**
- Modify: `app.go` (add `GetLogs` method)
- Modify: `app_system.go` or similar (choose the right file)
- Test: `app_test.go` for GetLogs

**Interfaces:**
- Consumes: `logging.Ring` (global `*RingBuffer` from Task 2)
- Produces: `App.GetLogs(afterID int) []LogEntry` — Wails-bound method

- [ ] **Step 1: Write the failing test**

Add to an existing test file or create new:
```go
// internal/logging/integration_test.go
package logging

import (
	"testing"
)

func TestGlobalRingInit(t *testing.T) {
	if Ring == nil {
		t.Fatal("expected Ring to be initialized")
	}
	Ring.Push(LogEntry{Message: "integration test"})
	lines := Ring.Lines(0, 1)
	if len(lines) != 1 {
		t.Fatal("expected 1 line")
	}
}

func TestGlobalRingCapacity(t *testing.T) {
	if Ring == nil {
		t.Skip("Ring not initialized")
	}
	// After Setup, Ring should have 5000 capacity
	// We just verify it's not nil and accepts writes
	Ring.Push(LogEntry{Message: "capacity test"})
}
```

- [ ] **Step 2: Add GetLogs method to App**

Edit `app.go`, add after `GetVersion()` or near other system methods:

```go
// GetLogs returns log entries after the given ID (0 = all).
// Used by the frontend LogPanel to poll for new entries.
func (a *App) GetLogs(afterID int) []LogEntry {
	return logging.Ring.Lines(int64(afterID), 200)
}
```

Also add the import for `"quantflow/internal/logging"` if not already present — check existing imports. (`logging` is already imported in `app.go` at line 24.)

- [ ] **Step 3: Write app-level test**

Add or update `app_test.go`:

```go
func TestGetLogs(t *testing.T) {
	// GetLogs should not panic and return valid slice
	app := &App{}
	logging.Setup("debug")
	slog.Info("test log for GetLogs")
	logs := app.GetLogs(0)
	if len(logs) == 0 {
		t.Fatal("expected at least 1 log entry")
	}
	lastID := logs[len(logs)-1].ID
	// GetLogs with afterID=lastID should return 0 new entries
	newLogs := app.GetLogs(int(lastID))
	if len(newLogs) != 0 {
		t.Fatalf("expected 0 new entries after last ID, got %d", len(newLogs))
	}
	// Write another log and verify it appears
	slog.Info("second test log")
	newLogs = app.GetLogs(int(lastID))
	if len(newLogs) != 1 {
		t.Fatalf("expected 1 new entry, got %d", len(newLogs))
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./... -run "TestGetLogs|TestGlobalRing" -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add app.go app_test.go internal/logging/integration_test.go
git commit -m "feat(app): expose GetLogs API for log panel polling"
```

---

### Task 4: 前端类型 + wails.ts 助手

**Files:**
- Modify: `frontend/src/lib/wails.ts`

- [ ] **Step 1: Add LogEntry type and GetLogs wrapper**

Add to `frontend/src/lib/wails.ts` (at the end, before `export {}` if there is one, or just before the last line):

```typescript
// ── Log Panel ────────────────────────────────────────────────────────

export interface LogEntry {
  id: number
  time: string
  level: string
  message: string
  attrs?: Record<string, any>
}

export async function GetLogs(afterID: number): Promise<LogEntry[]> {
  return wailsCall<LogEntry[]>('GetLogs', afterID)
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/lib/wails.ts
git commit -m "feat(frontend): add LogEntry type and GetLogs helper"
```

---

### Task 5: useLogger composable

**Files:**
- Create: `frontend/src/lib/composables/useLogger.ts`
- Create: `frontend/src/lib/composables/__tests__/useLogger.test.ts`

- [ ] **Step 1: Write the composable**

```typescript
// frontend/src/lib/composables/useLogger.ts
import { ref, onMounted, onUnmounted } from 'vue'
import { GetLogs, type LogEntry } from '@/lib/wails'

export interface LogFilter {
  levels: Set<string>
  search: string
}

export function useLogger(pollInterval = 1000) {
  const entries = ref<LogEntry[]>([])
  const lastID = ref(0)
  const maxEntries = 2000
  const filter = ref<LogFilter>({ levels: new Set(['info', 'warn', 'error']), search: '' })
  let timer: ReturnType<typeof setInterval> | null = null

  async function poll() {
    try {
      const newEntries = await GetLogs(lastID.value)
      if (newEntries.length > 0) {
        lastID.value = newEntries[newEntries.length - 1].id
        entries.value.push(...newEntries)
        if (entries.value.length > maxEntries) {
          entries.value = entries.value.slice(entries.value.length - maxEntries)
        }
      }
    } catch (e) {
      // Silently fail — log panel should never crash the app
    }
  }

  function filteredEntries(): LogEntry[] {
    return entries.value.filter(e => {
      if (!filter.value.levels.has(e.level)) return false
      if (filter.value.search) {
        const q = filter.value.search.toLowerCase()
        const msg = e.message.toLowerCase()
        const attrs = e.attrs ? JSON.stringify(e.attrs).toLowerCase() : ''
        return msg.includes(q) || attrs.includes(q)
      }
      return true
    })
  }

  function toggleLevel(level: string) {
    const s = new Set(filter.value.levels)
    if (s.has(level)) {
      if (s.size > 1) s.delete(level) // keep at least one level
    } else {
      s.add(level)
    }
    filter.value = { ...filter.value, levels: s }
  }

  function setSearch(q: string) {
    filter.value = { ...filter.value, search: q }
  }

  function clear() {
    entries.value = []
    lastID.value = 0
  }

  onMounted(() => {
    poll()
    timer = setInterval(poll, pollInterval)
  })

  onUnmounted(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })

  return { entries, filter, filteredEntries, toggleLevel, setSearch, clear, poll }
}
```

- [ ] **Step 2: Write the composable test**

```typescript
// frontend/src/lib/composables/__tests__/useLogger.test.ts
import { describe, it, expect } from 'vitest'
import { LogEntry } from '@/lib/wails'

describe('useLogger', () => {
  it('LogEntry type has required fields', () => {
    const entry: LogEntry = {
      id: 1,
      time: '2026-07-06T10:00:00Z',
      level: 'info',
      message: 'test',
      attrs: { key: 'val' },
    }
    expect(entry.id).toBe(1)
    expect(entry.level).toBe('info')
    expect(entry.message).toBe('test')
  })

  it('LogEntry works without attrs', () => {
    const entry: LogEntry = {
      id: 2,
      time: '2026-07-06T10:00:01Z',
      level: 'error',
      message: 'no attrs',
    }
    expect(entry.attrs).toBeUndefined()
  })
})
```

- [ ] **Step 3: Verify**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit && npx vitest run`
Expected: TypeScript clean, tests PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/lib/composables/useLogger.ts frontend/src/lib/composables/__tests__/useLogger.test.ts
git commit -m "feat(frontend): add useLogger composable with polling"
```

---

### Task 6: LogPanel.vue 组件

**Files:**
- Create: `frontend/src/terminal/panels/LogPanel.vue`
- Create: `frontend/src/terminal/panels/__tests__/LogPanel.test.ts`

- [ ] **Step 1: Write LogPanel.vue**

```vue
<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useLogger } from '@/lib/composables/useLogger'
import { confirmDialog } from '@/lib/wails'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const LEVELS = ['debug', 'info', 'warn', 'error']

const LEVEL_COLORS: Record<string, string> = {
  debug: '#888',
  info: '#e0e0e0',
  warn: '#f0ad4e',
  error: '#ef4444',
}

const {
  entries, filter, filteredEntries,
  toggleLevel, setSearch, clear, poll,
} = useLogger()

const scrollContainer = ref<HTMLElement | null>(null)
const autoScroll = ref(true)

const displayEntries = computed(() => filteredEntries().slice(-500))

function onScroll() {
  if (!scrollContainer.value) return
  const el = scrollContainer.value
  const threshold = 30
  autoScroll.value = (el.scrollHeight - el.scrollTop - el.clientHeight) < threshold
}

watch(displayEntries, async () => {
  if (autoScroll.value) {
    await nextTick()
    if (scrollContainer.value) {
      scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
    }
  }
})

async function handleClear() {
  const ok = await confirmDialog('确定清空所有日志？')
  if (ok) clear()
}

function levelClass(lvl: string): string {
  return `log-level-${lvl}`
}

function formatTime(t: string): string {
  try {
    const d = new Date(t)
    return d.toLocaleTimeString('zh-CN', { hour12: false })
  } catch {
    return t
  }
}

function formatAttrs(attrs?: Record<string, any>): string {
  if (!attrs || Object.keys(attrs).length === 0) return ''
  return Object.entries(attrs)
    .map(([k, v]) => `${k}=${v}`)
    .join(' ')
}

function highlightSearch(text: string): string {
  if (!filter.value.search) return text
  const q = filter.value.search.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const re = new RegExp(`(${q})`, 'gi')
  return text.replace(re, '<mark class="log-highlight">$1</mark>')
}
</script>

<template>
  <div class="log-panel">
    <!-- Toolbar -->
    <div class="log-toolbar">
      <div class="log-levels">
        <button
          v-for="lvl in LEVELS"
          :key="lvl"
          class="log-level-btn"
          :class="{ active: filter.levels.has(lvl), [levelClass(lvl)]: true }"
          @click="toggleLevel(lvl)"
        >
          {{ lvl.toUpperCase() }}
        </button>
      </div>
      <div class="log-toolbar-right">
        <input
          class="log-search"
          type="text"
          :value="filter.search"
          @input="setSearch(($event.target as HTMLInputElement).value)"
          placeholder="搜索日志..."
        />
        <button class="log-clear-btn" @click="handleClear">清空</button>
      </div>
    </div>

    <!-- Log entries -->
    <div class="log-entries" ref="scrollContainer" @scroll="onScroll">
      <div
        v-for="entry in displayEntries"
        :key="entry.id"
        class="log-line"
        :class="levelClass(entry.level)"
      >
        <span class="log-time">{{ formatTime(entry.time) }}</span>
        <span class="log-level-tag">[{{ entry.level.toUpperCase() }}]</span>
        <span class="log-msg" v-html="highlightSearch(entry.message)"></span>
        <span v-if="entry.attrs && Object.keys(entry.attrs).length > 0" class="log-attrs">
          {{ formatAttrs(entry.attrs) }}
        </span>
      </div>
      <div v-if="displayEntries.length === 0" class="log-empty">
        暂无日志
      </div>
    </div>
  </div>
</template>

<style scoped>
.log-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1a2e;
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  color: #e0e0e0;
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  background: #16162a;
  border-bottom: 1px solid #2a2a4a;
  flex-shrink: 0;
  gap: 8px;
}

.log-levels {
  display: flex;
  gap: 4px;
}

.log-level-btn {
  padding: 2px 8px;
  border: 1px solid #333;
  border-radius: 3px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 10px;
  font-family: inherit;
  transition: all 0.15s;
}

.log-level-btn.active {
  border-color: #555;
}

.log-level-btn.log-level-debug.active { color: #888; border-color: #888; }
.log-level-btn.log-level-info.active { color: #e0e0e0; border-color: #e0e0e0; }
.log-level-btn.log-level-warn.active { color: #f0ad4e; border-color: #f0ad4e; }
.log-level-btn.log-level-error.active { color: #ef4444; border-color: #ef4444; }

.log-toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.log-search {
  padding: 2px 8px;
  border: 1px solid #333;
  border-radius: 3px;
  background: #0f0f1e;
  color: #e0e0e0;
  font-size: 11px;
  font-family: inherit;
  width: 160px;
  outline: none;
}

.log-search:focus {
  border-color: #555;
}

.log-clear-btn {
  padding: 2px 10px;
  border: 1px solid #333;
  border-radius: 3px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 10px;
  font-family: inherit;
}

.log-clear-btn:hover {
  color: #ef4444;
  border-color: #ef4444;
}

.log-entries {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}

.log-entries::-webkit-scrollbar {
  width: 6px;
}

.log-entries::-webkit-scrollbar-track {
  background: transparent;
}

.log-entries::-webkit-scrollbar-thumb {
  background: #333;
  border-radius: 3px;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 1px 8px;
  line-height: 1.5;
  white-space: nowrap;
  font-size: 11px;
}

.log-line:hover {
  background: rgba(255, 255, 255, 0.03);
}

.log-time {
  color: #666;
  flex-shrink: 0;
  width: 60px;
}

.log-level-tag {
  flex-shrink: 0;
  width: 52px;
  font-weight: 500;
}

.log-line.log-level-debug { color: #888; }
.log-line.log-level-info { color: #e0e0e0; }
.log-line.log-level-warn { color: #f0ad4e; }
.log-line.log-level-error { color: #ef4444; }

.log-msg {
  flex-shrink: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}

.log-attrs {
  color: #666;
  flex-shrink: 0;
  margin-left: auto;
}

.log-empty {
  padding: 20px;
  text-align: center;
  color: #555;
  font-style: italic;
}

:deep(.log-highlight) {
  background: rgba(240, 173, 78, 0.3);
  border-radius: 2px;
  padding: 0 1px;
}
</style>
```

- [ ] **Step 2: Write the panel test**

```typescript
// frontend/src/terminal/panels/__tests__/LogPanel.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LogPanel from '../LogPanel.vue'

describe('LogPanel', () => {
  it('renders without crashing', () => {
    const wrapper = mount(LogPanel, {
      props: { panelId: 'log-viewer' },
      global: {
        stubs: {
          transition: false,
          'transition-group': false,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无日志')
  })

  it('displays level filter buttons', () => {
    const wrapper = mount(LogPanel, {
      props: { panelId: 'log-viewer' },
    })
    const buttons = wrapper.findAll('.log-level-btn')
    expect(buttons.length).toBe(4)
    expect(buttons[0].text()).toBe('DEBUG')
    expect(buttons[1].text()).toBe('INFO')
    expect(buttons[2].text()).toBe('WARN')
    expect(buttons[3].text()).toBe('ERROR')
  })
})
```

- [ ] **Step 3: Register panel in registry**

Add to `frontend/src/terminal/panels/registry.ts` after the system panels section (line ~119):

```typescript
register('log-viewer', () => import('./LogPanel.vue'), { label: '日志面板', category: '系统', description: '实时系统日志' })
```

- [ ] **Step 4: Update panelToNode.ts (optional — LogPanel doesn't need workflow mapping)**

No changes needed — LogPanel is system-only, no workflow node equivalent.

- [ ] **Step 5: Verify**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit && npx vitest run`
Expected: All TypeScript clean, all tests PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/terminal/panels/LogPanel.vue frontend/src/terminal/panels/__tests__/LogPanel.test.ts frontend/src/terminal/panels/registry.ts
git commit -m "feat(frontend): add LogPanel.vue with level filtering and search"
```

---

### Task 7: 更新 CHANGELOG 和版本日期

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `frontend/package.json` (version)
- Modify: `README.md` (version badge if needed)

- [ ] **Step 1: Update CHANGELOG.md**

Add at the top under the latest version:

```markdown
## [2026.7.6] - 2026-07-06

### Added
- [Terminal] 日志面板：集成 stderr 日志到前端 LogPanel.vue，无需独立终端窗口
- [Engine] RingBuffer 日志环形缓冲区（5000 条容量），双写 stderr + 内存
- [Frontend] useLogger composable：轮询 GetLogs API 获取增量日志
```

- [ ] **Step 2: Update version in frontend/package.json**

Read current version and update if stale (should be `2026.7.6` or later).

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md frontend/package.json README.md
git commit -m "docs: update CHANGELOG for log panel integration"
```

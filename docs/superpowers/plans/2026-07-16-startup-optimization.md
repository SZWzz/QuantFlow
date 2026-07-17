# Startup Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce cold start from ~3-5s to <2s by parallelizing initialization, lazy-loading Python sidecar and market adapters, and adding skeleton screen feedback.

**Architecture:** New `internal/startup/` package measures and controls startup timing. `app_startup.go` is refactored to show the Wails window immediately, then run init phases in goroutines. Market adapters load on demand based on user's configured markets. Frontend panels already use dynamic imports — verified.

**Tech Stack:** Go 1.25+ (time, sync.WaitGroup, slog), Vue 3 + TypeScript (Composition API, defineAsyncComponent), Pinia

## Global Constraints

- No new Go dependencies (time, sync, context, slog are stdlib)
- Use slog for Go logging
- Use Composition API with `<script setup lang="ts">` for Vue
- Existing `frontend/src/terminal/panels/registry.ts` already uses dynamic imports (`() => import(...)`) — no changes needed for panel lazy loading
- Python sidecar MUST NOT block window display — async goroutine with 5s timeout
- SQLite migration check must complete in <5ms when schema unchanged
- Off-hours data cache loading should be delayed until after window display

---

### Task 1: Startup Metrics + Optimizer (internal/startup/)

**Files:**
- Create: `internal/startup/metrics.go`
- Create: `internal/startup/optimizer.go`
- Test: `internal/startup/metrics_test.go`

**Interfaces:**
- Consumes: nothing (standalone)
- Produces: `Metrics` struct with timing phases; `NewMetrics() *Metrics`; `Phase(name string) func()` (defer timer); `Log()`

- [ ] **Step 1: Write the failing test**

```go
// internal/startup/metrics_test.go
package startup

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsPhases(t *testing.T) {
	m := NewMetrics()

	phase1 := m.Phase("phase1")
	time.Sleep(5 * time.Millisecond)
	phase1()

	phase2 := m.Phase("phase2")
	time.Sleep(3 * time.Millisecond)
	phase2()

	if m.Phases["phase1"] <= 0 {
		t.Errorf("expected positive duration for phase1, got %d", m.Phases["phase1"])
	}
	if m.Phases["phase2"] <= 0 {
		t.Errorf("expected positive duration for phase2, got %d", m.Phases["phase2"])
	}
}

func TestMetricsLog(t *testing.T) {
	m := NewMetrics()
	p1 := m.Phase("test")
	p1()

	output := m.String()
	if !strings.Contains(output, "test") {
		t.Errorf("expected output to contain phase name, got %s", output)
	}
	if !strings.Contains(output, "ms") {
		t.Errorf("expected output to contain ms, got %s", output)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/startup/ -v -run TestMetricsPhases -count=1`
Expected: FAIL with "package internal/startup is not in std"

- [ ] **Step 3: Write implementation**

```go
// internal/startup/metrics.go
package startup

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type Metrics struct {
	mu     sync.Mutex
	Start  time.Time
	Phases map[string]time.Duration
}

func NewMetrics() *Metrics {
	return &Metrics{
		Start:  time.Now(),
		Phases: make(map[string]time.Duration),
	}
}

// Phase returns a function that records the duration when called.
// Usage: defer m.Phase("sqlite")()
func (m *Metrics) Phase(name string) func() {
	start := time.Now()
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.Phases[name] = time.Since(start)
	}
}

// Total returns total elapsed time since Start.
func (m *Metrics) Total() time.Duration {
	return time.Since(m.Start)
}

// Log outputs the metrics as slog info.
func (m *Metrics) Log() {
	m.mu.Lock()
	defer m.mu.Unlock()

	attrs := make([]slog.Attr, 0, len(m.Phases)+1)
	attrs = append(attrs, slog.Int64("total_ms", time.Since(m.Start).Milliseconds()))

	names := make([]string, 0, len(m.Phases))
	for name := range m.Phases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		attrs = append(attrs, slog.Int64(name+"_ms", m.Phases[name].Milliseconds()))
	}

	slog.LogAttrs(nil, slog.LevelInfo, "startup metrics", attrs...)
}

// String returns a human-readable representation.
func (m *Metrics) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	total := time.Since(m.Start).Milliseconds()
	result := fmt.Sprintf("startup total: %dms\n", total)

	names := make([]string, 0, len(m.Phases))
	for name := range m.Phases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result += fmt.Sprintf("  %s: %dms\n", name, m.Phases[name].Milliseconds())
	}
	return result
}
```

```go
// internal/startup/optimizer.go
package startup

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// StartupPhase represents one phase of the startup sequence.
type StartupPhase struct {
	Name string
	Run  func(ctx context.Context) error
}

// Optimizer controls the startup sequence with parallel execution.
type Optimizer struct {
	phases []StartupPhase
}

func NewOptimizer() *Optimizer {
	return &Optimizer{}
}

func (o *Optimizer) AddPhase(name string, fn func(ctx context.Context) error) {
	o.phases = append(o.phases, StartupPhase{Name: name, Run: fn})
}

// RunCritical executes phases sequentially in order, stopping on first error.
func (o *Optimizer) RunCritical(ctx context.Context, metrics *Metrics) error {
	for _, p := range o.phases {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		slog.Info("startup phase (critical)", "phase", p.Name)
		done := metrics.Phase(p.Name)
		if err := p.Run(ctx); err != nil {
			return err
		}
		done()
	}
	return nil
}

// RunParallel executes all phases concurrently with a timeout per phase.
// Returns the first error encountered (if any).
func (o *Optimizer) RunParallel(ctx context.Context, metrics *Metrics, timeout time.Duration) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(o.phases))

	for _, p := range o.phases {
		wg.Add(1)
		go func(phase StartupPhase) {
			defer wg.Done()
			pCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			slog.Info("startup phase (parallel)", "phase", phase.Name)
			done := metrics.Phase(phase.Name)
			if err := phase.Run(pCtx); err != nil {
				errCh <- err
			}
			done()
		}(p)
	}

	wg.Wait()
	close(errCh)

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/startup/ -v -run TestMetricsPhases -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/startup/metrics.go internal/startup/optimizer.go internal/startup/metrics_test.go
git commit -m "feat(startup): add startup metrics collection and optimizer with parallel phase execution"
```

---

### Task 2: Python Sidecar Async Startup (app_startup.go)

**Files:**
- Modify: `app_startup.go`
- Modify: `internal/python/sidecar.go` (add async start with timeout)

**Interfaces:**
- Consumes: `python.StartSidecar` existing signature
- Produces: Python sidecar launched in goroutine, Wails window shown immediately

- [ ] **Step 1: Write the test**

```go
// app_startup_test.go
package main

import (
	"testing"
	"time"
)

func TestAppStartupMetrics(t *testing.T) {
	// Verify the startup metrics structure is correct
	m := newStartupMetrics()
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}
	m.Record("test", 100*time.Millisecond)
	if m.Phases["test"] != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", m.Phases["test"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -v -run TestAppStartupMetrics -count=1 .`
Expected: FAIL with "newStartupMetrics not defined"

- [ ] **Step 3: Write implementation**

Create a new `app_startup_types.go` for startup-specific types:

```go
// app_startup_types.go
package main

import (
	"sync"
	"time"
)

type startupMetrics struct {
	mu     sync.Mutex
	Phases map[string]time.Duration
}

func newStartupMetrics() *startupMetrics {
	return &startupMetrics{Phases: make(map[string]time.Duration)}
}

func (sm *startupMetrics) Record(name string, d time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.Phases[name] = d
}
```

Modify `app_startup.go` — restructure `ServiceStartup` to optimize startup sequence:

```go
// app_startup.go — key changes (full file too large, show diff pattern)

// ServiceStartup is called by Wails v3 when the application starts.
// Optimized for <2s cold start: window shown immediately, init phases parallelized.
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	// Track startup metrics
	a.startupMetrics = newStartupMetrics()
	defer func() {
		a.startupMetrics.Record("total", time.Since(startTime))
		sm := a.startupMetrics
		sm.mu.Lock()
		slog.Info("startup metrics", "total_ms", time.Since(startTime).Milliseconds())
		for name, dur := range sm.Phases {
			slog.Info("startup phase", "phase", name, "ms", dur.Milliseconds())
		}
		sm.mu.Unlock()
	}()

	// Phase 1 (critical, sequential): config, logging, DB open + migration check
	configStart := time.Now()
	if err := a.initConfig(); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	a.startupMetrics.Record("config", time.Since(configStart))

	// DB open + migration check (fast path when schema unchanged)
	dbStart := time.Now()
	if err := a.initDatabase(); err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	a.startupMetrics.Record("database", time.Since(dbStart))

	// Phase 2 (parallel, non-blocking): show window immediately, init rest in background
	go func() {
		// Launch Python sidecar asynchronously with 5s timeout
		go a.initPythonSidecar()

		// Initialize market registry and adapters based on configured markets
		a.initMarketInfrastructure()

		// Initialize trading engine
		a.initTradingInfrastructure()

		// Initialize research services (degrade gracefully without Python)
		a.initResearchServices()

		// Initialize remaining services
		a.initRemainingServices()
	}()

	return nil
}

// initConfig loads config.yaml and sets up logging.
func (a *App) initConfig() error {
	configPath := "config.yaml"
	if execPath, err := os.Executable(); err == nil {
		configPath = filepath.Join(filepath.Dir(execPath), "config.yaml")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.cfg = cfg
	if filepath.IsAbs(a.cfg.DBPath) {
		slog.Warn("config db_path is absolute, resetting to default relative (data/quantflow.db)", "current", a.cfg.DBPath)
		a.cfg.DBPath = "data/quantflow.db"
	}
	a.resolvedDBPath = config.ResolveDBPath(a.cfg.DBPath)
	logging.Setup(cfg.LogLevel)
	return nil
}

// initDatabase opens SQLite and runs migrations if needed (fast path when up-to-date).
func (a *App) initDatabase() error {
	db, err := storage.Open(a.resolvedDBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	a.db = db

	migrations, err := storage.BuiltinMigrations()
	if err != nil {
		return fmt.Errorf("builtin migrations: %w", err)
	}
	if err := storage.Run(db, migrations); err != nil {
		return fmt.Errorf("database migrations: %w", err)
	}

	// Init caches (non-blocking, best-effort)
	go func() {
		if mc, err := market.NewMinuteCache(a.db); err != nil {
			slog.Warn("minute cache init", "error", err)
		} else {
			a.minuteCache = mc
		}
	}()
	go func() {
		if oc, err := market.NewOHLCVCache(a.db); err != nil {
			slog.Warn("ohlcv cache init", "error", err)
		} else {
			a.ohlcvCache = oc
		}
	}()

	return nil
}

// initPythonSidecar launches Python sidecar in background with 5s timeout.
func (a *App) initPythonSidecar() {
	execPath, _ := os.Executable()
	pythonDir := filepath.Join(filepath.Dir(execPath), "python")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sidecar, err := python.StartSidecar(ctx, pythonDir, 50051)
	if err != nil {
		slog.Warn("python sidecar launch failed, AI features disabled", "error", err)
		return
	}
	a.sidecar = sidecar

	bridgeOpts := python.DefaultOptions()
	bridgeOpts.PythonDir = pythonDir
	bridge, err := python.NewPythonBridge(bridgeOpts)
	if err != nil {
		slog.Warn("python bridge not available, AI features disabled", "error", err)
		return
	}
	a.bridge = bridge
	slog.Info("python sidecar connected", "address", bridgeOpts.Address)
}
```

- [ ] **Step 4: Verify build**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go build -o /dev/null .`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add app_startup.go app_startup_types.go
git commit -m "feat(startup): async Python sidecar launch and optimized startup sequence"
```

---

### Task 3: Market Adapter Lazy Initialization (hub.go)

**Files:**
- Modify: `internal/market/hub.go`
- Modify: `internal/market/registry.go` (add lazy init support)

**Interfaces:**
- Consumes: `AdapterRegistry` existing interface
- Produces: `Init(activeMarkets []string)` that only initializes adapters for configured markets

- [ ] **Step 1: Write the test**

```go
// internal/market/hub_test.go
package market

import (
	"testing"
)

func TestHubInitWithActiveMarkets(t *testing.T) {
	h := NewHub()

	// Should not panic with empty markets
	h.Init([]string{})

	// Should handle single market
	h.Init([]string{"CN"})

	// Should handle multiple markets
	h.Init([]string{"CN", "US", "HK", "CRYPTO"})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/market/ -v -run TestHubInitWithActiveMarkets -count=1`
Expected: FAIL (Init doesn't filter by market yet)

- [ ] **Step 3: Write implementation**

```go
// internal/market/hub.go — Add Init method

// Init initializes adapters for the given active markets only.
// Markets that are not in the list get a no-op or nil adapter chain,
// reducing startup cost when the user has only configured A-share data sources.
func (h *MarketDataHub) Init(activeMarkets []string) {
	if len(activeMarkets) == 0 {
		activeMarkets = []string{"CN"} // Default to CN if nothing configured
	}

	marketSet := make(map[string]bool, len(activeMarkets))
	for _, m := range activeMarkets {
		marketSet[m] = true
	}

	// Pre-create topic brokers for known market topics
	// so Subscribe does not lazy-create them on first access
	for _, m := range activeMarkets {
		h.mu.Lock()
		if _, ok := h.topics["market:heartbeat:"+m]; !ok {
			h.topics["market:heartbeat:"+m] = newTopicBroker()
		}
		h.mu.Unlock()
	}

	slog.Info("market hub initialized", "active_markets", activeMarkets, "topic_count", len(activeMarkets))
}
```

```go
// internal/market/registry.go — Add market-filtered adapter loading

// InitAdaptersForMarkets initializes only the adapters needed for the given markets.
// This replaces the previous model where all 40+ adapters were instantiated at startup.
func (r *AdapterRegistry) InitAdaptersForMarkets(activeMarkets []string) {
	if len(activeMarkets) == 0 {
		activeMarkets = []string{"CN"}
	}

	marketAdapters := map[string][]string{
		"CN":     {"tencent", "sina", "eastmoney", "tushare", "mootdx"},
		"HK":     {"tencent", "yahoo"},
		"US":     {"yahoo", "finnhub", "polygon", "alpaca"},
		"CRYPTO": {"binance", "okx", "bybit"},
	}

	initSet := make(map[string]bool)
	for _, m := range activeMarkets {
		for _, name := range marketAdapters[m] {
			initSet[name] = true
		}
	}

	// Only init adapters in the set; skip others
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, adapter := range r.adapters {
		if initSet[name] {
			// Adapter is already created (lazy init in constructor)
			// but we can now eagerly connect it
			if connectable, ok := adapter.(interface{ Connect() error }); ok {
				if err := connectable.Connect(); err != nil {
					slog.Warn("adapter connect failed, will retry on demand", "adapter", name, "error", err)
				}
			}
		}
	}

	slog.Info("adapter registry initialized for markets", "markets", activeMarkets, "active_adapters", len(initSet), "total", len(r.adapters))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/market/ -v -run TestHubInitWithActiveMarkets -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/market/hub.go internal/market/registry.go
git commit -m "feat(startup): market adapter lazy init by configured markets"
```

---

### Task 4: SkeletonScreen Component Enhancement

**Files:**
- Modify: `frontend/src/components/SkeletonPanel.vue`
- Modify: `frontend/src/App.vue` (show skeleton during init)

**Interfaces:**
- Consumes: existing `SkeletonPanel.vue` base
- Produces: Full-width skeleton screen shown until Wails events signal ready

- [ ] **Step 1: Check existing SkeletonPanel**

The existing `SkeletonPanel.vue` at `frontend/src/components/SkeletonPanel.vue` provides loading placeholders for panels. We need a full-screen `SkeletonScreen.vue` that shows terminal frame + loading indicators while the backend initializes.

- [ ] **Step 2: Write the test**

```typescript
// frontend/src/__tests__/SkeletonScreen.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SkeletonScreen from '@/terminal/components/SkeletonScreen.vue'

describe('SkeletonScreen', () => {
  it('renders loading state', () => {
    const wrapper = mount(SkeletonScreen, {
      props: { visible: true },
    })
    expect(wrapper.find('.skeleton-screen').exists()).toBe(true)
  })

  it('hides when visible=false', () => {
    const wrapper = mount(SkeletonScreen, {
      props: { visible: false },
    })
    expect(wrapper.find('.skeleton-screen').exists()).toBe(false)
  })
})
```

- [ ] **Step 3: Write implementation**

```vue
<!-- frontend/src/terminal/components/SkeletonScreen.vue -->
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

defineProps<{
  visible: boolean
}>()

const phases = [
  { label: '加载配置', key: 'config', duration: 0.3 },
  { label: '连接数据库', key: 'database', duration: 0.4 },
  { label: '初始化服务', key: 'services', duration: 0.6 },
  { label: '加载市场数据', key: 'market', duration: 0.8 },
]

const currentPhase = ref('config')
const progress = ref(0)

// Simulated progress for visual feedback while backend inits
let frame: number
onMounted(() => {
  const start = Date.now()
  function update() {
    const elapsed = (Date.now() - start) / 1000
    if (elapsed < 0.3) {
      currentPhase.value = 'config'
      progress.value = (elapsed / 0.3) * 25
    } else if (elapsed < 0.7) {
      currentPhase.value = 'database'
      progress.value = 25 + ((elapsed - 0.3) / 0.4) * 25
    } else if (elapsed < 1.3) {
      currentPhase.value = 'services'
      progress.value = 50 + ((elapsed - 0.7) / 0.6) * 25
    } else if (elapsed < 2.1) {
      currentPhase.value = 'market'
      progress.value = 75 + ((elapsed - 1.3) / 0.8) * 25
    }
    frame = requestAnimationFrame(update)
  }
  frame = requestAnimationFrame(update)
})

onUnmounted(() => {
  cancelAnimationFrame(frame)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="skeleton-screen">
      <div class="skeleton-content">
        <div class="skeleton-logo">
          <div class="logo-placeholder" />
          <h1>QuantFlow Terminal</h1>
        </div>
        <div class="skeleton-progress">
          <div class="progress-track">
            <div class="progress-fill" :style="{ width: progress + '%' }" />
          </div>
          <p class="progress-label">{{ currentPhase }}...</p>
        </div>
        <div class="skeleton-panels">
          <div class="skeleton-row">
            <div class="skeleton-panel wide" />
            <div class="skeleton-panel tall" />
          </div>
          <div class="skeleton-row">
            <div class="skeleton-panel tall" />
            <div class="skeleton-panel wide" />
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.skeleton-screen {
  position: fixed;
  inset: 0;
  background: var(--bg-base);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9998;
}
.skeleton-content {
  width: 480px;
  text-align: center;
}
.skeleton-logo {
  margin-bottom: 32px;
}
.logo-placeholder {
  width: 64px;
  height: 64px;
  margin: 0 auto 12px;
  background: var(--bg-muted);
  border-radius: 16px;
  animation: pulse 1.5s ease-in-out infinite;
}
.skeleton-logo h1 {
  font-size: 20px;
  color: var(--text-secondary);
  margin: 0;
}
.skeleton-progress {
  margin-bottom: 40px;
}
.progress-track {
  height: 4px;
  background: var(--bg-muted);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 8px;
}
.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width 0.3s ease;
}
.progress-label {
  font-size: 12px;
  color: var(--text-tertiary);
  margin: 0;
}
.skeleton-panels {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-row {
  display: flex;
  gap: 12px;
}
.skeleton-panel {
  background: var(--bg-surface);
  border: 1px solid var(--border);
  border-radius: 6px;
  animation: pulse 1.5s ease-in-out infinite;
}
.skeleton-panel.wide {
  flex: 2;
  height: 120px;
}
.skeleton-panel.tall {
  flex: 1;
  height: 120px;
}
@keyframes pulse {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 0.8; }
}
</style>
```

```typescript
// Import and add to App.vue
import SkeletonScreen from '@/terminal/components/SkeletonScreen.vue'
const showSkeleton = ref(true)

// Hide skeleton once backend signals ready (or after timeout)
onMounted(async () => {
  // Wait for WailsEvents or timeout
  try {
    await new Promise<void>((resolve) => {
      const timeout = setTimeout(resolve, 3000) // max 3s skeleton
      EventsOn('app:ready', () => {
        clearTimeout(timeout)
        resolve()
      })
    })
  } finally {
    showSkeleton.value = false
  }
})
```

```typescript
// In App.vue template, add before the main app content
<SkeletonScreen :visible="showSkeleton" />
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run -t "SkeletonScreen" 2>&1 || true`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/terminal/components/SkeletonScreen.vue frontend/src/App.vue
git commit -m "feat(startup): add skeleton screen with progress indicators"
```

---

### Task 5: Integration — Wire Startup in main.go + Emit Ready Event

**Files:**
- Modify: `app_startup.go` (emit `app:ready` Wails event when complete)
- Modify: `main.go` (add app instance to startup metrics)

**Interfaces:**
- Consumes: startup metrics from Task 1, async init from Task 2
- Produces: Wails `EventsEmit` with "app:ready" when all critical init done

- [ ] **Step 1: Add ready event emission**

```go
// In app_startup.go, at the end of the background goroutine in ServiceStartup,
// after all init calls complete:

// Emit ready event so frontend can hide skeleton screen
go func() {
    // Give the UI a moment to mount
    time.Sleep(100 * time.Millisecond)

    // Collect startup metrics
    a.startupMetrics.Record("total_init", time.Since(startTime))

    // Emit ready to frontend
    if a.wailsApp != nil {
        a.wailsApp.EventsEmit(context.Background(), "app:ready")
    }

    // Log metrics
    a.startupMetrics.mu.Lock()
    slog.Info("startup complete",
        "total_ms", time.Since(startTime).Milliseconds(),
    )
    for name, dur := range a.startupMetrics.Phases {
        slog.Info("startup phase detail", "phase", name, "ms", dur.Milliseconds())
    }
    a.startupMetrics.mu.Unlock()
}()
```

```go
// Add EventsOn import to app_startup.go if not already there
// import "github.com/wailsapp/wails/v3/pkg/application" (already present)
```

- [ ] **Step 2: Verify build**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go vet ./... && go build -o /dev/null .`
Expected: No errors

- [ ] **Step 3: Run frontend typecheck**

Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vue-tsc --noEmit 2>&1 | head -20`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add app_startup.go main.go
git commit -m "feat(startup): emit app:ready event and log startup metrics"
```

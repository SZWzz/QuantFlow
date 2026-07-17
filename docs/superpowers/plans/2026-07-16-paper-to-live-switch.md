# Paper→Live 实盘切换实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement safety-gated paper-to-live trading mode switch with red banner, safety checklist, and emergency shutdown.

**Architecture:** Add `TradingMode` type to `internal/trading/types.go`, wire mode enforcement into `Engine` and `RiskPipeline`, expose `SwitchToLive` IPC via `app_trading.go`, and render a permanent red banner on the frontend when live mode is active. Mode is persisted in SQLite.

**Tech Stack:** Go 1.25 (slog), Vue 3 `<script setup lang="ts">`, Pinia stores, SQLite WAL, Wails v3 IPC

## Global Constraints

- All Go tests use `package trading` (white-box) with table-driven patterns
- All frontend tests use `vitest` + `@vue/test-utils` with `setActivePinia(createPinia())` in `beforeEach`
- IPC bridge uses `(window as any)?.go?.main?.App` pattern with try/catch
- No `window.confirm()` or `window.alert()` — use `await confirmDialog(msg)` / `alertDialog(msg)` from `@/lib/wails`
- SQLite migrations numbered sequentially, never modified after deployment
- Module path: `quantflow` (from go.mod)

---

### Task 1: TradingMode and SafetyCheck types + test

**Files:**
- Modify: `internal/trading/types.go`
- Test: `internal/trading/types_test.go`

**Interfaces:**
- Consumes: nothing (new types)
- Produces: `TradingMode` type + constants, `SafetyCheck` struct, `SafetyReport` struct

- [ ] **Step 1: Write the failing test**

```go
// internal/trading/types_test.go
package trading

import "testing"

func TestTradingMode_Valid(t *testing.T) {
	tests := []struct {
		mode TradingMode
		want bool
	}{
		{TradingModePaper, true},
		{TradingModeLive, true},
		{TradingMode(""), false},
		{TradingMode("invalid"), false},
	}
	for _, tt := range tests {
		if got := tt.mode.Valid(); got != tt.want {
			t.Errorf("TradingMode(%q).Valid() = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestSafetyReport_Passed(t *testing.T) {
	r := SafetyReport{}
	if r.Passed() {
		t.Error("empty report should not pass")
	}
	r.Checks = []SafetyCheck{
		{Name: "Broker", OK: true, Blocking: true},
		{Name: "RiskRules", OK: true, Blocking: false},
	}
	if !r.Passed() {
		t.Error("all-ok report should pass")
	}
	r.Checks = append(r.Checks, SafetyCheck{Name: "APIKeys", OK: false, Blocking: true})
	if r.Passed() {
		t.Error("report with failing blocking check should not pass")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestTradingMode_Valid -v`
Expected: FAIL — `types.go` doesn't define `TradingMode.Valid()` or `SafetyReport`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/trading/types.go`:

```go
// TradingMode represents whether the engine is in paper or live trading mode.
type TradingMode string

const (
	TradingModeInvalid TradingMode = ""
	TradingModePaper   TradingMode = "paper"
	TradingModeLive    TradingMode = "live"
)

// Valid returns true if the mode is a known value.
func (m TradingMode) Valid() bool {
	return m == TradingModePaper || m == TradingModeLive
}

// SafetyCheck represents a single item in the pre-live safety checklist.
type SafetyCheck struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail,omitempty"`
	Blocking bool   `json:"blocking"` // if true, must pass for safe switch
}

// SafetyReport is the result of running all safety checks before going live.
type SafetyReport struct {
	Checks   []SafetyCheck `json:"checks"`
	PassedAt *string       `json:"passed_at,omitempty"` // ISO timestamp when passed
}

// Passed returns true if all blocking checks passed.
func (r *SafetyReport) Passed() bool {
	for _, c := range r.Checks {
		if c.Blocking && !c.OK {
			return false
		}
	}
	return len(r.Checks) > 0
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestTradingMode_Valid -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/types.go internal/trading/types_test.go
git commit -m "feat(trading): add TradingMode type and SafetyCheck/SafetyReport structs"
```

---

### Task 2: Engine mode management and safety checks + test

**Files:**
- Modify: `internal/trading/engine.go`
- Modify: `internal/trading/broker.go`
- Test: `internal/trading/engine_test.go`

**Interfaces:**
- Consumes: `TradingMode`, `SafetyReport`, `SafetyCheck` (from Task 1)
- Produces: `Engine.Mode() TradingMode`, `Engine.SetMode(m TradingMode) error`, `Engine.SafetyChecks() SafetyReport`, `Engine.EmergencyShutdown() error`, `Broker.CancelAllOrders(ctx) error`, `Broker.CloseAllPositions(ctx) error`

- [ ] **Step 1: Write the failing test**

Add to `internal/trading/engine_test.go`:

```go
func TestEngine_TradingMode(t *testing.T) {
	engine := NewEngine(100000.0)
	if engine.Mode() != TradingModePaper {
		t.Errorf("default mode = %q, want %q", engine.Mode(), TradingModePaper)
	}
	engine.SetMode(TradingModeLive)
	if engine.Mode() != TradingModeLive {
		t.Errorf("after SetMode(Live) = %q, want %q", engine.Mode(), TradingModeLive)
	}
	err := engine.SetMode("invalid")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
	if engine.Mode() != TradingModeLive {
		t.Errorf("mode should remain live after invalid set, got %q", engine.Mode())
	}
}

func TestEngine_SafetyChecks(t *testing.T) {
	engine := NewEngine(100000.0)
	report := engine.SafetyChecks()
	if report.Passed() {
		t.Error("safety report should not pass with no brokers configured")
	}
}

func TestEngine_EmergencyShutdown(t *testing.T) {
	engine := NewEngine(100000.0)
	engine.SetMode(TradingModeLive)
	err := engine.EmergencyShutdown()
	if err != nil {
		t.Fatalf("EmergencyShutdown error: %v", err)
	}
	if engine.Mode() != TradingModePaper {
		t.Errorf("after shutdown mode = %q, want %q", engine.Mode(), TradingModePaper)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_TradingMode -v`
Expected: FAIL — Engine doesn't have Mode(), SetMode(), SafetyChecks(), EmergencyShutdown()

- [ ] **Step 3: Write minimal implementation**

Modify `internal/trading/engine.go` — add to the `Engine` struct and add methods:

```go
// In the Engine struct, add:
type Engine struct {
	paperEngine *PaperEngine
	signalCh    chan Signal
	barCh       chan OHLCVBar
	done        chan struct{}
	mode        TradingMode
	brokers     map[string]Broker
}

// In NewEngine, add initialisation:
func NewEngine(initialCapital float64) *Engine {
	oms := NewOMS()
	riskConfig := DefaultRiskConfig()
	paperEngine := NewPaperEngine(oms, riskConfig, initialCapital)
	return &Engine{
		paperEngine: paperEngine,
		signalCh:    make(chan Signal, 256),
		barCh:       make(chan OHLCVBar, 1024),
		done:        make(chan struct{}),
		mode:        TradingModePaper,
		brokers:     make(map[string]Broker),
	}
}

// Add these methods after Done():
func (e *Engine) Mode() TradingMode { return e.mode }

func (e *Engine) SetMode(mode TradingMode) error {
	if !mode.Valid() {
		return fmt.Errorf("invalid trading mode: %q", mode)
	}
	e.mode = mode
	slog.Info("trading mode changed", "mode", mode)
	return nil
}

func (e *Engine) RegisterBroker(name string, broker Broker) {
	e.brokers[name] = broker
}

func (e *Engine) SafetyChecks() SafetyReport {
	var checks []SafetyCheck
	checks = append(checks, SafetyCheck{
		Name: "broker_configured", OK: len(e.brokers) > 0,
		Detail: fmt.Sprintf("%d broker(s) registered", len(e.brokers)), Blocking: true,
	})
	for name, broker := range e.brokers {
		connected := broker.IsConnected()
		detail := "connected"
		if !connected {
			detail = "disconnected"
		}
		checks = append(checks, SafetyCheck{
			Name: "broker_" + name, OK: connected,
			Detail: detail, Blocking: true,
		})
	}
	rc := e.paperEngine.riskPipeline.Config()
	checks = append(checks, SafetyCheck{
		Name: "risk_max_position_pct", OK: rc.MaxPositionPct > 0 && rc.MaxPositionPct <= 1.0,
		Detail: fmt.Sprintf("MaxPositionPct = %.1f%%", rc.MaxPositionPct*100), Blocking: false,
	})
	checks = append(checks, SafetyCheck{
		Name: "risk_max_drawdown", OK: rc.MaxDrawdownPct > 0 && rc.MaxDrawdownPct <= 1.0,
		Detail: fmt.Sprintf("MaxDrawdownPct = %.1f%%", rc.MaxDrawdownPct*100), Blocking: false,
	})
	return SafetyReport{Checks: checks}
}

func (e *Engine) EmergencyShutdown() error {
	slog.Warn("emergency shutdown initiated")
	for name, broker := range e.brokers {
		if broker.IsConnected() {
			ctx := context.Background()
			if err := broker.CancelAllOrders(ctx); err != nil {
				slog.Error("cancel orders during shutdown", "broker", name, "error", err)
			}
			if err := broker.CloseAllPositions(ctx); err != nil {
				slog.Error("close positions during shutdown", "broker", name, "error", err)
			}
		}
	}
	e.mode = TradingModePaper
	slog.Warn("emergency shutdown complete, reverted to paper mode")
	return nil
}
```

Add to `internal/trading/broker.go` — extend the Broker interface:

```go
type Broker interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	Name() string
	SubmitOrder(ctx context.Context, order *Order) (*BrokerOrderResult, error)
	CancelOrder(ctx context.Context, orderID string) error
	ModifyOrder(ctx context.Context, orderID string, newPrice, newQty float64) error
	GetOrders(ctx context.Context) ([]*Order, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	GetAccount(ctx context.Context) (*AccountInfo, error)
	CancelAllOrders(ctx context.Context) error
	CloseAllPositions(ctx context.Context) error
	OnOrderUpdate(func(order *Order))
	OnTradeUpdate(func(trade *Trade))
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_TradingMode -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/engine.go internal/trading/engine_test.go internal/trading/broker.go
git commit -m "feat(trading): add mode management, safety checks, emergency shutdown to Engine; extend Broker interface"
```

---

### Task 3: RiskPipeline live mode enforcement + PaperEngine live routing + test

**Files:**
- Modify: `internal/trading/risk_pipeline.go`
- Modify: `internal/trading/paper_engine.go`
- Test: `internal/trading/risk_pipeline_test.go`

**Interfaces:**
- Consumes: `RiskPipeline.Config()` (Task 2), `Engine.Mode()`
- Produces: `RiskPipeline.SetLiveMode(enabled bool)`, `PaperEngine.PlaceOrder()` routes to broker in live mode

- [ ] **Step 1: Write the failing test**

Add to `internal/trading/risk_pipeline_test.go`:

```go
func TestRiskPipeline_LiveModeConfig(t *testing.T) {
	rp := NewRiskPipeline(DefaultRiskConfig())
	rp.SetLiveMode(false)
	if rp.IsLiveMode() {
		t.Error("expected live mode false after SetLiveMode(false)")
	}
	rp.SetLiveMode(true)
	if !rp.IsLiveMode() {
		t.Error("expected live mode true after SetLiveMode(true)")
	}
	cfg := rp.Config()
	if cfg.MaxPositionPct != 0.25 {
		t.Errorf("Config() MaxPositionPct = %f, want 0.25", cfg.MaxPositionPct)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestRiskPipeline_LiveModeConfig -v`
Expected: FAIL — RiskPipeline doesn't have SetLiveMode, IsLiveMode, Config

- [ ] **Step 3: Write minimal implementation**

Add `liveMode bool` field to `RiskPipeline` struct, and add methods:

```go
// In RiskPipeline struct, add field:
type RiskPipeline struct {
	mu       sync.Mutex
	config   RiskConfig
	liveMode bool
}

// Add methods:
func (r *RiskPipeline) SetLiveMode(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.liveMode = enabled
}

func (r *RiskPipeline) IsLiveMode() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.liveMode
}

func (r *RiskPipeline) Config() RiskConfig {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.config
}
```

Modify `internal/trading/paper_engine.go` — add `broker` field and routing:

```go
type PaperEngine struct {
	oms            *OMS
	matcher        *OrderMatcher
	riskPipeline   *RiskPipeline
	portfolioValue float64
	broker         Broker
}

// SetBroker sets the live broker for live-mode order routing.
func (pe *PaperEngine) SetBroker(broker Broker) {
	pe.broker = broker
}

// PlaceOrder routes to broker in live mode, otherwise paper engine.
func (pe *PaperEngine) PlaceOrder(symbol string, side OrderSide, orderType OrderType, qty, price float64) (*Order, error) {
	if pe.riskPipeline.IsLiveMode() && pe.broker != nil {
		ctx := context.Background()
		order := &Order{
			Symbol:    symbol,
			Side:      side,
			OrderType: orderType,
			Quantity:  qty,
			Price:     price,
		}
		result, err := pe.broker.SubmitOrder(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("live broker submit failed: %w", err)
		}
		order.ID = result.BrokerOrderID
		order.Status = result.Status
		return order, nil
	}
	// Paper mode: use existing risk-checked path
	pos := pe.oms.GetPosition(symbol)
	tempOrder := &Order{Symbol: symbol, Side: side, OrderType: orderType, Quantity: qty, Price: price}
	if err := pe.riskPipeline.CheckOrder(tempOrder, pos, pe.portfolioValue); err != nil {
		return nil, fmt.Errorf("risk check failed: %w", err)
	}
	return pe.oms.PlaceOrder(symbol, side, orderType, "", qty, price)
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestRiskPipeline_LiveModeConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/risk_pipeline.go internal/trading/risk_pipeline_test.go internal/trading/paper_engine.go
git commit -m "feat(trading): add live mode flag and Config() to RiskPipeline; live broker routing in PaperEngine"
```

---

### Task 4: SwitchToLive IPC binding in app_trading.go + test

**Files:**
- Modify: `app_trading.go`
- Test: `app_trading_test.go` (new)

**Interfaces:**
- Consumes: `Engine.SwitchToLive(skipChecks bool)`, `Engine.SafetyChecks()`, `Engine.EmergencyShutdown()`
- Produces: `App.SwitchToLive(skipChecks bool) {report SafetyReport, err error}`, `App.SafetyStatus()`, `App.DoEmergencyShutdown()`

- [ ] **Step 1: Write the failing test**

```go
// app_trading_test.go
package main

import (
	"testing"
)

func TestSwitchToLive_RejectsWithoutBrokers(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	report, err := app.SwitchToLive(false)
	if err == nil {
		t.Fatal("expected error for no brokers configured")
	}
	if report == nil || len(report.Checks) == 0 {
		t.Error("expected non-empty safety report")
	}
}

func TestSafetyStatus_ReturnsCurrentMode(t *testing.T) {
	app := &App{engine: NewEngine(100000)}
	status := app.SafetyStatus()
	if status.Mode != "paper" {
		t.Errorf("expected paper mode, got %s", status.Mode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestSwitchToLive_RejectsWithoutBrokers -v`
Expected: FAIL — App has no SwitchToLive method

- [ ] **Step 3: Write minimal implementation**

Add to `app_trading.go`:

```go
// LiveModeStatus is the IPC response for current live mode state.
type LiveModeStatus struct {
	Mode        TradingMode `json:"mode"`
	InLiveMode  bool        `json:"in_live_mode"`
	LastReport  *SafetyReport `json:"last_report,omitempty"`
}

// SwitchToLive runs safety checks and if skipChecks=false, aborts if any
// blocking check fails. Returns the full SafetyReport.
func (a *App) SwitchToLive(skipChecks bool) (*SafetyReport, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	report := a.engine.SafetyChecks()
	if !skipChecks && !report.Passed() {
		return &report, fmt.Errorf("safety checks failed: %d blocking issue(s) remain",
			countBlockingFails(report.Checks))
	}
	a.engine.SetMode(TradingModeLive)
	now := time.Now().Format(time.RFC3339)
	report.PassedAt = &now
	return &report, nil
}

func countBlockingFails(checks []SafetyCheck) int {
	n := 0
	for _, c := range checks {
		if c.Blocking && !c.OK {
			n++
		}
	}
	return n
}

// SafetyStatus returns current mode and latest report.
func (a *App) SafetyStatus() *LiveModeStatus {
	if a.engine == nil {
		return &LiveModeStatus{Mode: "paper"}
	}
	return &LiveModeStatus{
		Mode:       a.engine.Mode(),
		InLiveMode: a.engine.Mode() == TradingModeLive,
	}
}

// DoEmergencyShutdown triggers emergency shutdown from the UI.
func (a *App) DoEmergencyShutdown() error {
	if a.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return a.engine.EmergencyShutdown()
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test -run TestSwitchToLive_RejectsWithoutBrokers -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add app_trading.go app_trading_test.go
git commit -m "feat(app): add SwitchToLive, SafetyStatus, DoEmergencyShutdown IPC bindings"
```

---

### Task 5: Frontend LiveModeBanner component + TradingJournalPanel + test

**Files:**
- Create: `frontend/src/terminal/components/LiveModeBanner.vue`
- Create: `frontend/src/terminal/panels/TradingJournalPanel.vue`
- Modify: `frontend/src/stores/terminal.ts`
- Test: `frontend/src/terminal/components/__tests__/LiveModeBanner.test.ts`

**Interfaces:**
- Consumes: `useTerminalStore().tradingMode`, `useTerminalStore().setTradingMode(mode)`, IPC `SwitchToLive`, `SafetyStatus`, `DoEmergencyShutdown`
- Produces: LiveModeBanner renders at top of terminal, TradingJournalPanel shows safety checklist + mode indicator

- [ ] **Step 1: Write the failing test**

```typescript
// frontend/src/terminal/components/__tests__/LiveModeBanner.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import LiveModeBanner from '../LiveModeBanner.vue'
import { useTerminalStore } from '@/stores/terminal'

describe('LiveModeBanner', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should not render in paper mode', () => {
    const store = useTerminalStore()
    store.tradingMode = 'paper'
    const wrapper = mount(LiveModeBanner)
    expect(wrapper.find('[data-testid="live-banner"]').exists()).toBe(false)
  })

  it('should render red banner in live mode', () => {
    const store = useTerminalStore()
    store.tradingMode = 'live'
    const wrapper = mount(LiveModeBanner)
    expect(wrapper.find('[data-testid="live-banner"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('LIVE')
  })

  it('should have emergency shutdown button in live mode', () => {
    const store = useTerminalStore()
    store.tradingMode = 'live'
    const wrapper = mount(LiveModeBanner)
    expect(wrapper.find('[data-testid="emergency-shutdown"]').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/components/__tests__/LiveModeBanner.test.ts`
Expected: FAIL — LiveModeBanner.vue and terminal store tradingMode don't exist yet

- [ ] **Step 3: Write minimal implementation**

`frontend/src/terminal/components/LiveModeBanner.vue`:

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { confirmDialog } from '@/lib/wails'

const store = useTerminalStore()
const visible = computed(() => store.tradingMode === 'live')

async function handleEmergencyShutdown() {
  const ok = await confirmDialog('确定平掉所有持仓？此操作不可撤销。')
  if (!ok) return
  try {
    const app = (window as any)?.go?.main?.App
    if (app?.DoEmergencyShutdown) {
      await app.DoEmergencyShutdown()
    }
    store.setTradingMode('paper')
  } catch (e) {
    console.error('[LiveModeBanner] emergency shutdown failed:', e)
  }
}
</script>

<template>
  <div v-if="visible" data-testid="live-banner" class="live-mode-banner">
    <span class="banner-icon">🔴</span>
    <span class="banner-text">LIVE MODE — 实盘交易中</span>
    <button
      data-testid="emergency-shutdown"
      class="emergency-btn"
      @click="handleEmergencyShutdown"
    >
      🛑 紧急平仓
    </button>
  </div>
</template>

<style scoped>
.live-mode-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 16px;
  background: #d32f2f;
  color: #fff;
  font-weight: 700;
  font-size: 13px;
  position: sticky;
  top: 0;
  z-index: 1000;
}
.banner-icon { font-size: 16px; }
.banner-text { flex: 1; }
.emergency-btn {
  background: #fff;
  color: #d32f2f;
  border: none;
  border-radius: 4px;
  padding: 2px 12px;
  cursor: pointer;
  font-weight: 600;
  font-size: 12px;
}
.emergency-btn:hover { background: #ffcdd2; }
</style>
```

`frontend/src/terminal/panels/TradingJournalPanel.vue`:

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTerminalStore } from '@/stores/terminal'
import { confirmDialog, alertDialog } from '@/lib/wails'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = useTerminalStore()
const report = ref<any>(null)
const loading = ref(false)

const isLive = computed(() => store.tradingMode === 'live')

async function runSafetyCheck() {
  loading.value = true
  try {
    const app = (window as any)?.go?.main?.App
    if (!app?.SafetyStatus) return
    const status = await app.SafetyStatus()
    store.setTradingMode(status.mode)
    if (!app?.SwitchToLive) return
    const r = await app.SwitchToLive(false)
    report.value = r
  } catch (e: any) {
    const app = (window as any)?.go?.main?.App
    if (app?.SafetyStatus) {
      const status = await app.SafetyStatus()
      report.value = status.last_report
    }
  } finally {
    loading.value = false
  }
}

async function forceSwitch() {
  const ok = await confirmDialog('安全检查未通过。强制切换将绕过未通过的项目，确认？')
  if (!ok) return
  try {
    const app = (window as any)?.go?.main?.App
    if (!app?.SwitchToLive) return
    const r = await app.SwitchToLive(true)
    report.value = r
    store.setTradingMode('live')
    await alertDialog('已切换到实盘模式。注意风险！')
  } catch (e) {
    console.error('[TradingJournal] force switch failed:', e)
  }
}

onMounted(runSafetyCheck)
</script>

<template>
  <div class="trading-journal-panel" data-testid="trading-journal-panel">
    <div class="mode-indicator">
      <span v-if="isLive" class="mode-live">🔴 LIVE MODE</span>
      <span v-else class="mode-paper">🟢 Paper Mode</span>
    </div>

    <div v-if="loading" class="section">检查中...</div>

    <div v-else-if="report" class="safety-report">
      <h3>安全检查报告</h3>
      <div
        v-for="check in report.checks"
        :key="check.name"
        :class="['check-item', check.ok ? 'pass' : 'fail']"
      >
        <span class="check-icon">{{ check.ok ? '✅' : '❌' }}</span>
        <span class="check-name">{{ check.name }}</span>
        <span class="check-detail">{{ check.detail }}</span>
      </div>
      <div class="actions">
        <button v-if="!isLive" class="btn-go-live" @click="forceSwitch">强制切换到实盘</button>
        <button class="btn-refresh" @click="runSafetyCheck">重新检查</button>
      </div>
    </div>

    <div v-else class="section">未找到安全检查报告。点击刷新运行检查。</div>
  </div>
</template>

<style scoped>
.trading-journal-panel { padding: 12px; }
.mode-indicator { margin-bottom: 12px; font-weight: 700; font-size: 14px; }
.mode-live { color: #d32f2f; }
.mode-paper { color: #388e3c; }
.safety-report h3 { margin: 0 0 8px; font-size: 14px; }
.check-item { display: flex; gap: 8px; padding: 4px 0; font-size: 13px; align-items: center; }
.check-item.pass { color: #388e3c; }
.check-item.fail { color: #d32f2f; }
.check-detail { color: #666; font-size: 12px; }
.actions { margin-top: 12px; display: flex; gap: 8px; }
.btn-go-live { background: #d32f2f; color: #fff; border: none; border-radius: 4px; padding: 6px 16px; cursor: pointer; }
.btn-refresh { background: #1976d2; color: #fff; border: none; border-radius: 4px; padding: 6px 16px; cursor: pointer; }
</style>
```

Modify `frontend/src/stores/terminal.ts` — add tradingMode state:

```typescript
// Add to the store's state:
const tradingMode = ref<string>('paper')
const lastSafetyReport = ref<any>(null)

// Add functions:
function setTradingMode(mode: string) {
  tradingMode.value = mode
}
function setSafetyReport(report: any) {
  lastSafetyReport.value = report
}

// Add to return object:
return {
  // ... existing returns ...
  tradingMode, lastSafetyReport,
  setTradingMode, setSafetyReport,
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow/frontend && npx vitest run src/terminal/components/__tests__/LiveModeBanner.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend/src/terminal/components/LiveModeBanner.vue frontend/src/terminal/panels/TradingJournalPanel.vue frontend/src/stores/terminal.ts
git commit -m "feat(frontend): add LiveModeBanner, TradingJournalPanel, tradingMode store"
```

---

### Task 6: Wire engine mode to PaperEngine risk pipeline + integration test

**Files:**
- Modify: `internal/trading/engine.go`
- Test: `internal/trading/engine_test.go`

**Interfaces:**
- Consumes: `RiskPipeline.SetLiveMode(bool)`, `Engine.mode` (Task 2)
- Produces: Engine.SetMode propagates to RiskPipeline, full integration test

- [ ] **Step 1: Write the failing test**

Add to `internal/trading/engine_test.go`:

```go
func TestEngine_ModePropagatesToRiskPipeline(t *testing.T) {
	engine := NewEngine(100000.0)
	rp := engine.GetPaperEngine().RiskPipeline()
	if rp.IsLiveMode() {
		t.Error("expected live mode false by default")
	}
	engine.SetMode(TradingModeLive)
	if !rp.IsLiveMode() {
		t.Error("expected live mode true after SetMode(Live)")
	}
	engine.SetMode(TradingModePaper)
	if rp.IsLiveMode() {
		t.Error("expected live mode false after SetMode(Paper)")
	}
}

func TestEngine_PlaceOrderRouting(t *testing.T) {
	// In paper mode, order goes through OMS
	engine := NewEngine(100000.0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go engine.Start(ctx)

	engine.SubmitSignal(Signal{
		Symbol: "AAPL", Direction: "buy", Quantity: 10, Timestamp: time.Now(),
	})
	engine.SubmitBar(OHLCVBar{Date: "2024-01-01", Symbol: "AAPL",
		Open: 195, High: 196, Low: 194, Close: 195.5})
	time.Sleep(20 * time.Millisecond)

	pos := engine.GetPaperEngine().GetOMS().GetPosition("AAPL")
	if pos == nil || pos.Quantity != 10 {
		t.Errorf("expected position qty 10 in paper mode, got %v", pos)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_ModePropagatesToRiskPipeline -v`
Expected: FAIL — SetMode doesn't propagate to RiskPipeline

- [ ] **Step 3: Write minimal implementation**

Modify `internal/trading/engine.go` — propagate mode in SetMode:

```go
func (e *Engine) SetMode(mode TradingMode) error {
	if !mode.Valid() {
		return fmt.Errorf("invalid trading mode: %q", mode)
	}
	e.mode = mode
	e.paperEngine.riskPipeline.SetLiveMode(mode == TradingModeLive)
	slog.Info("trading mode changed", "mode", mode)
	return nil
}

// Also ensure Engine.EmergencyShutdown also resets RiskPipeline:
func (e *Engine) EmergencyShutdown() error {
	slog.Warn("emergency shutdown initiated")
	for name, broker := range e.brokers {
		if broker.IsConnected() {
			ctx := context.Background()
			if err := broker.CancelAllOrders(ctx); err != nil {
				slog.Error("cancel orders during shutdown", "broker", name, "error", err)
			}
			if err := broker.CloseAllPositions(ctx); err != nil {
				slog.Error("close positions during shutdown", "broker", name, "error", err)
			}
		}
	}
	e.mode = TradingModePaper
	e.paperEngine.riskPipeline.SetLiveMode(false)
	slog.Warn("emergency shutdown complete, reverted to paper mode")
	return nil
}
```

Add `RiskPipeline()` accessor to `PaperEngine`:

```go
func (pe *PaperEngine) RiskPipeline() *RiskPipeline {
	return pe.riskPipeline
}
```

- [ ] **Step 4: Run test to verify it passes**
Run: `cd /Volumes/shenzy/vibe_coding/QuantFlow && go test ./internal/trading/ -run TestEngine_ModePropagatesToRiskPipeline -v`
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/trading/engine.go internal/trading/engine_test.go internal/trading/paper_engine.go
git commit -m "feat(trading): propagate SetMode to RiskPipeline; add RiskPipeline() accessor"
```

---

### Execution Order

```
Task 1 (types) → Task 2 (engine + broker) → Task 3 (risk pipeline + paper engine)
  → Task 4 (IPC bindings) → Task 5 (frontend) → Task 6 (integration wiring)
```

Tasks 1-4 are sequential (each builds on the previous). Task 5 can begin after Task 2 (needs `SwitchToLive` IPC). Task 6 wires everything together.

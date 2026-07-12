# Product Usability — A+B Hybrid Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 3 core flow breaks (order routing, backtest dead-end, broker status) and wire 5 remaining mock panels to make QuantFlow daily-usable.

**Architecture:** Go backend (Wails v3 auto-exposes 139 `*App` methods) → Vue 3 frontend (calls via `window.go.main.App.MethodName()` Proxy shim). All new wiring follows existing patterns: Go adds/modifies an exported method, frontend calls it via the existing Proxy.

**Tech Stack:** Go 1.25+ (Wails v3), Vue 3 + TypeScript (Composition API), Python 3.12+ (gRPC sidecar), SQLite WAL

## Global Constraints

- No new dependencies in Go or frontend
- Follow existing naming patterns (slog for logging, Composition API for Vue)
- Each task ends with a passing test or manual verification
- Commit each task independently
- Update CHANGELOG.md with each commit
- No `window.confirm()` / `window.alert()` — use `confirmDialog`/`alertDialog` from `@/lib/wails`

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `app_trading.go` | Wails API: order routing, broker status, backtest | Modify |
| `internal/trading/oms.go` | Order matching, broker delegation | Modify |
| `internal/trading/broker.go` | Broker interface (read-only reference) | — |
| `internal/trading/brokers/` | Broker implementations (ibkr, binance, futu, alpaca) | Read for BrokerStatus wiring |
| `frontend/src/terminal/panels/OrderEntryPanel.vue` | Order form UI | Modify |
| `frontend/src/terminal/panels/BrokerStatusPanel.vue` | Broker status cards | Modify |
| `frontend/src/lib/composables/useBrokerStatus.ts` | Broker status fetching logic | — (already correct) |
| `frontend/src/terminal/panels/BrokerConfig.vue` | Broker API key management | Modify |
| `frontend/src/terminal/panels/BasketOrderPanel.vue` | Basket order builder | Modify |
| `frontend/src/terminal/panels/ChanlunPanel.vue` | Chanlun analysis | Modify |
| `frontend/src/terminal/panels/FactorAnalysisPanel.vue` | Factor catalog | Modify |
| `frontend/src/terminal/panels/StockScannerPanel.vue` | Stock scanner | Modify |
| `frontend/src/terminal/panels/BacktestPanel.vue` | Backtest list/detail | Modify (remove RunBacktest button) |
| `python/conftest.py` | Pytest path setup | Create |
| `internal/market/adapters/gdelt_test.go` | GDELT tests | Modify |
| `docs/specs/2026-06-18-proposal-implementation-status.md` | Status doc | Modify |
| `CHANGELOG.md` | Changelog | Modify (every commit) |

---

## Phase 1: Fix Order Flow

### Task 1.1: Fix market detection in OrderEntryPanel fetchQuote

**Files:**
- Modify: `frontend/src/terminal/panels/OrderEntryPanel.vue:26-43`

**Interfaces:**
- Consumes: `(window as any).go.main.App.GetQuote(market, symbol)` — existing Go method at `app_market.go:172`
- Produces: market-aware quote fetching for any symbol

- [ ] **Step 1: Create a market detection helper**

Add a `detectMarket` function to OrderEntryPanel.vue that infers market from symbol format.

```typescript
// Add after the broker ref (line 16), before the estimatedTotal computed
function detectMarket(sym: string): string {
  // Crypto: explicit pairs like BTC/USDT, ETH/USDT
  if (sym.includes('/')) return 'CRYPTO'
  // US: 1-5 uppercase letters (AAPL, TSLA, BRK.A)
  if (/^[A-Z]{1,5}(\.[A-Z])?$/.test(sym)) return 'US'
  // HK: 4-5 digits with leading zeros (00001, 00700)
  if (/^\d{4,5}$/.test(sym)) return 'HK'
  // Default: CN (6 digits, or any other format)
  return 'CN'
}
```

- [ ] **Step 2: Update fetchQuote to use detectMarket**

Replace the hardcoded `'CN'` parameter.

```typescript
async function fetchQuote() {
  const app = (window as any).go?.main?.App
  if (!app?.GetQuote) return
  quoteLoading.value = true
  loadError.value = ''
  try {
    const market = detectMarket(symbol.value)
    const { data: result } = await fetchWithCache<any>(`quote:${symbol.value}`, () => app.GetQuote(market, symbol.value), 60 * 1000)
    const quote = Array.isArray(result) ? result[0] : result
    if (quote?.last) {
      lastPrice.value = quote.last
      price.value = quote.last
    }
  } catch (e: any) {
    loadError.value = e?.message || String(e)
  } finally {
    quoteLoading.value = false
  }
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit src/terminal/panels/OrderEntryPanel.vue 2>&1 | tail -5
```

Expected: No errors related to OrderEntryPanel.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/OrderEntryPanel.vue CHANGELOG.md
git commit -m "fix(frontend): OrderEntryPanel market detection — replace hardcoded 'CN' with detectMarket()"
```

---

### Task 1.2: Add brokerName parameter to PlaceOrder in Go

**Files:**
- Modify: `app_trading.go:13-30`
- Modify: `internal/trading/oms.go:97-122`
- Modify: `frontend/src/terminal/panels/OrderEntryPanel.vue:53-65`

**Interfaces:**
- Consumes: `trading.Broker` interface (`internal/trading/broker.go:9-25`), `oms.PlaceOrder` existing signature
- Produces: `PlaceOrder(symbol, side, orderType, brokerName string, qty, price float64) (*trading.Order, error)` — brokerName routed through OMS

- [ ] **Step 1: Update OMS.PlaceOrder signature and add broker routing**

Modify `internal/trading/oms.go`, updating the `PlaceOrder` method to accept `brokerName` and route to the live broker when specified.

```go
// PlaceOrder creates and registers a new order. If brokerName is non-empty and
// not "paper", routes the order through the attached live broker.
func (o *OMS) PlaceOrder(symbol string, side OrderSide, orderType OrderType, brokerName string, qty, price float64) (*Order, error) {
	if qty <= 0 {
		return nil, fmt.Errorf("quantity must be positive, got %f", qty)
	}
	if orderType == TypeLimit && price <= 0 {
		return nil, fmt.Errorf("limit order requires a positive price")
	}

	// Route to live broker if one is attached and brokerName is specified.
	if brokerName != "" && brokerName != "paper" {
		o.mu.RLock()
		br := o.broker
		o.mu.RUnlock()
		if br == nil {
			return nil, fmt.Errorf("broker %q not attached", brokerName)
		}
		if br.Name() != brokerName {
			return nil, fmt.Errorf("broker %q not attached (active: %s)", brokerName, br.Name())
		}
		order := &Order{
			ID:        uuid.New().String()[:12],
			Symbol:    symbol,
			Side:      side,
			OrderType: orderType,
			Quantity:  qty,
			Price:     price,
			Status:    StatusPending,
			PlacedAt:  time.Now(),
		}
		ctx := context.Background()
		result, err := br.SubmitOrder(ctx, order)
		if err != nil {
			order.Status = StatusRejected
			o.mu.Lock()
			o.orders[order.ID] = order
			o.mu.Unlock()
			return order, fmt.Errorf("broker submit: %w", err)
		}
		order.ID = result.BrokerOrderID
		order.Status = result.Status
		o.mu.Lock()
		o.orders[order.ID] = order
		o.mu.Unlock()
		return order, nil
	}

	// Paper trading path (existing logic).
	o.mu.Lock()
	defer o.mu.Unlock()

	order := &Order{
		ID:        uuid.New().String()[:12],
		Symbol:    symbol,
		Side:      side,
		OrderType: orderType,
		Quantity:  qty,
		Price:     price,
		Status:    StatusPending,
		PlacedAt:  time.Now(),
	}
	order.Name = o.getName(symbol)
	o.orders[order.ID] = order
	return order, nil
}
```

Note: `PlaceOrderLive` (lines 438-487) can now be deprecated since `PlaceOrder` handles broker routing, but keep it for backwards compatibility.

- [ ] **Step 2: Update App.PlaceOrder to pass brokerName through**

Modify `app_trading.go` lines 13-30.

```go
// PlaceOrder submits an order to the trading engine (paper or live broker).
func (a *App) PlaceOrder(symbol, side, orderType, brokerName string, qty, price float64) (*trading.Order, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}

	// Configure daily price limit from cached last close.
	if a.lastClose != nil {
		if prevClose, ok := a.lastClose[symbol]; ok && prevClose > 0 {
			ratio := 0.10
			if strings.HasPrefix(symbol, "300") || strings.HasPrefix(symbol, "301") || strings.HasPrefix(symbol, "688") {
				ratio = 0.20
			}
			a.oms.SetPriceLimit(symbol, prevClose, ratio)
		}
	}

	return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), brokerName, qty, price)
}
```

- [ ] **Step 3: Run existing OMS tests to verify no regression**

```bash
go test ./internal/trading/... -v -count=1 -run TestOMS 2>&1 | tail -20
```

Expected: All OMS tests pass.

- [ ] **Step 4: Update frontend OrderEntryPanel to pass broker**

Modify `frontend/src/terminal/panels/OrderEntryPanel.vue` lines 53-65.

```typescript
function placeOrder() {
  try {
    const app = (window as any).go?.main?.App
    if (app?.PlaceOrder) {
      app.PlaceOrder(
        symbol.value, side.value, orderType.value, broker.value,
        quantity.value,
        orderType.value === 'market' ? 0 : price.value
      )
    }
  } catch (e) {
    console.warn('PlaceOrder not available:', e)
  }
}
```

Note: The `broker` ref is already typed as `ref<'paper' | 'binance' | 'futu'>('paper')` (line 16). Adding IBKR and Alpaca to the broker options is covered in Task 1.4.

- [ ] **Step 5: Verify TypeScript compiles and frontend tests pass**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
cd frontend && npx vitest run 2>&1 | tail -5
```

Expected: No new TS errors. All 198 tests pass.

- [ ] **Step 6: Commit**

```bash
git add app_trading.go internal/trading/oms.go frontend/src/terminal/panels/OrderEntryPanel.vue CHANGELOG.md
git commit -m "feat(trading): add brokerName parameter to PlaceOrder — route to live broker when specified"
```

---

### Task 1.3: Wire BrokerStatusPanel to probe real brokers

**Files:**
- Modify: `app_trading.go:86-100`

**Interfaces:**
- Consumes: `trading.Broker` interface `Name()` and `IsConnected()` methods
- Produces: `GetBrokerStatuses() []BrokerStatus` — returns real broker states

- [ ] **Step 1: Update GetBrokerStatuses to probe registered brokers**

Replace the hardcoded implementation in `app_trading.go` lines 96-100.

```go
// GetBrokerStatuses returns connection status of all registered brokers.
func (a *App) GetBrokerStatuses() []BrokerStatus {
	statuses := []BrokerStatus{
		{Name: "paper", Label: "Paper Trading", Market: "模拟", Connected: true, Detail: "本地模拟撮合"},
	}

	// Probe registered live brokers.
	brokerNames := []string{"futu", "binance", "alpaca", "ibkr"}
	brokerLabels := map[string]string{
		"futu":    "富途牛牛",
		"binance": "Binance",
		"alpaca":  "Alpaca",
		"ibkr":    "Interactive Brokers",
	}
	brokerMarkets := map[string]string{
		"futu":    "港股/A股/美股",
		"binance": "加密",
		"alpaca":  "美股",
		"ibkr":    "全球",
	}

	for _, name := range brokerNames {
		br := a.brokerByName(name)
		connected := false
		detail := "未配置"
		if br != nil {
			connected = br.IsConnected()
			if connected {
				detail = "已连接"
			} else {
				detail = "已配置，未连接"
			}
		}
		label := brokerLabels[name]
		if label == "" {
			label = name
		}
		statuses = append(statuses, BrokerStatus{
			Name:      name,
			Label:     label,
			Market:    brokerMarkets[name],
			Connected: connected,
			Detail:    detail,
		})
	}

	return statuses
}
```

- [ ] **Step 2: Add brokerByName helper to app.go or app_trading.go**

Search for the broker registry in the App struct. If there's a map of broker instances, add a lookup method. If not, add a simple lookup.

Check if `a.brokers` map exists:

```bash
grep -n "brokers" app.go | head -10
```

If a broker registry exists, use it. Otherwise, add a minimal `brokerByName` method:

```go
// brokerByName looks up a broker instance by name. Returns nil if not registered.
func (a *App) brokerByName(name string) trading.Broker {
	if a.brokers == nil {
		return nil
	}
	return a.brokers[name]
}
```

If no broker registry exists in the App struct, first check how brokers are instantiated:

```bash
grep -rn "NewFutuBroker\|NewBinanceBroker\|NewAlpacaBroker\|NewIBKRBroker" --include="*.go" . | grep -v _test.go
```

Then add a `brokers map[string]trading.Broker` field to the App struct and populate it during startup. If this is too invasive for this task, fall back to returning status for paper + a note that live broker config is done via BrokerConfig panel.

- [ ] **Step 3: Run Go tests**

```bash
go test ./... -count=1 2>&1 | tail -5
```

Expected: 981+ pass, no new failures.

- [ ] **Step 4: Commit**

```bash
git add app_trading.go app.go CHANGELOG.md
git commit -m "feat(trading): BrokerStatusPanel probes real broker connection states"
```

---

### Task 1.4: Add IBKR/Alpaca to OrderEntryPanel broker dropdown

**Files:**
- Modify: `frontend/src/terminal/panels/OrderEntryPanel.vue:16,81-86`

**Interfaces:**
- Consumes: `broker` ref type already supports `'paper' | 'binance' | 'futu'`
- Produces: Extended broker type with ibkr/alpaca options

- [ ] **Step 1: Extend broker type and add dropdown options**

```typescript
// Line 16: extend type
const broker = ref<'paper' | 'binance' | 'futu' | 'ibkr' | 'alpaca'>('paper')
```

```html
<!-- Lines 81-86: add ibkr and alpaca options -->
<select v-model="broker" class="form-input">
  <option value="paper">{{ $t('trade.paper') }}</option>
  <option value="binance">{{ $t('trade.binance') }}</option>
  <option value="futu">{{ $t('trade.futu') }}</option>
  <option value="ibkr">IBKR</option>
  <option value="alpaca">Alpaca</option>
</select>
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/terminal/panels/OrderEntryPanel.vue CHANGELOG.md
git commit -m "feat(frontend): add IBKR and Alpaca to OrderEntryPanel broker dropdown"
```

---

## Phase 2: Fix Review Flow

### Task 2.1: Remove RunBacktest dead-end API

**Files:**
- Modify: `app_trading.go:102-106`

**Interfaces:**
- Consumes: BacktestPanel may reference `RunBacktest`
- Produces: Clean removal — backtesting unified under workflow path

- [ ] **Step 1: Check if RunBacktest is called from frontend**

```bash
grep -r "RunBacktest" frontend/src/ --include="*.vue" --include="*.ts"
```

Expected: No frontend calls to `RunBacktest`. If found, those calls will be updated in Step 2.

- [ ] **Step 2: Remove RunBacktest from app_trading.go**

Delete lines 102-106.

- [ ] **Step 3: Run Go tests**

```bash
go test ./... -count=1 2>&1 | tail -5
```

Expected: All tests pass. The RunBacktest method was never tested (returned error always).

- [ ] **Step 4: Commit**

```bash
git add app_trading.go CHANGELOG.md
git commit -m "refactor(trading): remove dead RunBacktest API — unified under workflow backtest path"
```

---

### Task 2.2: TradeHistory/PortfolioSummary panels — verify data chain works

**Files:**
- Read only (verification): `app_trading.go:32-54,56-79`
- Read only: `frontend/src/stores/portfolio.ts`
- Read only: `frontend/src/terminal/panels/TradeHistory.vue`
- Read only: `frontend/src/terminal/panels/PortfolioSummary.vue`

**Interfaces:**
- Consumes: `GetPositions()`, `GetOrders()`, `GetTrades()`, `GetPortfolioSummary()` — all exist
- Produces: Verified data chain from paper OMS to frontend panels

- [ ] **Step 1: Verify the complete data chain works end-to-end**

Start the app in dev mode and verify:
1. Place a paper trade via OrderEntryPanel
2. Check PositionPanel shows the position
3. Check TradeHistory shows the trade
4. Check PortfolioSummary shows updated totals

This is a manual verification task — no code changes needed. The data chain from Paper OMS → Go API → Pinia store → Vue panels is already wired.

If the chain is broken (e.g., portfolio store not auto-refreshing), fix the specific issue found. The expected fix would be in the store's `fetch*` methods.

- [ ] **Step 2: Document findings and commit any fixes**

```bash
git add CHANGELOG.md
git commit -m "docs: verify TradeHistory/PortfolioSummary data chain — paper trading path confirmed"
```

---

## Phase 3: Quality Cleanup

### Task 3.1: Fix Python test collection

**Files:**
- Create: `python/conftest.py`
- Read: `python/pyproject.toml` (verify pytest config)

**Interfaces:**
- Produces: `python -m pytest tests/` collects and runs tests

- [ ] **Step 1: Create conftest.py with path setup**

```python
"""Pytest configuration — add src/ to sys.path for imports."""
import sys
from pathlib import Path

# Add python/src to the import path so tests can import from src.*
_src = Path(__file__).resolve().parent / "src"
if str(_src) not in sys.path:
    sys.path.insert(0, str(_src))
```

- [ ] **Step 2: Run test collection to verify**

```bash
cd python && python -m pytest tests/ --collect-only -q 2>&1 | tail -30
```

Expected: Lists test functions from all 20 test files.

- [ ] **Step 3: Run tests to check pass/fail status**

```bash
cd python && python -m pytest tests/ -x -q 2>&1 | tail -30
```

Expected: Tests either pass or skip (some may fail due to missing optional dependencies like grpcio-health-checking — those are pre-existing issues, not regressions).

- [ ] **Step 4: Commit**

```bash
git add python/conftest.py CHANGELOG.md
git commit -m "fix(python): add conftest.py — fix test collection by adding src/ to sys.path"
```

---

### Task 3.2: Fix GDELT test rate-limit failures

**Files:**
- Modify: `internal/market/adapters/gdelt_test.go`

**Interfaces:**
- Consumes: GDELT adapter public API
- Produces: Tests pass or auto-skip on rate limit

- [ ] **Step 1: Add rate-limit detection helper**

Add a helper function at the top of the test file that detects HTTP 429 responses and skips the test.

```go
// skipIfRateLimited calls t.Skip if err indicates an HTTP 429 rate limit.
func skipIfRateLimited(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "429") {
		t.Skip("GDELT API rate limited (HTTP 429), skipping integration test")
	}
}
```

Add `"strings"` to imports.

- [ ] **Step 2: Update TestGDELTAdapter_FetchTopicVolume and FetchTopicTone**

Wrap the error check with `skipIfRateLimited`:

```go
func TestGDELTAdapter_FetchTopicVolume(t *testing.T) {
	a := NewGDELTAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("GDELT API not reachable")
	}

	points, err := a.FetchTopicVolume(ctx, "taiwan-strait", "7d")
	skipIfRateLimited(t, err)
	if err != nil {
		t.Fatalf("FetchTopicVolume error: %v", err)
	}
	t.Logf("got %d volume points", len(points))
	if len(points) > 0 {
		t.Logf("first point: date=%s value=%.2f", points[0].Date, points[0].Value)
	}
}

func TestGDELTAdapter_FetchTopicTone(t *testing.T) {
	a := NewGDELTAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	if !a.IsAvailable(ctx) {
		t.Skip("GDELT API not reachable")
	}

	points, err := a.FetchTopicTone(ctx, "taiwan-strait", "7d")
	skipIfRateLimited(t, err)
	if err != nil {
		t.Fatalf("FetchTopicTone error: %v", err)
	}
	t.Logf("got %d tone points", len(points))
	if len(points) > 0 {
		t.Logf("first point: date=%s tone=%.2f", points[0].Date, points[0].Tone)
	}
}
```

- [ ] **Step 3: Run GDELT tests**

```bash
go test ./internal/market/adapters/ -v -count=1 -run TestGDELT 2>&1 | tail -20
```

Expected: Tests pass or skip (no FAIL).

- [ ] **Step 4: Commit**

```bash
git add internal/market/adapters/gdelt_test.go CHANGELOG.md
git commit -m "fix(test): GDELT tests auto-skip on HTTP 429 rate limit"
```

---

### Task 3.3: Wire BrokerConfig to credential store

**Files:**
- Modify: `frontend/src/terminal/panels/BrokerConfig.vue:1-50`

**Interfaces:**
- Consumes: `app.SaveCredential(name, credType, keys)`, `app.ListCredentials()`, `app.ListCredentialNames()`
- Produces: Working broker config with persistence

- [ ] **Step 1: Replace alert stubs with real Go calls**

```typescript
<script setup lang="ts">
import { ref, onMounted } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const broker = ref<'binance' | 'futu' | 'ibkr' | 'alpaca'>('binance')
const binanceKey = ref('')
const binanceSecret = ref('')
const binanceTestnet = ref(true)
const futuHost = ref('localhost')
const futuPort = ref(11111)
const ibkrHost = ref('localhost')
const ibkrPort = ref(7497)
const ibkrClientId = ref(1)
const alpacaKey = ref('')
const alpacaSecret = ref('')
const alpacaPaper = ref(true)
const connectionStatus = ref<'unknown' | 'connected' | 'disconnected'>('unknown')
const savedMsg = ref('')

async function testConnection() {
  connectionStatus.value = 'unknown'
  try {
    const app = (window as any).go?.main?.App
    if (!app?.TestBrokerConnection) {
      connectionStatus.value = 'disconnected'
      return
    }
    const config = getCurrentConfig()
    const result = await app.TestBrokerConnection(broker.value, config)
    connectionStatus.value = result?.connected ? 'connected' : 'disconnected'
  } catch (e: any) {
    console.warn('TestConnection failed:', e)
    connectionStatus.value = 'disconnected'
  }
}

function getCurrentConfig(): Record<string, string> {
  switch (broker.value) {
    case 'binance':
      return { api_key: binanceKey.value, secret_key: binanceSecret.value, testnet: String(binanceTestnet.value) }
    case 'futu':
      return { host: futuHost.value, port: String(futuPort.value) }
    case 'ibkr':
      return { host: ibkrHost.value, port: String(ibkrPort.value), client_id: String(ibkrClientId.value) }
    case 'alpaca':
      return { api_key: alpacaKey.value, secret_key: alpacaSecret.value, paper: String(alpacaPaper.value) }
    default:
      return {}
  }
}

async function saveConfig() {
  savedMsg.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.SaveCredential) return
    const config = getCurrentConfig()
    await app.SaveCredential(`broker_${broker.value}`, broker.value, config)
    savedMsg.value = '配置已保存'
    setTimeout(() => { savedMsg.value = '' }, 3000)
  } catch (e: any) {
    savedMsg.value = `保存失败: ${e?.message || e}`
  }
}

async function loadConfig() {
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetCredential) return
    const cred = await app.GetCredential(`broker_${broker.value}`)
    if (!cred?.keys) return
    const k = cred.keys
    switch (broker.value) {
      case 'binance':
        binanceKey.value = k.api_key || ''; binanceSecret.value = k.secret_key || ''; binanceTestnet.value = k.testnet !== 'false'; break
      case 'futu':
        futuHost.value = k.host || 'localhost'; futuPort.value = parseInt(k.port || '11111'); break
      case 'ibkr':
        ibkrHost.value = k.host || 'localhost'; ibkrPort.value = parseInt(k.port || '7497'); ibkrClientId.value = parseInt(k.client_id || '1'); break
      case 'alpaca':
        alpacaKey.value = k.api_key || ''; alpacaSecret.value = k.secret_key || ''; alpacaPaper.value = k.paper !== 'false'; break
    }
  } catch (e: any) {
    console.warn('LoadConfig failed:', e)
  }
}

onMounted(loadConfig)
</script>
```

- [ ] **Step 2: Update template to include IBKR and Alpaca sections**

Add to the broker select:
```html
<option value="ibkr">Interactive Brokers</option>
<option value="alpaca">Alpaca</option>
```

Add IBKR config section:
```html
<div v-if="broker === 'ibkr'" class="config-section">
  <h4 class="section-title">IBKR Configuration</h4>
  <div class="form-group"><label>Host</label><input v-model="ibkrHost" class="form-input" /></div>
  <div class="form-group"><label>Port</label><input v-model.number="ibkrPort" type="number" class="form-input" /></div>
  <div class="form-group"><label>Client ID</label><input v-model.number="ibkrClientId" type="number" class="form-input" /></div>
</div>
```

Add Alpaca config section:
```html
<div v-if="broker === 'alpaca'" class="config-section">
  <h4 class="section-title">Alpaca Configuration</h4>
  <div class="form-group"><label>API Key</label><input v-model="alpacaKey" type="password" class="form-input" /></div>
  <div class="form-group"><label>Secret Key</label><input v-model="alpacaSecret" type="password" class="form-input" /></div>
  <div class="form-group checkbox-group"><label><input v-model="alpacaPaper" type="checkbox" /> Paper Trading</label></div>
</div>
```

Add save status indicator below the action buttons:
```html
<span v-if="savedMsg" class="save-msg">{{ savedMsg }}</span>
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/BrokerConfig.vue CHANGELOG.md
git commit -m "feat(frontend): wire BrokerConfig to SaveCredential/GetCredential — real broker config persistence"
```

---

### Task 3.4: Wire BasketOrderPanel to PlaceOrder

**Files:**
- Modify: `frontend/src/terminal/panels/BasketOrderPanel.vue`

**Interfaces:**
- Consumes: `app.PlaceOrder(symbol, side, orderType, brokerName, qty, price)`
- Produces: Working basket order submission

- [ ] **Step 1: Read current BasketOrderPanel to find submission logic**

```bash
grep -n "submit\|place\|order\|PlaceOrder" frontend/src/terminal/panels/BasketOrderPanel.vue
```

- [ ] **Step 2: Wire the submit function to call app.PlaceOrder for each row**

Find the submission handler and replace stub with real calls:

```typescript
async function submitBasket() {
  const app = (window as any).go?.main?.App
  if (!app?.PlaceOrder) return
  
  for (const row of basketOrders.value) {
    try {
      await app.PlaceOrder(row.symbol, row.side, row.orderType, 'paper', row.qty, row.price || 0)
      row.status = 'submitted'
    } catch (e: any) {
      row.status = 'failed'
      row.error = e?.message || String(e)
    }
  }
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/BasketOrderPanel.vue CHANGELOG.md
git commit -m "feat(frontend): wire BasketOrderPanel submit to PlaceOrder API"
```

---

### Task 3.5: Wire ChanlunPanel to GetChanlun

**Files:**
- Modify: `frontend/src/terminal/panels/ChanlunPanel.vue`

**Interfaces:**
- Consumes: `app.GetChanlun(symbol)` — existing Go method at `app_research.go`
- Produces: Chanlun analysis with real Python-computed data

- [ ] **Step 1: Read current ChanlunPanel to understand UI structure**

```bash
grep -n "symbol\|fetch\|load\|compute\|data\|ref(" frontend/src/terminal/panels/ChanlunPanel.vue | head -30
```

- [ ] **Step 2: Add data fetching via GetChanlun**

If the panel has a symbol input and a load/compute button, wire it:

```typescript
const symbol = ref('')
const loading = ref(false)
const chanlunData = ref<any>(null)
const error = ref('')

async function loadChanlun() {
  if (!symbol.value) return
  loading.value = true
  error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.GetChanlun) {
      error.value = 'GetChanlun not available'
      return
    }
    const result = await app.GetChanlun(symbol.value)
    chanlunData.value = result
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/ChanlunPanel.vue CHANGELOG.md
git commit -m "feat(frontend): wire ChanlunPanel to GetChanlun Go backend"
```

---

### Task 3.6: Wire FactorAnalysisPanel to Python factor registry

**Files:**
- Modify: `frontend/src/terminal/panels/FactorAnalysisPanel.vue`

**Interfaces:**
- Consumes: `app.ListMLModels()` (existing) or Python factor list via gRPC
- Produces: Dynamic factor catalog from backend, not hardcoded

- [ ] **Step 1: Read current FactorAnalysisPanel**

```bash
grep -n "ref(\[" frontend/src/terminal/panels/FactorAnalysisPanel.vue | head -5
```

- [ ] **Step 2: Replace hardcoded factor list with fetched data**

If the factor list is hardcoded in a `ref([...])`:

```typescript
const factors = ref<FactorDef[]>([])
const loading = ref(false)

async function loadFactors() {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    // Try ListMLModels or a dedicated factor listing endpoint
    const result = app?.ListFactors ? await app.ListFactors() : []
    factors.value = Array.isArray(result) ? result : []
  } catch (e) {
    console.warn('Factor list fetch failed, using built-in catalog:', e)
    // Keep the hardcoded list as fallback
  } finally {
    loading.value = false
  }
}

onMounted(loadFactors)
```

If there's no `ListFactors` Go method, this task becomes: add `ListFactors` to `app_ml.go` that returns the Python factor registry, then wire the frontend. For now, the hardcoded list serves as a complete catalog and this task can note the dependency.

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/FactorAnalysisPanel.vue CHANGELOG.md
git commit -m "feat(frontend): wire FactorAnalysisPanel to fetch factor list from backend"
```

---

### Task 3.7: Wire StockScannerPanel to ScanStocks

**Files:**
- Modify: `frontend/src/terminal/panels/StockScannerPanel.vue`

**Interfaces:**
- Consumes: `app.ScanStocks(strategyName)` — existing Go method at `app_research.go`
- Produces: Real stock scanning results

- [ ] **Step 1: Read current StockScannerPanel to find scan trigger**

```bash
grep -n "scan\|search\|run\|fetch\|mock\|hardcoded" frontend/src/terminal/panels/StockScannerPanel.vue | head -20
```

- [ ] **Step 2: Wire the scan button to call ScanStocks**

```typescript
const results = ref<any[]>([])
const loading = ref(false)
const error = ref('')

async function runScan() {
  if (!selectedStrategy.value) return
  loading.value = true
  error.value = ''
  try {
    const app = (window as any).go?.main?.App
    if (!app?.ScanStocks) {
      error.value = 'ScanStocks not available'
      return
    }
    const result = await app.ScanStocks(selectedStrategy.value)
    results.value = Array.isArray(result?.stocks) ? result.stocks : []
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    loading.value = false
  }
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/terminal/panels/StockScannerPanel.vue CHANGELOG.md
git commit -m "feat(frontend): wire StockScannerPanel to ScanStocks Go backend"
```

---

### Task 3.8: Update implementation status document

**Files:**
- Modify: `docs/specs/2026-06-18-proposal-implementation-status.md`

**Interfaces:**
- Produces: Accurate status document reflecting real project state

- [ ] **Step 1: Update key counts**

In the document, update:
- Section 7.1 "面板目录 (规划 50+, 已实现 22)" → "已实现 93"
- Section 3.2 "54 节点" → "196 节点"
- Section 7.3 "7 stores" → "8 stores" (add symbolContext)

- [ ] **Step 2: Mark panels that were previously listed as 📋 as ✅**

Scan the 93 panels and update status for panels that exist:
- HeatmapPanel, CryptoOverviewPanel, SurfaceChartPanel, DistributionPanel, BrokerStatusPanel, ActionCenterPanel → ✅
- SentimentPanel, CongressTradingPanel, FinancialsPanel, AnalystEstimatesPanel, PeerComparisonPanel, InsiderTradingPanel → ✅
- MonteCarloPanel, GeopoliticsPanel, SatellitePanel, GovDataPanel, PredictionMarketPanel → ✅

- [ ] **Step 3: Add note about wiring status**

Add a section noting:
- 82/93 panels wired (88%)
- 5 mock panels (BrokerConfig, BasketOrder, Chanlun, FactorAnalysis, StockScanner)
- 1 static placeholder (RLMonitor)
- 5 panels use client-side logic (MonteCarlo, etc.)

- [ ] **Step 4: Commit**

```bash
git add docs/specs/2026-06-18-proposal-implementation-status.md CHANGELOG.md
git commit -m "docs: update implementation status — 93 panels, 196 nodes, 88% wired"
```

---

## Verification Checklist

Before claiming completion, verify:

- [ ] `go test ./...` passes (981+ pass, max 2 skip)
- [ ] `cd frontend && npx vitest run` passes (198 tests)
- [ ] `cd python && python -m pytest tests/ --collect-only` shows test functions
- [ ] `cd frontend && npx vue-tsc --noEmit` has no new errors
- [ ] OrderEntryPanel can place orders for CN/HK/US/Crypto symbols
- [ ] BrokerStatusPanel shows real broker states (not just "Paper Trading")
- [ ] BrokerConfig can save/load credentials via Go backend
- [ ] BasketOrderPanel submits orders via PlaceOrder
- [ ] StockScannerPanel calls ScanStocks
- [ ] GDELT tests pass or skip (no FAIL)
- [ ] Implementation status doc reflects 93 panels / 196 nodes

# Workflow Node ↔ Panel Parity

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bridge the gap between the 84 terminal panels and ~90 workflow nodes. Create new node types for high-value panels that lack a workflow counterpart, enabling users to compose those panels' data/actions in the graphical workflow editor.

**Architecture:** Each new node follows the existing pattern in `internal/workflow/nodes/` — a struct implementing `BaseNode` with `NodeType()`, `Category()`, `Execute()`, `InputPorts()`, `OutputPorts()`, and self-registration via `register.go`. The Vue side gets a corresponding workflow canvas node component under `frontend/src/workflow/canvas/nodes/`.

**Priority mapping (missing panel → node):**

| Panel | Existing Node? | Action |
|-------|---------------|--------|
| MarketScannerPanel | ❌ | Create `market_scanner.go` — scan by criteria → return list |
| WatchlistPanel | ❌ | Create `watchlist.go` — query user watchlist → return symbols |
| OrderEntryPanel | partial (`place_order`) | Enhance `place_order.go` to support all order types from the panel |
| BasketOrderPanel | ❌ | Create `basket_order.go` — multi-leg basket order |
| TradeHistory | ❌ | Create `trade_history.go` — query & filter trade history |
| DepthComparisonPanel | ❌ | Create `orderbook_depth.go` — fetch orderbook depth snapshot |
| FundingRatePanel | ❌ | Create `funding_rate.go` — fetch perpetual funding rates |
| LiquidationPanel | ❌ | Create `liquidations.go` — fetch recent liquidations |
| DarkPoolPanel | ❌ | Create `darkpool.go` — fetch dark pool / block trades |
| GasFeePanel | ❌ | Create `gas_fee.go` — fetch current gas fees |
| WhaleTransactionsPanel | ❌ | Create `whale_tracker.go` — fetch large wallet moves |
| AuditPanel | ❌ | Create `audit_log.go` — query system audit log |
| CorrelationPanel | ❌ | Create `correlation.go` — compute pairwise correlations |

**Tech Stack:** Go 1.25+ (workflow nodes), Vue 3 + vue-flow (canvas nodes).

---

### Task 1: Create MarketScanner workflow node

**Files:**
- Create: `internal/workflow/nodes/market_scanner.go`
- Test: `internal/workflow/nodes/market_scanner_test.go`
- Create: `frontend/src/workflow/canvas/nodes/MarketScannerNode.vue`
- Modify: `internal/workflow/nodes/register.go`

**Node design:**
```go
func (n *MarketScannerNode) NodeType() string { return "market_scanner" }
func (n *MarketScannerNode) Category() string { return "data" }
// Consumes: market, filters (market/cap/sector/pattern)
// Produces: results ([]ScannerResult)
```

- [ ] **Step 1: Write failing Go test**

Create `internal/workflow/nodes/market_scanner_test.go`:

```go
package nodes

import (
	"context"
	"testing"
)

func TestMarketScannerNode_NodeType(t *testing.T) {
	n := &MarketScannerNode{}
	if n.NodeType() != "market_scanner" {
		t.Errorf("NodeType() = %q, want %q", n.NodeType(), "market_scanner")
	}
}

func TestMarketScannerNode_Execute_EmptyFilters(t *testing.T) {
	n := &MarketScannerNode{}
	ctx := context.Background()
	inputs := map[string]any{}
	outputs, err := n.Execute(ctx, inputs)
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	results, ok := outputs["results"]
	if !ok {
		t.Fatal("Execute() missing 'results' output")
	}
	_ = results // should return empty list, not error
}

func TestMarketScannerNode_InputPorts(t *testing.T) {
	n := &MarketScannerNode{}
	ports := n.InputPorts()
	if len(ports) == 0 {
		t.Error("InputPorts() should define at least one port")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/workflow/nodes/ -run TestMarketScannerNode -v
```
Expected: compilation error (no MarketScannerNode defined)

- [ ] **Step 3: Create node implementation**

Create `internal/workflow/nodes/market_scanner.go`:

```go
package nodes

import (
	"context"
	"fmt"
	"github.com/quantflow/workflow" // adjust to actual import path
)

type MarketScannerNode struct {
	workflow.BaseNode
	Market string   `json:"market"`
	Filters []string `json:"filters"`
}

func NewMarketScannerNode(id string, params map[string]any) (workflow.BaseNode, error) {
	n := &MarketScannerNode{}
	n.SetID(id)
	if m, ok := params["market"].(string); ok {
		n.Market = m
	}
	if fs, ok := params["filters"].([]any); ok {
		for _, f := range fs {
			if s, ok := f.(string); ok {
				n.Filters = append(n.Filters, s)
			}
		}
	}
	return n, nil
}

func (n *MarketScannerNode) NodeType() string  { return "market_scanner" }
func (n *MarketScannerNode) Category() string  { return "data" }

func (n *MarketScannerNode) InputPorts() []workflow.Port {
	return []workflow.Port{
		{Name: "filters", Type: "object", Description: "Filter criteria (market, cap, sector)"},
	}
}

func (n *MarketScannerNode) OutputPorts() []workflow.Port {
	return []workflow.Port{
		{Name: "results", Type: "array", Description: "Scanned market results"},
	}
}

func (n *MarketScannerNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// Resolve market data adapter runtime
	// Delegate to MarketScanner or datahub
	// Return []ScannerResult
	return map[string]any{"results": []any{}}, nil
}
```

- [ ] **Step 4: Register in register.go**

Add to `internal/workflow/nodes/register.go`:
```go
"market_scanner": {Factory: NewMarketScannerNode, Category: "data"},
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/workflow/nodes/ -run TestMarketScannerNode -v
```
Expected: PASS

- [ ] **Step 6: Create Vue canvas node component**

Create `frontend/src/workflow/canvas/nodes/MarketScannerNode.vue` following existing canvas node patterns.

- [ ] **Step 7: Commit**

```bash
git add internal/workflow/nodes/market_scanner.go internal/workflow/nodes/market_scanner_test.go frontend/src/workflow/canvas/nodes/MarketScannerNode.vue
git commit -m "feat(workflow): add market_scanner node"
```

---

### Task 2: Create Watchlist workflow node

**Files:**
- Create: `internal/workflow/nodes/watchlist.go`
- Test: `internal/workflow/nodes/watchlist_test.go`
- Create: `frontend/src/workflow/canvas/nodes/WatchlistNode.vue`
- Modify: `internal/workflow/nodes/register.go`

- [ ] **Step 1: Write test → run (fails) → create implementation → register → run test → create Vue node → commit**

```bash
git commit -m "feat(workflow): add watchlist node"
```

---

### Task 3: Enhance PlaceOrder node for full order types

**Files:**
- Modify: `internal/workflow/nodes/place_order.go`
- Modify: `internal/workflow/nodes/place_order_test.go`

Current `place_order.go` likely handles basic market/limit orders. Enhance to support all types from `OrderEntryPanel`:
- Market, Limit, Stop, Stop-Limit, IOC, FOK
- Take-profit, Stop-loss (bracket orders)
- Multi-leg (if not in BasketOrder node)

- [ ] **Step 1: Read existing place_order.go → identify gaps → extend InputPorts → add test cases → commit**

```bash
git commit -m "feat(workflow): enhance place_order node with full order types (IOC, FOK, bracket)"
```

---

### Task 4: Create BasketOrder workflow node

**Files:**
- Create: `internal/workflow/nodes/basket_order.go`
- Test: `internal/workflow/nodes/basket_order_test.go`
- Create: `frontend/src/workflow/canvas/nodes/BasketOrderNode.vue`
- Modify: `internal/workflow/nodes/register.go`

- [ ] **Step 1: Write test → run (fails) → create → register → test → Vue node → commit**

```bash
git commit -m "feat(workflow): add basket_order node"
```

---

### Task 5: Create TradeHistory workflow node

**Files:**
- Create: `internal/workflow/nodes/trade_history.go`
- Test: `internal/workflow/nodes/trade_history_test.go`
- Create: `frontend/src/workflow/canvas/nodes/TradeHistoryNode.vue`
- Modify: `internal/workflow/nodes/register.go`

- [ ] **Step 1: Write test → run (fails) → create → register → test → Vue node → commit**

```bash
git commit -m "feat(workflow): add trade_history node"
```

---

### Task 6: Create MarketData snapshot nodes (orderbook_depth, funding_rate, liquidations, darkpool, gas_fee, whale_tracker)

These are lightweight read-only data nodes — each does 1 HTTP call → returns structured output. Group into 2 commits.

- [ ] **Task 6a: orderbook_depth + funding_rate + liquidations**

Files:
- Create: `internal/workflow/nodes/orderbook_depth.go`
- Create: `internal/workflow/nodes/funding_rate.go`
- Create: `internal/workflow/nodes/liquidations.go`
- Modify: `internal/workflow/nodes/register.go`
- Tests per file following MarketScanner pattern.

```bash
git commit -m "feat(workflow): add orderbook_depth, funding_rate, liquidations data nodes"
```

- [ ] **Task 6b: darkpool + gas_fee + whale_tracker**

Files:
- Create: `internal/workflow/nodes/darkpool.go`
- Create: `internal/workflow/nodes/gas_fee.go`
- Create: `internal/workflow/nodes/whale_tracker.go`
- Modify: `internal/workflow/nodes/register.go`

```bash
git commit -m "feat(workflow): add darkpool, gas_fee, whale_tracker data nodes"
```

---

### Task 7: Create AuditLog and Correlation workflow nodes

- [ ] **Task 7a: audit_log**

```bash
go test ./internal/workflow/nodes/ -run TestAuditLogNode -v
```

Files:
- Create: `internal/workflow/nodes/audit_log.go`
- Test: `internal/workflow/nodes/audit_log_test.go`
- Modify: `internal/workflow/nodes/register.go`

```bash
git commit -m "feat(workflow): add audit_log node"
```

- [ ] **Task 7b: correlation**

Files:
- Create: `internal/workflow/nodes/correlation.go`
- Test: `internal/workflow/nodes/correlation_test.go`
- Modify: `internal/workflow/nodes/register.go`

```bash
git commit -m "feat(workflow): add correlation node"
```

---

### Task 8: Verify full test suite

- [ ] **Step 1: Run all Go tests**

```bash
go test ./internal/...
```
Expected: PASS (1036+ tests)

- [ ] **Step 2: Run frontend tests**

```bash
cd frontend && npx vitest run
```
Expected: PASS

- [ ] **Step 3: Run type check**

```bash
cd frontend && npx vue-tsc --noEmit
```
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: verify workflow node parity — all tests pass"
```

---

### Task 9: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add entries**

```markdown
### Added
- [Workflow] New market_scanner node for criteria-based market scanning
- [Workflow] New watchlist node for user watchlist queries
- [Workflow] New basket_order node for multi-leg basket orders
- [Workflow] New trade_history node for historical trade queries
- [Workflow] New data nodes: orderbook_depth, funding_rate, liquidations, darkpool, gas_fee, whale_tracker
- [Workflow] New audit_log and correlation compute nodes
- [Workflow] Enhanced place_order node with IOC, FOK, stop-limit, bracket order support
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md && git commit -m "chore: update CHANGELOG for workflow node parity"
```

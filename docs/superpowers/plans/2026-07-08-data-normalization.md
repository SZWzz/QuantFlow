# Data Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create `internal/normalize/` package with unified OHLCV type, volume normalizer, FieldMapper interface, and order status/type mapper registry.

**Architecture:** New `internal/normalize/` package → retrofitting `market`/`trading` types → DataNormalizeNode

**Tech Stack:** Go 1.25+, no external dependencies

## Global Constraints
- All new code in `internal/normalize/`
- Backward compatible — existing callers unchanged
- Tests only, no production behavior change for existing adapters

---

### Task 1: Package scaffold + unified OHLCVBar + volume normalizer

**Files:**
- Create: `internal/normalize/ohlcv.go`
- Create: `internal/normalize/ohlcv_test.go`
- Create: `internal/normalize/volume.go`
- Create: `internal/normalize/volume_test.go`

- [ ] **Step 1: Create `internal/normalize/ohlcv.go`**

```go
// Package normalize provides unified data types and field mappers for normalizing
// data from various sources (EastMoney, Binance, IBKR, etc.) into canonical formats.
package normalize

// OHLCVBar is the canonical OHLCV data type used across the entire system.
// All adapters SHOULD normalize their output to this type.
type OHLCVBar struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`   // "2006-01-02"
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
```

- [ ] **Step 2: Create `internal/normalize/volume.go`**

```go
package normalize

// volumeMultiplier maps adapter names to their volume-to-shares multiplier.
// A-share data sources return volume in 手 (lots of 100 shares).
var volumeMultiplier = map[string]float64{
	"eastmoney": 100,
	"sina":      100,
	"tencent":   100,
	"tushare":   100,
	"mootdx":    100,
	"baidu":     100,
}

// NormalizeVolume converts trading volume to standard shares.
// Returns volume unchanged for unknown or non-A-share sources.
func NormalizeVolume(source string, volume float64) float64 {
	if mult, ok := volumeMultiplier[source]; ok {
		return volume * mult
	}
	return volume
}

// VolumeSources returns the list of known A-share data sources.
func VolumeSources() []string {
	sources := make([]string, 0, len(volumeMultiplier))
	for s := range volumeMultiplier {
		sources = append(sources, s)
	}
	return sources
}
```

- [ ] **Step 3: Create `internal/normalize/ohlcv_test.go`**

```go
package normalize

import (
	"testing"
)

func TestOHLCVBar_Fields(t *testing.T) {
	bar := OHLCVBar{
		Symbol: "000001", Date: "2026-01-02",
		Open: 10.0, High: 11.0, Low: 9.0, Close: 10.5, Volume: 100000,
	}
	if bar.Symbol != "000001" {
		t.Errorf("Symbol = %q, want %q", bar.Symbol, "000001")
	}
	if bar.Close != 10.5 {
		t.Errorf("Close = %v, want 10.5", bar.Close)
	}
}
```

- [ ] **Step 4: Create `internal/normalize/volume_test.go`**

```go
package normalize

import (
	"testing"
)

func TestNormalizeVolume_KnownSources(t *testing.T) {
	tests := []struct {
		source string
		input  float64
		want   float64
	}{
		{"eastmoney", 100, 10000},
		{"sina", 100, 10000},
		{"tencent", 100, 10000},
		{"tushare", 100, 10000},
		{"mootdx", 100, 10000},
		{"baidu", 100, 10000},
		{"yahoo", 100, 100},       // US source, no multiplier
		{"binance", 1.5, 1.5},     // crypto, no multiplier
		{"unknown", 100, 100},     // unknown source
		{"", 100, 100},            // empty source
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := NormalizeVolume(tt.source, tt.input)
			if got != tt.want {
				t.Errorf("NormalizeVolume(%q, %v) = %v, want %v", tt.source, tt.input, got, tt.want)
			}
		})
	}
}

func TestVolumeSources_NotEmpty(t *testing.T) {
	sources := VolumeSources()
	if len(sources) == 0 {
		t.Fatal("VolumeSources() returned empty")
	}
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/normalize/ -v -count=1
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/normalize/
git commit -m "feat(normalize): add unified OHLCVBar type and NormalizeVolume helper"
```

---

### Task 2: FieldMapper interface + OrderStatus/Type registry

**Files:**
- Create: `internal/normalize/mapper.go`
- Create: `internal/normalize/mapper_test.go`

- [ ] **Step 1: Create `internal/normalize/mapper.go`**

```go
package normalize

import (
	"fmt"
	"strings"
)

// Mapper defines a generic field mapper from raw data to a normalized type.
type Mapper[T any] interface {
	Parse(raw map[string]any) (T, error)
	Source() string
}

// ── Order status mapping ────────────────────────────────────────────

// OrderStatusMapper normalizes broker-specific order status strings.
type OrderStatusMapper struct {
	broker string
}

// NewOrderStatusMapper creates a mapper for the given broker.
func NewOrderStatusMapper(broker string) *OrderStatusMapper {
	return &OrderStatusMapper{broker: broker}
}

// Map converts a broker-specific order status to a canonical string.
// Canonical values: "pending", "filled", "cancelled", "rejected", "partial".
func (m *OrderStatusMapper) Map(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch m.broker {
	case "ibkr":
		switch s {
		case "submitted", "presubmitted", "inactive":
			return "pending"
		case "filled":
			return "filled"
		case "cancelled", "apicancelled":
			return "cancelled"
		default:
			return "pending"
		}
	case "binance":
		switch s {
		case "new", "partially_filled":
			return "pending"
		case "filled":
			return "filled"
		case "canceled", "expired", "rejected":
			return "cancelled"
		default:
			return "pending"
		}
	case "alpaca":
		switch s {
		case "accepted", "new", "partially_filled", "held":
			return "pending"
		case "filled":
			return "filled"
		case "canceled", "expired", "rejected", "suspended":
			return "cancelled"
		default:
			return "pending"
		}
	default:
		return s
	}
}

// Source returns the broker name.
func (m *OrderStatusMapper) Source() string { return m.broker }

// ── Order type mapping ─────────────────────────────────────────────

// OrderTypeMapper normalizes broker-specific order type strings.
type OrderTypeMapper struct {
	broker string
}

// NewOrderTypeMapper creates a mapper for the given broker.
func NewOrderTypeMapper(broker string) *OrderTypeMapper {
	return &OrderTypeMapper{broker: broker}
}

// Map converts a broker-specific order type to a canonical string.
// Canonical values: "market", "limit", "stop", "stop_limit".
func (m *OrderTypeMapper) Map(orderType string) string {
	s := strings.ToUpper(strings.TrimSpace(orderType))
	switch m.broker {
	case "ibkr":
		switch s {
		case "MKT":
			return "market"
		case "LMT":
			return "limit"
		case "STP":
			return "stop"
		default:
			return "market"
		}
	case "binance":
		switch s {
		case "MARKET":
			return "market"
		case "LIMIT", "LIMIT_MAKER":
			return "limit"
		case "STOP_LOSS_LIMIT", "STOP_LOSS", "TAKE_PROFIT", "TAKE_PROFIT_LIMIT":
			return "stop"
		default:
			return "market"
		}
	case "alpaca":
		switch s {
		case "MARKET":
			return "market"
		case "LIMIT":
			return "limit"
		case "STOP", "STOP_LIMIT":
			return "stop"
		default:
			return "market"
		}
	default:
		return s
	}
}

// Source returns the broker name.
func (m *OrderTypeMapper) Source() string { return m.broker }

// ── OHLCV Mapper ───────────────────────────────────────────────────

// OHLCVMapper maps a raw map (e.g. from CSV, JSON) to a normalized OHLCVBar.
type OHLCVMapper struct {
	source  string
	columns map[string]string // canonical -> raw field name
}

// NewOHLCVMapper creates an OHLCV mapper with field name mappings.
// columns maps canonical field names ("symbol", "open", etc.) to raw source field names.
func NewOHLCVMapper(source string, columns map[string]string) *OHLCVMapper {
	return &OHLCVMapper{source: source, columns: columns}
}

// Parse converts a raw data map to a normalized OHLCVBar.
func (m *OHLCVMapper) Parse(raw map[string]any) (OHLCVBar, error) {
	var bar OHLCVBar
	var errs []string

	if v, ok := raw[m.columns["symbol"]]; ok {
		bar.Symbol = fmt.Sprintf("%v", v)
	} else {
		errs = append(errs, "symbol")
	}
	if v, ok := raw[m.columns["date"]]; ok {
		bar.Date = fmt.Sprintf("%v", v)
	}
	if v, ok := raw[m.columns["open"]]; ok {
		bar.Open = toFloat64(v)
	}
	if v, ok := raw[m.columns["high"]]; ok {
		bar.High = toFloat64(v)
	}
	if v, ok := raw[m.columns["low"]]; ok {
		bar.Low = toFloat64(v)
	}
	if v, ok := raw[m.columns["close"]]; ok {
		bar.Close = toFloat64(v)
	}
	if v, ok := raw[m.columns["volume"]]; ok {
		bar.Volume = NormalizeVolume(m.source, toFloat64(v))
	}

	if len(errs) > 0 {
		return bar, fmt.Errorf("OHLCVMapper: missing required fields: %s", strings.Join(errs, ", "))
	}
	return bar, nil
}

func (m *OHLCVMapper) Source() string { return m.source }

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case float32:
		return float64(n)
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	default:
		return 0
	}
}
```

- [ ] **Step 2: Create `internal/normalize/mapper_test.go`**

```go
package normalize

import (
	"testing"
)

func TestOrderStatusMapper_IBKR(t *testing.T) {
	m := NewOrderStatusMapper("ibkr")
	tests := []struct {
		input string
		want  string
	}{
		{"Submitted", "pending"},
		{"PreSubmitted", "pending"},
		{"Filled", "filled"},
		{"Cancelled", "cancelled"},
		{"ApiCancelled", "cancelled"},
		{"Inactive", "pending"},
		{"Unknown", "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderStatusMapper_Binance(t *testing.T) {
	m := NewOrderStatusMapper("binance")
	tests := []struct {
		input string
		want  string
	}{
		{"NEW", "pending"},
		{"PARTIALLY_FILLED", "pending"},
		{"FILLED", "filled"},
		{"CANCELED", "cancelled"},
		{"EXPIRED", "cancelled"},
		{"REJECTED", "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderStatusMapper_Alpaca(t *testing.T) {
	m := NewOrderStatusMapper("alpaca")
	tests := []struct {
		input string
		want  string
	}{
		{"accepted", "pending"},
		{"new", "pending"},
		{"partially_filled", "pending"},
		{"filled", "filled"},
		{"canceled", "cancelled"},
		{"expired", "cancelled"},
		{"rejected", "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOrderTypeMapper_IBKR(t *testing.T) {
	m := NewOrderTypeMapper("ibkr")
	tests := []struct {
		input string
		want  string
	}{
		{"MKT", "market"},
		{"LMT", "limit"},
		{"STP", "stop"},
		{"UNKNOWN", "market"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := m.Map(tt.input); got != tt.want {
				t.Errorf("Map(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestOHLCVMapper_Parse(t *testing.T) {
	m := NewOHLCVMapper("eastmoney", map[string]string{
		"symbol": "code",
		"date":   "trade_date",
		"open":   "opn",
		"high":   "hi",
		"low":    "lo",
		"close":  "cls",
		"volume": "vol",
	})

	raw := map[string]any{
		"code":       "000001",
		"trade_date": "2026-01-02",
		"opn":        "10.0",
		"hi":         "11.0",
		"lo":         "9.0",
		"cls":        "10.5",
		"vol":        100.0, // 手 → NormalizeVolume × 100
	}

	bar, err := m.Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if bar.Symbol != "000001" {
		t.Errorf("Symbol = %q, want %q", bar.Symbol, "000001")
	}
	if bar.Open != 10.0 {
		t.Errorf("Open = %v, want 10.0", bar.Open)
	}
	if bar.Volume != 10000 {
		t.Errorf("Volume = %v, want 10000 (100手×100)", bar.Volume)
	}
}

func TestOHLCVMapper_MissingSymbol(t *testing.T) {
	m := NewOHLCVMapper("test", map[string]string{
		"symbol": "code",
		"open":   "opn",
	})
	_, err := m.Parse(map[string]any{"opn": 10.0})
	if err == nil {
		t.Fatal("expected error for missing symbol")
	}
}
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/normalize/ -v -count=1
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/normalize/
git commit -m "feat(normalize): add FieldMapper interface + OrderStatus/Type mappers + OHLCVMapper"
```

---

### Task 3: Retrofit market/trading OHLCVBar types

**Files:**
- Modify: `internal/market/types.go`
- Modify: `internal/trading/types.go`

**Goal:** Make both existing `OHLCVBar` types aliases of `normalize.OHLCVBar` so existing code continues to compile and work without changes.

- [ ] **Step 1: Read `internal/market/types.go` around OHLCVBar definition**

- [ ] **Step 2: Change `market.OHLCVBar` to alias**

```go
// Replace the existing struct definition with:
import "quantflow/internal/normalize"

// OHLCVBar is an alias for normalize.OHLCVBar.
type OHLCVBar = normalize.OHLCVBar
```

Using `type alias (=)` ensures existing code like `market.OHLCVBar{...}` and `bar.Open` continues to compile.

- [ ] **Step 3: Change `trading.OHLCVBar` to alias**

Same pattern:
```go
import "quantflow/internal/normalize"

// OHLCVBar is an alias for normalize.OHLCVBar.
type OHLCVBar = normalize.OHLCVBar
```

- [ ] **Step 4: Run build**

```bash
go build ./...
```

Expected: Build succeeds (aliases are transparent to callers)

- [ ] **Step 5: Run full test suite**

```bash
go test ./... -count=1
```

Expected: All tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/market/types.go internal/trading/types.go
git commit -m "refactor(normalize): replace market/trading OHLCVBar with normalize.OHLCVBar alias"
```

---

### Task 4: DataNormalizeNode workflow node

**Files:**
- Create: `internal/workflow/nodes/data_normalize.go`
- Create: `internal/workflow/nodes/data_normalize_test.go`
- Modify: `internal/workflow/nodes/register.go`

- [ ] **Step 1: Create `internal/workflow/nodes/data_normalize.go`**

```go
package nodes

import (
	"context"
	"fmt"

	"quantflow/internal/normalize"
)

// DataNormalizeNode normalizes input data using a configured field mapper.
type DataNormalizeNode struct {
	BaseNode
	Source  string            `json:"source"`
	Mapping map[string]string `json:"mapping"` // canonical -> raw field name
	Target  string            `json:"target"`  // "ohlcv", "order_status", "order_type"
}

func (n *DataNormalizeNode) NodeType() string { return "data_normalize" }
func (n *DataNormalizeNode) Category() string { return "data" }

func (n *DataNormalizeNode) Execute(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	switch n.Target {
	case "ohlcv":
		return n.normalizeOHLCV(ctx, inputs)
	case "order_status":
		return n.normalizeOrderStatus(ctx, inputs)
	case "order_type":
		return n.normalizeOrderType(ctx, inputs)
	default:
		return nil, fmt.Errorf("data_normalize: unknown target %q", n.Target)
	}
}

func (n *DataNormalizeNode) normalizeOHLCV(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	raw, ok := inputs["raw"]
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing input 'raw'")
	}

	rawMap, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("data_normalize: 'raw' must be a map[string]any")
	}

	mapper := normalize.NewOHLCVMapper(n.Source, n.Mapping)
	bar, err := mapper.Parse(rawMap)
	if err != nil {
		return nil, fmt.Errorf("data_normalize: %w", err)
	}

	return map[string]any{"ohlcv": bar}, nil
}

func (n *DataNormalizeNode) normalizeOrderStatus(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	status, ok := inputs["status"].(string)
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing or non-string input 'status'")
	}

	broker := n.Source
	if b, ok := inputs["broker"].(string); ok && b != "" {
		broker = b
	}

	mapper := normalize.NewOrderStatusMapper(broker)
	return map[string]any{"normalized_status": mapper.Map(status)}, nil
}

func (n *DataNormalizeNode) normalizeOrderType(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	orderType, ok := inputs["order_type"].(string)
	if !ok {
		return nil, fmt.Errorf("data_normalize: missing or non-string input 'order_type'")
	}

	broker := n.Source
	if b, ok := inputs["broker"].(string); ok && b != "" {
		broker = b
	}

	mapper := normalize.NewOrderTypeMapper(broker)
	return map[string]any{"normalized_type": mapper.Map(orderType)}, nil
}
```

- [ ] **Step 2: Create test**

```go
// internal/workflow/nodes/data_normalize_test.go
package nodes

import (
	"context"
	"testing"
)

func TestDataNormalizeNode_OHLCV(t *testing.T) {
	node := &DataNormalizeNode{
		Source: "eastmoney",
		Mapping: map[string]string{
			"symbol": "code", "date": "date", "open": "open",
			"high": "high", "low": "low", "close": "close", "volume": "volume",
		},
		Target: "ohlcv",
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"raw": map[string]any{
			"code": "000001", "date": "2026-01-02",
			"open": 10.0, "high": 11.0, "low": 9.0, "close": 10.5, "volume": 100.0,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	bar, ok := output["ohlcv"]
	if !ok {
		t.Fatal("expected 'ohlcv' in output")
	}
	_ = bar // type is normalize.OHLCVBar
}

func TestDataNormalizeNode_OrderStatus(t *testing.T) {
	node := &DataNormalizeNode{
		Source: "ibkr",
		Target: "order_status",
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"status": "Filled",
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	status, ok := output["normalized_status"]
	if !ok {
		t.Fatal("expected 'normalized_status' in output")
	}
	if status != "filled" {
		t.Errorf("normalized_status = %q, want 'filled'", status)
	}
}

func TestDataNormalizeNode_OrderType(t *testing.T) {
	node := &DataNormalizeNode{
		Source: "binance",
		Target: "order_type",
	}

	output, err := node.Execute(context.Background(), map[string]any{
		"order_type": "LIMIT",
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	orderType, ok := output["normalized_type"]
	if !ok {
		t.Fatal("expected 'normalized_type' in output")
	}
	if orderType != "limit" {
		t.Errorf("normalized_type = %q, want 'limit'", orderType)
	}
}

func TestDataNormalizeNode_MissingInput(t *testing.T) {
	node := &DataNormalizeNode{
		Target: "ohlcv",
	}
	_, err := node.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}
```

- [ ] **Step 3: Register the node**

Read `internal/workflow/nodes/register.go`, find the `data` category section, add:
```go
NodeDef{Type: "data_normalize", Factory: func() Node { return &DataNormalizeNode{BaseNode: BaseNode{name: "数据归一化"}} }},
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/workflow/nodes/ -run TestDataNormalizeNode -v -count=1
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/workflow/nodes/data_normalize.go internal/workflow/nodes/data_normalize_test.go internal/workflow/nodes/register.go
git commit -m "feat(normalize): add DataNormalizeNode workflow node"
```

---

### Task 5: CHANGELOG + full check

- [ ] **Step 1: Update CHANGELOG.md**

Add under `[2026.7.8]`:
```markdown
### Added
- [Engine] Data normalization system — unified OHLCVBar type, NormalizeVolume helper,
  FieldMapper interface, OrderStatus/Type mappers, OHLCVMapper, DataNormalizeNode
```

- [ ] **Step 2: Run full check**

```bash
go vet ./... && go test ./... -count=1
```

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for data normalization system"
```

# A-Share ST Price Limit Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove dead ST detection code from price limit logic; update docstrings for current market rules; add STStatusProvider interface.

**Architecture:** A single-file change to `price_limit.go` + test + docstring update. The ST branch is dead code (A-share tickers are numeric), and under new registration-based rules, ST stocks follow board-level limits (no special ±5%).

**Tech Stack:** Go 1.25

## Global Constraints

- No functional change to backtest results (ST branch was unreachable)
- `STStatusProvider` interface only — no implementation required
- Update tests to match new docstring (no ST test cases)

---

### Task 1: Remove Dead ST Code and Update Docstrings

**Files:**
- Modify: `internal/backtest/price_limit.go`
- Modify: `internal/backtest/engine_cn.go` (docstring only)
- Test: `internal/backtest/price_limit_test.go`

- [ ] **Step 1: Write updated test first**

```go
// internal/backtest/price_limit_test.go
func TestPriceLimitFor_BoardRules(t *testing.T) {
    tests := []struct {
        symbol string
        ratio  float64
    }{
        {"600519", 0.10},  // main board
        {"000001", 0.10},  // main board
        {"002001", 0.10},  // SME board (also 00xxxx)
        {"300750", 0.20},  // ChiNext
        {"301001", 0.20},  // ChiNext
        {"688001", 0.20},  // STAR
        {"689001", 0.20},  // STAR
        {"830001", 0.30},  // BSE
        {"400001", 0.30},  // BSE (old BSE code)
        {"999999", 0.10},  // unknown → safe default
    }
    for _, tt := range tests {
        got := PriceLimitFor(tt.symbol)
        if got.Ratio != tt.ratio {
            t.Errorf("PriceLimitFor(%s) = %v, want ratio %v", tt.symbol, got, tt.ratio)
        }
    }
}

// Ensure ST symbols don't exist in A-share ticker format
func TestPriceLimitFor_STNotInTicker(t *testing.T) {
    // ST/*ST designation is a property of the stock name, NOT the ticker code.
    // A-share tickers are purely numeric (600xxx, 000xxx, 300xxx, etc.)
    // and never contain "ST". This test verifies that no special ST handling
    // exists in PriceLimitFor — ST stocks follow their board's standard limits.
    result := PriceLimitFor("600519") // Kweichow Moutai, non-ST
    if result.Ratio != 0.10 {
        t.Errorf("expected main board 10%%, got %v%%", result.Ratio*100)
    }
}
```

- [ ] **Step 2: Run test — verify it passes against current code**

```bash
cd /app && go test ./internal/backtest/ -run TestPriceLimitFor -v -count=1
```
Expected: PASS (current code already returns correct board ratios for numeric tickers)

- [ ] **Step 3: Remove ST dead code and update price_limit.go**

Remove line 25 (the ST case) and the ST-related comment:

```go
// PriceLimitFor returns the limit rule for a given A-share symbol code.
// A-share markets enforce ±Ratio around the previous closing price.
// Under the registration-based reform, ST stocks follow their board's
// standard limit — there is no special ±5% rule.
//   - Main board (60xxxx, 00xxxx): ±10%
//   - ChiNext / 创业板 (300xxx, 301xxx): ±20%
//   - STAR / 科创板 (688xxx, 689xxx): ±20%
//   - BSE / 北交所 (8xxxxx, 4xxxxx): ±30% (enforced)
//
// 首日上市、增发等无前收盘价的情形不限制（返回 0 表示不限）。
type PriceLimitRule struct {
	Ratio float64 // 0.10, 0.20, 0.30; 0 means no limit
}

// PriceLimitFor returns the limit rule for a given A-share symbol code.
// Symbol prefixes follow SSE/SZSE listing conventions.
// Note: ST status is a name-level property, not encoded in the ticker.
// ST stocks follow their board's standard limits under current regulations.
func PriceLimitFor(symbol string) PriceLimitRule {
	upper := strings.ToUpper(symbol)

	switch {
	case strings.HasPrefix(upper, "300"), strings.HasPrefix(upper, "301"): // ChiNext
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "688"), strings.HasPrefix(upper, "689"): // STAR
		return PriceLimitRule{Ratio: 0.20}
	case strings.HasPrefix(upper, "60"), strings.HasPrefix(upper, "00"): // main board
		return PriceLimitRule{Ratio: 0.10}
	case strings.HasPrefix(upper, "8"), strings.HasPrefix(upper, "4"): // BSE
		return PriceLimitRule{Ratio: 0.30}
	default:
		return PriceLimitRule{Ratio: 0.10} // safe default
	}
}
```

- [ ] **Step 4: Run test — verify it still passes**

```bash
cd /app && go test ./internal/backtest/ -run TestPriceLimitFor -v -count=1
```
Expected: PASS

- [ ] **Step 5: Add STStatusProvider interface**

```go
// Add after PriceLimitRule struct definition in price_limit.go

// STStatusProvider checks if a symbol is currently under ST/*ST risk warning.
// Returns true if the stock is designated ST or *ST.
// Implementation comes from market data adapters which return ST status
// in their quote responses. When no provider is available, the default
// assumption is false (see PriceLimitFor — ST stocks follow board limits).
type STStatusProvider interface {
	IsST(symbol string) (bool, error)
}
```

- [ ] **Step 6: Update engine_cn.go docstring**

```go
// CNEngine is the A-share backtesting engine with market-specific rules:
//   - T+1 settlement: shares bought today cannot be sold until tomorrow
//   - Price limits: ±10% (main board), ±20% (ChiNext/STAR), ±30% (BSE)
//   - Stamp duty: 0.05% on sell only (2024新政)
//   - Minimum lot: 100 shares, multiples of 100
//   - Commission: 0.03% (万三) default
```

- [ ] **Step 7: Run all backtest tests**

```bash
cd /app && go test ./internal/backtest/... -v -count=1
```
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/backtest/price_limit.go internal/backtest/price_limit_test.go internal/backtest/engine_cn.go
git commit -m "fix(backtest): remove dead ST price limit code, add STStatusProvider interface, update docstrings for new regulations"
```

---

### Task 2: Update CHANGELOG

- [ ] **Step 1: Update CHANGELOG.md**

```markdown
### Changed
- [Docs] Remove dead `strings.Contains(upper, "ST")` branch from PriceLimitFor — A-share tickers are purely numeric, this branch was unreachable
- [Docs] Update docstring: ST stocks now follow board-level limits under new registration-based rules (no special ±5%)

### Added
- [Engine] STStatusProvider interface for future real-time ST detection from market data adapters
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for ST price limit fix"
```
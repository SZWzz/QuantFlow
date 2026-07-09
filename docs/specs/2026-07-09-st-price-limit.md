# A-Share ST Stock Price Limit Rules

## Motivation

`PriceLimitFor()` in `internal/backtest/price_limit.go` has two problems:

1. **Dead code**: `case strings.Contains(upper, "ST")` never fires because A-share ticker codes are purely numeric (600519, 000001, 300750, etc.). ST status is not encoded in the ticker.

2. **Wrong rule**: Under the new A-share registration-based reform rules (注册制全面实施后), ST stocks on the main board **no longer have a special ±5% price limit**. Instead:
   - ST stocks follow their board's standard limit (main board ±10%, ChiNext ±20%, STAR ±20%)
   - Some risk-warning stock categories may have no limit (0%) on the first ST designation day
   - The old ±5% ST rule has been abolished

The current code silently returns the board-standard ratio for all stocks (since the ST branch is unreachable), which happens to be partially correct under new rules. But:
- There is no way to know ST status at all (no data source integration)
- The dead code branch misleads developers
- The comment/docstring is wrong

## Design

### 1. Remove Dead Code

Delete the `strings.Contains(upper, "ST")` case and update the docstring/comments.

```go
// PriceLimitFor returns the limit rule for a given A-share symbol code.
// ST stocks now follow their board's standard limits (no special ±5% rule).
// ST status detection requires external data source — not available here.
func PriceLimitFor(symbol string) PriceLimitRule {
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

### 2. Add ST Status Provider Interface (For Future Use)

Add an optional ST detection hook that can be wired when a market data adapter provides ST status.

```go
// STStatusProvider checks if a symbol is currently under ST/*ST risk warning.
// Returns true if the stock is designated ST or *ST.
// Implementation comes from market data adapters (EastMoney, Sina, etc.)
// which return ST status in their quote responses.
type STStatusProvider interface {
    IsST(symbol string) (bool, error)
}
```

The `PriceLimitFor` function can optionally accept this provider, but the default path (no provider) applies board-standard limits, which matches current market rules.

### 3. Update `engine_cn.go` to Track ST Status

The `CNEngine` already has a `prevClose` map. Add a parallel `stStatus` map that gets populated from quote data during the run loop. This is a future enhancement — the initial fix is just cleaning up the dead code.

**Modified files:**
- `internal/backtest/price_limit.go` — Remove ST case; update docstring; add `STStatusProvider` interface
- `internal/backtest/price_limit_test.go` — Remove ST test cases; update for new rules
- `internal/backtest/engine_cn.go` — Update docstring at line 50 to remove "±5% ST" mention

### 4. Update `CNEngine` Docstring

Current: `Price limits: ±10% (main board) or ±20% (ChiNext/STAR)`
Target: Same (ST stocks follow board limits)

## Acceptance Criteria

- [ ] `strings.Contains(upper, "ST")` branch removed
- [ ] All test cases pass with updated rules
- [ ] `STStatusProvider` interface defined (can be implemented later)
- [ ] No functional change to backtest results (ST branch was dead code)
- [ ] Docstrings updated everywhere

## Risks / Trade-offs

- **No functional change**: Since the ST branch was dead code, this fix doesn't change behavior — it only improves code clarity. The real ST detection (from market data) is deferred.
- **First-day ST no limit**: The current code doesn't handle "first day of ST designation has no price limit" — this requires real-time ST status + designation date, which is out of scope for this fix.

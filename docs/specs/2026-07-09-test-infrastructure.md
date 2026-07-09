# Test & Type Infrastructure Repair

## Motivation

Phase 12 review found the test suite in a degraded state:

1. **49/187 tests failing across 23/66 test files** — CI is effectively red
2. **2 TypeScript errors** in `vue-tsc --noEmit`:
   - `DockView.test.ts(12,18)`: `Property 'value' does not exist on type`
   - `CorrelationPanel.test.ts(7,1)`: `Cannot find name 'global'`
3. **Broken data.test.ts**: `fetchMarketOverview` test doesn't mock `window.go`, so the function returns early at a guard — assertions silently pass with null data (false positive)
4. **Missing HK/US engine tests**: Only CN engine has backtest tests
5. **Python sidecar tests are integration-only**: All require live gRPC server, no unit tests for gRPC service implementations

This blocks CI/CD and erodes confidence in all future changes.

## Design

### 1. Fix TypeScript Errors

**`DockView.test.ts(12,18)`:**
```typescript
// Error: Property 'value' does not exist on type
// Root cause: DockView.vue's `defineProps` returns a type where `value` is not exposed
// Fix: cast or access via proper type
const wrapper = mount(DockView, { props: { /* ... */ } })
// Before: wrapper.find(...).value  → After: wrapper.find(...) access via vm
```

**`CorrelationPanel.test.ts(7,1)`:**
```typescript
// Error: Cannot find name 'global'
// Root cause: vitest `global` type not recognized
// Fix: add `/* global */` comment or configure in tsconfig.json types
// Or: use `globalThis` instead
```

**Modified files:**
- `frontend/src/terminal/DockView/__tests__/DockView.test.ts`
- `frontend/src/terminal/panels/__tests__/CorrelationPanel.test.ts`

### 2. Fix Mock Infrastructure for `data.test.ts`

The test calls `fetchMarketOverview()` which internally checks `if (!window.go?.main?.App)` and returns early. The mock needs to set up `window.go.main.App.fetchMarketOverview`.

```typescript
// vitest.setup.ts
beforeEach(() => {
    ;(window as any).go = {
        main: {
            App: {
                fetchMarketOverview: vi.fn().mockResolvedValue({ /* mock data */ }),
                // ... all other methods
            }
        }
    }
})
```

**Modified files:**
- `frontend/src/__tests__/setup.ts` or `vitest.setup.ts` — Add comprehensive Wails mock
- `frontend/src/terminal/stores/data.test.ts` — Fix assertions to match real data shapes

### 3. Test Results Triage

For each of the 23 failing files:

| File | Likely Root Cause | Fix Strategy |
|------|-------------------|-------------|
| `data.test.ts` | Missing Wails mock | #2 above |
| `DockView.test.ts` | Type error | #1 above |
| `CorrelationPanel.test.ts` | Type error | #1 above |
| `WatchlistPanel.test.ts` | Stale mock data | Update mock shapes |
| Other 19 files | Varies (mock mismatch, API change) | Read each, fix mock or assertion |

**Approach:** Run `npx vitest run --reporter=verbose` to get the full failure list. Fix in tiers:
- Tier 1: Type errors (2 files, trivial)
- Tier 2: Mock infrastructure (1 file, unlocks many)
- Tier 3: Remaining 20 files (pair each failure with its code change)

### 4. Add HK/US Backtest Engine Tests

```go
func TestUSEngine_Run(t *testing.T) {
    // Basic US backtest: buy-and-hold on SPY-like data
    // Verify: no PDT trigger, fractional shares, no stamp duty
}

func TestUSEngine_PDTTrigger(t *testing.T) {
    // 4 day trades in 5 days with <$25k → PDT restricted
    // Verify: 5th day trade rejected
}

func TestHKEngine_Run(t *testing.T) {
    // Basic HK backtest: verify stamp duty (0.13%) and trade fee
}

func TestHKEngine_T2Settlement(t *testing.T) {
    // Sell → funds available after T+2
}
```

**Modified files:**
- `internal/backtest/engine_us_test.go` — Add 3+ test functions
- `internal/backtest/engine_hk_test.go` — Add 3+ test functions
- `internal/backtest/runner_test.go` — Augment existing (test multi-bar scenarios)

### 5. Python Sidecar Unit Tests

Add unit tests that don't require a live gRPC server:

```python
# tests/test_factor_registry.py  — Pure computation tests
def test_sma_factor():
    data = pd.DataFrame({"close": [1,2,3,4,5]})
    result = factor.registry.compute("sma_5", data)
    assert result is not None

# tests/test_llm_providers.py — Mock HTTP responses
@patch("httpx.AsyncClient.post")
def test_openai_chat(mock_post):
    mock_post.return_value.json.return_value = {...}
    provider = OpenAIProvider(api_key="test")
    response = await provider.chat([{"role": "user", "content": "hi"}])
    assert response is not None

# tests/test_factor_zoo.py — Test every registered factor
def test_all_factors_register():
    registry = FactorRegistry()
    assert len(registry.factors) >= 20  # all factors registered
```

**Modified files:**
- `python/tests/test_factor_registry.py` — New file
- `python/tests/test_llm_providers.py` — New file
- `python/tests/test_factor_zoo.py` — New file (or extend existing)

## Acceptance Criteria

- [ ] `npx vue-tsc --noEmit` passes with 0 errors
- [ ] `npx vitest run` passes with 0 failures (all 66 test files, 187+ tests)
- [ ] `cd internal/backtest && go test ./... -v -count=1` includes HK + US engine tests
- [ ] Python `pytest tests/ -x -q` passes without live gRPC server for pure unit tests
- [ ] CI pipeline is green after merge

## Risks / Trade-offs

- **Mock drift**: Fixing mocks to match current API shapes is correct, but mocks can silently diverge again. Mitigation: add a CI step that verifies mock shapes against real IPC calls.
- **Test quality over quantity**: Focus on fixing false positives (tests that pass with wrong data) rather than just making tests pass. A green suite with weak assertions is worse than a yellow suite.

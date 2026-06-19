# Market Wiring + Mootdx Client Cache — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Spec:** `docs/specs/2026-06-18-market-wiring-and-mootdx-cache.md`

**Goal:** (A) Cache the mootdx `Quotes` client + make `MootdxAdapter.IsAvailable` a cheap
nil-check; (B) wire `AdapterRegistry` + all adapters (incl. mootdx) into `App.startup()` and
expose `GetQuote` / `FetchOHLCV` IPC; (C) tests for both.

**Tech Stack:** Go 1.22+, Python 3.12+, stdlib `testing`, `unittest.mock`.

## Global Constraints
- TDD: write failing test → implement → pass → commit, one task at a time.
- No network calls in tests — mock the `DataClient` / mootdx client / registry.
- Python cache uses `threading.Lock`; Go `IsAvailable` must be lock-free.
- No proto change, no SQLite migration.
- Follow existing code idioms: `slog` logging, explicit error returns, table-driven Go tests.

---

### Task 1: Python — cache the mootdx Quotes client

**Files:**
- Modify: `python/src/data/fetcher.py`

Add a module-level cache + accessor after the `_init_mootdx_client` definition:

```python
import threading

_mootdx_client_lock = threading.Lock()
_mootdx_client = None


def _get_mootdx_client():
    """Return a cached mootdx Quotes client, creating it once.

    mootdx_config.setup() + Quotes.factory() are expensive (TCP probing); call them
    once and reuse. Guarded by a lock because mootdx client thread-safety is undocumented
    and gRPC may dispatch concurrent fetches. On a cached client we do NOT refresh
    automatically here — refresh-on-failure is handled by the caller via _reset_mootdx_client.
    """
    global _mootdx_client
    if _mootdx_client is not None:
        return _mootdx_client
    with _mootdx_client_lock:
        if _mootdx_client is None:
            _mootdx_client = _init_mootdx_client()
    return _mootdx_client


def _reset_mootdx_client():
    """Drop the cached client so the next fetch rebuilds it (call after a failure)."""
    global _mootdx_client
    with _mootdx_client_lock:
        _mootdx_client = None
```

In `_fetch_mootdx_ohlcv` and `_fetch_mootdx_quote`, replace `client = _init_mootdx_client()`
with `client = _get_mootdx_client()`. In the `except Exception` handlers around
`client.bars` / `client.minute`, after logging the warning, call `_reset_mootdx_client()`
so a broken client is rebuilt on the next request.

**Commit:** `perf(mootdx): cache Quotes client (avoid per-fetch setup() probe)`

---

### Task 2: Python — test the client cache

**Files:**
- Modify: `python/tests/test_data_fetcher.py`

Add two tests:

```python
@pytest.mark.asyncio
async def test_mootdx_client_is_cached():
    """_get_mootdx_client builds the client once and reuses it across fetches."""
    fake_client = MagicMock()
    fake_client.bars.return_value = _fake_bars_payload()
    with patch.object(fetcher, "_init_mootdx_client", return_value=fake_client) as init_mock, \
         patch.object(fetcher, "_HAS_MOOTDX", True):
        # Reset cache state for a clean test
        fetcher._reset_mootdx_client()
        svc = fetcher.DataService()
        await svc.FetchData(_make_bars_request("1D"), context=None)
        await svc.FetchData(_make_bars_request("1D"), context=None)
        await svc.FetchData(_make_bars_request("1D"), context=None)
    assert init_mock.call_count == 1, "client should be built once, not per fetch"
    fetcher._reset_mootdx_client()  # cleanup


@pytest.mark.asyncio
async def test_mootdx_client_reset_after_failure():
    """A bars() failure resets the cache so the next fetch rebuilds the client."""
    good = MagicMock()
    good.bars.return_value = _fake_bars_payload()
    bad = MagicMock()
    bad.bars.side_effect = RuntimeError("tdx connection reset")

    with patch.object(fetcher, "_init_mootdx_client", side_effect=[bad, good]), \
         patch.object(fetcher, "_HAS_MOOTDX", True):
        fetcher._reset_mootdx_client()
        svc = fetcher.DataService()
        first = await svc.FetchData(_make_bars_request("1D"), context=None)
        # first call fails (bad client), but the error is caught and cache reset
        second = await svc.FetchData(_make_bars_request("1D"), context=None)
    # second call should succeed with the rebuilt (good) client
    assert not second.error, f"expected recovery after reset, got: {second.error}"
    fetcher._reset_mootdx_client()
```

Run: `python -m pytest tests/test_data_fetcher.py -q` → all green.

**Commit:** `test(mootdx): assert Quotes client is cached and reset on failure`

---

### Task 3: Go — make IsAvailable a cheap nil-check

**Files:**
- Modify: `internal/market/adapters/mootdx.go`

Replace the `IsAvailable` body:

```go
func (a *MootdxAdapter) IsAvailable(ctx context.Context) bool {
	// Cheap check: a live TDX round-trip here would double the connections per quote
	// (registry probes IsAvailable, then calls FetchQuote). The real liveness signal is
	// FetchQuote itself; on failure the fallback chain moves to the next adapter.
	return a.dataClient != nil
}
```

**Commit:** `perf(mootdx): IsAvailable is a cheap nil-check (no TDX probe)`

---

### Task 4: Go — test IsAvailable nil-check semantics

**Files:**
- Modify: `internal/market/adapters/mootdx_test.go`

Add (no network, no real DataClient needed):

```go
func TestMootdxAdapter_IsAvailable_NilClientFalse(t *testing.T) {
	a := NewMootdxAdapter(nil)
	if a.IsAvailable(context.Background()) {
		t.Error("IsAvailable should be false when dataClient is nil")
	}
}

// isAvailableClient is a minimal stub satisfying *python.DataClient shape for the
// non-nil case. Because IsAvailable now only checks nil-ness, we can pass any non-nil
// pointer; we use a typed nil-free stub via a fake. See note below.
```

Since `IsAvailable` now only tests `a.dataClient != nil`, the non-nil case is verified by
constructing `NewMootdxAdapter(&python.DataClient{})` — but `DataClient` has unexported
fields. Instead, assert via a package-level helper: add a test-only constructor is overkill;
the nil case + the existing nil-bridge tests already cover the new semantics. Keep only the
nil test; document in the test comment that non-nil ⇒ true by construction.

**Commit:** `test(mootdx): IsAvailable returns false on nil dataClient (no network)`

---

### Task 5: Go — wire AdapterRegistry + adapters into App.startup

**Files:**
- Modify: `app.go`

Add import: `"quantflow/internal/market"` and `"quantflow/internal/market/adapters"`.

Add field to `App`:
```go
marketReg *market.AdapterRegistry
```

In `startup()`, after the bridge block (after `a.bridge = bridge` / the else), add:

```go
// Wire market-data adapters. mootdx uses the Python sidecar via DataClient;
// all others are pure-Go HTTP adapters. Adapters degrade gracefully when their
// backend is absent (mootdx with nil bridge → IsAvailable false → chain skips it).
var dataClient *python.DataClient
if a.bridge != nil {
	dataClient = python.NewDataClient(a.bridge)
}
a.marketReg = market.NewAdapterRegistry()
a.marketReg.Register(adapters.NewMootdxAdapter(dataClient))
a.marketReg.Register(adapters.NewSinaAdapter())
a.marketReg.Register(adapters.NewTuShareAdapter())
a.marketReg.Register(adapters.NewEastMoneyAdapter())
a.marketReg.Register(adapters.NewTencentAdapter())
a.marketReg.Register(adapters.NewBaiduAdapter())
a.marketReg.Register(adapters.NewAKShareAdapter())
a.marketReg.Register(adapters.NewYahooAdapter())
a.marketReg.Register(adapters.NewPolygonAdapter())
a.marketReg.Register(adapters.NewOKXAdapter())
a.marketReg.Register(adapters.NewBinanceAdapter())
a.marketReg.Register(adapters.NewCoinGeckoAdapter())
slog.Info("market adapter registry initialized", "count", a.marketReg.Count())
```

**Commit:** `feat(market): wire AdapterRegistry + all adapters into App.startup`

---

### Task 6: Go — expose GetQuote / FetchOHLCV IPC methods

**Files:**
- Modify: `app.go`

Add after `MarkNotificationRead`:

```go
// GetQuote fetches a real-time quote for a symbol via the market's fallback chain.
// Returns the snapshot and the name of the adapter that succeeded.
func (a *App) GetQuote(ctx context.Context, marketName, symbol string) (*market.QuoteSnapshot, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchQuoteWithFallback(ctx, marketName, symbol)
}

// FetchOHLCV fetches OHLCV bars for a symbol via the market's fallback chain.
// interval: "1D", "1W", "1M", "1m", "5m", "15m", "30m", "1H". start/end are Unix seconds.
// Returns the bars and the name of the adapter that succeeded.
func (a *App) FetchOHLCV(ctx context.Context, marketName, symbol, interval string, start, end int64) ([]market.OHLCVBar, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchOHLCVWithFallback(ctx, marketName, symbol, interval, start, end)
}
```

**Commit:** `feat(market): expose GetQuote + FetchOHLCV IPC to frontend`

---

### Task 7: Go — test App wiring + IPC routing

**Files:**
- Create: `app_test.go`

```go
package main

import (
	"context"
	"testing"

	"quantflow/internal/market"
)

func TestApp_RegistryWiresAllAdapters(t *testing.T) {
	a := &App{}
	// We can't call startup() (needs config/storage); instead directly replicate the
	// registration to assert the set. The real wiring is exercised via startup in a
	// smoke test if feasible; here we assert the public contract.
	a.marketReg = market.NewAdapterRegistry()
	// Reuse the same registration list as startup by calling a shared helper if
	// refactored; otherwise assert Count after a minimal registration.
	// (See Task 7b if a shared helper is extracted.)
}

func TestApp_GetQuote_NoRegistry(t *testing.T) {
	a := &App{}
	_, _, err := a.GetQuote(context.Background(), "CN", "600519")
	if err == nil {
		t.Fatal("GetQuote should error when registry is nil")
	}
}
```

Task 7b (refactor for testability): extract the adapter registration list in `app.go` into
a method `func (a *App) registerMarketAdapters(dc *python.DataClient)` so the test can call
it without `startup()`. Then `app_test.go` asserts `Count() == 12` and that `Get("mootdx")`
is non-nil with `IsAvailable()==false` when `dc==nil`.

**Commit:** `test(market): assert App registers all adapters and routes GetQuote/FetchOHLCV`

---

### Task 8: Docs + CHANGELOG + version date

**Files:**
- Modify: `CHANGELOG.md`

Under `[2026.6.18]`:

```markdown
### Added
- [MarketData] Wired AdapterRegistry into App.startup: all 12 adapters registered (mootdx, sina, tushare, eastmoney, tencent, baidu, akshare, yahoo, polygon, okx, binance, coingecko). The CN fallback chain (mootdx first) now takes effect at runtime.
- [MarketData] Exposed Wails IPC methods App.GetQuote and App.FetchOHLCV — frontend/dataStore can now pull real quotes and K-line via the fallback chain (previously placeholder-only).

### Changed
- [MarketData] MootdxAdapter.IsAvailable is now a cheap nil-check (no TDX round-trip); avoids doubling TDX connections per quote (registry probes IsAvailable then calls FetchQuote).
- [Python] mootdx Quotes client is now cached (module-level, lock-guarded) instead of rebuilt per fetch; setup()/factory() run once. Broken client is reset on failure and rebuilt next call.
```

Verify version date in `frontend/package.json`, `README.md` badge, and CHANGELOG header all
show today's date.

**Commit:** `docs: update CHANGELOG for market wiring + mootdx client cache`

---

## Execution order
Tasks 1→2 (Python cache), 3→4 (Go IsAvailable), 5→6 (wiring + IPC), 7 (tests), 8 (docs).
Each task: implement → test → commit. Review gate between tasks per subagent-driven-development.

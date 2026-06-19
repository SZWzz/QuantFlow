# Wire Market Data Subsystem into App + Cache Mootdx Client

## Motivation

The Mootdx adapter (commits b1ec058 / bb5342b) shipped adapter + Python sidecar + tests, but
**is never constructed in production**: `NewMootdxAdapter` and `NewDataClient` have zero
non-test call sites, and the `AdapterRegistry` is never populated at runtime. The headline
change — `mootdx` placed first in the CN `FallbackChains` — therefore has no runtime effect:
`Get("mootdx")` returns nil and is skipped.

More broadly, the entire market-data subsystem is unwired from the Wails app:
- `App` has no `MarketDataHub` or `AdapterRegistry` field.
- No IPC method exposes quotes / OHLCV to the frontend.
- `ai/capabilities/quote.go` ships placeholder data ("quotes use placeholder data until
  MarketDataHub integration").
- The frontend `dataStore` is fed from the UI side, not from real adapters.

This spec wires the subsystem in so the fallback chain is real, and fixes a performance
defect in the same Mootdx feature: every mootdx fetch rebuilds the `Quotes` client (running
the expensive `mootdx_config.setup()` probe) and `IsAvailable` does a live TDX round-trip,
so each CN quote opens **2 fresh TDX TCP connections** (probe + real fetch).

## Design

### Phase A — Cache the mootdx Quotes client + cheap IsAvailable (issue 4)

Python side (`python/src/data/fetcher.py`):
- Module-level cached client behind a lock: `_get_mootdx_client()` calls
  `_init_mootdx_client()` once, caches the result, refreshes on failure.
  `_fetch_mootdx_ohlcv` / `_fetch_mootdx_quote` call `_get_mootdx_client()` instead of
  `_init_mootdx_client()`.

Go side (`internal/market/adapters/mootdx.go`):
- `IsAvailable` becomes a cheap check: returns `a.dataClient != nil` (no network round-trip).
  The Python side's retry/timeout + the real `FetchQuote` failure already surface
  unavailability; the registry's `FetchQuote` call is the real liveness signal.
  Rationale: probing every adapter per quote doubles TDX connections and adds latency.

Concurrency note (rule 4 critical): the Python cache uses a `threading.Lock` (mootdx
client is not guaranteed thread-safe). The Go `IsAvailable` is now lock-free (nil check).

### Phase B — Wire AdapterRegistry + MootdxAdapter into the App (issue 2)

`app.go`:
- New `App` field: `marketReg *market.AdapterRegistry`.
- In `startup()`, after the bridge is (optionally) created:
  - `a.marketReg = market.NewAdapterRegistry()`
  - Build `dataClient` only if `a.bridge != nil`: `dc := python.NewDataClient(a.bridge)`,
    else `dc := (*python.DataClient)(nil)`.
  - Register all adapters in dependency order. Mootdx gets the `dataClient` (or nil →
    graceful degradation). All others take no args:
    ```go
    a.marketReg.Register(adapters.NewMootdxAdapter(dc))
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
    ```
- New exported IPC methods (Wails-bound, frontend-callable):
  - `GetQuote(ctx, market, symbol string) (*market.QuoteSnapshot, string, error)` —
    delegates to `marketReg.FetchQuoteWithFallback`, returns the snapshot + which adapter
    succeeded.
  - `FetchOHLCV(ctx, market, symbol, interval string, start, end int64) ([]market.OHLCVBar, string, error)`
    — delegates to `marketReg.FetchOHLCVWithFallback`.

Data flow:
```
Vue panel / dataStore
   └─ Wails IPC → App.GetQuote / App.FetchOHLCV
        └─ AdapterRegistry.FetchQuoteWithFallback (CN chain: mootdx→sina→…)
             └─ MootdxAdapter.FetchQuote → DataClient.FetchData (gRPC)
                  └─ Python DataService → cached mootdx Quotes client → TDX TCP
```

No SQLite schema change. No proto change (FetchDataRequest already has `params`).

### Phase C — Tests

- `mootdx.go`: add a mock-`DataClient` test asserting `IsAvailable` returns false on nil
  client and true on non-nil (no network). (Current nil-bridge test stays.)
- Python: add a test asserting `_get_mootdx_client` caches — second call returns the same
  object and `_init_mootdx_client` is invoked once (`patch` the init, count calls).
- `app.go` wiring: add `app_test.go` asserting `startup()` registers all 12 adapters and
  that `GetQuote`/`FetchOHLCV` route through the registry (mock the registry or assert the
  call path via a registered stub adapter).

## Acceptance Criteria

- [ ] `IsAvailable` on `MootdxAdapter` performs no network round-trip (nil-check only).
- [ ] Python `_get_mootdx_client` calls `_init_mootdx_client` exactly once across many
  fetches (cached); a failed client is refreshed on the next call.
- [ ] `App.startup()` registers ≥12 adapters including `mootdx`.
- [ ] `App.GetQuote("CN", "600519")` reaches `FetchQuoteWithFallback` and the CN chain is
  tried in order (mootdx first); with nil bridge, mootdx is skipped (IsAvailable false) and
      the chain falls through.
- [ ] `App.FetchOHLCV("CN", "600519", "1W", start, end)` returns weekly bars (interval
  forwarded end-to-end via the issue-1 fix).
- [ ] All existing Go + Python tests still pass; new tests pass.
- [ ] CHANGELOG updated; version date current.

## Risks / Trade-offs

- **`IsAvailable` no longer probes the network.** A registered-but-dead adapter now reaches
  `FetchQuote` before failing, then the chain moves on. Net: one failed `FetchQuote` per
  dead adapter per call instead of one failed `IsAvailable`. Acceptable — and strictly
  better for mootdx (avoids the 2× connection). Other adapters already did cheap
  `IsAvailable` checks, so behavior change is minimal for them.
- **Mootdx client caching + thread safety.** mootdx's `Quotes` client thread-safety is
  undocumented; the Python sidecar is async-single-threaded per request, but gRPC may
  concurrency. Mitigated by a `threading.Lock` around client creation/refresh; the cached
  client is reused read-only after creation. If contention appears, per-request clients can
  be reintroduced behind the cache.
- **Scope creep risk.** Full `MarketDataHub` pub/sub integration with the frontend
  `dataStore` is out of scope here — this spec only adds synchronous IPC `GetQuote`/
  `FetchOHLCV`. The Hub + real-time push is a separate later phase. Keeping the blast
  radius small.
- **All-other-adapters registration** means network-capable adapters (yahoo, binance, …)
  become live in the chain. They were already implemented and tested; this just turns them
  on. Their `IsAvailable`/fetch may hit real networks in dev — acceptable for a terminal
  app, and the fallback chain handles failures.

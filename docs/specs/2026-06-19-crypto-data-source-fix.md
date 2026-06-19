# Crypto Market Data Source Fix

## Diagnosis

All 3 existing crypto adapters fail from mainland China:
- **Binance** (`api.binance.com`) — timeout
- **OKX** (`okx.com`) — timeout  
- **CoinGecko** (`api.coingecko.com`) — timeout

Root cause: these domains are blocked or throttled by the Great Firewall.

## Solution: Gate.io Adapter

**Gate.io** (`api.gateio.ws`) is accessible from China without VPN. Public REST endpoints require no API key.

### API Endpoints

| Endpoint | URL | Purpose |
|----------|-----|---------|
| Ticker | `/api/v4/spot/tickers?currency_pair=BTC_USDT` | Real-time quote |
| Candlesticks | `/api/v4/spot/candlesticks?currency_pair=BTC_USDT&interval=1d&limit=100` | OHLCV |
| Currency Pairs | `/api/v4/spot/currency_pairs` | Available pairs list |

### Symbol Format

Gate.io uses `BTC_USDT` (underscore). QuantFlow's existing adapters use `BTCUSDT` (no separator). Adapter will auto-convert.

### K-line Format

```json
[[timestamp, volume_quote, close, high, low, open, volume_base, is_complete]]
```

### Rate Limits

~200 requests per 10 seconds for public endpoints. Sufficient for individual use.

## Files

- Create: `internal/market/adapters/gateio.go` — GateIOAdapter
- Create: `internal/market/adapters/gateio_test.go` — unit tests
- Modify: `app.go` — register GateIOAdapter in crypto fallback chain (before Binance/OKX)
- Modify: `CHANGELOG.md`

## Acceptance Criteria

- [ ] `GateIOAdapter.FetchQuote("BTCUSDT")` returns real-time BTC price
- [ ] `GateIOAdapter.FetchOHLCV("BTCUSDT", "1d", ...)` returns OHLCV bars
- [ ] Live test passes from China
- [ ] Unit tests pass with mock server
- [ ] Registered in adapter registry (first in crypto chain)
- [ ] `go vet` clean, no test regressions

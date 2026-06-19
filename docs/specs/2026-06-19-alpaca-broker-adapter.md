# Alpaca Broker Adapter Spec

## Motivation

QuantFlow currently has two broker adapters: Futu (HK, stub) and Binance (Crypto, real REST). No US market broker. Alpaca Markets provides a free Paper Trading API with a clean REST interface — ideal for US equities trading. The 5 new trading panels (OrderBlotter, Execution, BasketOrder, BrokerStatus, ActionCenter) need real order/position data to be more than mock shells.

## Design

### Alpaca API

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v2/account` | GET | Account info (cash, buying power, portfolio value) |
| `/v2/orders` | GET/POST | List/submit orders |
| `/v2/orders/{id}` | GET/DELETE | Get/cancel specific order |
| `/v2/positions` | GET | List open positions |
| `/v2/positions/{symbol}` | GET/DELETE | Get/close position |
| `/v2/clock` | GET | Market open/close status |

**Auth**: Headers `APCA-API-KEY-ID` + `APCA-API-SECRET-KEY`. Paper URL: `https://paper-api.alpaca.markets`. Live URL: `https://api.alpaca.markets`.

### Config

```go
type AlpacaConfig struct {
    APIKey    string // APCA-API-KEY-ID
    SecretKey string // APCA-API-SECRET-KEY
    BaseURL   string // default: https://paper-api.alpaca.markets
}
```

Env vars: `ALPACA_API_KEY`, `ALPACA_SECRET_KEY`, `ALPACA_BASE_URL` (optional, defaults to paper).

### Implementation

`AlpacaBroker` implements `trading.Broker` with real HTTP calls. Follows `BinanceBroker` pattern:

```
AlpacaBroker struct {
    cfg       AlpacaConfig
    client    *http.Client (30s timeout)
    connected bool
    mu        sync.RWMutex
    orderCbs  []func(*trading.Order)
    tradeCbs  []func(*trading.Trade)
}
```

All methods perform HTTP requests with API key headers. Errors from Alpaca API (4xx/5xx) are wrapped with context.

### Symbol Mapping

Alpaca uses standard US tickers (AAPL, TSLA, NVDA). No conversion needed for US stocks. The `Broker` interface receives symbols already normalized.

### Data Flow

```
OrderBlotterPanel → portfolioStore.fetchOrders()
  → App.GetOrders() → TradingHub → OMS → AlpacaBroker.GetOrders()
                                        → GET /v2/orders → JSON parse → []*trading.Order
```

### Graceful Degradation

If `ALPACA_API_KEY` not set, `Connect()` returns error "alpaca: API key not configured". All other methods return descriptive errors. BrokerStatusPanel shows "Not Configured" state.

## Files

### New
- `internal/trading/brokers/alpaca.go` — AlpacaBroker struct + all interface methods
- `internal/trading/brokers/alpaca_test.go` — Unit tests (mock HTTP server)

### Modified
- `app.go` — Create AlpacaBroker in startup(), wire to TradingHub
- `CHANGELOG.md` — Record addition

## Acceptance Criteria

- [ ] `AlpacaBroker` implements all 11 `trading.Broker` interface methods
- [ ] `Connect()` hits `/v2/clock` to verify connectivity and API key validity
- [ ] `GetAccount()` returns `AccountInfo` with buying power, cash, portfolio value
- [ ] `SubmitOrder()` posts to `/v2/orders` with symbol/qty/side/type/time_in_force
- [ ] `CancelOrder()` sends DELETE to `/v2/orders/{id}`
- [ ] `GetOrders()` returns order list with status filter support
- [ ] `GetPositions()` returns open positions
- [ ] Unit tests with `httptest.Server` mock pass
- [ ] `go vet ./internal/trading/...` clean
- [ ] `go test ./internal/trading/...` passes (no regressions)
- [ ] AlpacaBroker is created and wired in `app.go` startup()

## Risks / Trade-offs

- **Paper trading only by default**: Live trading requires explicit `ALPACA_BASE_URL` override. Safe default.
- **No WebSocket**: This spec covers REST only. Alpaca WebSocket streaming (account/order updates) can be a follow-up.
- **US stocks only**: Alpaca supports US equities/options/crypto. Initial implementation focuses on equities.
- **Rate limits**: Alpaca allows 200 requests/min. Our 30s auto-refresh from frontend is well within limits.

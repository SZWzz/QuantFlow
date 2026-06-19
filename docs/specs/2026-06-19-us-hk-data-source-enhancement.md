# US/HK Market Data Source Enhancement Spec

## Motivation

Current adapter coverage is heavily skewed toward A-shares (20+ adapters). US market has only 3 adapters (Yahoo/broken, Polygon/needs key, CoinGecko/crypto-only) and HK market has zero dedicated adapters. The new Alpaca broker provides US order execution — but without reliable US market data, users can't make informed trading decisions.

## Research Findings

### US Market — Free Data Sources

| Source | Type | Real-time | Rate Limit | Key Required |
|--------|------|-----------|------------|--------------|
| **Yahoo Finance v8** | OHLCV + Quote | ~15min delay | ~2 req/s | No |
| **Finnhub** | Quote + OHLCV + News | Real-time | 60/min free | Yes (free tier) |
| **Alpha Vantage** | OHLCV + Fundamentals | 15min delay | 25/day free | Yes (free tier) |
| **Polygon** | Quote + OHLCV + News | Real-time | 5/min free | Yes (free tier) |
| **Tiingo** | OHLCV + Fundamentals | EOD | 50/hr free | Yes (free tier) |

**Recommendation**: Fix Yahoo (no key, best coverage) → Add Finnhub (real-time supplement with free key) → Polygon already exists.

### HK Market — Free Data Sources

| Source | Type | Real-time | Key Required |
|--------|------|-----------|--------------|
| **AkShare** (Python) | Quote + OHLCV | Near real-time | No |
| **Yahoo Finance** (`0700.HK`) | OHLCV + Quote | 15min delay | No |
| **Futu OpenD** (already integrated) | Quote + OHLCV + Depth | Real-time | Futu account |
| **新浪港股** | Quote + fundamentals | Near real-time | No |
| **腾讯港股** | Quote + K-line | Near real-time | No |

**Recommendation**: Extend AkShare → Fix Yahoo for HK tickers → Enable Futu market data (already have broker connection) → Add Sina HK.

## Design

### Phase 1: Quick Wins (1-2 files each)

#### 1a. Fix Yahoo Adapter
**Problem**: `yahoo.go` returns HTTP 403 from China. Yahoo's v8 endpoint requires proper User-Agent + crumb cookie.
**Fix**: 
- Add `User-Agent` header (simulate browser)
- Add crumb acquisition flow (GET `/v1/test/getcrumb` with cookie jar)
- Fallback to v7 endpoint if v8 fails
- Add `Accept: application/json` header
**File**: `internal/market/adapters/yahoo.go`

#### 1b. Extend AkShare for HK Stocks
AkShare Python library already supports `ak.stock_hk_spot_em()` and `ak.stock_hk_hist()`. Our existing `akshare.go` adapter only calls CN endpoints.
**Fix**:
- Add `FetchQuote()` and `FetchOHLCV()` calls for HK market in Python side
- Update Go adapter to route HK symbols to new Python functions
**Files**: `internal/market/adapters/akshare.go`, `python/src/data/fetcher.py`

#### 1c. Add Finnhub US Adapter
Finnhub provides free real-time US quotes with a free API key.
**New file**: `internal/market/adapters/finnhub.go`
**Auth**: `X-Finnhub-Token: <FINNHUB_API_KEY>` header
**Endpoints**: `/quote?symbol=AAPL`, `/stock/candle?symbol=AAPL&resolution=D`

### Phase 2: Deeper Integration

#### 2a. Enable Futu Market Data
The Futu broker already exists (stub). FutuOpenD provides extensive market data APIs:
- `GetMarketSnapshot` — real-time quotes
- `GetCurKline` — K-line data
- `GetOrderBook` — depth of market
**Approach**: Add `FutuMarketDataAdapter` that wraps the existing FutuOpenD connection.

#### 2b. Sina HK + Tencent HK
新浪 and 腾讯 both provide free HK stock data via HTTP endpoints similar to their A-share endpoints.
**New files**: `sina_hk.go`, `tencent_hk.go` (or extend existing adapters)

### Data Flow

```
US Stock Query
  → MarketDataHub
    → YahooAdapter.FetchQuote("AAPL")     [no key, primary]
    → FinnhubAdapter.FetchQuote("AAPL")    [free key, fallback]
    → PolygonAdapter.FetchQuote("AAPL")    [existing, key-required]
    → AlpacaBroker (for trading)

HK Stock Query
  → MarketDataHub
    → AkShareAdapter.FetchQuote("00700")   [no key, primary]
    → YahooAdapter.FetchQuote("0700.HK")   [no key, fallback]
    → FutuMarketDataAdapter                [existing connection]
    → SinaHKAdapter / TencentHKAdapter     [no key, fallback]
```

## Files

### Phase 1
- Modify: `internal/market/adapters/yahoo.go` — fix HTTP 403
- Modify: `internal/market/adapters/yahoo_test.go` — update tests
- Modify: `internal/market/adapters/akshare.go` — add HK routing
- Modify: `python/src/data/fetcher.py` — add HK functions
- Create: `internal/market/adapters/finnhub.go`
- Create: `internal/market/adapters/finnhub_test.go`

### Phase 2
- Create: `internal/market/adapters/futu_market.go` — FutuOpenD market data
- Create/modify: Sina HK + Tencent HK endpoints

## Acceptance Criteria

### Phase 1
- [ ] `YahooAdapter.FetchQuote("AAPL")` returns quote without HTTP 403
- [ ] `YahooAdapter.FetchOHLCV("AAPL", ...)` returns OHLCV bars
- [ ] `AkShareAdapter.FetchQuote("00700")` returns HK stock quote
- [ ] `AkShareAdapter.FetchOHLCV("00700", ...)` returns HK OHLCV
- [ ] `FinnhubAdapter.FetchQuote("AAPL")` returns US real-time quote
- [ ] All existing adapter tests pass (no regressions)
- [ ] `go test ./internal/market/...` passes

### Phase 2
- [ ] `FutuMarketDataAdapter` provides HK real-time quotes via FutuOpenD
- [ ] Sina HK adapter provides HK stock fundamentals
- [ ] Tencent HK adapter provides HK K-line data

## Risks / Trade-offs

- **Yahoo fix**: Yahoo may change their crumb mechanism again. Keep fallback chain short.
- **AkShare for HK**: Requires Python sidecar. Gracefully degrade when sidecar offline.
- **Finnhub free tier**: 60 req/min is sufficient for individual use but not for heavy screeners.
- **FutuOpenD market data**: Requires Futu account + FutuOpenD running. Already a dependency for Futu broker.
- **Hong Kong real-time is expensive**: True HKEX real-time requires licensing. Our free sources will be near-real-time (1-5 min delay).

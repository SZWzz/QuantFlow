package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
	"quantflow/internal/python"
)

// registerMarketAdapters populates the adapter registry with every data source,
// in the order the fallback chains expect (see market.FallbackChains). mootdx
// alone needs the Python sidecar; it degrades to IsAvailable()==false when the
// bridge is absent. Extracted from startup() so tests can exercise registration
// without config/storage.
func (a *App) registerMarketAdapters() {
	var dataClient *python.DataClient
	if a.bridge != nil {
		dataClient = python.NewDataClient(a.bridge)
	}
	// CN chain: tencent(quickest)→eastmoney→mootdx(intraday)→...
	// Tencent ~76ms HTTP, EastMoney ~350ms HTTPS, mootdx ~4s via Python sidecar.
	a.marketReg.Register(adapters.NewMootdxAdapter(dataClient))
	a.marketReg.Register(adapters.NewSinaAdapter())
	a.marketReg.Register(adapters.NewTuShareAdapter())
	a.marketReg.Register(adapters.NewEastMoneyAdapter())
	a.marketReg.Register(adapters.NewTencentAdapter())
	a.marketReg.Register(adapters.NewBaiduAdapter())
	a.marketReg.Register(adapters.NewAKShareAdapter())
	// US / HK / CRYPTO chains.
	a.marketReg.Register(adapters.NewYahooAdapter())
	finnhubAdpt := adapters.NewFinnhubAdapter()
	finnhubAdpt.SetAPIKey(a.cfg.GetAPIKey("finnhub"))
	a.marketReg.Register(finnhubAdpt)
	a.marketReg.Register(adapters.NewPolygonAdapter())
	a.marketReg.Register(adapters.NewGateIOAdapter()) // primary crypto (accessible from CN)
	a.marketReg.Register(adapters.NewOKXAdapter())
	a.marketReg.Register(adapters.NewBinanceAdapter())
	a.marketReg.Register(adapters.NewCoinGeckoAdapter())
}

// GetQuote fetches a real-time quote for a symbol via the market's fallback
// chain (e.g. "CN" → mootdx→sina→tushare→…). Returns the snapshot and the name
// of the adapter that succeeded. marketName is one of "CN", "US", "HK", "CRYPTO".
func (a *App) GetQuote(ctx context.Context, marketName, symbol string) (*market.QuoteSnapshot, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchQuoteWithFallback(ctx, marketName, symbol)
}

// GetMinuteLine returns today's intraday minute-by-minute ticks for a CN symbol.
// If sinceTimestamp is 0, returns all ticks for today.
// If sinceTimestamp > 0, returns only ticks after the given Unix timestamp.
// Data is cached in SQLite + LRU; source data comes from mootdx when not cached.
func (a *App) GetMinuteLine(ctx context.Context, symbol string, sinceTimestamp int64) ([]market.MinuteTick, string, error) {
	if a.minuteCache == nil {
		return nil, "unavailable", fmt.Errorf("minute cache not initialized")
	}

	mkt := market.MarketForSymbol(symbol)

	// Non-CN markets: minute data not available via free adapters,
	// return daily OHLCV as fallback (frontend will display as daily bars).
	if mkt != "CN" {
		return nil, "unavailable", fmt.Errorf("minute data not available for market %s, use 1d interval instead", mkt)
	}

	// 1. Try cache first (SQLite + LRU).
	ticks, err := a.minuteCache.GetIncremental(symbol, sinceTimestamp)
	if err != nil {
		slog.Warn("minute_cache: get failed", "symbol", symbol, "err", err)
	}

	// 2. If cache has data and the request is incremental (since > 0),
	//    return cached data. For initial load (since == 0), if cache
	//    is empty, fall through to live fetch.
	if len(ticks) > 0 || sinceTimestamp > 0 {
		return ticks, "cache", nil
	}

	// 3. Live fetch via mootdx (CN only).
	adpt := a.getMootdxAdapter()
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	liveTicks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}

	// 4. Persist live data to cache.
	if len(liveTicks) > 0 {
		today := time.Now().Format("2006-01-02")
		if err := a.minuteCache.SaveTicks(symbol, today, liveTicks); err != nil {
			slog.Warn("minute_cache: save failed", "symbol", symbol, "err", err)
		}
	}

	return liveTicks, "mootdx", nil
}

// FetchOHLCV fetches OHLCV bars for a symbol via the market's fallback chain.
// interval is one of "1D", "1W", "1M", "1m", "5m", "15m", "30m", "1H"; start/end
// are Unix timestamps in seconds. Returns the bars and the adapter name that
// succeeded.
func (a *App) FetchOHLCV(ctx context.Context, marketName, symbol, interval string, start, end int64) ([]market.OHLCVBar, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchOHLCVWithFallback(ctx, marketName, symbol, interval, start, end)
}

// GetFundFlow returns capital flow data for a symbol.
// flowType: "minute" (今日分钟级) or "daily" (120日日级).
func (a *App) GetFundFlow(symbol string, flowType string) (interface{}, error) {
	if a.fundFlowSvc == nil {
		return nil, fmt.Errorf("fund flow service not initialized")
	}
	ctx := context.Background()
	switch flowType {
	case "minute":
		return a.fundFlowSvc.GetMinuteFlow(ctx, symbol)
	case "daily":
		return a.fundFlowSvc.GetDailyFlow(ctx, symbol)
	default:
		return nil, fmt.Errorf("invalid flowType: %s (use 'minute' or 'daily')", flowType)
	}
}

// GetNorthboundFlow returns northbound capital flow data.
func (a *App) GetNorthboundFlow() (map[string]interface{}, error) {
	if a.northboundSvc == nil {
		return nil, fmt.Errorf("northbound service not initialized")
	}
	ctx := context.Background()
	result := map[string]interface{}{}
	if data, err := a.northboundSvc.GetMinuteFlow(ctx); err == nil {
		result["minute_flow"] = data
	}
	if data, err := a.northboundSvc.GetHistory(20); err == nil {
		result["history"] = data
	}
	return result, nil
}

// getMootdxAdapter retrieves the mootdx adapter from the market registry.
// Returns nil if the Python sidecar is not connected.
func (a *App) getMootdxAdapter() *adapters.MootdxAdapter {
	if a.marketReg == nil {
		return nil
	}
	adpt := a.marketReg.Get("mootdx")
	if adpt == nil {
		return nil
	}
	mootdx, ok := adpt.(*adapters.MootdxAdapter)
	if !ok {
		return nil
	}
	return mootdx
}

// getMarketReg returns the market adapter registry or nil.
func (a *App) getMarketReg() *market.AdapterRegistry {
	return a.marketReg
}

// GetMarketOverview returns major market indices for the given market.
func (a *App) GetMarketOverview(mkt string) (map[string]interface{}, error) {
	ctx := context.Background()
	type idxDef struct{ code, name string }
	var indices []idxDef
	switch mkt {
	case "HK":
		indices = []idxDef{
			{"^HSI", "恒生指数"},
			{"^HSCE", "国企指数"},
			{"^HSTECH", "恒生科技"},
		}
	case "US":
		indices = []idxDef{
			{"^GSPC", "S&P 500"},
			{"^IXIC", "NASDAQ"},
			{"^DJI", "Dow Jones"},
		}
	default:
		indices = []idxDef{
			{"000001.SH", "上证指数"},
			{"399001.SZ", "深证成指"},
			{"399006.SZ", "创业板指"},
			{"000688.SH", "科创50"},
			{"000300.SH", "沪深300"},
		}
	}
	sina := a.marketReg.Get("sina")
	result := make([]map[string]interface{}, 0, len(indices))
	for _, idx := range indices {
		var snap *market.QuoteSnapshot
		var err error
		if mkt == "CN" || mkt == "" {
			if sina != nil && sina.IsAvailable(ctx) {
				sc := idx.code
				parts := strings.Split(sc, ".")
				if len(parts) == 2 {
					sc = strings.ToLower(parts[1]) + parts[0]
				}
				snap, err = sina.FetchQuote(ctx, sc)
			} else {
				snap, _, err = a.GetQuote(ctx, "CN", idx.code)
			}
		} else {
			snap, _, err = a.GetQuote(ctx, mkt, idx.code)
		}
		if err != nil {
			slog.Warn("GetMarketOverview: failed for", "code", idx.code, "error", err)
			continue
		}
		result = append(result, map[string]interface{}{
			"code":       idx.code,
			"name":       idx.name,
			"price":      snap.Last,
			"change":     snap.Change,
			"change_pct": snap.ChangePct,
		})
	}
	return map[string]interface{}{
		"indices": result,
		"breadth": map[string]int{"advancers": 0, "decliners": 0, "unchanged": 0},
	}, nil
}

// GetCryptoOverview returns quotes for major crypto pairs.
func (a *App) GetCryptoOverview(ctx context.Context, symbols []string) (map[string]interface{}, error) {
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "DOTUSDT"}
	}
	reg := a.getMarketReg()
	results := make([]map[string]interface{}, 0)
	for _, sym := range symbols {
		snap, _, err := reg.FetchQuoteWithFallback(ctx, "CRYPTO", sym)
		if err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"symbol":     sym,
			"price":      snap.Last,
			"change_pct": snap.ChangePct,
		})
	}
	return map[string]interface{}{"cryptos": results}, nil
}

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
	pb "quantflow/internal/python/proto"
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
	a.marketReg.Register(adapters.NewBinanceFuturesAdapter())
	a.marketReg.Register(adapters.NewCoinGeckoAdapter())
}

// GetQuote fetches a real-time quote for a symbol via the market's fallback
// chain (e.g. "CN" → mootdx→sina→tushare→…). Returns the snapshot and the name
// of the adapter that succeeded. marketName is one of "CN", "US", "HK", "CRYPTO".
func (a *App) GetQuote(ctx context.Context, marketName, symbol string) (*market.QuoteSnapshot, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	quote, adapter, err := a.marketReg.FetchQuoteWithFallback(ctx, marketName, symbol)
	if err != nil {
		return nil, adapter, err
	}
	if a.oms != nil && quote != nil && quote.Name != "" {
		a.oms.SetQuoteName(symbol, quote.Name)
	}
	return quote, adapter, nil
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

	if mkt != "CN" {
		return nil, "unavailable", fmt.Errorf("minute data not available for market %s, use 1d interval instead", mkt)
	}

	ticks, err := a.minuteCache.GetIncremental(symbol, sinceTimestamp)
	if err != nil {
		slog.Warn("minute_cache: get failed", "symbol", symbol, "err", err)
	}

	adpt := a.getMootdxAdapter()

	// Incremental request (sinceTimestamp > 0): if cache already has new data,
	// return it fast. Otherwise try live fetch to refresh the cache, then re-query.
	if sinceTimestamp > 0 {
		if len(ticks) > 0 {
			return ticks, "cache", nil
		}
		if adpt != nil {
			if liveTicks, err := adpt.FetchMinuteLine(symbol); err == nil && len(liveTicks) > 0 {
				today := time.Now().Format("2006-01-02")
				if saveErr := a.minuteCache.SaveTicks(symbol, today, liveTicks); saveErr != nil {
					slog.Warn("minute_cache: save failed", "symbol", symbol, "err", saveErr)
				}
				freshTicks, _ := a.minuteCache.GetIncremental(symbol, sinceTimestamp)
				return freshTicks, "mootdx", nil
			}
		}
		return ticks, "cache", nil
	}

	// Initial load (sinceTimestamp == 0): serve from cache if available.
	if len(ticks) > 0 {
		return ticks, "cache", nil
	}

	// Live fetch via mootdx.
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	liveTicks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}

	if len(liveTicks) > 0 {
		today := time.Now().Format("2006-01-02")
		if saveErr := a.minuteCache.SaveTicks(symbol, today, liveTicks); saveErr != nil {
			slog.Warn("minute_cache: save failed", "symbol", symbol, "err", saveErr)
		}
		return liveTicks, "mootdx", nil
	}

	recentTicks, recentDate, err := a.minuteCache.GetRecentTicks(symbol, 5)
	if err != nil {
		slog.Warn("minute_cache: recent lookup failed", "symbol", symbol, "err", err)
	}
	if len(recentTicks) > 0 {
		slog.Info("minute_cache: using recent data", "symbol", symbol, "date", recentDate)
		return recentTicks, "cache", nil
	}

	return nil, "unavailable", fmt.Errorf("no minute data available for %s (market closed)", symbol)
}

// FetchOHLCV fetches OHLCV bars for a symbol via the market's fallback chain.
// interval is one of "1D", "1W", "1M", "1m", "5m", "15m", "30m", "1H"; start/end
// are Unix timestamps in seconds. fqfactor controls price adjustment (复权):
// "" (不复权), "qfq" (前复权), "hfq" (后复权) — only applicable to CN-market adapters.
// Returns the bars and the adapter name that succeeded.
func (a *App) FetchOHLCV(ctx context.Context, marketName, symbol, interval, fqfactor string, start, end int64) ([]market.OHLCVBar, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchOHLCVWithFallback(ctx, marketName, symbol, interval, fqfactor, start, end)
}

// FetchData proxies a data request to the Python sidecar's DataService gRPC endpoint.
// Supported sources: mootdx, akshare, ccxt, sec, macro.
// dataType varies per source (e.g. "financials", "fundflow", "ticker", "financials").
func (a *App) FetchData(source, dataType string, symbols []string, startDate, endDate string, params map[string]string) (map[string]interface{}, error) {
	if a.bridge == nil {
		return nil, fmt.Errorf("Python sidecar not available")
	}
	ctx := context.Background()
	req := &pb.FetchDataRequest{
		Source:    source,
		DataType:  dataType,
		Symbols:   symbols,
		StartDate: startDate,
		EndDate:   endDate,
		Params:    params,
	}
	resp, err := a.bridge.DataClient.FetchData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("FetchData(%s/%s): %w", source, dataType, err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("FetchData(%s/%s): %s", source, dataType, resp.Error)
	}
	return map[string]interface{}{
		"data":         string(resp.Data),
		"source":       resp.Source,
		"fetch_time_ms": resp.FetchTimeMs,
	}, nil
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

// GetCryptoFundingRates returns perpetual swap funding rates for crypto symbols.
func (a *App) GetCryptoFundingRates(ctx context.Context, symbols []string) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	// Use binance_futures adapter directly (we need specific funding rate API, not quote fallback)
	adpt := reg.Get("binance_futures")
	if adpt == nil {
		return nil, fmt.Errorf("binance_futures adapter not available")
	}
	bf, ok := adpt.(*adapters.BinanceFuturesAdapter)
	if !ok {
		return nil, fmt.Errorf("binance_futures adapter type assertion failed")
	}
	rates, err := bf.FetchFundingRates(ctx, symbols)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(rates))
	for _, r := range rates {
		result = append(result, map[string]interface{}{
			"symbol":            r.Symbol,
			"mark_price":        r.MarkPrice,
			"index_price":       r.IndexPrice,
			"funding_rate":      r.FundingRate,
			"next_funding_time": r.NextFundingTime,
		})
	}
	return result, nil
}

// GetCryptoLiquidations returns recent forced liquidation orders for a crypto symbol.
func (a *App) GetCryptoLiquidations(ctx context.Context, symbol string, limit int) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	adpt := reg.Get("binance_futures")
	if adpt == nil {
		return nil, fmt.Errorf("binance_futures adapter not available")
	}
	bf, ok := adpt.(*adapters.BinanceFuturesAdapter)
	if !ok {
		return nil, fmt.Errorf("binance_futures adapter type assertion failed")
	}
	liquidations, err := bf.FetchLiquidations(ctx, symbol, limit)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(liquidations))
	for _, l := range liquidations {
		result = append(result, map[string]interface{}{
			"symbol":     l.Symbol,
			"side":       l.Side,
			"price":      l.Price,
			"qty":        l.Qty,
			"amount":     l.Amount,
			"time":       l.Time,
			"order_side": l.OrderSide,
		})
	}
	return result, nil
}

// GetCryptoDepth returns order book depth for a crypto pair on a given exchange via CCXT.
func (a *App) GetCryptoDepth(ctx context.Context, exchange, symbol string, limit int) (map[string]interface{}, error) {
	return a.FetchData("ccxt", "orderbook", []string{symbol}, "", "", map[string]string{
		"exchange": exchange,
		"limit":    fmt.Sprintf("%d", limit),
	})
}

// GetDeFiTVL returns top DeFi protocols by TVL via DeFi Llama.
func (a *App) GetDeFiTVL(ctx context.Context) (map[string]interface{}, error) {
	return a.FetchData("crypto_extras", "defi_tvl", nil, "", "", nil)
}

// GetWhaleTransactions returns large crypto transactions via Etherscan.
func (a *App) GetWhaleTransactions(ctx context.Context, address string) (map[string]interface{}, error) {
	symbols := []string{}
	if address != "" {
		symbols = []string{address}
	}
	return a.FetchData("crypto_extras", "whale", symbols, "", "", nil)
}

// GetGasFees returns current Ethereum gas fees via Etherscan Gas Tracker.
func (a *App) GetGasFees(ctx context.Context) (map[string]interface{}, error) {
	return a.FetchData("crypto_extras", "gas_fees", nil, "", "", nil)
}

// GetShortInterest returns short interest data for a US stock symbol via Finnhub.
func (a *App) GetShortInterest(ctx context.Context, symbol string) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	adpt := reg.Get("finnhub")
	if adpt == nil {
		return nil, fmt.Errorf("finnhub adapter not available")
	}
	fh, ok := adpt.(*adapters.FinnhubAdapter)
	if !ok {
		return nil, fmt.Errorf("finnhub adapter type assertion failed")
	}
	data, err := fh.FetchShortInterest(ctx, symbol)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(data))
	for _, d := range data {
		result = append(result, map[string]interface{}{
			"symbol":         d.Symbol,
			"date":           d.Date,
			"short_interest": d.ShortInterest,
			"avg_daily_vol":  d.AvgDailyVolume,
			"days_to_cover":  d.DaysToCover,
			"short_pct":      d.ShortPercent,
		})
	}
	return result, nil
}

// GetEarningsCalendar returns upcoming US earnings events via Finnhub.
func (a *App) GetEarningsCalendar(ctx context.Context, from, to string) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	adpt := reg.Get("finnhub")
	if adpt == nil {
		return nil, fmt.Errorf("finnhub adapter not available")
	}
	fh, ok := adpt.(*adapters.FinnhubAdapter)
	if !ok {
		return nil, fmt.Errorf("finnhub adapter type assertion failed")
	}
	events, err := fh.FetchEarningsCalendar(ctx, from, to)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(events))
	for _, e := range events {
		result = append(result, map[string]interface{}{
			"symbol":           e.Symbol,
			"date":             e.Date,
			"hour":             e.Hour,
			"quarter":          e.Quarter,
			"year":             e.Year,
			"eps_actual":       e.EPSActual,
			"eps_estimate":     e.EPSEstimate,
			"revenue_actual":   e.RevenueActual,
			"revenue_estimate": e.RevenueEstimate,
		})
	}
	return result, nil
}

// GetUSOptionChain returns option chain data for a US stock via Finnhub.
func (a *App) GetUSOptionChain(symbol string) ([]adapters.OptionChainItem, error) {
	ctx := context.Background()
	reg := a.getMarketReg()
	adpt := reg.Get("finnhub")
	if adpt == nil {
		return nil, fmt.Errorf("finnhub adapter not available")
	}
	fh, ok := adpt.(*adapters.FinnhubAdapter)
	if !ok {
		return nil, fmt.Errorf("finnhub adapter type assertion failed")
	}
	return fh.FetchOptionChain(ctx, symbol)
}

// GetSECFilings returns recent SEC filings for a US stock via Finnhub.
func (a *App) GetSECFilings(symbol string) ([]adapters.FinnhubSECFiling, error) {
	ctx := context.Background()
	reg := a.getMarketReg()
	adpt := reg.Get("finnhub")
	if adpt == nil {
		return nil, fmt.Errorf("finnhub adapter not available")
	}
	fh, ok := adpt.(*adapters.FinnhubAdapter)
	if !ok {
		return nil, fmt.Errorf("finnhub adapter type assertion failed")
	}
	return fh.FetchSECFilings(ctx, symbol)
}

// GetCBArbitrageData returns convertible bond arbitrage data from AKShare (集思录).
// Uses Python sidecar via FetchData. Returns JSL convertible bond list with
// premium rates, conversion prices, and forced redemption warnings.
func (a *App) GetCBArbitrageData() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	if a.bridge == nil {
		return nil, fmt.Errorf("Python sidecar not available")
	}

	jslData, err := a.FetchData("akshare", "cb_arbitrage", []string{"all"}, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("cb_arbitrage jsl: %w", err)
	}
	result["bonds"] = jslData

	redeemData, err := a.FetchData("akshare", "cb_redeem", []string{"all"}, "", "", nil)
	if err != nil {
		// redeem data is optional; don't fail the whole request
		result["redeem"] = nil
	} else {
		result["redeem"] = redeemData
	}
	return result, nil
}

// ── Hong Kong Market Data ───────────────────────────────────────────

// GetHKIPOCalendar returns HK IPO subscription/listing data via Python sidecar.
func (a *App) GetHKIPOCalendar(year int) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	subData, err := a.FetchData("akshare", "hk_ipo", []string{"all"}, "", "", map[string]string{"cmd": "get_hk_ipo_subscription"})
	if err != nil {
		return nil, fmt.Errorf("hk ipo subscription: %w", err)
	}
	result["subscription"] = subData

	listData, err := a.FetchData("akshare", "hk_ipo", []string{"all"}, "", "", map[string]string{"cmd": "get_hk_ipo_record"})
	if err != nil {
		result["listing"] = nil
	} else {
		result["listing"] = listData
	}
	return result, nil
}

// GetHKDerivatives returns HK CBBC and warrants data via Python sidecar.
func (a *App) GetHKDerivatives() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	cbbcData, err := a.FetchData("akshare", "hk_cbbc", []string{"all"}, "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("hk cbbc: %w", err)
	}
	result["cbbc"] = cbbcData

	warrantData, err := a.FetchData("akshare", "hk_warrants", []string{"all"}, "", "", nil)
	if err != nil {
		result["warrants"] = nil
	} else {
		result["warrants"] = warrantData
	}
	return result, nil
}

// GetHKTradingCalendar returns HK trading calendar for a given year.
func (a *App) GetHKTradingCalendar(year int) (map[string]interface{}, error) {
	return a.FetchData("akshare", "hk_trade_cal", []string{fmt.Sprintf("%d", year)}, "", "", nil)
}

// HKSettlementInfo holds static HK market settlement rules.
type HKSettlementInfo struct {
	Market          string  `json:"market"`
	SettlementDays  int     `json:"settlement_days"`
	StampDuty       float64 `json:"stamp_duty"`
	ExchangeFee     float64 `json:"exchange_fee"`
	SFCLevy         float64 `json:"sfc_levy"`
	TradingFee      float64 `json:"trading_fee"`
	FRCLevy         float64 `json:"frc_levy"`
	HasPriceLimits  bool    `json:"has_price_limits"`
	LotSizeMin      int     `json:"lot_size_min"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
}

// GetHKSettlementInfo returns static HK market settlement rules.
func (a *App) GetHKSettlementInfo() HKSettlementInfo {
	return HKSettlementInfo{
		Market:         "HK",
		SettlementDays: 2,
		StampDuty:      0.13,
		ExchangeFee:    0.00565,
		SFCLevy:        0.00278,
		TradingFee:     0.005,
		FRCLevy:        0.00015,
		HasPriceLimits: false,
		LotSizeMin:     100,
		Currency:       "HKD",
		Description:    "港股 T+2 交收，无涨跌停限制，每手 100 股",
	}
}

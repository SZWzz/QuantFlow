package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
	"quantflow/internal/portfolio"
)

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

	if len(ticks) > 0 || sinceTimestamp > 0 {
		return ticks, "cache", nil
	}

	adpt := a.getMootdxAdapter()
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	liveTicks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}

	if len(liveTicks) > 0 {
		today := time.Now().Format("2006-01-02")
		if err := a.minuteCache.SaveTicks(symbol, today, liveTicks); err != nil {
			slog.Warn("minute_cache: save failed", "symbol", symbol, "err", err)
		}
	}

	return liveTicks, "mootdx", nil
}

// FetchOHLCV fetches OHLCV bars for a symbol via the market's fallback chain.
func (a *App) FetchOHLCV(ctx context.Context, marketName, symbol, interval string, start, end int64) ([]market.OHLCVBar, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchOHLCVWithFallback(ctx, marketName, symbol, interval, start, end)
}

// getMootdxAdapter retrieves the mootdx adapter from the market registry.
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

// GetMarketSnapshot returns batch quotes for a list of symbols.
func (a *App) GetMarketSnapshot(ctx context.Context, symbols []string) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	result := make([]map[string]interface{}, 0, len(symbols))
	for _, sym := range symbols {
		mkt := market.MarketForSymbol(sym)
		snap, _, err := reg.FetchQuoteWithFallback(ctx, mkt, sym)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"symbol":     sym,
			"price":      snap.Last,
			"change":     snap.Change,
			"change_pct": snap.ChangePct,
			"volume":     snap.Volume,
		})
	}
	return result, nil
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

// GetCorrelationMatrix computes the Pearson correlation matrix for a set of symbols.
func (a *App) GetCorrelationMatrix(ctx context.Context, symbols []string, lookback int) (map[string]map[string]float64, error) {
	reg := a.getMarketReg()
	returns := make(map[string][]float64)
	end := time.Now().Unix()
	start := end - int64(lookback*86400)
	for _, sym := range symbols {
		mkt := market.MarketForSymbol(sym)
		bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, sym, "1d", start, end)
		if err != nil || len(bars) < 2 {
			continue
		}
		rets := make([]float64, 0, len(bars)-1)
		for i := 1; i < len(bars); i++ {
			if bars[i-1].Close > 0 {
				rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
			}
		}
		returns[sym] = rets
	}
	return portfolio.CorrelationMatrix(returns), nil
}

// GetReturnDistribution computes a histogram of daily log returns for a symbol.
func (a *App) GetReturnDistribution(ctx context.Context, symbol string, lookback int, bins int) (map[string]interface{}, error) {
	reg := a.getMarketReg()
	mkt := market.MarketForSymbol(symbol)
	end := time.Now().Unix()
	start := end - int64(lookback*86400)
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", start, end)
	if err != nil || len(bars) < 2 {
		return nil, fmt.Errorf("insufficient data for %s: %w", symbol, err)
	}
	rets := make([]float64, 0, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
	}
	histBins, histCounts := portfolio.ReturnDistribution(rets, bins)
	return map[string]interface{}{
		"symbol": symbol,
		"bins":   histBins,
		"counts": histCounts,
	}, nil
}

// GetVolatilitySurface computes historical volatility across multiple time windows.
func (a *App) GetVolatilitySurface(ctx context.Context, symbol string) ([][]float64, error) {
	reg := a.getMarketReg()
	mkt := market.MarketForSymbol(symbol)
	end := time.Now().Unix()
	start := end - int64(365*86400)
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", start, end)
	if err != nil || len(bars) < 5 {
		return nil, fmt.Errorf("insufficient data for %s: %w", symbol, err)
	}
	rets := make([]float64, 0, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
	}
	return portfolio.VolatilitySurface(rets, []int{5, 10, 20, 30, 60, 90, 120, 252}), nil
}

// SearchSymbols searches A-share stocks by code, name, or pinyin abbreviation.
func (a *App) SearchSymbols(query string) ([]market.StockEntry, error) {
	if a.searchSvc == nil {
		return []market.StockEntry{}, nil
	}
	return a.searchSvc.Search(query, 20), nil
}

// GetCapitalData returns capital/fundamental data for a symbol.
func (a *App) GetCapitalData(symbol string) (map[string]interface{}, error) {
	if a.capitalSvc == nil {
		return nil, fmt.Errorf("capital service not initialized")
	}
	ctx := context.Background()
	result := map[string]interface{}{}
	if data, err := a.capitalSvc.GetMarginTrading(ctx, symbol, 30); err == nil {
		result["margin_trading"] = data
	}
	if data, err := a.capitalSvc.GetBlockTrades(ctx, symbol, 20); err == nil {
		result["block_trades"] = data
	}
	if data, err := a.capitalSvc.GetHolderChanges(ctx, symbol, 10); err == nil {
		result["holder_changes"] = data
	}
	if data, err := a.capitalSvc.GetDividendHistory(ctx, symbol, 20); err == nil {
		result["dividend_history"] = data
	}
	return result, nil
}

// GetFundFlow returns capital flow data for a symbol.
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

// GetDragonTiger returns dragon tiger board data for a symbol.
func (a *App) GetDragonTiger(symbol string, endDate string, lookBack int) ([]adapters.DragonTigerRecord, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	if lookBack <= 0 {
		lookBack = 30
	}
	return a.signalsAdpt.FetchDragonTigerStock(context.Background(), symbol, endDate, lookBack)
}

// GetDailyDragonTiger returns market-wide dragon tiger board for a trading date.
func (a *App) GetDailyDragonTiger(date string, minNetBuy float64) ([]adapters.DragonTigerStock, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	return a.signalsAdpt.FetchDailyDragonTiger(context.Background(), date, minNetBuy)
}

// GetLockupExpiry returns lockup expiry data (解禁) for a symbol.
func (a *App) GetLockupExpiry(symbol string) ([]adapters.LockupExpiry, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	return a.signalsAdpt.FetchLockupExpiry(context.Background(), symbol)
}

// GetIndustryRanks returns industry ranking by change percent.
func (a *App) GetIndustryRanks(topN int) ([]adapters.IndustryRank, error) {
	if a.signalsAdpt == nil {
		return []adapters.IndustryRank{}, nil
	}
	if topN <= 0 {
		topN = 20
	}
	ranks, err := a.signalsAdpt.FetchIndustryRanks(context.Background(), topN)
	if err != nil {
		slog.Warn("GetIndustryRanks failed, returning empty", "error", err)
		return []adapters.IndustryRank{}, nil
	}
	return ranks, nil
}

// GetConceptBlocks returns the concept/industry/sector blocks a stock belongs to.
func (a *App) GetConceptBlocks(symbol string) ([]adapters.ConceptBlock, error) {
	if a.conceptAdpt == nil {
		return nil, fmt.Errorf("concept adapter not initialized")
	}
	return a.conceptAdpt.FetchConceptBlocks(context.Background(), symbol)
}

// GetSatelliteSnapshots returns satellite energy data snapshots for all 5 regions.
func (a *App) GetSatelliteSnapshots() (map[string]interface{}, error) {
	if a.satelliteSvc == nil {
		return nil, fmt.Errorf("satellite service not initialized")
	}
	snapshots, err := a.satelliteSvc.GetRegionSnapshots(context.Background())
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"regions": snapshots, "count": len(snapshots)}, nil
}

// GetSatelliteDetail returns detailed satellite data for a single region.
func (a *App) GetSatelliteDetail(regionID string) (map[string]interface{}, error) {
	if a.satelliteSvc == nil {
		return nil, fmt.Errorf("satellite service not initialized")
	}
	ctx := context.Background()
	snapshot, _, err := a.satelliteSvc.GetRegionDetail(ctx, regionID)
	if err != nil {
		return nil, err
	}
	solarPts, windPts, err := a.satelliteSvc.GetRegionEnergyData(ctx, regionID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"snapshot":    snapshot,
		"solar_data":  solarPts,
		"wind_data":   windPts,
		"solar_chart": solarPts,
		"wind_chart":  windPts,
	}, nil
}

// GetCommodityQuotes returns real-time commodity prices.
func (a *App) GetCommodityQuotes() map[string]interface{} {
	return queryCommodityQuotes(a.marketReg)
}

// NewsItem is a lightweight news article for the frontend news panel.
type NewsItem struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Time   string `json:"time"`
	URL    string `json:"url,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

// GetNews fetches recent news. When symbol is empty, fetches global market news.
func (a *App) GetNews(symbol string, limit int) ([]NewsItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	ctx := context.Background()

	if symbol == "" && a.globalNewsAdpt != nil {
		articles, err := a.globalNewsAdpt.FetchGlobalNews(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("global news: %w", err)
		}
		items := make([]NewsItem, 0, len(articles))
		for _, art := range articles {
			items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL})
		}
		return items, nil
	}

	if a.newsAdpt == nil {
		return nil, fmt.Errorf("news adapter not available")
	}
	articles, err := a.newsAdpt.FetchStockNews(ctx, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("stock news: %w", err)
	}
	items := make([]NewsItem, 0, len(articles))
	for _, art := range articles {
		items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL, Symbol: art.Symbol})
	}
	return items, nil
}

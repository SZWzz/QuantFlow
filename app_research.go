package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
)

// GetSentiment returns sentiment analysis for a symbol.
func (a *App) GetSentiment(symbol string) (*research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db), a.newsAdpt)
	return engine.AnalyzeSentiment(context.Background(), symbol, "", "news", detectLanguage(symbol))
}

// detectLanguage returns "zh" for A-share symbols (starts with 0/3/6 and is 6-digit numeric, or has .SZ/.SH suffix), "en" otherwise.
func detectLanguage(symbol string) string {
	sym := symbol
	// Strip market suffix if present
	if len(sym) > 3 && sym[len(sym)-3] == '.' {
		sym = sym[:len(sym)-3]
	}
	// 6-digit numeric = A-share
	if len(sym) == 6 {
		for _, c := range sym {
			if c < '0' || c > '9' {
				return "en"
			}
		}
		return "zh"
	}
	return "en"
}

// GetSentimentHistory returns historical sentiment for a symbol.
func (a *App) GetSentimentHistory(symbol string, days int) ([]research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db), a.newsAdpt)
	return engine.GetSentimentHistory(context.Background(), symbol, days)
}

// GetStockResearch returns multi-dimensional research data for a symbol.
func (a *App) GetStockResearch(symbol string, tabs []string) (*research.StockResearchResult, error) {
	slog.Info("GetStockResearch called", "symbol", symbol, "tabs", tabs)
	repo := research.NewResearchRepo(a.db)
	finSvc := research.NewFinancialsService(a.sinaFinAdpt, a.getMootdxAdapter())
	peerSvc := research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt)
	estSvc := research.NewAnalystEstimatesService(a.reportAdpt, a.consensusAdpt)
	insSvc := research.NewInsiderTradingService()

	result := &research.StockResearchResult{
		Symbol: symbol,
		Overview: map[string]interface{}{
			"symbol": symbol, "name": symbol,
			"sector": "N/A", "market_cap": "N/A",
		},
	}

	// Try EastMoney stock_info for overview data and market cap
	var emInfo *adapters.EastMoneyStockInfo
	if a.eastmoneyAdpt != nil {
		if info, err := a.eastmoneyAdpt.FetchStockInfo(context.Background(), symbol); err == nil {
			emInfo = info
			result.Overview["name"] = info.Name
			result.Overview["sector"] = info.Industry
			result.Overview["market_cap"] = fmt.Sprintf("%.0f亿", info.MarketCap/1e8)
			result.Overview["total_shares"] = fmt.Sprintf("%.2f亿股", info.TotalShares/1e8)
			result.Overview["float_shares"] = fmt.Sprintf("%.2f亿股", info.FloatShares/1e8)
			result.Overview["list_date"] = info.ListDate
			result.Overview["price"] = info.Price
		} else {
			slog.Warn("eastmoney stock_info failed", "symbol", symbol, "error", err)
		}
	} else {
		slog.Warn("eastmoney adapter not initialized", "symbol", symbol)
	}

	for _, tab := range tabs {
		switch tab {
		case "financials":
			fd, _ := finSvc.GetFinancials(context.Background(), symbol)
			// Sina financials adapter does not provide market cap — fill from EastMoney
			if fd != nil && fd.MarketCap == 0 && emInfo != nil {
				fd.MarketCap = emInfo.MarketCap
			}
			if fd != nil && fd.TotalDebt == 0 {
				// Some Sina responses omit total liabilities — use total assets - total equity as fallback
				if fd.TotalAssets > 0 && fd.TotalEquity > 0 {
					fd.TotalDebt = fd.TotalAssets - fd.TotalEquity
				}
			}
			result.Financials = &research.FinancialsBundle{
				Data:   fd,
				Ratios: finSvc.ComputeRatios(fd),
			}
		case "peers":
			peers, _ := peerSvc.GetPeers(context.Background(), symbol)
			result.Peers = peers
		case "estimates":
			est, _ := estSvc.GetEstimates(context.Background(), symbol)
			result.Estimates = est
		case "insider":
			txns, _ := insSvc.GetInsiderTrades(context.Background(), symbol)
			result.InsiderTxns = txns
		case "sentiment":
			engine := research.NewSentimentEngine(a.bridge, repo, a.newsAdpt)
			s, err := engine.AnalyzeSentiment(context.Background(), symbol, "", "news", detectLanguage(symbol))
			if err != nil {
				slog.Warn("sentiment analysis error", "symbol", symbol, "error", err)
			}
			slog.Info("GetStockResearch sentiment", "symbol", symbol, "lang", detectLanguage(symbol), "has_data", s != nil, "score", s.Score, "label", s.Label)
			result.Sentiment = s
			// Also embed in overview for reliability (Wails serialization fallback)
			result.Overview["sentiment_score"] = s.Score
			result.Overview["sentiment_label"] = s.Label
			result.Overview["sentiment_confidence"] = s.Confidence
		}
	}

	return result, nil
}

// GetCongressTrades returns recent US Congress trading activity.
// Used by the CongressTradingPanel frontend.
func (a *App) GetCongressTrades() ([]research.CongressTrade, error) {
	svc := research.NewCongressTradingService(a.congressAdpt)
	return svc.GetCongressTrades(context.Background())
}

// GetPredictionMarkets returns prediction market events for a category.
// category: "", "economics", "crypto", "politics", "sports", "tech", "all".
func (a *App) GetPredictionMarkets(category string, limit int) (map[string]interface{}, error) {
	if a.predictionMarketSvc == nil {
		return nil, fmt.Errorf("prediction market service not initialized")
	}
	events, err := a.predictionMarketSvc.GetEvents(context.Background(), category, limit)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"events": events, "count": len(events)}, nil
}

// GetPredictionEventDetail returns detail + price history for a prediction event.
func (a *App) GetPredictionEventDetail(eventID string) (map[string]interface{}, error) {
	if a.predictionMarketSvc == nil {
		return nil, fmt.Errorf("prediction market service not initialized")
	}
	ctx := context.Background()
	event, err := a.predictionMarketSvc.GetEventDetail(ctx, eventID)
	if err != nil {
		return nil, err
	}
	prices, _ := a.predictionMarketSvc.GetPriceHistory(ctx, eventID, "1d", 30)
	return map[string]interface{}{"event": event, "prices": prices}, nil
}

// GetPredictionSignals extracts trading signals from prediction market data.
func (a *App) GetPredictionSignals(category string, minProbChange float64) (map[string]interface{}, error) {
	if a.predictionMarketSvc == nil {
		return nil, fmt.Errorf("prediction market service not initialized")
	}
	output, err := a.predictionMarketSvc.ExtractSignals(context.Background(), category, minProbChange)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"events":       output.Events,
		"signal":       output.Signal,
		"category":     output.Category,
		"generated_at": output.GeneratedAt.Format(time.RFC3339),
	}, nil
}

// GetGeopoliticsRisks returns risk assessments for all 10 geopolitical topics.
func (a *App) GetGeopoliticsRisks() (map[string]interface{}, error) {
	if a.geopoliticsSvc == nil {
		return nil, fmt.Errorf("geopolitics service not initialized")
	}
	risks, err := a.geopoliticsSvc.GetTopicRisks(context.Background())
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"risks": risks, "count": len(risks)}, nil
}

// GetEconomicIndicators returns macro signals for all 15 FRED indicators.
func (a *App) GetEconomicIndicators() (map[string]interface{}, error) {
	if a.govDataSvc == nil {
		return nil, fmt.Errorf("govdata service not initialized")
	}
	signals, err := a.govDataSvc.GetAllSignals(context.Background())
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"signals": signals, "count": len(signals)}, nil
}

// GetIndicatorData returns time series data for a specific FRED indicator.
func (a *App) GetIndicatorData(seriesID string, limit int) (map[string]interface{}, error) {
	if a.govDataSvc == nil {
		return nil, fmt.Errorf("govdata service not initialized")
	}
	points, err := a.govDataSvc.GetIndicator(context.Background(), seriesID, limit)
	if err != nil {
		return nil, err
	}
	meta := adapters.FREDIndicators[seriesID]
	return map[string]interface{}{
		"series_id": seriesID,
		"name":      meta.Name,
		"name_cn":   meta.NameCN,
		"unit":      meta.Unit,
		"category":  meta.Category,
		"data":      points,
		"count":     len(points),
	}, nil
}

// GetGeopoliticsDetail returns volume + tone time series for a single topic.
func (a *App) GetGeopoliticsDetail(topicID, timespan string) (map[string]interface{}, error) {
	if a.geopoliticsSvc == nil {
		return nil, fmt.Errorf("geopolitics service not initialized")
	}
	return a.geopoliticsSvc.GetTopicDetail(context.Background(), topicID, timespan)
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

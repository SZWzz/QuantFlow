package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
	"quantflow/internal/trading"
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

func detectMarketForSymbol(symbol string) string {
	if strings.HasSuffix(symbol, ".HK") {
		return "HK"
	}
	if strings.HasSuffix(symbol, ".SZ") || strings.HasSuffix(symbol, ".SH") {
		return "CN"
	}
	if len(symbol) == 6 && (symbol[0] == '0' || symbol[0] == '3' || symbol[0] == '6') {
		return "CN"
	}
	if len(symbol) == 5 && symbol[0] == '0' {
		return "HK"
	}
	if len(symbol) <= 4 && allUpper(symbol) {
		return "US"
	}
	return "CN"
}

func allUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}

func detectST(symbol string) bool {
	return strings.Contains(strings.ToUpper(symbol), "ST")
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
	peerSvc := research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt, a.eastmoneyAdpt, a.marketReg)
	estSvc := research.NewAnalystEstimatesService(a.reportAdpt, a.consensusAdpt)
	insSvc := research.NewInsiderTradingService(a.bridge)

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
			// Fallback: try quote pipeline (Tencent/Sina) for name, price, and market cap
			if a.marketReg != nil {
				if quote, _, qErr := a.marketReg.FetchQuoteWithFallback(context.Background(), "CN", symbol); qErr == nil && quote != nil {
					result.Overview["name"] = quote.Name
					result.Overview["price"] = quote.Last
					if quote.MarketCap > 0 {
						result.Overview["market_cap"] = fmt.Sprintf("%.0f亿", quote.MarketCap/1e8)
					}
					slog.Info("stock_info fallback via quote pipeline succeeded", "symbol", symbol)
				}
			}
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

// GetChanlun returns chanlun (缠论) analysis for a symbol.
func (a *App) GetChanlun(symbol string) (map[string]any, error) {
	if a.bridge == nil {
		return nil, fmt.Errorf("python bridge not available")
	}
	return a.bridge.AnalyzeChanlun(symbol)
}

// ComputeIndicator calculates a technical indicator for a symbol.
func (a *App) ComputeIndicator(symbol string, indicatorName string, params map[string]any) (map[string]any, error) {
	if a.bridge == nil {
		return nil, fmt.Errorf("python bridge not available")
	}
	return a.bridge.ComputeIndicator(symbol, indicatorName, params)
}

// ScanStocks scans all A-share stocks for a given strategy and returns ranked results.
func (a *App) ScanStocks(strategyName string) (map[string]any, error) {
	if a.bridge == nil {
		return map[string]any{
			"strategy": strategyName, "results": []map[string]any{}, "scanned": 0,
		}, nil
	}
	return a.bridge.ScanStocks(strategyName, 50)
}

// ── MAC Protocol IPC Methods ──────────────────────────────────────────────────

// GetBlockRank returns block trade ranking from MAC protocol.
func (a *App) GetBlockRank(market int, sortField int, count int) ([]adapters.BlockRank, error) {
	if a.macAdpt == nil {
		return nil, fmt.Errorf("MAC adapter not initialized")
	}
	return a.macAdpt.GetBlockRank(market, sortField, count)
}

// GetMACCapitalFlow returns real-time capital flow for a symbol via MAC protocol.
func (a *App) GetMACCapitalFlow(symbol string) (*adapters.CapitalFlow, error) {
	if a.macAdpt == nil {
		return nil, fmt.Errorf("MAC adapter not initialized")
	}
	return a.macAdpt.GetCapitalFlow(symbol)
}

// GetAuction returns pre-market call auction data for a symbol.
func (a *App) GetAuction(symbol string) ([]adapters.AuctionItem, error) {
	if a.macAdpt == nil {
		return nil, fmt.Errorf("MAC adapter not initialized")
	}
	return a.macAdpt.GetAuction(symbol)
}

// GetAbnormalStocks returns stocks with abnormal price/volume behavior.
func (a *App) GetAbnormalStocks(market int) ([]adapters.AbnormalStock, error) {
	if a.macAdpt == nil {
		return nil, fmt.Errorf("MAC adapter not initialized")
	}
	return a.macAdpt.GetAbnormalStocks(market)
}

// GetIPOCalendar returns upcoming and recent IPO listing calendar.
// startDate/endDate format: "YYYY-MM-DD". Uses EastMoney datacenter API.
func (a *App) GetIPOCalendar(startDate, endDate string) ([]adapters.IPORecord, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	return a.signalsAdpt.FetchIPOCalendar(context.Background(), startDate, endDate, 200)
}

// ── Financial Deep Analysis ──────────────────────────────────────────

// GetFinancialAnalysis returns deep financial report analysis for a symbol.
func (a *App) GetFinancialAnalysis(symbol string) (map[string]interface{}, error) {
	finJSON, err := a.fetchFinancialJSON(symbol)
	if err != nil {
		return nil, err
	}
	return a.FetchData("analyzer", "report_analysis", []string{finJSON}, "", "", nil)
}

// GetValuation returns DCF valuation (3 scenarios) for a symbol.
func (a *App) GetValuation(symbol string) (map[string]interface{}, error) {
	finJSON, err := a.fetchFinancialJSON(symbol)
	if err != nil {
		return nil, err
	}
	quote := a.fetchQuoteJSON(symbol)
	return a.FetchData("analyzer", "valuation", []string{finJSON}, "", "",
		map[string]string{"quote": quote})
}

// GetAuditFindings returns audit risk detection results for a symbol.
func (a *App) GetAuditFindings(symbol string) (map[string]interface{}, error) {
	finJSON, err := a.fetchFinancialJSON(symbol)
	if err != nil {
		return nil, err
	}
	return a.FetchData("analyzer", "audit", []string{finJSON}, "", "", nil)
}

// GetForecast returns financial forecast (3 scenarios) for a symbol.
func (a *App) GetForecast(symbol string) (map[string]interface{}, error) {
	finJSON, err := a.fetchFinancialJSON(symbol)
	if err != nil {
		return nil, err
	}
	return a.FetchData("analyzer", "forecast", []string{finJSON}, "", "", nil)
}

// GetFinancialStatements returns raw financial statements (利润表/资产负债表/现金流量表) for a symbol.
func (a *App) GetFinancialStatements(symbol string) (map[string]interface{}, error) {
	if a.sinaFinAdpt == nil {
		return nil, fmt.Errorf("sina financials adapter not available")
	}
	ctx := context.Background()
	income, _ := a.sinaFinAdpt.FetchIncomeStatement(ctx, symbol, 12)
	balance, _ := a.sinaFinAdpt.FetchBalanceSheet(ctx, symbol, 12)
	cashflow, _ := a.sinaFinAdpt.FetchCashFlow(ctx, symbol, 12)
	if income == nil && balance == nil {
		return nil, fmt.Errorf("no financial data for %s", symbol)
	}
	return map[string]interface{}{
		"income":   formatFinPeriodsRaw(income),
		"balance":  formatFinPeriodsRaw(balance),
		"cashflow": formatFinPeriodsRaw(cashflow),
	}, nil
}

// fetchFinancialJSON fetches financial statements via Sina adapter and returns JSON
// in the flat mapped format expected by the Python analyzer (formatFinPeriods).
func (a *App) fetchFinancialJSON(symbol string) (string, error) {
	if a.sinaFinAdpt == nil {
		return "", fmt.Errorf("sina financials adapter not available")
	}
	ctx := context.Background()
	income, _ := a.sinaFinAdpt.FetchIncomeStatement(ctx, symbol, 12)
	balance, _ := a.sinaFinAdpt.FetchBalanceSheet(ctx, symbol, 12)
	cashflow, _ := a.sinaFinAdpt.FetchCashFlow(ctx, symbol, 12)
	if income == nil && balance == nil {
		return "", fmt.Errorf("no financial data for %s", symbol)
	}
	payload := map[string]interface{}{
		"income":   formatFinPeriods(income),
		"balance":  formatFinPeriods(balance),
		"cashflow": formatFinPeriods(cashflow),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal financials: %w", err)
	}
	return string(b), nil
}

// fetchDelistingFinancialJSON fetches financial statements and returns JSON in the
// raw adapter format (item_title/item_value/item_tongbi) expected by trading.ExtractFinancialMetrics.
func (a *App) fetchDelistingFinancialJSON(symbol string) (string, error) {
	if a.sinaFinAdpt == nil {
		return "", fmt.Errorf("sina financials adapter not available")
	}
	ctx := context.Background()
	income, _ := a.sinaFinAdpt.FetchIncomeStatement(ctx, symbol, 12)
	balance, _ := a.sinaFinAdpt.FetchBalanceSheet(ctx, symbol, 12)
	cashflow, _ := a.sinaFinAdpt.FetchCashFlow(ctx, symbol, 12)
	if income == nil && balance == nil {
		return "", fmt.Errorf("no financial data for %s", symbol)
	}
	payload := map[string]interface{}{
		"income":   income,
		"balance":  balance,
		"cashflow": cashflow,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal financials: %w", err)
	}
	return string(b), nil
}

// sinaToAnalyzer maps Sina Chinese financial item names to the standardized
// short names that the Python analyzer expects.
var sinaToAnalyzer = map[string]string{
	"营业总收入":                  "营业总收入",
	"营业收入":                   "营业总收入",
	"营业成本":                   "营业成本",
	"营业总成本":                  "营业总成本",
	"营业利润":                   "营业利润",
	"利润总额":                   "利润总额",
	"净利润":                     "净利润",
	"归属于母公司所有者的净利润":        "归母净利润",
	"归属于母公司股东的净利润":         "归母净利润",
	"扣除非经常性损益后的净利润":       "扣非净利润",
	"扣非净利润":                  "扣非净利润",
	"资产总计":                    "总资产",
	"流动资产合计":                 "流动资产",
	"货币资金":                    "货币资金",
	"应收账款":                    "应收账款",
	"存货":                       "存货",
	"固定资产":                    "固定资产",
	"商誉":                       "商誉",
	"负债合计":                    "总负债",
	"流动负债合计":                 "流动负债",
	"短期借款":                    "短期借款",
	"长期借款":                    "长期借款",
	"股东权益合计":                 "股东权益",
	"归属于母公司股东权益合计":         "股东权益",
	"未分配利润":                  "未分配利润",
	"经营活动产生的现金流量净额":       "经营现金流净额",
	"投资活动产生的现金流量净额":       "投资现金流净额",
	"筹资活动产生的现金流量净额":       "筹资现金流净额",
	"购建固定资产、无形资产和其他长期资产支付的现金": "资本支出",
	"购建固定资产、无形资产和其他长期资产所支付的现金": "资本支出",
	"企业自由现金流量":                "自由现金流",
	"销售费用":                    "销售费用",
	"管理费用":                    "管理费用",
	"研发费用":                    "研发费用",
	"财务费用":                    "财务费用",
}

// formatFinPeriods converts FinancialStatementPeriod slice to a map-slice format
// that the Python analyzer expects, with standardized field names.
func formatFinPeriods(periods []adapters.FinancialStatementPeriod) []map[string]interface{} {
	if periods == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(periods))
	for _, p := range periods {
		row := map[string]interface{}{"报告期": p.Period}
		for _, item := range p.Items {
			name := item.Title
			if mapped, ok := sinaToAnalyzer[name]; ok {
				name = mapped
			}
			row[name] = item.Value
		}
		result = append(result, row)
	}
	return result
}

// formatFinPeriodsRaw converts FinancialStatementPeriod slice to a map-slice format
// that preserves the original Chinese item titles (no mapping to analyzer names).
func formatFinPeriodsRaw(periods []adapters.FinancialStatementPeriod) []map[string]interface{} {
	if periods == nil {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(periods))
	for _, p := range periods {
		row := map[string]interface{}{"report_date": p.Period}
		for _, item := range p.Items {
			row[item.Title] = item.Value
		}
		result = append(result, row)
	}
	return result
}

// fetchQuoteJSON fetches current quote + shares outstanding and returns JSON string.
func (a *App) fetchQuoteJSON(symbol string) string {
	ctx := context.Background()
	quote, _, err := a.GetQuote(ctx, "CN", symbol)
	if err != nil || quote == nil {
		return "{}"
	}
	payload := map[string]interface{}{
		"price":      quote.Last,
		"change_pct": quote.ChangePct,
		"volume":     quote.Volume,
	}
	// Try to get total shares from EastMoney
	if a.eastmoneyAdpt != nil {
		if info, err := a.eastmoneyAdpt.FetchStockInfo(ctx, symbol); err == nil && info != nil {
			payload["total_shares"] = info.TotalShares
			payload["market_cap"] = info.MarketCap
		}
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

// GetExDividendCalendar returns market-wide ex-dividend records by date range.
// startDate/endDate format: "YYYY-MM-DD". Uses EastMoney datacenter API.
func (a *App) GetExDividendCalendar(startDate, endDate string) ([]adapters.ExDividendCalendarRecord, error) {
	if a.capitalAdpt == nil {
		return nil, fmt.Errorf("capital adapter not initialized")
	}
	return a.capitalAdpt.FetchExDividendCalendar(context.Background(), startDate, endDate)
}

// GetDelistingRisk returns delisting risk assessment for a symbol.
func (a *App) GetDelistingRisk(symbol string) (*trading.DelistingRiskResult, error) {
	market := detectMarketForSymbol(symbol)
	isST := detectST(symbol)

	finJSON, err := a.fetchDelistingFinancialJSON(symbol)
	if err != nil {
		slog.Warn("delisting risk: financial data unavailable", "symbol", symbol, "err", err)
		return a.computeDelistingRisk(symbol, market, isST, nil)
	}

	metrics, err := trading.ExtractFinancialMetrics(finJSON)
	if err != nil {
		slog.Warn("delisting risk: failed to parse financial data", "symbol", symbol, "err", err)
		return a.computeDelistingRisk(symbol, market, isST, nil)
	}

	return a.computeDelistingRisk(symbol, market, isST, metrics)
}

func (a *App) computeDelistingRisk(symbol, market string, isST bool, metrics *trading.FinancialMetrics) (*trading.DelistingRiskResult, error) {
	ctx := context.Background()
	price, marketCap, volume := 0.0, 0.0, 0.0
	totalShares := 0.0
	board := trading.DetectBoard(symbol)

	if a.eastmoneyAdpt != nil {
		info, err := a.eastmoneyAdpt.FetchStockInfo(ctx, symbol)
		if err == nil && info != nil {
			marketCap = info.MarketCap
			totalShares = info.TotalShares
		}
	}
	quote, _, err := a.GetQuote(ctx, market, symbol)
	if err == nil && quote != nil {
		price = quote.Last
		volume = quote.Volume
		if quote.MarketCap > 0 {
			marketCap = quote.MarketCap
		}
	}

	var categories []trading.DelistingCategory
	switch market {
	case "CN":
		categories = trading.AssessCN(metrics, board, price, marketCap, volume, totalShares)
	case "HK":
		categories = trading.AssessHK(price, marketCap, volume, totalShares)
	case "US":
		categories = trading.AssessUS(price, marketCap)
	default:
		categories = trading.AssessCN(metrics, board, price, marketCap, volume, totalShares)
	}

	overallRisk := trading.AssessOverall(categories)
	summary := trading.GenerateSummary(categories, overallRisk)

	return &trading.DelistingRiskResult{
		Market:      market,
		Board:       board,
		IsST:        isST,
		OverallRisk: overallRisk,
		Categories:  categories,
		Summary:     summary,
	}, nil
}

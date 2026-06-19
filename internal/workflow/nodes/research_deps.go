package nodes

import (
	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
)

// Package-level dependencies for research nodes.
// Set before workflow execution via app.go startup.

var sentimentEngine *research.SentimentEngine
var newsAdapter adapters.NewsAdapter
var globalNewsAdapter adapters.GlobalNewsAdapter
var financialsService *research.FinancialsService
var peerComparisonService *research.PeerComparisonService
var analystEstimatesService *research.AnalystEstimatesService
var insiderTradingService *research.InsiderTradingService
var congressTradingService *research.CongressTradingService
var capitalService *research.CapitalService
var fundFlowService *research.FundFlowService
var northboundService *research.NorthboundService
var announcementService *research.AnnouncementService
var predictionMarketService *research.PredictionMarketService
var geopoliticsService *research.GeopoliticsService
var govDataService *research.GovDataService
var satelliteService *research.SatelliteService

// SetSentimentEngine injects the sentiment engine for use by research nodes.
func SetSentimentEngine(e *research.SentimentEngine) {
	sentimentEngine = e
}

// SetFinancialsService injects the financials service.
func SetFinancialsService(s *research.FinancialsService) {
	financialsService = s
}

// SetPeerComparisonService injects the peer comparison service.
func SetPeerComparisonService(s *research.PeerComparisonService) {
	peerComparisonService = s
}

// SetAnalystEstimatesService injects the analyst estimates service.
func SetAnalystEstimatesService(s *research.AnalystEstimatesService) {
	analystEstimatesService = s
}

// SetInsiderTradingService injects the insider trading service.
func SetInsiderTradingService(s *research.InsiderTradingService) {
	insiderTradingService = s
}

// SetCongressTradingService injects the congress trading service.
func SetCongressTradingService(s *research.CongressTradingService) {
	congressTradingService = s
}

// SetCapitalService injects the capital data service.
func SetCapitalService(s *research.CapitalService) {
	capitalService = s
}

// SetFundFlowService injects the fund flow service.
func SetFundFlowService(s *research.FundFlowService) {
	fundFlowService = s
}

// SetNorthboundService injects the northbound flow service.
func SetNorthboundService(s *research.NorthboundService) {
	northboundService = s
}

// SetAnnouncementService injects the announcement service.
func SetAnnouncementService(s *research.AnnouncementService) {
	announcementService = s
}

// SetNewsAdapter injects the news adapter for use by NewsFetcherNode and
// the SentimentEngine news-fetching fallback.
func SetNewsAdapter(a adapters.NewsAdapter) {
	newsAdapter = a
}

// SetGlobalNewsAdapter injects the global news adapter for market-wide news.
func SetGlobalNewsAdapter(a adapters.GlobalNewsAdapter) {
	globalNewsAdapter = a
}

// SetPredictionMarketService injects the prediction market service.
func SetPredictionMarketService(s *research.PredictionMarketService) {
	predictionMarketService = s
}

// SetGeopoliticsService injects the geopolitics service.
func SetGeopoliticsService(s *research.GeopoliticsService) {
	geopoliticsService = s
}

// SetGovDataService injects the govdata service.
func SetGovDataService(s *research.GovDataService) {
	govDataService = s
}

// SetSatelliteService injects the satellite service.
func SetSatelliteService(s *research.SatelliteService) {
	satelliteService = s
}

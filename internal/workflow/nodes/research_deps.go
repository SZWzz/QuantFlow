package nodes

import "quantflow/internal/research"

// Package-level dependencies for research nodes.
// Set before workflow execution via app.go startup.

var sentimentEngine *research.SentimentEngine
var financialsService *research.FinancialsService
var peerComparisonService *research.PeerComparisonService
var analystEstimatesService *research.AnalystEstimatesService
var insiderTradingService *research.InsiderTradingService
var congressTradingService *research.CongressTradingService

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

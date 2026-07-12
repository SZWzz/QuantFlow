package workflow

import (
	"context"

	"quantflow/internal/ai"
	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
	"quantflow/internal/research"
	"quantflow/internal/trading"
)

// OMSService exposes the subset of trading.OMS methods used by workflow nodes.
type OMSService interface {
	HasBroker() bool
	PlaceOrder(symbol string, side trading.OrderSide, orderType trading.OrderType, brokerName string, qty, price float64) (*trading.Order, error)
	PlaceOrderLive(ctx context.Context, symbol string, side trading.OrderSide, orderType trading.OrderType, qty, price, stopPrice float64) (*trading.Order, error)
	CancelOrder(orderID string) error
	GetOrders() []*trading.Order
	GetTrades() []*trading.Trade
	GetPosition(symbol string) *trading.Position
	GetAllPositions() []*trading.Position
}

// MarketRegService exposes the subset of market.AdapterRegistry methods used by workflow nodes.
type MarketRegService interface {
	FetchOHLCVWithFallback(ctx context.Context, mkt, symbol, interval, fqfactor string, start, end int64) ([]market.OHLCVBar, string, error)
}

// ProfileMgrService exposes the subset of ai.ProfileManager methods used by workflow nodes.
type ProfileMgrService interface {
	Get(name string) (*ai.AgentProfile, error)
}

// NewsAdapterService exposes the subset of adapters.NewsAdapter methods used by workflow nodes.
type NewsAdapterService interface {
	FetchStockNews(ctx context.Context, symbol string, limit int) ([]adapters.NewsArticle, error)
}

// SentimentEngineService exposes the subset of research.SentimentEngine methods used by workflow nodes.
type SentimentEngineService interface {
	AnalyzeSentiment(ctx context.Context, symbol, text, textType, language string) (*research.SentimentOutput, error)
}

// FinancialsServiceInterface exposes the subset of research.FinancialsService methods used by nodes.
type FinancialsServiceInterface interface {
	GetFinancials(ctx context.Context, symbol string) (*research.FinancialData, error)
	ComputeRatios(data *research.FinancialData) *research.FinancialRatios
}

// PeerComparisonServiceInterface exposes the subset of research.PeerComparisonService methods.
type PeerComparisonServiceInterface interface {
	GetPeers(ctx context.Context, symbol string) ([]research.PeerComparisonData, error)
}

// AnalystEstimatesServiceInterface exposes the subset of research.AnalystEstimatesService methods.
type AnalystEstimatesServiceInterface interface {
	GetEstimates(ctx context.Context, symbol string) ([]research.AnalystEstimate, error)
}

// InsiderTradingServiceInterface exposes the subset of research.InsiderTradingService methods.
type InsiderTradingServiceInterface interface {
	GetInsiderTrades(ctx context.Context, symbol string) ([]research.InsiderTransaction, error)
}

// PredictionMarketServiceInterface exposes the subset of research.PredictionMarketService methods.
type PredictionMarketServiceInterface interface {
	ExtractSignals(ctx context.Context, category string, minProbChange float64) (*research.SignalOutput, error)
	GetEvents(ctx context.Context, category string, limit int) ([]adapters.PredictionEvent, error)
}

// GeopoliticsServiceInterface exposes the subset of research.GeopoliticsService methods.
type GeopoliticsServiceInterface interface {
	ExtractRiskSignals(ctx context.Context, minVolChange float64) ([]research.TopicRisk, error)
}

// GovDataServiceInterface exposes the subset of research.GovDataService methods.
type GovDataServiceInterface interface {
	GetAllSignals(ctx context.Context) ([]research.MacroSignal, error)
}

// SatelliteServiceInterface exposes the subset of research.SatelliteService methods.
type SatelliteServiceInterface interface {
	ExtractSignals(ctx context.Context) ([]research.SatelliteSignal, error)
	GetRegionSnapshots(ctx context.Context) ([]adapters.RegionSnapshot, error)
}

// ModelRegistryService exposes the subset of ml.ModelRegistry methods used by workflow nodes.
// Uses any to avoid import cycle (ml → storage → workflow → ml in tests).
type ModelRegistryService interface {
	Create(ctx context.Context, model any) error
}

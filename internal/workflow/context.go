package workflow

import "context"

// NodeContext holds all shared service dependencies available to workflow nodes.
// Services are stored as interface{} to avoid import cycles between the workflow
// package and packages like trading, python, research, etc.
// Nodes that need these services should type-assert the interface{} fields.
type NodeContext struct {
	// Trading
	OMS interface{} // *trading.OMS

	// Python bridge for ML/AI nodes
	Bridge interface{} // *python.PythonBridge

	// AI agent dependencies
	CapRegistry interface{} // *ai.CapabilityRegistry
	Emitter     interface{} // *ai.EventEmitter
	ProfileMgr  interface{} // *ai.ProfileManager

	// ML model registry
	ModelRegistry interface{} // *ml.ModelRegistry

	// Market adapters
	MarketReg         interface{} // *market.AdapterRegistry — real OHLCV/quote fetching
	NewsAdapter       interface{} // adapters.NewsAdapter
	GlobalNewsAdapter interface{} // adapters.GlobalNewsAdapter

	// Research services
	SentimentEngine         interface{} // *research.SentimentEngine
	FinancialsService       interface{} // *research.FinancialsService
	PeerComparisonService   interface{} // *research.PeerComparisonService
	AnalystEstimatesService interface{} // *research.AnalystEstimatesService
	InsiderTradingService   interface{} // *research.InsiderTradingService
	CongressTradingService  interface{} // *research.CongressTradingService
	CapitalService          interface{} // *research.CapitalService
	FundFlowService         interface{} // *research.FundFlowService
	NorthboundService       interface{} // *research.NorthboundService
	AnnouncementService     interface{} // *research.AnnouncementService
	PredictionMarketService interface{} // *research.PredictionMarketService
	GeopoliticsService      interface{} // *research.GeopoliticsService
	GovDataService          interface{} // *research.GovDataService
	SatelliteService        interface{} // *research.SatelliteService

	// Backtest window — set by ExecuteBacktest before each window
	BacktestStart string
	BacktestEnd   string

	// RunID is the current workflow execution run identifier, set by Engine.Execute
	// before executing nodes. Nodes can include this in their outputs for traceability.
	RunID string

	// SubWorkflowRunner executes a child workflow by ID with given inputs.
	// Set by app.go to wire the engine's Execute into sub_workflow nodes.
	SubWorkflowRunner func(ctx context.Context, workflowID string, inputs map[string]any) (map[string]any, error)
}

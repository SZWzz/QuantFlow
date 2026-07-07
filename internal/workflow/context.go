package workflow

import "context"

// NodeContext holds all shared service dependencies available to workflow nodes.
type NodeContext struct {
	// Trading
	OMS OMSService

	// Python bridge for ML/AI nodes (passed to constructors, stays as any)
	Bridge interface{}

	// AI agent dependencies
	CapRegistry interface{} // *ai.CapabilityRegistry (passed to constructors)
	Emitter     interface{} // *ai.EventEmitter (passed to constructors)
	ProfileMgr  ProfileMgrService

	// ML model registry
	ModelRegistry ModelRegistryService

	// Market adapters
	MarketReg         MarketRegService
	NewsAdapter       NewsAdapterService
	GlobalNewsAdapter interface{} // adapters.GlobalNewsAdapter (unused by nodes)

	// Research services
	SentimentEngine         SentimentEngineService
	FinancialsService       FinancialsServiceInterface
	PeerComparisonService   PeerComparisonServiceInterface
	AnalystEstimatesService AnalystEstimatesServiceInterface
	InsiderTradingService   InsiderTradingServiceInterface
	CongressTradingService  interface{}
	CapitalService          interface{}
	FundFlowService         interface{}
	NorthboundService       interface{}
	AnnouncementService     interface{}
	PredictionMarketService PredictionMarketServiceInterface
	GeopoliticsService      GeopoliticsServiceInterface
	GovDataService          GovDataServiceInterface
	SatelliteService        SatelliteServiceInterface

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

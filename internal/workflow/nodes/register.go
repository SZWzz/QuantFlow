// Package nodes provides built-in workflow node implementations.
package nodes

import "quantflow/internal/workflow"

// RegisterAll registers all built-in node types with the given registry.
func RegisterAll(r *workflow.NodeRegistry) {
	r.RegisterWithCategory("data_loader", NewDataLoaderNode, "data")
	r.RegisterWithCategory("sma", NewSMANode, "indicator")
	r.RegisterWithCategory("cross_signal", NewCrossSignalNode, "signal")
	r.RegisterWithCategory("log_output", NewLogOutputNode, "output")
	r.RegisterWithCategory("loop", NewLoopNode, "control")
	r.RegisterWithCategory("factor", NewFactorNode, "alpha")
	r.RegisterWithCategory("pct_change", NewPctChangeNode, "alpha")
	r.RegisterWithCategory("delta", NewDeltaNode, "alpha")
	r.RegisterWithCategory("std_dev", NewStdDevNode, "alpha")
	r.RegisterWithCategory("rank", NewRankNode, "alpha")
	r.RegisterWithCategory("scale", NewScaleNode, "alpha")
	r.RegisterWithCategory("cross_over", NewCrossOverNode, "alpha")
	r.RegisterWithCategory("compare", NewCompareNode, "alpha")
	r.RegisterWithCategory("bool_combine", NewBoolCombineNode, "alpha")
	r.RegisterWithCategory("rolling_maxmin", NewRollingMaxMinNode, "alpha")
	r.RegisterWithCategory("rolling_zscore", NewRollingZScoreNode, "alpha")
	r.RegisterWithCategory("arithmetic", NewArithmeticNode, "alpha")
	r.RegisterWithCategory("if_else", NewIfElseNode, "alpha")
	r.RegisterWithCategory("strategy", NewStrategyNode, "strategy")
	r.RegisterWithCategory("backtest", NewBacktestNode, "backtest")
	r.RegisterWithCategory("agent", NewAgentNode, "ai")

	// Phase 5: Trading
	r.RegisterWithCategory("place_order", NewPlaceOrderNode, "trading")
	r.RegisterWithCategory("cancel_order", NewCancelOrderNode, "trading")
	r.RegisterWithCategory("position_query", NewPositionQueryNode, "trading")
	r.RegisterWithCategory("order_query", NewOrderQueryNode, "trading")

	// Phase 5: Notify
	r.RegisterWithCategory("notify", NewNotifyNode, "notify")
	r.RegisterWithCategory("alert", NewAlertNode, "notify")

	// Phase 5: Schedule
	r.RegisterWithCategory("schedule", NewScheduleNode, "schedule")
	r.RegisterWithCategory("wait", NewWaitNode, "schedule")

	// Phase 5: Portfolio/Risk
	r.RegisterWithCategory("portfolio_summary", NewPortfolioSummaryNode, "portfolio")
	r.RegisterWithCategory("risk_metrics", NewRiskMetricsNode, "risk")
	r.RegisterWithCategory("allocation", NewAllocationNode, "portfolio")

	// Phase 8: Node Expansion
	r.RegisterWithCategory("macd", NewMACDNode, "indicator")
	r.RegisterWithCategory("rsi", NewRSINode, "indicator")
	r.RegisterWithCategory("bollinger", NewBollingerNode, "indicator")
	r.RegisterWithCategory("ema", NewEMANode, "indicator")
	r.RegisterWithCategory("merge", NewMergeNode, "data")
	r.RegisterWithCategory("filter", NewFilterNode, "data")
	r.RegisterWithCategory("resample", NewResampleNode, "data")
	r.RegisterWithCategory("threshold_signal", NewThresholdSignalNode, "signal")
	r.RegisterWithCategory("signal_combine", NewSignalCombineNode, "signal")
	r.RegisterWithCategory("rank_select", NewRankSelectNode, "signal")
	r.RegisterWithCategory("hold_signal", NewHoldSignalNode, "signal")
	r.RegisterWithCategory("rebalance", NewRebalanceNode, "signal")
	r.RegisterWithCategory("entry_signal", NewEntrySignalNode, "signal")
	r.RegisterWithCategory("exit_signal", NewExitSignalNode, "signal")
	r.RegisterWithCategory("stop_loss", NewStopLossNode, "risk")
	r.RegisterWithCategory("position_sizer", NewPositionSizerNode, "risk")
	r.RegisterWithCategory("http_request", NewHTTPRequestNode, "utility")
	r.RegisterWithCategory("math_op", NewMathOpNode, "utility")
	r.RegisterWithCategory("json_parse", NewJSONParseNode, "utility")

	// Phase 9 (continued): Control/Output
	r.RegisterWithCategory("if_condition", NewIfConditionNode, "control")
	r.RegisterWithCategory("sub_workflow", NewSubWorkflowNode, "control")
	r.RegisterWithCategory("chart_data", NewChartDataNode, "output")

	// Phase 10: ML
	r.RegisterWithCategory("feature_engineer", NewFeatureEngineerNode, "ml")
	r.RegisterWithCategory("train_model", NewTrainModelNode, "ml")
	r.RegisterWithCategory("predict", NewPredictNode, "ml")
	r.RegisterWithCategory("evaluate_model", NewEvaluateModelNode, "ml")

	// Phase 10.2: Alpha Mining
	r.RegisterWithCategory("alpha_mining", NewAlphaMiningNode, "ml")

	// Phase 10.3: RL Trading
	r.RegisterWithCategory("rl_env", NewRLEnvNode, "ml")
	r.RegisterWithCategory("rl_train", NewRLTrainNode, "ml")
	r.RegisterWithCategory("rl_predict", NewRLPredictNode, "ml")

	// Phase 10.4: Risk Modeling
	r.RegisterWithCategory("risk_model", NewRiskModelNode, "risk")

	// Phase 13: Research
	r.RegisterWithCategory("sentiment", NewSentimentNode, "research")
	r.RegisterWithCategory("news_fetcher", NewNewsFetcherNode, "research")

	// Phase 14: Research Nodes
	r.RegisterWithCategory("stock_research", NewStockResearchNode, "research")
	r.RegisterWithCategory("financials", NewFinancialsNode, "research")
	r.RegisterWithCategory("peer_compare", NewPeerCompareNode, "research")
	r.RegisterWithCategory("analyst_estimates", NewAnalystEstimatesNode, "research")
	r.RegisterWithCategory("insider_trades", NewInsiderTradesNode, "research")

	// Phase 15: Alternative Data
	r.RegisterWithCategory("prediction_market", NewPredictionMarketNode, "alternative_data")
	r.RegisterWithCategory("geopolitics", NewGeopoliticsNode, "alternative_data")

	// GovData: FRED economic indicators + SEC EDGAR filings
	r.RegisterWithCategory("gov_data", NewGovDataNode, "alternative_data")
}

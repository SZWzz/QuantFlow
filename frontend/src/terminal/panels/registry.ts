import { defineAsyncComponent, type Component } from 'vue'

const panelRegistry = new Map<string, Component>()

// Register built-in panels
function register(id: string, loader: () => Promise<any>) {
  panelRegistry.set(id, defineAsyncComponent(loader))
}

register('watchlist', () => import('./WatchlistPanel.vue'))
register('quote-detail', () => import('./QuoteDetailPanel.vue'))
register('candlestick', () => import('./CandlestickPanel.vue'))
register('order-entry', () => import('./OrderEntryPanel.vue'))
register('position', () => import('./PositionPanel.vue'))
register('news', () => import('./NewsPanel.vue'))
register('ai-chat', () => import('./AIChatPanel.vue'))
register('system-monitor', () => import('./SystemMonitorPanel.vue'))
register('backtest-result', () => import('./BacktestResultPanel.vue'))
register('factor-analysis', () => import('./FactorAnalysisPanel.vue'))
register('portfolio-summary', () => import('./PortfolioSummary.vue'))
register('position-detail', () => import('./PositionDetail.vue'))
register('risk-dashboard', () => import('./RiskDashboard.vue'))
register('trade-history', () => import('./TradeHistory.vue'))
register('schedule-panel', () => import('./SchedulePanel.vue'))
register('notify-panel', () => import('./NotifyPanel.vue'))
register('broker-config', () => import('./BrokerConfig.vue'))
register('settings', () => import('./SettingsPanel.vue'))
register('model-registry', () => import('./ModelRegistryPanel.vue'))
register('prediction-dashboard', () => import('./PredictionDashboardPanel.vue'))
register('alpha-mining', () => import('./AlphaMiningWorkspacePanel.vue'))
register('rl-monitor', () => import('./RLMonitorPanel.vue'))
register('sentiment', () => import('./SentimentPanel.vue'))
register('stock-research', () => import('./StockResearchPanel.vue'))
register('financials', () => import('./FinancialsPanel.vue'))
register('peer-comparison', () => import('./PeerComparisonPanel.vue'))
register('analyst-estimates', () => import('./AnalystEstimatesPanel.vue'))
register('insider-trading', () => import('./InsiderTradingPanel.vue'))
register('congress-trading', () => import('./CongressTradingPanel.vue'))
// Batch 1: Market panels (5)
register('market-overview', () => import('./MarketOverviewPanel.vue'))
register('market-depth', () => import('./MarketDepthPanel.vue'))
register('heatmap', () => import('./HeatmapPanel.vue'))
register('ticker-tape', () => import('./TickerTapePanel.vue'))
register('crypto-overview', () => import('./CryptoOverviewPanel.vue'))
// Batch 1: Chart & Portfolio panels (7)
register('equity-curve', () => import('./EquityCurvePanel.vue'))
register('surface-chart', () => import('./SurfaceChartPanel.vue'))
register('correlation', () => import('./CorrelationPanel.vue'))
register('distribution', () => import('./DistributionPanel.vue'))
register('drawing', () => import('./DrawingPanel.vue'))
register('monte-carlo', () => import('./MonteCarloPanel.vue'))
register('rebalance', () => import('./RebalancePanel.vue'))
// Batch 1: Trading panels (5)
register('order-blotter', () => import('./OrderBlotterPanel.vue'))
register('execution', () => import('./ExecutionPanel.vue'))
register('basket-order', () => import('./BasketOrderPanel.vue'))
register('broker-status', () => import('./BrokerStatusPanel.vue'))
register('action-center', () => import('./ActionCenterPanel.vue'))
// Alternative Data panels
register('prediction-market', () => import('./PredictionMarketPanel.vue'))
register('geopolitics', () => import('./GeopoliticsPanel.vue'))
register('gov-data', () => import('./GovDataPanel.vue'))
// Satellite: NASA POWER solar/wind + FIRMS wildfire
register('satellite', () => import('./SatellitePanel.vue'))

export function getPanelComponent(panelId: string): Component | undefined {
  return panelRegistry.get(panelId)
}

export function getRegisteredPanels(): string[] {
  return Array.from(panelRegistry.keys())
}

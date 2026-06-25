# i18n Migration Plan — 3 Phases, 10 Tasks

## Phase 1: Expand zh.ts

**Task 1**: `frontend/src/lib/i18n/zh.ts` — 写完整 translation key（~400 key，18 domain），保留现有 key 不变，追加新 key。

Commit: `[i18n] zh.ts: complete translation keys for all panels`

---

## Phase 2: Replace Hardcoded Strings (batched by domain)

每个 task 替换一组面板的硬编码字符串为 `$t('domain.key')`。

| Task | Domain | Files |
|------|--------|-------|
| 2.1 | Terminal chrome | StatusBar, CommandBar, SymbolBar, SymbolSearch, PushPinBar, TerminalMode |
| 2.2 | Quote/K-line | CandlestickPanel, QuoteDetailPanel, WatchlistPanel, TickerTapePanel, MarketOverviewPanel, HeatmapPanel |
| 2.3 | Trading | OrderEntryPanel, OrderBlotterPanel, PositionPanel, TradeHistory, ExecutionPanel, PositionDetail, BasketOrderPanel, ActionCenterPanel |
| 2.4 | Portfolio/Risk/Broker | PortfolioSummary, EquityCurvePanel, RiskDashboard, BrokerStatusPanel, BrokerConfig, RebalancePanel |
| 2.5 | Research | StockResearchPanel, SentimentPanel, NewsPanel, FinancialsPanel, AnalystEstimatesPanel, InsiderTradingPanel, PeerComparisonPanel, CongressTradingPanel |
| 2.6 | Macro/Geo/Satellite | GovDataPanel, GeopoliticsPanel, SatellitePanel |
| 2.7 | Prediction/AI/Misc | PredictionMarketPanel, AIChatPanel, MonteCarloPanel, DistributionPanel, CorrelationPanel, DrawingPanel, SurfaceChartPanel, BacktestResultPanel |
| 2.8 | ML/Workflow/System | AlphaMiningWorkspacePanel, FactorAnalysisPanel, ModelRegistryPanel, PredictionDashboardPanel, SystemMonitorPanel, SchedulePanel, NotifyPanel |
| 2.9 | Remaining | WelcomePanel, CryptoOverviewPanel, MarketDepthPanel, RLMonitorPanel |

Each task commits with `[i18n] <domain> panels: migrate to $t()`.

---

## Phase 3: en.ts English Translations

**Task 3**: `frontend/src/lib/i18n/en.ts` — 所有 key 的英文翻译。

Commit: `[i18n] en.ts: complete English translations`

---

## Task 10: Build + CHANGELOG

`npm run build` → CHANGELOG.

Commit: `[Chore] CHANGELOG: i18n migration`

---

## Execution Order

Task 1 → 2.1 → 2.2 → ... → 2.9 → Task 3 → Task 10

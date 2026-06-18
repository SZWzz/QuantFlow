package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"quantflow/internal/ai"
	"quantflow/internal/ai/capabilities"
	"quantflow/internal/config"
	"quantflow/internal/logging"
	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
	"quantflow/internal/notify"
	"quantflow/internal/portfolio"
	"quantflow/internal/research"
	"quantflow/internal/python"
	pb "quantflow/internal/python/proto"
	"quantflow/internal/storage"
	"quantflow/internal/trading"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

// App is the Wails-bound application struct. All exported methods are
// available to the frontend via the generated TypeScript bindings.
type App struct {
	cfg         *config.Config
	registry    *workflow.NodeRegistry
	engine      *workflow.Engine
	bridge      *python.PythonBridge
	capRegistry *ai.CapabilityRegistry
	emitter     *ai.EventEmitter
	profileMgr  *ai.ProfileManager

	// Shared DB connection (opened once at startup, reused across IPC calls).
	db *sql.DB

	// Phase 5
	oms          *trading.OMS
	notifyMgr    *notify.Manager
	portfolioSvc *portfolio.Service

	// Market data: adapter registry + fallback chains (wired in startup).
	marketReg *market.AdapterRegistry
}

// startup is called by Wails when the application starts.
func (a *App) startup() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.cfg = cfg
	// Resolve relative DB path against executable directory so dev/build/.app all share the same DB.
	a.cfg.DBPath = config.ResolveDBPath(a.cfg.DBPath)
	logging.Setup(cfg.LogLevel)

	// Open shared DB connection and run migrations once at startup.
	db, err := storage.Open(a.cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	a.db = db
	migrations, migErr := storage.BuiltinMigrations()
	if migErr == nil {
		if err := storage.Run(db, migrations); err != nil {
			slog.Warn("migrations failed", "error", err)
		}
	}

	a.registry = workflow.NewRegistry()
	nodes.RegisterAll(a.registry)

	engine, err := workflow.NewEngine(a.registry, 256)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}
	a.engine = engine

	// Initialize PythonBridge (optional — app works without Python sidecar)
	bridgeOpts := python.DefaultOptions()
	bridge, err := python.NewPythonBridge(bridgeOpts)
	if err != nil {
		slog.Warn("python sidecar not available, AI features disabled", "error", err)
	} else {
		a.bridge = bridge
		nodes.SetPythonBridge(a.bridge)
		slog.Info("python sidecar connected", "address", bridgeOpts.Address)
	}

	// Wire market-data adapters with fallback chains. mootdx rides the Python
	// sidecar via DataClient; all others are pure-Go HTTP adapters. When the
	// bridge is absent, mootdx gets a nil DataClient → IsAvailable()==false and
	// the CN chain falls through to sina/tushare/eastmoney/…
	a.marketReg = market.NewAdapterRegistry()
	a.registerMarketAdapters()
	slog.Info("market adapter registry initialized", "count", a.marketReg.Count())

	// Initialize CapabilityRegistry
	a.capRegistry = ai.NewCapabilityRegistry()
	capabilities.RegisterQuoteCapabilities(a.capRegistry)
	// Wire market registry so AI agent quotes use real data, not placeholders.
	capabilities.SetMarketRegistry(a.marketReg)
	if a.bridge != nil {
		capabilities.RegisterFactorCapabilities(a.capRegistry, a.bridge)
	}
	capabilities.RegisterSkillCapabilities(a.capRegistry)

	// Initialize EventEmitter
	a.emitter = ai.NewEventEmitter()

	// Initialize ProfileManager
	a.profileMgr = ai.NewProfileManager()
	if err := a.profileMgr.LoadDir("resources/agent-profiles"); err != nil {
		slog.Warn("failed to load agent profiles", "error", err)
	}

	// Set agent dependencies for workflow AgentNode
	if a.bridge != nil {
		nodes.SetAgentDependencies(a.bridge, a.capRegistry, a.emitter, a.profileMgr)
	}

	// Wire ML bridge and model registry to workflow nodes.
	// ModelRegistry needs a DB connection (not yet shared at startup); pass nil
	// so that model persistence is gracefully skipped. The bridge is the critical
	// dependency — it enables gRPC communication with the Python sidecar.
	nodes.SetModelRegistry(nil)

	// Phase 5: Initialize trading OMS and wire to workflow nodes
	a.oms = trading.NewOMS()
	nodes.SetTradingOMS(a.oms)

	// Phase 5: Initialize notification manager (reuses shared DB connection).
	a.notifyMgr = notify.NewManager(a.db)
	a.notifyMgr.Register(notify.NewInAppNotifier())
	slog.Info("notification manager initialized")

	// Phase 5: Initialize portfolio service
	a.portfolioSvc = portfolio.NewService(a.oms)
	slog.Info("portfolio service initialized")

	// Initialize research services (degrade gracefully without Python)
	researchRepo := research.NewResearchRepo(a.db)
	if a.bridge != nil {
		sentimentEngine := research.NewSentimentEngine(a.bridge, researchRepo)
		nodes.SetSentimentEngine(sentimentEngine)
		slog.Info("sentiment engine initialized with Python bridge")
	} else {
		slog.Info("sentiment engine initialized in mock mode (no Python bridge)")
	}
	nodes.SetFinancialsService(research.NewFinancialsService())
	nodes.SetPeerComparisonService(research.NewPeerComparisonService())
	nodes.SetAnalystEstimatesService(research.NewAnalystEstimatesService())
	nodes.SetInsiderTradingService(research.NewInsiderTradingService())
	nodes.SetCongressTradingService(research.NewCongressTradingService())
	slog.Info("research services initialized")

	return nil
}

// GetVersion returns the application version.
func (a *App) GetVersion() string {
	if a.cfg == nil {
		return "unknown"
	}
	return a.cfg.Version
}

// ListNodes returns metadata for all registered workflow node types.
func (a *App) ListNodes() []workflow.NodeMeta {
	if a.registry == nil {
		return nil
	}
	return a.registry.ListAll()
}

// ValidateWorkflow parses and validates a workflow JSON definition.
// Returns "valid" on success, or an error message.
func (a *App) ValidateWorkflow(jsonDef string) (string, error) {
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	if err := workflow.Validate(&wf); err != nil {
		return "", err
	}
	return "valid", nil
}

// RunWorkflow executes a workflow from its JSON definition and returns the result.
func (a *App) RunWorkflow(ctx context.Context, jsonDef string) (*workflow.ExecutionResult, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return a.engine.Execute(ctx, &wf)
}

// LoadWorkflow loads a saved workflow by ID from storage.
func (a *App) LoadWorkflow(id string) (*workflow.Workflow, error) {
	repo := storage.NewWorkflowRepo(a.db)
	return repo.Load(id, nil)
}

// SaveWorkflow persists a workflow definition to storage.
func (a *App) SaveWorkflow(jsonDef string) (string, error) {
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	repo := storage.NewWorkflowRepo(a.db)
	if err := repo.Save(&wf); err != nil {
		return "", err
	}
	return wf.ID, nil
}

// ListWorkflows returns all saved workflows.
func (a *App) ListWorkflows() ([]storage.WorkflowMeta, error) {
	repo := storage.NewWorkflowRepo(a.db)
	return repo.List()
}

// Chat sends a message to the AI agent and returns the response.
// This is called by the AIChatPanel frontend.
func (a *App) Chat(profileName string, model string, message string) (string, error) {
	if a.bridge == nil {
		return "", fmt.Errorf("Python sidecar not connected. Start it with: cd python && python -m src.server")
	}
	if a.profileMgr == nil {
		return "", fmt.Errorf("agent profiles not loaded")
	}

	profile, err := a.profileMgr.Get(profileName)
	if err != nil {
		return "", fmt.Errorf("profile %q: %w", profileName, err)
	}

	pbMessages := []*pb.ChatMessage{
		{Role: "user", Content: message},
	}

	loop := ai.NewAgentLoop(a.bridge, a.capRegistry, a.emitter)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	runID := fmt.Sprintf("chat_%d", time.Now().UnixNano())

	result, err := loop.Run(ctx, runID, pbMessages, profile, model, 0.7)
	if err != nil {
		if err == ai.ErrMaxStepsExceeded {
			return result.FinalContent, nil
		}
		return "", err
	}

	return result.FinalContent, nil
}

// ListProfiles returns available agent profiles for the frontend dropdown.
func (a *App) ListProfiles() []*ai.AgentProfile {
	if a.profileMgr == nil {
		return nil
	}
	return a.profileMgr.List()
}

// — Phase 5: Trading —

// PlaceOrder submits an order to the trading engine (paper or live broker).
func (a *App) PlaceOrder(symbol, side, orderType string, qty, price float64) (*trading.Order, error) {
	if a.oms == nil {
		return nil, fmt.Errorf("OMS not initialized")
	}
	return a.oms.PlaceOrder(symbol, trading.OrderSide(side), trading.OrderType(orderType), qty, price)
}

// GetPositions returns all current positions.
func (a *App) GetPositions() []*trading.Position {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetAllPositions()
}

// GetOrders returns all orders.
func (a *App) GetOrders() []*trading.Order {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetOrders()
}

// GetTrades returns all filled trades.
func (a *App) GetTrades() []*trading.Trade {
	if a.oms == nil {
		return nil
	}
	return a.oms.GetTrades()
}

// — Phase 5: Portfolio —

// GetPortfolioSummary returns current portfolio summary.
func (a *App) GetPortfolioSummary() map[string]interface{} {
	if a.portfolioSvc == nil {
		return map[string]interface{}{"total_value": 0}
	}
	// Cash is tracked separately from OMS positions. Use the OMS to derive
	// market value and P&L; cash must be provided by the caller/trading engine.
	// For the live path, cash starts at 0 until a proper cash ledger is wired.
	cash := 0.0
	if a.oms != nil {
		// Derive estimated cash from trade history: net cash flow from filled orders.
		for _, t := range a.oms.GetTrades() {
			if t.Side == "buy" {
				cash -= t.Price * t.Quantity
			} else {
				cash += t.Price * t.Quantity
			}
		}
	}
	s := a.portfolioSvc.GetSummary(cash)
	return map[string]interface{}{
		"total_value": s.TotalValue, "cash_balance": s.CashBalance,
		"market_value": s.MarketValue, "total_pnl": s.TotalPnL, "total_pnl_pct": s.TotalPnLPct,
	}
}

// GetPortfolioAllocation returns allocation breakdowns.
func (a *App) GetPortfolioAllocation() *portfolio.Allocation {
	if a.portfolioSvc == nil {
		return &portfolio.Allocation{}
	}
	return a.portfolioSvc.GetAllocation()
}

// — Phase 5: Notifications —

// GetNotifications returns recent notifications.
func (a *App) GetNotifications(limit, offset int) []*notify.Notification {
	if a.notifyMgr == nil {
		return nil
	}
	notifications, _ := a.notifyMgr.GetHistory(limit, offset)
	return notifications
}

// MarkNotificationRead marks a notification as read.
func (a *App) MarkNotificationRead(id int64) error {
	if a.notifyMgr == nil {
		return fmt.Errorf("notify manager not initialized")
	}
	return a.notifyMgr.MarkRead(id)
}

// — Market Data —

// registerMarketAdapters populates the adapter registry with every data source,
// in the order the fallback chains expect (see market.FallbackChains). mootdx
// alone needs the Python sidecar; it degrades to IsAvailable()==false when the
// bridge is absent. Extracted from startup() so tests can exercise registration
// without config/storage.
func (a *App) registerMarketAdapters() {
	var dataClient *python.DataClient
	if a.bridge != nil {
		dataClient = python.NewDataClient(a.bridge)
	}
	// CN chain order: mootdx → sina → tushare → eastmoney → tencent → baidu → akshare.
	a.marketReg.Register(adapters.NewMootdxAdapter(dataClient))
	a.marketReg.Register(adapters.NewSinaAdapter())
	a.marketReg.Register(adapters.NewTuShareAdapter())
	a.marketReg.Register(adapters.NewEastMoneyAdapter())
	a.marketReg.Register(adapters.NewTencentAdapter())
	a.marketReg.Register(adapters.NewBaiduAdapter())
	a.marketReg.Register(adapters.NewAKShareAdapter())
	// US / HK / CRYPTO chains.
	a.marketReg.Register(adapters.NewYahooAdapter())
	a.marketReg.Register(adapters.NewPolygonAdapter())
	a.marketReg.Register(adapters.NewOKXAdapter())
	a.marketReg.Register(adapters.NewBinanceAdapter())
	a.marketReg.Register(adapters.NewCoinGeckoAdapter())
}

// GetQuote fetches a real-time quote for a symbol via the market's fallback
// chain (e.g. "CN" → mootdx→sina→tushare→…). Returns the snapshot and the name
// of the adapter that succeeded. marketName is one of "CN", "US", "HK", "CRYPTO".
func (a *App) GetQuote(ctx context.Context, marketName, symbol string) (*market.QuoteSnapshot, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchQuoteWithFallback(ctx, marketName, symbol)
}

// FetchOHLCV fetches OHLCV bars for a symbol via the market's fallback chain.
// interval is one of "1D", "1W", "1M", "1m", "5m", "15m", "30m", "1H"; start/end
// are Unix timestamps in seconds. Returns the bars and the adapter name that
// succeeded.
func (a *App) FetchOHLCV(ctx context.Context, marketName, symbol, interval string, start, end int64) ([]market.OHLCVBar, string, error) {
	if a.marketReg == nil {
		return nil, "", fmt.Errorf("market registry not initialized")
	}
	return a.marketReg.FetchOHLCVWithFallback(ctx, marketName, symbol, interval, start, end)
}

// — Research —

// GetSentiment returns sentiment analysis for a symbol.
func (a *App) GetSentiment(symbol string) (*research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db))
	return engine.AnalyzeSentiment(context.Background(), symbol, "", "news", "en")
}

// GetSentimentHistory returns historical sentiment for a symbol.
func (a *App) GetSentimentHistory(symbol string, days int) ([]research.SentimentOutput, error) {
	engine := research.NewSentimentEngine(a.bridge, research.NewResearchRepo(a.db))
	return engine.GetSentimentHistory(context.Background(), symbol, days)
}

// GetStockResearch returns multi-dimensional research data for a symbol.
func (a *App) GetStockResearch(symbol string, tabs []string) (*research.StockResearchResult, error) {
	repo := research.NewResearchRepo(a.db)
	finSvc := research.NewFinancialsService()
	peerSvc := research.NewPeerComparisonService()
	estSvc := research.NewAnalystEstimatesService()
	insSvc := research.NewInsiderTradingService()

	result := &research.StockResearchResult{
		Symbol: symbol,
		Overview: map[string]interface{}{
			"symbol": symbol, "name": symbol,
			"sector": "Mock", "market_cap": "N/A",
		},
	}

	for _, tab := range tabs {
		switch tab {
		case "financials":
			fd, _ := finSvc.GetFinancials(context.Background(), symbol)
			result.Financials = fd
			result.Ratios = finSvc.ComputeRatios(fd)
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
			if a.bridge != nil {
				engine := research.NewSentimentEngine(a.bridge, repo)
				s, _ := engine.AnalyzeSentiment(context.Background(), symbol, "", "news", "en")
				result.Sentiment = s
			}
		}
	}

	return result, nil
}

// Shutdown performs graceful cleanup: closes the Python sidecar connection,
// shared DB connection, and releases any resources held by the application.
func (a *App) Shutdown() {
	if a.bridge != nil {
		if err := a.bridge.Close(); err != nil {
			slog.Warn("failed to close python bridge", "error", err)
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			slog.Warn("failed to close database", "error", err)
		}
	}
	slog.Info("app shutdown complete")
}

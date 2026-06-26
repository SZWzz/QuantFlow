package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

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
	"quantflow/internal/schedule"
	"quantflow/internal/storage"
	"quantflow/internal/trading"
	"quantflow/internal/trading/brokers"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
)

var startTime = time.Now() // used by GetSystemStats for uptime

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
	minuteCache *market.MinuteCache

	// Shared DB connection (opened once at startup, reused across IPC calls).
	db *sql.DB

	// Phase 5
	oms          *trading.OMS
	notifyMgr     *notify.Manager
	scheduleRepo  *schedule.Repo
	sched         *schedule.Scheduler
	portfolioSvc *portfolio.Service

	// Market data: adapter registry + fallback chains (wired in startup).
	marketReg     *market.AdapterRegistry
	newsAdpt      adapters.NewsAdapter // news source for sentiment analysis
	globalNewsAdpt *adapters.EastMoneyGlobalNewsAdapter
	conceptAdpt   *adapters.EastMoneyConceptAdapter
	signalsAdpt   *adapters.EastMoneySignalsAdapter
	capitalAdpt   *adapters.EastMoneyCapitalAdapter
	fundFlowAdpt  *adapters.EastMoneyFundFlowAdapter
	sinaFinAdpt   *adapters.SinaFinancialsAdapter
	reportAdpt     *adapters.EastMoneyReportAdapter
	consensusAdpt  *adapters.THSConsensusAdapter
	thsHotAdpt     *adapters.THSHotAdapter
	northboundAdpt *adapters.THSNorthboundAdapter
	cninfoAdpt     *adapters.CninfoAdapter
	iwencaiAdpt    *adapters.IwencaiAdapter
	eastmoneyAdpt  *adapters.EastMoneyAdapter       // stock info for research overview
	congressAdpt   *adapters.CongressTradesAdapter  // US congressional trades

	polymarketAdpt      adapters.PolymarketAdapter  // prediction market data source
	predictionMarketSvc *research.PredictionMarketService

	geopoliticsAdpt adapters.GeopoliticsAdapter // geopolitical data source (GDELT)
	geopoliticsSvc  *research.GeopoliticsService

	govDataAdpt adapters.GovDataAdapter // economic indicator data source (FRED + SEC)
	govDataSvc  *research.GovDataService

	satelliteAdpt adapters.SatelliteAdapter // satellite energy data source (NASA POWER + FIRMS)
	satelliteSvc  *research.SatelliteService

	// Research services (wired in startup, degrade gracefully without adapters).
	capitalSvc      *research.CapitalService
	fundFlowSvc     *research.FundFlowService
	northboundSvc   *research.NorthboundService
	announcementSvc *research.AnnouncementService

	// Symbol search (in-memory index, built at startup).
	searchSvc *market.SymbolSearchService

	// Python sidecar subprocess (auto-launched).
	sidecar *python.SidecarProcess
}

// ServiceStartup is called by Wails v3 when the application starts.
// It initializes all services: config, DB, market adapters, research, etc.
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
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

	mc, err := market.NewMinuteCache(a.db)
	if err != nil {
		slog.Error("failed to init minute cache", "err", err)
	} else {
		a.minuteCache = mc
	}

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

	// Auto-start Python sidecar (launches mootdx/TDX, AI, etc.)
	// Resolve python dir relative to executable so it works regardless of cwd.
	execPath, _ := os.Executable()
	pythonDir := filepath.Join(filepath.Dir(execPath), "python")
	sidecar, sidecarErr := python.StartSidecar(context.Background(), pythonDir, 50051)
	if sidecarErr != nil {
		slog.Warn("python sidecar launch failed, AI features disabled", "error", sidecarErr, "python_dir", pythonDir)
	} else {
		a.sidecar = sidecar
		if sidecar != nil {
			slog.Info("python sidecar launched")
			sidecar.Wait() // wait for readiness
		}
	}

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

	// Initialize MarketDataHub for real-time pub/sub (audit fix M7).
	// Currently a stub — full topic broker activation pending per-symbol subscription UI.
	_ = market.NewHub() // hub created, topic subscriptions deferred
	slog.Info("market data hub initialized")

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

	// Phase 5: Initialize broker adapters. Alpaca (US equities) is optional —
	// when ALPACA_API_KEY is not set, the broker stays disconnected and panels
	// show "Not Configured" state gracefully.
	if alpacaBroker := brokers.NewAlpacaBroker(brokers.AlpacaConfig{}); alpacaBroker != nil {
		if err := alpacaBroker.Connect(context.Background()); err != nil {
			slog.Warn("alpaca broker not available — US trading disabled", "error", err)
		} else {
			a.oms.SetBroker(alpacaBroker)
			slog.Info("alpaca broker connected — US equities trading enabled")
		}
	}

	// Phase 5: Initialize notification manager (reuses shared DB connection).
	a.notifyMgr = notify.NewManager(a.db)
	a.notifyMgr.Register(notify.NewInAppNotifier())
	slog.Info("notification manager initialized")

	// Phase 5: Initialize portfolio service
	a.portfolioSvc = portfolio.NewService(a.oms)
	slog.Info("portfolio service initialized")

	// Phase 5: Initialize schedule task repo (CRUD for scheduled workflow runs).
	a.scheduleRepo = schedule.NewRepo(a.db)
	slog.Info("schedule task repo initialized")

	// Start cron scheduler so scheduled tasks actually execute.
	a.sched = schedule.New(a.db, workflowExecutorAdapter{a: a}, nil)
	if err := a.sched.Start(); err != nil {
		slog.Warn("cron scheduler start skipped", "error", err)
	} else {
		slog.Info("cron scheduler started")
	}

	// Initialize research services (degrade gracefully without Python)
	researchRepo := research.NewResearchRepo(a.db)
	a.newsAdpt = adapters.NewEastMoneyNewsAdapter()
	a.globalNewsAdpt = adapters.NewEastMoneyGlobalNewsAdapter()
	nodes.SetGlobalNewsAdapter(a.globalNewsAdpt)
	a.conceptAdpt = adapters.NewEastMoneyConceptAdapter()
	a.signalsAdpt = adapters.NewEastMoneySignalsAdapter()
	a.capitalAdpt = adapters.NewEastMoneyCapitalAdapter()
	a.fundFlowAdpt = adapters.NewEastMoneyFundFlowAdapter()
	a.eastmoneyAdpt = adapters.NewEastMoneyAdapter()
	a.northboundAdpt = adapters.NewTHSNorthboundAdapter()
	a.sinaFinAdpt = adapters.NewSinaFinancialsAdapter()

	// Phase 3: Research report + announcement adapters
	a.reportAdpt = adapters.NewEastMoneyReportAdapter()
	a.consensusAdpt = adapters.NewTHSConsensusAdapter()
	a.cninfoAdpt = adapters.NewCninfoAdapter()
	a.iwencaiAdpt = adapters.NewIwencaiAdapter()
	a.congressAdpt = adapters.NewCongressTradesAdapter()

	nodes.SetNewsAdapter(a.newsAdpt)
	if a.bridge != nil {
		sentimentEngine := research.NewSentimentEngine(a.bridge, researchRepo, a.newsAdpt)
		nodes.SetSentimentEngine(sentimentEngine)
		slog.Info("sentiment engine initialized with Python bridge")
	} else {
		sentimentEngine := research.NewSentimentEngine(nil, researchRepo, a.newsAdpt)
		nodes.SetSentimentEngine(sentimentEngine)
		slog.Info("sentiment engine initialized in mock mode (no Python bridge)")
	}
	nodes.SetFinancialsService(research.NewFinancialsService(a.sinaFinAdpt, a.getMootdxAdapter()))
	nodes.SetPeerComparisonService(research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt))
	nodes.SetAnalystEstimatesService(research.NewAnalystEstimatesService(a.reportAdpt, a.consensusAdpt))
	nodes.SetInsiderTradingService(research.NewInsiderTradingService())
	nodes.SetCongressTradingService(research.NewCongressTradingService(a.congressAdpt))
	slog.Info("research services initialized")

	// Symbol search service (in-memory A-share index)
	searchSvc, err := market.NewSymbolSearchService(context.Background())
	if err != nil {
		slog.Warn("symbol search service init failed", "error", err)
	} else {
		a.searchSvc = searchSvc
		slog.Info("symbol search service initialized", "stocks", searchSvc.Size())
	}
	a.capitalSvc = research.NewCapitalService(a.capitalAdpt)
	nodes.SetCapitalService(a.capitalSvc)
	a.fundFlowSvc = research.NewFundFlowService(a.fundFlowAdpt)
	nodes.SetFundFlowService(a.fundFlowSvc)
	a.northboundSvc = research.NewNorthboundService(a.northboundAdpt)
	nodes.SetNorthboundService(a.northboundSvc)
	a.announcementSvc = research.NewAnnouncementService(a.cninfoAdpt)
	nodes.SetAnnouncementService(a.announcementSvc)

	// Alternative data: prediction market (Polymarket)
	a.polymarketAdpt = adapters.NewPolymarketAdapter()
	a.predictionMarketSvc = research.NewPredictionMarketService(a.polymarketAdpt)
	nodes.SetPredictionMarketService(a.predictionMarketSvc)
	slog.Info("prediction market service initialized")

	// Alternative data: geopolitics (GDELT)
	a.geopoliticsAdpt = adapters.NewGDELTAdapter()
	a.geopoliticsSvc = research.NewGeopoliticsService(a.geopoliticsAdpt)
	nodes.SetGeopoliticsService(a.geopoliticsSvc)
	slog.Info("geopolitics service initialized")

	// Alternative data: govdata (FRED + SEC EDGAR)
	a.govDataAdpt = adapters.NewGovDataAdapter()
	a.govDataSvc = research.NewGovDataService(a.govDataAdpt)
	// Inject FRED API key from config (with env var fallback in NewGovDataAdapter)
	if gha, ok := a.govDataAdpt.(*adapters.GovDataHTTPAdapter); ok {
		gha.SetAPIKey(a.cfg.GetAPIKey("fred"))
	}
	nodes.SetGovDataService(a.govDataSvc)
	slog.Info("govdata service initialized")

	// Alternative data: satellite (NASA POWER + FIRMS)
	a.satelliteAdpt = adapters.NewSatelliteAdapter()
	a.satelliteSvc = research.NewSatelliteService(a.satelliteAdpt)
	nodes.SetSatelliteService(a.satelliteSvc)
	slog.Info("satellite service initialized")
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

// GetNodePorts returns the input/output port definitions for a given node type.
func (a *App) GetNodePorts(nodeType string) (map[string]any, error) {
	if a.registry == nil {
		return nil, fmt.Errorf("registry not initialized")
	}
	node, err := a.registry.Create(nodeType, "__dummy__", nil)
	if err != nil {
		return nil, fmt.Errorf("create node %q: %w", nodeType, err)
	}
	inputs := make([]map[string]any, 0)
	for _, p := range node.InputPorts() {
		inputs = append(inputs, map[string]any{"name": p.Name, "type": string(p.Type)})
	}
	outputs := make([]map[string]any, 0)
	for _, p := range node.OutputPorts() {
		outputs = append(outputs, map[string]any{"name": p.Name, "type": string(p.Type)})
	}
	return map[string]any{"inputs": inputs, "outputs": outputs}, nil
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

// ExecuteWorkflowByID loads a saved workflow by ID and executes it.
// Used by the cron scheduler's WorkflowExecutor interface.
func (a *App) ExecuteWorkflowByID(ctx context.Context, workflowID string) (string, error) {
	repo := storage.NewWorkflowRepo(a.db)
	wf, err := repo.Load(workflowID, nil)
	if err != nil {
		return "", fmt.Errorf("load workflow %q: %w", workflowID, err)
	}
	result, err := a.engine.Execute(ctx, wf)
	if err != nil {
		return result.WorkflowID, err
	}
	return result.WorkflowID, nil
}

// workflowExecutorAdapter adapts App to the schedule.WorkflowExecutor interface.
type workflowExecutorAdapter struct {
	a *App
}

func (w workflowExecutorAdapter) Execute(ctx context.Context, workflowID string) (string, error) {
	return w.a.ExecuteWorkflowByID(ctx, workflowID)
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
	// CN chain: tencent(quickest)→eastmoney→mootdx(intraday)→...
	// Tencent ~76ms HTTP, EastMoney ~350ms HTTPS, mootdx ~4s via Python sidecar.
	a.marketReg.Register(adapters.NewMootdxAdapter(dataClient))
	a.marketReg.Register(adapters.NewSinaAdapter())
	a.marketReg.Register(adapters.NewTuShareAdapter())
	a.marketReg.Register(adapters.NewEastMoneyAdapter())
	a.marketReg.Register(adapters.NewTencentAdapter())
	a.marketReg.Register(adapters.NewBaiduAdapter())
	a.marketReg.Register(adapters.NewAKShareAdapter())
	// US / HK / CRYPTO chains.
	a.marketReg.Register(adapters.NewYahooAdapter())
	finnhubAdpt := adapters.NewFinnhubAdapter()
	finnhubAdpt.SetAPIKey(a.cfg.GetAPIKey("finnhub"))
	a.marketReg.Register(finnhubAdpt)
	a.marketReg.Register(adapters.NewPolygonAdapter())
	a.marketReg.Register(adapters.NewGateIOAdapter()) // primary crypto (accessible from CN)
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

// GetMinuteLine returns today's intraday minute-by-minute ticks for a CN symbol.
// If sinceTimestamp is 0, returns all ticks for today.
// If sinceTimestamp > 0, returns only ticks after the given Unix timestamp.
// Data is cached in SQLite + LRU; source data comes from mootdx when not cached.
func (a *App) GetMinuteLine(ctx context.Context, symbol string, sinceTimestamp int64) ([]market.MinuteTick, string, error) {
	if a.minuteCache == nil {
		return nil, "unavailable", fmt.Errorf("minute cache not initialized")
	}

	mkt := market.MarketForSymbol(symbol)

	// Non-CN markets: minute data not available via free adapters,
	// return daily OHLCV as fallback (frontend will display as daily bars).
	if mkt != "CN" {
		return nil, "unavailable", fmt.Errorf("minute data not available for market %s, use 1d interval instead", mkt)
	}

	// 1. Try cache first (SQLite + LRU).
	ticks, err := a.minuteCache.GetIncremental(symbol, sinceTimestamp)
	if err != nil {
		slog.Warn("minute_cache: get failed", "symbol", symbol, "err", err)
	}

	// 2. If cache has data and the request is incremental (since > 0),
	//    return cached data. For initial load (since == 0), if cache
	//    is empty, fall through to live fetch.
	if len(ticks) > 0 || sinceTimestamp > 0 {
		return ticks, "cache", nil
	}

	// 3. Live fetch via mootdx (CN only).
	adpt := a.getMootdxAdapter()
	if adpt == nil {
		return nil, "unavailable", fmt.Errorf("mootdx adapter not available")
	}
	liveTicks, err := adpt.FetchMinuteLine(symbol)
	if err != nil {
		return nil, "unavailable", err
	}

	// 4. Persist live data to cache.
	if len(liveTicks) > 0 {
		today := time.Now().Format("2006-01-02")
		if err := a.minuteCache.SaveTicks(symbol, today, liveTicks); err != nil {
			slog.Warn("minute_cache: save failed", "symbol", symbol, "err", err)
		}
	}

	return liveTicks, "mootdx", nil
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
	peerSvc := research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt)
	estSvc := research.NewAnalystEstimatesService(a.reportAdpt, a.consensusAdpt)
	insSvc := research.NewInsiderTradingService()

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

// SearchSymbols searches A-share stocks by code, name, or pinyin abbreviation.
// Returns up to 20 matches sorted by relevance. Returns empty slice (not error)
// when the search service is unavailable.
func (a *App) SearchSymbols(query string) ([]market.StockEntry, error) {
	if a.searchSvc == nil {
		return []market.StockEntry{}, nil
	}
	return a.searchSvc.Search(query, 20), nil
}

// SearchResearch performs NL semantic search over research reports, announcements,
// and news via the iwencai (爱问财) API. Requires IWENCAI_API_KEY to be configured.
// channel: "report", "announcement", or "news". size: max results (capped at 50).
func (a *App) SearchResearch(query string, channel string, size int) ([]adapters.IwencaiArticle, error) {
	if a.iwencaiAdpt == nil {
		return nil, fmt.Errorf("iwencai adapter not initialized")
	}
	if !a.iwencaiAdpt.IsAvailable(context.Background()) {
		return nil, fmt.Errorf("iwencai not available: IWENCAI_API_KEY not set or endpoint unreachable")
	}
	return a.iwencaiAdpt.Search(context.Background(), query, channel, size)
}

// ── Capital Data (融资融券 / 大宗交易 / 股东户数 / 分红) ──────────────────

// GetCapitalData returns capital/fundamental data for a symbol: margin trading,
// block trades, holder changes, and dividend history.
func (a *App) GetCapitalData(symbol string) (map[string]interface{}, error) {
	if a.capitalSvc == nil {
		return nil, fmt.Errorf("capital service not initialized")
	}
	ctx := context.Background()
	result := map[string]interface{}{}
	if data, err := a.capitalSvc.GetMarginTrading(ctx, symbol, 30); err == nil {
		result["margin_trading"] = data
	}
	if data, err := a.capitalSvc.GetBlockTrades(ctx, symbol, 20); err == nil {
		result["block_trades"] = data
	}
	if data, err := a.capitalSvc.GetHolderChanges(ctx, symbol, 10); err == nil {
		result["holder_changes"] = data
	}
	if data, err := a.capitalSvc.GetDividendHistory(ctx, symbol, 20); err == nil {
		result["dividend_history"] = data
	}
	return result, nil
}

// ── Fund Flow (资金流向) ──────────────────────────────────────────────────

// GetFundFlow returns capital flow data for a symbol.
// flowType: "minute" (今日分钟级) or "daily" (120日日级).
func (a *App) GetFundFlow(symbol string, flowType string) (interface{}, error) {
	if a.fundFlowSvc == nil {
		return nil, fmt.Errorf("fund flow service not initialized")
	}
	ctx := context.Background()
	switch flowType {
	case "minute":
		return a.fundFlowSvc.GetMinuteFlow(ctx, symbol)
	case "daily":
		return a.fundFlowSvc.GetDailyFlow(ctx, symbol)
	default:
		return nil, fmt.Errorf("invalid flowType: %s (use 'minute' or 'daily')", flowType)
	}
}

// ── Northbound Flow (北向资金) ────────────────────────────────────────────

// GetNorthboundFlow returns northbound capital flow data.
func (a *App) GetNorthboundFlow() (map[string]interface{}, error) {
	if a.northboundSvc == nil {
		return nil, fmt.Errorf("northbound service not initialized")
	}
	ctx := context.Background()
	result := map[string]interface{}{}
	if data, err := a.northboundSvc.GetMinuteFlow(ctx); err == nil {
		result["minute_flow"] = data
	}
	if data, err := a.northboundSvc.GetHistory(20); err == nil {
		result["history"] = data
	}
	return result, nil
}

// ── Announcements (公告) ──────────────────────────────────────────────────

// GetAnnouncements returns company announcements for a symbol from 巨潮资讯.
func (a *App) GetAnnouncements(symbol string, pageSize int) ([]adapters.Announcement, error) {
	if a.announcementSvc == nil {
		return nil, fmt.Errorf("announcement service not initialized")
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	return a.announcementSvc.GetAnnouncements(context.Background(), symbol, pageSize)
}

// ── Dragon Tiger (龙虎榜) ─────────────────────────────────────────────────

// GetDragonTiger returns dragon tiger board data for a symbol.
func (a *App) GetDragonTiger(symbol string, endDate string, lookBack int) ([]adapters.DragonTigerRecord, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	if lookBack <= 0 {
		lookBack = 30
	}
	return a.signalsAdpt.FetchDragonTigerStock(context.Background(), symbol, endDate, lookBack)
}

// GetDailyDragonTiger returns market-wide dragon tiger board for a trading date.
func (a *App) GetDailyDragonTiger(date string, minNetBuy float64) ([]adapters.DragonTigerStock, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	return a.signalsAdpt.FetchDailyDragonTiger(context.Background(), date, minNetBuy)
}

// GetLockupExpiry returns lockup expiry data (解禁) for a symbol.
func (a *App) GetLockupExpiry(symbol string) ([]adapters.LockupExpiry, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	return a.signalsAdpt.FetchLockupExpiry(context.Background(), symbol)
}

// ── Industry Ranks (行业板块排名) ─────────────────────────────────────────

// GetIndustryRanks returns industry ranking by change percent.
// Returns empty slice on error (eastmoney push2 API is frequently unavailable).
func (a *App) GetIndustryRanks(topN int) ([]adapters.IndustryRank, error) {
	if a.signalsAdpt == nil {
		return []adapters.IndustryRank{}, nil
	}
	if topN <= 0 {
		topN = 20
	}
	ranks, err := a.signalsAdpt.FetchIndustryRanks(context.Background(), topN)
	if err != nil {
		slog.Warn("GetIndustryRanks failed, returning empty", "error", err)
		return []adapters.IndustryRank{}, nil
	}
	return ranks, nil
}

// ── Concept Blocks (概念板块归属) ─────────────────────────────────────────

// GetConceptBlocks returns the concept/industry/sector blocks a stock belongs to.
func (a *App) GetConceptBlocks(symbol string) ([]adapters.ConceptBlock, error) {
	if a.conceptAdpt == nil {
		return nil, fmt.Errorf("concept adapter not initialized")
	}
	return a.conceptAdpt.FetchConceptBlocks(context.Background(), symbol)
}

// ── Prediction Market (预测市场) ──────────────────────────────────────

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

// ── Geopolitics (地缘政治风险) ──────────────────────────────────────────────

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

// ── GovData / Economic Indicators (FRED + SEC EDGAR) ─────────────────────

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

// ── Satellite (卫星能源数据) ─────────────────────────────────────────────

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

// getMootdxAdapter retrieves the mootdx adapter from the market registry.
// Returns nil if the Python sidecar is not connected.
func (a *App) getMootdxAdapter() *adapters.MootdxAdapter {
	if a.marketReg == nil {
		return nil
	}
	adpt := a.marketReg.Get("mootdx")
	if adpt == nil {
		return nil
	}
	mootdx, ok := adpt.(*adapters.MootdxAdapter)
	if !ok {
		return nil
	}
	return mootdx
}

// getMarketReg returns the market adapter registry or nil.
func (a *App) getMarketReg() *market.AdapterRegistry {
	return a.marketReg
}

// — Market Overview & Analytics —

// GetMarketOverview returns index snapshots for major A-share indices.
// Queries sina directly — mootdx confuses index codes with individual stocks
// (e.g. 000001.SH returns 平安银行's price instead of 上证指数).
func (a *App) GetMarketOverview() (map[string]interface{}, error) {
	ctx := context.Background()
	// Index codes: use sina format (sh/sz prefix) for sina adapter,
	// bare codes for GetQuote fallback (which does its own normalization).
	indices := []struct{ code, sinaCode, name string }{
		{"000001.SH", "sh000001", "上证指数"},
		{"399001.SZ", "sz399001", "深证成指"},
		{"399006.SZ", "sz399006", "创业板指"},
		{"000688.SH", "sh000688", "科创50"},
		{"000300.SH", "sh000300", "沪深300"},
	}
	// Use sina adapter directly — mootdx doesn't handle index codes correctly.
	sina := a.marketReg.Get("sina")
	result := make([]map[string]interface{}, 0, len(indices))
	for _, idx := range indices {
		var snap *market.QuoteSnapshot
		var err error
		if sina != nil && sina.IsAvailable(ctx) {
			snap, err = sina.FetchQuote(ctx, idx.sinaCode)
		} else {
			snap, _, err = a.GetQuote(ctx, "CN", idx.code)
		}
		if err != nil {
			slog.Warn("GetMarketOverview: failed for", "code", idx.code, "error", err)
			continue
		}
		result = append(result, map[string]interface{}{
			"code":       idx.code,
			"name":       idx.name,
			"price":      snap.Last,
			"change":     snap.Change,
			"change_pct": snap.ChangePct,
		})
	}
	return map[string]interface{}{
		"indices": result,
		"breadth": map[string]int{"advancers": 0, "decliners": 0, "unchanged": 0},
	}, nil
}

// GetMarketSnapshot returns batch quotes for a list of symbols.
func (a *App) GetMarketSnapshot(ctx context.Context, symbols []string) ([]map[string]interface{}, error) {
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	result := make([]map[string]interface{}, 0, len(symbols))
	for _, sym := range symbols {
		mkt := market.MarketForSymbol(sym)
		snap, _, err := reg.FetchQuoteWithFallback(ctx, mkt, sym)
		if err != nil {
			continue
		}
		result = append(result, map[string]interface{}{
			"symbol":     sym,
			"price":      snap.Last,
			"change":     snap.Change,
			"change_pct": snap.ChangePct,
			"volume":     snap.Volume,
		})
	}
	return result, nil
}

// GetCryptoOverview returns quotes for major crypto pairs.
func (a *App) GetCryptoOverview(ctx context.Context, symbols []string) (map[string]interface{}, error) {
	if len(symbols) == 0 {
		symbols = []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "XRPUSDT", "ADAUSDT", "DOGEUSDT", "DOTUSDT"}
	}
	reg := a.getMarketReg()
	results := make([]map[string]interface{}, 0)
	for _, sym := range symbols {
		snap, _, err := reg.FetchQuoteWithFallback(ctx, "CRYPTO", sym)
		if err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"symbol":     sym,
			"price":      snap.Last,
			"change_pct": snap.ChangePct,
		})
	}
	return map[string]interface{}{"cryptos": results}, nil
}

// GetCorrelationMatrix computes the Pearson correlation matrix for a set of symbols.
func (a *App) GetCorrelationMatrix(ctx context.Context, symbols []string, lookback int) (map[string]map[string]float64, error) {
	reg := a.getMarketReg()
	returns := make(map[string][]float64)
	end := time.Now().Unix()
	start := end - int64(lookback*86400)
	for _, sym := range symbols {
		mkt := market.MarketForSymbol(sym)
		bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, sym, "1d", start, end)
		if err != nil || len(bars) < 2 {
			continue
		}
		rets := make([]float64, 0, len(bars)-1)
		for i := 1; i < len(bars); i++ {
			if bars[i-1].Close > 0 {
				rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
			}
		}
		returns[sym] = rets
	}
	return portfolio.CorrelationMatrix(returns), nil
}

// GetReturnDistribution computes a histogram of daily log returns for a symbol.
func (a *App) GetReturnDistribution(ctx context.Context, symbol string, lookback int, bins int) (map[string]interface{}, error) {
	reg := a.getMarketReg()
	mkt := market.MarketForSymbol(symbol)
	end := time.Now().Unix()
	start := end - int64(lookback*86400)
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", start, end)
	if err != nil || len(bars) < 2 {
		return nil, fmt.Errorf("insufficient data for %s: %w", symbol, err)
	}
	rets := make([]float64, 0, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
	}
	histBins, histCounts := portfolio.ReturnDistribution(rets, bins)
	return map[string]interface{}{
		"symbol": symbol,
		"bins":   histBins,
		"counts": histCounts,
	}, nil
}

// GetVolatilitySurface computes historical volatility across multiple time windows.
func (a *App) GetVolatilitySurface(ctx context.Context, symbol string) ([][]float64, error) {
	reg := a.getMarketReg()
	mkt := market.MarketForSymbol(symbol)
	end := time.Now().Unix()
	start := end - int64(365*86400)
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", start, end)
	if err != nil || len(bars) < 5 {
		return nil, fmt.Errorf("insufficient data for %s: %w", symbol, err)
	}
	rets := make([]float64, 0, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		if bars[i-1].Close > 0 {
			rets = append(rets, math.Log(bars[i].Close/bars[i-1].Close))
		}
	}
	return portfolio.VolatilitySurface(rets, []int{5, 10, 20, 30, 60, 90, 120, 252}), nil
}

// GetRebalanceSuggestions returns rebalance advice.
func (a *App) GetRebalanceSuggestions(ctx context.Context) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// GetSystemStats returns runtime statistics for the system monitor panel.
func (a *App) GetSystemStats(ctx context.Context) map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"mem_alloc_mb":  int(m.Alloc / 1024 / 1024),
		"mem_sys_mb":    int(m.Sys / 1024 / 1024),
		"num_gc":        int(m.NumGC),
		"go_version":    runtime.Version(),
		"uptime_seconds": int(time.Since(startTime).Seconds()),
	}
}

// GetConfig returns the current application configuration (non-sensitive).
func (a *App) GetConfig() map[string]interface{} {
	// Frontend settings (theme/language/density) live in localStorage.
	// Only expose non-sensitive config here. api_keys are NEVER sent to frontend (audit fix C4).
	return map[string]interface{}{
		"version":  a.cfg.Version,
		"logLevel": a.cfg.LogLevel,
		"dbPath":   a.cfg.DBPath,
	}
}

// UpdateConfig merges partial config into the current config and persists to config.yaml.
func (a *App) UpdateConfig(ctx context.Context, patch map[string]interface{}) error {
	if keys, ok := patch["api_keys"].(map[string]interface{}); ok {
		for k, v := range keys {
			if s, ok := v.(string); ok {
				a.cfg.APIKeys[k] = s
			}
		}
	}
	return a.cfg.Save()
}

// GetCommodityQuotes returns real-time WTI crude oil and natural gas prices from Sina futures.
// ── News ─────────────────────────────────────────────────────────────────────

// NewsItem is a lightweight news article for the frontend news panel.
type NewsItem struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Time   string `json:"time"`
	URL    string `json:"url,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

// GetNews fetches recent news. When symbol is empty, fetches global market news.
func (a *App) GetNews(symbol string, limit int) ([]NewsItem, error) {
	if limit <= 0 { limit = 20 }
	if limit > 50 { limit = 50 }
	ctx := context.Background()

	if symbol == "" && a.globalNewsAdpt != nil {
		articles, err := a.globalNewsAdpt.FetchGlobalNews(ctx, limit)
		if err != nil { return nil, fmt.Errorf("global news: %w", err) }
		items := make([]NewsItem, 0, len(articles))
		for _, art := range articles {
			items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL})
		}
		return items, nil
	}

	if a.newsAdpt == nil { return nil, fmt.Errorf("news adapter not available") }
	articles, err := a.newsAdpt.FetchStockNews(ctx, symbol, limit)
	if err != nil { return nil, fmt.Errorf("stock news: %w", err) }
	items := make([]NewsItem, 0, len(articles))
	for _, art := range articles {
		items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL, Symbol: art.Symbol})
	}
	return items, nil
}

// ── Broker Status ───────────────────────────────────────────────────────────

// BrokerStatus reports connection state for a broker.
type BrokerStatus struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Market    string `json:"market"`
	Connected bool   `json:"connected"`
	Detail    string `json:"detail"`
}

// GetBrokerStatuses returns connection status of all registered brokers.
func (a *App) GetBrokerStatuses() []BrokerStatus {
	return []BrokerStatus{
		{Name: "paper", Label: "Paper Trading", Market: "模拟", Connected: true, Detail: "本地模拟撮合"},
	}
}

// ── Schedule Tasks ──────────────────────────────────────────────────────────

func (a *App) ListScheduleTasks() ([]schedule.Task, error) {
	if a.scheduleRepo == nil { return nil, fmt.Errorf("schedule not available") }
	tasks, err := a.scheduleRepo.List()
	if err != nil { return nil, err }
	result := make([]schedule.Task, 0, len(tasks))
	for _, t := range tasks {
		if t != nil { result = append(result, *t) }
	}
	return result, nil
}

func (a *App) SaveScheduleTask(task schedule.Task) error {
	if a.scheduleRepo == nil { return fmt.Errorf("schedule not available") }
	if task.ID == "" { return a.scheduleRepo.Create(&task) }
	return a.scheduleRepo.Update(&task)
}

func (a *App) DeleteScheduleTask(id string) error {
	if a.scheduleRepo == nil { return fmt.Errorf("schedule not available") }
	return a.scheduleRepo.Delete(id)
}

func (a *App) ToggleScheduleTask(id string, enabled bool) error {
	if a.scheduleRepo == nil { return fmt.Errorf("schedule not available") }
	tasks, err := a.scheduleRepo.List()
	if err != nil { return err }
	for _, t := range tasks {
		if t != nil && t.ID == id {
			t.Enabled = enabled
			return a.scheduleRepo.Update(t)
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

func (a *App) GetCommodityQuotes() map[string]interface{} {
	return queryCommodityQuotes(a.marketReg)
}

// ServiceShutdown performs graceful cleanup: closes the Python sidecar connection,
// shared DB connection, and releases any resources held by the application.



// RunBacktest executes a backtest from a workflow JSON definition.
// Returns key metrics including total return, Sharpe, max drawdown, and win rate.
func (a *App) RunBacktest(jsonDef string) (map[string]interface{}, error) {
	_ = jsonDef // TODO: wire to backtest.Runner when API stabilizes
	return nil, fmt.Errorf("backtest engine available but RunBacktest not yet wired — see internal/backtest/runner.go")
}

func (a *App) ServiceShutdown() error {
	if a.sidecar != nil {
		a.sidecar.Stop()
	}
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
	return nil
}

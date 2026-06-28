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

	// Cache of last close prices for price-limit validation.
	lastClose map[string]float64

	// Shared DB connection (opened once at startup, reused across IPC calls).
	db *sql.DB

	// Phase 5
	oms          *trading.OMS
	notifyMgr    *notify.Manager
	scheduleRepo *schedule.Repo
	sched        *schedule.Scheduler
	portfolioSvc *portfolio.Service

	// Market data: adapter registry + fallback chains (wired in startup).
	marketReg     *market.AdapterRegistry
	newsAdpt      adapters.NewsAdapter // news source for sentiment analysis
	globalNewsAdpt *adapters.EastMoneyGlobalNewsAdapter
	conceptAdpt   *adapters.EastMoneyConceptAdapter
	macAdpt       *adapters.MacAdapter
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

	// Build shared NodeContext for workflow execution (replaces global setters).
	nctx := &workflow.NodeContext{}
	engine, err := workflow.NewEngine(a.registry, 256, nctx)
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
		nctx.Bridge = a.bridge
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

	// Set agent dependencies to NodeContext for workflow AgentNode
	if a.bridge != nil {
		nctx.CapRegistry = a.capRegistry
		nctx.Emitter = a.emitter
		nctx.ProfileMgr = a.profileMgr
	}

	// Wire ML bridge and model registry to workflow nodes.
	// ModelRegistry needs a DB connection (not yet shared at startup); pass nil
	// so that model persistence is gracefully skipped. The bridge is the critical
	// dependency — it enables gRPC communication with the Python sidecar.
	nctx.ModelRegistry = nil

	// Phase 5: Initialize trading OMS and wire to workflow nodes
	a.oms = trading.NewOMS()
	nctx.OMS = a.oms

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
	nctx.GlobalNewsAdapter = a.globalNewsAdpt
	a.conceptAdpt = adapters.NewEastMoneyConceptAdapter()
	a.signalsAdpt = adapters.NewEastMoneySignalsAdapter()
	a.capitalAdpt = adapters.NewEastMoneyCapitalAdapter()
	a.fundFlowAdpt = adapters.NewEastMoneyFundFlowAdapter()
	a.macAdpt = adapters.NewMacAdapter("")
	a.eastmoneyAdpt = adapters.NewEastMoneyAdapter()
	a.northboundAdpt = adapters.NewTHSNorthboundAdapter()
	a.sinaFinAdpt = adapters.NewSinaFinancialsAdapter()

	// Phase 3: Research report + announcement adapters
	a.reportAdpt = adapters.NewEastMoneyReportAdapter()
	a.consensusAdpt = adapters.NewTHSConsensusAdapter()
	a.cninfoAdpt = adapters.NewCninfoAdapter()
	a.iwencaiAdpt = adapters.NewIwencaiAdapter()
	a.congressAdpt = adapters.NewCongressTradesAdapter()

	nctx.NewsAdapter = a.newsAdpt
	if a.bridge != nil {
		sentimentEngine := research.NewSentimentEngine(a.bridge, researchRepo, a.newsAdpt)
		nctx.SentimentEngine = sentimentEngine
		slog.Info("sentiment engine initialized with Python bridge")
	} else {
		sentimentEngine := research.NewSentimentEngine(nil, researchRepo, a.newsAdpt)
		nctx.SentimentEngine = sentimentEngine
		slog.Info("sentiment engine initialized in mock mode (no Python bridge)")
	}
	nctx.FinancialsService = research.NewFinancialsService(a.sinaFinAdpt, a.getMootdxAdapter())
	nctx.PeerComparisonService = research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt, a.eastmoneyAdpt)
	nctx.AnalystEstimatesService = research.NewAnalystEstimatesService(a.reportAdpt, a.consensusAdpt)
	nctx.InsiderTradingService = research.NewInsiderTradingService(a.bridge)
	nctx.CongressTradingService = research.NewCongressTradingService(a.congressAdpt)
	slog.Info("research services initialized")

	// Symbol search service (in-memory A-share index)
	searchSvc, err := market.NewSymbolSearchService(context.Background(), a.db)
	if err != nil {
		slog.Warn("symbol search service init failed", "error", err)
	} else {
		a.searchSvc = searchSvc
		slog.Info("symbol search service initialized", "stocks", searchSvc.Size())
	}
	a.capitalSvc = research.NewCapitalService(a.capitalAdpt)
	nctx.CapitalService = a.capitalSvc
	a.fundFlowSvc = research.NewFundFlowService(a.fundFlowAdpt)
	nctx.FundFlowService = a.fundFlowSvc
	a.northboundSvc = research.NewNorthboundService(a.northboundAdpt)
	nctx.NorthboundService = a.northboundSvc
	a.announcementSvc = research.NewAnnouncementService(a.cninfoAdpt)
	nctx.AnnouncementService = a.announcementSvc

	// Alternative data: prediction market (Polymarket)
	a.polymarketAdpt = adapters.NewPolymarketAdapter()
	a.predictionMarketSvc = research.NewPredictionMarketService(a.polymarketAdpt)
	nctx.PredictionMarketService = a.predictionMarketSvc
	slog.Info("prediction market service initialized")

	// Alternative data: geopolitics (GDELT)
	a.geopoliticsAdpt = adapters.NewGDELTAdapter()
	a.geopoliticsSvc = research.NewGeopoliticsService(a.geopoliticsAdpt)
	nctx.GeopoliticsService = a.geopoliticsSvc
	slog.Info("geopolitics service initialized")

	// Alternative data: govdata (FRED + SEC EDGAR)
	a.govDataAdpt = adapters.NewGovDataAdapter()
	a.govDataSvc = research.NewGovDataService(a.govDataAdpt)
	// Inject FRED API key from config (with env var fallback in NewGovDataAdapter)
	if gha, ok := a.govDataAdpt.(*adapters.GovDataHTTPAdapter); ok {
		gha.SetAPIKey(a.cfg.GetAPIKey("fred"))
	}
	nctx.GovDataService = a.govDataSvc
	slog.Info("govdata service initialized")

	// Alternative data: satellite (NASA POWER + FIRMS)
	a.satelliteAdpt = adapters.NewSatelliteAdapter()
	a.satelliteSvc = research.NewSatelliteService(a.satelliteAdpt)
	nctx.SatelliteService = a.satelliteSvc
	slog.Info("satellite service initialized")
	return nil
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

// RunBacktestWorkflow executes a walk-forward backtest from a workflow JSON definition.
func (a *App) RunBacktestWorkflow(ctx context.Context, jsonDef string, cfg workflow.BacktestConfig) (*workflow.BacktestResult, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return a.engine.ExecuteBacktest(ctx, &wf, cfg)
}

// OptimizeWorkflow runs parameter optimization on a workflow and returns top results.
func (a *App) OptimizeWorkflow(ctx context.Context, jsonDef string, cfg workflow.OptimizeConfig) (*workflow.OptimizeResult, error) {
	if a.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return a.engine.OptimizeParams(ctx, &wf, cfg)
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

// SearchSymbols searches A-share stocks by code, name, or pinyin abbreviation.
// Returns up to limit matches sorted by relevance. limit defaults to 20 if <= 0.
// Returns empty slice (not error) when the search service is unavailable.
func (a *App) SearchSymbols(query string, limit int) ([]market.StockEntry, error) {
	if a.searchSvc == nil {
		return []market.StockEntry{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	return a.searchSvc.Search(query, limit), nil
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

// GetConceptBlocks returns the concept/industry/sector blocks a stock belongs to.
func (a *App) GetConceptBlocks(symbol string) ([]adapters.ConceptBlock, error) {
	if a.conceptAdpt == nil {
		return nil, fmt.Errorf("concept adapter not initialized")
	}
	return a.conceptAdpt.FetchConceptBlocks(context.Background(), symbol)
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

// GetCorrelationMatrix computes the Pearson correlation matrix for a set of symbols.
func (a *App) GetCorrelationMatrix(ctx context.Context, symbols []string, lookback int) (map[string]map[string]float64, error) {
	reg := a.getMarketReg()
	returns := make(map[string][]float64)
	end := time.Now().Unix()
	start := end - int64(lookback*86400)
	for _, sym := range symbols {
		mkt := market.MarketForSymbol(sym)
		bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, sym, "1d", "", start, end)
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
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", "", start, end)
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
	bars, _, err := reg.FetchOHLCVWithFallback(ctx, mkt, symbol, "1d", "", start, end)
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
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	ctx := context.Background()

	if symbol == "" && a.globalNewsAdpt != nil {
		articles, err := a.globalNewsAdpt.FetchGlobalNews(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("global news: %w", err)
		}
		items := make([]NewsItem, 0, len(articles))
		for _, art := range articles {
			items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL})
		}
		return items, nil
	}

	if a.newsAdpt == nil {
		return nil, fmt.Errorf("news adapter not available")
	}
	articles, err := a.newsAdpt.FetchStockNews(ctx, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("stock news: %w", err)
	}
	items := make([]NewsItem, 0, len(articles))
	for _, art := range articles {
		items = append(items, NewsItem{Title: art.Title, Source: art.Source, Time: art.Time, URL: art.URL, Symbol: art.Symbol})
	}
	return items, nil
}

// ListScheduleTasks returns all scheduled tasks.
func (a *App) ListScheduleTasks() ([]schedule.Task, error) {
	if a.scheduleRepo == nil {
		return nil, fmt.Errorf("schedule not available")
	}
	tasks, err := a.scheduleRepo.List()
	if err != nil {
		return nil, err
	}
	result := make([]schedule.Task, 0, len(tasks))
	for _, t := range tasks {
		if t != nil {
			result = append(result, *t)
		}
	}
	return result, nil
}

// SaveScheduleTask creates or updates a scheduled task.
func (a *App) SaveScheduleTask(task schedule.Task) error {
	if a.scheduleRepo == nil {
		return fmt.Errorf("schedule not available")
	}
	if task.ID == "" {
		return a.scheduleRepo.Create(&task)
	}
	return a.scheduleRepo.Update(&task)
}

// DeleteScheduleTask deletes a scheduled task by ID.
func (a *App) DeleteScheduleTask(id string) error {
	if a.scheduleRepo == nil {
		return fmt.Errorf("schedule not available")
	}
	return a.scheduleRepo.Delete(id)
}

// ToggleScheduleTask enables or disables a scheduled task.
func (a *App) ToggleScheduleTask(id string, enabled bool) error {
	if a.scheduleRepo == nil {
		return fmt.Errorf("schedule not available")
	}
	tasks, err := a.scheduleRepo.List()
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t != nil && t.ID == id {
			t.Enabled = enabled
			return a.scheduleRepo.Update(t)
		}
	}
	return fmt.Errorf("task not found: %s", id)
}

// GetCommodityQuotes returns real-time commodity prices.
func (a *App) GetCommodityQuotes() map[string]interface{} {
	return queryCommodityQuotes(a.marketReg)
}

// ServiceShutdown performs graceful cleanup: closes the Python sidecar connection,
// shared DB connection, and releases any resources held by the application.
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

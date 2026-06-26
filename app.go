package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
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
	signalsAdpt   *adapters.EastMoneySignalsAdapter
	capitalAdpt   *adapters.EastMoneyCapitalAdapter
	fundFlowAdpt  *adapters.EastMoneyFundFlowAdapter
	sinaFinAdpt   *adapters.SinaFinancialsAdapter
	reportAdpt    *adapters.EastMoneyReportAdapter
	consensusAdpt *adapters.THSConsensusAdapter
	thsHotAdpt    *adapters.THSHotAdapter
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

// registerMarketAdapters populates the adapter registry with every data source,
// in the order the fallback chains expect (see market.FallbackChains). mootdx
// alone needs the Python sidecar; it degrades to IsAvailable()==false when the
// bridge is absent.
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

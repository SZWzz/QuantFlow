package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/ai"
	"quantflow/internal/ai/capabilities"
	"quantflow/internal/auth"
	"quantflow/internal/backtest"
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
	"quantflow/internal/ws"
)

var startTime = time.Now() // used by GetSystemStats for uptime
var execQueue *workflow.ExecutionQueue

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
	execRepo     *storage.ExecutionRepo
	btRepo       *storage.BacktestRepo
	credMgr      *auth.CredentialManager

	// FetchData TTL cache (macro summaries, akshare endpoints, etc.).
	dataCache     *fetchDataCache

	// Market data: adapter registry + fallback chains (wired in startup).
	marketReg     *market.AdapterRegistry

	// Market data hub for in-process pub/sub.
	marketHub     *market.MarketDataHub

	// WebSocket service wrapper (set during ServiceStartup, registered in main.go).
	wsSvc         *ws.MarketWSService

	// QuotePoller for periodic quote fetch + WebSocket broadcast.
	quotePoller   *market.QuotePoller
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
	bridgeOpts.PythonDir = pythonDir
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
	nctx.MarketReg = a.marketReg

	// Load persisted last quotes (weekend/off-hours display).
	lastQuotePath := filepath.Join(filepath.Dir(a.cfg.DBPath), "last_quote.json")
	a.marketReg.SetLastQuotePath(lastQuotePath)
	if err := a.marketReg.LoadLastQuotes(); err != nil {
		slog.Warn("load last quotes", "error", err)
	}

	// FetchData cache: prevents redundant Python sidecar calls for slow
	// AKShare endpoints (macro_cn_summary takes 60-90s).
	a.dataCache = newFetchDataCache()

	// Initialize MarketDataHub for real-time pub/sub (audit fix M7).
	// Currently a stub — full topic broker activation pending per-symbol subscription UI.
	a.marketHub = market.NewHub()
	slog.Info("market data hub initialized")

	// Create and start the WebSocket hub for real-time market data push.
	wsHub := ws.NewHub()
	go wsHub.Run()

	// Wire the WebSocket hub into the MarketWSService (registered in main.go)
	// so the /ws/market endpoint can upgrade connections.
	if a.wsSvc != nil {
		a.wsSvc.Hub = wsHub
	}

	// Start the QuotePoller: subscribes/unsubscribes based on frontend WS topics,
	// periodically fetches quotes and broadcasts via wsHub.
	a.quotePoller = market.NewQuotePoller(a.marketReg, a.marketHub, wsHub)
	go a.quotePoller.Run(ctx)
	slog.Info("quote poller started on ws hub")

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

	// Execution history persistence
	a.execRepo = storage.NewExecutionRepo(a.db)
	slog.Info("execution history repo initialized")

	a.btRepo = storage.NewBacktestRepo(a.db)
	slog.Info("backtest repo initialized")

	// Credential manager
	credMgr, err := auth.NewCredentialManager(a.db)
	if err != nil {
		slog.Warn("credential manager init failed", "error", err)
	} else {
		a.credMgr = credMgr
		slog.Info("credential manager initialized")
	}

	// Initialize async execution queue
	execQueue = workflow.NewExecutionQueue(a.engine)

	// Wire execution saver so every workflow run is recorded
	a.engine.SetExecutionSaver(func(runID string, wf *workflow.Workflow, result *workflow.ExecutionResult) {
		wfJSON, _ := json.Marshal(wf)
		nodeResults, _ := json.Marshal(result.NodeResults)
		if err := a.execRepo.Save(runID, wf.ID, wf.Name, string(wfJSON), len(wf.Nodes), nodeResults, result.StartedAt, "manual"); err != nil {
			slog.Warn("failed to save execution start", "run_id", runID, "error", err)
			return
		}
		status := "completed"
		if result.Error != "" {
			status = "failed"
		}
		nodeResults, _ = json.Marshal(result.NodeResults)
		if err := a.execRepo.Complete(runID, status, nodeResults, result.FinishedAt, result.Error); err != nil {
			slog.Warn("failed to complete execution record", "run_id", runID, "error", err)
		}

		persistBacktestResults(a, runID, wf, result)
	})

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
	a.marketReg.Register(a.signalsAdpt) // register for industry rank fallback chain
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
	nctx.PeerComparisonService = research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt, a.eastmoneyAdpt, a.marketReg)
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

	// Wire sub-workflow runner: allows sub_workflow nodes to execute saved workflows
	nctx.SubWorkflowRunner = func(ctx context.Context, wfID string, inputs map[string]any) (map[string]any, error) {
		repo := storage.NewWorkflowRepo(a.db)
		wf, err := repo.Load(wfID, nil)
		if err != nil {
			return nil, fmt.Errorf("sub_workflow: load %q: %w", wfID, err)
		}
		// Inject inputs into the child workflow's first node params
		if len(wf.Nodes) > 0 && len(inputs) > 0 {
			for k, v := range inputs {
				wf.Nodes[0].Params[k] = v
			}
		}
		result, err := a.engine.Execute(ctx, wf)
		if err != nil {
			return nil, fmt.Errorf("sub_workflow: execute %q: %w", wfID, err)
		}
		// Collect outputs from all completed nodes
		outputs := make(map[string]any)
		for _, nr := range result.NodeResults {
			if nr.Status == "completed" {
				for k, v := range nr.Outputs {
					outputs[nr.NodeID+"."+k] = v
				}
			}
		}
		return outputs, nil
	}
	slog.Info("sub-workflow runner wired")

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
	if err := workflow.ValidateEdgeTypes(&wf, a.registry); err != nil {
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

// QueueWorkflow enqueues a workflow for asynchronous execution.
// Returns immediately with a runID. Frontend polls GetExecutionStatus for progress.
func (a *App) QueueWorkflow(jsonDef string) (string, error) {
	if a.engine == nil {
		return "", fmt.Errorf("engine not initialized")
	}
	var wf workflow.Workflow
	if err := json.Unmarshal([]byte(jsonDef), &wf); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	return execQueue.Enqueue(&wf)
}

// GetExecutionStatus returns the current state of a queued/running/completed workflow.
func (a *App) GetExecutionStatus(runID string) (*workflow.QueuedWorkflow, error) {
	status := execQueue.GetStatus(runID)
	if status == nil {
		return nil, fmt.Errorf("run %q not found", runID)
	}
	return status, nil
}

// CancelExecution cancels a queued workflow execution.
func (a *App) CancelExecution(runID string) error {
	execQueue.Cancel(runID)
	return nil
}

// GetExecutionHistory returns recent workflow execution records.
func (a *App) GetExecutionHistory(limit int) ([]storage.ExecutionRecord, error) {
	if a.execRepo == nil {
		return nil, fmt.Errorf("execution history not initialized")
	}
	return a.execRepo.List(limit)
}

// GetExecution returns a single execution record by run ID.
func (a *App) GetExecution(runID string) (*storage.ExecutionRecord, error) {
	if a.execRepo == nil {
		return nil, fmt.Errorf("execution history not initialized")
	}
	return a.execRepo.Get(runID)
}

// ListBacktestHistory returns recent backtest results with pagination.
func (a *App) ListBacktestHistory(ctx context.Context, limit, offset int) ([]storage.StoredBacktestSummary, error) {
	if a.btRepo == nil {
		return nil, fmt.Errorf("backtest repo not initialized")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return a.btRepo.List(ctx, limit, offset)
}

// GetStoredBacktestResult returns a single backtest result by ID with full JSON blobs.
func (a *App) GetStoredBacktestResult(ctx context.Context, id int) (*storage.StoredBacktest, error) {
	if a.btRepo == nil {
		return nil, fmt.Errorf("backtest repo not initialized")
	}
	return a.btRepo.GetByID(ctx, id)
}

// GetStoredBacktestByRunID returns the summary for a stored backtest by run ID.
// Returns nil (not error) if not found.
func (a *App) GetStoredBacktestByRunID(ctx context.Context, runID string) (*storage.StoredBacktestSummary, error) {
	if a.btRepo == nil {
		return nil, nil
	}
	return a.btRepo.GetByRunID(ctx, runID)
}

// DeleteBacktestResult deletes a backtest result by ID. Returns true on success.
func (a *App) DeleteBacktestResult(ctx context.Context, id int) (bool, error) {
	if a.btRepo == nil {
		return false, fmt.Errorf("backtest repo not initialized")
	}
	if err := a.btRepo.Delete(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

// ClearBacktestResults deletes ALL backtest results from the database. Returns the count of deleted records.
func (a *App) ClearBacktestResults(ctx context.Context) (int64, error) {
	if a.btRepo == nil {
		return 0, fmt.Errorf("backtest repo not initialized")
	}
	return a.btRepo.ClearAll(ctx)
}

// ── Credential Management ───────────────────────────────────────────

// ListCredentials returns all stored credentials with decrypted keys.
func (a *App) ListCredentials() ([]auth.Credential, error) {
	if a.credMgr == nil {
		return nil, fmt.Errorf("credential manager not initialized")
	}
	return a.credMgr.List()
}

// SaveCredential creates or updates a credential with encrypted storage.
func (a *App) SaveCredential(name, credType string, keys map[string]string) error {
	if a.credMgr == nil {
		return fmt.Errorf("credential manager not initialized")
	}
	return a.credMgr.Save(name, credType, keys)
}

// DeleteCredential removes a credential by name.
func (a *App) DeleteCredential(name string) error {
	if a.credMgr == nil {
		return fmt.Errorf("credential manager not initialized")
	}
	return a.credMgr.Delete(name)
}

// ListCredentialNames returns credential names for dropdown use.
func (a *App) ListCredentialNames() ([]string, error) {
	if a.credMgr == nil {
		return []string{}, nil
	}
	return a.credMgr.Names()
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

// ListLLMModels returns available LLM models from the Python sidecar.
func (a *App) ListLLMModels() ([]map[string]interface{}, error) {
	if a.bridge == nil {
		return nil, fmt.Errorf("Python sidecar not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := a.bridge.ListModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	models := make([]map[string]interface{}, 0, len(resp))
	for _, m := range resp {
		models = append(models, map[string]interface{}{
			"id":              m.Id,
			"provider":        m.Provider,
			"display_name":    m.DisplayName,
			"context_window":  m.ContextWindow,
			"supports_tools":  m.SupportsTools,
			"supports_vision": m.SupportsVision,
		})
	}
	return models, nil
}

// TestLLMConnection verifies connectivity to an LLM provider by sending a
// minimal request (list models) to the provider's API endpoint.
func (a *App) TestLLMConnection(provider, apiKey, baseUrl string) (map[string]interface{}, error) {
	if apiKey == "" {
		return map[string]interface{}{"ok": false, "error": "API key is required"}, nil
	}
	if baseUrl == "" {
		return map[string]interface{}{"ok": false, "error": "base URL is required"}, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	raw := strings.TrimRight(baseUrl, "/")

	var url string
	switch provider {
	case "google":
		url = strings.TrimSuffix(raw, "/v1beta") + "/v1beta/models"
	case "ollama":
		url = strings.TrimSuffix(raw, "/v1") + "/api/tags"
	default:
		if strings.HasSuffix(raw, "/v1") {
			url = raw + "/models"
		} else {
			url = raw + "/v1/models"
		}
	}

	start := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": fmt.Sprintf("create request: %v", err)}, nil
	}
	if provider != "ollama" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": fmt.Sprintf("http request: %v", err)}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	latencyMs := time.Since(start).Milliseconds()

	if resp.StatusCode != 200 {
		return map[string]interface{}{"ok": false, "error": fmt.Sprintf("API error (%d): %.200s", resp.StatusCode, string(body))}, nil
	}

	return map[string]interface{}{"ok": true, "latencyMs": latencyMs}, nil
}

// ListProviderModels fetches available models from a provider's /v1/models API.
// This calls the provider directly (OpenAI-compatible) rather than going through
// the Python sidecar, so it works even when the sidecar is not connected.
func (a *App) ListProviderModels(provider, apiKey, baseUrl string) ([]map[string]interface{}, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if baseUrl == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	raw := strings.TrimRight(baseUrl, "/")

	// Some providers already have /v1 at the end of base URL; avoid double /v1
	var url string
	switch provider {
	case "google":
		url = strings.TrimSuffix(raw, "/v1beta") + "/v1beta/models"
	case "ollama":
		url = strings.TrimSuffix(raw, "/v1") + "/api/tags"
	default:
		if strings.HasSuffix(raw, "/v1") {
			url = raw + "/models"
		} else {
			url = raw + "/v1/models"
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if provider != "ollama" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Google Gemini uses a different response format
	if provider == "google" {
		var googleResult struct {
			Models []struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &googleResult); err != nil {
			return nil, fmt.Errorf("parse google response: %w", err)
		}
		models := make([]map[string]interface{}, 0, len(googleResult.Models))
		for _, m := range googleResult.Models {
			models = append(models, map[string]interface{}{
				"id":           provider + "/" + m.Name,
				"provider":     provider,
				"display_name": m.DisplayName,
			})
		}
		return models, nil
	}

	// Ollama uses /api/tags with a different format
	if provider == "ollama" {
		var ollamaResult struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &ollamaResult); err != nil {
			return nil, fmt.Errorf("parse ollama response: %w", err)
		}
		models := make([]map[string]interface{}, 0, len(ollamaResult.Models))
		for _, m := range ollamaResult.Models {
			models = append(models, map[string]interface{}{
				"id":       provider + "/" + m.Name,
				"provider": provider,
			})
		}
		return models, nil
	}

	// OpenAI-compatible response format
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]map[string]interface{}, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, map[string]interface{}{
			"id":       provider + "/" + m.ID,
			"provider": provider,
		})
	}
	return models, nil
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

// GetIndustryRanks returns industry ranking by change percent for a given market.
// Uses per-market fallback chains: CN→eastmoney_signals, HK→tencent, US→finnhub.
func (a *App) GetIndustryRanks(mkt string, topN int) ([]market.IndustryRank, error) {
	if topN <= 0 {
		topN = 20
	}
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	return reg.FetchIndustryRanksWithFallback(context.Background(), mkt, topN)
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

// persistBacktestResults checks all node results for successful backtest nodes
// and persists their outputs to the backtest_results table.
func persistBacktestResults(a *App, runID string, wf *workflow.Workflow, result *workflow.ExecutionResult) {
	if a.btRepo == nil {
		return
	}
	for _, nr := range result.NodeResults {
		if nr.NodeType != "backtest" || nr.Status != "completed" {
			continue
		}
		outputs := nr.Outputs
		if outputs == nil {
			continue
		}

		rawResult, ok := outputs["result"]
		if !ok {
			continue
		}

		jsonBytes, err := json.Marshal(rawResult)
		if err != nil {
			slog.Warn("persistBacktest: marshal result", "node", nr.NodeID, "error", err)
			continue
		}
		var btResult backtest.Result
		if err := json.Unmarshal(jsonBytes, &btResult); err != nil {
			slog.Warn("persistBacktest: unmarshal result", "node", nr.NodeID, "error", err)
			continue
		}

		ohlcvJSON := "[]"
		ohlcv := findUpstreamOhlcv(wf, nr.NodeID, result.NodeResults)
		if ohlcv != nil {
			if b, err := json.Marshal(ohlcv); err == nil {
				ohlcvJSON = string(b)
			}
		}

		configJSON, _ := json.Marshal(btResult.Config)
		equityJSON, _ := json.Marshal(btResult.EquityCurve)
		tradesJSON, _ := json.Marshal(btResult.Trades)

		bt := storage.StoredBacktest{
			RunID:         runID,
			WorkflowName:  wf.Name,
			StrategyName:  extractStr(outputs, "strategy_name"),
			Symbol:        extractStr(outputs, "symbol"),
			EngineType:    extractStr(outputs, "engine_type"),
			TotalReturn:   btResult.Metrics.TotalReturn,
			CAGR:          btResult.Metrics.CAGR,
			MaxDrawdown:   btResult.Metrics.MaxDrawdown,
			SharpeRatio:   btResult.Metrics.SharpeRatio,
			SortinoRatio:  btResult.Metrics.SortinoRatio,
			CalmarRatio:   btResult.Metrics.CalmarRatio,
			WinRate:       btResult.Metrics.WinRate,
			ProfitFactor:  btResult.Metrics.ProfitFactor,
			TotalTrades:   btResult.Metrics.TotalTrades,
			ConfigJSON:    string(configJSON),
			EquityCurve:   string(equityJSON),
			TradesJSON:    string(tradesJSON),
			OHLCVData:     ohlcvJSON,
			BacktestStart: extractStr(outputs, "backtest_start"),
			BacktestEnd:   extractStr(outputs, "backtest_end"),
			StartedAt:     result.StartedAt.Format(time.RFC3339),
			FinishedAt:    result.FinishedAt.Format(time.RFC3339),
		}

		if _, err := a.btRepo.Save(context.Background(), bt); err != nil {
			slog.Error("persistBacktest: save failed", "node", nr.NodeID, "error", err)
		}
	}
}

func findUpstreamOhlcv(wf *workflow.Workflow, backtestNodeID string, nodeResults []workflow.NodeResult) any {
	for _, edge := range wf.Edges {
		if edge.ToNode == backtestNodeID && edge.ToPort == "ohlcv_data" {
			for _, nr := range nodeResults {
				if nr.NodeID == edge.FromNode {
					if ohlcv, ok := nr.Outputs["ohlcv"]; ok {
						return ohlcv
					}
				}
			}
		}
	}
	return nil
}

func extractStr(outputs map[string]any, key string) string {
	if v, ok := outputs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

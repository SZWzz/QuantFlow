package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/ai"
	"quantflow/internal/ai/capabilities"
	"quantflow/internal/auth"
	"quantflow/internal/config"
	"quantflow/internal/crash"
	"quantflow/internal/logging"
	"quantflow/internal/market"
	"quantflow/internal/market/adapters"
	"quantflow/internal/notify"
	"quantflow/internal/portfolio"
	"quantflow/internal/research"
	"quantflow/internal/python"
	"quantflow/internal/schedule"
	"quantflow/internal/storage"
	"quantflow/internal/trading"
	"quantflow/internal/trading/brokers"
	"quantflow/internal/workflow"
	"quantflow/internal/workflow/nodes"
	"quantflow/internal/ws"
)

// ServiceStartup is called by Wails v3 when the application starts.
// It initializes all services: config, DB, market adapters, research, etc.
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	configPath := "config.yaml"
	if execPath, err := os.Executable(); err == nil {
		configPath = filepath.Join(filepath.Dir(execPath), "config.yaml")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	a.cfg = cfg

	// Wire crash reporter with app version and state getters.
	// State getters are closures evaluated at crash time, so they safely capture
	// app state that may not be fully initialized yet at this point.
	crash.SetAppInfo(cfg.Version, "prod")
	crash.SetStateGetters(
		func() int64 { return int64(time.Since(startTime).Seconds()) },
		func() int { return 0 }, // panel count — not tracked in the backend
		func() int {
			// Approximate active workflow count. An exact count would require
			// querying storage.WorkflowRepo.List() which allocates a new repo each call.
			return 0
		},
		func() (names []string) {
			// a.brokers may still be populated by ServiceStartup when a crash
			// fires; recover so a concurrent map iteration panic can never
			// escape from inside the crash handler itself.
			defer func() { _ = recover() }()
			names = make([]string, 0, len(a.brokers))
			for name := range a.brokers {
				names = append(names, name)
			}
			return names
		},
		func() string {
			return "unknown"
		},
	)

	// Wire recent log lines into crash reports (spec: last 100 lines).
	// Evaluated at crash time inside the crash handler, so it must never panic.
	crash.SetLogsGetter(func() (lines []string) {
		defer func() { _ = recover() }()
		entries := logging.Ring.LastN(100)
		lines = make([]string, 0, len(entries))
		for _, e := range entries {
			lines = append(lines, fmt.Sprintf("%s %s %s", e.Time.Format(time.RFC3339), e.Level, e.Message))
		}
		return lines
	})

	// If db_path was persisted as absolute from a previous version, reset to default
	// relative path so it resolves correctly against the current executable location.
	// This handles config.yaml left behind by the old code that overwrote cfg.DBPath
	// with the ResolveDBPath result before saving.
	if filepath.IsAbs(a.cfg.DBPath) {
		slog.Warn("config db_path is absolute, resetting to default relative (data/quantflow.db)", "current", a.cfg.DBPath)
		a.cfg.DBPath = "data/quantflow.db"
	}
	// Resolve relative DB path against executable directory so dev/build/.app all share the same DB.
	// Store in resolvedDBPath — NOT in a.cfg.DBPath — so Save() never persists an absolute path.
	a.resolvedDBPath = config.ResolveDBPath(a.cfg.DBPath)
	logging.Setup(cfg.LogLevel)

	// Open shared DB connection and run migrations once at startup.
	db, err := storage.Open(a.resolvedDBPath)
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

	oc, err := market.NewOHLCVCache(a.db)
	if err != nil {
		slog.Error("failed to init ohlcv cache", "err", err)
	} else {
		a.ohlcvCache = oc
	}

	migrations, migErr := storage.BuiltinMigrations()
	if migErr == nil {
		if err := storage.Run(db, migrations); err != nil {
			return fmt.Errorf("database migrations failed: %w", err)
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
	a.sectorSvc = market.NewSectorService(a.marketReg, a.db)
	nctx.MarketReg = a.marketReg

	if a.ohlcvCache != nil {
		a.marketReg.SetOHLCVCache(a.ohlcvCache)
	}

	// Load persisted last quotes (weekend/off-hours display).
	lastQuotePath := filepath.Join(filepath.Dir(a.resolvedDBPath), "last_quote.json")
	a.marketReg.SetLastQuotePath(lastQuotePath)
	if err := a.marketReg.LoadLastQuotes(); err != nil {
		slog.Warn("load last quotes", "error", err)
	}

	// Initialize off-hours data caches (persisted JSON in data/offhours/).
	offDir := filepath.Join(filepath.Dir(a.resolvedDBPath), "offhours")

	a.industryRanksCache = market.NewOffHoursCache[[]market.IndustryRank]("industry_ranks")
	a.industryRanksCache.SetPath(filepath.Join(offDir, "industry_ranks.json"))
	if err := a.industryRanksCache.Load(); err != nil {
		slog.Warn("load industry ranks cache", "error", err)
	}

	a.depthCache = market.NewOffHoursCache[*market.DepthSnapshot]("depth")
	a.depthCache.SetPath(filepath.Join(offDir, "depth.json"))
	if err := a.depthCache.Load(); err != nil {
		slog.Warn("load depth cache", "error", err)
	}

	a.abnormalStocksCache = market.NewOffHoursCache[[]adapters.AbnormalStock]("abnormal_stocks")
	a.abnormalStocksCache.SetPath(filepath.Join(offDir, "abnormal_stocks.json"))
	if err := a.abnormalStocksCache.Load(); err != nil {
		slog.Warn("load abnormal stocks cache", "error", err)
	}

	a.dragonTigerCache = market.NewOffHoursCache[[]adapters.DragonTigerRecord]("dragon_tiger")
	a.dragonTigerCache.SetPath(filepath.Join(offDir, "dragon_tiger.json"))
	if err := a.dragonTigerCache.Load(); err != nil {
		slog.Warn("load dragon tiger cache", "error", err)
	}

	a.fundFlowCache = market.NewOffHoursCache[interface{}]("fund_flow")
	a.fundFlowCache.SetPath(filepath.Join(offDir, "fund_flow.json"))
	if err := a.fundFlowCache.Load(); err != nil {
		slog.Warn("load fund flow cache", "error", err)
	}

	// FetchData cache: prevents redundant Python sidecar calls for slow
	// AKShare endpoints (macro_cn_summary takes 60-90s).
	a.dataCache = newFetchDataCache()

	// Initialize MarketDataHub for real-time pub/sub (audit fix M7).
	// Currently a stub — full topic broker activation pending per-symbol subscription UI.
	a.marketHub = market.NewHub()
	slog.Info("market data hub initialized")

	// Create and start the WebSocket hub for real-time market data push.
	a.wsHub = ws.NewHub()
	go a.wsHub.Run()

	// Wire the WebSocket hub into the MarketWSService (registered in main.go)
	// so the /ws/market endpoint can upgrade connections.
	if a.wsSvc != nil {
		a.wsSvc.Hub = a.wsHub
	}

	// Start the QuotePoller: subscribes/unsubscribes based on frontend WS topics,
	// periodically fetches quotes and broadcasts via wsHub.
	a.quotePoller = market.NewQuotePoller(a.marketReg, a.marketHub, a.wsHub)
	go a.quotePoller.Run(ctx)
	slog.Info("quote poller started on ws hub")

	// Start the MinutePoller: fetches minute ticks for subscribed symbols
	// and pushes deltas via wsHub under market:minute:* topics.
	a.minutePoller = market.NewMinutePoller(a.wsHub, func(symbol string, since int64) ([]market.MinuteTick, error) {
		reqCtx, cancel := market.RequestCtx()
		defer cancel()
		ticks, _, err := a.GetMinuteLine(reqCtx, symbol, since)
		return ticks, err
	})
	go a.minutePoller.Run(ctx)
	slog.Info("minute poller started on ws hub")

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
	if a.bridge != nil {
		nctx.MLClient = python.NewMLClient(a.bridge)
	}

	// Phase 5: Initialize trading OMS and wire to workflow nodes
	a.oms = trading.NewOMS()
	a.tradingMode = trading.NewEngineMode(a.oms)
	a.brokers = make(map[string]trading.Broker)
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
		a.brokers["alpaca"] = alpacaBroker
	}

	// IBKR broker adapter (optional) — configured via IBKR_HOST, IBKR_PORT, IBKR_ACCOUNT_ID env vars.
	ibkrHost := os.Getenv("IBKR_HOST")
	if ibkrHost == "" {
		ibkrHost = "localhost"
	}
	ibkrCfg := brokers.IBKRConfig{
		Host:      ibkrHost,
		Port:      5000,
		AccountID: os.Getenv("IBKR_ACCOUNT_ID"),
	}
	if ibkrCfg.AccountID != "" {
		ibkrBroker := brokers.NewIBKRBroker(ibkrCfg)
		if err := ibkrBroker.Connect(context.Background()); err != nil {
			slog.Warn("ibkr broker not available — IBKR trading disabled", "error", err)
		} else {
			a.oms.SetBroker(ibkrBroker)
			slog.Info("ibkr broker connected — IBKR trading enabled")
		}
		a.brokers["ibkr"] = ibkrBroker
	}

	// Initialize Futu broker adapter (optional).
	futuCfg := brokers.FutuConfig{}
	if fpStr := os.Getenv("FUTU_PORT"); fpStr != "" {
		if p, err := strconv.Atoi(fpStr); err == nil {
			futuCfg.Port = p
		}
	}
	futuBroker := brokers.NewFutuBroker(futuCfg)
	if err := futuBroker.Connect(context.Background()); err != nil {
		slog.Warn("futu broker not available — A/HK trading via Futu disabled", "error", err)
	} else {
		a.oms.SetBroker(futuBroker)
		slog.Info("futu broker connected — A/HK trading enabled")
	}
	a.brokers["futu"] = futuBroker

	// Initialize Binance broker adapter (optional).
	binanceCfg := brokers.BinanceConfig{
		APIKey:    os.Getenv("BINANCE_API_KEY"),
		SecretKey: os.Getenv("BINANCE_SECRET_KEY"),
	}
	if binanceCfg.APIKey != "" && binanceCfg.SecretKey != "" {
		binanceBroker := brokers.NewBinanceBroker(binanceCfg)
		if err := binanceBroker.Connect(context.Background()); err != nil {
			slog.Warn("binance broker not available — crypto trading disabled", "error", err)
		} else {
			a.oms.SetBroker(binanceBroker)
			slog.Info("binance broker connected — crypto trading enabled")
		}
		a.brokers["binance"] = binanceBroker
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

	// Migrate config api_keys to CredentialManager.
	// This is a one-time migration for existing config files.
	// After migration, the api_keys map is emptied so no plaintext keys
	// remain in the process memory from the YAML file.
	// New installations should use CredentialManager directly.
	if a.credMgr != nil {
		for name, key := range a.cfg.APIKeys {
			if key != "" {
				if err := a.credMgr.Save(name+"_api_key", "api_key", map[string]string{"api_key": key}); err != nil {
					slog.Warn("migrate config api_key to credential manager", "name", name, "error", err)
				} else {
					delete(a.cfg.APIKeys, name)
					slog.Info("migrated config api_key to credential manager", "name", name)
				}
			}
		}
		if a.cfg.APIKeys == nil {
			a.cfg.APIKeys = map[string]string{}
		}
		a.cfg.SetCredentialManager(a.credMgr)
	}

	// Initialize async execution queue
	execQueue = workflow.NewExecutionQueue(a.engine)

	// Wire execution saver so every workflow run is recorded
	a.engine.SetExecutionSaver(func(runID string, wf *workflow.Workflow, result *workflow.ExecutionResult) {
		wfJSON, err := json.Marshal(wf)
		if err != nil {
			slog.Warn("marshal workflow failed", "error", err)
			return
		}
		nodeResults, err := json.Marshal(result.NodeResults)
		if err != nil {
			slog.Warn("marshal node results failed", "error", err)
			return
		}
		if err := a.execRepo.Save(runID, wf.ID, wf.Name, string(wfJSON), len(wf.Nodes), nodeResults, result.StartedAt, "manual"); err != nil {
			slog.Warn("failed to save execution start", "run_id", runID, "error", err)
			return
		}
		status := "completed"
		if result.Error != "" {
			status = "failed"
		}
		nodeResults, err = json.Marshal(result.NodeResults)
		if err != nil {
			slog.Warn("marshal node results failed", "error", err)
			return
		}
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
	a.finSvc = research.NewFinancialsService(a.sinaFinAdpt, a.getMootdxAdapter())
	nctx.FinancialsService = a.finSvc
	a.peerSvc = research.NewPeerComparisonService(a.conceptAdpt, a.signalsAdpt, a.eastmoneyAdpt, a.marketReg)
	nctx.PeerComparisonService = a.peerSvc
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

	// Initialize tear-off window tracking map
	a.tearOffWindows = make(map[string]*tearOffEntry)
	slog.Info("tear-off window tracking initialized")
	return nil
}

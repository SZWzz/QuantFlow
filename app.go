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
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"quantflow/internal/ai"
	"quantflow/internal/auth"
	"quantflow/internal/backtest"
	"quantflow/internal/config"
	"quantflow/internal/crash"
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
	"quantflow/internal/workflow"
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
	ohlcvCache  *market.OHLCVCache

	// Cache of last close prices for price-limit validation.
	lastClose map[string]float64

	// Shared DB connection (opened once at startup, reused across IPC calls).
	db *sql.DB

	// Phase 5
	oms          *trading.OMS
	tradingMode  *trading.EngineMode      // paper/live mode manager
	brokers      map[string]trading.Broker // all registered live brokers by name
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

	// Sector service for industry dashboard.
	sectorSvc     *market.SectorService

	// Research services (exposed for analysis panels)
	finSvc       *research.FinancialsService
	peerSvc      *research.PeerComparisonService
	macroSvc     *market.MacroService
	styleSvc     *market.StyleService
	eventStudySvc *research.EventStudyService

	// WebSocket service wrapper (set during ServiceStartup, registered in main.go).
	wsSvc         *ws.MarketWSService
	wsHub         *ws.Hub

	// QuotePoller for periodic quote fetch + WebSocket broadcast.
	quotePoller   *market.QuotePoller
	// MinutePoller for real-time minute tick push via WebSocket.
	minutePoller  *market.MinutePoller
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

	// Resolved absolute DB path (kept separate from cfg.DBPath so Save()
	// never persists an absolute path that would break after moving the project).
	resolvedDBPath string

	// Off-hours data caches for weekend/after-hours display.
	industryRanksCache  *market.OffHoursCache[[]market.IndustryRank]
	depthCache          *market.OffHoursCache[*market.DepthSnapshot]
	abnormalStocksCache *market.OffHoursCache[[]adapters.AbnormalStock]
	dragonTigerCache    *market.OffHoursCache[[]adapters.DragonTigerRecord]
	fundFlowCache       *market.OffHoursCache[interface{}]

	// Python sidecar subprocess (auto-launched).
	sidecar *python.SidecarProcess

	// Wails application reference (set in main.go).
	wailsApp *application.App

	// Crash report store (initialized in main.go, used by ServiceShutdown).
	crashStore *crash.Store

	// Tear-off window tracking.
	tearOffWindows   map[string]*tearOffEntry // instanceId → entry
	tearOffWindowsMu sync.RWMutex
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

// GetCredential returns a single credential by name.
func (a *App) GetCredential(name string) (*auth.Credential, error) {
	if a.credMgr == nil {
		return nil, fmt.Errorf("credential manager not initialized")
	}
	creds, err := a.credMgr.List()
	if err != nil {
		return nil, err
	}
	for _, c := range creds {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("credential %q not found", name)
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

// GetLogs returns log entries after the given ID (0 = all).
// Used by the frontend LogPanel to poll for new entries.
func (a *App) GetLogs(afterID int) []logging.LogEntry {
	return logging.Ring.Lines(int64(afterID), 200)
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
	ictx, icancel := market.RequestCtx()
	defer icancel()
	if !a.iwencaiAdpt.IsAvailable(ictx) {
		return nil, fmt.Errorf("iwencai not available: IWENCAI_API_KEY not set or endpoint unreachable")
	}
	return a.iwencaiAdpt.Search(ictx, query, channel, size)
}

// GetCapitalData returns capital/fundamental data for a symbol: margin trading,
// block trades, holder changes, and dividend history.
func (a *App) GetCapitalData(symbol string) (map[string]interface{}, error) {
	if a.capitalSvc == nil {
		return nil, fmt.Errorf("capital service not initialized")
	}
	ctx, cancel := market.RequestCtx()
	defer cancel()
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
	annCtx, annCancel := market.RequestCtx()
	defer annCancel()
	return a.announcementSvc.GetAnnouncements(annCtx, symbol, pageSize)
}

// GetDragonTiger returns dragon tiger board data for a symbol.
func (a *App) GetDragonTiger(symbol string, endDate string, lookBack int) ([]adapters.DragonTigerRecord, error) {
	if !market.IsTradingHours(market.MarketForSymbol(symbol)) {
		cacheKey := symbol + ":" + endDate + ":" + fmt.Sprint(lookBack)
		var cached []adapters.DragonTigerRecord
		if err := a.dragonTigerCache.Get(cacheKey, &cached); err == nil {
			return cached, nil
		}
		return nil, fmt.Errorf("market %q is currently closed (no cached data)", market.MarketForSymbol(symbol))
	}
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	if lookBack <= 0 {
		lookBack = 30
	}
	dtCtx, dtCancel := market.RequestCtx()
	defer dtCancel()
	records, err := a.signalsAdpt.FetchDragonTigerStock(dtCtx, symbol, endDate, lookBack)
	if err != nil {
		return nil, err
	}
	cacheKey := symbol + ":" + endDate + ":" + fmt.Sprint(lookBack)
	a.dragonTigerCache.Set(cacheKey, records)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("dragon tiger cache save panicked", "recover", r)
			}
		}()
		if e := a.dragonTigerCache.Save(); e != nil {
			slog.Warn("save dragon tiger cache", "error", e)
		}
	}()
	return records, nil
}

// GetDailyDragonTiger returns market-wide dragon tiger board for a trading date.
func (a *App) GetDailyDragonTiger(date string, minNetBuy float64) ([]adapters.DragonTigerStock, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	ddtCtx, ddtCancel := market.RequestCtx()
	defer ddtCancel()
	return a.signalsAdpt.FetchDailyDragonTiger(ddtCtx, date, minNetBuy)
}

// GetLockupExpiry returns lockup expiry data (解禁) for a symbol.
func (a *App) GetLockupExpiry(symbol string) ([]adapters.LockupExpiry, error) {
	if a.signalsAdpt == nil {
		return nil, fmt.Errorf("signals adapter not initialized")
	}
	lkCtx, lkCancel := market.RequestCtx()
	defer lkCancel()
	return a.signalsAdpt.FetchLockupExpiry(lkCtx, symbol)
}

// GetIndustryRanks returns industry ranking by change percent for a given market.
// Uses per-market fallback chains: CN→eastmoney_signals, HK→tencent, US→finnhub.
// Off-hours: returns cached last-known data.
func (a *App) GetIndustryRanks(mkt string, topN int) ([]market.IndustryRank, error) {
	if topN <= 0 {
		topN = 20
	}
	if !market.IsTradingHours(resolveMarket(mkt)) {
		var cached []market.IndustryRank
		if err := a.industryRanksCache.Get(mkt, &cached); err == nil {
			if len(cached) > topN {
				cached = cached[:topN]
			}
			return cached, nil
		}
		// No cache — fall through to live fetch
	}
	reg := a.getMarketReg()
	if reg == nil {
		return nil, fmt.Errorf("market registry not initialized")
	}
	irCtx, irCancel := market.RequestCtx()
	defer irCancel()
	ranks, err := reg.FetchIndustryRanksWithFallback(irCtx, mkt, topN)
	if err != nil {
		return nil, err
	}
	a.industryRanksCache.Set(mkt, ranks)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("industry ranks cache save panicked", "recover", r)
			}
		}()
		if e := a.industryRanksCache.Save(); e != nil {
			slog.Warn("save industry ranks cache", "error", e)
		}
	}()
	return ranks, nil
}

// GetConceptBlocks returns the concept/industry/sector blocks a stock belongs to.
func (a *App) GetConceptBlocks(symbol string) ([]adapters.ConceptBlock, error) {
	if a.conceptAdpt == nil {
		return nil, fmt.Errorf("concept adapter not initialized")
	}
	cbCtx, cbCancel := market.RequestCtx()
	defer cbCancel()
	return a.conceptAdpt.FetchConceptBlocks(cbCtx, symbol)
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
	ctx, cancel := market.RequestCtx()
	defer cancel()

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

// resolveMarket maps market abbreviations to canonical market names for
// IsTradingHours checks. "SH"/"SZ" → "CN", others passed through.
func resolveMarket(mkt string) string {
	switch mkt {
	case "SH", "SZ":
		return "CN"
	}
	return mkt
}

// ServiceShutdown performs graceful cleanup: closes the Python sidecar connection,
// shared DB connection, and releases any resources held by the application.
func (a *App) ServiceShutdown() error {
	// Shut down the workflow execution queue first (no new tasks).
	if execQueue != nil {
		execQueue.Shutdown()
	}

	// Shut down WebSocket hub (stops Run loop, signals clients).
	if a.wsHub != nil {
		a.wsHub.Shutdown()
	}

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

		configJSON, err := json.Marshal(btResult.Config)
		if err != nil {
			slog.Warn("persistBacktest: marshal config", "error", err)
			configJSON = []byte("{}")
		}
		equityJSON, err := json.Marshal(btResult.EquityCurve)
		if err != nil {
			slog.Warn("persistBacktest: marshal equity", "error", err)
			equityJSON = []byte("[]")
		}
		tradesJSON, err := json.Marshal(btResult.Trades)
		if err != nil {
			slog.Warn("persistBacktest: marshal trades", "error", err)
			tradesJSON = []byte("[]")
		}

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

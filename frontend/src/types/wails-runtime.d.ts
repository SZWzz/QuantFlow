// Type declarations for Wails v3 runtime modules
// See: https://v3.wails.io/reference/frontend-runtime

declare module '@wailsio/runtime' {
  export namespace Call {
    /**
     * Calls a bound Go service method by its fully-qualified name
     * in the format 'package.struct.method'.
     * @returns The return value(s) from the Go method.
     *          Multi-value returns come back as an array.
     *          If the Go method's last return is a non-nil error,
     *          the promise rejects with a RuntimeError.
     */
    export function ByName(methodName: string, ...args: any[]): Promise<any>

    /**
     * Calls a bound Go service method by its numeric (uint32 hash) ID.
     */
    export function ByID(methodID: number, ...args: any[]): Promise<any>

    /**
     * Low-level call with a CallOptions descriptor.
     */
    export function Call(options: {
      methodName?: string
      methodID?: number
      args?: any[]
    }): Promise<any>
  }

  export namespace Events {
    export function On(event: string, callback: (event: { data: any }) => void): () => void
    export function Emit(event: string, data?: any): Promise<boolean>
  }

  export namespace System {
    export function invoke(message: string): void
  }

  export namespace Window {
    export function SetTitle(title: string): Promise<void>
    export function Center(): Promise<void>
    export function Minimise(): Promise<void>
    export function Maximise(): Promise<void>
    export function Unmaximise(): Promise<void>
    export function SetFullscreen(fullscreen: boolean): Promise<void>
    export function Close(): Promise<void>
  }

  export namespace Clipboard {
    export function SetText(text: string): Promise<void>
    export function Text(): Promise<string>
  }

  export namespace Dialogs {
    export function Question(options: {
      Title: string
      Message: string
      Buttons: Array<{ Label: string; IsDefault?: boolean }>
    }): Promise<{ Response: number }>
  }

  export { Call as default }
}

// Wails v2-style window.go shim types
// These mirror the Go App methods exposed via main.go services.
// Used by setupWailsBridge() to provide transparent v2→v3 compatibility.

interface AppMethods {
  // --- System ---
  GetVersion(): Promise<string>
  GetConfig(): Promise<Record<string, any>>
  UpdateConfig(patch: Record<string, any>): Promise<void>
  GetSystemStats(): Promise<Record<string, any>>

  // --- Market Data ---
  GetQuote(marketName: string, symbol: string): Promise<{
    symbol: string
    name?: string
    last: number
    bid: number
    ask: number
    volume: number
    change: number
    change_pct: number
    changePct?: number
    timestamp?: number
  }>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<Array<{
    date: string
    open: number
    high: number
    low: number
    close: number
    volume: number
  }>>
  FetchOHLCV(marketName: string, symbol: string, interval: string, fqfactor: string, start: number, end: number): Promise<Array<{
    date: string
    open: number
    high: number
    low: number
    close: number
    volume: number
  }>>
  GetFundFlow(symbol: string, flowType: string): Promise<Record<string, any>>
  GetNorthboundFlow(): Promise<Record<string, any>>
  GetMarketOverview(mkt: string): Promise<Record<string, any>>
  GetCryptoOverview(symbols: string[]): Promise<Record<string, any>>
  GetMarketSnapshot(symbols: string[]): Promise<Array<Record<string, any>>>
  GetCryptoFundingRates(symbols: string[]): Promise<Array<Record<string, any>>>
  GetCryptoLiquidations(symbol: string, limit: number): Promise<Array<Record<string, any>>>
  GetShortInterest(symbol: string): Promise<Array<Record<string, any>>>
  GetUSOptionChain(symbol: string): Promise<Array<Record<string, any>>>
  GetSECFilings(symbol: string): Promise<Array<Record<string, any>>>
  CheckWashSale(symbol: string): Promise<Array<Record<string, any>>>
  GetEarningsCalendar(from: string, to: string): Promise<Array<Record<string, any>>>
  GetCryptoDepth(exchange: string, symbol: string, limit: number): Promise<Record<string, any>>
  GetDeFiTVL(): Promise<Record<string, any>>
  GetWhaleTransactions(address: string): Promise<Record<string, any>>
  GetGasFees(): Promise<Record<string, any>>
  GetMultiDayMinute(symbol: string, days: number): Promise<Array<Record<string, any>>>
  GetCapitalData(symbol: string): Promise<Record<string, any>>
  GetCorrelationMatrix(symbols: string[], lookback: number): Promise<Record<string, Record<string, number>>>
  GetVolatilitySurface(symbol: string): Promise<number[][]>
  ComputeOptionPrice(req: {
    option_type: string; spot_price: number; strike: number;
    time_to_expiry: number; risk_free_rate: number; volatility: number;
    market_price?: number; steps?: number; american?: boolean;
  }): Promise<{
    price: number; binomial_price?: number;
    greeks: { delta: number; gamma: number; theta: number; vega: number; rho: number };
    implied_vol?: number
  }>

  // --- Research ---
  GetSentiment(symbol: string): Promise<Record<string, any>>
  GetSentimentHistory(symbol: string, days: number): Promise<any[]>
  GetStockResearch(symbol: string, tabs: string[]): Promise<Record<string, any>>
  GetCongressTrades(): Promise<any[]>
  ListMLModels(): Promise<any[]>
  GetPredictions(modelID: string, symbol: string): Promise<any[]>
  RunAlphaMining(params: any): Promise<any[]>
  AssessRisk(symbols: string[], modelType: string): Promise<Record<string, any> | null>
  GetFinancialAnalysis(symbol: string): Promise<Record<string, any>>
  GetValuation(symbol: string): Promise<Record<string, any>>
  GetAuditFindings(symbol: string): Promise<Record<string, any>>
  GetForecast(symbol: string): Promise<Record<string, any>>
  GetPredictionMarkets(category: string, limit: number): Promise<Record<string, any>>
  GetPredictionEventDetail(eventID: string): Promise<Record<string, any>>
  GetPredictionSignals(category: string, minProbChange: number): Promise<Record<string, any>>
  GetGeopoliticsRisks(): Promise<Record<string, any>>
  GetEconomicIndicators(): Promise<Record<string, any>>
  GetIndicatorData(seriesID: string, limit: number): Promise<Record<string, any>>
  GetGeopoliticsDetail(topicID: string, timespan: string): Promise<Record<string, any>>
  GetSatelliteSnapshots(): Promise<Record<string, any>>
  GetSatelliteDetail(regionID: string): Promise<Record<string, any>>
  GetChanlun(symbol: string): Promise<Record<string, any>>
  ComputeIndicator(symbol: string, indicatorName: string, params: Record<string, any>): Promise<Record<string, any>>
  ScanStocks(strategyName: string): Promise<Record<string, any>>
  GetBlockRank(market: number, sortField: number, count: number): Promise<any[]>
  GetMACCapitalFlow(symbol: string): Promise<Record<string, any>>
  GetAuction(symbol: string): Promise<any[]>
  GetAbnormalStocks(market: number): Promise<any[]>
  GetAnnouncements(symbol: string, pageSize: number): Promise<any[]>
  GetDragonTiger(symbol: string, endDate: string, lookBack: number): Promise<any[]>
  GetDailyDragonTiger(date: string, minNetBuy: number): Promise<any[]>
  GetLockupExpiry(symbol: string): Promise<any[]>
  GetIndustryRanks(mkt: string, topN: number): Promise<Array<{ name: string; change_pct?: number; changePct?: number }>>
  GetConceptBlocks(symbol: string): Promise<any[]>
  GetReturnDistribution(symbol: string, lookback: number, bins: number): Promise<Record<string, any>>
  GetNews(symbol: string, limit: number): Promise<any[]>
  GetIPOCalendar(startDate: string, endDate: string): Promise<any[]>
  GetExDividendCalendar(startDate: string, endDate: string): Promise<any[]>
  GetCBArbitrageData(): Promise<Record<string, any>>
  GetHKIPOCalendar(year: number): Promise<Record<string, any>>
  GetHKDerivatives(): Promise<Record<string, any>>
  GetHKTradingCalendar(year: number): Promise<Record<string, any>>
  GetHKSettlementInfo(): Promise<Record<string, any>>

  // --- Symbol Search ---
  SearchSymbols(query: string, limit?: number): Promise<Array<{ symbol: string; name?: string; market: string; type: string }>>
  SearchResearch(query: string, channel: string, size: number): Promise<any[]>

  // --- AI ---
  Chat(profileName: string, model: string, message: string): Promise<string>
  ListProfiles(): Promise<Array<{ id: string; name: string; model: string }>>
  GetNotifications(limit: number, offset: number): Promise<Array<{ id: number; title: string; message: string; read: boolean; created_at: string }>>
  MarkNotificationRead(id: number): Promise<void>

  // --- Trading ---
  PlaceOrder(symbol: string, side: string, orderType: string, brokerName: string, qty: number, price: number): Promise<string>
  PlaceOrderWithStop(symbol: string, side: string, orderType: string, brokerName: string, qty: number, price: number, stopPrice: number): Promise<string>
  GetPositions(): Promise<Array<{
    Symbol: string
    Quantity: number
    AvgPrice: number
    MarketPrice: number
    PnL: number
    PnLPct: number
    Name?: string
  }>>
  GetOrders(): Promise<Array<{
    ID: string
    Symbol: string
    Side: string
    Quantity: number
    Price: number
    Status: string
    PlacedAt: string
  }>>
  GetTrades(): Promise<Array<{
    ID: string
    Symbol: string
    Side: string
    Quantity: number
    Price: number
    Timestamp: string
    PnL: number
  }>>
  GetPortfolioSummary(): Promise<{
    total_value: number
    cash_balance: number
    market_value: number
    total_pnl: number
    total_pnl_pct: number
  }>
  GetPortfolioAllocation(): Promise<Record<string, any>>
  GetRebalanceSuggestions(): Promise<Array<Record<string, any>>>
  GetBrokerStatuses(): Promise<Array<{ name: string; connected: boolean; mode: string }>>
  GetEquityCurve(): Promise<Array<{ date: string; nav: number; benchmark: number }>>
  CancelOrder(orderId: string): Promise<void>

  // --- Credential Management ---
  SaveCredential(name: string, credentialType: string, keys: Record<string, string>): Promise<void>
  GetCredential(name: string): Promise<{ name: string; type: string; keys: Record<string, string> }>
  DeleteCredential(name: string): Promise<void>
  ListCredentialNames(): Promise<string[]>
  ListCredentials(): Promise<Array<{ id: number; name: string; type: string; keys: Record<string, string> }>>

  // --- Workflow ---
  ListNodes(): Promise<Array<{ node_type: string; category: string }>>
  GetNodePorts(nodeType: string): Promise<{ inputs: Array<{ name: string; type: string }>; outputs: Array<{ name: string; type: string }> }>
  ValidateWorkflow(jsonDef: string): Promise<string>
  RunWorkflow(jsonDef: string): Promise<any>
  LoadWorkflow(id: string): Promise<any>
  SaveWorkflow(jsonDef: string): Promise<string>
  ListWorkflows(): Promise<Array<{ id: string; name: string; description: string; updated_at: string }>>

  // --- Commodities ---
  GetCommodityQuotes(): Promise<Record<string, any>>

  // --- Schedule ---
  ListScheduleTasks(): Promise<any[]>
  SaveScheduleTask(task: any): Promise<void>
  DeleteScheduleTask(id: string): Promise<void>
  ToggleScheduleTask(id: string, enabled: boolean): Promise<void>

  // --- Tear-off Windows ---
  TearOffPanel(panelId: string, instanceId: string, label: string, paramsJson: string): Promise<void>
  CloseTearOffWindow(instanceId: string): Promise<void>
  GetTearOffPanelInfo(instanceId: string): Promise<[string, string, string]>
  ListTearOffWindows(): Promise<string[]>

  // --- Data Lifecycle ---
  GetStorageStats(): Promise<Array<{ table: string; rows: number; size_bytes: number; oldest: string; newest: string }>>
  ArchiveData(source: string, symbol: string, before: string): Promise<{ id: number; source: string; symbol: string; interval: string; date_from: string; date_to: string; row_count: number; compressed_bytes: number }>
  ExportData(table: string, symbol: string, interval: string, format: string, dateFrom: string, dateTo: string): Promise<string>
  ImportData(filePath: string, table: string): Promise<number>
  CleanupData(table: string, symbol: string, before: string, dryRun: boolean): Promise<{ affected_rows: number; preview: Record<string, unknown>[]; table: string; dry_run: boolean }>

  // --- Logs ---
  GetLogs(afterID: number): Promise<Array<{ id: number; time: string; level: string; message: string; attrs?: Record<string, any> }>>

  // --- Update Management ---
  CheckUpdate(): Promise<{
    has_update: boolean
    current_version: string
    latest_version: string
    asset_url: string
    asset_size: number
    checksum: string
    changelog: string
  }>
  ApplyUpdate(assetURL: string, checksum: string): Promise<void>
  GetUpdateInterval(): Promise<string>
  SetUpdateInterval(interval: string): Promise<void>

  // --- Connection Status ---
  GetConnectionStatus(): Promise<{ markets: Record<string, string>; brokers: Record<string, string>; python: string }>

  // --- Layout Templates ---
  SaveLayout(name: string, layoutJSON: string): Promise<void>
  LoadLayout(name: string): Promise<string>
  ListLayouts(): Promise<string[]>
  DeleteLayout(name: string): Promise<void>

  // --- Daily Reports ---
  GetDailyReport(date: string): Promise<{
    date: string
    market_value: number
    day_pnl: number
    day_pnl_percent: number
    total_pnl: number
    total_pnl_percent: number
    trades: number
    commission: number
    tax: number
    max_drawdown: number
    positions: Array<{ symbol: string; quantity: number; market_val: number; pnl: number; pnl_pct: number }>
    notes: string
  } | null>
  ListDailyReports(limit: number): Promise<Array<{
    date: string
    market_value: number
    day_pnl: number
    day_pnl_percent: number
    total_pnl: number
    total_pnl_percent: number
    trades: number
    commission: number
    tax: number
    max_drawdown: number
    positions: Array<{ symbol: string; quantity: number; market_val: number; pnl: number; pnl_pct: number }>
    notes: string
  }>>
  GenerateDailyReport(date: string): Promise<{
    date: string
    market_value: number
    day_pnl: number
    day_pnl_percent: number
    total_pnl: number
    total_pnl_percent: number
    trades: number
    commission: number
    tax: number
    max_drawdown: number
    positions: Array<{ symbol: string; quantity: number; market_val: number; pnl: number; pnl_pct: number }>
    notes: string
  }>
  ExportReportCSV(date: string): Promise<void>

  // --- Trading Mode ---
  GetTradingMode(): Promise<string>
  SwitchToLive(skipChecks: boolean): Promise<{ checks: Array<{ name: string; ok: boolean; message: string; blocking: boolean }>; all_clear: boolean }>
  SwitchToPaper(): Promise<void>
  EmergencyClose(): Promise<Record<string, any>>

  // --- Position Reconciliation ---
  ReconcileAll(): Promise<Array<{
    id: number
    created_at: string
    broker_name: string
    match_count: number
    diff_count: number
    diffs: Array<{ symbol: string; oms_quantity: number; broker_quantity: number; oms_avg_price: number; broker_avg_price: number }>
    oms_only: string[]
    broker_only: string[]
  }>>
  GetReconciliationReports(limit: number): Promise<Array<{
    id: number
    created_at: string
    broker_name: string
    match_count: number
    diff_count: number
    diffs: Array<{ symbol: string; oms_quantity: number; broker_quantity: number; oms_avg_price: number; broker_avg_price: number }>
    oms_only: string[]
    broker_only: string[]
  }>>

  // --- Crash Reports ---
  ListCrashReports(): Promise<Array<{
    id: string; timestamp: string; version: string; go_version: string;
    os: string; arch: string; build_mode: string; panic: string; stack: string;
    logs: string[] | null;
    app_state: { trading_mode: string; active_brokers: string[]; panel_count: number; workflow_count: number; uptime_seconds: number }
  }>>
  DeleteCrashReport(id: string): Promise<void>
  UploadCrashReport(id: string): Promise<void>
  GetCrashDir(): Promise<string>

  // --- Sector Dashboard ---
  GetSectorHeatmap(market: string): Promise<Array<{ name: string; change_pct: number; volume: number; pe: number; pe_pct: number }>>
  GetSectorValuation(market: string): Promise<Array<{ name: string; pe: number; pe_pct: number; pb: number; pb_pct: number; roe: number }>>

  // --- Valuation & Dupont ---
  GetPriceBand(symbol: string, market: string, interval: string, lookbackDays: number): Promise<{
    symbol: string; metric: string; current: number; mean: number; stddev: number; percentile: number;
    points: Array<{ date: string; close: number; band_1: number; band_2: number; band_3: number; band_4: number; band_5: number }>
  }>
  GetDupontAnalysis(symbol: string): Promise<{ symbol: string; roe: number; net_margin: number; asset_turnover: number; equity_multiplier: number; gross_margin: number; eps: number }>
  GetPeerRadar(symbol: string): Promise<Array<{ symbol: string; name: string; metrics: Record<string, number> }>>

  // --- P1 Analysis ---
  GetMacroSnapshot(country: string): Promise<{
    growth: Array<{ country: string; name: string; value: number; unit: string; date: string; change: number }>
    inflation: Array<{ country: string; name: string; value: number; unit: string; date: string; change: number }>
    monetary: Array<{ country: string; name: string; value: number; unit: string; date: string; change: number }>
    policy: Array<{ country: string; name: string; value: number; unit: string; date: string; change: number }>
    updated_at: string
  }>
  GetStyleQuadrant(market: string): Promise<Array<{ index: string; size: number; style: number; return_1m: number }>>
  GetMarketSentiment(market: string): Promise<{ limit_up: number; limit_down: number; turnover: number; northbound_cum: number }>
  ComputeEventStudy(symbol: string, market: string, interval: string, eventDate: string, window: number): Promise<{
    symbol: string; event_date: string; event_type: string; window: number;
    car: number;
    daily_ar: Array<{ date: string; day: number; ar: number; car: number; stock_r: number; bench_r: number }>
  }>

  // --- P2 Analysis ---
  GetTop10Holders(symbol: string): Promise<Array<{ name: string; type: string; shares: number; pct: number; change: number; market_value: number; report_date: string }>>
  GetUnlockCalendar(days: number): Promise<Array<{ symbol: string; name: string; unlock_date: string; unlock_shares: number; unlock_pct: number; float_ratio: number; market_value: number }>>
  GetFactorAttribution(totalReturn: number): Promise<{ total_return: number; market_beta: number; style_factors: Record<string, number>; industry_factors: Record<string, number>; alpha: number }>
}

declare global {
  interface Window {
    go?: {
      main: {
        App: AppMethods
      }
    }
  }
}

export {}

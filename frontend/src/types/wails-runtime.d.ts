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

  // --- Market Data ---
  GetQuote(marketName: string, symbol: string): Promise<any>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<any>
  FetchOHLCV(marketName: string, symbol: string, interval: string, fqfactor: string, start: number, end: number): Promise<any>
  GetFundFlow(symbol: string, flowType: string): Promise<any>
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

  // --- Research ---
  GetSentiment(symbol: string): Promise<any>
  GetSentimentHistory(symbol: string, days: number): Promise<any[]>
  GetStockResearch(symbol: string, tabs: string[]): Promise<any>
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
  GetMACCapitalFlow(symbol: string): Promise<any>
  GetAuction(symbol: string): Promise<any[]>
  GetAbnormalStocks(market: number): Promise<any[]>
  GetMultiDayMinute(symbol: string, days: number): Promise<any>
  GetCapitalData(symbol: string): Promise<Record<string, any>>
  GetAnnouncements(symbol: string, pageSize: number): Promise<any[]>
  GetDragonTiger(symbol: string, endDate: string, lookBack: number): Promise<any[]>
  GetDailyDragonTiger(date: string, minNetBuy: number): Promise<any[]>
  GetLockupExpiry(symbol: string): Promise<any[]>
  GetIndustryRanks(mkt: string, topN: number): Promise<any[]>
  GetConceptBlocks(symbol: string): Promise<any[]>
  GetCorrelationMatrix(symbols: string[], lookback: number): Promise<Record<string, Record<string, number>>>
  GetReturnDistribution(symbol: string, lookback: number, bins: number): Promise<Record<string, any>>
  GetVolatilitySurface(symbol: string): Promise<number[][]>
  GetNews(symbol: string, limit: number): Promise<any[]>
  GetIPOCalendar(startDate: string, endDate: string): Promise<any[]>
  GetExDividendCalendar(startDate: string, endDate: string): Promise<any[]>
  GetCBArbitrageData(): Promise<Record<string, any>>
  GetHKIPOCalendar(year: number): Promise<Record<string, any>>
  GetHKDerivatives(): Promise<Record<string, any>>
  GetHKTradingCalendar(year: number): Promise<Record<string, any>>
  GetHKSettlementInfo(): Promise<Record<string, any>>

  // --- Symbol Search ---
  SearchSymbols(query: string): Promise<any[]>
  SearchResearch(query: string, channel: string, size: number): Promise<any[]>

  // --- AI ---
  Chat(profileName: string, model: string, message: string): Promise<string>
  ListProfiles(): Promise<any[]>
  GetNotifications(limit: number, offset: number): Promise<any[]>
  MarkNotificationRead(id: number): Promise<void>

  // --- Trading ---
  PlaceOrder(symbol: string, side: string, orderType: string, qty: number, price: number): Promise<any>
  GetPositions(): Promise<any[]>
  GetOrders(): Promise<any[]>
  GetTrades(): Promise<any[]>
  GetPortfolioSummary(): Promise<Record<string, any>>
  GetPortfolioAllocation(): Promise<any>
  GetRebalanceSuggestions(): Promise<Array<Record<string, any>>>
  GetBrokerStatuses(): Promise<any[]>
  RunBacktest(jsonDef: string): Promise<Record<string, any>>

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

  // --- System Monitor ---
  GetSystemStats(): Promise<Record<string, any>>

  // --- Tear-off Windows ---
  TearOffPanel(panelId: string, instanceId: string, label: string, paramsJson: string): Promise<void>
  CloseTearOffWindow(instanceId: string): Promise<void>
  GetTearOffPanelInfo(instanceId: string): Promise<[string, string, string]>
  ListTearOffWindows(): Promise<string[]>

  // Index signature for dynamic calls via proxy
  [key: string]: (...args: any[]) => Promise<any>
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

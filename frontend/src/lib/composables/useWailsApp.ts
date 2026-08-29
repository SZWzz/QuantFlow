export interface QuoteData {
  symbol: string
  name?: string
  last?: number
  price?: number
  change?: number
  change_percent?: number
  change_pct?: number
  changePct?: number
  high?: number
  low?: number
  open?: number
  volume?: number
  amount?: number
  bid?: number
  ask?: number
  turnover_rate?: number
  volume_ratio?: number
  amplitude?: number
  avg_price?: number
  inside_volume?: number
  outside_volume?: number
  market_cap?: number
  marketCap?: number
  pe_ratio?: number
  pe?: number
  limit_up?: number
  limit_down?: number
  prevClose?: number
  turnover?: number
}

export interface OHLCVBar {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface MinuteTick {
  time: string
  price: number
  avg_price: number
  volume: number
  amount: number
}

export interface WailsApp {
  FetchOHLCV(market: string, symbol: string, interval: string, fq: string, start: number, end: number): Promise<[OHLCVBar[], string]>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<[MinuteTick[], string]>
  GetQuote(market: string, symbol: string): Promise<[QuoteData, string]>
  GetAuditFindings(symbol: string): Promise<Record<string, any>>
  GetFinancialAnalysis(symbol: string): Promise<Record<string, any>>
  GetDelistingRisk(symbol: string): Promise<Record<string, any>>

  // --- market data ---
  GetMarketOverview(mkt: string): Promise<Record<string, any>>
  GetCryptoOverview(symbols: string[]): Promise<Record<string, any>>
  GetFundFlow(symbol: string, flowType?: string): Promise<Record<string, any>>
  GetNorthboundFlow(): Promise<Record<string, any>>
  GetFinancialStatements(symbol: string): Promise<Record<string, any>>
  GetIndustryRanks(market: string, lookback?: number): Promise<Record<string, any>[]>
  GetPredictionMarkets(category: string, limit: number): Promise<Record<string, any>>
  GetSatelliteSnapshots(): Promise<Record<string, any>>
  GetSECFilings(symbol: string): Promise<Record<string, any>[]>
  GetVolatilitySurface(symbol: string): Promise<Record<string, any>>
  GetValuationDCF(symbol: string): Promise<Record<string, any>>
  GetGeopoliticsRisks(): Promise<Record<string, any>>
  GetSystemStats(): Promise<Record<string, any>>
  GetIPOData(market: string): Promise<Record<string, any>[]>

  // --- trading ---
  GetTradingMode(): Promise<string>
  ListBacktestHistory(limit: number, offset: number): Promise<Record<string, any>[]>

  // --- HK Connect ---
  GetHKConnectFlow(): Promise<Record<string, any>>
  GetHKSettlementInfo(): Promise<Record<string, any>>

  // --- Congress trading ---
  GetCongressTrades(): Promise<Record<string, any>[]>

  // --- Cache (for offline cache plan) ---
  CacheGet(key: string): Promise<string>
  CacheSet(key: string, data: string, ttlSeconds: number, category?: string): Promise<void>
  CacheClear(keyOrCategory: string): Promise<void>

  // --- generic data fetching ---
  FetchData(source: string, dataType: string, symbols: string[], startDate: string, endDate: string, params: Record<string, string>): Promise<Record<string, any>>

  // --- AI ---
  Chat(profileName: string, model: string, message: string): Promise<string>

  // --- trading / orders ---
  PlaceOrder(symbol: string, side: string, orderType: string, brokerName: string, qty: number, price: number): Promise<Record<string, any>>
  PlaceOrderWithStop(symbol: string, side: string, orderType: string, brokerName: string, qty: number, price: number, stopPrice: number): Promise<Record<string, any>>
  CheckWashSale(symbol: string): Promise<Record<string, any>[]>

  // --- backtest persistence ---
  GetStoredBacktestResult(id: number): Promise<Record<string, any>>
  DeleteBacktestResult(id: number): Promise<boolean>
  ClearBacktestResults(): Promise<number>

  // --- credentials / brokers ---
  SaveCredential(name: string, credType: string, keys: Record<string, string>): Promise<void>
  /** Declared for guarded use; the Go backend does not implement this yet. */
  TestBrokerConnection?(broker: string, config: Record<string, string>): Promise<Record<string, any>>

  // --- market data (panel wrappers) ---
  GetNews(symbol: string, limit: number): Promise<Record<string, any>[]>
  GetDepth(mkt: string, symbol: string): Promise<Record<string, any>>
  GetAbnormalStocks(mkt: string): Promise<Record<string, any>[]>
  GetCommodityQuotes(): Promise<Record<string, any>>
  GetCorrelationMatrix(symbols: string[], lookback: number): Promise<Record<string, Record<string, number>>>
  GetReturnDistribution(symbol: string, lookback: number, bins: number): Promise<Record<string, any>>
  GetEconomicIndicators(): Promise<Record<string, any>>
  GetIndicatorData(seriesID: string, limit: number): Promise<Record<string, any>>
  GetEarningsCalendar(from: string, to: string): Promise<Record<string, any>[]>
  GetExDividendCalendar(startDate: string, endDate: string): Promise<Record<string, any>[]>
  GetDailyDragonTiger(date: string, minNetBuy: number): Promise<Record<string, any>[]>
  GetDragonTiger(symbol: string, endDate: string, lookBack: number): Promise<Record<string, any>[]>
  GetCBArbitrageData(): Promise<Record<string, any>>
  GetChanlun(symbol: string): Promise<Record<string, any>>
  GetHKDerivatives(): Promise<Record<string, any>>
  GetHKTradingCalendar(year: number): Promise<Record<string, any>>
  GetHKIPOCalendar(year: number): Promise<Record<string, any>>
  GetHKFinancialStatements(symbol: string): Promise<Record<string, any>>
  GetUSOptionChain(symbol: string): Promise<Record<string, any>[]>
  GetShortInterest(symbol: string): Promise<Record<string, any>[]>
  GetForecast(symbol: string): Promise<Record<string, any>>
  GetSatelliteDetail(regionID: string): Promise<Record<string, any>>
  GetGeopoliticsDetail(topicID: string, timespan: string): Promise<Record<string, any>>
  GetPredictionEventDetail(eventID: string): Promise<Record<string, any>>
  GetPredictionSignals(category: string, minProbChange: number): Promise<Record<string, any>>
  ScanStocks(strategyName: string): Promise<Record<string, any>>
  SearchSymbols(query: string, limit: number): Promise<Record<string, any>[]>
  /** Declared for guarded use; factor listing is served by the Python sidecar. */
  ListFactors?(): Promise<Record<string, any>>

  // --- crypto ---
  GetCryptoDepth(exchange: string, symbol: string, limit: number): Promise<Record<string, any>>
  GetCryptoFundingRates(symbols: string[]): Promise<Record<string, any>[]>
  GetCryptoLiquidations(symbol: string, limit: number): Promise<Record<string, any>[]>
  GetDeFiTVL(): Promise<Record<string, any>>
  GetGasFees(): Promise<Record<string, any>>
  GetWhaleTransactions(address: string): Promise<Record<string, any>>

  // --- misc ---
  OpenURL(url: string): Promise<void>
  TearOffPanel(panelId: string, instanceId: string, label: string, paramsJson: string): Promise<void>
}

let cachedApp: WailsApp | null = null

/** Reset the cached reference — used in tests to provide a fresh mock between runs. */
export function resetWailsApp(): void {
  cachedApp = null
}

export function useWailsApp(): WailsApp | null {
  if (cachedApp) return cachedApp
  const app = (window as any)?.go?.main?.App
  if (!app) return null
  cachedApp = app as WailsApp
  return cachedApp
}

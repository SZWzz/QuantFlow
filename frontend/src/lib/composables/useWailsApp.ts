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
  ListBacktestHistory(limit: number, offset: number): Promise<Record<string, any>>

  // --- HK Connect ---
  GetHKConnectFlow(): Promise<Record<string, any>>
  GetHKSettlementInfo(): Promise<Record<string, any>>

  // --- Congress trading ---
  GetCongressTrades(): Promise<Record<string, any>[]>

  // --- Cache (for offline cache plan) ---
  CacheGet(key: string): Promise<string>
  CacheSet(key: string, data: string, ttlSeconds: number, category?: string): Promise<void>
  CacheClear(keyOrCategory: string): Promise<void>
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

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
}

export interface GetMinuteLineResult {
  ticks: MinuteTick[]
  last_time?: string
}

export interface MultiDayData {
  date: string
  ticks: MinuteTick[]
}

export interface MultiDayMinute {
  symbol: string
  days: MultiDayData[]
}

export interface WailsApp {
  FetchOHLCV(market: string, symbol: string, interval: string, fq: string, start: number, end: number): Promise<[OHLCVBar[], string]>
  GetMinuteLine(symbol: string, sinceTimestamp: number): Promise<GetMinuteLineResult>
  GetMultiDayMinute(symbol: string, days: number): Promise<MultiDayMinute>
  GetQuote(market: string, symbol: string): Promise<[QuoteData, string]>
  GetAuditFindings(symbol: string): Promise<Record<string, any>>
  GetFinancialAnalysis(symbol: string): Promise<Record<string, any>>
  GetDelistingRisk(symbol: string): Promise<Record<string, any>>
}

let cachedApp: WailsApp | null = null

export function useWailsApp(): WailsApp | null {
  if (cachedApp) return cachedApp
  const app = (window as any)?.go?.main?.App
  if (!app) return null
  cachedApp = app as WailsApp
  return cachedApp
}

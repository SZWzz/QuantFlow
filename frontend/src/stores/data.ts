import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface QuoteSnapshot {
  symbol: string
  last: number
  bid: number
  ask: number
  volume: number
  change: number
  changePct: number
  timestamp: number
}

export interface OHLCVBar {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface DataSourceStatus {
  name: string
  status: 'connected' | 'disconnected' | 'error'
  lastUpdate: number
}

export interface IndexSnapshot {
  symbol: string
  name: string
  last: number
  changePct: number
  prevClose: number
  sparkline: number[]
  ohlcv?: { open: number; high: number; low: number; close: number }[]
}

export interface MarketBreadth {
  advancers: number
  decliners: number
  unchanged: number
}

export interface SectorRanking {
  name: string
  changePct: number
}

export interface MarketSentiment {
  limitUp: number
  limitDown: number
  northboundFlow: number
  totalVolume: number
}

export interface MarketOverview {
  indices: IndexSnapshot[]
  breadth: MarketBreadth
  sentiment?: MarketSentiment
  sectors: SectorRanking[]
  updatedAt: number
}

interface CacheEntry<T = any> {
  data: T
  timestamp: number
  ttl: number
}

export const useDataStore = defineStore('data', () => {
  const quotes = ref<Map<string, QuoteSnapshot>>(new Map())
  const ohlcvCache = ref<Map<string, OHLCVBar[]>>(new Map())
  const sourceStatus = ref<DataSourceStatus[]>([])
  const isOffline = ref(false)
  const marketOverview = ref<MarketOverview | null>(null)
  const marketLoading = ref(false)
  const selectedIndexSymbol = ref('')
  const error = ref<string | null>(null)
  const cache = ref<Map<string, CacheEntry>>(new Map())

  function updateQuote(symbol: string, quote: QuoteSnapshot) {
    quotes.value.set(symbol, quote)
  }

  function getQuote(symbol: string): QuoteSnapshot | undefined {
    return quotes.value.get(symbol)
  }

  function setOHLCV(key: string, bars: OHLCVBar[]) {
    ohlcvCache.value.set(key, bars)
  }

  function getOHLCV(key: string): OHLCVBar[] | undefined {
    return ohlcvCache.value.get(key)
  }

  function toggleOffline() {
    isOffline.value = !isOffline.value
  }

  function setCached<T>(key: string, data: T, ttlMs: number): void {
    cache.value.set(key, { data, timestamp: Date.now(), ttl: ttlMs })
  }

  function getCached<T>(key: string): T | null {
    const entry = cache.value.get(key)
    if (!entry) return null
    if (Date.now() - entry.timestamp > entry.ttl) {
      cache.value.delete(key)
      return null
    }
    return entry.data as T
  }

  function clearCached(key?: string): void {
    if (key) cache.value.delete(key)
    else cache.value = new Map()
  }

  function setSelectedIndex(symbol: string) {
    selectedIndexSymbol.value = symbol
  }

  async function fetchMarketOverview(market = 'CN') {
    const app = (window as any).go?.main?.App
    if (!app) return

    const sectorsCacheKey = `industryRanks:${market}`
    const cachedSectors = getCached<SectorRanking[]>(sectorsCacheKey)

    marketLoading.value = true
    try {
      // Run GetMarketOverview and GetIndustryRanks in parallel
      const [overviewResult, industryResult] = await Promise.all([
        (async () => {
          try {
            return await app.GetMarketOverview(market)
          } catch {
            return null
          }
        })(),
        (async () => {
          // Use cache if available, skip the Go call entirely
          if (cachedSectors) return null
          if (!app.GetIndustryRanks) return null
          try {
            return await app.GetIndustryRanks(market, 30)
          } catch {
            return null
          }
        })(),
      ])

      // Process indices + breadth from market overview
      let indices: IndexSnapshot[] = []
      let breadth: MarketBreadth = { advancers: 0, decliners: 0, unchanged: 0 }
      let sentiment: MarketSentiment = { limitUp: 0, limitDown: 0, northboundFlow: 0, totalVolume: 0 }
      if (overviewResult) {
        if (overviewResult.indices) {
          indices = (overviewResult.indices as any[]).map((idx: any) => ({
            symbol: idx.code,
            name: idx.name,
            last: idx.price,
            changePct: idx.change_pct,
            prevClose: idx.prev_close ?? 0,
            sparkline: [],
            ohlcv: idx.ohlcv as { open: number; high: number; low: number; close: number }[] | undefined,
          }))
        }
        if (overviewResult.breadth) {
          breadth = {
            advancers: overviewResult.breadth.advancers ?? 0,
            decliners: overviewResult.breadth.decliners ?? 0,
            unchanged: overviewResult.breadth.unchanged ?? 0,
          }
        }
        if (overviewResult.sentiment) {
          sentiment = {
            limitUp: overviewResult.sentiment.limit_up ?? overviewResult.sentiment.limitUp ?? 0,
            limitDown: overviewResult.sentiment.limit_down ?? overviewResult.sentiment.limitDown ?? 0,
            northboundFlow: overviewResult.sentiment.northbound_flow ?? overviewResult.sentiment.northboundFlow ?? 0,
            totalVolume: overviewResult.sentiment.total_volume ?? overviewResult.sentiment.totalVolume ?? 0,
          }
        }
      }

      // Process sector ranks (use cache if fresh, otherwise use API result)
      let sectors: SectorRanking[] = []
      if (cachedSectors) {
        sectors = cachedSectors
      } else if (industryResult) {
        sectors = (industryResult as any[]).map((s: any) => ({
          name: s.name,
          changePct: s.change_pct ?? s.changePct ?? 0,
        }))
        setCached(sectorsCacheKey, sectors, 5 * 60 * 1000)
      }

      marketOverview.value = { indices, breadth, sentiment, sectors, updatedAt: Date.now() }
    } catch (e) {
      error.value = String(e)
    } finally {
      marketLoading.value = false
    }
  }

  async function fetchMinuteLine(symbol: string, sinceTimestamp: number) {
    const app = (window as any).go?.main?.App
    if (!app) throw new Error('Wails bridge not available')
    return app.GetMinuteLine(symbol, sinceTimestamp)
  }

  return {
    quotes,
    ohlcvCache,
    sourceStatus,
    isOffline,
    marketOverview,
    marketLoading,
    error,
    cache,
    updateQuote,
    getQuote,
    setOHLCV,
    getOHLCV,
    toggleOffline,
    setCached,
    fetchMinuteLine,
    getCached,
    clearCached,
    fetchMarketOverview,
    selectedIndexSymbol,
    setSelectedIndex,
  }
})


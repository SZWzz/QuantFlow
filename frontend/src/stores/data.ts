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
  sparkline: number[]
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

export interface MarketOverview {
  indices: IndexSnapshot[]
  breadth: MarketBreadth
  sectors: SectorRanking[]
  updatedAt: number
}

export const useDataStore = defineStore('data', () => {
  const quotes = ref<Map<string, QuoteSnapshot>>(new Map())
  const ohlcvCache = ref<Map<string, OHLCVBar[]>>(new Map())
  const sourceStatus = ref<DataSourceStatus[]>([])
  const isOffline = ref(false)
  const marketOverview = ref<MarketOverview | null>(null)
  const marketLoading = ref(false)

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

  async function fetchMarketOverview() {
    marketLoading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (!app) return

      // Fetch indices from real API
      let indices: IndexSnapshot[] = []
      let breadth: MarketBreadth = { advancers: 0, decliners: 0, unchanged: 0 }
      let sectors: SectorRanking[] = []

      try {
        const overview = await app.GetMarketOverview()
        if (overview?.indices) {
          indices = (overview.indices as any[]).map((idx: any) => ({
            symbol: idx.code,
            name: idx.name,
            last: idx.price,
            changePct: idx.change_pct,
            sparkline: [],
          }))
        }
        if (overview?.breadth) {
          breadth = {
            advancers: overview.breadth.advancers ?? 0,
            decliners: overview.breadth.decliners ?? 0,
            unchanged: overview.breadth.unchanged ?? 0,
          }
        }
      } catch {
        // indices remain empty
      }

      // Fetch sector ranks
      if (app.GetIndustryRanks) {
        try {
          sectors = await app.GetIndustryRanks(30)
        } catch {
          // sectors remain empty
        }
      }

      marketOverview.value = { indices, breadth, sectors, updatedAt: Date.now() }
    } catch (e) {
      console.warn('fetchMarketOverview failed:', e)
    } finally {
      marketLoading.value = false
    }
  }

  return {
    quotes,
    ohlcvCache,
    sourceStatus,
    isOffline,
    marketOverview,
    marketLoading,
    updateQuote,
    getQuote,
    setOHLCV,
    getOHLCV,
    toggleOffline,
    fetchMarketOverview,
  }
})


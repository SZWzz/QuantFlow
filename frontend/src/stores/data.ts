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
      if (app?.GetIndustryRanks) {
        const sectors = await app.GetIndustryRanks(30)
        marketOverview.value = {
          indices: generateMockIndices(),
          breadth: { advancers: 2147, decliners: 1832, unchanged: 345 },
          sectors: sectors ?? generateMockSectors(),
          updatedAt: Date.now(),
        }
      } else {
        marketOverview.value = {
          indices: generateMockIndices(),
          breadth: { advancers: 2147, decliners: 1832, unchanged: 345 },
          sectors: generateMockSectors(),
          updatedAt: Date.now(),
        }
      }
    } catch (e) {
      console.warn('fetchMarketOverview failed:', e)
      marketOverview.value = {
        indices: generateMockIndices(),
        breadth: { advancers: 2147, decliners: 1832, unchanged: 345 },
        sectors: generateMockSectors(),
        updatedAt: Date.now(),
      }
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

function generateMockIndices(): IndexSnapshot[] {
  return [
    { symbol: '000001.SH', name: '上证指数', last: 3356.28, changePct: 0.42, sparkline: mockSparkline(3356, 20) },
    { symbol: '399001.SZ', name: '深证成指', last: 10872.51, changePct: 0.67, sparkline: mockSparkline(10872, 20) },
    { symbol: '399006.SZ', name: '创业板指', last: 2178.33, changePct: 1.12, sparkline: mockSparkline(2178, 20) },
    { symbol: '000688.SH', name: '科创50', last: 956.42, changePct: -0.25, sparkline: mockSparkline(956, 20) },
    { symbol: 'HSI', name: '恒生指数', last: 19638.12, changePct: 0.85, sparkline: mockSparkline(19638, 20) },
    { symbol: 'SPX', name: 'S&P 500', last: 5482.67, changePct: -0.18, sparkline: mockSparkline(5482, 20) },
    { symbol: 'IXIC', name: 'Nasdaq', last: 17689.54, changePct: 0.33, sparkline: mockSparkline(17689, 20) },
  ]
}

function generateMockSectors(): SectorRanking[] {
  return [
    { name: '半导体', changePct: 3.45 },
    { name: '软件开发', changePct: 2.87 },
    { name: '光伏设备', changePct: 2.31 },
    { name: '消费电子', changePct: 2.15 },
    { name: '汽车整车', changePct: 1.92 },
    { name: '医疗器械', changePct: 1.56 },
    { name: '化学制药', changePct: 1.23 },
    { name: '食品饮料', changePct: 0.98 },
    { name: '银行', changePct: 0.45 },
    { name: '电力', changePct: 0.32 },
    { name: '房地产', changePct: -0.56 },
    { name: '钢铁', changePct: -1.23 },
    { name: '煤炭开采', changePct: -1.89 },
    { name: '航运港口', changePct: -2.34 },
    { name: '影视院线', changePct: -2.78 },
  ]
}

function mockSparkline(base: number, count: number): number[] {
  const result: number[] = []
  let v = base * 0.98
  for (let i = 0; i < count; i++) {
    v += v * (Math.random() - 0.48) * 0.008
    result.push(v)
  }
  return result
}

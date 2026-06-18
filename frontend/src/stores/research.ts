import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface SentimentOutput {
  symbol: string
  score: number
  label: string
  confidence: number
  keywords: string[]
  entities: string[]
  source: string
  compute_time_ms: number
}

export interface FinancialData {
  symbol: string
  revenue: number
  net_income: number
  eps: number
  total_assets: number
  total_equity: number
  total_debt: number
  free_cash_flow: number
  market_cap: number
}

export interface StockResearchResult {
  symbol: string
  overview: Record<string, any>
  financials?: { data: FinancialData; ratios: Record<string, number> }
  sentiment?: SentimentOutput
  peers?: any[]
  estimates?: any[]
  insider?: any[]
}

export const useResearchStore = defineStore('research', () => {
  const sentiment = ref<SentimentOutput | null>(null)
  const research = ref<StockResearchResult | null>(null)
  const sentimentHistory = ref<SentimentOutput[]>([])
  const loading = ref(false)
  const isBridgeAvailable = ref(false)

  async function fetchSentiment(symbol: string) {
    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetSentiment) {
        sentiment.value = await app.GetSentiment(symbol)
        isBridgeAvailable.value = (sentiment.value?.compute_time_ms ?? 0) > 0
      } else {
        sentiment.value = {
          symbol, score: 0, label: 'neutral', confidence: 0,
          keywords: ['frontend_mock'], entities: [], source: 'mock', compute_time_ms: 0,
        }
      }
    } catch (e) {
      console.warn('GetSentiment unavailable:', e)
      sentiment.value = {
        symbol, score: 0, label: 'neutral', confidence: 0,
        keywords: ['frontend_mock'], entities: [], source: 'mock', compute_time_ms: 0,
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchStockResearch(symbol: string, tabs: string[] = ['overview', 'financials', 'sentiment']) {
    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetStockResearch) {
        research.value = await app.GetStockResearch(symbol, tabs)
      } else {
        research.value = {
          symbol,
          overview: { symbol, name: symbol, sector: 'Mock', market_cap: 0 },
        }
      }
    } catch (e) {
      console.warn('GetStockResearch unavailable:', e)
      research.value = {
        symbol,
        overview: { symbol, name: symbol, sector: 'Mock', market_cap: 0 },
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchSentimentHistory(symbol: string, days: number = 30) {
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetSentimentHistory) {
        sentimentHistory.value = await app.GetSentimentHistory(symbol, days)
      }
    } catch (e) {
      console.warn('GetSentimentHistory unavailable:', e)
    }
  }

  const congressTrades = ref<any[] | null>(null)

  async function fetchCongressTrades() {
    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetCongressTrades) {
        congressTrades.value = await app.GetCongressTrades()
      } else {
        congressTrades.value = [
          { name: 'Nancy Pelosi', chamber: 'House', party: 'Democrat', symbol: 'NVDA', type: 'Buy', amount: '$1M-$5M', date: '2026-06-15' },
          { name: 'Nancy Pelosi', chamber: 'House', party: 'Democrat', symbol: 'MSFT', type: 'Buy', amount: '$500K-$1M', date: '2026-06-10' },
          { name: 'Dan Crenshaw', chamber: 'House', party: 'Republican', symbol: 'XOM', type: 'Buy', amount: '$100K-$250K', date: '2026-06-08' },
          { name: 'Tommy Tuberville', chamber: 'Senate', party: 'Republican', symbol: 'COIN', type: 'Sell', amount: '$250K-$500K', date: '2026-06-05' },
          { name: 'Josh Gottheimer', chamber: 'House', party: 'Democrat', symbol: 'GOOGL', type: 'Buy', amount: '$50K-$100K', date: '2026-06-03' },
          { name: 'John Curtis', chamber: 'Senate', party: 'Republican', symbol: 'PLTR', type: 'Buy', amount: '$100K-$250K', date: '2026-06-01' },
          { name: 'Nancy Pelosi', chamber: 'House', party: 'Democrat', symbol: 'AAPL', type: 'Buy', amount: '$500K-$1M', date: '2026-05-28' },
          { name: 'Rick Scott', chamber: 'Senate', party: 'Republican', symbol: 'TSLA', type: 'Sell', amount: '$250K-$500K', date: '2026-05-25' },
          { name: 'Ro Khanna', chamber: 'House', party: 'Democrat', symbol: 'AMD', type: 'Buy', amount: '$100K-$250K', date: '2026-05-22' },
          { name: 'Pat Toomey', chamber: 'Senate', party: 'Republican', symbol: 'BTC ETF', type: 'Sell', amount: '$1M-$5M', date: '2026-05-20' },
          { name: 'Mark Green', chamber: 'House', party: 'Republican', symbol: 'UNH', type: 'Buy', amount: '$50K-$100K', date: '2026-05-18' },
          { name: 'Kyrsten Sinema', chamber: 'Senate', party: 'Independent', symbol: 'AMZN', type: 'Buy', amount: '$100K-$250K', date: '2026-05-15' },
        ]
      }
    } catch (e) {
      console.warn('GetCongressTrades unavailable:', e)
      congressTrades.value = null
    } finally {
      loading.value = false
    }
  }

  return {
    sentiment, research, sentimentHistory, loading, isBridgeAvailable,
    congressTrades,
    fetchSentiment, fetchStockResearch, fetchSentimentHistory, fetchCongressTrades,
  }
})

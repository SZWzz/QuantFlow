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

  return {
    sentiment, research, sentimentHistory, loading, isBridgeAvailable,
    fetchSentiment, fetchStockResearch, fetchSentimentHistory,
  }
})

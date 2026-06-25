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
  const isBridgeAvailable = ref<boolean | null>(null) // null = not checked yet

  // Check bridge availability on first load
  async function checkBridge() {
    if (isBridgeAvailable.value !== null) return
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetVersion) {
        // Try a lightweight call to verify the bridge
        await app.GetVersion()
        isBridgeAvailable.value = true
      }
    } catch {
      isBridgeAvailable.value = false
    }
  }

  async function fetchSentiment(symbol: string) {
    loading.value = true
    await checkBridge()
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetSentiment && isBridgeAvailable.value) {
        sentiment.value = await app.GetSentiment(symbol)
      } else {
        sentiment.value = null
      }
    } catch (e) {
      console.warn('GetSentiment unavailable:', e)
      sentiment.value = null
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
        research.value = null
      }
    } catch (e) {
      console.warn('GetStockResearch unavailable:', e)
      research.value = null
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
        congressTrades.value = null
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

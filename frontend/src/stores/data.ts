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

export const useDataStore = defineStore('data', () => {
  const quotes = ref<Map<string, QuoteSnapshot>>(new Map())
  const ohlcvCache = ref<Map<string, OHLCVBar[]>>(new Map())
  const sourceStatus = ref<DataSourceStatus[]>([])
  const isOffline = ref(false)

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

  return {
    quotes,
    ohlcvCache,
    sourceStatus,
    isOffline,
    updateQuote,
    getQuote,
    setOHLCV,
    getOHLCV,
    toggleOffline,
  }
})

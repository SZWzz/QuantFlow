import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface PortfolioSummary {
  total_value: number; cash_balance: number; market_value: number
  total_pnl: number; total_pnl_pct: number
}
export interface Allocation {
  by_market: Record<string, number>; by_sector: Record<string, number>; by_currency: Record<string, number>
}
export interface PositionDetail {
  symbol: string; quantity: number; avg_price: number; market_price: number
  pnl: number; pnl_pct: number; market: string; currency: string; cost_basis: number; alloc_pct: number
}

export interface Order {
  order_id: string
  symbol: string
  side: 'buy' | 'sell'
  type: 'market' | 'limit' | 'stop'
  quantity: number
  price: number
  filled_qty: number
  status: 'filled' | 'partial' | 'cancelled' | 'pending' | 'rejected'
  created_at: string
  updated_at: string
}

export interface Trade {
  trade_id: string
  order_id: string
  symbol: string
  side: 'buy' | 'sell'
  price: number
  quantity: number
  value: number
  fee: number
  executed_at: string
}

export interface EquityCurvePoint {
  date: string
  nav: number
  benchmark: number
}

export const usePortfolioStore = defineStore('portfolio', () => {
  const summary = ref<PortfolioSummary | null>(null)
  const allocation = ref<Allocation | null>(null)
  const positions = ref<PositionDetail[]>([])
  const orders = ref<Order[]>([])
  const trades = ref<Trade[]>([])
  const equityCurve = ref<EquityCurvePoint[] | null>(null)
  const error = ref<string | null>(null)

  async function fetchSummary() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      summary.value = await app.GetPortfolioSummary()
    } catch (e) { error.value = String(e) }
  }
  async function fetchAllocation() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      allocation.value = await app.GetPortfolioAllocation()
    } catch (e) { error.value = String(e) }
  }
  async function fetchPositions() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      positions.value = await app.GetPositions()
    } catch (e) { error.value = String(e) }
  }

  async function fetchOrders() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.GetOrders) {
        orders.value = await app.GetOrders()
      } else {
        orders.value = []
      }
    } catch (e) {
      error.value = String(e)
      orders.value = []
    }
  }

  async function fetchTrades() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.GetTrades) {
        trades.value = await app.GetTrades()
      } else {
        trades.value = []
      }
    } catch (e) {
      error.value = String(e)
      trades.value = []
    }
  }

  async function fetchEquityCurve() {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.GetEquityCurve) {
        equityCurve.value = await app.GetEquityCurve()
      } else {
        equityCurve.value = null
      }
    } catch (e) {
      error.value = String(e)
      equityCurve.value = null
    }
  }

  async function cancelOrder(orderId: string) {
    error.value = null
    try {
      const app = (window as any).go?.main?.App
      if (!app) { error.value = 'Bridge unavailable'; return }
      if (app.CancelOrder) {
        await app.CancelOrder(orderId)
      }
      // Update local state optimistically
      const idx = orders.value.findIndex(o => o.order_id === orderId)
      if (idx >= 0) {
        orders.value[idx] = { ...orders.value[idx], status: 'cancelled' }
      }
    } catch (e) {
      error.value = String(e)
    }
  }

  const timer = ref<ReturnType<typeof setInterval> | null>(null)
  function startAutoRefresh() {
    fetchSummary(); fetchAllocation(); fetchPositions()
    timer.value = setInterval(() => { fetchSummary(); fetchAllocation(); fetchPositions() }, 10000)
  }
  function stopAutoRefresh() { if (timer.value) { clearInterval(timer.value); timer.value = null } }
  return {
    summary, allocation, positions, orders, trades, equityCurve, error,
    fetchSummary, fetchAllocation, fetchPositions, fetchOrders, fetchTrades, fetchEquityCurve, cancelOrder,
    startAutoRefresh, stopAutoRefresh, timer,
  }
})

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

  async function fetchSummary() {
    try { summary.value = await (window as any).go.main.App.GetPortfolioSummary() }
    catch (e) { console.warn('GetPortfolioSummary not available:', e) }
  }
  async function fetchAllocation() {
    try { allocation.value = await (window as any).go.main.App.GetPortfolioAllocation() }
    catch (e) { console.warn('GetPortfolioAllocation not available:', e) }
  }
  async function fetchPositions() {
    try { positions.value = await (window as any).go.main.App.GetPositions() }
    catch (e) { console.warn('GetPositions not available:', e) }
  }

  async function fetchOrders() {
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetOrders) {
        orders.value = await app.GetOrders()
      } else {
        orders.value = generateMockOrders()
      }
    } catch (e) {
      console.warn('GetOrders unavailable:', e)
      orders.value = generateMockOrders()
    }
  }

  async function fetchTrades() {
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetTrades) {
        trades.value = await app.GetTrades()
      } else {
        trades.value = generateMockTrades()
      }
    } catch (e) {
      console.warn('GetTrades unavailable:', e)
      trades.value = generateMockTrades()
    }
  }

  async function fetchEquityCurve() {
    try {
      const app = (window as any).go?.main?.App
      if (app?.GetPortfolioSummary) {
        const s = await app.GetPortfolioSummary()
        equityCurve.value = generateMockEquityCurve(s?.total_value ?? 100000)
      } else {
        equityCurve.value = generateMockEquityCurve(100000)
      }
    } catch (e) {
      console.warn('fetchEquityCurve failed:', e)
      equityCurve.value = generateMockEquityCurve(100000)
    }
  }

  async function cancelOrder(orderId: string) {
    try {
      const app = (window as any).go?.main?.App
      if (app?.CancelOrder) {
        await app.CancelOrder(orderId)
      }
      // Update local state optimistically
      const idx = orders.value.findIndex(o => o.order_id === orderId)
      if (idx >= 0) {
        orders.value[idx] = { ...orders.value[idx], status: 'cancelled' }
      }
    } catch (e) {
      console.warn('CancelOrder failed:', e)
    }
  }

  const timer = ref<ReturnType<typeof setInterval> | null>(null)
  function startAutoRefresh() {
    fetchSummary(); fetchAllocation(); fetchPositions()
    timer.value = setInterval(() => { fetchSummary(); fetchAllocation(); fetchPositions() }, 10000)
  }
  function stopAutoRefresh() { if (timer.value) { clearInterval(timer.value); timer.value = null } }
  return {
    summary, allocation, positions, orders, trades, equityCurve,
    fetchSummary, fetchAllocation, fetchPositions, fetchOrders, fetchTrades, fetchEquityCurve, cancelOrder,
    startAutoRefresh, stopAutoRefresh, timer,
  }
})

function generateMockOrders(): Order[] {
  const symbols = ['600519', '000858', '000001', '300750', '002594', 'AAPL', 'TSLA', 'NVDA']
  const sides: Array<'buy' | 'sell'> = ['buy', 'sell']
  const types: Array<'market' | 'limit' | 'stop'> = ['market', 'limit', 'limit', 'market', 'limit', 'stop']
  const statuses: Array<'filled' | 'partial' | 'cancelled' | 'pending' | 'rejected'> = ['filled', 'filled', 'filled', 'partial', 'pending', 'cancelled', 'filled', 'rejected']
  return Array.from({ length: 18 }, (_, i) => {
    const side = sides[i % 2]
    const price = side === 'buy' ? 150 + i * 3 + Math.random() * 20 : 200 - i * 2 + Math.random() * 15
    const qty = Math.floor(Math.random() * 900 + 100)
    const status = statuses[i % statuses.length]
    const filledQty = status === 'filled' ? qty : status === 'partial' ? Math.floor(qty * 0.6) : 0
    return {
      order_id: `ORD-${String(i + 1).padStart(6, '0')}`,
      symbol: symbols[i % symbols.length],
      side,
      type: types[i % types.length],
      quantity: qty,
      price: Math.round(price * 100) / 100,
      filled_qty: filledQty,
      status,
      created_at: new Date(Date.now() - i * 3600000).toISOString(),
      updated_at: new Date(Date.now() - i * 1800000).toISOString(),
    }
  })
}

function generateMockTrades(): Trade[] {
  const symbols = ['600519', '000858', 'AAPL', 'TSLA', 'NVDA', '300750']
  const sides: Array<'buy' | 'sell'> = ['buy', 'sell']
  return Array.from({ length: 35 }, (_, i) => {
    const price = 100 + Math.random() * 200
    const qty = Math.floor(Math.random() * 500 + 10)
    const side = sides[i % 2]
    return {
      trade_id: `TRD-${String(i + 1).padStart(8, '0')}`,
      order_id: `ORD-${String(Math.floor(i / 2) + 1).padStart(6, '0')}`,
      symbol: symbols[i % symbols.length],
      side,
      price: Math.round(price * 100) / 100,
      quantity: qty,
      value: Math.round(price * qty * 100) / 100,
      fee: Math.round(price * qty * 0.0005 * 100) / 100,
      executed_at: new Date(Date.now() - i * 900000).toISOString(),
    }
  })
}

function generateMockEquityCurve(currentValue: number): EquityCurvePoint[] {
  const points: EquityCurvePoint[] = []
  let nav = currentValue * 0.85
  let benchmark = currentValue * 0.88
  const days = 252
  for (let i = days; i >= 0; i--) {
    const date = new Date(Date.now() - i * 86400000).toISOString().slice(0, 10)
    points.push({ date, nav: Math.round(nav * 100) / 100, benchmark: Math.round(benchmark * 100) / 100 })
    nav += nav * (Math.random() - 0.49) * 0.025
    benchmark += benchmark * (Math.random() - 0.495) * 0.018
  }
  // Anchor last point to current value
  const lastActual = points[points.length - 1]
  if (lastActual) {
    const ratio = currentValue / lastActual.nav
    for (const p of points) { p.nav = Math.round(p.nav * ratio * 100) / 100 }
  }
  return points
}

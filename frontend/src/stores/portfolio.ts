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

export const usePortfolioStore = defineStore('portfolio', () => {
  const summary = ref<PortfolioSummary | null>(null)
  const allocation = ref<Allocation | null>(null)
  const positions = ref<PositionDetail[]>([])

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

  const timer = ref<ReturnType<typeof setInterval> | null>(null)
  function startAutoRefresh() {
    fetchSummary(); fetchAllocation(); fetchPositions()
    timer.value = setInterval(() => { fetchSummary(); fetchAllocation(); fetchPositions() }, 10000)
  }
  function stopAutoRefresh() { if (timer.value) { clearInterval(timer.value); timer.value = null } }
  return { summary, allocation, positions, fetchSummary, fetchAllocation, fetchPositions, startAutoRefresh, stopAutoRefresh, timer }
})

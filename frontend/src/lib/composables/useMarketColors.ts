import { detectMarket } from '@/lib/wails'

/** Red in CN, green elsewhere. */
export function marketUpColor(symbol: string): string {
  return detectMarket(symbol) === 'CN' ? '#ef4444' : '#22c55e'
}
/** Green in CN, red elsewhere. */
export function marketDownColor(symbol: string): string {
  return detectMarket(symbol) === 'CN' ? '#22c55e' : '#ef4444'
}
/** Market-aware change color for a numeric changePct value. */
export function marketChangeColor(symbol: string, changePct: number): string {
  if (changePct === 0) return 'var(--color-text-secondary)'
  return changePct > 0 ? marketUpColor(symbol) : marketDownColor(symbol)
}
/** Market-aware polarity: true if changePct is considered "up" for this market. */
export function marketIsUp(symbol: string, changePct: number): boolean {
  if (detectMarket(symbol) === 'CN') return changePct > 0
  return changePct < 0
}

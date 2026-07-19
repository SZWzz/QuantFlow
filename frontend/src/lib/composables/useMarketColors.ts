import { cssVar } from '@/lib/cssVar'

// Fallbacks preserve the previous hardcoded hex values; in the app the
// --color-up / --color-down tokens in themes.css always resolve (they also
// carry the user's 红涨绿跌 / 绿涨红跌 preference via body.color-us).
const FALLBACK_UP = '#ef4444'
const FALLBACK_DOWN = '#22c55e'

/**
 * Up color from the active theme token.
 * Read at call time, so callers that build echarts options get the correct
 * color on every rebuild (useChartTheme triggers rebuilds on theme switch).
 */
export function marketUpColor(_symbol: string): string {
  return cssVar('--color-up', FALLBACK_UP)
}
/** Down color from the active theme token. */
export function marketDownColor(_symbol: string): string {
  return cssVar('--color-down', FALLBACK_DOWN)
}
/** Market-aware change color for a numeric changePct value. */
export function marketChangeColor(symbol: string, changePct: number): string {
  if (changePct === 0) return 'var(--color-text-secondary)'
  return changePct > 0 ? marketUpColor(symbol) : marketDownColor(symbol)
}

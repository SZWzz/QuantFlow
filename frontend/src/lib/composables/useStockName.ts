import { ref, watch, type Ref } from 'vue'

const MIN_SYMBOL_LEN = 4

function looksLikeSymbol(s: string): boolean {
  if (s.length < MIN_SYMBOL_LEN) return false
  if (s.includes('.')) return s.length >= 6
  if (/^[A-Z]{1,5}$/.test(s)) return true
  if (/^\d{4,6}$/.test(s)) return true
  if (/^[A-Z0-9]{6,12}$/.test(s)) return true
  return false
}

export function useStockName(symbol: Ref<string | undefined>) {
  const name = ref('')

  async function fetchName(sym: string) {
    const app = (window as any).go?.main?.App
    if (!app) return
    // Use SearchSymbols first — local SQLite cache, works offline
    if (app.SearchSymbols) {
      try {
        const results = await app.SearchSymbols(sym, 1)
        if (Array.isArray(results) && results.length > 0 && results[0].name) {
          name.value = results[0].name
          return
        }
      } catch { /* fall through to GetQuote */ }
    }
    // Fallback: GetQuote via network adapter
    try {
      const market = /^\d{6}$/.test(sym) || sym.includes('.SH') || sym.includes('.SZ') ? 'CN'
        : sym.includes('.HK') || /^\d{4,5}$/.test(sym) ? 'HK'
        : 'US'
      const result = await app.GetQuote(market, sym)
      const quote = Array.isArray(result) ? result[0] : result
      if (quote?.name) name.value = quote.name
    } catch { /* best-effort */ }
  }

  watch(symbol, (sym) => {
    if (!sym || !looksLikeSymbol(sym)) { name.value = ''; return }
    fetchName(sym)
  }, { immediate: true })

  return { name }
}

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
  let timer: ReturnType<typeof setTimeout> | null = null

  watch(symbol, (sym) => {
    if (timer) clearTimeout(timer)
    if (!sym || !looksLikeSymbol(sym)) { name.value = ''; return }
    timer = setTimeout(async () => {
      try {
        const app = (window as any).go?.main?.App
        if (!app) return
        const market = /^\d{6}$/.test(sym) || sym.includes('.SH') || sym.includes('.SZ') ? 'CN'
          : sym.includes('.HK') || /^\d{4,5}$/.test(sym) ? 'HK'
          : 'US'
        const result = await app.GetQuote(market, sym)
        const quote = Array.isArray(result) ? result[0] : result
        if (quote?.name) name.value = quote.name
      } catch { /* best-effort */ }
    }, 400)
  }, { immediate: true })

  return { name }
}

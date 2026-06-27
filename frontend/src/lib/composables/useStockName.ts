import { ref, watch, type Ref } from 'vue'

export function useStockName(symbol: Ref<string | undefined>) {
  const name = ref('')
  watch(symbol, async (sym) => {
    if (!sym) { name.value = ''; return }
    try {
      const app = (window as any).go?.main?.App
      if (!app) return
      const result = await app.GetQuote('CN', sym)
      const quote = Array.isArray(result) ? result[0] : result
      if (quote?.name) name.value = quote.name
    } catch { /* best-effort name resolution */ }
  }, { immediate: true })
  return { name }
}

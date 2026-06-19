import { ref, watch } from 'vue'

export interface StockEntry {
  code: string
  name: string
  market: string
  pinyin: string
}

/**
 * Composable for debounced symbol search via the Wails Go backend.
 * Usage:
 *   const { query, results, loading } = useSymbolSearch()
 *   query.value = '茅台'  // automatically searches after 200ms debounce
 */
export function useSymbolSearch() {
  const query = ref('')
  const results = ref<StockEntry[]>([])
  const loading = ref(false)
  let timer: ReturnType<typeof setTimeout> | null = null

  async function doSearch(q: string) {
    if (!q || q.trim().length === 0) {
      results.value = []
      loading.value = false
      return
    }

    loading.value = true
    try {
      const app = (window as any).go?.main?.App
      if (app?.SearchSymbols) {
        results.value = await app.SearchSymbols(q.trim())
      } else {
        // Mock fallback for dev without Go backend
        results.value = mockSearch(q.trim())
      }
    } catch (e) {
      console.warn('SearchSymbols failed:', e)
      results.value = []
    } finally {
      loading.value = false
    }
  }

  watch(query, (newVal) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => doSearch(newVal), 200)
  })

  return { query, results, loading }
}

/** Minimal mock for development without the Go backend running. */
function mockSearch(q: string): StockEntry[] {
  const all: StockEntry[] = [
    // CN
    { code: '600519', name: '贵州茅台', market: 'SH', pinyin: 'gzmt' },
    { code: '000001', name: '平安银行', market: 'SZ', pinyin: 'payx' },
    { code: '300750', name: '宁德时代', market: 'SZ', pinyin: 'ndsd' },
    { code: '002594', name: '比亚迪', market: 'SZ', pinyin: 'byd' },
    // HK
    { code: '00700', name: '腾讯控股', market: 'HK', pinyin: 'txkg' },
    { code: '09988', name: '阿里巴巴-SW', market: 'HK', pinyin: 'albb' },
    { code: '00388', name: '香港交易所', market: 'HK', pinyin: 'xgjys' },
    // US
    { code: 'AAPL', name: 'Apple Inc.', market: 'US', pinyin: '' },
    { code: 'TSLA', name: 'Tesla Inc.', market: 'US', pinyin: '' },
    { code: 'MSFT', name: 'Microsoft Corp.', market: 'US', pinyin: '' },
    { code: 'NVDA', name: 'NVIDIA Corp.', market: 'US', pinyin: '' },
  ]
  const ql = q.toLowerCase()
  const qu = q.toUpperCase()
  return all.filter(e =>
    e.code.toUpperCase().startsWith(qu) ||
    e.name.toLowerCase().includes(ql) ||
    e.pinyin.startsWith(ql)
  )
}

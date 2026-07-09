import { ref } from 'vue'
import { logger } from '@/lib/logger'

/**
 * Universal data fetching composable with loading/error/data triad.
 *
 * Usage:
 *   const { data, loading, error, execute } = useDataFetch(() => app.GetQuote(symbol))
 *   onMounted(() => execute())
 */
export function useDataFetch<T>(fetcher: () => Promise<T>) {
  const data = ref<T | null>(null) as ReturnType<typeof ref<T | null>>
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function execute() {
    loading.value = true
    error.value = null
    try {
      data.value = await fetcher()
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      error.value = msg
      logger.error('[useDataFetch]', msg)
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, execute }
}

import { useDataStore } from '@/stores/data'

const DEFAULT_TTL = 5 * 60 * 1000

export function usePanelCache() {
  const dataStore = useDataStore()

  function getCache<T>(key: string): T | null {
    return dataStore.getCached<T>(key)
  }

  function setCache<T>(key: string, data: T, ttlMs = DEFAULT_TTL): void {
    dataStore.setCached(key, data, ttlMs)
  }

  function clearCache(key?: string): void {
    dataStore.clearCached(key)
  }

  async function fetchWithCache<T>(
    key: string,
    fetcher: () => Promise<T>,
    ttlMs = DEFAULT_TTL,
  ): Promise<{ data: T; fromCache: boolean }> {
    const cached = getCache<T>(key)
    if (cached !== null) {
      return { data: cached, fromCache: true }
    }
    const data = await fetcher()
    setCache(key, data, ttlMs)
    return { data, fromCache: false }
  }

  return { getCache, setCache, clearCache, fetchWithCache }
}

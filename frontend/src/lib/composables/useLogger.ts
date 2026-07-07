import { ref, onMounted, onUnmounted } from 'vue'
import { GetLogs, type LogEntry } from '@/lib/wails'

export interface LogFilter {
  levels: Set<string>
  search: string
}

export function useLogger(pollInterval = 1000) {
  const entries = ref<LogEntry[]>([])
  const lastID = ref(0)
  const maxEntries = 2000
  const error = ref<string | null>(null)
  const filter = ref<LogFilter>({ levels: new Set(['info', 'warn', 'error']), search: '' })
  let timer: ReturnType<typeof setInterval> | null = null

  async function poll() {
    try {
      const newEntries: LogEntry[] = await GetLogs(lastID.value)
      if (newEntries && newEntries.length > 0) {
        lastID.value = newEntries[newEntries.length - 1].id
        entries.value.push(...newEntries)
        if (entries.value.length > maxEntries) {
          entries.value = entries.value.slice(entries.value.length - maxEntries)
        }
      }
      error.value = null
    } catch (e) {
      error.value = String(e)
      console.error('[useLogger] poll error:', e)
    }
  }

  function filteredEntries(): LogEntry[] {
    return entries.value.filter(e => {
      if (e.level && !filter.value.levels.has(e.level.toLowerCase())) return false
      if (filter.value.search) {
        const q = filter.value.search.toLowerCase()
        const msg = e.message.toLowerCase()
        const attrs = e.attrs ? JSON.stringify(e.attrs).toLowerCase() : ''
        return msg.includes(q) || attrs.includes(q)
      }
      return true
    })
  }

  function toggleLevel(level: string) {
    const s = new Set(filter.value.levels)
    if (s.has(level)) {
      if (s.size > 1) s.delete(level)
    } else {
      s.add(level)
    }
    filter.value = { ...filter.value, levels: s }
  }

  function setSearch(q: string) {
    filter.value = { ...filter.value, search: q }
  }

  function clear() {
    entries.value = []
    lastID.value = 0
    error.value = null
  }

  onMounted(() => {
    poll()
    timer = setInterval(poll, pollInterval)
  })

  onUnmounted(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })

  return { entries, filter, filteredEntries, toggleLevel, setSearch, clear, poll, error }
}

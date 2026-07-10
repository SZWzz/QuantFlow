import { ref, shallowRef, onUnmounted } from 'vue'
import type { Ref, ShallowRef } from 'vue'
import { useDataStore } from '@/stores/data'
import { useWailsApp, type MinuteTick } from '@/lib/composables/useWailsApp'

function getTodayDateString(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function parseMinuteTimeToUnix(timeStr: string): number {
  const today = getTodayDateString()
  const d = new Date(`${today}T${timeStr}:00+08:00`)
  return Math.floor(d.getTime() / 1000)
}

export function useMinuteChart(
  symbol: Ref<string>,
  prevClose: Ref<number>,
  opts?: { polling?: boolean; pollingInterval?: number },
) {
  const minuteTicks = shallowRef<MinuteTick[]>([])
  const minuteLoading = ref(false)
  let loadSeq = 0
  let minuteTimer: ReturnType<typeof setInterval> | null = null

  async function loadMinuteLine() {
    const seq = ++loadSeq
    const app = useWailsApp()
    if (!app) return
    minuteLoading.value = true
    try {
      const lastTick = minuteTicks.value.length > 0
        ? minuteTicks.value[minuteTicks.value.length - 1]
        : null
      const sinceTimestamp = lastTick
        ? parseMinuteTimeToUnix(lastTick.time)
        : 0

      const dataStore = useDataStore()
      const result = await dataStore.fetchMinuteLine(symbol.value, sinceTimestamp)
      if (seq !== loadSeq) return
      const ticks: MinuteTick[] = result[0]
      if (!Array.isArray(ticks) || ticks.length === 0) return

      if (sinceTimestamp === 0) {
        minuteTicks.value = ticks
        if (ticks.length > 0 && prevClose.value === 0) {
          prevClose.value = ticks[0].price
        }
      } else {
        const existing = new Map(minuteTicks.value.map(t => [t.time, t]))
        for (const t of ticks) {
          existing.set(t.time, t)
        }
        minuteTicks.value = Array.from(existing.values()).sort((a, b) => a.time.localeCompare(b.time))
      }
    } catch (e) {
      console.error('[useMinuteChart] fetch:', e)
    } finally {
      minuteLoading.value = false
    }
  }

  function startPolling() {
    stopPolling()
    if (!opts?.polling) return
    const interval = opts.pollingInterval ?? 5000
    loadMinuteLine()
    minuteTimer = window.setInterval(() => loadMinuteLine(), interval)
  }

  function stopPolling() {
    if (minuteTimer) {
      clearInterval(minuteTimer)
      minuteTimer = null
    }
  }

  onUnmounted(() => stopPolling())

  return { minuteTicks, minuteLoading, loadMinuteLine, startPolling, stopPolling }
}

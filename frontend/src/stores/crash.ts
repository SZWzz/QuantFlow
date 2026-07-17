/**
 * Crash report store.
 *
 * Manages crash reports saved by the Go crash reporter (internal/crash).
 * Because a crash kills the process, the recovery dialog is shown on the
 * NEXT launch: init() lists saved reports and flags the newest one as
 * "pending" if it has not been acknowledged yet (acknowledgement watermark
 * kept in localStorage). It also subscribes to the `crash:detected` Wails
 * event so future in-session crash notifications surface immediately.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  ListCrashReports,
  DeleteCrashReport,
  UploadCrashReport,
  GetCrashDir,
  onCrashReport,
  type CrashReport,
} from '@/lib/wails'
import { logger } from '@/lib/logger'

export type { CrashReport } from '@/lib/wails'

/** localStorage key holding the timestamp of the newest acknowledged crash. */
const ACK_KEY = 'quantflow-crash-acked'

export const useCrashStore = defineStore('crash', () => {
  const reports = ref<CrashReport[]>([])
  const loading = ref(false)
  /** Newest crash report from a previous session not yet shown to the user. */
  const pending = ref<CrashReport | null>(null)
  /** Directory where crash reports are stored (for display in the dialog). */
  const crashDir = ref('')

  let unsubscribe: (() => void) | null = null

  async function list() {
    loading.value = true
    try {
      const result = await ListCrashReports()
      // Newest first for display.
      reports.value = (result ?? []).slice().sort(
        (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
      )
    } catch (e) {
      logger.error('[Crash] list reports failed:', e)
      reports.value = []
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string) {
    try {
      await DeleteCrashReport(id)
      reports.value = reports.value.filter(r => r.id !== id)
      if (pending.value?.id === id) pending.value = null
    } catch (e) {
      logger.error('[Crash] delete report failed:', e)
      throw e
    }
  }

  async function upload(id: string): Promise<boolean> {
    try {
      await UploadCrashReport(id)
      return true
    } catch (e) {
      logger.error('[Crash] upload report failed:', e)
      return false
    }
  }

  /**
   * Detects a pending (unacknowledged) crash from a previous session and
   * subscribes to live crash events. Call once at app startup.
   */
  async function init() {
    try {
      crashDir.value = await GetCrashDir()
    } catch {
      crashDir.value = ''
    }

    if (!unsubscribe) {
      try {
        unsubscribe = onCrashReport((report) => {
          pending.value = report
        })
      } catch (e) {
        logger.error('[Crash] event subscription failed:', e)
      }
    }

    await list()
    const newest = reports.value[0]
    if (!newest) return
    const acked = localStorage.getItem(ACK_KEY)
    if (!acked || new Date(newest.timestamp).getTime() > new Date(acked).getTime()) {
      pending.value = newest
    }
  }

  /** Marks the pending crash as seen so it is not shown again next launch. */
  function ack() {
    if (pending.value) {
      localStorage.setItem(ACK_KEY, pending.value.timestamp)
    } else if (reports.value[0]) {
      localStorage.setItem(ACK_KEY, reports.value[0].timestamp)
    }
    pending.value = null
  }

  function dispose() {
    unsubscribe?.()
    unsubscribe = null
  }

  return { reports, loading, pending, crashDir, list, remove, upload, init, ack, dispose }
})

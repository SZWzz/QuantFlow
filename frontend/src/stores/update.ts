import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Call } from '@wailsio/runtime'
import { useSettingsStore } from '@/stores/settings'

export interface UpdateInfo {
  has_update: boolean
  current_version: string
  latest_version: string
  asset_url: string
  asset_size: number
  checksum: string
  changelog: string
}

export const useUpdateStore = defineStore('update', () => {
  const updateInfo = ref<UpdateInfo | null>(null)
  const checking = ref(false)
  const downloading = ref(false)
  const downloadProgress = ref(0)
  const error = ref('')

  async function check() {
    checking.value = true
    error.value = ''
    try {
      const info = await Call.ByName('main.App.CheckUpdate') as UpdateInfo
      updateInfo.value = info
      return info
    } catch (e: any) {
      error.value = e.message || 'Check failed'
      return null
    } finally {
      checking.value = false
    }
  }

  async function apply() {
    if (!updateInfo.value?.has_update) return
    downloading.value = true
    downloadProgress.value = 0
    error.value = ''
    try {
      await Call.ByName('main.App.ApplyUpdate', updateInfo.value.asset_url, updateInfo.value.checksum)
    } catch (e: any) {
      error.value = e.message || 'Update failed'
    } finally {
      downloading.value = false
    }
  }

  function shouldCheck(): boolean {
    const settings = useSettingsStore()
    const interval = (settings.settings as Record<string, any>)['updateCheckInterval'] || 'daily'
    if (interval === 'never') return false
    if (interval === 'always') return true
    const lastCheck = localStorage.getItem('quantflow-last-update-check')
    if (!lastCheck) return true
    const last = new Date(lastCheck)
    const now = new Date()
    return last.toDateString() !== now.toDateString()
  }

  function markChecked() {
    localStorage.setItem('quantflow-last-update-check', new Date().toISOString())
  }

  return { updateInfo, checking, downloading, downloadProgress, error, check, apply, shouldCheck, markChecked }
})

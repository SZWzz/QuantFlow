import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface SettingsState {
  language: string
  defaultBroker: string
  defaultQty: number
  maxPositionPct: number
  stopLossPct: number
  currency: string
  decimals: number
  dateFormat: string
  telegramToken: string
  telegramChatId: string
  dataSource: string
  cacheTtlDays: number
  fredApiKey: string
  finnhubApiKey: string
  iwencaiApiKey: string
  colorScheme: string  // 'cn' = 红涨绿跌 (A股), 'us' = 绿涨红跌 (美股)
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<SettingsState>(loadSettings())

  function loadSettings(): SettingsState {
    const saved = localStorage.getItem('quantflow-settings')
    if (saved) {
      try {
        return { ...defaultSettings(), ...JSON.parse(saved) }
      } catch {
        return defaultSettings()
      }
    }
    return defaultSettings()
  }

  function defaultSettings(): SettingsState {
    return {
      language: 'zh',
      defaultBroker: 'paper',
      defaultQty: 100,
      maxPositionPct: 25,
      stopLossPct: 5,
      currency: '$',
      decimals: 2,
      dateFormat: 'YYYY-MM-DD',
      telegramToken: '',
      telegramChatId: '',
      dataSource: 'auto',
      cacheTtlDays: 30,
      fredApiKey: '',
      finnhubApiKey: '',
      iwencaiApiKey: '',
      colorScheme: 'cn',
    }
  }

  function save() {
    localStorage.setItem('quantflow-settings', JSON.stringify(settings.value))
  }

  function update(key: keyof SettingsState, value: string | number) {
    ;(settings.value as Record<string, unknown>)[key] = value
    save()
  }

  function reset() {
    settings.value = defaultSettings()
    save()
  }

  return { settings, save, update, loadSettings, reset }
})

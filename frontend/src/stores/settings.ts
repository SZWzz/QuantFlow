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
  colorScheme: string  // 'cn' = 红涨绿跌 (A股), 'us' = 绿涨红跌 (美股)
  llmOpenaiBaseUrl: string
  llmAnthropicBaseUrl: string
  llmDeepseekBaseUrl: string
  llmOllamaBaseUrl: string
  llmGoogleBaseUrl: string
  llmMistralBaseUrl: string
  llmGroqBaseUrl: string
  llmSiliconflowBaseUrl: string
  llmZhipuBaseUrl: string
  llmOpenrouterBaseUrl: string
  llmOpencodeBaseUrl: string
  llmCustomBaseUrl: string
  llmCustomName: string
  llmCustomModels: string
  llmDefaultModel: string
  llmOpenaiConfigured: boolean
  llmAnthropicConfigured: boolean
  llmDeepseekConfigured: boolean
  llmGoogleConfigured: boolean
  llmMistralConfigured: boolean
  llmGroqConfigured: boolean
  llmSiliconflowConfigured: boolean
  llmZhipuConfigured: boolean
  llmOpenrouterConfigured: boolean
  llmOpencodeConfigured: boolean
  llmCustomConfigured: boolean
}

export const useSettingsStore = defineStore('settings', () => {
  const settings = ref<SettingsState>(loadSettings())

  function loadSettings(): SettingsState {
    const saved = localStorage.getItem('quantflow-settings')
    if (saved) {
      try {
        const parsed = JSON.parse(saved)
        const migrated: any = { ...defaultSettings(), ...parsed }
        for (const key of Object.keys(parsed)) {
          if (key.endsWith('Key') && parsed[key]) {
            const provider = key.replace('Key', '')
            migrated[provider + 'Configured'] = true
          }
          if (key.endsWith('Key') || key === 'fredApiKey' || key === 'finnhubApiKey' || key === 'iwencaiApiKey') {
            delete migrated[key]
          }
        }
        return migrated as SettingsState
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
      colorScheme: 'cn',
      llmOpenaiBaseUrl: '',
      llmAnthropicBaseUrl: '',
      llmDeepseekBaseUrl: '',
      llmOllamaBaseUrl: '',
      llmGoogleBaseUrl: '',
      llmMistralBaseUrl: '',
      llmGroqBaseUrl: '',
      llmSiliconflowBaseUrl: '',
      llmZhipuBaseUrl: '',
      llmOpenrouterBaseUrl: '',
      llmOpencodeBaseUrl: '',
      llmCustomBaseUrl: '',
      llmCustomName: '',
      llmCustomModels: '',
      llmDefaultModel: '',
      llmOpenaiConfigured: false,
      llmAnthropicConfigured: false,
      llmDeepseekConfigured: false,
      llmGoogleConfigured: false,
      llmMistralConfigured: false,
      llmGroqConfigured: false,
      llmSiliconflowConfigured: false,
      llmZhipuConfigured: false,
      llmOpenrouterConfigured: false,
      llmOpencodeConfigured: false,
      llmCustomConfigured: false,
    }
  }

  function save() {
    localStorage.setItem('quantflow-settings', JSON.stringify(settings.value))
  }

  function update<K extends keyof SettingsState>(key: K, value: SettingsState[K]) {
    settings.value[key] = value
    save()
  }

  function reset() {
    settings.value = defaultSettings()
    save()
  }

  return { settings, save, update, loadSettings, reset }
})

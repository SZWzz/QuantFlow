import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type MarketKey = 'CN' | 'HK' | 'US'

export interface SessionUI {
  theme: 'light' | 'dark'
  density: 'compact' | 'default' | 'comfortable'
  language: 'zh' | 'en'
  mode: 'terminal' | 'workflow'
  activeMarket: MarketKey
}

const LOCALE_KEY = 'qf-locale' // 与 i18n module 保持一致

export const useSessionStore = defineStore('session', () => {
  const stored = localStorage.getItem('quantflow-session')
  const savedLocale = localStorage.getItem(LOCALE_KEY)
  const defaults: SessionUI = {
    theme: 'light',
    density: 'default',
    language: savedLocale === 'en' ? 'en' : 'zh',
    mode: 'terminal',
    activeMarket: 'CN',
  }

  const saved = stored ? { ...defaults, ...JSON.parse(stored) } : defaults
  const ui = ref<SessionUI>(saved)

  watch(
    ui,
    (val) => {
      localStorage.setItem('quantflow-session', JSON.stringify(val))
      localStorage.setItem(LOCALE_KEY, val.language)
    },
    { deep: true }
  )

  function toggleMode() {
    ui.value.mode = ui.value.mode === 'terminal' ? 'workflow' : 'terminal'
  }

  function setTheme(theme: 'light' | 'dark') {
    ui.value.theme = theme
  }

  function setDensity(density: 'compact' | 'default' | 'comfortable') {
    ui.value.density = density
  }

  function setLanguage(language: 'zh' | 'en') {
    ui.value.language = language
  }

  function setActiveMarket(m: MarketKey) {
    ui.value.activeMarket = m
  }

  const onboardingDone = ref(false)

  function completeOnboarding() {
    onboardingDone.value = true
    localStorage.setItem('quantflow_onboarding_done', 'true')
  }

  function initOnboarding() {
    onboardingDone.value = localStorage.getItem('quantflow_onboarding_done') === 'true'
  }

  return { ui, onboardingDone, completeOnboarding, initOnboarding, toggleMode, setTheme, setDensity, setLanguage, setActiveMarket }
})

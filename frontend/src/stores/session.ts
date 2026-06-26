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

export const useSessionStore = defineStore('session', () => {
  const stored = localStorage.getItem('quantflow-session')
  const defaults: SessionUI = {
    theme: 'dark',
    density: 'default',
    language: 'zh',
    mode: 'terminal',
    activeMarket: 'CN',
  }

  const saved = stored ? { ...defaults, ...JSON.parse(stored) } : defaults
  const ui = ref<SessionUI>(saved)

  watch(
    ui,
    (val) => {
      localStorage.setItem('quantflow-session', JSON.stringify(val))
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

  return { ui, toggleMode, setTheme, setDensity, setLanguage, setActiveMarket }
})

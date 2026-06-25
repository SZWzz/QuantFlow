import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')
  const density = ref<Density>((localStorage.getItem('density') as Density) || 'default')

  function apply() {
    document.documentElement.className = `theme-${theme.value} density-${density.value}`

    // Color scheme: read from settings localStorage (avoid circular store dependency)
    const settings = getSettingsFromLocalStorage()
    const flop = settings.colorScheme === 'us' // true = green up/red down
    document.documentElement.style.setProperty('--color-up', flop ? '#22c55e' : '#ef4444')
    document.documentElement.style.setProperty('--color-down', flop ? '#ef4444' : '#22c55e')
    document.documentElement.style.setProperty('--color-up-soft', flop ? 'rgba(34,197,94,0.12)' : 'rgba(239,68,68,0.12)')
    document.documentElement.style.setProperty('--color-down-soft', flop ? 'rgba(239,68,68,0.12)' : 'rgba(34,197,94,0.12)')
  }

  function applyColorScheme(scheme: string) {
    const flop = scheme === 'us'
    document.documentElement.style.setProperty('--color-up', flop ? '#22c55e' : '#ef4444')
    document.documentElement.style.setProperty('--color-down', flop ? '#ef4444' : '#22c55e')
    document.documentElement.style.setProperty('--color-up-soft', flop ? 'rgba(34,197,94,0.12)' : 'rgba(239,68,68,0.12)')
    document.documentElement.style.setProperty('--color-down-soft', flop ? 'rgba(239,68,68,0.12)' : 'rgba(34,197,94,0.12)')
  }

  function setTheme(t: Theme) {
    theme.value = t
    localStorage.setItem('theme', t)
    apply()
  }

  function setDensity(d: Density) {
    density.value = d
    localStorage.setItem('density', d)
    apply()
  }

  apply()
  return { theme, density, setTheme, setDensity, apply, applyColorScheme }
})

/** Read colorScheme from localStorage settings without importing the settings store. */
function getSettingsFromLocalStorage(): { colorScheme: string } {
  try {
    const raw = localStorage.getItem('quantflow-settings')
    if (raw) return JSON.parse(raw)
  } catch {}
  return { colorScheme: 'cn' }
}

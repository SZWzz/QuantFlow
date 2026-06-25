import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')
  const density = ref<Density>((localStorage.getItem('density') as Density) || 'default')

  function apply() {
    const flop = getColorScheme() === 'us' ? 'color-us' : 'color-cn'
    const cls = `theme-${theme.value} density-${density.value} ${flop}`
    document.documentElement.className = cls
    // Also set on body for resilience
    document.body.className = cls
  }

  function applyColorScheme(scheme: string) {
    const el = document.documentElement
    el.classList.remove('color-cn', 'color-us')
    el.classList.add(scheme === 'us' ? 'color-us' : 'color-cn')
    document.body.classList.remove('color-cn', 'color-us')
    document.body.classList.add(scheme === 'us' ? 'color-us' : 'color-cn')
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
function getColorScheme(): string {
  try {
    const raw = localStorage.getItem('quantflow-settings')
    if (raw) return JSON.parse(raw).colorScheme || 'cn'
  } catch {}
  return 'cn'
}

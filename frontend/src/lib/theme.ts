import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useSessionStore } from '@/stores/session'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

function readLS(key: string, fallback: string): string {
  try { return localStorage.getItem(key) || fallback } catch { return fallback }
}

export const useThemeStore = defineStore('theme', () => {
  const session = useSessionStore()

  const theme = computed<Theme>(() => session.ui.theme as Theme)
  const density = computed<Density>(() => session.ui.density as Density)

  function apply() {
    const body = document.body
    body.classList.remove('theme-dark', 'theme-light', 'color-cn', 'color-us', 'density-default', 'density-compact', 'density-comfortable')
    body.classList.add(`theme-${theme.value}`, `density-${density.value}`)
    const cs = readLS('quantflow-color-scheme', 'cn')
    if (cs === 'us') body.classList.add('color-us')
    else body.classList.add('color-cn')
  }

  function applyColorScheme(scheme: string) {
    try { localStorage.setItem('quantflow-color-scheme', scheme) } catch {}
    apply()
  }

  function setTheme(t: Theme) {
    session.setTheme(t)
    apply()
  }

  function setDensity(d: Density) {
    session.setDensity(d)
    apply()
  }

  apply()
  return { theme, density, setTheme, setDensity, apply, applyColorScheme }
})

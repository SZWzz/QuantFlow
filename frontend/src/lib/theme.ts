import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

function readLS(key: string, fallback: string): string {
  try { return localStorage.getItem(key) || fallback } catch { return fallback }
}

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>(readLS('theme', 'dark') as Theme)
  const density = ref<Density>(readLS('density', 'default') as Density)

  function apply() {
    const body = document.body
    body.classList.remove('theme-dark', 'theme-light', 'color-cn', 'color-us', 'density-default', 'density-compact', 'density-comfortable')
    body.classList.add(`theme-${theme.value}`, `density-${density.value}`)
    // Color scheme from settings or separate key
    const cs = readLS('quantflow-color-scheme', readLS('quantflow-settings-fallback', 'cn'))
    if (cs === 'us') body.classList.add('color-us')
    else body.classList.add('color-cn')
  }

  function applyColorScheme(scheme: string) {
    try { localStorage.setItem('quantflow-color-scheme', scheme) } catch {}
    apply()
  }

  function setTheme(t: Theme) {
    theme.value = t
    try { localStorage.setItem('theme', t) } catch {}
    apply()
  }

  function setDensity(d: Density) {
    density.value = d
    try { localStorage.setItem('density', d) } catch {}
    apply()
  }

  apply()
  return { theme, density, setTheme, setDensity, apply, applyColorScheme }
})
import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')
  const density = ref<Density>((localStorage.getItem('density') as Density) || 'default')

  function apply() {
    document.documentElement.className = `theme-${theme.value} density-${density.value}`
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
  return { theme, density, setTheme, setDensity, apply }
})

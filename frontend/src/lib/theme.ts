import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')
  const density = ref<Density>((localStorage.getItem('density') as Density) || 'default')

  function apply() {
    const t = theme.value
    const cs = getColorScheme()
    const isLight = t === 'light'
    const isCN = cs === 'cn'

    // Direct CSS var override with !important — bypasses any class-based CSS
    const root = document.documentElement

    // Background/Text — light vs dark
    root.style.setProperty('--color-bg-app',    isLight ? '#f1f5f9' : '#0a0e17', 'important')
    root.style.setProperty('--color-bg-panel',  isLight ? '#ffffff' : '#111827', 'important')
    root.style.setProperty('--color-bg-subtle', isLight ? '#f8fafc' : '#1a2332', 'important')
    root.style.setProperty('--color-bg-input',  isLight ? '#f1f5f9' : '#0f1a2a', 'important')
    root.style.setProperty('--color-text-primary',   isLight ? '#0f172a' : '#e2e8f0', 'important')
    root.style.setProperty('--color-text-secondary', isLight ? '#475569' : '#94a3b8', 'important')
    root.style.setProperty('--color-text-tertiary',  isLight ? '#94a3b8' : '#64748b', 'important')
    root.style.setProperty('--color-border',         isLight ? '#e2e8f0' : '#1e293b', 'important')
    root.style.setProperty('--color-accent',         isLight ? '#2563eb' : '#3b82f6', 'important')

    // Up/Down colors
    root.style.setProperty('--color-up',   isCN ? '#ef4444' : '#22c55e', 'important')
    root.style.setProperty('--color-down', isCN ? '#22c55e' : '#ef4444', 'important')

    // Also set legacy aliases
    root.style.setProperty('--bg',    isLight ? '#f1f5f9' : '#0a0e17', 'important')
    root.style.setProperty('--card',  isLight ? '#ffffff' : '#111827', 'important')
    root.style.setProperty('--input', isLight ? '#f1f5f9' : '#0f1a2a', 'important')
    root.style.setProperty('--text',  isLight ? '#0f172a' : '#e2e8f0', 'important')
    root.style.setProperty('--muted', isLight ? '#475569' : '#94a3b8', 'important')
    root.style.setProperty('--up',    isCN ? '#ef4444' : '#22c55e', 'important')
    root.style.setProperty('--down',  isCN ? '#22c55e' : '#ef4444', 'important')
    root.style.setProperty('--accent', isLight ? '#2563eb' : '#3b82f6', 'important')
    root.style.setProperty('--border', isLight ? '#e2e8f0' : '#1e293b', 'important')
  }

  function applyColorScheme(scheme: string) {
    localStorage.setItem('quantflow-color-scheme', scheme)
    apply()
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

function getColorScheme(): string {
  try {
    const raw = localStorage.getItem('quantflow-settings')
    if (raw) return JSON.parse(raw).colorScheme || 'cn'
  } catch {}
  return 'cn'
}

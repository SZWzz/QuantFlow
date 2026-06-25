import { defineStore } from 'pinia'
import { ref } from 'vue'

export type Theme = 'dark' | 'light'
export type Density = 'compact' | 'default' | 'comfortable'

let _styleEl: HTMLStyleElement | null = null

function ensureStyleEl(): HTMLStyleElement {
  if (!_styleEl) {
    _styleEl = document.createElement('style')
    _styleEl.id = 'quantflow-theme-override'
    document.head.appendChild(_styleEl)
  }
  return _styleEl
}

function buildCSS(t: string, cs: string): string {
  const isLight = t === 'light'
  const isCN = cs === 'cn'
  const up = isCN ? '#ef4444' : '#22c55e'
  const down = isCN ? '#22c55e' : '#ef4444'
  return `
:root, body, #app {
  --color-bg-app: ${isLight ? '#f1f5f9' : '#0a0e17'};
  --color-bg-panel: ${isLight ? '#ffffff' : '#111827'};
  --color-bg-subtle: ${isLight ? '#f8fafc' : '#1a2332'};
  --color-bg-input: ${isLight ? '#f1f5f9' : '#0f1a2a'};
  --color-text-primary: ${isLight ? '#0f172a' : '#e2e8f0'};
  --color-text-secondary: ${isLight ? '#475569' : '#94a3b8'};
  --color-text-tertiary: ${isLight ? '#94a3b8' : '#64748b'};
  --color-border: ${isLight ? '#e2e8f0' : '#1e293b'};
  --color-accent: ${isLight ? '#2563eb' : '#3b82f6'};
  --color-up: ${up};
  --color-up-soft: ${isCN ? 'rgba(239,68,68,0.12)' : 'rgba(34,197,94,0.12)'};
  --color-down: ${down};
  --color-down-soft: ${isCN ? 'rgba(34,197,94,0.12)' : 'rgba(239,68,68,0.12)'};
  --bg: ${isLight ? '#f1f5f9' : '#0a0e17'};
  --card: ${isLight ? '#ffffff' : '#111827'};
  --input: ${isLight ? '#f1f5f9' : '#0f1a2a'};
  --text: ${isLight ? '#0f172a' : '#e2e8f0'};
  --muted: ${isLight ? '#475569' : '#94a3b8'};
  --up: ${up};
  --down: ${down};
  --accent: ${isLight ? '#2563eb' : '#3b82f6'};
  --border: ${isLight ? '#e2e8f0' : '#1e293b'};
  --hover: ${isLight ? 'rgba(0,0,0,0.03)' : 'rgba(255,255,255,0.04)'};
}
  `.trim()
}

export const useThemeStore = defineStore('theme', () => {
  const theme = ref<Theme>((localStorage.getItem('theme') as Theme) || 'dark')
  const density = ref<Density>((localStorage.getItem('density') as Density) || 'default')

  function apply() {
    const t = theme.value
    const cs = getColorScheme()
    const el = ensureStyleEl()
    el.textContent = buildCSS(t, cs)
  }

  function applyColorScheme(scheme: string) {
    localStorage.setItem('quantflow-color-scheme', scheme)
    // Also directly update settings store in localStorage
    try {
      const raw = localStorage.getItem('quantflow-settings')
      if (raw) {
        const s = JSON.parse(raw)
        s.colorScheme = scheme
        localStorage.setItem('quantflow-settings', JSON.stringify(s))
      }
    } catch {}
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

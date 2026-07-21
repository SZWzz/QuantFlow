import { reactive, onUnmounted } from 'vue'

export interface ChartThemeColors {
  textColor: string
  axisColor: string
  splitColor: string
  bgColor: string
  crossColor: string
  tooltipBg: string
  tooltipText: string
  upColor: string
  downColor: string
  gridColor: string
  palette: string[]
}

const root = typeof document !== 'undefined' ? document.documentElement : null
const body = typeof document !== 'undefined' ? document.body : null

// Fallbacks mirror the light theme values in assets/themes.css
const FALLBACK_PALETTE = ['#1d64d8', '#2e7d32', '#b45309', '#c62828', '#6d28d9', '#0e7490']

function fallbackTheme(): ChartThemeColors {
  return {
    textColor: '#333333',
    axisColor: '#888780',
    splitColor: '#e8e8e8',
    bgColor: '#ffffff',
    crossColor: '#e24b4a',
    tooltipBg: '#ffffff',
    tooltipText: '#333333',
    upColor: '#c62828',
    downColor: '#2e7d32',
    gridColor: 'rgba(16, 24, 40, 0.06)',
    palette: [...FALLBACK_PALETTE],
  }
}

function readTheme(): ChartThemeColors {
  if (!root) {
    return fallbackTheme()
  }
  try {
    const s = (v: string) => getComputedStyle(root).getPropertyValue(v).trim() || getComputedStyle(body!).getPropertyValue(v).trim()
    return {
      textColor: s('--color-text-primary') || '#333333',
      axisColor: s('--color-text-tertiary') || '#888780',
      splitColor: s('--color-border-subtle') || '#e8e8e8',
      bgColor: s('--color-bg-elevated') || '#ffffff',
      crossColor: s('--color-danger') || '#e24b4a',
      tooltipBg: s('--color-bg-glass') || '#ffffff',
      tooltipText: s('--color-text-primary') || '#333333',
      upColor: s('--color-up') || '#c62828',
      downColor: s('--color-down') || '#2e7d32',
      gridColor: s('--chart-grid') || 'rgba(16, 24, 40, 0.06)',
      palette: [1, 2, 3, 4, 5, 6].map(i => s(`--chart-${i}`) || FALLBACK_PALETTE[i - 1]),
    }
  } catch {
    return fallbackTheme()
  }
}

let globalTheme: ChartThemeColors | null = null
let subscribers: (() => void)[] = []
let observer: MutationObserver | null = null

function ensureObserver() {
  if (observer || !body) return
  observer = new MutationObserver(() => {
    globalTheme = readTheme()
    subscribers.forEach(fn => fn())
  })
  observer.observe(body, { attributes: true, attributeFilter: ['class'] })
}

export function useChartTheme(): ChartThemeColors {
  if (!globalTheme) globalTheme = readTheme()
  ensureObserver()

  const theme = reactive<ChartThemeColors>({ ...globalTheme })
  const update = () => { Object.assign(theme, globalTheme) }
  subscribers.push(update)

  onUnmounted(() => {
    subscribers = subscribers.filter(fn => fn !== update)
  })

  return theme
}

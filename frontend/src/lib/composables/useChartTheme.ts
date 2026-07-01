import { reactive, onUnmounted } from 'vue'

export interface ChartThemeColors {
  textColor: string
  axisColor: string
  splitColor: string
  bgColor: string
  crossColor: string
  tooltipBg: string
  tooltipText: string
}

const root = typeof document !== 'undefined' ? document.documentElement : null
const body = typeof document !== 'undefined' ? document.body : null

function readTheme(): ChartThemeColors {
  if (!root) {
    return { textColor: '#333333', axisColor: '#888780', splitColor: '#e8e8e8', bgColor: '#ffffff', crossColor: '#e24b4a', tooltipBg: '#ffffff', tooltipText: '#333333' }
  }
  try {
    const s = (v: string) => getComputedStyle(root).getPropertyValue(v).trim() || getComputedStyle(body!).getPropertyValue(v).trim()
    return {
      textColor: s('--color-text-primary') || '#333333',
      axisColor: s('--color-text-tertiary') || '#888780',
      splitColor: s('--color-border-subtle') || '#e8e8e8',
      bgColor: s('--color-bg-elevated') || '#ffffff',
      crossColor: s('--color-error') || '#e24b4a',
      tooltipBg: s('--color-bg-glass') || '#ffffff',
      tooltipText: s('--color-text-primary') || '#333333',
    }
  } catch {
    return { textColor: '#333333', axisColor: '#888780', splitColor: '#e8e8e8', bgColor: '#ffffff', crossColor: '#e24b4a', tooltipBg: '#ffffff', tooltipText: '#333333' }
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

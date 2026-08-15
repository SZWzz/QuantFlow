import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { useChartTheme } from '@/lib/composables/useChartTheme'

function mountTheme() {
  let theme: ReturnType<typeof useChartTheme>
  const Comp = defineComponent({
    setup() {
      theme = useChartTheme()
      return () => null
    },
  })
  mount(Comp)
  return theme!
}

describe('useChartTheme', () => {
  beforeEach(() => {
    document.body.style.cssText = `
      --color-text-primary: #111111;
      --color-text-tertiary: #555555;
      --color-border-subtle: #eeeeee;
      --color-bg-elevated: #ffffff;
      --color-danger: #cc0000;
      --color-bg-glass: rgba(255,255,255,0.9);
      --color-up: #c62828;
      --color-down: #2e7d32;
      --chart-grid: #dddddd;
      --chart-1: #1d64d8; --chart-2: #2e7d32; --chart-3: #b45309;
      --chart-4: #c62828; --chart-5: #6d28d9; --chart-6: #0e7490;
    `
  })

  it('reads colors from CSS variables', () => {
    const theme = mountTheme()
    expect(theme.textColor).toBe('#111111')
    expect(theme.crossColor).toBe('#cc0000')
  })

  it('exposes up/down colors, grid and 6-color palette', () => {
    const theme = mountTheme()
    expect(theme.upColor).toBe('#c62828')
    expect(theme.downColor).toBe('#2e7d32')
    expect(theme.gridColor).toBe('#dddddd')
    expect(theme.palette).toHaveLength(6)
    expect(theme.palette[0]).toBe('#1d64d8')
  })
})

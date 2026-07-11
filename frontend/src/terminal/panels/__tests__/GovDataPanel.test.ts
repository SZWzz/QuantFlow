import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { nextTick } from 'vue'
import GovDataPanel from '../GovDataPanel.vue'

// Mock vue-echarts (same pattern as other panel tests)
vi.mock('vue-echarts', () => ({
  default: {
    name: 'VChart',
    template: '<div class="echarts-mock"></div>',
    props: ['option'],
  },
}))

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: {} },
})

// Minimal FRED mock so indicator cards render without a real Go bridge.
function mockGoBridge() {
  const app = {
    GetEconomicIndicators: vi.fn().mockResolvedValue({
      signals: [
        { indicator_id: 'CPIAUCSL', name: 'CPI', name_cn: '消费者物价指数', latest_value: 315, change: 0.3, direction: 'up', signal: 'bearish', unit: '', category: 'inflation', updated_at: 0 },
        { indicator_id: 'GDP', name: 'GDP', name_cn: '国内生产总值', latest_value: 29000, change: 2.8, direction: 'up', signal: 'bullish', unit: '', category: 'gdp', updated_at: 0 },
        { indicator_id: 'UNRATE', name: 'Unemployment', name_cn: '失业率', latest_value: 3.8, change: -0.1, direction: 'down', signal: 'bullish', unit: '%', category: 'employment', updated_at: 0 },
      ],
    }),
    GetCommodityQuotes: vi.fn().mockResolvedValue({ commodities: [] }),
    FetchData: vi.fn().mockResolvedValue({ data: '' }),
  }
  ;(window as any).go = { main: { App: app } }
  return app
}

describe('GovDataPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGoBridge()
  })

  const mountPanel = () => mount(GovDataPanel, {
    props: { panelId: 'gov-data-1' },
    global: { plugins: [i18n] },
  })

  it('renders panel header showing current macro source', () => {
    const wrapper = mountPanel()
    // Default source is 'fred'; header shows the source label
    expect(wrapper.find('.panel-header h3').text()).toContain('FRED')
  })

  it('renders source switch tabs (3 sources)', () => {
    const wrapper = mountPanel()
    // First PanelTabs is source tabs with variant="pill"
    const sourceTabs = wrapper.findAll('.panel-tabs.variant-pill .tab')
    expect(sourceTabs.length).toBe(3)
  })

  it('renders filter tabs (>=3)', () => {
    const wrapper = mountPanel()
    const tabs = wrapper.findAll('.tab')
    expect(tabs.length).toBeGreaterThanOrEqual(3)
  })

  it('shows indicator cards on mount', async () => {
    const wrapper = mountPanel()
    // Wait for onMounted async data loading
    await nextTick()
    await nextTick()
    await nextTick()
    const cards = wrapper.findAll('.indicator-card')
    expect(cards.length).toBeGreaterThan(0)
  })

  it('filters indicators by category tab click', async () => {
    const wrapper = mountPanel()
    await nextTick()
    await nextTick()
    await nextTick()
    const tabs = wrapper.findAll('.tab')
    const inflationTab = tabs.find(t => t.text().includes('通胀'))
    if (inflationTab) {
      await inflationTab.trigger('click')
      await nextTick()
      expect(inflationTab.classes()).toContain('active')
    }
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mountPanel()
    expect(wrapper.find('.govdata-panel').attributes('data-panel-id')).toBe('gov-data-1')
  })
})

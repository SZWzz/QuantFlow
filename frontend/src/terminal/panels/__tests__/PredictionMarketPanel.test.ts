import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { mockWailsIPC } from '@/test-utils/mocks'
import PredictionMarketPanel from '../PredictionMarketPanel.vue'

// Mock vue-echarts (same pattern as other panel tests)
vi.mock('vue-echarts', () => ({
  default: {
    name: 'VChart',
    template: '<div class="echarts-mock"></div>',
    props: ['option'],
  },
}))

describe('PredictionMarketPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('renders panel header with 预测市场', () => {
    const wrapper = mount(PredictionMarketPanel, {
      props: { panelId: 'prediction-market-1' },
    })
    expect(wrapper.find('.panel-header h3').text()).toContain('预测市场')
  })

  it('renders category tabs (>=4)', () => {
    const wrapper = mount(PredictionMarketPanel, {
      props: { panelId: 'prediction-market-1' },
    })
    const tabs = wrapper.findAll('.tab')
    expect(tabs.length).toBeGreaterThanOrEqual(4)
  })

  it('shows mock events on mount', async () => {
    const wrapper = mount(PredictionMarketPanel, {
      props: { panelId: 'prediction-market-1' },
    })
    // Wait for onMounted async data loading
    await nextTick()
    await nextTick()
    await nextTick()
    const rows = wrapper.findAll('tbody tr')
    // After mount, mock data loads — should have event rows (not loading state)
    expect(rows.length).toBeGreaterThan(0)
  })

  it('switches category on tab click', async () => {
    const wrapper = mount(PredictionMarketPanel, {
      props: { panelId: 'prediction-market-1' },
    })
    const tabs = wrapper.findAll('.tab')
    const cryptoTab = tabs.find(t => t.text().includes('加密'))
    if (cryptoTab) {
      await cryptoTab.trigger('click')
      await nextTick()
      // After click, crypto tab should be active
      expect(cryptoTab.classes()).toContain('active')
    }
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mount(PredictionMarketPanel, {
      props: { panelId: 'test-panel-id' },
    })
    expect(wrapper.find('.prediction-market-panel').attributes('data-panel-id')).toBe('test-panel-id')
  })
})

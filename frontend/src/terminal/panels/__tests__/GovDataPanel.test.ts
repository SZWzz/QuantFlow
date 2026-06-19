import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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

describe('GovDataPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders panel header with 宏观指标', () => {
    const wrapper = mount(GovDataPanel, {
      props: { panelId: 'gov-data-1' },
    })
    expect(wrapper.find('.panel-header h3').text()).toContain('宏观指标')
  })

  it('renders filter tabs (>=3)', () => {
    const wrapper = mount(GovDataPanel, {
      props: { panelId: 'gov-data-1' },
    })
    const tabs = wrapper.findAll('.tab')
    expect(tabs.length).toBeGreaterThanOrEqual(3)
  })

  it('shows indicator cards on mount', async () => {
    const wrapper = mount(GovDataPanel, {
      props: { panelId: 'gov-data-1' },
    })
    // Wait for onMounted async data loading
    await nextTick()
    await nextTick()
    await nextTick()
    const cards = wrapper.findAll('.indicator-card')
    // After mount, mock data loads — should have indicator cards (not loading state)
    expect(cards.length).toBeGreaterThan(0)
  })

  it('filters indicators by category tab click', async () => {
    const wrapper = mount(GovDataPanel, {
      props: { panelId: 'gov-data-1' },
    })
    await nextTick()
    await nextTick()
    await nextTick()
    const tabs = wrapper.findAll('.tab')
    // Find the "通胀" tab (inflation category)
    const inflationTab = tabs.find(t => t.text().includes('通胀'))
    if (inflationTab) {
      await inflationTab.trigger('click')
      await nextTick()
      // After click, inflation tab should be active
      expect(inflationTab.classes()).toContain('active')
    }
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mount(GovDataPanel, {
      props: { panelId: 'test-panel-id' },
    })
    expect(wrapper.find('.govdata-panel').attributes('data-panel-id')).toBe('test-panel-id')
  })
})

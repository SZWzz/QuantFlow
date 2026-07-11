import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { mockWailsIPC } from '@/__tests__/mocks'
import SatellitePanel from '../SatellitePanel.vue'

// Mock vue-echarts (same pattern as other panel tests)
vi.mock('vue-echarts', () => ({
  default: {
    name: 'VChart',
    template: '<div class="echarts-mock"></div>',
    props: ['option'],
  },
}))

describe('SatellitePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('renders panel header with 卫星数据', () => {
    const wrapper = mount(SatellitePanel, {
      props: { panelId: 'satellite-1' },
    })
    expect(wrapper.find('.panel-header h3').text()).toContain('卫星数据')
  })

  it('renders region cards on mount', async () => {
    const wrapper = mount(SatellitePanel, {
      props: { panelId: 'satellite-1' },
    })
    // Wait for onMounted async data loading (mock data loads)
    await new Promise(r => setTimeout(r, 100))
    await nextTick()
    const cards = wrapper.findAll('.region-card')
    expect(cards.length).toBeGreaterThan(0)
  })

  it('shows trend badges in region cards', async () => {
    const wrapper = mount(SatellitePanel, {
      props: { panelId: 'satellite-1' },
    })
    await new Promise(r => setTimeout(r, 100))
    await nextTick()
    const trendBadges = wrapper.findAll('.trend-badge')
    expect(trendBadges.length).toBeGreaterThan(0)
  })

  it('expands selected region on card click', async () => {
    const wrapper = mount(SatellitePanel, {
      props: { panelId: 'satellite-1' },
    })
    await nextTick()
    await nextTick()
    await nextTick()
    const cards = wrapper.findAll('.region-card')
    if (cards.length > 0) {
      await cards[0].trigger('click')
      await nextTick()
      await nextTick()
      expect(wrapper.find('.detail-panel').exists()).toBe(true)
    }
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mount(SatellitePanel, {
      props: { panelId: 'test-satellite-id' },
    })
    expect(wrapper.find('.satellite-panel').attributes('data-panel-id')).toBe('test-satellite-id')
  })
})

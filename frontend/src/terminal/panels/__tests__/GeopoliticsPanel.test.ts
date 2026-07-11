import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { mockWailsIPC } from '@/__tests__/mocks'
import GeopoliticsPanel from '../GeopoliticsPanel.vue'

// Mock vue-echarts (same pattern as other panel tests)
vi.mock('vue-echarts', () => ({
  default: {
    name: 'VChart',
    template: '<div class="echarts-mock"></div>',
    props: ['option'],
  },
}))

describe('GeopoliticsPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('renders panel header with 地缘政治风险', () => {
    const wrapper = mount(GeopoliticsPanel, {
      props: { panelId: 'geopolitics-1' },
    })
    expect(wrapper.find('.panel-header h3').text()).toContain('地缘政治风险')
  })

  it('renders filter tabs (>=3)', () => {
    const wrapper = mount(GeopoliticsPanel, {
      props: { panelId: 'geopolitics-1' },
    })
    const tabs = wrapper.findAll('.tab')
    expect(tabs.length).toBeGreaterThanOrEqual(3)
  })

  it('shows topic cards on mount', async () => {
    const wrapper = mount(GeopoliticsPanel, {
      props: { panelId: 'geopolitics-1' },
    })
    // Wait for onMounted async data loading
    await nextTick()
    await nextTick()
    await nextTick()
    const cards = wrapper.findAll('.topic-card')
    // After mount, mock data loads — should have topic cards (not loading state)
    expect(cards.length).toBeGreaterThan(0)
  })

  it('filters by risk level on tab click', async () => {
    const wrapper = mount(GeopoliticsPanel, {
      props: { panelId: 'geopolitics-1' },
    })
    await nextTick()
    await nextTick()
    await nextTick()

    const tabs = wrapper.findAll('.tab')
    const highTab = tabs.find(t => t.text().includes('高风险'))
    if (highTab) {
      await highTab.trigger('click')
      await nextTick()
      // After click, high risk tab should be active
      expect(highTab.classes()).toContain('active')
      // Should show cards
      const cards = wrapper.findAll('.topic-card')
      // High risk filter should show at least 1 card (we have 3 high risk in mock)
      expect(cards.length).toBeGreaterThan(0)
      // All visible cards should have high risk badge
      for (const card of cards) {
        expect(card.find('.badge-high').exists()).toBe(true)
      }
    }
  })

  it('has data-panel-id attribute', () => {
    const wrapper = mount(GeopoliticsPanel, {
      props: { panelId: 'test-geopolitics-id' },
    })
    expect(wrapper.find('.geopolitics-panel').attributes('data-panel-id')).toBe('test-geopolitics-id')
  })
})

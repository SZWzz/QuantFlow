import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import MarketOverviewPanel from '../MarketOverviewPanel.vue'

describe('MarketOverviewPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    expect(wrapper.text()).toContain('Market Overview')
  })

  it('renders index cards after data loads', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    const cards = wrapper.findAll('.index-card')
    expect(cards.length).toBeGreaterThanOrEqual(1)
  })

  it('renders breadth section', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    expect(wrapper.find('.breadth-section').exists()).toBe(true)
  })
})

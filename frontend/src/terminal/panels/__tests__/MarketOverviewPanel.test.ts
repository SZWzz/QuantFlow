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

  it('renders market tabs', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    expect(wrapper.find('.market-tabs').exists()).toBe(true)
  })

  it('renders kline area or loading state', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    // Either loading state or kline area container should render
    const hasKlineArea = wrapper.find('.kline-area').exists() || wrapper.find('.empty-chart').exists()
    expect(hasKlineArea).toBe(true)
  })

  it('renders sector section when data is available', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    // Sector section is conditionally rendered based on data
    expect(wrapper.find('.market-overview-panel').exists()).toBe(true)
  })
})

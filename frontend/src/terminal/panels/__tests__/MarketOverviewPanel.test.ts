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

  it('renders PanelShell in loading state initially', () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    // PanelShell shows loading skeleton before data fetch completes
    expect(wrapper.find('.panel-shell-loading').exists()).toBe(true)
  })

  it('renders PanelShell wrapper', async () => {
    const wrapper = mount(MarketOverviewPanel, {
      props: { panelId: 'test-overview', params: {} },
    })
    await nextTick()
    // PanelShell root is always rendered regardless of state
    expect(wrapper.find('.panel-shell').exists()).toBe(true)
  })
})

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MarketDepthPanel from '../MarketDepthPanel.vue'

describe('MarketDepthPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(MarketDepthPanel, {
      props: { panelId: 'test-depth', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(MarketDepthPanel, {
      props: { panelId: 'test-depth', params: {} },
    })
    expect(wrapper.text()).toContain('Market Depth')
  })

  it('has symbol input', () => {
    const wrapper = mount(MarketDepthPanel, {
      props: { panelId: 'test-depth', params: {} },
    })
    const input = wrapper.find('.symbol-input')
    expect(input.exists()).toBe(true)
  })

  it('renders order book rows', () => {
    const wrapper = mount(MarketDepthPanel, {
      props: { panelId: 'test-depth', params: {} },
    })
    const rows = wrapper.findAll('.ob-row')
    expect(rows.length).toBeGreaterThanOrEqual(1)
  })
})

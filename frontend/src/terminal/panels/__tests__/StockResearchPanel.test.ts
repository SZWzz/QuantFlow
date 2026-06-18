import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import StockResearchPanel from '../StockResearchPanel.vue'

describe('StockResearchPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(StockResearchPanel, {
      props: { panelId: 'test-research', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(StockResearchPanel, {
      props: { panelId: 'test-research', params: {} },
    })
    expect(wrapper.text()).toContain('Stock Research')
  })

  it('renders tab buttons', () => {
    const wrapper = mount(StockResearchPanel, {
      props: { panelId: 'test-research', params: {} },
    })
    const tabs = wrapper.findAll('.tab-btn')
    expect(tabs.length).toBeGreaterThanOrEqual(6)
  })

  it('has symbol input', () => {
    const wrapper = mount(StockResearchPanel, {
      props: { panelId: 'test-research', params: {} },
    })
    const input = wrapper.find('.symbol-input')
    expect(input.exists()).toBe(true)
  })
})

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import OrderBlotterPanel from '../OrderBlotterPanel.vue'

describe('OrderBlotterPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(OrderBlotterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('renders title text', () => {
    const wrapper = mount(OrderBlotterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Order')
  })

  it('renders filter bar and table', () => {
    const wrapper = mount(OrderBlotterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.find('select.filter-select').exists()).toBe(true)
    expect(wrapper.find('input.filter-input').exists()).toBe(true)
    expect(wrapper.find('table').exists()).toBe(true)
  })

  it('renders stats footer', () => {
    const wrapper = mount(OrderBlotterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Orders Today')
    expect(wrapper.text()).toContain('Fill Rate')
    expect(wrapper.text()).toContain('Total Traded Value')
  })
})

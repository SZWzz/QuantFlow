import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BrokerStatusPanel from '../BrokerStatusPanel.vue'

describe('BrokerStatusPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('renders title text', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    // Cards should show broker names
    expect(wrapper.text()).toContain('Paper Trading')
  })

  it('renders broker cards', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const cards = wrapper.findAll('.broker-card')
    expect(cards.length).toBe(6)
    expect(wrapper.text()).toContain('Futu')
    expect(wrapper.text()).toContain('Binance')
    expect(wrapper.text()).toContain('Alpaca')
    expect(wrapper.text()).toContain('IBKR')
    expect(wrapper.text()).toContain('OKX')
  })

  it('renders Test Connection buttons', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const btns = wrapper.findAll('.test-btn')
    expect(btns.length).toBe(6)
  })

  it('shows Not Configured badge on future brokers', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Not Configured')
  })
})

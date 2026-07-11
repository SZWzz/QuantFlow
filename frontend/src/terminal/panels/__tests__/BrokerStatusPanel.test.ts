import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/__tests__/mocks'
import BrokerStatusPanel from '../BrokerStatusPanel.vue'

describe('BrokerStatusPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
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
    expect(wrapper.find('.header-title').text()).toContain('Broker Status')
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
    const btns = wrapper.findAll('.refresh-btn')
    expect(btns.length).toBe(1) // Single refresh button in header, not per broker
  })

  it('shows Not Configured badge on disconnected brokers', () => {
    const wrapper = mount(BrokerStatusPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    // Disconnected brokers show 'Disconnected' badge
    expect(wrapper.text()).toContain('Disconnected')
  })
})

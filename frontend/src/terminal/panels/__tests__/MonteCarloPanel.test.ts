import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import MonteCarloPanel from '../MonteCarloPanel.vue'

describe('MonteCarloPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.text()).toContain('蒙特卡洛模拟')
  })

  it('renders parameter inputs', () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    const inputs = wrapper.findAll('.param-input')
    expect(inputs.length).toBe(6)
  })

  it('renders Run Simulation button', () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.run-btn').exists()).toBe(true)
  })

  it('shows empty state initially', () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('shows stats after running simulation', async () => {
    const wrapper = mount(MonteCarloPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    // Click run button
    await wrapper.find('.run-btn').trigger('click')
    // Wait a tick for computation
    await wrapper.vm.$nextTick()
    await new Promise(r => setTimeout(r, 20))
    await wrapper.vm.$nextTick()
    // After simulation, stats should appear
    const cards = wrapper.findAll('.stat-card')
    expect(cards.length).toBe(5)
    expect(wrapper.text()).toContain('Median Terminal')
    expect(wrapper.text()).toContain('95% VaR')
    expect(wrapper.text()).toContain('95% CVaR')
  })
})

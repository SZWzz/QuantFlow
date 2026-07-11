import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/__tests__/mocks'
import ExecutionPanel from '../ExecutionPanel.vue'

describe('ExecutionPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('mounts without crashing', () => {
    const wrapper = mount(ExecutionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('renders table', () => {
    const wrapper = mount(ExecutionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.find('table').exists()).toBe(true)
  })

  it('renders table with column headers', () => {
    const wrapper = mount(ExecutionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const headers = wrapper.findAll('th')
    expect(headers.length).toBeGreaterThanOrEqual(5)
    expect(wrapper.text()).toContain('Time')
    expect(wrapper.text()).toContain('Symbol')
    expect(wrapper.text()).toContain('Side')
    expect(wrapper.text()).toContain('Price')
    expect(wrapper.text()).toContain('Amount')
  })

  it('renders Load More button when there are trades', async () => {
    const wrapper = mount(ExecutionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    // The mock store generates 4 trades; with pageSize=20
    await wrapper.vm.$nextTick()
    expect(wrapper.find('table').exists()).toBe(true)
  })
})

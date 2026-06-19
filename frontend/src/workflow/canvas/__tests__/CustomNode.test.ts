import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CustomNode from '../CustomNode.vue'

describe('CustomNode', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing with required props', () => {
    const wrapper = mount(CustomNode, {
      props: {
        id: 'test',
        data: {
          label: 'Test',
          nodeType: 'sma',
          params: {},
          inputs: [],
          outputs: [],
          status: 'idle',
        },
      },
      global: {
        stubs: {
          Handle: true,
          VChart: true,
          echarts: true,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('should mount with running status', () => {
    const wrapper = mount(CustomNode, {
      props: {
        id: 'test-running',
        data: {
          label: 'Running Node',
          nodeType: 'data_loader',
          params: { symbol: 'AAPL' },
          inputs: ['input'],
          outputs: ['output'],
          status: 'running',
        },
      },
      global: {
        stubs: {
          Handle: true,
          VChart: true,
          echarts: true,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should mount with failed status', () => {
    const wrapper = mount(CustomNode, {
      props: {
        id: 'test-failed',
        data: {
          label: 'Failed Node',
          nodeType: 'cross_signal',
          params: {},
          inputs: [],
          outputs: [],
          status: 'failed',
          error: 'Something went wrong',
        },
      },
      global: {
        stubs: {
          Handle: true,
          VChart: true,
          echarts: true,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

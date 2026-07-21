import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/test-utils/mocks'
import DistributionPanel from '../DistributionPanel.vue'

describe('DistributionPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('mounts without crashing', () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.panel-title').text()).toContain('收益率分布')
  })

  it('renders placeholder before compute', () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.find('.empty-state').exists()).toBe(true)
  })

  it('computes and shows stats after button click', async () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    const btn = wrapper.find('.btn-primary')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('Mean')
    expect(wrapper.text()).toContain('Std Dev')
    expect(wrapper.text()).toContain('Skewness')
    expect(wrapper.text()).toContain('Kurtosis')
    expect(wrapper.text()).toContain('Jarque-Bera')
  })
})

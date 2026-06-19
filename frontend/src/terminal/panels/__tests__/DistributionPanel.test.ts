import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import DistributionPanel from '../DistributionPanel.vue'

describe('DistributionPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
    expect(wrapper.text()).toContain('Return Distribution')
  })

  it('renders placeholder before compute', () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    expect(wrapper.text()).toContain('Enter a symbol and click Compute')
  })

  it('computes and shows stats after button click', async () => {
    const wrapper = mount(DistributionPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true } },
    })
    const btn = wrapper.find('button.compute-btn')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    await nextTick()
    // After compute, stats should appear (Mean, Std Dev, etc.)
    expect(wrapper.text()).toContain('Mean')
    expect(wrapper.text()).toContain('Std Dev')
    expect(wrapper.text()).toContain('Skewness')
    expect(wrapper.text()).toContain('Kurtosis')
    expect(wrapper.text()).toContain('Jarque-Bera')
  })
})

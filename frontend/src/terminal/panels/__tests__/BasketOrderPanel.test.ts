import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BasketOrderPanel from '../BasketOrderPanel.vue'

describe('BasketOrderPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(BasketOrderPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('renders title text', () => {
    const wrapper = mount(BasketOrderPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Basket')
    expect(wrapper.text()).toContain('Summary')
    expect(wrapper.text()).toContain('Execution Log')
  })

  it('has Add Row button that works', async () => {
    const wrapper = mount(BasketOrderPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const addBtn = wrapper.find('.row-actions .action-btn')
    expect(addBtn.exists()).toBe(true)

    const initialInputs = wrapper.findAll('.cell-input')
    const initialCount = initialInputs.length

    await addBtn.trigger('click')
    const newInputs = wrapper.findAll('.cell-input')
    expect(newInputs.length).toBeGreaterThan(initialCount)
  })

  it('has execute button', () => {
    const wrapper = mount(BasketOrderPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const execBtn = wrapper.find('.execute-btn')
    expect(execBtn.exists()).toBe(true)
    expect(execBtn.text()).toContain('Execute Basket')
  })
})

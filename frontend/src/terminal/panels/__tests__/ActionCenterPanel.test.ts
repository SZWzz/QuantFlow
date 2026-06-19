import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ActionCenterPanel from '../ActionCenterPanel.vue'

describe('ActionCenterPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })

  it('renders title', () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Action Center')
  })

  it('renders event items', () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const cards = wrapper.findAll('.event-card')
    expect(cards.length).toBe(12)
  })

  it('displays event types', () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.text()).toContain('Stop-Loss Triggered')
    expect(wrapper.text()).toContain('Take-Profit Triggered')
    expect(wrapper.text()).toContain('Dividend Announcement')
    expect(wrapper.text()).toContain('Stock Split')
    expect(wrapper.text()).toContain('Large Order Pending')
  })

  it('dismisses an event when Dismiss is clicked', async () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const initialCount = wrapper.findAll('.event-card').length
    const dismissBtns = wrapper.findAll('.dismiss-btn')
    expect(dismissBtns.length).toBeGreaterThan(0)

    await dismissBtns[0].trigger('click')
    const newCount = wrapper.findAll('.event-card').length
    expect(newCount).toBe(initialCount - 1)
  })

  it('approves a large-order event', async () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    const approveBtn = wrapper.find('.approve-btn')
    expect(approveBtn.exists()).toBe(true)
    await approveBtn.trigger('click')
    // The button should now show Confirmed
    expect(wrapper.text()).toContain('Confirmed')
  })
})

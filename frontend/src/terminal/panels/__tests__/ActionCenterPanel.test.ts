import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/__tests__/mocks'
import ActionCenterPanel from '../ActionCenterPanel.vue'

describe('ActionCenterPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
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
    expect(wrapper.find('.ac-title').text()).toContain('Action Center')
  })

  it('renders event cards from trades', async () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    // Wait for onMounted async fetch
    await new Promise(r => setTimeout(r, 50))
    const cards = wrapper.findAll('.event-card')
    expect(cards.length).toBeGreaterThanOrEqual(1)
  })

  it('dismisses an event when dismiss is clicked', async () => {
    const wrapper = mount(ActionCenterPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    await new Promise(r => setTimeout(r, 50))
    const initialCount = wrapper.findAll('.event-card').length
    const dismissBtns = wrapper.findAll('.dismiss-btn')
    if (dismissBtns.length > 0) {
      await dismissBtns[0].trigger('click')
      const newCount = wrapper.findAll('.event-card').length
      expect(newCount).toBe(initialCount - 1)
    }
  })
})

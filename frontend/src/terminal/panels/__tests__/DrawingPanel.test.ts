import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DrawingPanel from '../DrawingPanel.vue'

describe('DrawingPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.text()).toContain('Drawing Tools')
  })

  it('renders symbol badge from params', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: { symbol: '000001' } },
    })
    expect(wrapper.text()).toContain('000001')
  })

  it('renders tool buttons', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    const buttons = wrapper.findAll('.tool-btn')
    expect(buttons.length).toBeGreaterThanOrEqual(5)
  })

  it('renders canvas element', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    const canvas = wrapper.find('canvas')
    expect(canvas.exists()).toBe(true)
  })

  it('activates tool on button click', async () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    const buttons = wrapper.findAll('.tool-btn')
    // Click the trendline button (second button, index 1)
    await buttons[1].trigger('click')
    const activeBtns = wrapper.findAll('.tool-btn.active')
    expect(activeBtns.length).toBe(1)
  })

  it('renders clear button', () => {
    const wrapper = mount(DrawingPanel, {
      props: { panelId: 'test', params: {} },
    })
    expect(wrapper.find('.clear-btn').exists()).toBe(true)
  })
})

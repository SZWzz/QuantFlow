import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LogPanel from '../LogPanel.vue'

describe('LogPanel', () => {
  it('renders without crashing', () => {
    const wrapper = mount(LogPanel, {
      props: { panelId: 'log-viewer' },
      global: {
        stubs: {
          transition: false,
          'transition-group': false,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无日志')
  })

  it('displays level filter buttons', () => {
    const wrapper = mount(LogPanel, {
      props: { panelId: 'log-viewer' },
    })
    const buttons = wrapper.findAll('.log-level-btn')
    expect(buttons.length).toBe(4)
    expect(buttons[0].text()).toBe('DEBUG')
    expect(buttons[1].text()).toBe('INFO')
    expect(buttons[2].text()).toBe('WARN')
    expect(buttons[3].text()).toBe('ERROR')
  })
})

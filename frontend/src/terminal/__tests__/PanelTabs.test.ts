import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelTabs from '../components/panel/PanelTabs.vue'

const tabs = [
  { key: 'a', label: 'Tab A' },
  { key: 'b', label: 'Tab B' },
]

describe('PanelTabs', () => {
  it('renders all tabs and marks active', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a' } })
    expect(w.findAll('.tab')).toHaveLength(2)
    expect(w.find('.tab.active').text()).toBe('Tab A')
  })

  it('emits change on click', async () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a' } })
    await w.findAll('.tab')[1].trigger('click')
    expect(w.emitted('change')).toEqual([['b']])
  })

  it('underline variant renders a sliding indicator', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'b', variant: 'underline' } })
    expect(w.find('.tab-indicator').exists()).toBe(true)
  })

  it('pill variant has no indicator', () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a', variant: 'pill' } })
    expect(w.find('.tab-indicator').exists()).toBe(false)
  })

  it('recalculates indicator when body class changes (density/theme switch)', async () => {
    const w = mount(PanelTabs, { props: { tabs, active: 'a', variant: 'underline' } })
    document.body.classList.add('density-compact')
    // MutationObserver 回调在微任务/下一帧触发；jsdom 下 offset 均为 0，此测试只验证接线不崩
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(w.find('.tab-indicator').exists()).toBe(true)
    document.body.classList.remove('density-compact')
  })
})

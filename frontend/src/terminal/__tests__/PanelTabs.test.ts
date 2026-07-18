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
})

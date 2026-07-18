import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelHeader from '../components/panel/PanelHeader.vue'

describe('PanelHeader', () => {
  it('renders title and subtitle', () => {
    const w = mount(PanelHeader, { props: { title: '自选股', subtitle: '12 只' } })
    expect(w.find('.panel-title').text()).toBe('自选股')
    expect(w.find('.panel-subtitle').text()).toBe('12 只')
  })

  it('renders underline tabs and forwards tabChange', async () => {
    const w = mount(PanelHeader, {
      props: { title: 'T', tabs: [{ key: 'a', label: 'A' }, { key: 'b', label: 'B' }], activeTab: 'a' },
    })
    const tabs = w.findComponent({ name: 'PanelTabs' })
    expect(tabs.exists()).toBe(true)
    expect(tabs.props('variant')).toBe('underline')
    await tabs.vm.$emit('change', 'b')
    expect(w.emitted('tabChange')).toEqual([['b']])
  })

  it('renders controls and triggers action', async () => {
    let called = 0
    const w = mount(PanelHeader, {
      props: { title: 'T', controls: [{ icon: 'refresh', title: '刷新', action: () => { called++ } }] },
    })
    await w.find('.header-controls button').trigger('click')
    expect(called).toBe(1)
  })

  it('omits sections when props absent', () => {
    const w = mount(PanelHeader, { props: { title: 'T' } })
    expect(w.find('.header-controls').exists()).toBe(false)
  })
})

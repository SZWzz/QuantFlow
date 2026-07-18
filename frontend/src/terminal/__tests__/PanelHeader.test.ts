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

  it('renders #controls slot content after props controls', () => {
    const w = mount(PanelHeader, {
      props: { title: 'T', controls: [{ icon: 'refresh', title: '刷新', action: () => {} }] },
      slots: { controls: '<select class="market-select"><option>A股</option></select>' },
    })
    expect(w.find('.market-select').exists()).toBe(true)
    expect(w.find('.header-controls button').exists()).toBe(true)
  })

  it('renders #extra slot as a full-width second row', () => {
    const w = mount(PanelHeader, {
      props: { title: 'T' },
      slots: { extra: '<span class="signal-summary">3 看涨</span>' },
    })
    const extra = w.find('.header-extra')
    expect(extra.exists()).toBe(true)
    expect(extra.find('.signal-summary').text()).toBe('3 看涨')
  })

  it('omits .header-extra when no extra slot provided', () => {
    const w = mount(PanelHeader, { props: { title: 'T' } })
    expect(w.find('.header-extra').exists()).toBe(false)
  })
})

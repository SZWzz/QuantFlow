import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DockTab from '../DockTab.vue'
import type { DockTabState } from '../types'

describe('DockTab', () => {
  it('should mount with tabs data and active tab', () => {
    const tabs: DockTabState[] = [
      { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' },
      { id: 't2', panelId: 'quote', label: 'Quote', icon: '📈' },
    ]
    const wrapper = mount(DockTab, {
      props: { tabs, activeTab: 't1' },
      global: { stubs: { Component: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should mount with empty tabs', () => {
    const wrapper = mount(DockTab, {
      props: { tabs: [], activeTab: '' },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

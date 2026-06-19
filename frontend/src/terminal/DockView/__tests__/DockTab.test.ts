import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DockTab from '../DockTab.vue'
import type { DockTabState } from '../types'

describe('DockTab', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount with tabs data and active tab', () => {
    const tabs: DockTabState[] = [
      { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' },
      { id: 't2', panelId: 'quote', label: 'Quote', icon: '📈' },
    ]
    const wrapper = mount(DockTab, {
      props: { tabs, activeTab: 't1', leafId: 'test-leaf' },
      global: { stubs: { Component: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should mount with empty tabs', () => {
    const wrapper = mount(DockTab, {
      props: { tabs: [], activeTab: '', leafId: 'test-leaf' },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('shows close button for each tab', () => {
    const tabs: DockTabState[] = [
      { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' },
    ]
    const wrapper = mount(DockTab, {
      props: { tabs, activeTab: 't1', leafId: 'test-leaf' },
      global: { stubs: { Component: true } },
    })
    expect(wrapper.find('.tab-close').exists()).toBe(true)
  })
})

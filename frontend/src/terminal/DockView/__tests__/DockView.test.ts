import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useTerminalStore } from '@/stores/terminal'
import DockView from '../DockView.vue'

describe('DockView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    // Initialize store with welcome layout
    const store = useTerminalStore()
    store.layout.value = {
      id: 'root',
      type: 'tab',
      tabs: [{ id: 'welcome', panelId: 'welcome', label: 'Welcome', icon: '🏠' }],
      activeTab: 'welcome',
    }
  })

  it('should mount with default layout', () => {
    const wrapper = mount(DockView, {
      global: { stubs: { DockContainer: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

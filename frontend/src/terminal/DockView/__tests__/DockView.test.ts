import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DockView from '../DockView.vue'
import { createTabLeaf } from '../types'

describe('DockView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount with a simple layout', () => {
    const layout = createTabLeaf('root', { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' })
    const wrapper = mount(DockView, {
      props: { layout },
      global: { stubs: { DockContainer: true, DockTab: true, DockSplitter: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

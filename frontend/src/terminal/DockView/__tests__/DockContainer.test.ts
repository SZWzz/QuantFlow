import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DockContainer from '../DockContainer.vue'
import { createContainer, createTabLeaf } from '../types'

describe('DockContainer', () => {
  it('should mount with a container layout (row)', () => {
    const node = createContainer('root', 'row', [
      createTabLeaf('left', { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' }),
      createTabLeaf('right', { id: 't2', panelId: 'quote', label: 'Quote', icon: '📈' }),
    ])
    const wrapper = mount(DockContainer, {
      props: { node },
      global: { stubs: { DockContainer: true, DockTab: true, DockSplitter: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should mount with a container layout (column)', () => {
    const node = createContainer('root', 'column', [
      createTabLeaf('top', { id: 't1', panelId: 'watchlist', label: 'Watch', icon: '📊' }),
      createTabLeaf('bottom', { id: 't2', panelId: 'quote', label: 'Quote', icon: '📈' }),
    ])
    const wrapper = mount(DockContainer, {
      props: { node },
      global: { stubs: { DockContainer: true, DockTab: true, DockSplitter: true } },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

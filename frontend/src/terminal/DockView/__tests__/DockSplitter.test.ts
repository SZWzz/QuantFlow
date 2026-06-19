import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DockSplitter from '../DockSplitter.vue'

describe('DockSplitter', () => {
  it('should mount with direction row', () => {
    const wrapper = mount(DockSplitter, {
      props: { direction: 'row', index: 0, ratios: [0.5, 0.5] },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('should mount with direction column', () => {
    const wrapper = mount(DockSplitter, {
      props: { direction: 'column', index: 1, ratios: [0.3, 0.4, 0.3] },
    })
    expect(wrapper.exists()).toBe(true)
  })
})

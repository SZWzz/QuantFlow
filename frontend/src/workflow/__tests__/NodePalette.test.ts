import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import NodePalette from '../NodePalette.vue'

vi.mock('@/lib/wails', () => ({
  ListNodes: vi.fn().mockResolvedValue([
    { node_type: 'sma', category: 'indicator' },
    { node_type: 'ema', category: 'indicator' },
  ]),
}))

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      workflow: { node_palette: 'Node Palette' },
      common: { search: 'Search', no_data: 'No data' },
    },
  },
})

describe('NodePalette', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', async () => {
    const wrapper = mount(NodePalette, {
      global: { plugins: [i18n], stubs: { VChart: true, echarts: true } },
    })
    // Wait for the async ListNodes call to resolve
    await new Promise((r) => setTimeout(r, 50))
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})

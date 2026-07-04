import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BacktestPanel from '../BacktestPanel.vue'

describe('BacktestPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(BacktestPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, PanelHeader: true, PanelTable: true, PanelCard: true, EmptyState: true, LoadingState: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})

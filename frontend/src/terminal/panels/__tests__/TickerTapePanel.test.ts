import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { mockWailsIPC, mockI18n } from '@/__tests__/mocks'
import TickerTapePanel from '../TickerTapePanel.vue'

mockI18n()

describe('TickerTapePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
  })

  it('mounts without crashing', () => {
    const wrapper = mount(TickerTapePanel, {
      props: { panelId: 'test-ticker', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(TickerTapePanel, {
      props: { panelId: 'test-ticker', params: {} },
    })
    expect(wrapper.find('.tape-title').text()).toContain('Ticker Tape')
  })

  it('has scroll animation class', async () => {
    const wrapper = mount(TickerTapePanel, {
      props: { panelId: 'test-ticker', params: {} },
    })
    // Wait for async data fetch
    await new Promise(r => setTimeout(r, 100))
    const track = wrapper.find('.tape-track')
    expect(track.exists()).toBe(true)
  })
})

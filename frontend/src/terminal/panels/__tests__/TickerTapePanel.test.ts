import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TickerTapePanel from '../TickerTapePanel.vue'

describe('TickerTapePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
    expect(wrapper.text()).toContain('Ticker Tape')
  })

  it('contains stock symbols', async () => {
    const wrapper = mount(TickerTapePanel, {
      props: { panelId: 'test-ticker', params: {} },
    })
    await new Promise(r => setTimeout(r, 5))
    const html = wrapper.html()
    expect(html).toContain('600519')
    expect(html).toContain('贵州茅台')
  })

  it('has scroll animation class', () => {
    const wrapper = mount(TickerTapePanel, {
      props: { panelId: 'test-ticker', params: {} },
    })
    const track = wrapper.find('.tape-track')
    expect(track.exists()).toBe(true)
  })
})

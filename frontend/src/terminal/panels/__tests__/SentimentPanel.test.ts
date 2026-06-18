import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import SentimentPanel from '../SentimentPanel.vue'

describe('SentimentPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    expect(wrapper.text()).toContain('Sentiment Analysis')
  })

  it('shows mock banner when bridge unavailable', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    const banner = wrapper.find('.mock-banner')
    expect(banner.exists()).toBe(true)
  })

  it('has symbol input', () => {
    const wrapper = mount(SentimentPanel, {
      props: { panelId: 'test-sentiment', params: {} },
    })
    const input = wrapper.find('.symbol-input')
    expect(input.exists()).toBe(true)
  })
})

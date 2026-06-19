import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import CryptoOverviewPanel from '../CryptoOverviewPanel.vue'

describe('CryptoOverviewPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('mounts without crashing', () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders title', () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    expect(wrapper.text()).toContain('Crypto Overview')
  })

  it('renders crypto table rows', async () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    await nextTick()
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBeGreaterThanOrEqual(10)
  })

  it('shows BTC dominance', async () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    await nextTick()
    expect(wrapper.text()).toContain('BTC Dominance')
  })

  it('contains BTC and ETH in table', async () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    await nextTick()
    expect(wrapper.text()).toContain('BTC')
    expect(wrapper.text()).toContain('ETH')
  })
})

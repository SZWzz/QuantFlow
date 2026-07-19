import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { mockWailsIPC } from '@/test-utils/mocks'
import CryptoOverviewPanel from '../CryptoOverviewPanel.vue'

describe('CryptoOverviewPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockWailsIPC()
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
    expect(wrapper.find('.panel-header h3').text()).toContain('Crypto Overview')
  })

  it('renders crypto table rows', async () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    // Wait for async data fetch to complete
    await new Promise(r => setTimeout(r, 50))
    await nextTick()
    const rows = wrapper.findAll('.table-row')
    expect(rows.length).toBeGreaterThanOrEqual(2)
  })

  it('contains BTC and ETH in table', async () => {
    const wrapper = mount(CryptoOverviewPanel, {
      props: { panelId: 'test-crypto', params: {} },
    })
    await new Promise(r => setTimeout(r, 50))
    await nextTick()
    expect(wrapper.text()).toContain('BTC')
    expect(wrapper.text()).toContain('ETH')
  })
})

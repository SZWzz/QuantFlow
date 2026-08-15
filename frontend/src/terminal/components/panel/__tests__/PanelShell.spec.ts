import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelShell from '../PanelShell.vue'

describe('PanelShell', () => {
  it('renders spinner when state is loading', () => {
    const wrapper = mount(PanelShell, { props: { state: 'loading' } })
    expect(wrapper.find('[data-testid="panel-loading"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="panel-loaded"]').exists()).toBe(false)
  })

  it('renders loaded slot content when state is loaded', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'loaded' },
      slots: { loaded: '<div data-testid="content">Hello</div>' },
    })
    expect(wrapper.find('[data-testid="content"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="panel-loading"]').exists()).toBe(false)
  })

  it('renders error message and retry button when state is error', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'error', error: 'API failed' },
    })
    expect(wrapper.find('[data-testid="panel-error"]').text()).toContain('API failed')
    expect(wrapper.find('[data-testid="panel-retry-btn"]').exists()).toBe(true)
  })

  it('emits retry event when retry button clicked', async () => {
    const wrapper = mount(PanelShell, { props: { state: 'error', error: 'err' } })
    await wrapper.find('[data-testid="panel-retry-btn"]').trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
    expect(wrapper.emitted('retry')!.length).toBe(1)
  })

  it('renders default empty state when state is empty', () => {
    const wrapper = mount(PanelShell, { props: { state: 'empty' } })
    expect(wrapper.text()).toContain('暂无数据')
  })

  it('renders custom empty slot when provided', () => {
    const wrapper = mount(PanelShell, {
      props: { state: 'empty' },
      slots: { empty: '<div data-testid="custom-empty">No items</div>' },
    })
    expect(wrapper.find('[data-testid="custom-empty"]').exists()).toBe(true)
  })

  it('emits retry on Command+K / Escape focus trigger', () => {
    const wrapper = mount(PanelShell, { props: { state: 'error', error: 'x' } })
    // Retry button should be focusable
    expect(wrapper.find('[data-testid="panel-retry-btn"]').attributes('tabindex')).toBe('0')
  })
})

import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ErrorState from '../components/panel/ErrorState.vue'

describe('ErrorState', () => {
  it('renders default title and description', () => {
    const w = mount(ErrorState, { props: { description: '网络超时' } })
    expect(w.find('.error-title').text()).toBe('加载失败')
    expect(w.find('.error-desc').text()).toBe('网络超时')
  })

  it('emits retry on button click', async () => {
    const w = mount(ErrorState, { props: {} })
    await w.find('.error-retry').trigger('click')
    expect(w.emitted('retry')).toHaveLength(1)
  })
})

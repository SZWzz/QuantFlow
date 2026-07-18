import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from '../components/panel/EmptyState.vue'

describe('EmptyState', () => {
  it('renders title/description and no entrance animation class', () => {
    const w = mount(EmptyState, { props: { title: '暂无数据', description: '稍后再试' } })
    expect(w.find('.empty-title').text()).toBe('暂无数据')
    expect(w.find('.empty-desc').text()).toBe('稍后再试')
    expect(w.html()).not.toContain('empty-enter')
  })

  it('renders actions and triggers handler', async () => {
    let n = 0
    const w = mount(EmptyState, {
      props: { title: '空', actions: [{ label: '去添加', primary: true, handler: () => { n++ } }] },
    })
    await w.find('.empty-actions .btn-primary').trigger('click')
    expect(n).toBe(1)
  })
})

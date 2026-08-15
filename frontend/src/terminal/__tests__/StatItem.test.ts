import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatItem from '../components/panel/StatItem.vue'

describe('StatItem', () => {
  it('renders label and value', () => {
    const w = mount(StatItem, { props: { label: '总市值', value: '1,234.5亿' } })
    expect(w.find('.stat-label').text()).toBe('总市值')
    expect(w.find('.stat-value').text()).toBe('1,234.5亿')
  })

  it('renders positive delta with badge-up and plus sign', () => {
    const w = mount(StatItem, { props: { label: '盈亏', value: '100', delta: 1.234 } })
    const badge = w.find('.stat-delta')
    expect(badge.text()).toBe('+1.23%')
    expect(badge.classes()).toContain('badge-up')
  })

  it('renders negative delta with badge-down', () => {
    const w = mount(StatItem, { props: { label: '盈亏', value: '-50', delta: -0.5 } })
    const badge = w.find('.stat-delta')
    expect(badge.text()).toBe('-0.50%')
    expect(badge.classes()).toContain('badge-down')
  })

  it('omits delta badge when not provided', () => {
    const w = mount(StatItem, { props: { label: 'A', value: '1' } })
    expect(w.find('.stat-delta').exists()).toBe(false)
  })
})

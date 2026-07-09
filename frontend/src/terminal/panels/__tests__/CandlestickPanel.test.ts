import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CandlestickPanel from '../CandlestickPanel.vue'

function computeDataKey(ticks: { time: string; price: number }[]): string {
  if (ticks.length === 0) return '0|'
  const last = ticks[ticks.length - 1]
  return `${ticks.length}|${last.time}|${last.price}`
}

describe('CandlestickPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(CandlestickPanel, {
      props: { panelId: 'test', params: {} },
      global: { stubs: { VChart: true, echarts: true } },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})

describe('computeDataKey', () => {
  it('should return same key when data unchanged', () => {
    const ticks = [
      { time: '09:30', price: 100 },
      { time: '09:31', price: 101 },
    ]
    const key1 = computeDataKey(ticks)
    const key2 = computeDataKey([...ticks])
    expect(key1).toBe(key2)
  })

  it('should return different key when last price changes', () => {
    const key1 = computeDataKey([{ time: '09:31', price: 101 }])
    const key2 = computeDataKey([{ time: '09:31', price: 102 }])
    expect(key1).not.toBe(key2)
  })

  it('should return different key when new tick appended', () => {
    const key1 = computeDataKey([{ time: '09:30', price: 100 }])
    const key2 = computeDataKey([
      { time: '09:30', price: 100 },
      { time: '09:31', price: 101 },
    ])
    expect(key1).not.toBe(key2)
  })

  it('should handle empty array', () => {
    expect(computeDataKey([])).toBe('0|')
  })
})
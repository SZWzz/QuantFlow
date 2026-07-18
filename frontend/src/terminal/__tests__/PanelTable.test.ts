import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PanelTable from '../components/panel/PanelTable.vue'
import type { Column } from '../components/panel/types'

const cols: Column[] = [
  { key: 'name', label: '名称' },
  { key: 'price', label: '现价', align: 'right', format: 'price' },
  { key: 'chg', label: '涨跌', align: 'right', format: 'percent', colorize: true },
]
const data = [
  { name: '平安银行', price: 12.345, chg: 1.234 },
  { name: '万科A', price: 8.5, chg: -0.876 },
]

describe('PanelTable', () => {
  it('formats price/percent with sign', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const text = w.text()
    expect(text).toContain('12.35')
    expect(text).toContain('+1.23%')
    expect(text).toContain('-0.88%')
  })

  it('applies mono class automatically to numeric formats', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const firstRowTds = w.findAll('.table-row')[0].findAll('.td')
    expect(firstRowTds[0].classes()).not.toContain('mono')
    expect(firstRowTds[1].classes()).toContain('mono')
  })

  it('colorize paints up/down by sign', () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    const chgCells = w.findAll('.td.colorize')
    expect(chgCells[0].attributes('style')).toContain('var(--color-up)')
    expect(chgCells[1].attributes('style')).toContain('var(--color-down)')
  })

  it('shows loading state when loading and empty', () => {
    const w = mount(PanelTable, { props: { columns: cols, data: [], loading: true } })
    expect(w.find('.loading-state').exists()).toBe(true)
  })

  it('emits rowClick', async () => {
    const w = mount(PanelTable, { props: { columns: cols, data, clickable: true } })
    await w.findAll('.table-row')[0].trigger('click')
    expect(w.emitted('rowClick')).toHaveLength(1)
  })
})

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

  it('applies rowClass result to the row element', () => {
    const w = mount(PanelTable, {
      props: { columns: cols, data, rowClass: (row: any) => (row.chg > 0 ? 'flash-up' : 'flash-down') },
    })
    const rows = w.findAll('.table-row')
    expect(rows[0].classes()).toContain('flash-up')
    expect(rows[1].classes()).toContain('flash-down')
  })

  it('emits rowContextmenu with row and native event', async () => {
    const w = mount(PanelTable, { props: { columns: cols, data } })
    await w.findAll('.table-row')[1].trigger('contextmenu')
    const ev = w.emitted('rowContextmenu')
    expect(ev).toHaveLength(1)
    expect(ev![0][0]).toEqual(data[1])
    expect(ev![0][1]).toBeInstanceOf(MouseEvent)
  })

  const sortCols: Column[] = [
    { key: 'name', label: '名称' },
    { key: 'price', label: '现价', align: 'right', format: 'price', sortable: true },
  ]

  it('sortable th click emits sortChange: asc → desc → clear (old watchlist semantics)', async () => {
    // New column starts 'asc'
    const w = mount(PanelTable, { props: { columns: sortCols, data } })
    await w.findAll('.th')[1].trigger('click')
    expect(w.emitted('sortChange')![0]).toEqual(['price', 'asc'])

    // Same column flips asc → desc, then desc → cleared (null)
    await w.setProps({ sortKey: 'price', sortDir: 'asc' })
    await w.findAll('.th')[1].trigger('click')
    expect(w.emitted('sortChange')![1]).toEqual(['price', 'desc'])
    await w.setProps({ sortDir: 'desc' })
    await w.findAll('.th')[1].trigger('click')
    expect(w.emitted('sortChange')![2]).toEqual(['price', null])
  })

  it('shows sort arrow on the active sortable column', () => {
    const w = mount(PanelTable, { props: { columns: sortCols, data, sortKey: 'price', sortDir: 'desc' } })
    const ths = w.findAll('.th')
    expect(ths[0].find('.sort-arrow').exists()).toBe(false)
    expect(ths[1].find('.sort-arrow').exists()).toBe(true)
    expect(ths[1].find('.sort-arrow').text()).toBe('↓')
  })

  it('non-sortable th click does not emit sortChange', async () => {
    const w = mount(PanelTable, { props: { columns: sortCols, data } })
    await w.findAll('.th')[0].trigger('click')
    expect(w.emitted('sortChange')).toBeUndefined()
  })

  it('hideHeader removes the header row', () => {
    const withHeader = mount(PanelTable, { props: { columns: cols, data } })
    expect(withHeader.find('.table-header-row').exists()).toBe(true)
    const withoutHeader = mount(PanelTable, { props: { columns: cols, data, hideHeader: true } })
    expect(withoutHeader.find('.table-header-row').exists()).toBe(false)
  })

  it('applies sticky class to header when stickyHeader', () => {
    const w = mount(PanelTable, { props: { columns: cols, data, stickyHeader: true } })
    expect(w.find('.table-header-row').classes()).toContain('sticky')
  })

  it('colorize does not paint non-numeric placeholder', () => {
    const w = mount(PanelTable, { props: { columns: cols, data: [{ name: 'X', price: 1, chg: undefined }] } })
    const chgCell = w.findAll('.td.colorize')[0]
    expect(chgCell.attributes('style') ?? '').not.toContain('var(--color-down)')
  })

  it('mono:false overrides numeric format auto-mono', () => {
    const c2: Column[] = [{ key: 'v', label: 'V', format: 'price', mono: false }]
    const w = mount(PanelTable, { props: { columns: c2, data: [{ v: 1.5 }] } })
    expect(w.find('.td').classes()).not.toContain('mono')
  })

  it('renders col.title hook as td title attribute; columns without hook have none', () => {
    const c: Column[] = [
      { key: 'name', label: '名称', title: (row: any) => `full:${row.name}` },
      { key: 'price', label: '现价', align: 'right' },
    ]
    const w = mount(PanelTable, { props: { columns: c, data } })
    const tds = w.findAll('.table-row')[0].findAll('.td')
    expect(tds[0].attributes('title')).toBe('full:平安银行')
    expect(tds[1].attributes('title')).toBeUndefined()
  })

  it('applies col.cellClass result to the td element', () => {
    const c: Column[] = [
      { key: 'chg', label: '涨跌', align: 'right', cellClass: (row: any) => (row.chg > 1 ? 'cell-warn' : '') },
    ]
    const w = mount(PanelTable, { props: { columns: c, data } })
    const rows = w.findAll('.table-row')
    expect(rows[0].find('.td').classes()).toContain('cell-warn')
    expect(rows[1].find('.td').classes()).not.toContain('cell-warn')
  })
})

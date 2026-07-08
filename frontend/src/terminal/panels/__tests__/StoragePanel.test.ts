import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import StoragePanel from '../StoragePanel.vue'
import * as wails from '@/lib/wails'

vi.mock('@/lib/wails', () => ({
  GetStorageStats: vi.fn(),
  ArchiveData: vi.fn(),
  ExportData: vi.fn(),
  CleanupData: vi.fn(),
  confirmDialog: vi.fn().mockResolvedValue(true),
}))

const i18n = createI18n({
  locale: 'zh',
  legacy: false,
  messages: {
    zh: {
      storage: {
        title: '存储管理', table: '数据表', rows: '行数', size: '大小',
        oldest: '最早数据', newest: '最新数据', actions: '操作',
        export: '导出', archive: '归档', cleanup: '清理',
      },
    },
  },
})

describe('StoragePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('mounts without crashing', () => {
    ;(wails.GetStorageStats as any).mockResolvedValue([])
    const wrapper = mount(StoragePanel, {
      props: { panelId: 'storage', params: {} },
      global: { plugins: [i18n] },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('renders table stats after load', async () => {
    ;(wails.GetStorageStats as any).mockResolvedValue([
      { table: 'ohlcv_cache', rows: 100, size_bytes: 6400, oldest: '2024-01-01', newest: '2024-06-01' },
      { table: 'minute_cache', rows: 500, size_bytes: 24000, oldest: '2024-03-01', newest: '2024-06-01' },
      { table: 'data_archive', rows: 3, size_bytes: 890, oldest: '2024-01-01', newest: '2024-06-01' },
    ])

    const wrapper = mount(StoragePanel, {
      props: { panelId: 'storage', params: {} },
      global: { plugins: [i18n] },
    })

    // Wait for the async fetch to resolve
    await new Promise((r) => setTimeout(r, 50))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('OHLCV')
    expect(wrapper.text()).toContain('100')
  })
})

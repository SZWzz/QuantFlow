import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import LayoutTemplatePanel from '../LayoutTemplatePanel.vue'
import * as wails from '@/lib/wails'

vi.mock('@/lib/wails', () => ({
  SaveLayout: vi.fn(),
  LoadLayout: vi.fn(),
  ListLayouts: vi.fn(),
  DeleteLayout: vi.fn(),
  confirmDialog: vi.fn().mockResolvedValue(true),
}))

const i18n = createI18n({
  locale: 'zh',
  legacy: false,
  messages: {
    zh: {
      common: { save: '保存', delete: '删除', refresh: '刷新', loading: '加载中...' },
      layout: {
        title: '布局模板',
        saveNew: '保存当前布局',
        namePlaceholder: '布局名称...',
        empty: '暂无已保存的布局',
        confirmDelete: '确认删除布局 "{name}"？',
        hint: 'Ctrl+Shift+1..9 快速切换已保存布局',
      },
    },
  },
})

describe('LayoutTemplatePanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('mounts without crashing', () => {
    ;(wails.ListLayouts as any).mockResolvedValue([])
    const wrapper = mount(LayoutTemplatePanel, {
      global: { plugins: [i18n] },
    })
    expect(wrapper.exists()).toBe(true)
  })

  it('mounts and shows saved layouts', async () => {
    ;(wails.ListLayouts as any).mockResolvedValue(['trading', 'research'])

    const wrapper = mount(LayoutTemplatePanel, {
      global: { plugins: [i18n] },
    })

    await new Promise((r) => setTimeout(r, 50))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('trading')
    expect(wrapper.text()).toContain('research')
  })
})

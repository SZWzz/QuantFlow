import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import SettingsPanel from '../SettingsPanel.vue'

// Mock the Wails bridge — without a desktop backend these calls would hit the
// real @wailsio/runtime and produce unhandled ECONNREFUSED rejections.
vi.mock('@/lib/wails', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/wails')>()
  return {
    ...actual,
    GetCredential: vi.fn().mockResolvedValue(null),
    SaveCredential: vi.fn().mockResolvedValue(undefined),
    GetVersion: vi.fn().mockResolvedValue('test'),
    GetUpdateInterval: vi.fn().mockResolvedValue('24h'),
    SetUpdateInterval: vi.fn().mockResolvedValue(undefined),
    alertDialog: vi.fn().mockResolvedValue(undefined),
  }
})

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: { en: {} },
})

describe('SettingsPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(SettingsPanel, {
      props: { panelId: 'test', params: {} },
      global: {
        plugins: [i18n],
        stubs: { VChart: true, echarts: true },
      },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})

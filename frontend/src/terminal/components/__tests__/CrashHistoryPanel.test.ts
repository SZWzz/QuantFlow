import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

const sampleReports = [
  {
    id: 'crash-1',
    timestamp: '2026-07-15T14:30:00+08:00',
    version: '2026.7.15',
    go_version: 'go1.25',
    os: 'darwin',
    arch: 'arm64',
    build_mode: 'prod',
    panic: 'runtime error: invalid memory address or nil pointer dereference',
    stack: 'goroutine 1 [running]:\nmain.crash()',
    logs: ['INFO started', 'ERROR boom'],
    app_state: {
      trading_mode: 'paper',
      active_brokers: ['futu'],
      panel_count: 2,
      workflow_count: 0,
      uptime_seconds: 60,
    },
  },
  {
    id: 'crash-2',
    timestamp: '2026-07-14T09:15:00+08:00',
    version: '2026.7.14',
    go_version: 'go1.25',
    os: 'darwin',
    arch: 'arm64',
    build_mode: 'prod',
    panic: 'signal: SIGSEGV',
    stack: '',
    logs: [],
    app_state: {
      trading_mode: 'unknown',
      active_brokers: [],
      panel_count: 0,
      workflow_count: 0,
      uptime_seconds: 5,
    },
  },
]

vi.mock('@/lib/wails', () => ({
  ListCrashReports: vi.fn().mockResolvedValue([]),
  DeleteCrashReport: vi.fn().mockResolvedValue(undefined),
  UploadCrashReport: vi.fn().mockResolvedValue(undefined),
  GetCrashDir: vi.fn().mockResolvedValue('/tmp/QuantFlow/crashes'),
  onCrashReport: vi.fn().mockReturnValue(() => {}),
  confirmDialog: vi.fn().mockResolvedValue(true),
  alertDialog: vi.fn().mockResolvedValue(undefined),
}))

import CrashHistoryPanel from '@/terminal/components/CrashHistoryPanel.vue'
import { ListCrashReports, DeleteCrashReport, confirmDialog } from '@/lib/wails'

describe('CrashHistoryPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders empty state when no reports', async () => {
    vi.mocked(ListCrashReports).mockResolvedValueOnce([])
    const wrapper = mount(CrashHistoryPanel)
    await flushPromises()
    expect(wrapper.text()).toContain('崩溃历史 (0 次)')
    expect(wrapper.text()).toContain('暂无崩溃记录')
  })

  it('lists crash reports newest first', async () => {
    vi.mocked(ListCrashReports).mockResolvedValueOnce(sampleReports as any)
    const wrapper = mount(CrashHistoryPanel)
    await flushPromises()
    expect(wrapper.text()).toContain('崩溃历史 (2 次)')
    expect(wrapper.text()).toContain('nil pointer dereference')
    expect(wrapper.text()).toContain('SIGSEGV')
    const items = wrapper.findAll('.crash-item')
    expect(items).toHaveLength(2)
    // Newest (2026-07-15) sorted before older (2026-07-14)
    expect(items[0].text()).toContain('nil pointer dereference')
  })

  it('expands a report to show stack details', async () => {
    vi.mocked(ListCrashReports).mockResolvedValueOnce(sampleReports as any)
    const wrapper = mount(CrashHistoryPanel)
    await flushPromises()
    await wrapper.findAll('[data-test="view"]')[0].trigger('click')
    expect(wrapper.text()).toContain('goroutine 1 [running]:')
  })

  it('deletes a report after confirmDialog accepts', async () => {
    vi.mocked(ListCrashReports).mockResolvedValueOnce(sampleReports as any)
    const wrapper = mount(CrashHistoryPanel)
    await flushPromises()
    await wrapper.findAll('[data-test="delete"]')[0].trigger('click')
    await flushPromises()
    expect(confirmDialog).toHaveBeenCalled()
    expect(DeleteCrashReport).toHaveBeenCalledWith('crash-1')
    expect(wrapper.findAll('.crash-item')).toHaveLength(1)
  })

  it('does not delete when confirmDialog is declined', async () => {
    vi.mocked(ListCrashReports).mockResolvedValueOnce(sampleReports as any)
    vi.mocked(confirmDialog).mockResolvedValueOnce(false)
    const wrapper = mount(CrashHistoryPanel)
    await flushPromises()
    await wrapper.findAll('[data-test="delete"]')[0].trigger('click')
    await flushPromises()
    expect(DeleteCrashReport).not.toHaveBeenCalled()
    expect(wrapper.findAll('.crash-item')).toHaveLength(2)
  })
})

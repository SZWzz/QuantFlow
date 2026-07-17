import { describe, it, expect, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CrashDialog from '@/terminal/components/CrashDialog.vue'

// CrashDialog teleports to <body>, so tests attach to document.body and
// query the teleported DOM directly (same pattern as UpdatePrompt.test.ts).
const baseProps = {
  visible: true,
  crashTime: '2026-07-16T10:30:00',
  crashPath: '/path/to/report.json',
}

describe('CrashDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders crash info', () => {
    mount(CrashDialog, { props: baseProps, attachTo: document.body })
    expect(document.body.textContent).toContain('QuantFlow 崩溃了')
    expect(document.body.textContent).toContain('/path/to/report.json')
  })

  it('renders nothing when not visible', () => {
    mount(CrashDialog, {
      props: { ...baseProps, visible: false },
      attachTo: document.body,
    })
    expect(document.body.textContent).not.toContain('QuantFlow 崩溃了')
  })

  it('emits restart on restart click', () => {
    const wrapper = mount(CrashDialog, { props: baseProps, attachTo: document.body })
    const btn = document.body.querySelector('[data-test="restart"]') as HTMLElement
    btn?.click()
    expect(wrapper.emitted('restart')).toBeTruthy()
  })

  it('emits close on dismiss click', () => {
    const wrapper = mount(CrashDialog, { props: baseProps, attachTo: document.body })
    const btn = document.body.querySelector('[data-test="dismiss"]') as HTMLElement
    btn?.click()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits upload with checkbox state', async () => {
    const wrapper = mount(CrashDialog, { props: baseProps, attachTo: document.body })
    const checkbox = document.body.querySelector('[data-test="upload-optin"]') as HTMLInputElement
    expect(checkbox).toBeTruthy()
    expect(checkbox.checked).toBe(true) // opt-in checked by default
    checkbox.click()
    expect(wrapper.emitted('upload')).toBeTruthy()
    expect(wrapper.emitted('upload')![0]).toEqual([false])
  })

  it('renders panic message and collapsible stack/logs when report provided', () => {
    mount(CrashDialog, {
      props: {
        ...baseProps,
        report: {
          id: 'abc-123',
          timestamp: '2026-07-16T10:30:00+08:00',
          version: '2026.7.16',
          go_version: 'go1.25',
          os: 'darwin',
          arch: 'arm64',
          build_mode: 'prod',
          panic: 'runtime error: index out of range [5]',
          stack: 'goroutine 1 [running]:\nmain.main()',
          logs: ['INFO app started', 'ERROR something failed'],
          app_state: {
            trading_mode: 'paper',
            active_brokers: ['futu'],
            panel_count: 3,
            workflow_count: 1,
            uptime_seconds: 120,
          },
        },
      },
      attachTo: document.body,
    })
    expect(document.body.textContent).toContain('runtime error: index out of range [5]')
    // Stack trace and recent logs are inside collapsible <details> blocks
    const details = document.body.querySelectorAll('details')
    expect(details.length).toBeGreaterThanOrEqual(2)
    expect(document.body.textContent).toContain('goroutine 1 [running]:')
    expect(document.body.textContent).toContain('ERROR something failed')
  })
})

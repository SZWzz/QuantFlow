import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import UpdatePrompt from '@/terminal/components/UpdatePrompt.vue'

describe('UpdatePrompt', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('renders update info when available', () => {
    mount(UpdatePrompt, {
      props: {
        visible: true,
        currentVersion: '2026.7.14',
        latestVersion: '2026.7.20',
        changelog: 'Bug fixes and improvements',
      },
      attachTo: document.body,
    })
    expect(document.body.innerHTML).toContain('2026.7.20')
    expect(document.body.innerHTML).toContain('2026.7.14')
  })

  it('emits close on cancel', () => {
    const wrapper = mount(UpdatePrompt, {
      props: {
        visible: true,
        currentVersion: '2026.7.14',
        latestVersion: '2026.7.20',
        changelog: '',
      },
      attachTo: document.body,
    })
    const btn = document.body.querySelector('[data-test="remind-later"]') as HTMLElement
    btn?.click()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})

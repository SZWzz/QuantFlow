import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import OnboardingOverlay from '../OnboardingOverlay.vue'

describe('OnboardingOverlay', () => {
  it('renders step 1 by default', () => {
    const wrapper = mount(OnboardingOverlay)
    expect(wrapper.text()).toContain('欢迎')
  })
  it('emits done on skip', () => {
    const wrapper = mount(OnboardingOverlay)
    wrapper.find('[data-testid="onboarding-skip"]').trigger('click')
    expect(wrapper.emitted('done')).toBeTruthy()
  })
  it('emits done after completing all steps', async () => {
    const wrapper = mount(OnboardingOverlay)
    for (let i = 0; i < 4; i++) {
      await wrapper.find('[data-testid="onboarding-next"]').trigger('click')
    }
    await wrapper.find('[data-testid="onboarding-done"]').trigger('click')
    expect(wrapper.emitted('done')).toBeTruthy()
  })
  it('emits action with step number on next', async () => {
    const wrapper = mount(OnboardingOverlay)
    await wrapper.find('[data-testid="onboarding-next"]').trigger('click')
    expect(wrapper.emitted('action')?.[0]).toEqual([0])
  })
})

import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CommandBar from '../CommandBar.vue'

describe('CommandBar', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should mount without crashing', () => {
    const wrapper = mount(CommandBar, {
      props: { modelValue: false },
      global: {
        stubs: {
          Teleport: true,
          VChart: true,
          echarts: true,
        },
      },
    })
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.html()).toBeTruthy()
  })
})

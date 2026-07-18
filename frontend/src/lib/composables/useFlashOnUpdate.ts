import { ref, watch, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export type FlashClass = '' | 'flash-up' | 'flash-down'

/**
 * 监听数值变化，返回一个短暂设置的 CSS class（配合全局 .flash-up/.flash-down 动画）。
 * 仅在前后值均为有限数值且发生变化时闪烁。
 */
export function useFlashOnUpdate(
  source: Ref<number | null | undefined>,
  duration = 600,
): { flashClass: Ref<FlashClass> } {
  const flashClass = ref<FlashClass>('')
  let timer: ReturnType<typeof setTimeout> | undefined

  const stop = watch(source, (next, prev) => {
    if (typeof next !== 'number' || typeof prev !== 'number') return
    if (!Number.isFinite(next) || !Number.isFinite(prev) || next === prev) return
    flashClass.value = next > prev ? 'flash-up' : 'flash-down'
    clearTimeout(timer)
    timer = setTimeout(() => { flashClass.value = '' }, duration)
  })

  onUnmounted(() => {
    stop()
    clearTimeout(timer)
  })

  return { flashClass }
}

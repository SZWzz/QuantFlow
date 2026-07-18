import { ref, watch, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export type FlashClass = '' | 'flash-up' | 'flash-down'

/**
 * 监听数值变化，返回一个短暂设置的 CSS class（配合全局 .flash-up/.flash-down 动画）。
 * 仅在前后值均为有限数值且发生变化时闪烁。
 * 注意：duration 窗口内的连续同向变化只重置清除计时器、不重启 CSS 动画
 * （class 字符串不变，DOM 无变化）——高频行情下这是有意行为，避免动画频闪。
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

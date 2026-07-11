import { ref, inject, watch, type Ref } from 'vue'

/**
 * usePanelLifecycle — composable for panel visibility management.
 *
 * Injects `isVisible` from the parent DockTab to determine whether the panel
 * is the currently active tab. Panels can use the `onHidden` / `onVisible`
 * callbacks to pause/resume expensive operations (WebSocket subscriptions,
 * polling intervals, chart rendering) when they are not visible.
 *
 * @example
 * ```ts
 * const { isVisible } = usePanelLifecycle(
 *   () => dataStore.subscribe('market:quote:AAPL'),
 *   (unsub) => unsub()
 * )
 * ```
 */
export function usePanelLifecycle(
  onVisible?: () => (void | (() => void)),
  onHidden?: (cleanup?: () => void) => void
) {
  const isVisible = inject<Ref<boolean>>('isVisible', ref(true))
  let cleanupFn: (() => void) | undefined

  watch(
    isVisible,
    (visible) => {
      if (visible) {
        cleanupFn = onVisible?.() ?? undefined
      } else {
        onHidden?.(cleanupFn)
        cleanupFn = undefined
      }
    },
    { immediate: true }
  )

  return { isVisible }
}

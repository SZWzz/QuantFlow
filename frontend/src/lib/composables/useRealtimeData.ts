import { onMounted, onUnmounted, watch, type Ref, isRef } from 'vue'
import { useWebSocket } from '@/lib/composables/useWebSocket'

/**
 * Generic WebSocket real-time data hook.
 *
 * Subscribes to the given topics on mount, calls handler for each incoming
 * message, and unsubscribes on unmount. When topics change reactively, the
 * WS subscription is automatically refreshed.
 *
 * Usage:
 *   useRealtimeData<MinuteTick[]>(
 *     () => [`market:minute:${symbol.value}`],
 *     (topic, data) => { ... }
 *   )
 */
export function useRealtimeData<T = any>(
  topics: string[] | Ref<string[]> | (() => string[]),
  handler: (topic: string, data: T) => void,
) {
  const ws = useWebSocket()
  const wsUrl = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws/market`
  const unsubs: (() => void)[] = []

  function resolveTopics(): string[] {
    if (typeof topics === 'function') return topics()
    if (isRef(topics)) return topics.value
    return topics
  }

  function connect() {
    const t = resolveTopics()
    if (!t.length) return

    // Clean up previous handlers
    for (const unsub of unsubs) unsub()
    unsubs.length = 0

    // Disconnect old WS (reconnect triggers new subscription)
    ws.disconnect()
    ws.connect(wsUrl, t)

    // Register handler for all incoming messages
    unsubs.push(ws.onMessage('*', (msg: any) => {
      handler(msg.topic, msg.data as T)
    }))
  }

  onMounted(() => connect())

  onUnmounted(() => {
    for (const unsub of unsubs) unsub()
    ws.disconnect()
  })

  // If topics is a Ref or function, watch for changes
  if (isRef(topics) || typeof topics === 'function') {
    const source = isRef(topics) ? topics : null
    if (source) {
      watch(source, () => connect(), { deep: true })
    }
  }

  return { resubscribe: connect, connected: ws.connected }
}

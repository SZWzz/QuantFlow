import { ref, onUnmounted } from 'vue'

type MessageHandler = (data: any) => void

export interface KlineUpdate {
  symbol: string
  interval: string
  time: number
  open: number
  high: number
  low: number
  close: number
  volume: number
  is_closed: boolean
}

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const handlers = new Map<string, Set<MessageHandler>>()
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectAttempts = 0
  const maxReconnectDelay = 30000

  function connect(url: string, topics: string[]) {
    if (ws.value?.readyState === WebSocket.OPEN) return

    try {
      ws.value = new WebSocket(url)
    } catch (e) {
      console.error('[WS] connection failed:', e)
      scheduleReconnect(url, topics)
      return
    }

    ws.value.onopen = () => {
      connected.value = true
      reconnectAttempts = 0
      ws.value?.send(JSON.stringify({ type: 'subscribe', topics }))
    }

    ws.value.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        const topicHandlers = handlers.get(msg.topic)
        if (topicHandlers) {
          topicHandlers.forEach(h => h(msg.data))
        }
        const wildcard = handlers.get('*')
        if (wildcard) {
          wildcard.forEach(h => h(msg))
        }
      } catch (e) {
        console.error('[WS] parse error:', e)
      }
    }

    ws.value.onclose = () => {
      connected.value = false
      ws.value = null
      scheduleReconnect(url, topics)
    }

    ws.value.onerror = () => {
      ws.value?.close()
    }
  }

  function scheduleReconnect(url: string, topics: string[]) {
    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), maxReconnectDelay)
    reconnectAttempts++
    reconnectTimer = setTimeout(() => connect(url, topics), delay)
  }

  function onMessage(topic: string, handler: MessageHandler) {
    if (!handlers.has(topic)) handlers.set(topic, new Set())
    handlers.get(topic)!.add(handler)
    return () => handlers.get(topic)?.delete(handler)
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    handlers.clear()
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  onUnmounted(() => disconnect())

  return { connect, onMessage, disconnect, connected }
}

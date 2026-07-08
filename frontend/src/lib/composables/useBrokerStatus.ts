import { ref } from 'vue'

interface BrokerStatus {
  name: string
  label: string
  market: string
  connected: boolean
  detail: string
}

const brokers = ref<BrokerStatus[]>([])
const loading = ref(false)
const error = ref('')

async function fetchBrokerStatuses() {
  loading.value = true
  error.value = ''
  try {
    const result = await (window as any).go?.main?.App?.GetBrokerStatuses()
    brokers.value = Array.isArray(result) ? result : []
  } catch (e: any) {
    error.value = e?.message || String(e)
    brokers.value = []
  } finally {
    loading.value = false
  }
}

export function useBrokerStatus() {
  return { brokers, loading, error, fetchBrokerStatuses }
}

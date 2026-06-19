<script setup lang="ts">
import { useTerminalStore, type PushPin } from '@/stores/terminal'

const terminal = useTerminalStore()

function removePin(id: string) {
  terminal.pushPins = terminal.pushPins.filter((p) => p.id !== id)
}

function getLabel(pin: PushPin): string {
  switch (pin.type) {
    case 'symbol': return `$${pin.payload.symbol}`
    case 'panel': return pin.label
    case 'workflow': return `WF: ${pin.label}`
    default: return pin.label
  }
}
</script>

<template>
  <div v-if="terminal.pushPins.length > 0" class="pushpin-bar">
    <span class="bar-label">PINNED</span>
    <div
      v-for="pin in terminal.pushPins"
      :key="pin.id"
      class="pin-item"
      :class="pin.type"
      @click="removePin(pin.id)"
    >
      <span class="pin-label">{{ getLabel(pin) }}</span>
      <span class="pin-remove">✕</span>
    </div>
  </div>
</template>

<style scoped>
.pushpin-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  background: #0d1a2d;
  border-bottom: 1px solid #0f3460;
  min-height: 28px;
  overflow-x: auto;
}
.bar-label { font-size: 9px; color: #3a4a6c; text-transform: uppercase; letter-spacing: 1px; }
.pin-item {
  display: flex; align-items: center; gap: 3px; padding: 2px 8px;
  border-radius: 10px; font-size: 11px; cursor: pointer; white-space: nowrap;
}
.pin-item.symbol { background: rgba(88,166,255,0.1); color: #58a6ff; border: 1px solid rgba(88,166,255,0.2); }
.pin-item.panel { background: rgba(233,69,96,0.1); color: #e94560; border: 1px solid rgba(233,69,96,0.2); }
.pin-item.workflow { background: rgba(63,185,80,0.1); color: #3fb950; border: 1px solid rgba(63,185,80,0.2); }
.pin-remove { font-size: 9px; opacity: 0.5; }
.pin-item:hover .pin-remove { opacity: 1; }
</style>

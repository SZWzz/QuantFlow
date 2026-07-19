<script setup lang="ts">
import { useTerminalStore, type PushPin } from '@/stores/terminal'
import { useSymbolContext } from '@/stores/symbolContext'
import { useToast } from '@/lib/composables/useToast'
import { getIcon } from '@/lib/icons'
import type { DockLayoutTree } from '@/terminal/DockView/types'

const terminal = useTerminalStore()
const ctx = useSymbolContext()
const toast = useToast()

function removePin(id: string) {
  const idx = terminal.pushPins.findIndex((p) => p.id === id)
  if (idx < 0) return
  const [pin] = terminal.pushPins.splice(idx, 1)
  toast.addToast({
    type: 'info',
    title: '已删除',
    message: `已删除 pin「${getLabel(pin)}」`,
    duration: 5000,
    action: {
      label: '撤销',
      onClick: () => {
        if (!terminal.pushPins.some((p) => p.id === pin.id)) {
          terminal.pushPins.splice(Math.min(idx, terminal.pushPins.length), 0, pin)
        }
      },
    },
  })
}

/** Find the leaf (tab container) that holds a given tab id. */
function findTabLeaf(node: DockLayoutTree, tabId: string): DockLayoutTree | null {
  if (node.type === 'tab' && node.tabs?.some((t) => t.id === tabId)) return node
  if (node.children) {
    for (const child of node.children) {
      const found = findTabLeaf(child, tabId)
      if (found) return found
    }
  }
  return null
}

function navigateToPin(pin: PushPin) {
  switch (pin.type) {
    case 'symbol': {
      const symbol = pin.payload?.symbol
      if (symbol) ctx.setGroupSymbol(ctx.activeGroupId, symbol)
      break
    }
    case 'panel':
    case 'workflow': {
      const instanceId = pin.payload?.instanceId
      if (instanceId) {
        const leaf = findTabLeaf(terminal.layout, instanceId)
        if (leaf) {
          terminal.selectTab(leaf.id, instanceId)
          return
        }
      }
      if (pin.payload?.panelId) terminal.openPanel(pin.payload.panelId, pin.payload?.params)
      break
    }
  }
}

function getLabel(pin: PushPin): string {
  switch (pin.type) {
    case 'symbol': return `$${pin.payload.symbol}`
    case 'panel': return pin.label
    case 'workflow': return `WF: ${pin.label}`
    default: return pin.label
  }
}

function getIconForType(type: string): string {
  switch (type) {
    case 'symbol': return getIcon('quote')
    case 'panel': return getIcon('terminal')
    case 'workflow': return getIcon('workflow')
    default: return getIcon('pin')
  }
}
</script>

<template>
  <div v-if="terminal.pushPins.length > 0" class="pushpin-bar">
    <span class="bar-label">
      <span class="label-icon" v-html="getIcon('pin')" />
      {{ $t('misc.pinned') }}
    </span>
    <div
      v-for="pin in terminal.pushPins"
      :key="pin.id"
      class="pin-item"
      :class="pin.type"
      role="button"
      tabindex="0"
      :title="getLabel(pin)"
      @click="navigateToPin(pin)"
      @keydown.enter="navigateToPin(pin)"
    >
      <span class="pin-type-icon" v-html="getIconForType(pin.type)" />
      <span class="pin-label">{{ getLabel(pin) }}</span>
      <button
        class="pin-remove"
        :aria-label="`删除 ${getLabel(pin)}`"
        title="删除"
        @click.stop="removePin(pin.id)"
        v-html="getIcon('close')"
      />
    </div>
  </div>
</template>

<style scoped>
.pushpin-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: var(--color-bg-subtle);
  background-image: var(--gradient-card);
  border-bottom: 1px solid var(--color-border);
  min-height: 30px;
  overflow-x: auto;
}

.bar-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 1px;
  font-weight: 600;
  padding-right: 4px;
  border-right: 1px solid var(--color-border);
  flex-shrink: 0;
}

.label-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 10px;
  height: 10px;
  opacity: 0.5;
}

.label-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.pin-item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px 3px 6px;
  border-radius: var(--radius-lg);
  font-size: var(--font-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
  position: relative;
  overflow: hidden;
}

.pin-item::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(255,255,255,0.05) 0%, transparent 100%);
  opacity: 0;
  transition: opacity var(--transition-fast);
}

.pin-item:hover::before {
  opacity: 1;
}

.pin-item.symbol {
  background: var(--color-accent-soft);
  color: var(--color-accent);
  border: 1px solid var(--color-accent-glow);
}

.pin-item.panel {
  background: var(--color-brand-soft);
  color: var(--color-brand);
  border: 1px solid var(--color-brand-glow);
}

.pin-item.workflow {
  background: var(--color-up-soft);
  color: var(--color-up);
  border: 1px solid var(--color-up-glow);
}

.pin-item:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
}

.pin-type-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 10px;
  height: 10px;
  opacity: 0.6;
}

.pin-type-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.pin-label {
  font-weight: 500;
}

.pin-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
  opacity: 0;
  transition: all var(--transition-fast);
  margin-left: 2px;
}

.pin-remove :deep(svg) {
  width: 100%;
  height: 100%;
}

.pin-item:hover .pin-remove,
.pin-item:focus-within .pin-remove {
  opacity: 0.5;
}

.pin-remove:hover {
  opacity: 1 !important;
  color: var(--color-danger) !important;
}
</style>

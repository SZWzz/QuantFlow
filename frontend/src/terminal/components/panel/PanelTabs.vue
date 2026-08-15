<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'

interface Tab {
  key: string
  label: string
}

const props = withDefaults(defineProps<{
  tabs: Tab[]
  active: string
  variant?: 'pill' | 'underline' | 'button'
}>(), {
  variant: 'pill',
})

defineEmits<{
  (e: 'change', key: string): void
}>()

const root = ref<HTMLElement | null>(null)
const indicatorStyle = ref<{ left: string; width: string }>({ left: '0px', width: '0px' })

async function updateIndicator() {
  if (props.variant !== 'underline') return
  await nextTick()
  const el = root.value?.querySelector<HTMLElement>('.tab.active')
  if (!el) return
  indicatorStyle.value = { left: `${el.offsetLeft}px`, width: `${el.offsetWidth}px` }
}

let observer: MutationObserver | null = null

onMounted(() => {
  updateIndicator()
  // 密度/主题切换会改 body class，tab 几何变化后需重算下划线位置
  if (typeof document !== 'undefined' && document.body) {
    observer = new MutationObserver(() => { updateIndicator() })
    observer.observe(document.body, { attributes: true, attributeFilter: ['class'] })
  }
})

onUnmounted(() => {
  observer?.disconnect()
  observer = null
})
watch(() => [props.active, props.tabs], updateIndicator, { deep: true })
</script>

<template>
  <div ref="root" :class="['panel-tabs', `variant-${variant}`]">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      :class="['tab', { active: active === tab.key }]"
      @click="$emit('change', tab.key)"
    >
      {{ tab.label }}
    </button>
    <span v-if="variant === 'underline'" class="tab-indicator" :style="indicatorStyle" />
  </div>
</template>

<style scoped>
.panel-tabs {
  display: flex;
  gap: var(--space-xs);
  position: relative;
}

/* pill variant */
.panel-tabs.variant-pill .tab {
  padding: var(--tab-padding);
  height: var(--tab-height);
  font-size: var(--tab-font-size);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--tab-inactive-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.panel-tabs.variant-pill .tab:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.panel-tabs.variant-pill .tab.active {
  color: var(--color-accent);
  border-color: var(--tab-active-border);
  background: var(--tab-active-bg);
}

/* underline variant */
.panel-tabs.variant-underline .tab {
  padding: 4px 0;
  font-size: var(--tab-font-size);
  border: none;
  background: transparent;
  color: var(--tab-inactive-color);
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-right: var(--space-md);
}

.panel-tabs.variant-underline .tab:hover {
  color: var(--color-text-primary);
}

.panel-tabs.variant-underline .tab.active {
  color: var(--color-accent);
}

.panel-tabs.variant-underline .tab-indicator {
  position: absolute;
  bottom: 0;
  height: 2px;
  background: var(--color-accent);
  border-radius: 1px;
  transition: left var(--transition-normal), width var(--transition-normal);
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .panel-tabs.variant-underline .tab-indicator { transition: none; }
}

/* button variant */
.panel-tabs.variant-button .tab {
  padding: var(--tab-padding);
  height: var(--tab-height);
  font-size: var(--tab-font-size);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-subtle);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.panel-tabs.variant-button .tab:hover {
  border-color: var(--color-border-strong);
  background: var(--color-bg-hover);
}

.panel-tabs.variant-button .tab.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-text-inverse);
}
</style>

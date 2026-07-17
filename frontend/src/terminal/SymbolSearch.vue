<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import type { StockEntry } from '@/lib/symbolSearch'

const props = defineProps<{
  modelValue?: string
  placeholder?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'select': [entry: StockEntry]
}>()

const query = ref('')
const results = ref<StockEntry[]>([])
const loading = ref(false)
const marketFilter = ref<string>('ALL')
const open = ref(false)
const selectedIndex = ref(-1)

const filteredResults = computed(() => {
  if (marketFilter.value === 'ALL') return results.value
  return results.value.filter(e => e.market === marketFilter.value)
})

const MARKET_TABS = [
  { key: 'ALL', label: 'All' },
  { key: 'SH', label: '沪' },
  { key: 'SZ', label: '深' },
  { key: 'HK', label: '港' },
  { key: 'US', label: '美' },
  { key: 'BJ', label: '京' },
] as const
const inputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLUListElement | null>(null)

const displayValue = ref(props.modelValue ?? '')
const selectedName = ref('')

let debounceTimer: ReturnType<typeof setTimeout>

async function searchApi(q: string): Promise<StockEntry[]> {
  loading.value = true
  try {
    const app = (window as any).go?.main?.App
    if (app?.SearchSymbols) {
      return await app.SearchSymbols(q.trim(), 20)
    }
  } catch {}
  return []
}

watch(query, (newQuery) => {
  clearTimeout(debounceTimer)
  if (newQuery.length < 2) { results.value = []; loading.value = false; return }
  debounceTimer = setTimeout(() => {
    searchApi(newQuery).then(r => { results.value = r; loading.value = false })
  }, 300)
})

// When user picks a result
function select(entry: StockEntry) {
  displayValue.value = entry.code
  selectedName.value = entry.name
  query.value = ''
  open.value = false
  selectedIndex.value = -1
  emit('update:modelValue', entry.code)
  emit('select', entry)
}

// Keyboard navigation
function onKeydown(e: KeyboardEvent) {
  if (!open.value || results.value.length === 0) {
    if (e.key === 'Enter') {
      emit('update:modelValue', displayValue.value)
    }
    return
  }
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, results.value.length - 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      e.preventDefault()
      if (selectedIndex.value >= 0 && selectedIndex.value < results.value.length) {
        select(results.value[selectedIndex.value])
      }
      break
    case 'Escape':
      open.value = false
      selectedIndex.value = -1
      break
  }
}

function onFocus() {
  if (displayValue.value) {
    query.value = displayValue.value
  }
  open.value = true
}

function onBlur() {
  // Delay close so click on result can fire
  setTimeout(() => {
    open.value = false
    selectedIndex.value = -1
  }, 150)
}

function onInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  displayValue.value = val
  query.value = val
  selectedName.value = ''
  open.value = true
  selectedIndex.value = -1
}

function marketBadge(market: string): string {
  switch (market) {
    case 'SH': return '沪'
    case 'SZ': return '深'
    case 'BJ': return '京'
    case 'HK': return '港'
    case 'US': return '美'
    default: return market
  }
}

function marketColor(market: string): string {
  switch (market) {
    case 'SH': return '#ef4444'  // red
    case 'SZ': return '#22c55e'  // green
    case 'BJ': return '#3b82f6'  // blue
    case 'HK': return '#f59e0b'  // amber
    case 'US': return '#8b5cf6'  // purple
    default: return '#6b7280'
  }
}

// Close on outside click
function onClickOutside(e: MouseEvent) {
  const el = (e.target as HTMLElement)
  if (inputRef.value && !inputRef.value.contains(el) && listRef.value && !listRef.value.contains(el)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))
onUnmounted(() => clearTimeout(debounceTimer))
</script>

<template>
  <div class="symbol-search">
    <div class="input-wrapper">
      <input
        ref="inputRef"
        class="search-input"
        :value="selectedName || displayValue"
        :placeholder="placeholder ?? $t('common.search') + '...'"
        @input="onInput"
        @focus="onFocus"
        @blur="onBlur"
        @keydown="onKeydown"
        autocomplete="off"
      />
      <span v-if="loading" class="spinner">⏳</span>
    </div>

    <ul v-if="open && results.length > 0" ref="listRef" class="dropdown">
      <li class="filter-row">
        <button
          v-for="tab in MARKET_TABS" :key="tab.key"
          :class="['filter-tab', { active: marketFilter === tab.key }]"
          @mousedown.prevent="marketFilter = tab.key"
        >{{ tab.label }}</button>
      </li>
      <li
        v-for="(entry, i) in filteredResults"
        :key="entry.code"
        :class="['dropdown-item', { active: i === selectedIndex }]"
        @mousedown.prevent="select(entry)"
      >
        <span class="market-badge" :style="{ background: marketColor(entry.market) }">
          {{ marketBadge(entry.market) }}
        </span>
        <span class="item-code">{{ entry.code }}</span>
        <span class="item-name">{{ entry.name }}</span>
      </li>
      <li v-if="filteredResults.length === 0" class="empty-result">{{ $t('common.no_data') }}</li>
    </ul>

    <div v-if="open && query && !loading && results.length === 0" class="dropdown empty">
      {{ $t('common.no_data') }}
    </div>
  </div>
</template>

<style scoped>
.symbol-search {
  position: relative;
  width: 100%;
}
.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}
.search-input {
  width: 100%;
  padding: 6px 28px 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: 13px;
  outline: none;
  box-sizing: border-box;
}
.search-input:focus {
  border-color: var(--color-accent);
}
.search-input::placeholder {
  color: var(--color-text-secondary);
}
.spinner {
  position: absolute;
  right: 8px;
  font-size: 12px;
}
.dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  z-index: var(--z-dropdown);
  max-height: 240px;
  overflow-y: auto;
  margin: 2px 0 0 0;
  padding: 4px 0;
  list-style: none;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  box-shadow: var(--shadow-md);
}
.dropdown.empty {
  padding: 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
  text-align: center;
}
.dropdown-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
}
.dropdown-item:hover,
.dropdown-item.active {
  background: var(--color-bg-hover);
}
.filter-row {
  display: flex;
  gap: 4px;
  padding: 4px 10px 8px;
  border-bottom: 1px solid var(--color-border);
  margin-bottom: 4px;
  position: sticky;
  top: 0;
  background: var(--color-bg-elevated);
  z-index: 1;
}
.filter-tab {
  padding: 2px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: 11px;
}
.filter-tab.active {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-accent-soft);
}
.empty-result {
  padding: 8px 10px;
  color: var(--color-text-secondary);
  font-size: 12px;
  text-align: center;
}
.market-badge {
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
}
.item-code {
  font-weight: 600;
  color: var(--color-accent);
  font-variant-numeric: tabular-nums;
  width: 60px;
  flex-shrink: 0;
}
.item-name {
  color: var(--color-text-primary);
}
</style>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface TickerItem {
  symbol: string
  name: string
  price: number
  changePct: number
}

const SYMBOLS = ['600519', '000001', '300750', '601318', '000858', '600036', '601166', '600276']

const items = ref<TickerItem[]>([])

onMounted(async () => {
  const results: TickerItem[] = []
  for (const sym of SYMBOLS) {
    try {
      const [snapshot, _source] = await (window as any).go.main.App.GetQuote('CN', sym)
      results.push({
        symbol: snapshot.symbol ?? sym,
        name: snapshot.name ?? sym,
        price: snapshot.last ?? 0,
        changePct: snapshot.change_pct ?? snapshot.changePct ?? 0,
      })
    } catch {
      // skip failed symbols
    }
  }
  items.value = results
})
</script>

<template>
  <div class="ticker-tape-panel">
    <span class="tape-title">Ticker Tape</span>
    <div class="tape-track-container">
      <div class="tape-track">
        <span v-for="(item, idx) in items" :key="idx" class="tape-item">
          <span class="tape-symbol">{{ item.symbol }}</span>
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :style="{ color: item.changePct >= 0 ? '#ef4444' : '#22c55e' }">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
        <!-- Duplicate for seamless loop -->
        <span v-for="(item, idx) in items" :key="'dup-' + idx" class="tape-item">
          <span class="tape-symbol">{{ item.symbol }}</span>
          <span class="tape-name">{{ item.name }}</span>
          <span class="tape-price">{{ item.price.toFixed(2) }}</span>
          <span class="tape-change" :style="{ color: item.changePct >= 0 ? '#ef4444' : '#22c55e' }">
            {{ item.changePct >= 0 ? '+' : '' }}{{ item.changePct.toFixed(2) }}%
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ticker-tape-panel {
  height: 36px;
  display: flex;
  align-items: center;
  color: var(--color-text, #e5e7eb);
  background: #0f172a;
  border-bottom: 1px solid var(--color-border, #374151);
  overflow: hidden;
  padding: 0 8px;
  gap: 12px;
}
.tape-title {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  white-space: nowrap;
  flex-shrink: 0;
  margin-right: 4px;
}
.tape-track-container {
  flex: 1;
  overflow: hidden;
  mask-image: linear-gradient(to right, transparent 0%, black 2%, black 98%, transparent 100%);
}
.tape-track {
  display: inline-flex;
  gap: 24px;
  white-space: nowrap;
  animation: scroll 40s linear infinite;
}
.tape-track:hover {
  animation-play-state: paused;
}
.tape-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.tape-symbol {
  font-weight: 600;
  color: #e5e7eb;
}
.tape-name {
  color: #9ca3af;
  font-size: 11px;
}
.tape-price {
  color: #e5e7eb;
}
.tape-change {
  font-weight: 500;
}

@keyframes scroll {
  from { transform: translateX(0); }
  to { transform: translateX(-50%); }
}
</style>

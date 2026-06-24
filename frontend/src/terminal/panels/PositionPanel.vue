<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Position {
  symbol: string; quantity: number; avgPrice: number; marketPrice: number
  pnl: number; pnlPct: number; dayPnl: number
}

const positions = ref<Position[]>([
  { symbol: 'AAPL', quantity: 500, avgPrice: 188.50, marketPrice: 195.32, pnl: 3410, pnlPct: 3.62, dayPnl: 2050 },
  { symbol: 'GOOGL', quantity: 200, avgPrice: 140.00, marketPrice: 142.15, pnl: 430, pnlPct: 1.54, dayPnl: -170 },
  { symbol: 'NVDA', quantity: 100, avgPrice: 820.00, marketPrice: 875.28, pnl: 5528, pnlPct: 6.74, dayPnl: 1245 },
])

const totalPnl = computed(() => positions.value.reduce((s, p) => s + p.pnl, 0))
const totalDayPnl = computed(() => positions.value.reduce((s, p) => s + p.dayPnl, 0))

function fmt(n: number, dec = 2): string { return n.toFixed(dec) }
</script>

<template>
  <div class="position-panel">
    <div class="summary-row">
      <div class="summary-item">
        <span class="s-label">总盈亏</span>
        <span :class="['s-value', totalPnl >= 0 ? 'up' : 'down']">${{ fmt(totalPnl) }}</span>
      </div>
      <div class="summary-item">
        <span class="s-label">日内盈亏</span>
        <span :class="['s-value', totalDayPnl >= 0 ? 'up' : 'down']">${{ fmt(totalDayPnl) }}</span>
      </div>
    </div>

    <div class="position-list">
      <div v-for="pos in positions" :key="pos.symbol" class="position-row">
        <div class="pos-main">
          <span class="pos-symbol">{{ pos.symbol }}</span>
          <span class="pos-qty">{{ pos.quantity }} 手</span>
        </div>
        <div class="pos-prices">
          <span class="pos-avg">@{{ fmt(pos.avgPrice) }}</span>
          <span class="pos-mkt">→ {{ fmt(pos.marketPrice) }}</span>
        </div>
        <div class="pos-pnl" :class="pos.pnl >= 0 ? 'up' : 'down'">
          <span class="pnl-val">${{ fmt(pos.pnl) }}</span>
          <span class="pnl-pct">({{ pos.pnl >= 0 ? '+' : '' }}{{ fmt(pos.pnlPct) }}%)</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.position-panel { padding: 10px; background: #1a1a2e; height: 100%; overflow-y: auto; font-variant-numeric: tabular-nums; }
.summary-row { display: flex; gap: 8px; margin-bottom: 10px; }
.summary-item { flex: 1; padding: 8px; background: #16213e; border-radius: 4px; text-align: center; }
.s-label { display: block; font-size: 10px; color: #5a6380; text-transform: uppercase; }
.s-value { font-size: 16px; font-weight: 700; }
.s-value.up { color: #3fb950; } .s-value.down { color: #f85149; }

.position-row { padding: 8px; border-bottom: 1px solid #0f2137; }
.pos-main { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; }
.pos-symbol { font-weight: 600; font-size: 13px; color: #e0e0e0; }
.pos-qty { font-size: 11px; color: #5a6380; }
.pos-prices { font-size: 11px; color: #5a6380; margin-bottom: 2px; }
.pos-pnl { font-size: 12px; font-weight: 500; }
.pos-pnl.up { color: #3fb950; } .pos-pnl.down { color: #f85149; }
.pnl-pct { font-size: 10px; opacity: 0.7; }
</style>

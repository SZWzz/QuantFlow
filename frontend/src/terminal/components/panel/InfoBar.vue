<script setup lang="ts">
import type { QuoteData } from '@/lib/composables/useWailsApp'
import { computed } from 'vue'
import { marketChangeColor } from '@/lib/composables/useMarketColors'

const props = defineProps<{
  quote: QuoteData | null
  symbol: string
  name: string
}>()

const changeColor = computed(() => {
  if (!props.quote) return 'var(--color-text-primary)'
  return marketChangeColor(props.symbol, props.quote.change ?? 0)
})

function formatVolume(v?: number): string {
  if (v == null) return '--'
  if (v >= 10000) return (v / 10000).toFixed(1) + '\u4E07'
  return v.toFixed(0)
}
function formatMarketCap(v?: number): string {
  if (v == null) return '--'
  if (v >= 100000000) return (v / 100000000).toFixed(2) + '\u4EBF'
  if (v >= 10000) return (v / 10000).toFixed(2) + '\u4E07'
  return v.toFixed(0)
}
</script>

<template>
  <div class="info-bar">
    <span class="symbol">{{ symbol }}</span>
    <span class="name">{{ name }}</span>
    <span class="price" :style="{ color: changeColor }">
      {{ quote?.price?.toFixed(2) ?? '--' }}
    </span>
    <span class="change" :style="{ color: changeColor }">
      {{ quote?.change?.toFixed(2) ?? '--' }}
    </span>
    <span class="change-pct" :style="{ color: changeColor }">
      {{ quote?.change_percent != null ? (quote.change_percent >= 0 ? '+' : '') + quote.change_percent.toFixed(2) + '%' : '--' }}
    </span>
    <span class="sep">|</span>
    <span class="stat">{{ $t('kline.turnover') }} {{ quote?.turnover_rate != null ? quote.turnover_rate.toFixed(2) + '%' : '--' }}</span>
    <span class="stat">{{ $t('kline.volume_ratio') }} {{ quote?.volume_ratio?.toFixed(2) ?? '--' }}</span>
    <span class="stat">{{ $t('kline.amplitude') }} {{ quote?.amplitude?.toFixed(2) ?? '--' }}%</span>
    <span class="sep">|</span>
    <span class="stat">{{ $t('kline.avg_price') }} {{ quote?.avg_price?.toFixed(2) ?? '--' }}</span>
    <span class="stat">{{ $t('kline.inside') }} {{ formatVolume(quote?.inside_volume) }}</span>
    <span class="stat">{{ $t('kline.outside') }} {{ formatVolume(quote?.outside_volume) }}</span>
    <span class="stat">{{ $t('kline.market_cap') }} {{ formatMarketCap(quote?.market_cap) }}</span>
    <span class="stat">{{ $t('kline.pe') }} {{ quote?.pe_ratio?.toFixed(1) ?? '--' }}</span>
  </div>
</template>

<style scoped>
.info-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  overflow-x: auto;
  white-space: nowrap;
  min-height: 28px;
  flex-shrink: 0;
}
.symbol { font-weight: 700; color: var(--color-text-primary); }
.name { color: var(--color-text-tertiary); font-size: 11px; }
.price { font-weight: 700; font-size: 14px; }
.change, .change-pct { font-size: 13px; }
.sep { color: var(--color-border-subtle); }
.stat { color: var(--color-text-secondary); }
</style>

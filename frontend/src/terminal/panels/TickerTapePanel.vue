<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = defineProps<{ panelId: string; params?: Record<string, any> }>()

interface TickerItem {
  symbol: string
  name: string
  price: number
  changePct: number
}

const items = ref<TickerItem[]>([])

const mockStocks: TickerItem[] = [
  { symbol: '600519', name: '贵州茅台', price: 1850.50, changePct: 2.30 },
  { symbol: '000858', name: '五粮液', price: 156.20, changePct: -0.80 },
  { symbol: '300750', name: '宁德时代', price: 212.35, changePct: 3.15 },
  { symbol: '601318', name: '中国平安', price: 48.76, changePct: 0.55 },
  { symbol: '000001', name: '平安银行', price: 12.48, changePct: -1.20 },
  { symbol: '600036', name: '招商银行', price: 38.92, changePct: 1.05 },
  { symbol: '002594', name: '比亚迪', price: 287.40, changePct: 4.20 },
  { symbol: '600030', name: '中信证券', price: 22.56, changePct: -0.35 },
  { symbol: '000725', name: '京东方A', price: 4.82, changePct: 0.62 },
  { symbol: '300059', name: '东方财富', price: 18.35, changePct: 2.78 },
  { symbol: '600900', name: '长江电力', price: 28.15, changePct: -0.18 },
  { symbol: '688981', name: '中芯国际', price: 56.30, changePct: 5.60 },
  { symbol: '601899', name: '紫金矿业', price: 16.88, changePct: 1.92 },
  { symbol: '002415', name: '海康威视', price: 34.21, changePct: -2.10 },
  { symbol: '600809', name: '山西汾酒', price: 268.50, changePct: 0.88 },
  { symbol: '603259', name: '药明康德', price: 62.75, changePct: -3.40 },
  { symbol: '000333', name: '美的集团', price: 68.42, changePct: 1.15 },
  { symbol: '002230', name: '科大讯飞', price: 52.18, changePct: 3.85 },
  { symbol: '601012', name: '隆基绿能', price: 18.64, changePct: -1.55 },
  { symbol: '600276', name: '恒瑞医药', price: 45.30, changePct: 0.42 },
  { symbol: '002460', name: '赣锋锂业', price: 32.90, changePct: 6.20 },
  { symbol: '300274', name: '阳光电源', price: 89.45, changePct: 2.55 },
]

onMounted(() => {
  items.value = mockStocks
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

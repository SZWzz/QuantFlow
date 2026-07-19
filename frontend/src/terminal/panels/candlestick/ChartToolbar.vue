<script setup lang="ts">
import { computed } from 'vue'
import { PanelHeader } from '@/terminal/components/panel'

const props = defineProps<{
  symbol: string
  name: string
  isInWatchlist: boolean
  activeTab: 'kline' | 'minute'
  interval: string
  drawingMode: boolean
  addToWfControl?: { icon: string; label: string; title: string; action: () => Promise<void> } | null
  topOverlay: string
  bottomMode: string
  minuteBottomMode: string
  indexOverlaySymbol: string
  showDepth: boolean
}>()

const emit = defineEmits<{
  'toggleWatchlist': []
  'toggleDrawingMode': []
  'update:activeTab': [value: 'kline' | 'minute']
  'update:interval': [value: string]
  'update:topOverlay': [value: 'none' | 'ma' | 'bb' | 'sar' | 'ema']
  'update:bottomMode': [value: 'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv']
  'update:minuteBottomMode': [value: 'volume' | 'macd' | 'kdj' | 'rsi' | 'obv']
  'update:indexOverlaySymbol': [value: string]
  'toggleDepth': []
}>()

// 周期切换 → PanelHeader tabs；仅 K 线模式显示（与原 .interval-btns 的 v-if 行为一致）
const intervalTabs = computed(() =>
  ['1m', '5m', '15m', '30m', '1h', '1d', '1w'].map(i => ({ key: i, label: i })),
)
const headerControls = computed(() => (props.addToWfControl ? [props.addToWfControl] : []))

function onIntervalChange(key: string) {
  emit('update:interval', key)
}
</script>

<template>
  <PanelHeader
    :title="symbol"
    :subtitle="name"
    :tabs="activeTab === 'kline' ? intervalTabs : []"
    :active-tab="interval"
    :controls="headerControls"
    @tab-change="onIntervalChange"
  >
    <template #controls>
      <button
        class="btn btn-sm wl-btn"
        :class="{ inList: isInWatchlist }"
        @click="emit('toggleWatchlist')"
      >{{ isInWatchlist ? $t('watchlist.remove') : $t('watchlist.add') }}</button>
      <div class="mode-switch">
        <button class="btn btn-sm" :class="{ active: activeTab === 'kline' }" @click="emit('update:activeTab', 'kline')">{{ $t('kline.kline') }}</button>
        <button class="btn btn-sm" :class="{ active: activeTab === 'minute' }" @click="emit('update:activeTab', 'minute')">{{ $t('kline.minute') }}</button>
      </div>
      <button class="btn btn-sm" :class="{ active: drawingMode }" title="画线工具 (Shift+D)" @click="emit('toggleDrawingMode')">✏️</button>
    </template>
  </PanelHeader>

  <!-- 次级工具条：指标开关/叠加指数/深度，controls 无法表达，按 P1 保留 header 下方一行 -->
  <div v-if="activeTab === 'kline'" class="indicator-bar">
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.overlay') }}</span>
      <button class="btn btn-sm" :class="{ active: topOverlay === 'none' }" @click="emit('update:topOverlay', 'none')">无</button>
      <button class="btn btn-sm" :class="{ active: topOverlay === 'ma' }" @click="emit('update:topOverlay', 'ma')">MA</button>
      <button class="btn btn-sm" :class="{ active: topOverlay === 'bb' }" @click="emit('update:topOverlay', 'bb')">{{ $t('kline.bb') }}</button>
      <button class="btn btn-sm" :class="{ active: topOverlay === 'sar' }" @click="emit('update:topOverlay', 'sar')">SAR</button>
      <button class="btn btn-sm" :class="{ active: topOverlay === 'ema' }" @click="emit('update:topOverlay', 'ema')">EMA</button>
    </div>
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'volume' }" @click="emit('update:bottomMode', 'volume')">{{ $t('kline.volume') }}</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'macd' }" @click="emit('update:bottomMode', 'macd')">MACD</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'kdj' }" @click="emit('update:bottomMode', 'kdj')">KDJ</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'rsi' }" @click="emit('update:bottomMode', 'rsi')">RSI</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'wr' }" @click="emit('update:bottomMode', 'wr')">WR</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'cci' }" @click="emit('update:bottomMode', 'cci')">CCI</button>
      <button class="btn btn-sm" :class="{ active: bottomMode === 'obv' }" @click="emit('update:bottomMode', 'obv')">OBV</button>
    </div>
    <div class="indicator-group">
      <span class="indicator-label">叠加指数</span>
      <select :value="indexOverlaySymbol" class="index-select" @change="emit('update:indexOverlaySymbol', ($event.target as HTMLSelectElement).value)">
        <option value="">不叠加</option>
        <option value="000001">上证指数</option>
        <option value="399001">深证成指</option>
        <option value="399006">创业板指</option>
      </select>
    </div>
  </div>
  <div v-else class="indicator-bar">
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
      <button class="btn btn-sm" :class="{ active: minuteBottomMode === 'volume' }" @click="emit('update:minuteBottomMode', 'volume')">{{ $t('kline.volume') }}</button>
      <button class="btn btn-sm" :class="{ active: minuteBottomMode === 'macd' }" @click="emit('update:minuteBottomMode', 'macd')">MACD</button>
      <button class="btn btn-sm" :class="{ active: minuteBottomMode === 'kdj' }" @click="emit('update:minuteBottomMode', 'kdj')">KDJ</button>
      <button class="btn btn-sm" :class="{ active: minuteBottomMode === 'rsi' }" @click="emit('update:minuteBottomMode', 'rsi')">RSI</button>
      <button class="btn btn-sm" :class="{ active: minuteBottomMode === 'obv' }" @click="emit('update:minuteBottomMode', 'obv')">OBV</button>
    </div>
    <div class="indicator-group">
      <button class="btn btn-sm" :class="{ active: showDepth }" @click="emit('toggleDepth')">📊 {{ $t('misc.depth') }}</button>
    </div>
  </div>
</template>

<style scoped>
/* 全局 .btn 之上的紧凑修饰（与 AlphaMiningWorkspacePanel 的 .btn-sm 约定一致，尺寸 token 化） */
.btn-sm {
  padding: var(--space-xs) var(--space-sm);
  font-size: var(--font-xs);
}

.mode-switch { display: flex; gap: var(--space-xs); }

/* 自选按钮：accent 描边，已在列表转 down 色（沿用原 .watchlist-btn 语义） */
.wl-btn { color: var(--color-accent); border-color: var(--color-accent); background: transparent; }
.wl-btn:hover { background: var(--color-accent); border-color: var(--color-accent); color: var(--color-text-inverse); }
.wl-btn.inList { border-color: var(--color-down); color: var(--color-down); }
.wl-btn.inList:hover { background: var(--color-down); border-color: var(--color-down); color: var(--color-text-inverse); }

/* 开关型按钮 active 态（模式切换/画线/指标通用） */
.btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-text-inverse);
}

/* 次级工具条 */
.indicator-bar {
  display: flex; gap: var(--space-lg); align-items: center; flex-wrap: wrap;
  padding: var(--space-xs) var(--space-sm);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
}
.indicator-group { display: flex; align-items: center; gap: var(--space-xs); }
.indicator-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-right: var(--space-xs); }
.indicator-bar .btn { font-family: 'JetBrains Mono', monospace; }

.index-select {
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: 'JetBrains Mono', monospace;
  outline: none;
}
.index-select:hover { border-color: var(--color-accent); color: var(--color-accent); }
.index-select option { background: var(--color-bg-elevated); color: var(--color-text-primary); }
</style>

<script setup lang="ts">
import { getIcon } from '@/lib/icons'

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
  'addToWorkflow': []
  'update:activeTab': [value: 'kline' | 'minute']
  'update:interval': [value: string]
  'update:topOverlay': [value: 'none' | 'ma' | 'bb' | 'sar' | 'ema']
  'update:bottomMode': [value: 'volume' | 'macd' | 'kdj' | 'rsi' | 'wr' | 'cci' | 'obv']
  'update:minuteBottomMode': [value: 'volume' | 'macd' | 'kdj' | 'rsi' | 'obv']
  'update:indexOverlaySymbol': [value: string]
  'toggleDepth': []
}>()
</script>

<template>
  <div class="chart-header">
    <div class="header-left">
      <span class="symbol-display">{{ symbol }} {{ name }}</span>
      <button
        class="watchlist-btn"
        :class="{ inList: isInWatchlist }"
        @click="emit('toggleWatchlist')"
      >{{ isInWatchlist ? $t('watchlist.remove') : $t('watchlist.add') }}</button>
      <div class="tab-btns">
        <button :class="{ active: activeTab === 'kline' }" class="tab-btn" @click="emit('update:activeTab', 'kline')">{{ $t('kline.kline') }}</button>
        <button :class="{ active: activeTab === 'minute' }" class="tab-btn" @click="emit('update:activeTab', 'minute')">{{ $t('kline.minute') }}</button>
      </div>
      <button class="drawing-btn" @click="emit('toggleDrawingMode')" :class="{ active: drawingMode }" title="画线工具 (Shift+D)">
        ✏️
      </button>
      <button v-if="addToWfControl" class="wf-btn" @click="emit('addToWorkflow')" :title="$t('workflow.add_to_workflow')" v-html="getIcon('plus')" />
    </div>
    <div v-if="activeTab === 'kline'" class="interval-btns">
      <button v-for="i in ['1m','5m','15m','30m','1h','1d','1w']" :key="i"
        :class="{ active: interval === i }" class="interval-btn"
        @click="emit('update:interval', i)">{{ i }}</button>
    </div>
  </div>
  <div v-if="activeTab === 'kline'" class="indicator-bar">
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.overlay') }}</span>
      <button :class="{ active: topOverlay === 'none' }" class="indicator-btn" @click="emit('update:topOverlay', 'none')">无</button>
      <button :class="{ active: topOverlay === 'ma' }" class="indicator-btn" @click="emit('update:topOverlay', 'ma')">MA</button>
      <button :class="{ active: topOverlay === 'bb' }" class="indicator-btn" @click="emit('update:topOverlay', 'bb')">{{ $t('kline.bb') }}</button>
      <button :class="{ active: topOverlay === 'sar' }" class="indicator-btn" @click="emit('update:topOverlay', 'sar')">SAR</button>
      <button :class="{ active: topOverlay === 'ema' }" class="indicator-btn" @click="emit('update:topOverlay', 'ema')">EMA</button>
    </div>
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
      <button :class="{ active: bottomMode === 'volume' }" class="indicator-btn" @click="emit('update:bottomMode', 'volume')">{{ $t('kline.volume') }}</button>
      <button :class="{ active: bottomMode === 'macd' }" class="indicator-btn" @click="emit('update:bottomMode', 'macd')">MACD</button>
      <button :class="{ active: bottomMode === 'kdj' }" class="indicator-btn" @click="emit('update:bottomMode', 'kdj')">KDJ</button>
      <button :class="{ active: bottomMode === 'rsi' }" class="indicator-btn" @click="emit('update:bottomMode', 'rsi')">RSI</button>
      <button :class="{ active: bottomMode === 'wr' }" class="indicator-btn" @click="emit('update:bottomMode', 'wr')">WR</button>
      <button :class="{ active: bottomMode === 'cci' }" class="indicator-btn" @click="emit('update:bottomMode', 'cci')">CCI</button>
      <button :class="{ active: bottomMode === 'obv' }" class="indicator-btn" @click="emit('update:bottomMode', 'obv')">OBV</button>
    </div>
    <div class="indicator-group">
      <span class="indicator-label">叠加指数</span>
      <select :value="indexOverlaySymbol" class="toolbar-select" @change="emit('update:indexOverlaySymbol', ($event.target as HTMLSelectElement).value)">
        <option value="">不叠加</option>
        <option value="000001">上证指数</option>
        <option value="399001">深证成指</option>
        <option value="399006">创业板指</option>
      </select>
    </div>
  </div>
  <div v-if="activeTab === 'minute'" class="indicator-bar">
    <div class="indicator-group">
      <span class="indicator-label">{{ $t('kline.sub_chart') }}</span>
      <button :class="{ active: minuteBottomMode === 'volume' }" class="indicator-btn" @click="emit('update:minuteBottomMode', 'volume')">{{ $t('kline.volume') }}</button>
      <button :class="{ active: minuteBottomMode === 'macd' }" class="indicator-btn" @click="emit('update:minuteBottomMode', 'macd')">MACD</button>
      <button :class="{ active: minuteBottomMode === 'kdj' }" class="indicator-btn" @click="emit('update:minuteBottomMode', 'kdj')">KDJ</button>
      <button :class="{ active: minuteBottomMode === 'rsi' }" class="indicator-btn" @click="emit('update:minuteBottomMode', 'rsi')">RSI</button>
      <button :class="{ active: minuteBottomMode === 'obv' }" class="indicator-btn" @click="emit('update:minuteBottomMode', 'obv')">OBV</button>
    </div>
    <div class="indicator-group">
      <button class="indicator-btn depth-toggle" :class="{ active: showDepth }" @click="emit('toggleDepth')">📊 {{ $t('misc.depth') }}</button>
    </div>
  </div>
</template>

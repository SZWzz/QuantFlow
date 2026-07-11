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

<style scoped>
.chart-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 6px 10px; border-bottom: 1px solid var(--color-border);
}
.header-left {
  display: flex; align-items: center; gap: 12px;
}
.symbol-display {
  font-size: var(--font-lg); font-weight: 700;
  color: var(--color-brand);
}
.watchlist-btn {
  padding: 2px 10px; border: 1px solid var(--color-accent); border-radius: var(--radius-sm);
  background: transparent; color: var(--color-accent); cursor: pointer;
  font-size: 11px; white-space: nowrap; transition: all var(--transition-fast);
}
.watchlist-btn:hover { background: var(--color-accent); color: #fff; }
.watchlist-btn.inList { border-color: var(--color-down); color: var(--color-down); }
.watchlist-btn.inList:hover { background: var(--color-down); color: #fff; }
.tab-btns { display: flex; gap: 4px; }
.tab-btn {
  padding: 3px 12px; border: 1px solid var(--color-border-strong); border-radius: var(--radius-sm);
  background: var(--color-bg-elevated); color: var(--color-text-secondary); font-size: 12px; cursor: pointer;
}
.tab-btn.active { background: var(--color-border-strong); color: var(--color-text-primary); border-color: var(--color-accent); }
.interval-btns { display: flex; gap: 2px; }
.interval-btn {
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-tertiary);
  border-radius: var(--radius-sm); cursor: pointer;
  font-size: var(--font-xs); font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}
.interval-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.interval-btn.active {
  background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent);
}
.indicator-bar {
  display: flex; gap: 16px; align-items: center;
  padding: 4px 10px; border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-elevated);
}
.indicator-group { display: flex; align-items: center; gap: 4px; }
.indicator-label { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-right: 4px; }
.indicator-btn {
  padding: 2px 8px; border: 1px solid var(--color-border);
  background: transparent; color: var(--color-text-tertiary);
  border-radius: var(--radius-sm); cursor: pointer;
  font-size: var(--font-xs); font-family: 'JetBrains Mono', monospace;
  transition: all var(--transition-fast);
}
.indicator-btn:hover { border-color: var(--color-accent); color: var(--color-accent); }
.indicator-btn.active {
  background: var(--color-accent); color: var(--color-text-primary); border-color: var(--color-accent);
}
.depth-toggle { }
.toolbar-select {
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--font-xs);
  font-family: 'JetBrains Mono', monospace;
  outline: none;
}
.toolbar-select:hover { border-color: var(--color-accent); color: var(--color-accent); }
.toolbar-select option { background: var(--color-bg-elevated); color: var(--color-text-primary); }
.drawing-btn {
  padding: 3px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 14px;
  cursor: pointer;
  line-height: 1;
}
.drawing-btn:hover { border-color: var(--color-text-secondary); }
.drawing-btn.active {
  border-color: var(--color-accent);
  background: rgba(88, 166, 255, 0.15);
}
.wf-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  line-height: 1;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.wf-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: rgba(88, 166, 255, 0.1);
}
</style>

# Watchlist Panel Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 WatchlistPanel 从单列基础列表升级为 CSS Grid 表格，支持多列展示、排序、市场分组、右键菜单、拖拽排序、轮询刷新等专业终端功能。

**Architecture:** 纯前端改动 + i18n 扩展 + CandlestickPanel 小幅修改。后端 `GetQuote` 已返回所有所需字段，无需 Go 改动。

**Tech Stack:** Vue 3 Composition API, CSS Grid, HTML5 Drag & Drop API, localStorage

## Global Constraints
- 排序/分组/列配置存储在 localStorage (`quantflow-watchlist-config`)
- 轮询用 `setInterval(10000)` + `document.hidden` 暂停
- 右键菜单用 `@contextmenu.prevent` + 定位浮层
- 所有新增文案走 i18n
- 每次 build 通过 (`wails3 build`)

---

### Task 1: i18n key 扩展

**Files:**
- Modify: `frontend/src/lib/i18n/zh.ts`
- Modify: `frontend/src/lib/i18n/en.ts`

**新增 keys:**

zh.ts — `watchlist` section:
```
add: '加入自选', remove: '取消自选', speed: '涨速',
empty: '暂无自选股，在K线面板点击"加入自选"添加',
column_settings: '列设置', sort_by: '排序',
group_cn: 'A股', group_hk: '港股', group_us: '美股', group_crypto: '加密',
drag_hint: '拖拽排序',
context_open_kline: '跳转K线', context_copy: '复制代码', context_delete: '删除',
polling_paused: '自动刷新已暂停',
```

en.ts — `watchlist` section:
```
add: 'Add', remove: 'Remove', speed: 'Speed',
empty: 'No stocks yet. Add from the K-line panel.',
column_settings: 'Columns', sort_by: 'Sort',
group_cn: 'A-Share', group_hk: 'HK', group_us: 'US', group_crypto: 'Crypto',
drag_hint: 'Drag to reorder',
context_open_kline: 'Open K-line', context_copy: 'Copy Code', context_delete: 'Delete',
polling_paused: 'Auto-refresh paused',
```

---

### Task 2: WatchlistPanel.vue 重写 — 数据层

**File:** `frontend/src/terminal/panels/WatchlistPanel.vue`

**Changes:**

2a. 新增类型定义和配置持久化:
```ts
interface ColumnDef {
  key: string
  label: string
  visible: boolean
  sortable: boolean
  format?: 'price' | 'percent' | 'volume' | 'number'
}
interface SortState { key: string; dir: 'asc' | 'desc' | null }

const CONFIG_KEY = 'quantflow-watchlist-config'
const defaultColumns: ColumnDef[] = [
  { key: 'symbol', label: 'common.symbol', visible: true, sortable: false },
  { key: 'name', label: 'common.name', visible: true, sortable: false },
  { key: 'last', label: 'common.price', visible: true, sortable: true, format: 'price' },
  { key: 'changePct', label: 'quote.change_pct', visible: true, sortable: true, format: 'percent' },
  { key: 'change', label: 'quote.change', visible: false, sortable: true },
  { key: 'speed', label: 'watchlist.speed', visible: true, sortable: true, format: 'percent' },
  { key: 'volumeRatio', label: 'kline.volume_ratio', visible: true, sortable: true, format: 'number' },
  { key: 'turnoverRate', label: 'kline.turnover', visible: false, sortable: true, format: 'percent' },
  { key: 'amplitude', label: 'kline.amplitude', visible: false, sortable: true, format: 'percent' },
  { key: 'volume', label: 'common.volume', visible: false, sortable: true, format: 'volume' },
  { key: 'turnover', label: 'quote.turnover', visible: false, sortable: true, format: 'volume' },
  { key: 'high', label: 'quote.high', visible: false, sortable: true, format: 'price' },
  { key: 'low', label: 'quote.low', visible: false, sortable: true, format: 'price' },
]
```

2b. 扩展 `quotes` 数据结构存储上一次价格用于计算涨速:
```ts
interface QuoteRow {
  symbol: string; name: string; last: number; open: number; high: number; low: number;
  change: number; changePct: number; volume: number; turnover: number;
  turnoverRate: number; volumeRatio: number; amplitude: number;
  prevLast?: number; // for speed calculation
}
```

2c. `refreshQuote` 返回全部 QuoteSnapshot 字段 + 保存 `prevLast`

2d. 排序逻辑:
```ts
const sort = ref<SortState>({ key: '', dir: null })
const sortedSymbols = computed(() => {
  if (!sort.value.dir || !sort.value.key) return symbols.value
  const key = sort.value.key
  return [...symbols.value].sort((a, b) => {
    const va = getValue(a, key); const vb = getValue(b, key)
    return sort.value.dir === 'asc' ? va - vb : vb - va
  })
})
```

2e. 市场分组:
```ts
const groups = computed(() => {
  const map: Record<string, string[]> = { CN: [], HK: [], US: [], CRYPTO: [] }
  for (const sym of displaySymbols.value) {
    const m = detectMarket(sym); if (map[m]) map[m].push(sym); else map.CN.push(sym)
  }
  return map
})
const expandedGroups = ref<Record<string, boolean>>({ CN: true, HK: true, US: true, CRYPTO: true })
```

2f. 轮询:
```ts
let pollTimer: ReturnType<typeof setInterval> | null = null
function startPolling() {
  stopPolling()
  pollTimer = setInterval(() => { if (!document.hidden) refreshAll() }, 10000)
}
function stopPolling() { if (pollTimer) { clearInterval(pollTimer); pollTimer = null } }
onMounted(() => { startPolling(); document.addEventListener('visibilitychange', onVisibility) })
onUnmounted(() => { stopPolling(); document.removeEventListener('visibilitychange', onVisibility) })
```

---

### Task 3: WatchlistPanel.vue 重写 — 模板

CSS Grid 布局:
```vue
<div class="symbol-table">
  <!-- Header row -->
  <div class="table-header">
    <div v-for="col in visibleColumns" :key="col.key"
      class="th" :class="{ sortable: col.sortable, asc: sort.key === col.key && sort.dir === 'asc', desc: sort.key === col.key && sort.dir === 'desc' }"
      @click="col.sortable && toggleSort(col.key)">
      {{ $t(col.label) }}
      <span v-if="sort.key === col.key" class="sort-arrow">{{ sort.dir === 'asc' ? '↑' : '↓' }}</span>
    </div>
    <div class="th-actions">
      <button @click="showColumnSettings = !showColumnSettings" title="列设置">⚙</button>
    </div>
  </div>
  <!-- Group accordion -->
  <template v-for="(syms, market) in groups" :key="market">
    <div v-if="syms.length" class="group-header" @click="toggleGroup(market)">
      <span class="group-arrow">{{ expandedGroups[market] ? '▼' : '▶' }}</span>
      <span class="group-label">{{ $t('watchlist.group_' + market.toLowerCase()) }}</span>
      <span class="group-count">{{ syms.length }}</span>
    </div>
    <div v-if="expandedGroups[market]" v-for="sym in syms" :key="sym"
      class="table-row" :class="rowClasses(sym)" @click="selectSymbol(sym)"
      @contextmenu.prevent="openContextMenu($event, sym)" draggable="true"
      @dragstart="onDragStart($event, sym)" @dragover.prevent="onDragOver($event, sym)"
      @drop="onDrop($event, sym)">
      <!-- Empty state per cell -->
      <div v-for="col in visibleColumns" :key="col.key" class="td" :class="col.key">
        <template v-if="col.key === 'symbol'">{{ sym }}</template>
        <template v-else-if="col.key === 'name'">{{ getCell(sym, 'name') }}</template>
        <template v-else>{{ formatCell(sym, col) }}</template>
      </div>
      <div class="td-actions">
        <button class="remove-btn" @click.stop="removeSymbol(sym)">✕</button>
      </div>
    </div>
  </template>
</div>
```

---

### Task 4: CandlestickPanel i18n + 删除同步

**File:** `frontend/src/terminal/panels/CandlestickPanel.vue`

- 第 723 行: 两处硬编码文字改为 `$t('watchlist.add')` / `$t('watchlist.remove')`
- `removeSymbol` 在 WatchlistPanel 中 dispatch `watchlist-changed` event

---

### Task 5: 测试扩展

**File:** `frontend/src/terminal/panels/__tests__/WatchlistPanel.test.ts`

新增测试:
- 空状态渲染
- 排序切换
- 删除 symbol
- watchlist-changed event 响应
- localStorage 读写

---

### Task 6: 验证

```bash
wails3 build
go vet ./...
```

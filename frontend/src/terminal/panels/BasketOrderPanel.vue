<script setup lang="ts">
import { ref, computed } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

// -- 篮子 row --
interface 篮子Row {
  id: number
  symbol: string
  weight: number
  quantity: number
  price: number
}

let nextId = 1
const rows = ref<篮子Row[]>([
  { id: nextId++, symbol: 'AAPL', weight: 30, quantity: 50, price: 195.3 },
  { id: nextId++, symbol: '600519', weight: 40, quantity: 100, price: 1780.0 },
  { id: nextId++, symbol: 'TSLA', weight: 30, quantity: 80, price: 248.5 },
])

// -- 导入 CSV --
const csvText = ref('')
const showCsvImport = ref(false)

function addRow() {
  rows.value.push({ id: nextId++, symbol: '', weight: 0, quantity: 0, price: 0 })
}

function removeRow(id: number) {
  if (rows.value.length <= 1) return
  rows.value = rows.value.filter(r => r.id !== id)
}

function importCSV() {
  const lines = csvText.value.trim().split('\n')
  for (const line of lines) {
    const parts = line.split(/[,\t]/)
    if (parts.length >= 2) {
      rows.value.push({
        id: nextId++,
        symbol: parts[0].trim(),
        weight: parts[1] ? parseFloat(parts[1]) || 0 : 0,
        quantity: parts[2] ? parseInt(parts[2]) || 0 : 0,
        price: parts[3] ? parseFloat(parts[3]) || 0 : 0,
      })
    }
  }
  csvText.value = ''
  showCsvImport.value = false
}

// -- Execution mode --
const execMode = ref<'market' | 'limit' | 'weighted'>('limit')

// -- Execution log --
interface LogEntry {
  symbol: string
  status: 'pending' | 'executing' | 'filled' | 'failed'
  message: string
  time: string
}

const logs = ref<LogEntry[]>([])
const isExecuting = ref(false)

async function execute篮子() {
  isExecuting.value = true
  logs.value = []

  for (const row of rows.value) {
    if (!row.symbol) continue
    const entry: LogEntry = {
      symbol: row.symbol,
      status: 'pending',
      message: `Queued ${row.symbol}`,
      time: new Date().toLocaleTimeString(),
    }
    logs.value.push(entry)

    // Simulate sequential execution with 300ms delay
    await new Promise(resolve => setTimeout(resolve, 300))

    entry.status = 'executing'
    entry.message = `Executing ${row.symbol} — ${execMode.value} order`
    entry.time = new Date().toLocaleTimeString()

    await new Promise(resolve => setTimeout(resolve, 300))

    entry.status = 'filled'
    entry.message = `Filled ${row.symbol}: ${row.quantity} @ ${row.price.toFixed(2)}`
    entry.time = new Date().toLocaleTimeString()
  }

  isExecuting.value = false
}

// -- 摘要 --
const estimatedCost = computed(() => {
  return rows.value.reduce((sum, r) => sum + r.quantity * r.price, 0)
})

const symbolCount = computed(() => {
  return rows.value.filter(r => r.symbol.trim() !== '').length
})

function fmtMoney(n: number): string {
  if (Math.abs(n) >= 1e6) return '$' + (n / 1e6).toFixed(2) + 'M'
  if (Math.abs(n) >= 1e3) return '$' + (n / 1e3).toFixed(1) + 'K'
  return '$' + n.toFixed(2)
}

function statusDotClass(s: string): string {
  return 'dot dot-' + s
}
</script>

<template>
  <div class="basket-panel">
    <!-- Three-column grid -->
    <div class="basket-grid">
      <!-- Left: 篮子 Rows -->
      <div class="col col-left">
        <h3 class="col-title">篮子</h3>
        <div class="row-list">
          <div v-for="row in rows" :key="row.id" class="basket-row">
            <input v-model="row.symbol" type="text" placeholder="Symbol" class="cell-input cell-symbol" />
            <input v-model.number="row.weight" type="number" min="0" max="100" placeholder="Wt%" class="cell-input cell-num" />
            <input v-model.number="row.quantity" type="number" min="0" placeholder="Qty" class="cell-input cell-num" />
            <input v-model.number="row.price" type="number" step="0.01" placeholder="Price" class="cell-input cell-num" />
            <button class="row-btn row-remove" @click="removeRow(row.id)" :disabled="rows.length <= 1">x</button>
          </div>
        </div>
        <div class="row-actions">
          <button class="action-btn" @click="addRow">+ 添加行</button>
          <button class="action-btn" @click="showCsvImport = !showCsvImport">导入 CSV</button>
        </div>
        <div v-if="showCsvImport" class="csv-import">
          <textarea v-model="csvText" placeholder="Paste CSV: symbol,weight%,qty,price" rows="3" class="csv-textarea"></textarea>
          <button class="action-btn" @click="importCSV">Parse &amp; Import</button>
        </div>
      </div>

      <!-- Center: 摘要 -->
      <div class="col col-center">
        <h3 class="col-title">摘要</h3>
        <div class="summary-card">
          <div class="summary-row">
            <span class="s-label">代码数量</span>
            <span class="s-value">{{ symbolCount }}</span>
          </div>
          <div class="summary-row">
            <span class="s-label">预估总成本</span>
            <span class="s-value">{{ fmtMoney(estimatedCost) }}</span>
          </div>
          <div class="summary-row">
            <span class="s-label">执行模式</span>
            <select v-model="execMode" class="exec-select">
              <option value="market">全部市价</option>
              <option value="limit">全部限价</option>
              <option value="weighted">按权重</option>
            </select>
          </div>
        </div>
        <button
          class="execute-btn"
          :disabled="isExecuting || symbolCount === 0"
          @click="execute篮子"
        >
          {{ isExecuting ? '执行中...' : 'Execute 篮子' }}
        </button>
      </div>

      <!-- Right: 执行日志 -->
      <div class="col col-right">
        <h3 class="col-title">执行日志</h3>
        <div class="log-list">
          <div v-for="(entry, i) in logs" :key="i" class="log-entry">
            <span :class="statusDotClass(entry.status)"></span>
            <div class="log-body">
              <span class="log-symbol">{{ entry.symbol }}</span>
              <span class="log-msg">{{ entry.message }}</span>
              <span class="log-time">{{ entry.time }}</span>
            </div>
          </div>
          <div v-if="logs.length === 0" class="log-empty">
            暂无执行记录
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.basket-panel {
  padding: 10px;
  background: var(--bg);
  height: 100%;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}

.basket-grid {
  display: grid;
  grid-template-columns: 1fr 220px 1fr;
  gap: var(--spacing);
  height: 100%;
}

.col {
  display: flex;
  flex-direction: column;
  overflow-y: auto;
}

.col-title {
  font-size: var(--font-xs);
  text-transform: uppercase;
  color: var(--muted);
  letter-spacing: 0.5px;
  margin-bottom: 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--input);
}

/* -- 篮子 rows -- */
.row-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  overflow-y: auto;
}

.basket-row {
  display: flex;
  gap: 4px;
  align-items: center;
}

.cell-input {
  padding: 5px 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
  font-size: 11px;
  outline: none;
  min-width: 0;
}
.cell-input:focus { border-color: var(--accent); }

.cell-symbol { flex: 2; }
.cell-num { flex: 1; }

.row-btn {
  padding: 4px 6px;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 3px;
  color: var(--muted);
  font-size: 10px;
  cursor: pointer;
  flex-shrink: 0;
}
.row-btn:hover { color: var(--down); border-color: var(--down); }
.row-btn:disabled { opacity: 0.3; cursor: not-allowed; }

/* -- Row actions -- */
.row-actions {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}

.action-btn {
  padding: 4px 10px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s;
}
.action-btn:hover { background: var(--card); }

/* -- CSV import -- */
.csv-import {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.csv-textarea {
  padding: 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
  font-size: 11px;
  font-family: monospace;
  outline: none;
  resize: vertical;
}
.csv-textarea:focus { border-color: var(--accent); }

/* -- 摘要 -- */
.summary-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--card);
  border-radius: 4px;
  padding: 10px;
}

.summary-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.s-label {
  font-size: var(--font-xs);
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.s-value {
  font-size: 15px;
  font-weight: 700;
  color: var(--text);
}

.exec-select {
  padding: 5px 6px;
  background: var(--input);
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text);
  font-size: 11px;
  outline: none;
}

.execute-btn {
  margin-top: 12px;
  padding: 10px;
  background: var(--accent);
  border: none;
  border-radius: 6px;
  color: #000;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}
.execute-btn:hover:not(:disabled) { opacity: 0.85; }
.execute-btn:disabled { opacity: 0.4; cursor: not-allowed; }

/* -- Execution log -- */
.log-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.log-entry {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 6px 8px;
  background: var(--card);
  border-radius: 4px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
}
.dot-pending   { background: #f59e0b; }
.dot-executing { background: #3b82f6; animation: pulse 0.8s infinite; }
.dot-filled    { background: #22c55e; }
.dot-failed    { background: #ef4444; }

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.log-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.log-symbol {
  font-weight: 600;
  font-size: 11px;
  color: var(--text);
}

.log-msg {
  font-size: 10px;
  color: var(--muted);
}

.log-time {
  font-size: 9px;
  color: var(--muted);
}

.log-empty {
  text-align: center;
  color: var(--muted);
  padding: 24px;
  font-size: 11px;
}
</style>

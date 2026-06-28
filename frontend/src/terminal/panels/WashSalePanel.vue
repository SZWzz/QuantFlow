<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SkeletonPanel from '@/terminal/components/SkeletonPanel.vue'

const { t } = useI18n()
const app = window.go.main.App
const symbol = ref('AAPL')
const loading = ref(false)
const events = ref<any[]>([])

const totalDisallowed = computed(() =>
  events.value.reduce((sum: number, e: any) => sum + (Number(e.disallowed_loss) || 0), 0)
)

async function checkWashSale() {
  const sym = symbol.value.trim().toUpperCase()
  if (!sym) return
  loading.value = true
  try {
    const raw = await app.CheckWashSale(sym)
    events.value = Array.isArray(raw) ? raw : []
  } catch {
    events.value = []
  } finally {
    loading.value = false
  }
}

function formatCurrency(v: any): string {
  const n = Number(v) || 0
  const prefix = n < 0 ? '-' : ''
  return prefix + '$' + Math.abs(n).toFixed(2)
}

function isNegative(v: any): boolean {
  return (Number(v) || 0) < 0
}
</script>

<template>
  <div class="panel">
    <div class="panel-header">
      <h3>{{ t('wash_sale') }}</h3>
      <div class="header-controls">
        <input class="symbol-input" v-model="symbol" placeholder="Symbol" @keyup.enter="checkWashSale" />
        <button class="check-btn" @click="checkWashSale" :disabled="loading">
          {{ loading ? '...' : t('check_btn') }}
        </button>
      </div>
    </div>

    <SkeletonPanel v-if="loading" type="table" />

    <template v-else-if="events.length > 0">
      <div class="table-wrap">
        <table class="wash-table">
          <thead>
            <tr>
              <th>亏损卖出日</th>
              <th>亏损金额</th>
              <th>回购日</th>
              <th>窗口天数</th>
              <th>不允许亏损</th>
              <th>调整后成本基础</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(e, i) in events" :key="i">
              <td>{{ e.loss_date }}</td>
              <td :class="['num-cell', { negative: isNegative(e.loss_amount) }]">{{ formatCurrency(e.loss_amount) }}</td>
              <td>{{ e.repurchase_date }}</td>
              <td class="num-cell">{{ e.window_days }}</td>
              <td :class="['num-cell', { negative: isNegative(e.disallowed_loss) }]">{{ formatCurrency(e.disallowed_loss) }}</td>
              <td :class="['num-cell', { negative: isNegative(e.adjusted_basis) }]">{{ formatCurrency(e.adjusted_basis) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="total-row">
        <span class="total-label">{{ t('common.total') }}: </span>
        <span :class="['total-value', { negative: isNegative(totalDisallowed) }]">{{ formatCurrency(totalDisallowed) }}</span>
      </div>
    </template>

    <div v-else-if="!symbol.trim()" class="empty-state">
      输入代码开始检测
    </div>

    <div v-else class="empty-state">
      {{ t('no_wash_sale') }}
    </div>

    <div v-if="!loading" class="disclaimer">{{ t('wash_sale_disclaimer') }}</div>
  </div>
</template>

<style scoped>
.panel {
  padding: 16px;
  height: 100%;
  display: flex;
  flex-direction: column;
  color: var(--color-text, #e5e7eb);
  background: var(--color-bg, var(--color-bg-panel));
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.header-controls {
  display: flex;
  gap: 8px;
}
.symbol-input {
  width: 100px;
  padding: 4px 8px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  font-size: 13px;
  outline: none;
}
.check-btn {
  padding: 4px 10px;
  border: 1px solid var(--color-border-strong);
  border-radius: 4px;
  background: var(--color-bg-elevated);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 13px;
}
.check-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.table-wrap {
  flex: 1;
  overflow-y: auto;
}
.wash-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.wash-table th {
  text-align: left;
  padding: 6px 8px;
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border-strong);
  font-weight: 500;
  white-space: nowrap;
}
.wash-table td {
  padding: 6px 8px;
  border-bottom: 1px solid var(--color-bg-elevated);
}
.num-cell {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.negative {
  color: var(--color-danger, #ef4444);
}
.total-row {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  margin-top: 8px;
  border-top: 1px solid var(--color-border-strong);
  font-size: 13px;
  font-weight: 600;
}
.total-label {
  color: var(--color-text-secondary);
}
.total-value {
  font-variant-numeric: tabular-nums;
}
.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 13px;
}
.disclaimer {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--color-border-subtle);
  color: var(--color-text-tertiary);
  font-size: 11px;
  text-align: center;
}
</style>

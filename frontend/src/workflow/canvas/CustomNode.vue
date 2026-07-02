<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Handle, Position } from '@vue-flow/core'

const { t } = useI18n()

const props = defineProps<{
  id: string
  data: {
    nodeType: string
    label: string
    params: Record<string, any>
    inputs?: string[]
    outputs?: string[]
    status?: 'idle' | 'running' | 'success' | 'failed'
    error?: string
  }
  selected?: boolean
}>()

const NODE_LABELS: Record<string, string> = {
  data_loader: '数据加载', sma: '简单均线', ema: '指数均线', macd: 'MACD', rsi: 'RSI',
  bollinger: '布林带', merge: '合并', filter: '过滤', resample: '重采样',
  cross_signal: '交叉信号', threshold_signal: '阈值信号', signal_combine: '信号组合',
  rank_select: '排名选择', hold_signal: '持仓信号', rebalance: '再平衡',
  entry_signal: '入场信号', exit_signal: '离场信号',
  log_output: '日志输出', chart_data: '图表数据',
  loop: '循环', if_condition: '条件判断', sub_workflow: '子工作流',
  factor: '因子', pct_change: '涨跌幅', delta: '差值', std_dev: '标准差',
  rank: '排名', scale: '标准化', cross_over: '上穿', compare: '比较',
  bool_combine: '布尔组合', rolling_maxmin: '滚动极值', rolling_zscore: '滚动Z值',
  arithmetic: '算术运算', if_else: '条件分支',
  strategy: '策略', backtest: '回测', agent: 'AI代理',
  place_order: '下单', cancel_order: '取消订单', position_query: '持仓查询', order_query: '订单查询',
  notify: '通知', alert: '告警', schedule: '定时', wait: '等待',
  portfolio_summary: '组合概况', risk_metrics: '风险指标', allocation: '资产配置',
  stop_loss: '止损', position_sizer: '仓位计算', risk_model: '风险模型',
  http_request: 'HTTP请求', math_op: '数学运算', json_parse: 'JSON解析',
  feature_engineer: '特征工程', train_model: '训练模型', predict: '预测',
  evaluate_model: '评估模型', alpha_mining: 'Alpha挖掘', rl_env: 'RL环境',
  rl_train: 'RL训练', rl_predict: 'RL预测',
  sentiment: '情绪分析', news_fetcher: '新闻获取', stock_research: '股票研究',
  financials: '财务报表', peer_compare: '同业对比', analyst_estimates: '分析师预测',
  insider_trades: '内部交易',
  prediction_market: '预测市场', geopolitics: '地缘政治', gov_data: '宏观数据', satellite: '卫星数据',
}
function nodeLabel(type: string): string { return NODE_LABELS[type] || type.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase()) }
function portLabel(name: string): string { return t(`workflow.port_${name}`) !== `workflow.port_${name}` ? t(`workflow.port_${name}`) : name }

const CAT_ICONS: Record<string, string> = {
  data: '📊', indicator: '📈', signal: '⚡', trading: '💰', risk: '🛡️',
  portfolio: '📦', strategy: '🧠', ml: '🤖', ai: '🤖', output: '📝', control: '🔀',
  utility: '🔧', research: '🔍', alternative_data: '🛰️', notify: '🔔', schedule: '⏰',
  backtest: '📉', alpha: '⭐',
}
const CAT_COLORS: Record<string, string> = {
  data: '#58a6ff', indicator: '#3fb950', signal: '#f0883e', trading: '#e94560',
  risk: '#ef4444', portfolio: '#14b8a6', strategy: '#06b6d4', ml: '#a855f7',
  ai: '#ec4899', output: '#a371f7', control: '#6366f1', utility: '#64748b',
  research: '#0ea5e9', alternative_data: '#84cc16', notify: '#f97316', schedule: '#6366f1',
  backtest: '#8b5cf6', alpha: '#f59e0b',
}
const category = computed(() => (props.data as any).category || 'utility')
const catIcon = computed(() => CAT_ICONS[category.value] || '🔹')
const categoryColor = computed(() => CAT_COLORS[category.value] || CAT_COLORS.utility)

const statusClass = computed(() => `status-${props.data.status || 'idle'}`)

const paramSummary = computed(() => {
  const params = props.data.params || {}
  const keys = Object.keys(params)
  if (keys.length === 0) return ''
  return keys.map(k => `${k}=${params[k]}`).join(', ')
})
</script>

<template>
  <div class="custom-node" :class="[statusClass, { selected }]">
    <div class="node-header" :style="{ background: categoryColor }">
      <span class="cat-icon">{{ catIcon }}</span><span class="node-type">{{ nodeLabel(data.nodeType) }}</span>
    </div>

    <div class="node-body">
      <div v-if="paramSummary" class="node-params">
        {{ paramSummary }}
      </div>

      <!-- Input handles (left side) -->
      <div class="handles inputs">
        <div
          v-for="port in (data.inputs || ['input'])"
          :key="port"
          class="handle-row"
        >
          <Handle :type="'target'" :position="Position.Left" :id="port" class="port-handle" />
          <span class="port-label">{{ portLabel(port) }}</span>
        </div>
      </div>

      <!-- Output handles (right side) -->
      <div class="handles outputs">
        <div
          v-for="port in (data.outputs || ['output'])"
          :key="port"
          class="handle-row output-row"
        >
          <span class="port-label">{{ portLabel(port) }}</span>
          <Handle :type="'source'" :position="Position.Right" :id="port" class="port-handle" />
        </div>
      </div>
    </div>

    <!-- Status indicator -->
    <div v-if="data.status === 'running'" class="running-indicator" />
    <div v-if="data.status === 'success'" class="success-check">✓</div>
    <div v-if="data.status === 'failed'" class="failed-mark">✗ {{ data.error }}</div>
  </div>
</template>

<style scoped>
.custom-node {
  background: #1c2333;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  min-width: 150px;
  max-width: 220px;
  font-size: 12px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.custom-node.selected {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(88, 166, 255, 0.3);
}

.custom-node.status-running {
  border-color: #f0883e;
  animation: pulse 1.5s ease-in-out infinite;
}

.custom-node.status-success {
  border-color: #3fb950;
}

.custom-node.status-failed {
  border-color: #f85149;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.7; }
}

.node-header {
  padding: 6px 12px;
  border-radius: var(--radius-md) 6px 0 0;
  color: #fff;
  font-weight: 600;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-body {
  padding: 8px 0;
  position: relative;
}

.node-params {
  padding: 2px 12px;
  font-size: 10px;
  color: var(--color-text-tertiary);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.handles {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.handle-row {
  display: flex;
  align-items: center;
  padding: 2px 0;
}

.handle-row.output-row {
  justify-content: flex-end;
}

.port-label {
  font-size: 10px;
  color: var(--color-text-tertiary);
  margin: 0 8px;
}

.port-handle {
  width: 10px !important;
  height: 10px !important;
  background: var(--color-border) !important;
  border: 2px solid var(--color-accent) !important;
  border-radius: 50% !important;
}

.running-indicator {
  position: absolute;
  bottom: 4px;
  right: 4px;
  width: 8px;
  height: 8px;
  background: #f0883e;
  border-radius: 50%;
}

.success-check {
  position: absolute;
  bottom: 2px;
  right: 6px;
  color: #3fb950;
  font-size: 14px;
  font-weight: bold;
}

.failed-mark {
  position: absolute;
  bottom: 2px;
  right: 6px;
  color: #f85149;
  font-size: 10px;
}
</style>

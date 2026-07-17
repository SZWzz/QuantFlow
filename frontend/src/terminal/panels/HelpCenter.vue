<script setup lang="ts">
import { ref } from 'vue'

const sections = [
  {
    id: 'quickstart', title: '快速上手', items: [
      { q: '如何查看实时行情？', a: '打开 Watchlist 面板（Ctrl+Shift+W），输入股票代码如 600519.SH 即可查看实时报价和 K 线图。' },
      { q: '如何创建策略？', a: '点击右上角 Workflow 按钮切换到工作流模式，从左侧节点面板拖拽节点到画布，连线搭建策略。点击 ▶ 运行策略。' },
      { q: '如何切换市场？', a: 'QuantFlow 支持 A股/港股/美股/加密。在 Watchlist 中输入不同市场的代码即可：600519.SH (A股), 00700.HK (港股), AAPL (美股), BTCUSDT (加密)。' },
    ],
  },
  {
    id: 'panels', title: '面板操作', items: [
      { q: '面板快捷键', a: 'Ctrl+K 打开命令面板搜索，Ctrl+Shift+字母 快速打开特定面板：W=Watchlist, H=港股通, F=资金费率, L=涨跌停, D=龙虎榜。' },
      { q: '如何创建布局？', a: '拖拽面板标签页可以改变布局。创建好布局后，点击右上角设置 → 保存布局模板，下次可一键恢复。' },
      { q: '如何分离面板？', a: '右键面板标签页 → Tear Off 将面板分离到独立窗口，适合多显示器使用。' },
    ],
  },
  {
    id: 'trading', title: '交易相关', items: [
      { q: 'Paper 模式和 Live 模式有什么区别？', a: 'Paper 模式使用虚拟资金模拟交易，Live 模式直接向券商发送真实订单。切换前会有安全检查。Live 模式下顶部显示红色横幅。' },
      { q: '如何配置券商？', a: '打开 API 密钥管理面板，找到你的券商（如 Alpaca/Binance/富途），填入 API Key 和 Secret，点击保存后验证。' },
      { q: '如何查看日结报告？', a: '打开 日结报告 面板（组合与风控分类），选择日期点击生成报告，可查看当日盈亏、最佳/最差交易、持仓摘要。' },
    ],
  },
]

const expanded = ref<Record<string, boolean>>({})

function toggle(sectionId: string, idx: number) {
  const key = `${sectionId}-${idx}`
  expanded.value[key] = !expanded.value[key]
}
</script>

<template>
  <div class="help-center">
    <h3>📚 帮助中心</h3>
    <div v-for="section in sections" :key="section.id" class="help-section">
      <h4>{{ section.title }}</h4>
      <div v-for="(item, idx) in section.items" :key="idx" class="faq-item">
        <div class="faq-question" @click="toggle(section.id, idx)">
          <span class="faq-q">Q: {{ item.q }}</span>
          <span class="faq-toggle">{{ expanded[`${section.id}-${idx}`] ? '−' : '+' }}</span>
        </div>
        <div v-if="expanded[`${section.id}-${idx}`]" class="faq-answer">
          {{ item.a }}
        </div>
      </div>
    </div>

    <div class="shortcuts-section">
      <h4>⌨️ 快捷键</h4>
      <div class="shortcut-grid">
        <div class="shortcut-item"><kbd>Ctrl+K</kbd> 命令面板</div>
        <div class="shortcut-item"><kbd>Ctrl+W</kbd> 切换 Workflow</div>
        <div class="shortcut-item"><kbd>Ctrl+Shift+W</kbd> Watchlist</div>
        <div class="shortcut-item"><kbd>Ctrl+Shift+H</kbd> 港股通</div>
        <div class="shortcut-item"><kbd>Ctrl+Shift+F</kbd> 资金费率</div>
        <div class="shortcut-item"><kbd>Ctrl+Shift+L</kbd> 涨跌停</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.help-center { padding: 16px; overflow-y: auto; height: 100%; }
.help-center h3 { font-size: 15px; margin-bottom: 16px; }
.help-section { margin-bottom: 20px; }
.help-section h4 { font-size: 13px; color: var(--color-accent); margin-bottom: 8px; }
.faq-item { border-bottom: 1px solid var(--color-border); }
.faq-question {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 0; cursor: pointer;
  font-size: 13px; font-weight: 600;
}
.faq-question:hover { color: var(--color-accent); }
.faq-toggle { font-size: 16px; color: var(--color-text-tertiary); }
.faq-answer {
  padding: 0 0 12px 0; font-size: 12px; color: var(--color-text-secondary);
  line-height: 1.6;
}
.shortcut-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 6px; }
.shortcut-item { font-size: 11px; padding: 4px 8px; background: var(--color-bg-subtle); border-radius: var(--radius-sm); }
kbd {
  display: inline-block; padding: 1px 5px; font-size: 10px; font-family: 'JetBrains Mono', monospace;
  background: var(--color-bg-panel); border: 1px solid var(--color-border); border-radius: 3px; margin-right: 4px;
}
</style>

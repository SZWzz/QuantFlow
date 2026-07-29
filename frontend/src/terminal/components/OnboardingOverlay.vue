<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ done: []; action: [step: number] }>()

const steps = [
  { title: '欢迎使用 QuantFlow', desc: '你的双模式量化终端。我们先快速熟悉一下。' },
  { title: '打开行情面板', desc: '点击下一步打开 Watchlist 面板，查看实时行情。' },
  { title: '搜索任意标的', desc: '按 Ctrl+K 打开命令面板，输入标的代码即可搜索。' },
  { title: '管理投资组合', desc: '查看持仓、盈亏和风险指标，一站式监控。' },
  { title: '完成 🎉', desc: '可以随时按 Ctrl+W 切换工作流模式。开始探索吧！' },
]

const currentStep = ref(0)
function next() {
  if (currentStep.value < steps.length - 1) {
    emit('action', currentStep.value)
    currentStep.value++
  }
}
function skip() { emit('done') }
function finish() { emit('done') }
</script>

<template>
  <div class="onboarding-overlay" data-testid="onboarding-overlay">
    <div class="onboarding-card">
      <div class="onboarding-steps">
        <div v-for="(_, i) in steps" :key="i" class="onboarding-dot" :class="{ 'onboarding-dot--active': i === currentStep }" />
      </div>
      <h2>{{ steps[currentStep].title }}</h2>
      <p>{{ steps[currentStep].desc }}</p>
      <div class="onboarding-actions">
        <button data-testid="onboarding-skip" @click="skip">跳过</button>
        <button v-if="currentStep < steps.length - 1" data-testid="onboarding-next" @click="next">下一步</button>
        <button v-else data-testid="onboarding-done" @click="finish">完成</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.onboarding-overlay {
  position: fixed; inset: 0; z-index: 9999;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0,0,0,0.5); backdrop-filter: blur(4px);
}
.onboarding-card {
  background: var(--surface, #fff); border: 1px solid var(--border, #e0e0e0);
  border-radius: 12px; padding: 32px; max-width: 420px; width: 90%;
  box-shadow: 0 8px 32px rgba(0,0,0,0.15);
}
.onboarding-steps { display: flex; justify-content: center; gap: 8px; margin-bottom: 20px; }
.onboarding-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--border); }
.onboarding-dot--active { background: var(--accent); width: 24px; border-radius: 4px; }
.onboarding-card h2 { font-size: 20px; font-weight: 600; margin: 0 0 8px; }
.onboarding-card p { font-size: 14px; color: var(--muted, #666); margin: 0 0 24px; }
.onboarding-actions { display: flex; justify-content: flex-end; gap: 12px; }
.onboarding-actions button { padding: 8px 20px; border: none; border-radius: 6px; cursor: pointer; }
[data-testid="onboarding-next"],[data-testid="onboarding-done"] { background: var(--accent); color: #fff; }
</style>

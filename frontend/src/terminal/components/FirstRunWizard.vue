<script setup lang="ts">
import { onMounted } from 'vue'
import { useFirstRun, FIRST_RUN_STEPS } from '@/lib/composables/useFirstRun'

const wizard = useFirstRun()

onMounted(() => {
  wizard.check()
})

const current = () => FIRST_RUN_STEPS[wizard.currentStep.value]
</script>

<template>
  <Teleport to="body">
    <div v-if="wizard.show.value" class="wizard-overlay">
      <div class="wizard-card">
        <div class="wizard-header">
          <span class="wizard-step">{{ wizard.currentStep.value + 1 }} / {{ FIRST_RUN_STEPS.length }}</span>
          <button class="wizard-skip" @click="wizard.dismiss()">跳过</button>
        </div>

        <div class="wizard-body">
          <h2>{{ current().title }}</h2>
          <p>{{ current().description }}</p>
        </div>

        <div class="wizard-footer">
          <div class="wizard-dots">
            <span
              v-for="(_, i) in FIRST_RUN_STEPS"
              :key="i"
              class="dot"
              :class="{ active: i === wizard.currentStep.value }"
            />
          </div>
          <div class="wizard-actions">
            <button
              v-if="wizard.currentStep.value > 0"
              class="btn-prev"
              @click="wizard.prev()"
            >上一步</button>
            <button class="btn-next" @click="wizard.next()">
              {{ current().action || (wizard.currentStep.value < FIRST_RUN_STEPS.length - 1 ? '下一步' : '完成') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.wizard-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex; align-items: center; justify-content: center;
  z-index: var(--z-modal);
}
.wizard-card {
  background: var(--color-bg-app);
  border: 1px solid var(--color-border);
  border-radius: 16px;
  width: 420px; max-width: 90vw;
  padding: 24px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.4);
}
.wizard-header {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;
}
.wizard-step { font-size: 12px; color: var(--color-text-tertiary); font-weight: 600; }
.wizard-skip {
  background: none; border: none; color: var(--color-text-tertiary);
  font-size: 12px; cursor: pointer;
}
.wizard-body { margin-bottom: 24px; }
.wizard-body h2 { font-size: 18px; margin-bottom: 8px; }
.wizard-body p { font-size: 14px; color: var(--color-text-secondary); line-height: 1.6; }
.wizard-footer { display: flex; justify-content: space-between; align-items: center; }
.wizard-dots { display: flex; gap: 6px; }
.dot {
  width: 8px; height: 8px; border-radius: 50%;
  background: var(--color-border); transition: all 0.2s;
}
.dot.active { background: var(--color-accent); width: 24px; border-radius: 4px; }
.wizard-actions { display: flex; gap: 8px; }
.btn-prev {
  padding: 8px 16px; font-size: 13px;
  background: var(--color-bg-subtle); color: var(--color-text-secondary);
  border: 1px solid var(--color-border); border-radius: var(--radius-sm); cursor: pointer;
}
.btn-next {
  padding: 8px 20px; font-size: 13px; font-weight: 600;
  background: var(--color-accent); color: #fff;
  border: none; border-radius: var(--radius-sm); cursor: pointer;
}
</style>

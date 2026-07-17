import { ref } from 'vue'

const STORAGE_KEY = 'quantflow-first-run-completed'

export interface FirstRunStep {
  id: string
  title: string
  description: string
  action?: string  // optional action label
}

export const FIRST_RUN_STEPS: FirstRunStep[] = [
  { id: 'welcome', title: '欢迎使用 QuantFlow', description: '双模式量化金融终端 — 彭博式面板终端 + 可视化工作流编排' },
  { id: 'terminal', title: '终端模式', description: '使用 Ctrl+K 打开命令面板，Ctrl+Shift+字母 快速打开面板。右上角按钮切换面板和设置。' },
  { id: 'workflow', title: '工作流模式', description: '点击 Workflow 按钮进入可视化工作流编辑器。拖拽节点到画布，连线搭建策略。右键节点设置参数。' },
  { id: 'data', title: '数据配置', description: '在 API 密钥管理中配置行情源和券商。A股/港股/美股/加密 多市场支持，免费数据源默认可用。' },
  { id: 'ready', title: '准备就绪', description: '你现在可以开始使用了！建议先打开 Watchlist 面板添加自选股，或在工作流模式创建第一个策略。', action: '开始使用' },
]

export function useFirstRun() {
  const show = ref(false)
  const currentStep = ref(0)

  function check() {
    try {
      const completed = localStorage.getItem(STORAGE_KEY)
      show.value = completed !== 'true'
    } catch {
      show.value = true
    }
  }

  function next() {
    if (currentStep.value < FIRST_RUN_STEPS.length - 1) {
      currentStep.value++
    } else {
      dismiss()
    }
  }

  function prev() {
    if (currentStep.value > 0) {
      currentStep.value--
    }
  }

  function dismiss() {
    try {
      localStorage.setItem(STORAGE_KEY, 'true')
    } catch {}
    show.value = false
  }

  return { show, currentStep, check, next, prev, dismiss }
}

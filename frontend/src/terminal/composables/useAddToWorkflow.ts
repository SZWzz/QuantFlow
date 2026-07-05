import { computed, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPanelToNode } from '@/terminal/panelToNode'
import { useWorkflowStore } from '@/stores/workflow'
import { useSessionStore } from '@/stores/session'
import { confirmDialog } from '@/lib/wails'

function safeT(key: string, fallback: string): string {
  try {
    return useI18n().t(key)
  } catch {
    return fallback
  }
}

export function useAddToWorkflow(panelId: string, symbolRef?: Ref<string>) {
  const entry = getPanelToNode(panelId)
  const workflow = useWorkflowStore()
  const session = useSessionStore()

  async function addToWorkflow(symbol?: string) {
    if (!entry) return
    const sym = symbol || symbolRef?.value || '600519'
    workflow.addNode(entry.nodeType, { x: 200, y: 200 }, { symbol: sym })
    const ok = await confirmDialog(
      safeT('workflow.switch_confirm_body', 'The node has been added to the workflow canvas. Switch to workflow mode?'),
      safeT('workflow.switch_confirm_title', 'Switch to Workflow Mode?')
    )
    if (ok) {
      session.toggleMode()
    }
  }

  const control = computed(() => {
    if (!entry) return null
    return {
      icon: 'plus' as const,
      label: safeT('workflow.add_to_workflow', '+ Workflow'),
      title: `${safeT('workflow.add_to_workflow', '+ Workflow')}: ${entry.label}`,
      action: () => addToWorkflow(),
    }
  })

  return { control, addToWorkflow }
}
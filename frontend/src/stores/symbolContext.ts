import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

export interface LinkGroup {
  id: string
  color: string
  label: string
  activeSymbol: string | null
  symbolHistory: string[]
}

export const useSymbolContext = defineStore('symbolContext', () => {
  const linkGroups = reactive<Record<string, LinkGroup>>({
    'group-1': { id: 'group-1', color: '#ef4444', label: 'Red', activeSymbol: null, symbolHistory: [] },
    'group-2': { id: 'group-2', color: '#22c55e', label: 'Green', activeSymbol: null, symbolHistory: [] },
    'group-3': { id: 'group-3', color: '#f59e0b', label: 'Amber', activeSymbol: null, symbolHistory: [] },
    'group-4': { id: 'group-4', color: '#3b82f6', label: 'Blue', activeSymbol: null, symbolHistory: [] },
  })

  const activeGroupId = ref('group-1')
  const panelGroups = reactive<Record<string, { groupId: string; linked: boolean }>>({})

  function setGroupSymbol(groupId: string, symbol: string) {
    const group = linkGroups[groupId]
    if (!group || !symbol) return
    const s = symbol.trim().toUpperCase()
    if (s !== group.activeSymbol) {
      group.activeSymbol = s
      group.symbolHistory = [s, ...group.symbolHistory.filter(h => h !== s)].slice(0, 10)
    }
  }

  function getGroupSymbol(groupId: string): string | null {
    return linkGroups[groupId]?.activeSymbol ?? null
  }

  function setActiveGroup(groupId: string) {
    if (linkGroups[groupId]) activeGroupId.value = groupId
  }

  function getOrCreatePanelGroup(panelId: string): { groupId: string; linked: boolean } {
    if (!panelGroups[panelId]) {
      panelGroups[panelId] = { groupId: activeGroupId.value, linked: true }
    }
    return panelGroups[panelId]
  }

  function setPanelGroup(panelId: string, groupId: string) {
    panelGroups[panelId] = { groupId, linked: true }
  }

  function setPanelLinked(panelId: string, linked: boolean) {
    if (panelGroups[panelId]) panelGroups[panelId].linked = linked
  }

  function getPanelGroupId(panelId: string): string {
    return panelGroups[panelId]?.groupId || 'group-1'
  }

  function getActiveSymbolForPanel(panelId: string): string | null {
    const pg = panelGroups[panelId]
    if (!pg || !pg.linked) return null
    return linkGroups[pg.groupId]?.activeSymbol ?? null
  }

  function initFromLegacy(legacySymbol: string | null) {
    if (legacySymbol && !linkGroups['group-1'].activeSymbol) {
      linkGroups['group-1'].activeSymbol = legacySymbol
    }
  }

  return {
    linkGroups, activeGroupId, panelGroups,
    setGroupSymbol, getGroupSymbol, setActiveGroup,
    getOrCreatePanelGroup, setPanelGroup, setPanelLinked,
    getPanelGroupId, getActiveSymbolForPanel, initFromLegacy,
  }
})

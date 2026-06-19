export interface DockTabState {
  id: string
  panelId: string
  label: string
  icon: string
  params?: Record<string, any>
}

export interface DockLayoutTree {
  id: string
  type: 'container' | 'tab'
  // Container properties
  direction?: 'row' | 'column'
  splitRatios?: number[]
  children?: DockLayoutTree[]
  // Tab properties
  tabs?: DockTabState[]
  activeTab?: string
}

export function createTabLeaf(id: string, tab: DockTabState): DockLayoutTree {
  return {
    id,
    type: 'tab',
    tabs: [tab],
    activeTab: tab.id,
  }
}

export function createContainer(
  id: string,
  direction: 'row' | 'column',
  children: DockLayoutTree[]
): DockLayoutTree {
  const equalRatio = 1 / children.length
  return {
    id,
    type: 'container',
    direction,
    splitRatios: children.map(() => equalRatio),
    children,
  }
}

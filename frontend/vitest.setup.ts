import { vi } from 'vitest'

// Mock Wails runtime
vi.mock('@wailsio/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
  Call: vi.fn().mockResolvedValue({}),
}))

// Mock window.go bridge
;(window as any).go = {
  main: {
    App: {
      GetMarketOverview: vi.fn().mockResolvedValue([]),
      GetQuote: vi.fn().mockResolvedValue(null),
      ListNodes: vi.fn().mockResolvedValue([]),
      GetNodePorts: vi.fn().mockResolvedValue({ inputs: [], outputs: [] }),
    },
  },
}

// Prevent Wails drag.js window reference after teardown
const originalSetTimeout = global.setTimeout
global.setTimeout = ((fn: any, ms: any, ...args: any[]) => {
  if (typeof fn === 'string' && fn.includes('window')) return 0
  if (fn.toString().includes('window')) return 0
  return originalSetTimeout(fn, ms, ...args)
}) as any

// Global mocks for test environment
global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

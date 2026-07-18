import { vi } from 'vitest'
import { config } from '@vue/test-utils'
import { mockWailsIPC, mockWebSocket, mockI18n } from './src/test-utils/mocks'

// Re-export the t function from mocks for consistent $t mock
import { t as mockT } from './src/test-utils/mocks'

mockWailsIPC()
mockWebSocket()
mockI18n()

const originalSetTimeout = global.setTimeout
global.setTimeout = ((fn: any, ms: any, ...args: any[]) => {
  if (typeof fn === 'string' && fn.includes('window')) return 0
  if (fn.toString().includes('window')) return 0
  return originalSetTimeout(fn, ms, ...args)
}) as any

global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
  clearRect: vi.fn(),
  fillRect: vi.fn(),
  getImageData: vi.fn(() => ({ data: [] })),
  putImageData: vi.fn(),
  createImageData: vi.fn(() => []),
  setTransform: vi.fn(),
  drawImage: vi.fn(),
  save: vi.fn(),
  fillText: vi.fn(),
  restore: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  closePath: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  arc: vi.fn(),
  rect: vi.fn(),
  clip: vi.fn(),
  arcTo: vi.fn(),
  ellipse: vi.fn(),
  quadraticCurveTo: vi.fn(),
  bezierCurveTo: vi.fn(),
  strokeText: vi.fn(),
  setLineDash: vi.fn(),
  createLinearGradient: vi.fn(() => ({ addColorStop: vi.fn() })),
  createRadialGradient: vi.fn(() => ({ addColorStop: vi.fn() })),
  createPattern: vi.fn(() => ({})),
  measureText: vi.fn(() => ({ width: 10 })),
  canvas: { width: 0, height: 0 },
})) as any

// Use the same t function from mocks.ts — single source of truth
config.global.mocks.$t = mockT

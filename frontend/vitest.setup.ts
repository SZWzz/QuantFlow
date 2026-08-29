import { vi } from 'vitest'
import { config } from '@vue/test-utils'
import { mockWailsIPC, mockWebSocket, mockI18n } from './src/test-utils/mocks'

// Re-export the t function from mocks for consistent $t mock
import { t as mockT } from './src/test-utils/mocks'

mockWailsIPC()
mockWebSocket()
mockI18n()

// 全局 mock @wailsio/runtime：真实模块在 import 时（dist/index.js → drag.js）启动
// window.setInterval 轮询 Wails 环境，jsdom 环境 teardown 后定时器回调访问 window
// 抛 ReferenceError，造成 vitest "Uncaught Exception"（时序依赖，间歇性失败）。
// 单个测试文件（如 lib/__tests__/wails.test.ts）的 vi.mock 会覆盖此全局 mock。
vi.mock('@wailsio/runtime', () => ({
  Call: { ByName: vi.fn(async () => null) },
  Dialogs: {
    Question: vi.fn(async () => '确定'),
    Info: vi.fn(async () => ''),
    Warning: vi.fn(async () => ''),
    Error: vi.fn(async () => ''),
  },
  Events: {
    On: vi.fn(() => () => {}),
    Once: vi.fn(() => () => {}),
    Off: vi.fn(),
    Emit: vi.fn(),
  },
}))

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

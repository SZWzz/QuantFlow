// Global mocks for test environment
global.ResizeObserver = class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as any

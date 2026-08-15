import { test as base, expect } from '@playwright/test'
import { setupMocks } from './mock-app'

export const test = base.extend({
  mockPage: async ({ page }, use) => {
    await setupMocks(page)
    await page.goto('/')
    // Wait for Vue to mount
    await page.waitForSelector('.terminal-mode, .workflow-mode', { timeout: 10000 }).catch(() => {})
    await use(page)
  },
  // 预置 watchlist 面板布局（默认布局只有欢迎页，panel 测试需要种子布局）
  mockPageWithWatchlist: async ({ page }, use) => {
    await setupMocks(page)
    await page.addInitScript(() => {
      try {
        localStorage.setItem('quantflow-layout', JSON.stringify({
          id: 'root',
          type: 'tab',
          tabs: [{ id: 'watchlist-1', panelId: 'watchlist', label: '自选股', icon: '📊' }],
          activeTab: 'watchlist-1',
        }))
      } catch {}
    })
    await page.goto('/')
    await page.waitForSelector('.terminal-mode, .workflow-mode', { timeout: 10000 }).catch(() => {})
    await use(page)
  },
})

export { expect }

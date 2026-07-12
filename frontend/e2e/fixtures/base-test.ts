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
})

export { expect }

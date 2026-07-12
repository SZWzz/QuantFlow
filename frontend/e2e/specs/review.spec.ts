import { test, expect } from '../fixtures/base-test'

test.describe('Review Flow', () => {
  test('app loads and renders terminal mode', async ({ mockPage: page }) => {
    await expect(page.locator('.terminal-mode, .workflow-mode')).toBeVisible({ timeout: 15000 })
  })

  test('status bar shows system info', async ({ mockPage: page }) => {
    await expect(page.locator('.status-bar')).toBeVisible({ timeout: 15000 })
  })

  test('header renders with market tabs', async ({ mockPage: page }) => {
    await expect(page.locator('.terminal-header, header')).toBeVisible({ timeout: 15000 })
  })

  test('DockView container is present', async ({ mockPage: page }) => {
    await expect(page.locator('.dock-view, .dock-container')).toBeVisible({ timeout: 15000 })
  })
})

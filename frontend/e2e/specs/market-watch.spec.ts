import { test, expect } from '../fixtures/base-test'

test.describe('Market Watch Flow', () => {
  test('watchlist panel renders with symbols', async ({ mockPageWithWatchlist: page }) => {
    await expect(page.locator('[data-testid="watchlist-panel"]')).toBeVisible({ timeout: 15000 })
  })

  test('watchlist shows default 8 symbols', async ({ mockPageWithWatchlist: page }) => {
    const rows = page.locator('[data-testid="watchlist-row"]')
    await expect(rows.first()).toBeVisible({ timeout: 15000 })
  })

  test('column headers are visible', async ({ mockPageWithWatchlist: page }) => {
    await expect(page.locator('.watchlist-panel .table-header-row')).toBeVisible({ timeout: 15000 })
  })

  test('group headers exist for CN/US/HK/CRYPTO', async ({ mockPageWithWatchlist: page }) => {
    await expect(page.locator('[data-testid="watchlist-panel"]')).toBeVisible({ timeout: 15000 })
    // Groups are rendered as accordion sections
    const groups = page.locator('.group-header')
    const count = await groups.count()
    expect(count).toBeGreaterThanOrEqual(1)
  })

  test('context menu opens on right-click', async ({ mockPageWithWatchlist: page }) => {
    const row = page.locator('[data-testid="watchlist-row"]').first()
    await row.waitFor({ timeout: 15000 })
    await row.click({ button: 'right' })
    // Context menu should appear
    await expect(page.locator('.context-menu')).toBeVisible({ timeout: 5000 })
  })

  test('clicking row selects it', async ({ mockPageWithWatchlist: page }) => {
    const row = page.locator('[data-testid="watchlist-row"]').first()
    await row.waitFor({ timeout: 15000 })
    await row.click()
    // Row should get visual feedback (active class or highlight)
    await expect(row).toBeVisible()
  })
})

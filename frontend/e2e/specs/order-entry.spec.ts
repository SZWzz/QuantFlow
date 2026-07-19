import { test, expect, type Page } from '../fixtures/base-test'

/** Opens the order entry panel via the command bar (Ctrl+K → 下单). */
async function openOrderPanel(page: Page) {
  await page.keyboard.press('Control+k')
  const input = page.locator('.command-bar .search-input')
  await expect(input).toBeVisible({ timeout: 5000 })
  await input.fill('下单')
  await page.locator('.command-bar .result-item').first().click()
  await expect(page.locator('[data-testid="order-symbol-input"]')).toBeVisible({ timeout: 10000 })
}

/** Reads the recorded PlaceOrder/PlaceOrderWithStop calls from the mock. */
async function placedOrders(page: Page): Promise<any[]> {
  return page.evaluate(() => (window as any).__placedOrders || [])
}

test.describe('Order Entry Flow', () => {
  test('opens with empty symbol when no linked context (no demo default)', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await expect(page.locator('[data-testid="order-symbol-input"]')).toHaveValue('')
    // Cannot submit without a symbol
    await expect(page.locator('[data-testid="order-place-btn"]')).toBeDisabled()
  })

  test('quantity chips fill quantity', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await page.locator('[data-testid="order-qty-chip-500"]').click()
    await expect(page.locator('[data-testid="order-quantity-input"]')).toHaveValue('500')
  })

  test('symbol linkage: picking a symbol in SymbolBar updates the panel', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    // Pick 600519 via the SymbolBar search
    const search = page.locator('.symbol-bar .symbol-search .search-input')
    await search.click()
    await search.fill('600519')
    const item = page.locator('.symbol-bar .symbol-search .dropdown-item').first()
    await expect(item).toBeVisible({ timeout: 5000 })
    await item.dispatchEvent('mousedown')
    // Panel follows the linked group symbol
    await expect(page.locator('[data-testid="order-symbol-input"]')).toHaveValue('600519', { timeout: 5000 })
  })

  test('confirmation flow: summary → submitting disabled → toast → mock receives order', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await page.locator('[data-testid="order-symbol-input"]').fill('600519')
    // Quote auto-fills the limit price (mock returns 1650 for 600519)
    await expect(page.locator('[data-testid="order-price-input"]')).toHaveValue('1650', { timeout: 5000 })
    await page.locator('[data-testid="order-qty-chip-100"]').click()

    await page.locator('[data-testid="order-place-btn"]').click()
    // Inline confirmation summary replaces the form
    const confirmView = page.locator('[data-testid="order-confirm-view"]')
    await expect(confirmView).toBeVisible()
    await expect(confirmView).toContainText('600519')
    await expect(confirmView).toContainText('100')
    await expect(confirmView).toContainText('¥')

    const confirmBtn = page.locator('[data-testid="order-confirm-btn"]')
    await confirmBtn.click()
    // Submitting state: buttons disabled with 提交中…
    await expect(confirmBtn).toBeDisabled()
    await expect(confirmBtn).toHaveText('提交中…')
    await expect(page.locator('[data-testid="order-cancel-btn"]')).toBeDisabled()

    // Success toast appears and form returns
    await expect(page.locator('[data-test="toast"]').first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator('[data-testid="order-symbol-input"]')).toBeVisible()

    const orders = await placedOrders(page)
    expect(orders.length).toBe(1)
    expect(orders[0]).toMatchObject({ symbol: '600519', side: 'buy', orderType: 'limit', qty: 100, price: 1650, stopPrice: 0 })
  })

  test('cancel returns to the form without submitting', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await page.locator('[data-testid="order-symbol-input"]').fill('600519')
    await page.locator('[data-testid="order-place-btn"]').click()
    await expect(page.locator('[data-testid="order-confirm-view"]')).toBeVisible()
    await page.locator('[data-testid="order-cancel-btn"]').click()
    await expect(page.locator('[data-testid="order-symbol-input"]')).toBeVisible()
    expect(await placedOrders(page)).toHaveLength(0)
  })

  test('stopPrice is forwarded to the backend for stop orders', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await page.locator('[data-testid="order-symbol-input"]').fill('600519')
    await expect(page.locator('[data-testid="order-price-input"]')).toHaveValue('1650', { timeout: 5000 })
    await page.locator('[data-testid="order-type-select"]').selectOption('stop')
    await page.locator('[data-testid="order-stop-price-input"]').fill('1600')
    await page.locator('[data-testid="order-place-btn"]').click()
    // Summary shows the stop price
    await expect(page.locator('[data-testid="order-confirm-view"]')).toContainText('1600')
    await page.locator('[data-testid="order-confirm-btn"]').click()
    await expect(page.locator('[data-test="toast"]').first()).toBeVisible({ timeout: 5000 })

    const orders = await placedOrders(page)
    expect(orders.length).toBe(1)
    expect(orders[0].orderType).toBe('stop')
    expect(orders[0].stopPrice).toBe(1600)
  })

  test('failed order shows an error toast', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    await page.locator('[data-testid="order-symbol-input"]').fill('FAIL')
    await page.locator('[data-testid="order-price-input"]').fill('100')
    await page.locator('[data-testid="order-place-btn"]').click()
    await page.locator('[data-testid="order-confirm-btn"]').click()
    const toast = page.locator('[data-test="toast"]').first()
    await expect(toast).toBeVisible({ timeout: 5000 })
    await expect(toast).toContainText('下单失败')
    // Form stays in confirm state so the user can retry or cancel
    await expect(page.locator('[data-testid="order-confirm-btn"]')).toBeEnabled()
  })

  test('Ctrl+Enter triggers submit and confirm', async ({ mockPage: page }) => {
    await openOrderPanel(page)
    const symbolInput = page.locator('[data-testid="order-symbol-input"]')
    await symbolInput.fill('600519')
    await expect(page.locator('[data-testid="order-price-input"]')).toHaveValue('1650', { timeout: 5000 })
    // Ctrl+Enter from within the panel opens the confirmation
    await symbolInput.press('Control+Enter')
    await expect(page.locator('[data-testid="order-confirm-view"]')).toBeVisible()
    // Ctrl+Enter again confirms the order
    await page.locator('[data-testid="order-confirm-btn"]').press('Control+Enter')
    await expect(page.locator('[data-test="toast"]').first()).toBeVisible({ timeout: 5000 })
    expect(await placedOrders(page)).toHaveLength(1)
  })
})

import { test } from '../fixtures/base-test'

// 视觉走查截图（一次性验证脚本，不纳入回归）
// 输出到 ../../.superpowers/sdd/screenshots/
const OUT = '../.superpowers/sdd/screenshots'

test('visual walkthrough screenshots', async ({ mockPage: page }) => {
  // mock 环境不渲染 watchlist 面板（main 上亦然），等待外壳即可
  await page.locator('.terminal-mode').waitFor({ timeout: 20000 })
  // 关掉首次运行向导（mock 环境恒为首跑）
  const skip = page.locator('button:has-text("跳过")')
  if (await skip.count()) await skip.first().click()
  await page.waitForTimeout(3000)

  // 1. 亮主题 · 默认密度
  await page.screenshot({ path: `${OUT}/01-light-default.png` })

  // 2. 暗主题
  await page.evaluate(() => document.body.classList.add('theme-dark'))
  await page.waitForTimeout(600)
  await page.screenshot({ path: `${OUT}/02-dark.png` })
  await page.evaluate(() => document.body.classList.remove('theme-dark'))

  // 3. 紧凑密度
  await page.evaluate(() => document.body.classList.add('density-compact'))
  await page.waitForTimeout(600)
  await page.screenshot({ path: `${OUT}/03-compact.png` })
  await page.evaluate(() => document.body.classList.remove('density-compact'))

  // 4. 宽松密度
  await page.evaluate(() => document.body.classList.add('density-comfortable'))
  await page.waitForTimeout(600)
  await page.screenshot({ path: `${OUT}/04-comfortable.png` })
  await page.evaluate(() => document.body.classList.remove('density-comfortable'))

  // 5. 自选股面板特写（mock 环境不渲染则跳过）
  const wl = page.locator('[data-testid="watchlist-panel"]')
  if (await wl.count()) {
    await wl.screenshot({ path: `${OUT}/05-watchlist.png` })
  }
})

/**
 * stats.ts — Pure frontend statistical functions for QuantFlow panels.
 * Zero external dependencies. All functions are synchronous and non-blocking.
 *
 * Used by: CorrelationPanel, DistributionPanel, MonteCarloPanel, EquityCurvePanel
 */

// ---------------------------------------------------------------------------
// Pearson Correlation Matrix
// ---------------------------------------------------------------------------

/**
 * Compute Pearson correlation matrix from a 2D array of returns.
 * @param returns  returns[i] = array of daily returns for asset i (same length)
 * @returns N×N matrix where [i][j] = Pearson r between asset i and asset j
 */
export function pearsonMatrix(returns: number[][]): number[][] {
  const n = returns.length
  if (n === 0) return []
  const matrix: number[][] = Array.from({ length: n }, () => Array(n).fill(1))
  for (let i = 0; i < n; i++) {
    for (let j = i + 1; j < n; j++) {
      const r = pearson(returns[i], returns[j])
      matrix[i][j] = r
      matrix[j][i] = r
    }
  }
  return matrix
}

function pearson(xs: number[], ys: number[]): number {
  const n = Math.min(xs.length, ys.length)
  if (n < 2) return 0
  const mx = mean(xs.slice(0, n))
  const my = mean(ys.slice(0, n))
  let cov = 0, vx = 0, vy = 0
  for (let i = 0; i < n; i++) {
    const dx = xs[i] - mx, dy = ys[i] - my
    cov += dx * dy; vx += dx * dx; vy += dy * dy
  }
  if (vx === 0 || vy === 0) return 0
  return cov / Math.sqrt(vx * vy)
}

function mean(arr: number[]): number {
  return arr.reduce((a, b) => a + b, 0) / arr.length
}

// ---------------------------------------------------------------------------
// Histogram Bins
// ---------------------------------------------------------------------------

/** Result of histogram binning */
export interface HistogramBin {
  x: number   // bin center
  y: number   // frequency count
}

/**
 * Bin data into `binCount` equal-width bins.
 */
export function histogramBins(data: number[], binCount: number = 30): HistogramBin[] {
  if (data.length === 0) return []
  const min = Math.min(...data), max = Math.max(...data)
  const range = max - min || 1
  const binWidth = range / binCount
  const bins = new Array<number>(binCount).fill(0)
  for (const v of data) {
    let idx = Math.floor((v - min) / binWidth)
    if (idx >= binCount) idx = binCount - 1
    if (idx < 0) idx = 0
    bins[idx]++
  }
  return bins.map((y, i) => ({ x: min + binWidth * (i + 0.5), y }))
}

// ---------------------------------------------------------------------------
// Geometric Brownian Motion Monte Carlo
// ---------------------------------------------------------------------------

export interface GBMInput {
  initialCapital: number
  annualReturn: number   // e.g., 0.08 = 8%
  annualVol: number       // e.g., 0.20 = 20%
  years: number
  simulations: number     // 100–5000
  stepsPerYear?: number   // default 252
}

export interface GBMOutput {
  paths: number[][]       // [sim][step], each path includes initial capital at step 0
  terminalValues: number[]
  medianTerminal: number
  var95: number
  cvar95: number
  probLoss: number
  probDouble: number
}

/**
 * Simulate lognormal price paths using Geometric Brownian Motion.
 */
export function simulateGBM(input: GBMInput): GBMOutput {
  const { initialCapital, annualReturn, annualVol, years, simulations } = input
  const stepsPerYear = input.stepsPerYear ?? 252
  const totalSteps = Math.round(years * stepsPerYear)
  const dt = 1 / stepsPerYear
  const drift = (annualReturn - 0.5 * annualVol * annualVol) * dt
  const vol = annualVol * Math.sqrt(dt)

  const paths: number[][] = []
  const terminalValues: number[] = []

  for (let s = 0; s < simulations; s++) {
    const path: number[] = [initialCapital]
    for (let t = 1; t <= totalSteps; t++) {
      const z = boxMuller()
      const price = path[t - 1] * Math.exp(drift + vol * z)
      path.push(Math.max(price, 0.01))
    }
    paths.push(path)
    terminalValues.push(path[totalSteps])
  }

  const sorted = [...terminalValues].sort((a, b) => a - b)
  const medianTerminal = sorted[Math.floor(sorted.length / 2)]
  const varIdx = Math.floor(sorted.length * 0.05)
  const var95 = initialCapital - sorted[varIdx]
  const tail = sorted.slice(0, varIdx + 1)
  const cvar95 = initialCapital - (tail.reduce((a, b) => a + b, 0) / tail.length)
  const probLoss = terminalValues.filter(v => v < initialCapital).length / simulations
  const probDouble = terminalValues.filter(v => v >= initialCapital * 2).length / simulations

  return { paths, terminalValues, medianTerminal, var95, cvar95, probLoss, probDouble }
}

/** Box-Muller transform — generate standard normal random variable */
function boxMuller(): number {
  let u1 = 0, u2 = 0
  while (u1 === 0) u1 = Math.random()
  while (u2 === 0) u2 = Math.random()
  return Math.sqrt(-2 * Math.log(u1)) * Math.cos(2 * Math.PI * u2)
}

// ---------------------------------------------------------------------------
// Drawdown Computation
// ---------------------------------------------------------------------------

export interface DrawdownPoint {
  date?: string
  nav: number
  peak: number
  drawdown: number  // negative number, e.g. -0.15 = 15% drawdown
}

/**
 * Compute running drawdown from equity curve.
 * @param navs  Array of portfolio NAV values, chronological
 * @returns DrawdownPoint[] with peak and drawdown at each step
 */
export function computeDrawdowns(navs: number[]): DrawdownPoint[] {
  let peak = navs[0] ?? 0
  return navs.map(nav => {
    if (nav > peak) peak = nav
    return { nav, peak, drawdown: peak > 0 ? (nav - peak) / peak : 0 }
  })
}

// ---------------------------------------------------------------------------
// Sharpe Ratio
// ---------------------------------------------------------------------------

/**
 * Compute annualized Sharpe ratio from daily returns.
 * Assumes risk-free rate = 0.
 */
export function sharpeRatio(dailyReturns: number[]): number {
  if (dailyReturns.length < 2) return 0
  const avg = mean(dailyReturns)
  const variance = dailyReturns.reduce((s, r) => s + (r - avg) ** 2, 0) / (dailyReturns.length - 1)
  const stdDaily = Math.sqrt(variance)
  if (stdDaily === 0) return 0
  return (avg / stdDaily) * Math.sqrt(252)
}

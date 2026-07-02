/** 客户端技术指标计算 — 纯 TS，零依赖。 */

export interface BBResult { upper: (number | null)[]; middle: (number | null)[]; lower: (number | null)[] }
export interface MACDResult { dif: (number | null)[]; dea: (number | null)[]; hist: (number | null)[] }
export interface KDJResult { k: (number | null)[]; d: (number | null)[]; j: (number | null)[] }

/** Simple Moving Average */
export function sma(data: number[], period: number): (number | null)[] {
  const r: (number | null)[] = []
  for (let i = 0; i < data.length; i++) {
    if (i < period - 1) { r.push(null); continue }
    let s = 0
    for (let j = i - period + 1; j <= i; j++) s += data[j]
    r.push(s / period)
  }
  return r
}

/** Exponential Moving Average (同花顺风格: 首值种子, 从第1个值开始) */
export function ema(data: number[], period: number): number[] {
  const r: number[] = []
  const k = 2 / (period + 1)
  let prev = data[0]
  r.push(prev) // i=0: 首值种子
  for (let i = 1; i < data.length; i++) {
    prev = (data[i] - prev) * k + prev
    r.push(prev)
  }
  return r
}

/** Bollinger Bands */
export function bb(data: number[], period = 20, k = 2): BBResult {
  const m = sma(data, period)
  const upper: (number | null)[] = []
  const lower: (number | null)[] = []
  for (let i = 0; i < data.length; i++) {
    if (m[i] === null) { upper.push(null); lower.push(null); continue }
    let ss = 0
    const start = i - period + 1
    for (let j = start; j <= i; j++) ss += (data[j] - m[i]!) ** 2
    const std = Math.sqrt(ss / period)
    upper.push(m[i]! + k * std)
    lower.push(m[i]! - k * std)
  }
  return { middle: m, upper, lower }
}

/** MACD (同花顺风格: 首值种子 EMA, BAR=2×(DIFF-DEA)) */
export function macd(data: number[], fast = 12, slow = 26, signal = 9): MACDResult {
  const ef = ema(data, fast)
  const es = ema(data, slow)
  const dif: number[] = []
  for (let i = 0; i < data.length; i++) {
    dif.push(ef[i] - es[i])
  }
  const dea = ema(dif, signal)
  const hist: number[] = []
  for (let i = 0; i < data.length; i++) {
    hist.push((dif[i] - dea[i]) * 2) // 同花顺: BAR=2×(DIFF-DEA)
  }
  return { dif, dea, hist }
}

/** KDJ (同花顺/通达信风格: 递归加权平均, 首值种子50) */
export function kdj(close: number[], high: number[], low: number[], n = 9, m1 = 3, m2 = 3): KDJResult {
  const rsv: (number | null)[] = []
  for (let i = 0; i < close.length; i++) {
    if (i < n - 1) { rsv.push(null); continue }
    let hh = -Infinity, ll = Infinity
    for (let j = i - n + 1; j <= i; j++) {
      if (high[j] > hh) hh = high[j]
      if (low[j] < ll) ll = low[j]
    }
    rsv.push(hh === ll ? 50 : (close[i] - ll) / (hh - ll) * 100)
  }
  const kFull: (number | null)[] = []; const dFull: (number | null)[] = []; const jFull: (number | null)[] = []
  let prevK = 50, prevD = 50
  const k1 = 2 / (m1 + 1)
  const k2 = 2 / (m2 + 1)
  for (let i = 0; i < close.length; i++) {
    const r = rsv[i]
    if (r === null) { kFull.push(null); dFull.push(null); jFull.push(null); continue }
    const kv = (r - prevK) * k1 + prevK
    prevK = kv
    kFull.push(kv)
    const dv = (kv - prevD) * k2 + prevD
    prevD = dv
    dFull.push(dv)
    jFull.push(3 * kv - 2 * dv)
  }
  return { k: kFull, d: dFull, j: jFull }
}

/** RSI */
export function rsi(data: number[], period = 14): (number | null)[] {
  const r: (number | null)[] = [null]
  let gain = 0, loss = 0
  for (let i = 1; i < data.length; i++) {
    const diff = data[i] - data[i - 1]
    if (diff > 0) gain += diff; else loss -= diff
    if (i < period) { r.push(null); continue }
    if (i === period) {
      const ag = gain / period, al = loss / period
      r.push(al === 0 ? 100 : 100 - 100 / (1 + ag / al))
      continue
    }
    const prev = r[r.length - 1]!
    const prevRs = prev === 100 ? Infinity : prev / (100 - prev)
    const curGain = diff > 0 ? diff : 0
    const curLoss = diff < 0 ? -diff : 0
    const avgGain = (gain / period * (period - 1) + curGain) / period
    const avgLoss = (loss / period * (period - 1) + curLoss) / period
    gain = avgGain * period
    loss = avgLoss * period
    r.push(avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss))
  }
  return r
}

/** Williams %R */
export function wr(close: number[], high: number[], low: number[], period = 14): (number | null)[] {
  const r: (number | null)[] = []
  for (let i = 0; i < close.length; i++) {
    if (i < period - 1) { r.push(null); continue }
    let hh = -Infinity, ll = Infinity
    for (let j = i - period + 1; j <= i; j++) {
      if (high[j] > hh) hh = high[j]
      if (low[j] < ll) ll = low[j]
    }
    r.push(hh === ll ? -50 : (hh - close[i]) / (hh - ll) * -100)
  }
  return r
}

/** Parabolic Stop and Reverse (SAR) */
export function sar(high: number[], low: number[], close: number[], acceleration = 0.02, maxAcceleration = 0.2): (number | null)[] {
  const result: (number | null)[] = []
  if (high.length < 2) return high.map(() => null)

  let isLong = true
  let af = acceleration
  let ep = low[0]
  let sarVal = high[0]

  for (let i = 1; i < high.length; i++) {
    if (isLong) {
      sarVal = sarVal + af * (ep - sarVal)
      if (sarVal > low[i]) {
        isLong = false
        af = acceleration
        sarVal = ep = high[i]
        result.push(null)
        continue
      }
      if (high[i] > ep) {
        ep = high[i]
        af = Math.min(af + acceleration, maxAcceleration)
      }
    } else {
      sarVal = sarVal + af * (ep - sarVal)
      if (sarVal < high[i]) {
        isLong = true
        af = acceleration
        sarVal = ep = low[i]
        result.push(null)
        continue
      }
      if (low[i] < ep) {
        ep = low[i]
        af = Math.min(af + acceleration, maxAcceleration)
      }
    }
    result.push(sarVal)
  }

  while (result.length < high.length) {
    result.unshift(null)
  }

  return result
}

/** Commodity Channel Index (CCI) */
export function cci(high: number[], low: number[], close: number[], period = 20): (number | null)[] {
  const tp = high.map((h, i) => (h + low[i] + close[i]) / 3)
  const result: (number | null)[] = []

  for (let i = 0; i < tp.length; i++) {
    if (i < period - 1) {
      result.push(null)
      continue
    }
    const slice = tp.slice(i - period + 1, i + 1)
    const mean = slice.reduce((a, b) => a + b, 0) / period
    const mad = slice.reduce((sum, v) => sum + Math.abs(v - mean), 0) / period
    result.push(mad === 0 ? 0 : (tp[i] - mean) / (0.015 * mad))
  }
  return result
}

/** On-Balance Volume (OBV) */
export function obv(close: number[], volume: number[]): number[] {
  if (!close.length || !volume.length) return []
  const result: number[] = [volume[0]]
  for (let i = 1; i < close.length; i++) {
    if (close[i] > close[i - 1]) {
      result.push(result[i - 1] + volume[i])
    } else if (close[i] < close[i - 1]) {
      result.push(result[i - 1] - volume[i])
    } else {
      result.push(result[i - 1])
    }
  }
  return result
}

/** Memoization wrapper for indicator computations */
export function createIndicatorCache() {
  const cache = new Map<string, any>()
  return {
    getCached<T>(key: string, fn: () => T): T {
      if (cache.has(key)) return cache.get(key) as T
      const r = fn()
      cache.set(key, r)
      return r
    },
    clear() { cache.clear() },
    delete(key: string) { cache.delete(key) },
  }
}

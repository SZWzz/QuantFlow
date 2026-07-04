import type { ECBasicOption } from 'echarts/types/dist/shared'
import type { ChartThemeColors } from '@/lib/composables/useChartTheme'
import type { MinuteTick } from '@/lib/composables/useWailsApp'
import type { EventMarker } from '@/lib/chart/EventMarker'

export interface TradeSignal {
  dataIndex: number
  direction: 'buy' | 'sell'
  price: number
  label?: string
}
import { sma, bb, macd, kdj, rsi, wr, ema, sar, cci, obv } from '@/lib/composables/useIndicators'
import { marketUpColor, marketDownColor } from '@/lib/composables/useMarketColors'
import { i18n } from '@/lib/i18n'

export interface KlineDataItem {
  date: string; open: number; high: number; low: number; close: number; volume: number
}

export interface IndicatorCache {
  getCached<T>(key: string, fn: () => T): T
  clear(): void
  delete(key: string): void
}

/** Shared grid/axis/tooltip base config — spread into per-chart options */
export function mergeBaseOption(theme: ChartThemeColors) {
  return {
    animation: false,
    backgroundColor: 'transparent',
    grid: [
      { left: 54, right: 16, top: 8, height: '52%' },
      { left: 54, right: 16, top: '68%', height: '26%' },
    ],
    xAxis: [{
      type: 'category' as const, show: true,
      axisLine: { lineStyle: { color: theme.splitColor } },
      axisLabel: { show: false }, gridIndex: 0,
      splitLine: { show: true, lineStyle: { color: theme.splitColor, type: 'dashed' as const } },
    }],
    yAxis: [{
      scale: true,
      splitLine: { show: true, lineStyle: { color: theme.splitColor, type: 'dashed' as const } },
      axisLabel: { color: theme.axisColor, fontSize: 10 },
    }],
    tooltip: { trigger: 'axis' as const, axisPointer: { type: 'cross' as const }, backgroundColor: theme.tooltipBg, borderColor: theme.splitColor },
    dataZoom: [{ type: 'inside' as const, xAxisIndex: [0, 1] }],
  }
}

export function buildKlineOption(
  data: KlineDataItem[],
  topOverlay: string,
  bottomMode: string,
  theme: ChartThemeColors,
  cache: IndicatorCache,
  symbol: string,
  interval: string,
  eventMarkers?: EventMarker[],
  indexOverlay?: { name: string; symbol: string; data: any[] } | null,
  tradeSignals?: TradeSignal[],
): ECBasicOption {
  if (!data.length) return {} as ECBasicOption

  const dates = data.map(d => d.date)
  const kdata = data.map(d => [d.open, d.close, d.low, d.high])
  const close = data.map(d => d.close)
  const high = data.map(d => d.high)
  const low = data.map(d => d.low)
  const upCol = marketUpColor(symbol)
  const downCol = marketDownColor(symbol)
  const vdata = data.map(d => ({
    value: d.volume / 10000,
    itemStyle: { color: d.close >= d.open ? upCol : downCol },
  }))

  const cacheKey = `${symbol}-${interval}-${data.length}-${topOverlay}-${bottomMode}`

  const t = i18n.global.t
  const series: any[] = [
    {
      type: 'candlestick', name: t('kline.kline'),
      data: kdata, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
      itemStyle: { color: upCol, color0: downCol, borderColor: upCol, borderColor0: downCol },
    },
  ]

  if (topOverlay === 'ma') {
    ;[5, 10, 20, 60].forEach(p => {
      series.push({
        type: 'line', name: `MA${p}`, data: cache.getCached(`sma-${cacheKey}-${p}`, () => sma(close, p)),
        gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
        symbol: 'none', lineStyle: { width: 1 },
      })
    })
  } else if (topOverlay === 'bb') {
    const b = cache.getCached(`bb-${cacheKey}-20-2`, () => bb(close, 20, 2))
    series.push({ type: 'line', name: 'BB上轨', data: b.upper, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#4caf50' } })
    series.push({ type: 'line', name: 'BB中轨', data: b.middle, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } })
    series.push({ type: 'line', name: 'BB下轨', data: b.lower, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#4caf50' } })
  } else if (topOverlay === 'sar') {
    const sarData = cache.getCached(`sar-${cacheKey}`, () => sar(high, low, close))
    series.push({
      type: 'scatter', name: 'SAR', data: sarData,
      gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0,
      symbol: 'circle', symbolSize: 4,
      itemStyle: { color: '#f06292' },
    })
  } else if (topOverlay === 'ema') {
    const ema12 = cache.getCached(`ema-${cacheKey}-12`, () => ema(close, 12))
    const ema26 = cache.getCached(`ema-${cacheKey}-26`, () => ema(close, 26))
    series.push({ type: 'line', name: 'EMA12', data: ema12, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#42a5f5' } })
    series.push({ type: 'line', name: 'EMA26', data: ema26, gridIndex: 0, xAxisIndex: 0, yAxisIndex: 0, symbol: 'none', lineStyle: { width: 1, color: '#ff7043' } })
  }

  if (bottomMode === 'volume') {
    series.push({ type: 'bar', name: t('kline.volume'), data: vdata, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1 })
  } else if (bottomMode === 'macd') {
    const m = cache.getCached(`macd-${cacheKey}`, () => macd(close))
    series.push(
      { type: 'line', name: 'DIF', data: m.dif, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'DEA', data: m.dea, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'bar', name: 'MACD', data: m.hist.map((v: number | null) => {
        if (v === null) return null
        return { value: v, itemStyle: { color: v >= 0 ? '#ef5350' : '#66bb6a' } }
      }), gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1 },
    )
  } else if (bottomMode === 'kdj') {
    const kd = cache.getCached(`kdj-${cacheKey}`, () => kdj(close, high, low))
    series.push(
      { type: 'line', name: 'K', data: kd.k, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'D', data: kd.d, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'line', name: 'J', data: kd.j, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1, symbol: 'none', lineStyle: { width: 1, color: '#ab47bc' } },
    )
  } else if (bottomMode === 'rsi') {
    const r = cache.getCached(`rsi-${cacheKey}-14`, () => rsi(close, 14))
    series.push({
      type: 'line', name: 'RSI', data: r, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#ec407a' },
      markLine: { silent: true, symbol: 'none', data: [
        { yAxis: 70, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
        { yAxis: 30, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
      ]},
    })
  } else if (bottomMode === 'wr') {
    const w = cache.getCached(`wr-${cacheKey}-14`, () => wr(close, high, low, 14))
    series.push({
      type: 'line', name: 'WR', data: w, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#42a5f5' },
      markLine: { silent: true, symbol: 'none', data: [
        { yAxis: -20, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
        { yAxis: -80, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
      ]},
    })
  } else if (bottomMode === 'cci') {
    const cciData = cache.getCached(`cci-${cacheKey}-20`, () => cci(high, low, close, 20))
    series.push({
      type: 'line', name: 'CCI', data: cciData, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#ce93d8' },
      markLine: { silent: true, symbol: 'none', data: [
        { yAxis: 100, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
        { yAxis: -100, label: { show: false }, lineStyle: { type: 'dashed', color: 'rgba(255,255,255,0.2)' } },
      ]},
    })
  } else if (bottomMode === 'obv') {
    const vol = data.map(d => d.volume)
    const obvData = cache.getCached(`obv-${cacheKey}`, () => obv(close, vol))
    series.push({
      type: 'line', name: 'OBV', data: obvData, gridIndex: 1, xAxisIndex: 1, yAxisIndex: 1,
      symbol: 'none', lineStyle: { width: 1, color: '#66bb6a' },
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: '#66bb6a30' },
        { offset: 1, color: 'rgba(0,0,0,0)' },
      ]}},
    })
  }

  if (eventMarkers && eventMarkers.length > 0) {
      series[0].markLine = {
        silent: true,
        symbol: 'none',
        data: eventMarkers.map(m => ({
          xAxis: m.dataIndex,
          label: { formatter: m.label, color: m.color, fontSize: 10, fontWeight: 'bold' as const, position: 'start' as const },
          lineStyle: { color: m.color, type: 'dashed' as const, width: 1, opacity: 0.6 },
        })),
      }
  }

  if (tradeSignals && tradeSignals.length > 0) {
    series[0].markPoint = {
      silent: true,
      symbolSize: 28,
      data: tradeSignals.map(t => ({
        name: t.label || (t.direction === 'buy' ? 'B' : 'S'),
        coord: [t.dataIndex, t.price],
        itemStyle: { color: t.direction === 'buy' ? '#f85149' : '#3fb950' },
        symbol: t.direction === 'buy' ? 'pin' : 'pin',
        symbolRotate: t.direction === 'buy' ? 180 : 0,
        label: { formatter: t.direction === 'buy' ? 'B' : 'S', color: '#fff', fontSize: 10, fontWeight: 'bold' as const },
      })),
    }
  }

  if (indexOverlay) {
    series.push({
      type: 'line',
      name: indexOverlay.name || indexOverlay.symbol,
      data: indexOverlay.data,
      smooth: true,
      lineStyle: { width: 1, color: '#a78bfa' },
      showSymbol: false,
      xAxisIndex: 0,
      yAxisIndex: 0,
      z: 5,
    })
  }

  // Default zoom window: show last ~250 candles (~1 year for daily)
  const totalPoints = data.length
  const windowSize = Math.min(totalPoints, 250)
  const startPct = totalPoints > windowSize ? ((totalPoints - windowSize) / totalPoints * 100) : 0

  const volUnit = '万'
  let bottomYAxis: any = { type: 'value', gridIndex: 1, axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false } }
  if (bottomMode === 'volume') {
    bottomYAxis = { ...bottomYAxis, axisLabel: { ...bottomYAxis.axisLabel, formatter: (v: number) => v >= 1 ? v.toFixed(1) + volUnit : String(v) } }
  } else if (bottomMode === 'kdj' || bottomMode === 'rsi') {
    bottomYAxis = { ...bottomYAxis, min: 0, max: 100 }
  } else if (bottomMode === 'wr') {
    bottomYAxis = { ...bottomYAxis, min: -100, max: 0 }
  }

  return {
    backgroundColor: 'transparent',
    grid: [
      { left: 60, right: 10, top: 24, height: '50%' },
      { left: 60, right: 10, top: '68%', height: '26%' },
    ],
    xAxis: [
      { type: 'category', data: dates, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } } },
      { type: 'category', data: dates, gridIndex: 1, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } } },
    ],
    yAxis: [
      { type: 'value', gridIndex: 0, scale: true, axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } } },
      { ...bottomYAxis, scale: true },
    ],
    series,
    tooltip: {
      trigger: 'axis' as const,
      formatter: (ps: any[]) => {
        if (!ps?.length) return ''
        const lines: string[] = [`<div style="font-size:12px">${ps[0].name || ''}</div>`]
        for (const p of ps) {
          const d = p.data
          if (p.seriesType === 'candlestick' && Array.isArray(d) && d.length >= 4) {
            lines.push(`<div style="margin-top:4px">${t('kline.open')}: ${d[0].toFixed(2)}</div>`)
            lines.push(`<div>${t('kline.close')}: ${d[1].toFixed(2)}</div>`)
            lines.push(`<div>${t('kline.low')}: ${d[2].toFixed(2)}</div>`)
            lines.push(`<div>${t('kline.high')}: ${d[3].toFixed(2)}</div>`)
          } else if (p.seriesType === 'bar') {
            lines.push(`<div>${p.seriesName}: ${p.value != null ? Number(p.value).toFixed(1) : '--'}</div>`)
          } else if (p.seriesType === 'line') {
            lines.push(`<div>${p.seriesName}: ${p.value != null ? Number(p.value).toFixed(2) : '--'}</div>`)
          }
        }
        return lines.join('')
      },
    },
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: startPct, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], start: startPct, end: 100, bottom: 0, height: 20 },
    ],
  } as ECBasicOption
}

export function buildMinuteOption(
  ticks: MinuteTick[],
  prevClose: number,
  bottomMode: string,
  theme: ChartThemeColors,
  cache: IndicatorCache,
  symbol: string,
): ECBasicOption {
  if (!ticks.length) return {} as ECBasicOption

  const t = i18n.global.t
  const volUnit = '万'
  const times = ticks.map(t => t.time)
  const prices = ticks.map(t => t.price)
  const volumes = ticks.map(t => t.volume / 10000)
  const rawVolumes = ticks.map(t => t.volume)
  const amounts = ticks.map(t => t.amount ?? 0)
  const upCol = marketUpColor(symbol)
  const downCol = marketDownColor(symbol)
  const isUp = prices.length > 0 && prices[prices.length - 1] >= prevClose
  const lineColor = isUp ? upCol : downCol
  const cacheKey = `${ticks.length}-${bottomMode}`

  const grid: any[] = []
  const xAxis: any[] = []
  const yAxis: any[] = []
  const series: any[] = []

  const priceBot = bottomMode === 'volume' ? '78%' : '55%'
  grid.push({ left: 60, right: 20, top: 24, height: bottomMode === 'volume' ? '60%' : '38%' })
  xAxis.push({ type: 'category', data: times, gridIndex: 0, axisLabel: { show: false }, axisLine: { lineStyle: { color: theme.splitColor } }, axisTick: { show: false } })
  yAxis.push({ type: 'value', gridIndex: 0, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { lineStyle: { color: theme.bgColor } },
    min: (val: { min: number; max: number }) => Math.floor(val.min * 0.995 * 100) / 100,
    max: (val: { min: number; max: number }) => Math.ceil(val.max * 1.005 * 100) / 100,
  })
  series.push(
    { type: 'line', name: t('common.price'), data: prices, xAxisIndex: 0, yAxisIndex: 0, smooth: false, symbol: 'none', lineStyle: { color: lineColor, width: 1.5 },
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [
        { offset: 0, color: isUp ? upCol + '40' : downCol + '40' },
        { offset: 1, color: 'rgba(0,0,0,0)' },
      ]}},
      markLine: prevClose > 0 ? { silent: true, symbol: 'none', lineStyle: { color: theme.axisColor, type: 'dashed', width: 1 }, data: [{ yAxis: prevClose, label: { formatter: `${t('kline.prev_close')} ${prevClose.toFixed(2)}`, color: theme.axisColor, fontSize: 10, position: 'start' as const } }] } : undefined,
    },
    { type: 'line', name: t('kline.avg_price'), data: ticks.map(tick => tick.avg_price), xAxisIndex: 0, yAxisIndex: 0, smooth: true, symbol: 'none', lineStyle: { color: '#f59e0b', width: 1, type: 'dashed' } },
  )

  const botGridIdx = 1
  const botAxisIdx = 1
  grid.push({ left: 60, right: 20, top: priceBot, height: bottomMode === 'volume' ? '15%' : '35%' })
  xAxis.push({ type: 'category', data: times, gridIndex: botGridIdx, axisLabel: { color: theme.axisColor, fontSize: 10, interval: 30 }, axisLine: { lineStyle: { color: theme.splitColor } } })

  if (bottomMode === 'volume') {
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10, formatter: (v: number) => v >= 1 ? v.toFixed(1) + volUnit : String(v) }, splitLine: { show: false } })
    series.push({ type: 'bar', name: t('kline.volume'), data: volumes, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, itemStyle: { color: theme.splitColor }, barWidth: 1 })
  } else if (bottomMode === 'macd') {
    const m = cache.getCached(`minute-macd-${cacheKey}`, () => macd(prices))
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true })
    series.push(
      { type: 'line', name: 'DIF', data: m.dif, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'DEA', data: m.dea, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'bar', name: 'MACD', data: m.hist.map((v: number | null) => v === null ? null : { value: v, itemStyle: { color: v >= 0 ? '#ef5350' : '#66bb6a' } }), xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx },
    )
  } else if (bottomMode === 'kdj') {
    const n = 9
    const minPrices: number[] = prices.map((_, i) => {
      const start = Math.max(0, i - n + 1)
      return Math.min(...prices.slice(start, i + 1))
    })
    const maxPrices: number[] = prices.map((_, i) => {
      const start = Math.max(0, i - n + 1)
      return Math.max(...prices.slice(start, i + 1))
    })
    const kd = cache.getCached(`minute-kdj-${cacheKey}`, () => kdj(prices, maxPrices, minPrices, n, 3, 3))
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true })
    series.push(
      { type: 'line', name: 'K', data: kd.k, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: theme.axisColor } },
      { type: 'line', name: 'D', data: kd.d, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ff9800' } },
      { type: 'line', name: 'J', data: kd.j, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ab47bc' } },
    )
  } else if (bottomMode === 'rsi') {
    const r = cache.getCached(`minute-rsi-${cacheKey}`, () => rsi(prices))
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true, min: 0, max: 100 })
    series.push(
      { type: 'line', name: 'RSI', data: r, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#ab47bc' } },
    )
  } else if (bottomMode === 'obv') {
    const o = cache.getCached(`minute-obv-${cacheKey}`, () => obv(prices, rawVolumes))
    yAxis.push({ type: 'value', gridIndex: botGridIdx, position: 'left', axisLabel: { color: theme.axisColor, fontSize: 10 }, splitLine: { show: false }, scale: true })
    series.push(
      { type: 'line', name: 'OBV', data: o, xAxisIndex: botAxisIdx, yAxisIndex: botAxisIdx, symbol: 'none', lineStyle: { width: 1, color: '#42a5f5' } },
    )
  }

  return {
    animation: false, animationDurationUpdate: 0, animationEasingUpdate: 'linear',
    backgroundColor: 'transparent', grid, xAxis, yAxis, series,
    tooltip: { trigger: 'axis' as const, formatter: (ps: any[]) => {
      if (!ps?.length) return ''
      const idx = ps[0].dataIndex
      const tick = ticks[idx]
      if (!tick) return ''
      const chg = tick.price - prevClose
      const chgPct = prevClose > 0 ? (chg / prevClose * 100) : 0
      const chgStr = (chg >= 0 ? '+' : '') + chg.toFixed(2)
      const pctStr = (chgPct >= 0 ? '+' : '') + chgPct.toFixed(2) + '%'
      const chgColor = chg >= 0 ? '#ef5350' : '#66bb6a'
      return `<div style="font-size:12px;font-weight:600;margin-bottom:4px">${tick.time}</div>
<div>${t('common.price')}: <b>${tick.price.toFixed(2)}</b> <span style="color:${chgColor}">${chgStr} ${pctStr}</span></div>
<div>${t('kline.avg_price')}: ${tick.avg_price.toFixed(2)}</div>
<div>${t('kline.volume')}: ${tick.volume.toFixed(0)}</div>
${tick.amount ? `<div>${t('common.amount')}: ${(tick.amount / 10000).toFixed(2)}万</div>` : ''}`
    }},
  } as ECBasicOption
}



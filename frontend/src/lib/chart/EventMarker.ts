import type { KlineDataItem } from '@/lib/buildChartOption'

export interface EventMarker {
  dataIndex: number
  label: string
  color: string
}

export function detectLimitUpDown(data: KlineDataItem[]): EventMarker[] {
  const markers: EventMarker[] = []
  if (data.length < 2) return markers
  for (let i = 1; i < data.length; i++) {
    const prevClose = data[i - 1].close
    const bar = data[i]
    if (!prevClose || !bar) continue

    const upLimit = prevClose * 1.098
    const downLimit = prevClose * 0.902

    if (bar.close >= upLimit) {
      markers.push({ dataIndex: i, label: '涨停', color: '#f87171' })
    } else if (bar.close <= downLimit) {
      markers.push({ dataIndex: i, label: '跌停', color: '#4ade80' })
    }
  }
  return markers
}

export function exDividendMarkers(exDates: number[], timestamps: number[]): EventMarker[] {
  if (!exDates.length || !timestamps.length) return []
  const markers: EventMarker[] = []
  for (const exDate of exDates) {
    const exDateMs = exDate * 1000
    let bestIdx = -1
    let bestDiff = Infinity
    for (let i = 0; i < timestamps.length; i++) {
      const diff = Math.abs(timestamps[i] * 1000 - exDateMs)
      if (diff < bestDiff) {
        bestDiff = diff
        bestIdx = i
      }
    }
    if (bestIdx >= 0 && bestDiff < 86400000 * 2) {
      markers.push({ dataIndex: bestIdx, label: '除权', color: '#818cf8' })
    }
  }
  return markers
}

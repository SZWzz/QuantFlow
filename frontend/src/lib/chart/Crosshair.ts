export interface CrosshairData {
  time: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  change: number
  changePercent: number
}

export class Crosshair {
  private echarts: any = null
  private canvas: HTMLCanvasElement | null = null
  private ctx: CanvasRenderingContext2D | null = null
  private visible = false
  private mouseX = 0
  private mouseY = 0
  private data: CrosshairData | null = null
  private onHover: ((d: CrosshairData | null) => void) | null = null

  mount(echarts: any, canvas: HTMLCanvasElement, onHover?: (d: CrosshairData | null) => void) {
    this.echarts = echarts
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.onHover = onHover ?? null
  }

  destroy() {
    this.echarts = null
    this.canvas = null
    this.ctx = null
  }

  show(x: number, y: number) {
    this.visible = true
    this.mouseX = x
    this.mouseY = y
    this.updateData()
    this.render()
  }

  hide() {
    this.visible = false
    this.data = null
    this.onHover?.(null)
    this.render()
  }

  private updateData() {
    if (!this.echarts) return
    const coord = this.echarts.convertFromPixel({ seriesIndex: 0 }, [this.mouseX, this.mouseY])
    if (!coord || !Array.isArray(coord)) { this.data = null; return }
    const dataIndex = Math.round(coord[0])

    const model = this.echarts.getModel()
    const series = model.getSeriesByIndex(0)
    if (!series) { this.data = null; return }
    const rawData = series.getRawData()
    if (!rawData || dataIndex < 0 || dataIndex >= rawData.length) { this.data = null; return }
    const item = rawData.getValues(dataIndex) as any

    this.data = {
      time: String(item[0] || ''),
      open: Number(item[1]),
      high: Number(item[2] !== undefined ? item[2] : item[3]),
      low: Number(item[3] !== undefined ? item[3] : item[2]),
      close: Number(item[4] !== undefined ? item[4] : item[2]),
      volume: Number(item[5] || item[6] || 0),
      change: 0,
      changePercent: 0,
    }

    if (dataIndex > 0) {
      const prev = rawData.getValues(dataIndex - 1) as any
      const prevClose = Number(prev[4] !== undefined ? prev[4] : prev[2])
      this.data.change = this.data.close - prevClose
      this.data.changePercent = prevClose !== 0 ? (this.data.change / prevClose) * 100 : 0
    }
    this.onHover?.(this.data)
  }

  render() {
    const ctx = this.ctx
    const canvas = this.canvas
    if (!ctx || !canvas) return
    canvas.width = canvas.clientWidth
    canvas.height = canvas.clientHeight
    ctx.clearRect(0, 0, canvas.width, canvas.height)

    if (!this.visible) return

    const w = canvas.width
    const h = canvas.height

    ctx.strokeStyle = 'rgba(128, 128, 128, 0.5)'
    ctx.lineWidth = 1
    ctx.setLineDash([4, 4])
    ctx.beginPath(); ctx.moveTo(this.mouseX, 0); ctx.lineTo(this.mouseX, h); ctx.stroke()
    ctx.beginPath(); ctx.moveTo(0, this.mouseY); ctx.lineTo(w, this.mouseY); ctx.stroke()
    ctx.setLineDash([])

    if (this.data) {
      const priceText = this.data.close.toFixed(2)
      ctx.fillStyle = 'rgba(0,0,0,0.7)'
      ctx.fillRect(w - 80, this.mouseY - 8, 80, 16)
      ctx.fillStyle = '#fff'
      ctx.font = '11px monospace'
      ctx.fillText(priceText, w - 76, this.mouseY + 4)

      const timeText = this.data.time
      ctx.fillStyle = 'rgba(0,0,0,0.7)'
      const tw = ctx.measureText(timeText).width
      ctx.fillRect(this.mouseX - tw / 2 - 4, h - 20, tw + 8, 20)
      ctx.fillStyle = '#fff'
      ctx.fillText(timeText, this.mouseX - tw / 2, h - 6)
    }

    if (this.data) {
      const lines = [
        `T: ${this.data.time}`,
        `O: ${this.data.open.toFixed(2)}`,
        `H: ${this.data.high.toFixed(2)}`,
        `L: ${this.data.low.toFixed(2)}`,
        `C: ${this.data.close.toFixed(2)}`,
        `Chg: ${this.data.change >= 0 ? '+' : ''}${this.data.change.toFixed(2)}`,
        `${this.data.changePercent >= 0 ? '+' : ''}${this.data.changePercent.toFixed(2)}%`,
        `Vol: ${(this.data.volume / 10000).toFixed(0)}万`,
      ]
      const lineH = 16
      const boxW = 130
      const boxH = lines.length * lineH + 8
      const boxX = Math.min(this.mouseX + 16, w - boxW - 8)
      const boxY = Math.max(4, Math.min(this.mouseY - boxH / 2, h - boxH - 4))

      ctx.fillStyle = 'rgba(0, 0, 0, 0.8)'
      ctx.fillRect(boxX, boxY, boxW, boxH)
      ctx.fillStyle = '#fff'
      ctx.font = '11px monospace'
      lines.forEach((line, i) => {
        ctx.fillText(line, boxX + 6, boxY + 12 + i * lineH)
      })
    }
  }
}

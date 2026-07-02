import type { DrawingShape, DrawingMode, DataPoint } from './types'

export class DrawingController {
  private echarts: any = null
  private canvas: HTMLCanvasElement | null = null
  private ctx: CanvasRenderingContext2D | null = null
  mode: DrawingMode = 'cursor'
  private color = '#58a6ff'
  private drawings: DrawingShape[] = []
  private activeSymbol = ''
  private isDrawing = false
  private startPoint: DataPoint | null = null
  private currentPixel: { x: number; y: number } | null = null
  private nextId = 1

  private storageKey(): string { return `drawings-v2/${this.activeSymbol}` }

  mount(echarts: any, canvas: HTMLCanvasElement, symbol: string) {
    this.echarts = echarts
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.activeSymbol = symbol
    this.loadDrawings()
    this.render()
  }

  destroy() {
    this.echarts = null
    this.canvas = null
    this.ctx = null
    this.drawings = []
  }

  updateSymbol(symbol: string) {
    this.saveDrawings()
    this.activeSymbol = symbol
    this.loadDrawings()
    this.render()
  }

  setMode(mode: DrawingMode) {
    this.mode = mode
    this.isDrawing = false
    this.startPoint = null
    this.currentPixel = null
  }

  setColor(color: string) { this.color = color }

  loadDrawings() {
    try {
      const raw = localStorage.getItem(this.storageKey()) || '[]'
      this.drawings = JSON.parse(raw)
      this.nextId = this.drawings.reduce((max, d) => Math.max(max, d.id), 0) + 1
    } catch {
      this.drawings = []
    }
  }

  saveDrawings() {
    localStorage.setItem(this.storageKey(), JSON.stringify(this.drawings))
  }

  clearAll() {
    this.drawings = []
    this.saveDrawings()
    this.render()
  }

  render() {
    const ctx = this.ctx
    const canvas = this.canvas
    if (!ctx || !canvas || !this.echarts) return
    canvas.width = canvas.clientWidth
    canvas.height = canvas.clientHeight

    ctx.clearRect(0, 0, canvas.width, canvas.height)

    for (const d of this.drawings) {
      this.drawShape(ctx, d)
    }

    if (this.isDrawing && this.startPoint && this.currentPixel) {
      const startPixel = this.echarts.convertToPixel({ seriesIndex: 0 }, [this.startPoint.dataIndex, this.startPoint.price])
      if (!startPixel) return
      this.drawShapePixels(ctx, {
        id: -1,
        type: this.mode as DrawingShape['type'],
        points: [{ x: startPixel[0], y: startPixel[1] }, { x: this.currentPixel.x, y: this.currentPixel.y }],
        color: this.color,
      } as any)
    }
  }

  private toDataPoint(pixelX: number, pixelY: number): DataPoint | null {
    if (!this.echarts) return null
    const coord = this.echarts.convertFromPixel({ seriesIndex: 0 }, [pixelX, pixelY])
    if (!coord || !Array.isArray(coord) || coord.length < 2) return null
    return { dataIndex: Math.round(coord[0]), price: coord[1] }
  }

  private drawShape(ctx: CanvasRenderingContext2D, d: DrawingShape) {
    if (!this.echarts) return
    const pixels = d.points.map(p => this.echarts.convertToPixel({ seriesIndex: 0 }, [p.dataIndex, p.price]))
    if (pixels.some(p => !p)) return
    this.drawShapePixels(ctx, {
      ...d,
      points: pixels.filter(Boolean).map(p => ({ x: p![0], y: p![1] })),
    } as any)
  }

  private drawShapePixels(ctx: CanvasRenderingContext2D, d: any) {
    ctx.strokeStyle = d.color
    ctx.fillStyle = d.color
    ctx.lineWidth = 2
    ctx.font = '13px monospace'
    ctx.setLineDash([])

    const [a, b] = d.points
    if (!b && d.type !== 'text') return

    switch (d.type) {
      case 'trendline':
        ctx.beginPath(); ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y); ctx.stroke()
        break
      case 'horizontal':
        ctx.beginPath(); ctx.moveTo(0, b.y); ctx.lineTo(ctx.canvas.width, b.y); ctx.stroke()
        ctx.fillText(b.y.toFixed(2), 6, b.y - 4)
        break
      case 'fibonacci': {
        const dx = b.x - a.x
        const dy = b.y - a.y
        const ratios = [0, 0.236, 0.382, 0.5, 0.618, 0.786, 1]
        const colors = ['#f87171', '#fb923c', '#fbbf24', '#4ade80', '#22d3ee', '#818cf8', '#e879f9']
        for (let i = 0; i < ratios.length; i++) {
          const y = a.y + ratios[i] * dy
          ctx.strokeStyle = colors[i]
          ctx.lineWidth = 1
          ctx.setLineDash([4, 4])
          ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(ctx.canvas.width, y); ctx.stroke()
          ctx.fillText((ratios[i] * 100).toFixed(1) + '%', 6, y - 4)
        }
        ctx.setLineDash([])
        break
      }
      case 'text': {
        const p = d.points[0]
        if (p) ctx.fillText(d.text || '', p.x, p.y)
        break
      }
    }
  }

  onMouseDown(e: MouseEvent) {
    if (this.mode === 'cursor' || !this.canvas) return
    this.isDrawing = true
    const rect = this.canvas.getBoundingClientRect()
    const px = e.clientX - rect.left
    const py = e.clientY - rect.top
    this.currentPixel = { x: px, y: py }
    const dp = this.toDataPoint(px, py)
    if (dp) this.startPoint = dp
  }

  onMouseMove(e: MouseEvent) {
    if (!this.isDrawing || !this.canvas) return
    const rect = this.canvas.getBoundingClientRect()
    this.currentPixel = { x: e.clientX - rect.left, y: e.clientY - rect.top }
    this.render()
  }

  onMouseUp(_e: MouseEvent) {
    if (!this.isDrawing || !this.startPoint || !this.canvas) return
    this.isDrawing = false

    if (this.mode === 'text') {
      const text = prompt('输入文字:')
      if (!text) { this.render(); return }
      this.drawings.push({
        id: this.nextId++,
        type: 'text',
        points: [this.startPoint],
        color: this.color,
        text,
      })
    } else {
      const endDp = this.toDataPoint(this.currentPixel!.x, this.currentPixel!.y)
      if (!endDp) { this.render(); return }
      this.drawings.push({
        id: this.nextId++,
        type: this.mode as DrawingShape['type'],
        points: [this.startPoint, endDp],
        color: this.color,
      })
    }

    this.startPoint = null
    this.currentPixel = null
    this.saveDrawings()
    this.render()
  }
}

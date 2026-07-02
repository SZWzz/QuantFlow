export interface DataPoint {
  dataIndex: number
  price: number
}

export type DrawingType = 'trendline' | 'horizontal' | 'fibonacci' | 'text'

export interface DrawingShape {
  id: number
  type: DrawingType
  points: DataPoint[]
  color: string
  text?: string
}

export type DrawingMode = 'cursor' | DrawingType

export interface Column {
  key: string
  label: string
  width?: number
  flex?: number
  align?: 'left' | 'right' | 'center'
  format?: 'price' | 'percent' | 'volume' | 'number'
  colorize?: boolean
  formatter?: (val: any) => string
}

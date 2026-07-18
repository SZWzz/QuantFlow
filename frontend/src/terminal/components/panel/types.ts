export interface Column {
  key: string
  label: string
  width?: number
  flex?: number
  align?: 'left' | 'right' | 'center'
  format?: 'price' | 'percent' | 'volume' | 'number'
  /** 等宽数字列；缺省时 format 为 price/percent/volume/number 自动为 true */
  mono?: boolean
  colorize?: boolean
  formatter?: (val: any) => string
}

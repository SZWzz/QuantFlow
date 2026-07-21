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
  /** 表头可点击排序；排序状态由父组件经 sortKey/sortDir 受控传入 */
  sortable?: boolean
  /** 单元格 title 属性（原生 tooltip），按行计算；缺省不输出 title */
  title?: (row: any) => string
  /** 单元格附加 class，按行计算（如阈值高亮）；返回空串/undefined 不加 */
  cellClass?: (row: any) => string
}

/**
 * useChartTheme — reads CSS custom properties and returns concrete color values
 * for ECharts configuration. ECharts Canvas renderer does not parse CSS variables,
 * so we must resolve them at runtime via getComputedStyle.
 */
export interface ChartThemeColors {
  /** Primary text color (axis labels, legend) */
  textColor: string
  /** Tertiary/secondary text (axis names, tooltip text) */
  axisColor: string
  /** Grid/split line color */
  splitColor: string
  /** Chart background color */
  bgColor: string
  /** Accent/crosshair color */
  crossColor: string
  /** Tooltip background */
  tooltipBg: string
  /** Tooltip text */
  tooltipText: string
}

export function useChartTheme(): ChartThemeColors {
  try {
    const styles = getComputedStyle(document.documentElement)
    return {
      textColor:
        styles.getPropertyValue('--color-text-primary').trim() || '#333333',
      axisColor:
        styles.getPropertyValue('--color-text-tertiary').trim() || '#888780',
      splitColor:
        styles.getPropertyValue('--color-border-subtle').trim() || '#e8e8e8',
      bgColor:
        styles.getPropertyValue('--color-bg-elevated').trim() || '#ffffff',
      crossColor:
        styles.getPropertyValue('--color-error').trim() || '#e24b4a',
      tooltipBg:
        styles.getPropertyValue('--color-bg-glass').trim() || '#ffffff',
      tooltipText:
        styles.getPropertyValue('--color-text-primary').trim() || '#333333',
    }
  } catch {
    return {
      textColor: '#333333',
      axisColor: '#888780',
      splitColor: '#e8e8e8',
      bgColor: '#ffffff',
      crossColor: '#e24b4a',
      tooltipBg: '#ffffff',
      tooltipText: '#333333',
    }
  }
}

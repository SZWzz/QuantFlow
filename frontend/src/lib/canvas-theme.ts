/**
 * Canvas theme utility — reads CSS custom properties and returns
 * concrete color values for Canvas rendering (which does not parse CSS variables).
 */
export interface ColorScheme {
  bg: string
  grid: string
  text: string
  cursor: string
  select: string
  crosshair: string
}

export function useCanvasTheme(): ColorScheme {
  try {
    const styles = getComputedStyle(document.documentElement)
    return {
      bg: styles.getPropertyValue('--color-bg-elevated').trim() || '#ffffff',
      grid: styles.getPropertyValue('--color-border-subtle').trim() || '#e8e8e8',
      text: styles.getPropertyValue('--color-text-primary').trim() || '#333333',
      cursor: styles.getPropertyValue('--color-text-secondary').trim() || '#888888',
      select: styles.getPropertyValue('--color-primary').trim() || '#3b82f6',
      crosshair: styles.getPropertyValue('--color-danger').trim() || '#e24b4a',
    }
  } catch {
    return {
      bg: '#ffffff',
      grid: '#e8e8e8',
      text: '#333333',
      cursor: '#888888',
      select: '#3b82f6',
      crosshair: '#e24b4a',
    }
  }
}

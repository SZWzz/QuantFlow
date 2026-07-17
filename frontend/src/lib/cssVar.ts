/**
 * Read a CSS custom property from document.body (where theme tokens live).
 *
 * For JS-driven colors that cannot use var() directly — e.g. SVG presentation
 * attributes or component props serialized into attributes (vue-flow
 * Background pattern-color, MiniMap mask-color).
 *
 * Read once at setup: correct on page load for the active theme, but not
 * live-reactive to theme switching (a reload applies the new theme).
 * Colors that go through inline `style` (edge styles, SVG style bindings)
 * should prefer literal 'var(--token)' strings, which switch live.
 */
export function cssVar(name: string, fallback: string): string {
  if (typeof document === 'undefined' || !document.body) return fallback
  const v = getComputedStyle(document.body).getPropertyValue(name).trim()
  return v || fallback
}

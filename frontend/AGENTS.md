# Frontend 开发约定

## 面板开发

- 新面板必须用 `@/terminal/components/panel` 的共享组件搭建：`PanelHeader` / `PanelTable` / `PanelTabs` / `StatItem` / `EmptyState` / `ErrorState` / `LoadingState`

## 样式规则

- scoped style 只允许 CSS 变量：禁 hex 颜色、禁 px font-size；padding/margin 用 `--space-*`；border-radius 用 `--radius-*`
- 面板组件禁止自绘外框 / box-shadow / border-radius 外壳（外壳由 DockTab 统一提供）；阴影只给浮层（`var(--shadow-md)`）
- 图表颜色一律走 `useChartTheme()`；涨跌表达 = 颜色 + `+/-` 号（色盲友好）

## 提交前检查

- `npm run lint:styles`（当前 warn 级，Phase 2 结束后转 error）
- `npm run typecheck`
- `npm run test`

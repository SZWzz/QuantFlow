# Phase 7: Theme System + i18n + Settings Panel

## Motivation

Phase 1-6 完成了核心功能，但终端缺少主题定制、多语言支持和集中设置管理。Phase 7 将这些打磨能力加入。

## Design

### 模块一：CSS Variables 主题系统

**架构**：CSS Variables + Pinia themeStore + `<html>` class 切换

**CSS 变量**（`assets/themes.css`）：

| 变量 | 暗色 | 亮色 | 用途 |
|------|------|------|------|
| `--bg` | #1a1a2e | #f0f2f5 | 面板背景 |
| `--card` | #16213e | #ffffff | 卡片背景 |
| `--input` | #0f2137 | #f9fafb | 输入框背景 |
| `--text` | #e0e0e0 | #111827 | 主文字 |
| `--muted` | #5a6380 | #6b7280 | 次要文字 |
| `--up` | #3fb950 | #059669 | 涨/盈 |
| `--down` | #f85149 | #dc2626 | 跌/亏 |
| `--accent` | #58a6ff | #2563eb | 强调色 |
| `--border` | #1a3a5c | #e5e7eb | 边框 |
| `--hover` | #16213e | #f3f4f6 | 悬停 |

**密度变量**：`--spacing`(8/4/12px)、`--font-kpi`(16/14/18px)、`--row-height`(36/28/44px)、`--padding-panel`(10/6/16px)

**切换**：`themeStore` → `<html class="theme-dark density-default">` → CSS 自动生效

### 模块二：i18n (vue-i18n)

- 中/英文双语言，~150 keys each
- 覆盖：面板标题、按钮、状态、通知级别、设置分区
- `$t('portfolio.total_value')` 方式使用
- Settings 面板切换语言，localStorage 持久化

### 模块三：Settings 面板

- 左侧 Tab 导航（9 分区）+ 右侧表单
- 分区：外观、语言、通知、数据、交易、显示、快捷键、存储、关于
- `settingsStore` (Pinia) 管理，localStorage 持久化

### 面板改造

5 个核心面板硬编码颜色 → CSS 变量引用（PortfolioSummary、PositionDetail、RiskDashboard、TradeHistory、OrderEntryPanel）

### 文件清单

```
frontend/src/
├── lib/theme.ts + lib/i18n/{index,zh,en}.ts
├── assets/themes.css
├── stores/settings.ts
├── terminal/panels/SettingsPanel.vue (+ registry 修改)
└── 5 面板颜色迁移
```

## Acceptance Criteria

- [ ] 亮/暗双主题切换正常，5 核心面板适配
- [ ] 密度三档生效
- [ ] 中/英文覆盖完整，即时切换
- [ ] Settings 9 分区完整
- [ ] `vue-tsc --noEmit` 无新增错误

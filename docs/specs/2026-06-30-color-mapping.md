# 颜色映射表 — QuantFlow 前端美化

> 生成时间：2026-06-30
> 用途：将 60+ 面板中的硬编码 Hex 颜色统一替换为 CSS 变量

---

## 映射规则（中国A股市场惯例：涨红跌绿）

| 硬编码颜色 | 映射为 | 语义 |
|-----------|--------|------|
| `#ef4444` `#dc2626` `#f87171` `#f85149` `#d32f2f` `#ff6b6b` `#ff4d4f` `#e03131` | `var(--color-up)` | 上涨/红色 |
| `#22c55e` `#16a34a` `#3fb950` `#4ade80` `#2f9e44` `#51cf66` `#0a3d1a` | `var(--color-down)` | 下跌/绿色 |
| `#3b82f6` `#60a5fa` `#58a6ff` `#1e90ff` `#8b5cf6` `#534ab7` `#4a90d9` `#2563eb` `#93c5fd` `#818cf8` | `var(--color-accent)` | 强调/蓝色/紫色 |
| `#f59e0b` `#fbbf24` `#f0883e` `#eab308` `#ff9800` `#f97316` `#f57c00` `#d97706` `#713f12` | `var(--color-accent)` | 警告/橙色/黄色 |
| `#e5e7eb` | `var(--color-border)` | 边框 |
| `#6b7280` `#9ca3af` | `var(--color-text-tertiary)` | 次要文字 |
| `#4b5563` | `var(--color-text-secondary)` | 次次要文字 |
| `#1f2937` `#111827` | `var(--color-text-primary)` | 主要文字 |
| `#1a1a2e` `#16162a` | `var(--color-bg-panel)` | 面板背景 |
| `#2a2a3e` `#1a1f2e` | `var(--color-bg-elevated)` |  elevated 背景 |
| `#0f172a` `#000000` | `var(--color-bg-base)` | 基础背景 |
| `#ffffff` `#FFFFFF` `#fff` `#FFF` | `var(--color-text-primary)` | 白色文字 |
| `#fee2e2` `#7f1d1d` `#991b1b` | `var(--color-up)` / `var(--color-up-bg)` | 红色系背景/文字 |
| `#d1fae5` `#065f46` `#14532d` `#388e3c` | `var(--color-down)` / `var(--color-down-bg)` | 绿色系背景/文字 |
| `#3d2e0a` `#78350f` | `var(--color-accent-soft)` | 深色 accent 背景 |
| `#f7931a` `#627eea` | `var(--color-accent)` | 加密货币/品牌色 |
| `#1976d2` | `var(--color-accent)` | Material 蓝色 |
| `#3a4a6c` | `var(--color-text-secondary)` | 蓝灰色文字 |
| `#f9fafb` | `var(--color-bg-elevated)` | 浅灰背景 |

---

## 替换范围

- **<style> 标签**：全部替换（71 个文件，275+26=301 处替换）
- **<script> 标签**：保留（ECharts 图表配置需要 hex 值，共 35 个文件含图表颜色）
- **已迁移面板**：10 个（AlphaMining/Indicator/MarketOverview/LimitUpDown/GovData/Watchlist/Backtest/Welcome/PredictionDashboard/StockScanner）

---

## 验证结果

- ✅ <style> 标签内零硬编码颜色
- ✅ 废弃 CSS 变量（--border-color/--term-bg-dim/--term-accent-dim/--text-muted）清零
- ✅ 类型错误修复（GovDataPanel 4 处 + DrawingPanel 1 处）
- ⚠️ <script> 中 35 个文件保留图表配置颜色（ECharts API 限制，合理）

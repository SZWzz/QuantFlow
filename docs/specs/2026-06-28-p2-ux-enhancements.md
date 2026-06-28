# P2 UX Enhancements (Skeleton + ErrorBoundary + Shortcuts + WelcomePanel)

## Motivation

用户体验是桌面终端的核心竞争壁垒。当前面板存在以下问题：(1) 所有面板 loading 态不统一（纯文字 "loading..."），缺少骨架屏；(2) 面板异常时整个 DockView 可能白屏，无错误边界；(3) 高频面板缺少快捷键直达；(4) WelcomePanel 静态列表，缺少动态仪表盘功能。

## Design

### 1. SkeletonPanel Component

通用骨架屏组件，所有面板的 loading 态统一使用：

```vue
<SkeletonPanel :rows="5" :type="'table' | 'card' | 'chart'" />
```

三种类型：
- `table`: 表头 + 5 行数据行闪烁
- `card`: 3 列卡片式占位
- `chart`: ECharts 区域占位

**Data Flow:** 纯展示组件，无数据依赖。

**Files:**
| File | Change |
|------|--------|
| `frontend/src/terminal/components/SkeletonPanel.vue` | **新增** |
| 所有新面板 + 现有面板 | 逐步替换 `loading` 态 |

### 2. ErrorBoundary Component

面板级错误边界，包裹 `DockTab.vue` 中的动态组件渲染：

```vue
<ErrorBoundary :panel-id="panelId">
  <component :is="panelComponent" ... />
</ErrorBoundary>
```

捕获 `errorCaptured` 钩子，显示备用 UI：
```
┌─ Panel Error ─────────────────┐
│ ⚠ 面板加载异常                  │
│ [重试] [关闭面板]              │
│ 错误信息: xxx                  │
└───────────────────────────────┘
```

**Files:**
| File | Change |
|------|--------|
| `frontend/src/terminal/components/ErrorBoundary.vue` | **新增** |
| `frontend/src/terminal/DockView/DockTab.vue` | 包裹 ErrorBoundary |

### 3. Keyboard Shortcuts

为高频面板添加快捷键（映射到 `CommandBar`）：

| 快捷键 | 面板 | 说明 |
|--------|------|------|
| `Ctrl+Shift+D` | dragon-tiger | 龙虎榜 |
| `Ctrl+Shift+L` | limit-up-down | 涨跌停 |
| `Ctrl+Shift+H` | hk-connect | 港股通 |
| `Ctrl+Shift+F` | funding-rate | 资金费率 |
| `Ctrl+Shift+Q` | sector-rotation | 板块轮动 |
| `Ctrl+Shift+E` | economic-calendar | 经济日历 |
| `Ctrl+Shift+W` | watchlist | 自选股（已有） |

实现：在 `TerminalMode.vue` 的 `onMounted` 时注册 `keydown` 事件监听，调用 `terminal.openPanel()`。

**Files:**
| File | Change |
|------|--------|
| `frontend/src/terminal/TerminalMode.vue` | 新增快捷键注册 |
| `frontend/src/stores/terminal.ts` | 可选：增加快捷键配置 |

### 4. WelcomePanel Dynamic Upgrade

WelcomePanel 从静态面板列表 → 动态仪表盘：

```
┌─ 欢迎回来 ───────────────────────┐
│ 上次打开: 龙虎榜, 港股通          │
│                                    │
│ ┌─ 市场快照 ───┐ ┌─ 系统状态 ──┐  │
│ │ 上证 3,128 +0.5│ │ 内存 45%   │  │
│ │ 恒指 18,234 -0.3│ │ 面板 12个  │  │
│ └────────────┘ └────────────┘  │
│                                    │
│ ┌─ 最近面板 ─────────────────┐   │
│ │ [龙虎榜] [港股通] [自选股]    │   │
│ └─────────────────────────────┘   │
│                                    │
│ [所有面板分类网格 - 现有保留]       │
└────────────────────────────────────┘
```

**Files:**
| File | Change |
|------|--------|
| `frontend/src/terminal/panels/WelcomePanel.vue` | 修改 |
| `frontend/src/stores/session.ts` | 可选：添加 recentPanels 历史 |

## Acceptance Criteria

- [ ] `<SkeletonPanel>` 组件三种类型均正常工作
- [ ] 龙虎榜面板使用 `<SkeletonPanel type="table">` 作为 loading 态
- [ ] ErrorBoundary 捕获异常并显示备用 UI | 重试按钮有效
- [ ] ErrorBoundary 不破坏其他面板渲染
- [ ] `Ctrl+Shift+D` 打开龙虎榜面板
- [ ] 快捷键仅在 terminal mode 下生效（不在 workflow mode）
- [ ] WelcomePanel 显示最近打开面板（从 session store）
- [ ] WelcomePanel 显示市场快照（上证指数、恒生指数）
- [ ] WelcomePanel 保留底部全部分类网格

## Risks / Trade-offs

- ErrorBoundary 使用 Vue 3 `onErrorCaptured`，仅捕获渲染/组件异常，不捕获异步错误
- 快捷键可能与其他应用快捷键冲突（如系统级 `Ctrl+Shift+D` Chrome 书签）
- WelcomePanel 市场快照需要定期刷新（60s），增加 dataStore 请求

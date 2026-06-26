# 前端质量全面整改 — 设计文档

> 目标：消灭所有静默吞错、缺失 loading/空状态、Store 绕过、localStorage 混乱等问题，建立统一数据获取标准。

## 现状问题

| 类别 | 数量 | 典型问题 |
|------|------|---------|
| Critical — 无 loading 指示器 | 3 | TickerTape、CryptoOverview、SystemMonitor — 数据获取时用户无反馈 |
| Critical — 空 catch 静默吞错 | 2 | SystemMonitor（每5s轮询静默失败）、Watchlist（localStorage 解析静默失败） |
| Critical — 显式 `/* silent */` catch | 5 | CryptoOverview、MarketDepth、Candlestick、PredictionMarket、TickerTape |
| Important — 14 个面板 catch 无日志 | 14 | 失败时只有 `data = []`，不知道发生了什么 |
| Important — 内容区缺 loading | 7 | 仅按钮 disabled，内容区无反馈 |
| Important — 缺空状态 | 5 | TickerTape 等 |
| Important — Store 无 error 状态 | 10 | 无法区分"加载中/空/失败" |
| Important — 面板绕过 Store | 6 | 直接调 Go，不用 Store 已有方法 |
| Important — BUG | 5 | panelGroups 泄漏、布局无持久化、API Key 明文、theme/session 重叠、closeTab 忽略 leafId |

---

## 分层设计

### Phase 1 — 止血：基础设施 + 修 Critical（本次重点）

**核心交付：** `useDataFetch` composable，统一 `{ data, loading, error }` 三元组。

```typescript
// src/lib/composables/useDataFetch.ts
function useDataFetch<T>(fetcher: () => Promise<T>) {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function execute() {
    loading.value = true
    error.value = null
    try {
      data.value = await fetcher()
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      console.error('[useDataFetch]', e)
    } finally {
      loading.value = false
    }
  }

  return { data, loading, error, execute }
}
```

**模板标准：**
```html
<div v-if="loading">加载中...</div>
<div v-else-if="error">错误: {{ error }}</div>
<div v-else-if="!data?.length">暂无数据</div>
<div v-else><!-- 正常内容 --></div>
```

**改动清单（6 个面板 + 1 个新文件）：**

| 文件 | 操作 | 改什么 |
|------|------|--------|
| `src/lib/composables/useDataFetch.ts` | **新建** | 统一数据获取 composable |
| `CandlestickPanel.vue` | 修改 | minute 部分 catch 加 console.error |
| `SystemMonitorPanel.vue` | 修改 | 接入 useDataFetch，加 loading/error 状态 |
| `TickerTapePanel.vue` | 修改 | 接入 useDataFetch，加 loading |
| `CryptoOverviewPanel.vue` | 修改 | 接入 useDataFetch，加 loading/error/空状态 |
| `MarketDepthPanel.vue` | 修改 | catch 加 console.error |
| `PredictionMarketPanel.vue` | 修改 | catch 加 console.error |
| `WatchlistPanel.vue` | 修改 | localStorage catch 加 console.error |

---

### Phase 2 — 规范化：全部面板标准化

**改动范围：**

| 类别 | 数量 | 改动 |
|------|------|------|
| 14 个静默重置面板 | 14 | 接入 useDataFetch 或加 console.error |
| 7 个内容区缺 loading | 7 | 模板加 `v-if="loading"` 骨架屏 |
| 5 个缺空状态 | 5 | 模板加 `t('common.no_data')` |
| 6 个绕过 Store | 6 | 改走 Store 方法 |

---

### Phase 3 — 重构：Store 层统一

**改动范围：**

| 类别 | 数量 | 改动 |
|------|------|------|
| 10 个 Store 加 error 状态 | 10 | 每个异步方法加 `error` ref |
| IPC 调用风格统一 | 10 | 统一为"可选链检查 + useDataFetch"模式 |
| symbolContext panelGroups 泄漏 | 1 | `onUnmounted` 中清理 `delete panelGroups[panelId]` |
| theme/session localStorage 重叠 | 2 | 合并到 session.ts 单一入口 |
| closeTab 忽略 leafId | 1 | 从指定 leaf 开始搜索 |
| 布局持久化 | 1 | terminal store 布局序列化到 localStorage |
| instanceId | 1 | `Date.now()` → `crypto.randomUUID()` |
| settings 类型安全 | 1 | 按 key 做具体类型约束 |

---

## 验证标准

1. 所有面板：loading 态 → 有视觉反馈
2. 所有面板：错误态 → 有 console.error + 用户可见提示
3. 所有面板：空数据态 → 显示"暂无数据"
4. 所有 Store：异步方法有 error 状态
5. 面板不再绕过 Store 直接调 Go
6. 刷新页面 → 布局恢复
7. npm run build 通过

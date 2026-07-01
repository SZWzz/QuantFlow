# 自选股面板：移除搜索框 + K 线图联动加入/取消

## Motivation

当前 WatchlistPanel 自带 SymbolSearch 搜索框，但添加自选股的更自然位置是 K 线图（用户在看图时决定加入自选）。同时搜索框在面板内占用空间，与已有顶栏 SymbolBar 功能重复。

## Design

### WatchlistPanel 改动
- 移除 `SymbolSearch` 组件和 `PanelToolbar` 搜索区
- 保留 refresh 按钮（移到 header controls）

### CandlestickPanel 改动
- symbol 旁新增「加入自选/取消自选」按钮
- 读取 `localStorage('quantflow-watchlist')` 判断当前股票是否在自选中
- 点击后：写 localStorage + 切换按钮状态
- 同时更新 WatchlistPanel（通过 localStorage 共享 + 同次会话中通过 DataStore 事件通知）

### 数据流
```
CandlestickPanel
  ├── getWatchlist() → localStorage.getItem('quantflow-watchlist')
  ├── isInWatchlist = computed
  ├── toggleWatchlist() → localStorage.setItem(...) + dispatch event
  │
  └── WatchlistPanel
      └── 监听事件或 onMounted 重新加载
```

### 修改文件

| 文件 | 改动 |
|------|------|
| `WatchlistPanel.vue` | 移除 SymbolSearch + PanelToolbar |
| `CandlestickPanel.vue` | 新增加入/取消自选按钮 |
| `data.ts` (store) | 新增 `watchlist` store 或事件 |

### Watchlist 事件（跨组件同步）

用 `dataStore` 的 pub/sub 机制广播 watchlist 变更：

```typescript
// 任意组件
dataStore.notify('watchlist:changed')

// WatchlistPanel
dataStore.subscribe('watchlist:changed', () => loadSymbols())
```

### Acceptance Criteria

- [ ] WatchlistPanel 不再显示搜索框
- [ ] K 线图显示加入自选/取消自选按钮
- [ ] 加入后 WatchlistPanel 同步更新
- [ ] 取消后 WatchlistPanel 同步移除该股

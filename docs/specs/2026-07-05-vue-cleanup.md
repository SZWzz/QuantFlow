# Vue Component Cleanup — Duplicate Watches, Computed Side Effects, Unused Dependencies

## Motivation

前端 Vue 组件审计发现三类代码质量问题：

1. **CandlestickPanel 重复 watch**（`frontend/src/terminal/panels/CandlestickPanel.vue`）— `symbol` 和 `interval` 各有两个 watch，导致同一变化触发两次加载
2. **SymbolSearch.vue computed 副作用**（`frontend/src/lib/SymbolSearch.vue`）— computed property 内执行 side effect（HTTP 请求/状态变更），违反 Vue 组合式 API 约定，应用 `watch`
3. **面板组件依赖未清理** — 部分组件 import 了未使用的库或组件

## Design

### 1. CandlestickPanel 重复 watch

**`CandlestickPanel.vue`** 当前结构：
```vue
watch(symbol, () => loadData())
watch(interval, () => loadData())
watch([symbol, interval], () => loadData(), { immediate: true })
```

第一个 watch 和第三个重复（第三个已监听 symbol 变化）。修复：

```vue
// 只有这一个 watch，同时监听两个源
watch([symbol, interval], () => loadData(), { immediate: true })
```

### 2. SymbolSearch.vue computed 副作用

当前模式：
```vue
const results = computed(() => {
  if (query.value.length < 2) return []
  searchApi(query.value) // side effect in computed — BAD
  return filteredResults.value
})
```

修复：将搜索触发移到 `watch`：
```vue
watch(query, (newQuery) => {
  if (newQuery.length >= 2) {
    searchApi(newQuery)
  } else {
    results.value = []
  }
})
```

### 3. 检查未使用的 import

对 91 个面板文件运行 `vue-tsc --noEmit` 检测 unused imports。逐一删除。

## Acceptance Criteria

- [ ] CandlestickPanel 只有一个 `watch([symbol, interval], ...)`
- [ ] SymbolSearch 的搜索触发在 `watch` 而非 `computed` 中
- [ ] `npx vue-tsc --noEmit` 通过（不因这些改动新增错误）
- [ ] `npx vitest run` 通过

## Risks / Trade-offs

- 修复 computed 副作用可能改变用户输入时的搜索体验。原代码依赖 computed 的"即时"特性，改为 watch 后如果前面有 `<template>` 渲染会稍有延迟。解决方案：用 `watch` + `debounce`。

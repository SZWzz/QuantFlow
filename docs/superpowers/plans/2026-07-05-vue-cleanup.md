# 实施计划：Vue Component Cleanup

参考：`docs/specs/2026-07-05-vue-cleanup.md`

## Task 1: CandlestickPanel 重复 watch

**`frontend/src/terminal/panels/CandlestickPanel.vue`**：

```vue
<script setup lang="ts">
// before
watch(symbol, () => loadData())
watch(interval, () => loadData())
watch([symbol, interval], () => loadData(), { immediate: true })

// after — 只有一个 watch
watch([symbol, interval], () => loadData(), { immediate: true })
</script>
```

---

## Task 2: SymbolSearch computed 副作用

**`frontend/src/lib/SymbolSearch.vue`**：

```vue
<script setup lang="ts">
// before
const results = computed(() => {
  if (query.value.length < 2) return []
  searchApi(query.value)  // side effect
  return filteredResults.value
})

// after
const results = ref<SearchResult[]>([])

let debounceTimer: ReturnType<typeof setTimeout>
watch(query, (newQuery) => {
  clearTimeout(debounceTimer)
  if (newQuery.length < 2) {
    results.value = []
    return
  }
  debounceTimer = setTimeout(() => {
    searchApi(newQuery).then(r => results.value = r)
  }, 300)
})

onUnmounted(() => clearTimeout(debounceTimer))
</script>
```

---

## Task 3: 检查未使用的 import

```bash
cd frontend && npx vue-tsc --noEmit 2>&1 | grep "is declared but its value is never read" | awk '{print $1}'
```

逐一删除。

---

## Task 4: 验证

```bash
cd frontend && npx vue-tsc --noEmit
cd frontend && npx vitest run
```

# 实施计划：K线指标预热数据不足修复

## 关联文档
- **Spec**: `docs/specs/2026-06-29-kline-indicator-warmup.md`

## 任务

### Task 1: 修改 lookbackDays 值

**文件**: `frontend/src/terminal/panels/CandlestickPanel.vue`

**目标行**: 第 114 行

**修改前**:
```javascript
const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 180 : 90
```

**修改后**:
```javascript
const lookbackDays = ['1m','5m','15m','30m','1h'].includes(iv) ? 5 : iv === '1w' ? 450 : 365
```

**验证**: `npx vue-tsc --noEmit` + `npx vitest run` 无报错

### Task 2: 更新 CHANGELOG

在 `CHANGELOG.md` 的 `[2026.6.29]` 版本下新增条目。先检查该版本是否已存在。

### Task 3: 构建验证

`wails3 build` 通过

### Task 4: 提交

```bash
git add -A
git commit -m "fix(frontend): increase daily K-line lookback from 90 to 365 days for indicator warmup"
```

## 滚动条

无。此更改极简（一行数值），无需滚动执行。

# Fix 4 Data-Damage Bugs — TuShare / NumPy / NaN / Migration Tx

> 审查报告 ref: 问题.md #4, #6, #7, #8

## Motivation

4 条数据损坏/正确性 bug：

1. **#4 TuShare API 解析错位**：TuShare 返回 `data.fields + data.items`（列名+嵌套数组），适配器读顶层 `result.Items`（`[]map[string]any`，API 从不填充）。CN 回退链第 3 位永远返回空 → 不报错、不回退
2. **#6 RL engine numpy 未 import**：`engine.py:159` 用 `np.float32` 但模块顶部没有 `import numpy as np` → RLTrain RPC 一调用就 `NameError`，被 `except Exception` 吞掉 → 流式返回空，用户以为训练没产出
3. **#7 因子 NaN→0 数据污染**：`factor/engine.py:58` 把 NaN 转成 `0.0` → 每个因子的预热窗口（如 20 行动量/波动率 NaN）变成 "零变动/零风险"，喂进 z-score/ML 特征 → 类 look-ahead 的数据污染
4. **#8 迁移无事务包裹**：`migrate.go:46-52` 每条迁移 SQL 和版本 insert 是两条独立 `db.Exec` → 进程被强杀时 schema 半应用、版本未记录 → 下次启动重跑非幂等语句

## Design

### #4: TuShare 解析修复

**根因**：TuShare HTTP API 返回格式是：
```json
{"code": 0, "data": {"fields": ["ts_code","trade_date","close",...], "items": [["000001.SZ","20240601",12.5,...], ...]}}
```
适配器的 `tushareResponse` struct 同时定义了顶层 `Items []map[string]any` 和 `Data tushareData`（含 `Fields []string` + `Items [][]any`）。但 `FetchQuote`/`FetchOHLCV` 读的是顶层 `result.Items`，TuShare API 从不填充这个顶层字段。

**修复**：在 `callAPI` 返回后，将 `result.Data.Fields` + `result.Data.Items` 转换为 `result.Items`（`[]map[string]any`），使现有 `FetchQuote`/`FetchOHLCV` 代码无需改动。

具体：在 `callAPI` 返回前，调用一个新函数 `zipFieldsAndItems`：
```go
result.Items = zipFieldsAndItems(result.Data.Fields, result.Data.Items)
```

### #6: numpy import 修复

**根因**：`engine.py` 未 import numpy，但 `RLTrain` 方法用 `np.float32`。

**修复**：在 `engine.py` 顶部添加 `import numpy as np`。

### #7: NaN→0 修复

**根因**：`factor/engine.py:58` 无条件将 NaN 转 0.0。

**修复**：将 `0.0` 改为 `float('nan')`，下游区分「无数据」和「值为零」。下游代码（feature_engineer 等）已经有 NaN 处理逻辑（`math.IsNaN` 检查后用 fillValue），能正确传播或填充 NaN。

### #8: 迁移事务修复

**根因**：`migrate.go:46-52` 两条 `db.Exec` 无事务包裹。

**修复**：用 `db.Begin()` + `tx.Exec()` + `tx.Commit()` 包裹每条迁移的 SQL + version insert。

## Acceptance Criteria

- [ ] **#4**: TuShare 返回的真实数据能被正确解析为 `[]map[string]any`
- [ ] **#4**: 新增 `zipFieldsAndItems` 函数有单元测试
- [ ] **#6**: `python -c "from src.ml.engine import MLService"` 不报 NameError
- [ ] **#7**: 因子 NaN 值保留为 NaN（而非 0.0）
- [ ] **#8**: 迁移 SQL + version insert 在同一事务中
- [ ] **#8**: 迁移中途失败 → schema 不变，version 不记录
- [ ] `go build ./...` + `go test ./...` 全绿
- [ ] Python 测试全绿
- [ ] CHANGELOG 更新

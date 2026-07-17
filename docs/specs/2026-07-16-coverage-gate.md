# 测试覆盖率门禁 (Coverage Gate)

## Motivation

现有有 967 个 Go 测试和 198 个前端测试，但缺乏覆盖率目标和门禁机制。在没有覆盖率指标的情况下，可能出现：

1. 新代码未经测试引入
2. 核心包（trading, backtest, market）覆盖率下降不被察觉
3. 重构时无安全网

已有 `docs/specs/2026-07-05-test-coverage.md` 为零覆盖包增加了测试，但缺少持续的门禁机制。

## Design

### 覆盖率目标

| 包 | 当前覆盖率 | 目标覆盖率 | 理由 |
|----|:---------:|:---------:|------|
| `internal/trading/` | ~70% | 80% | 资金安全，最高优先级 |
| `internal/backtest/` | ~75% | 80% | 回测正确性直接影响策略 |
| `internal/market/` | ~65% | 70% | 数据正确性 |
| `internal/workflow/` | ~80% | 80% | 已达标，维持 |
| `internal/storage/` | ~60% | 70% | 数据持久化 |
| `internal/ai/` | ~50% | 60% | LLM 输出非确定性，降低要求 |
| `internal/ws/` | 0% | 50% | 从零开始 |
| `internal/auth/` | 0% | 60% | 安全敏感 |
| `internal/notify/` | 0% | 50% | 通知依赖外部，集成测试 |
| `internal/schedule/` | 0% | 50% | 定时器逻辑 |
| 其余包 | N/A | 50% | 稳定后逐步提升 |

### 门禁机制

#### CI 覆盖率检查

```yaml
# .github/workflows/ci.yml 新增 step
- name: Coverage Gate
  run: |
    go test ./internal/trading/... -coverprofile=trading.out -coverpkg=./internal/trading/...
    go tool cover -func=trading.out | grep 'total:' | awk '{print $3}' | sed 's/%//' > coverage.txt
    COV=$(cat coverage.txt)
    if [ "$COV" -lt "80" ]; then
      echo "❌ internal/trading coverage ${COV}% < 80%"
      exit 1
    fi
    echo "✅ internal/trading coverage ${COV}% >= 80%"
```

每个核心包单独检查，避免总覆盖率被非核心包拉高掩盖问题。

#### 配置文件

`coverage-gate.json`：

```json
{
  "thresholds": {
    "internal/trading/": 80,
    "internal/backtest/": 80,
    "internal/market/": 70,
    "internal/workflow/": 80,
    "internal/storage/": 70,
    "internal/ai/": 60,
    "internal/ws/": 50,
    "internal/auth/": 60,
    "internal/notify/": 50,
    "internal/schedule/": 50,
    "frontend/": 40
  }
}
```

解释成 CI step 或 Makefile target。

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `coverage-gate.json` | 新建 | 覆盖率阈值配置 |
| `Makefile` | 修改 | 新增 `coverage-gate` target |
| `.github/workflows/ci.yml` | 修改 | 新增 Coverage Gate step |

### 降低豁免

- 新包（< 3 个月）有 6 个月宽限期达到目标
- 外部依赖包装（如 broker adapter wrappers）可豁免
- 豁免需在 `coverage-gate.json` 中添加 `"exempt": true` + 理由

### 报告可视化

可选的 HTML 覆盖率报告（Go 内建）：

```bash
make coverage-html
# 生成 coverage.html 可在浏览器中查看
```

### 前端覆盖率

```bash
# frontend/package.json 新增 script
"coverage": "vitest run --coverage"
```

配合 `@vitest/coverage-v8` 收集前端覆盖率。门禁线 40% 行覆盖率——前端测试偏集成测试，行覆盖率低是正常的。

## Acceptance Criteria

- [ ] `coverage-gate.json` 定义核心包的覆盖率阈值
- [ ] CI 中每个核心包独立检查覆盖率
- [ ] 覆盖率低于阈值时 CI 失败（红色 ❌）
- [ ] Makefile 有 `coverage-gate` target 可在本地运行
- [ ] 前端 vitest 配置覆盖率收集
- [ ] 前端 CI 检查覆盖率（40% 门禁线）
- [ ] 豁免机制文档化

## Risks / Trade-offs

- **风险**: 门禁太严导致开发速度下降。→ 只对核心包设门禁，非核心包有宽限期
- **风险**: 前端 40% 门禁太低无意义。→ 初期先建立基线，后续逐步提高到 50-60%
- **Trade-off**: 不使用 codecov/sonarcloud 第三方服务（隐私 + 成本），纯自建 CI 检查

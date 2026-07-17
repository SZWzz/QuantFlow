# 错误处理审计 (Error Handling Audit)

## Motivation

19 个 Go 包横跨多年开发周期，错误处理风格不统一：

1. 部分函数返回裸错误（`return err`），调用方丢失 context
2. 部分用 `fmt.Errorf("some context: %v", err)` 而非 `%w`，无法 `errors.Is/As`
3. 部分用 `log.Printf` 而非 `slog`（已有 spec `2026-07-07-fix-code-quality.md` 提及）
4. 部分错误静默吞没（`_ = err`）
5. 部分函数 panic 而非返回 error（特别是 JSON 解析、类型断言）

已有 `docs/specs/2026-07-07-fix-code-quality.md` 覆盖部分代码质量问题，但需要专门的错误处理审计。

## Design

### 审计范围

| 维度 | 检查项 | 严重程度 |
|------|--------|:-------:|
| Error wrapping | `%v` vs `%w`，能否 `errors.Is/As` | P1 |
| Error swallowing | `_ = err` / `err != nil { return }` 但不处理 | P0 |
| Panic | 非 recover 的 panic，尤其 JSON `.([]any)` | P0 |
| Logging | `log.Print` vs `slog` | P2 |
| Error return | 函数签名返回 `*T, error` 但永远返回 nil error | P1 |
| Context | 错误信息是否有 context（"file.go:123: failed to X: Y"） | P2 |
| Sentry | 是否需要/已有错误上报（见 crash-reporter spec） | P3 |

### 审计方法

通过 `codegraph` 或 `grep` 扫描：

```bash
# 1. 查找裸 `return err`（没有 wrap）
rg 'return err$' internal/ --include '*.go' | grep -v '_test.go'

# 2. 查找 `%v` 包裹 error（应为 %w）
rg 'fmt\.Errorf.*%v.*err' internal/ --include '*.go'

# 3. 查找 `_ = err`
rg '_ = err' internal/ --include '*.go'

# 4. 查找 panic
rg 'panic(' internal/ --include '*.go' | grep -v 'init()' | grep -v 'testing'

# 5. 查找 `log\.Print` / `log\.Printf` 而非 slog
rg 'log\.(Print|Printf|Println)' internal/ --include '*.go'
```

### 修复策略

#### P0: Panic + 静默错误

JSON 类型断言 → `json.RawMessage` + 显式 error 返回：

```go
// Bad
data := result.([]any)

// Good
raw, ok := result.([]any)
if !ok {
    return fmt.Errorf("unexpected response format: got %T, want []any", result)
}
```

所有 `panic` 改为 `slog.Error` + `return error`（除非是真正不该发生的程序 bug）。

#### P1: Error wrapping

```go
// Bad — 不能 errors.Is/As
return fmt.Errorf("fetch quote failed: %v", err)

// Good — 可链式 unwrap
return fmt.Errorf("fetch quote failed: %w", err)
```

#### P2: Error context

```go
// Bad — 不知道是哪个 symbol
if err != nil { return err }

// Good — 包含 symbol + adapter 信息
return fmt.Errorf("%s: fetch quote for %s: %w", adapter.Name(), symbol, err)
```

### 新增/修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `docs/superpowers/plans/2026-07-16-error-handling-audit.md` | 新建 | 实施计划（按包逐步修复） |

审计结果不体现在代码中，而是记录在 plan 文件，逐步修复。

### 优先顺序

1. `internal/trading/` — 资金安全相关
2. `internal/backtest/` — 回测正确性
3. `internal/market/` — 数据正确性
4. `internal/workflow/` — 影响所有用户
5. 其余包

每个包修复后，`go vet ./pkg/...` 确保无新增 error 问题。

### CI 检查

在 `.golangci.yml` 中启用 `errcheck` linter（已启用），确认其配置覆盖：

- 检查所有 exported function 的 error return 是否被处理
- 检查 `_ = err` 模式（仅允许在明确注释的场景）

```yaml
# .golangci.yml — 当前已有 errcheck，确认配置
linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
```

## Acceptance Criteria

- [ ] 审计扫描识别所有 P0/P1/P2 错误处理问题
- [ ] `internal/trading/` 修复：所有 error 正确 wrap + 无 panic + 无静默错误
- [ ] `internal/backtest/` 修复：同上
- [ ] `internal/market/` 修复：同上
- [ ] `go vet ./...` 通过无新 warning
- [ ] `golangci-lint` 的 errcheck 正确配置
- [ ] 审计结果记录到 plan 文件中，逐步修复

## Risks / Trade-offs

- **风险**: `%v` → `%w` 修改可能破坏下游 `errors.Is` 检查。→ 修改后运行全量测试确认
- **风险**: 大规模 error 修复引入回归。→ 按包分批修复，每个包修复后跑全量测试
- **Trade-off**: 不动第三方依赖的 error 处理（如 `github.com/coder/websocket` 的 error 风格无法控制）

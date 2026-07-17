# 错误处理审计 (Error Handling Audit) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Audit and fix P0/P1 error handling violations across `internal/trading/`, `internal/backtest/`, `internal/market/`, `internal/workflow/` — fixing `%v`→`%w`, removing `_ = err`, converting panic→error, and switching `log.Print`→`slog`.

**Architecture:** No new files. Fixes applied per-package in priority order. Each package gets `%v`→`%w` wrapping for all `fmt.Errorf` calls, elimination of `_ = err` patterns, replacement of non-recover panics with `slog.Error` + `return error`, and `log.Print*`→`slog.*`. After each package, `go vet ./pkg/...` and full test run confirm no regressions.

**Tech Stack:** Go 1.25, `slog`, `fmt.Errorf(%w)`, `go vet`, `golangci-lint` errcheck

## Global Constraints

- `%v` wrapping an `error` argument → `%w` (allows `errors.Is/As` chain)
- `_ = err` → handle error or explicitly discard with comment (`// intentionally discarded`)
- `panic(...)` → `slog.ErrorContext(ctx, ...)` + `return fmt.Errorf(...)` (unless truly unrecoverable)
- `log.Print*` → `slog.*` with context attributes
- Bare `return err` where call site has context → `return fmt.Errorf("context: %w", err)`
- After each package fix: `go vet ./internal/<pkg>/...` must pass, full test suite must pass
- Never modify third-party library error handling
- Commit per-package (5 commits total: audit script + 4 packages)

---

### Task 1: Audit Scan Script

**Files:**
- Create: `scripts/audit-errors.sh`

**Interfaces:**
- Produces: audit script that lists all violations per package, used by following tasks

- [ ] **Step 1: Write the audit script**

```bash
mkdir -p scripts
```

Create `scripts/audit-errors.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Error Handling Audit Script
# Usage: bash scripts/audit-errors.sh [package-dir]
# If no package given, scans all of internal/

PKG="${1:-internal/}"

echo "=== P0: Panic calls ==="
rg -n 'panic\(' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P0: _ = err (error swallowing) ==="
rg -n '_ = err' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P0: _ = <expr> (generic blank discard) ==="
rg -n '_, _ = ' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P1: %v wrapping error (should be %w) ==="
rg -n 'fmt\.Errorf.*%v.*err' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P1: bare return err (consider adding context) ==="
rg -n '^\s+return err$' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P2: log.Print / log.Printf / log.Println (should use slog) ==="
rg -n 'log\.(Print|Printf|Println)' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"

echo ""
echo "=== P2: JSON type assertion without ok check ==="
rg -n '\.\(\[\].*\)' "$PKG" --include='*.go' -g '!_test.go' || echo "(none)"
```

- [ ] **Step 2: Make it executable and run once**

```bash
chmod +x scripts/audit-errors.sh
bash scripts/audit-errors.sh internal/trading/
```

Expected: lists all violations in trading package.

- [ ] **Step 3: Commit**

```bash
git add scripts/audit-errors.sh
git commit -m "feat: add error handling audit scan script"
```

---

### Task 2: Fix `internal/trading/` (P0 Highest Priority)

**Files:**
- Modify: Various files in `internal/trading/`

**Interfaces:**
- Consumes: `scripts/audit-errors.sh` output for trading/

- [ ] **Step 1: Run audit to find specific violations**

```bash
bash scripts/audit-errors.sh internal/trading/
```

Record the output. For each violation, apply fixes per the patterns below.

- [ ] **Step 2: Fix `%v`→`%w` in error wrapping**

In every `fmt.Errorf` that wraps an `error` argument, change `%v` to `%w`.

Example fix in `internal/trading/engine.go` (representative — actual lines differ):

```go
// Before
return nil, fmt.Errorf("engine: process order failed: %v", err)

// After
return nil, fmt.Errorf("engine: process order failed: %w", err)
```

```go
// Before: return fmt.Errorf("engine snapshot failed: %v", err)
// After:  return fmt.Errorf("engine snapshot failed: %w", err)
```

- [ ] **Step 3: Remove `_ = err` patterns**

Replace each `_ = err` with proper error handling or an explicit discard with comment.

```go
// Before
_ = err

// After
if err != nil {
    slog.Warn("trading: non-fatal error discarded", "error", err)
}
```

If the error is truly intentionally discarded (e.g., cleanup in defer), add:

```go
// intentionally discarded — cleanup context already cancelled
```

- [ ] **Step 4: Replace panic with error return**

```go
// Before
data := result.([]any)

// After
data, ok := result.([]any)
if !ok {
    return nil, fmt.Errorf("trading: unexpected response type %T, want []any", result)
}
```

```go
// Before (non-recoverable program bugs stay as panic, but domain panics change)
panic(fmt.Sprintf("unexpected order type: %d", order.Type))

// After
return nil, fmt.Errorf("trading: unexpected order type: %d", order.Type)
```

- [ ] **Step 5: Replace `log.Print*` with `slog`**

```go
// Before
log.Printf("order executed: %s", order.ID)

// After
slog.Info("order executed", "order_id", order.ID, "symbol", order.Symbol)
```

- [ ] **Step 6: Run vet and tests**

```bash
go vet ./internal/trading/...
go test ./internal/trading/... -count=1
```

Expected: `go vet` passes, tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/trading/
git commit -m "fix: error handling audit - internal/trading P0/P1/P2 fixes"
```

---

### Task 3: Fix `internal/backtest/`

**Files:**
- Modify: Various files in `internal/backtest/`

**Interfaces:**
- Consumes: audit script output for backtest/

- [ ] **Step 1: Run audit**

```bash
bash scripts/audit-errors.sh internal/backtest/
```

- [ ] **Step 2: Apply all fixes (same patterns as Task 2)**

Fix all `%v`→`%w`:

```go
// Before: return fmt.Errorf("backtest: run failed: %v", err)
// After:  return fmt.Errorf("backtest: run failed: %w", err)
```

Fix `_ = err` patterns, panics, and log statements using same patterns as Task 2 Steps 2-5.

- [ ] **Step 3: Run vet and tests**

```bash
go vet ./internal/backtest/...
go test ./internal/backtest/... -count=1
```

Expected: `go vet` passes, tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/backtest/
git commit -m "fix: error handling audit - internal/backtest P0/P1/P2 fixes"
```

---

### Task 4: Fix `internal/market/`

**Files:**
- Modify: Various files in `internal/market/`

**Interfaces:**
- Consumes: audit script output for market/

- [ ] **Step 1: Run audit**

```bash
bash scripts/audit-errors.sh internal/market/
```

- [ ] **Step 2: Apply all fixes (same patterns as Task 2)**

Fix all `%v`→`%w`:

```go
// Before: return fmt.Errorf("adapter %s: fetch failed: %v", name, err)
// After:  return fmt.Errorf("adapter %s: fetch failed: %w", name, err)
```

Fix `_ = err` patterns, panics, and log statements using same patterns as Task 2 Steps 2-5.

- [ ] **Step 3: Run vet and tests**

```bash
go vet ./internal/market/...
go test ./internal/market/... -count=1
```

Expected: `go vet` passes, tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/market/
git commit -m "fix: error handling audit - internal/market P0/P1/P2 fixes"
```

---

### Task 5: Fix `internal/workflow/` + CI errcheck Config

**Files:**
- Modify: Various files in `internal/workflow/`
- Modify: `.golangci.yml`

**Interfaces:**
- Consumes: audit script output for workflow/
- Produces: hardened `.golangci.yml` errcheck config

- [ ] **Step 1: Run audit**

```bash
bash scripts/audit-errors.sh internal/workflow/
```

- [ ] **Step 2: Apply all fixes (same patterns as Task 2)**

Fix all `%v`→`%w`:

```go
// Before: return fmt.Errorf("workflow: execute node failed: %v", err)
// After:  return fmt.Errorf("workflow: execute node failed: %w", err)
```

Fix `_ = err` patterns, panics, and log statements using same patterns as Task 2 Steps 2-5.

- [ ] **Step 3: Harden `.golangci.yml` errcheck config**

Update `.golangci.yml` to enable type assertion and blank check:

```yaml
linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
```

- [ ] **Step 4: Run vet and full lint**

```bash
go vet ./internal/workflow/...
golangci-lint run ./internal/workflow/... --timeout 5m
go test ./internal/workflow/... -count=1
```

Expected: `go vet`, `golangci-lint`, and tests all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/workflow/ .golangci.yml
git commit -m "fix: error handling audit - internal/workflow P0/P1/P2 fixes + errcheck hardening"
```

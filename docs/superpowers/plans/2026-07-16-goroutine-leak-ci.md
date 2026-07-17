# Goroutine 泄漏检测 (Goroutine Leak Detection in CI) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate `go.uber.org/goleak` into all high-risk goroutine packages (`ws/`, `market/`, `workflow/`) and add a CI leak check step.

**Architecture:** Each high-risk package's `TestMain` calls `goleak.VerifyTestMain(m)` with `goleak.IgnoreAny()` to filter Wails-internal goroutines. CI runs a dedicated `go test -run TestMain` step per package. `go.uber.org/goleak` is added as a test-only dependency via `go get`.

**Tech Stack:** Go 1.25, `go.uber.org/goleak`, `go test -run TestMain`

## Global Constraints

- `goleak.VerifyTestMain(m)` in `TestMain` of each target package
- `goleak.IgnoreAny()` to suppress Wails framework background goroutines
- No modifications to production code — test-only changes
- CI step checks `internal/ws/`, `internal/market/`, `internal/workflow/`
- `go test -run TestMain ./internal/<pkg>/... -count=1 -timeout 30s` per package in CI
- Default goleak timeout of 1s (implicit)
- Table-driven tests in existing test files remain unchanged

---

### Task 1: Add `go.uber.org/goleak` Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum` (auto-generated)

- [ ] **Step 1: Add the dependency**

```bash
go get go.uber.org/goleak@latest
```

Expected output:
```
go: added go.uber.org/goleak v1.3.0
```

- [ ] **Step 2: Verify go.mod**

```bash
grep goleak go.mod
```

Expected: `go.uber.org/goleak v1.3.0` appears in require block.

- [ ] **Step 3: Tidy**

```bash
go mod tidy
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add go.uber.org/goleak for goroutine leak detection in tests"
```

---

### Task 2: Add goleak to `internal/ws/` Tests

**Files:**
- Modify: `internal/ws/hub_test.go` (add TestMain)

- [ ] **Step 1: Add `TestMain` with goleak to hub_test.go**

Prepend to `internal/ws/hub_test.go`:

```go
package ws

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAny(),
	)
}
```

Note: `"testing"` import already exists in the file; remove the duplicate.

The full file becomes:

```go
package ws

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAny(),
	)
}

func TestHubSubscribeBroadcast(t *testing.T) {
	// ... existing tests unchanged
```

- [ ] **Step 2: Run tests to verify no leaks**

```bash
go test ./internal/ws/... -count=1 -timeout 30s
```

Expected: all existing tests pass with no leak detection errors.

- [ ] **Step 3: Commit**

```bash
git add internal/ws/hub_test.go
git commit -m "test: add goroutine leak detection to internal/ws tests"
```

---

### Task 3: Add goleak to `internal/market/` Tests

**Files:**
- Modify: `internal/market/hub_test.go` (add TestMain)

- [ ] **Step 1: Add `TestMain` with goleak to hub_test.go**

Prepend to `internal/market/hub_test.go`:

```go
package market

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAny(),
	)
}
```

Note: `"testing"` import already exists; remove the duplicate. The `go.uber.org/goleak` import is new.

- [ ] **Step 2: Run tests to verify no leaks**

```bash
go test ./internal/market/... -count=1 -timeout 30s
```

Expected: all existing tests pass with no leak detection errors.

- [ ] **Step 3: Commit**

```bash
git add internal/market/hub_test.go
git commit -m "test: add goroutine leak detection to internal/market tests"
```

---

### Task 4: Add goleak to `internal/workflow/` Tests

**Files:**
- Modify: `internal/workflow/engine_test.go` (add TestMain)

- [ ] **Step 1: Add `TestMain` with goleak to engine_test.go**

Prepend to `internal/workflow/engine_test.go`:

```go
package workflow

import (
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAny(),
	)
}
```

- [ ] **Step 2: Run tests to verify no leaks**

```bash
go test ./internal/workflow/... -count=1 -timeout 30s
```

Expected: all existing tests pass with no leak detection errors.

- [ ] **Step 3: Commit**

```bash
git add internal/workflow/engine_test.go
git commit -m "test: add goroutine leak detection to internal/workflow tests"
```

---

### Task 5: Add Goroutine Leak Check to CI

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Add leak check step to CI**

After the `go test -race` step in the backend job, add:

```yaml
      - run: go test -run TestMain ./internal/ws/... -count=1 -timeout 30s
      - run: go test -run TestMain ./internal/market/... -count=1 -timeout 30s
      - run: go test -run TestMain ./internal/workflow/... -count=1 -timeout 30s
```

The full backend job becomes:

```yaml
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25' }
      - uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
      - run: go build ./...
      - run: go vet ./...
      - run: golangci-lint run ./... --timeout 5m
      - run: go test -race ./... -count=1
      - run: go test -run TestMain ./internal/ws/... -count=1 -timeout 30s
      - run: go test -run TestMain ./internal/market/... -count=1 -timeout 30s
      - run: go test -run TestMain ./internal/workflow/... -count=1 -timeout 30s
```

- [ ] **Step 2: Validate CI YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML valid')"
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add goroutine leak detection step for ws/market/workflow packages"
```

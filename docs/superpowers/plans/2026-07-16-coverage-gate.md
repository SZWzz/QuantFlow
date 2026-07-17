# 测试覆盖率门禁 (Coverage Gate) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-package coverage thresholds with CI enforcement and local Makefile target.

**Architecture:** A JSON config file defines per-package coverage targets; a Makefile target loops over them with `go test -coverprofile` and `go tool cover -func`, failing if any package is below threshold. CI runs this target alongside existing steps. Frontend uses vitest `--coverage` with a 40% line-coverage gate.

**Tech Stack:** Go `go test -cover`, bash `awk/sed`, vitest `--coverage`, `@vitest/coverage-v8`

## Global Constraints

- Threshold config lives in `coverage-gate.json` at repository root
- Each core package checked independently (not aggregate coverage)
- Frontend coverage: 40% line-coverage minimum
- New packages have 6-month grace period (`"exempt": true` + reason)
- No third-party coverage services (no codecov/sonarcloud)
- Go version in CI: 1.25
- Node version in CI: 20

---

### Task 1: Coverage Gate Config File

**Files:**
- Create: `coverage-gate.json`

**Interfaces:**
- Produces: `coverage-gate.json` — consumed by Makefile target and CI step

- [ ] **Step 1: Define the config file**

Create `coverage-gate.json` at repo root with thresholds from the spec:

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
  },
  "exemptions": [
    {
      "package": "internal/ws/",
      "reason": "New package, 6-month grace period until 2027-01-16",
      "exempt_until": "2027-01-16"
    },
    {
      "package": "internal/auth/",
      "reason": "New package, 6-month grace period until 2027-01-16",
      "exempt_until": "2027-01-16"
    },
    {
      "package": "internal/notify/",
      "reason": "New package, 6-month grace period until 2027-01-16",
      "exempt_until": "2027-01-16"
    },
    {
      "package": "internal/schedule/",
      "reason": "New package, 6-month grace period until 2027-01-16",
      "exempt_until": "2027-01-16"
    }
  ]
}
```

- [ ] **Step 2: Validate JSON**

```bash
python3 -m json.tool coverage-gate.json > /dev/null
echo "JSON is valid"
```

- [ ] **Step 3: Commit**

```bash
git add coverage-gate.json
git commit -m "feat: add coverage gate config with per-package thresholds"
```

---

### Task 2: Makefile Coverage Gate Target

**Files:**
- Modify: `Makefile:35-38`

**Interfaces:**
- Consumes: `coverage-gate.json` (thresholds + exemptions)
- Produces: `make coverage-gate` command, `make coverage-html` command

- [ ] **Step 1: Add the `coverage-gate` and `coverage-html` targets to Makefile**

Replace the existing `coverage` block:

```makefile
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-gate:
	@echo "→ Running coverage gate checks..."
	@has_fail=0; \
	for pkg in $$(python3 -c "import json; d=json.load(open('coverage-gate.json')); [print(k) for k in d['thresholds'] if not k.startswith('frontend')]"); do \
		exempt=$$(python3 -c "import json; d=json.load(open('coverage-gate.json')); print(any(e['package']=='$$pkg' for e in d.get('exemptions',[])))"); \
		if [ "$$exempt" = "True" ]; then \
			echo "  ⏩ Skipping $${pkg} (exempted)"; \
			continue; \
		fi; \
		threshold=$$(python3 -c "import json; d=json.load(open('coverage-gate.json')); print(d['thresholds']['$$pkg'])"); \
		out=$$(echo "$$pkg" | tr '/' '_'); \
		go test "./$$pkg" -coverprofile="/tmp/cov_$${out}.out" -coverpkg="./$$pkg/..." 2>/dev/null; \
		cov=$$(go tool cover -func="/tmp/cov_$${out}.out" 2>/dev/null | grep 'total:' | awk '{print $$3}' | sed 's/%//'); \
		if [ -z "$$cov" ]; then cov=0; fi; \
		if [ "$$(echo "$$cov < $$threshold" | bc -l)" -eq 1 ]; then \
			echo "  ❌ $${pkg} coverage $${cov}% < $${threshold}%"; \
			has_fail=1; \
		else \
			echo "  ✅ $${pkg} coverage $${cov}% >= $${threshold}%"; \
		fi; \
	done; \
	if [ "$$has_fail" -eq 1 ]; then exit 1; fi; \
	echo "→ Coverage gate passed"

coverage-html:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "→ Open coverage.html in browser"

clean:
	rm -f coverage.out coverage.html
```

- [ ] **Step 2: Verify the Makefile syntax**

```bash
make -n coverage-gate 2>&1 | head -5
```

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "feat: add coverage-gate and coverage-html Makefile targets"
```

---

### Task 3: CI Coverage Gate Step

**Files:**
- Modify: `.github/workflows/ci.yml:21`

**Interfaces:**
- Consumes: `coverage-gate.json`, `make coverage-gate`
- Produces: CI coverage gate check in backend job

- [ ] **Step 1: Add coverage gate step to CI**

Insert after `go test -race` in the backend job:

```yaml
      - run: go test -race ./... -count=1
      - run: make coverage-gate
        env:
          GOFLAGS: -count=1
      - run: go build ./...
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
      - run: make coverage-gate
        env:
          GOFLAGS: -count=1
```

- [ ] **Step 2: Validate YAML syntax**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML valid')"
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "feat: add coverage gate step to CI pipeline"
```

---

### Task 4: Frontend Coverage Configuration

**Files:**
- Modify: `frontend/package.json:13-16`
- Modify: `frontend/vitest.config.ts` (if it exists, create if not)

**Interfaces:**
- Consumes: `coverage-gate.json` (frontend threshold)
- Produces: `npm run coverage` command that generates coverage report

- [ ] **Step 1: Read existing vitest config**

```bash
cat frontend/vitest.config.ts 2>/dev/null || echo "not found"
```

- [ ] **Step 2: Add coverage script to frontend/package.json and configure vitest**

Add `"coverage"` script to `scripts` block (after `"test:watch"`):

```json
    "coverage": "vitest run --coverage"
```

Add `@vitest/coverage-v8` dev dependency and configure vitest config:

```bash
cd frontend && npm install --save-dev @vitest/coverage-v8
```

Update `frontend/vitest.config.ts` (create if not exists):

```typescript
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
      exclude: ['src/**/*.test.ts', 'src/**/*.spec.ts', 'src/**/*.d.ts'],
      thresholds: {
        lines: 40,
      },
    },
  },
})
```

- [ ] **Step 3: Run frontend coverage to verify it works**

```bash
cd frontend && npm run coverage
```

Expected: vitest runs tests and outputs coverage summary with line coverage >= 40%.

- [ ] **Step 4: Add frontend coverage step to CI**

In the `frontend` job, replace the vitest run step with:

```yaml
      - run: cd frontend && npx vitest run --coverage --bail=5
```

- [ ] **Step 5: Validate CI YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml')); print('YAML valid')"
```

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/vitest.config.ts .github/workflows/ci.yml
git commit -m "feat: add frontend coverage with vitest v8 provider and CI gate"
```

# Fix Code Quality Issues

## Motivation

Structural and quality issues across the codebase:

1. **Duplicated market resolution logic** — `MarketForSymbol` in `registry.go` and `resolveMarketFromSymbol` in `app.go` are near-identical but diverge over time.
2. **Backtest engine uses `log.Printf`** — Bypasses structured logging and ring buffer; messages invisible in LogPanel.
3. **Vite build no optimization** — No `manualChunks`, `rollupOptions`, bundle analysis.
4. **No frontend ESLint** — Only `vue-tsc` for type checking, no style/lint enforcement.
5. **`.golangci.yml` only 9 linters** — Missing `gofmt`/`gofumpt`, `gocyclo`, `revive`, `gocritic`.
6. **`app.go` >1500 lines** — Monolithic.
7. **`main.go` uses `"log"` package** — Inconsistent with `log/slog` used elsewhere.
8. **`go.mod` version mismatch** — Says `go 1.25.0`, CLAUDE.md says `Go 1.22+`.
9. **Package-level docstrings missing** — Several critical packages lack documentation.

## Design

### 1. Deduplicate market resolution

**File**: `internal/market/registry.go`

Make `MarketForSymbol` the single source of truth. Export it:

```go
func MarketForSymbol(symbol string) string { ... }
```

**File**: `app.go` — Remove `resolveMarketFromSymbol`. Replace all calls with `market.MarketForSymbol`.

### 2. Replace log.Printf with slog in backtest

**File**: `internal/backtest/engine_cn.go`

Replace:
```go
log.Printf("Order filled: %s %d@%.2f", ...)
```
With:
```go
slog.Debug("order filled", "symbol", ..., "quantity", ..., "price", ...)
```

Apply to all ~6 occurrences in `engine_cn.go` and `engine_us.go`.

### 3. Optimize Vite build

**File**: `frontend/vite.config.ts`

```ts
build: {
  outDir: 'dist',
  emptyOutDir: true,
  target: 'es2020',
  chunkSizeWarningLimit: 500,
  rollupOptions: {
    output: {
      manualChunks: {
        'vendor-vue': ['vue', 'vue-router', 'vue-i18n', 'pinia'],
        'vendor-flow': ['@vue-flow/core', '@vue-flow/background', '@vue-flow/controls', '@vue-flow/minimap'],
        'vendor-chart': ['echarts', 'vue-echarts'],
        'vendor-wails': ['@wailsio/runtime'],
      },
    },
  },
}
```

### 4. Add ESLint config

**File**: `frontend/.eslintrc.cjs`

```js
module.exports = {
  root: true,
  env: { browser: true, es2021: true, node: true },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:vue/vue3-recommended',
  ],
  parser: 'vue-eslint-parser',
  parserOptions: { parser: '@typescript-eslint/parser', ecmaVersion: 'latest' },
  rules: {
    'vue/multi-word-component-names': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    'no-console': ['warn', { allow: ['warn', 'error'] }],
  },
}
```

Add lint script to `frontend/package.json`:
```json
"lint": "eslint src --ext .ts,.vue --fix"
```

### 5. Expand golangci-lint config

**File**: `.golangci.yml`

Enable additional linters:

```yaml
linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - bodyclose
    - gosimple
    - ineffassign
    - unused
    - misspell
    - gofumpt
    - gocyclo
    - revive
    - goconst
    - gocritic
```

### 6. Split app.go

**File**: `app.go` — Extract `ServiceStartup` (~400 lines) into `app_startup.go`:

```go
// app_startup.go — Application initialization and service wiring
func (a *App) ServiceStartup(ctx context.Context, options service.ServiceOptions) error { ... }
```

Also extract `ServiceShutdown` into `app_shutdown.go` if it grows.

### 7 & 8. Fix misc issues

**File**: `main.go` — Replace `log.Fatal` with `slog.Error` + `os.Exit(1)`.

**File**: `CLAUDE.md` — Update version from `Go 1.22+` to `Go 1.25+`.

### 9. Add package-level docstrings

Add docstrings to:

```go
// Package market provides real-time and historical market data access
// with automatic adapter selection and fallback. Key abstractions:
// Registry, MarketDataHub, and OffHoursCache.
package market

// Package workflow implements a Kahn-based DAG execution engine
// for composing trading strategies from reusable node types.
package workflow
```

Files to update: `internal/market/`, `internal/workflow/`, `internal/backtest/`, `internal/schedule/`, `internal/notify/`.

### Modified files

| File | Change |
|------|--------|
| `internal/market/registry.go` | Export `MarketForSymbol`, add docstring |
| `app.go` | Remove `resolveMarketFromSymbol`, call `market.MarketForSymbol` |
| `internal/backtest/engine_cn.go` | `log.Printf` → `slog.Debug` |
| `internal/backtest/engine_us.go` | Same |
| `frontend/vite.config.ts` | Add build optimization |
| `frontend/.eslintrc.cjs` | **New** — ESLint config |
| `frontend/package.json` | Add `lint` script |
| `.golangci.yml` | Add more linters |
| `app_startup.go` | **New** — extracted from app.go |
| `main.go` | `log.Fatal` → `slog.Error + os.Exit(1)` |
| `CLAUDE.md` | Go version → 1.25+ |
| `internal/market/` | Add package docstring |
| `internal/workflow/` | Add package docstring |
| `internal/backtest/` | Add package docstring |
| `internal/schedule/` | Add package docstring |
| `internal/notify/` | Add package docstring |

### API changes

- `MarketForSymbol` exported (was unexported)
- `resolveMarketFromSymbol` removed (use `market.MarketForSymbol` instead)
- No gRPC or frontend changes

## Acceptance Criteria

- [ ] `go vet ./...` passes
- [ ] `golangci-lint run ./...` passes with all enabled linters
- [ ] `cd frontend && npx eslint src --ext .ts,.vue` passes
- [ ] `cd frontend && npx vue-tsc --noEmit` passes
- [ ] `cd frontend && npm run build` produces optimized chunks
- [ ] All Go tests pass
- [ ] Backtest log messages appear in LogPanel
- [ ] CLAUDE.md version matches go.mod

## Risks / Trade-offs

- **Vite manualChunks**: May increase initial load size by splitting vendor code. Mitigation: verify with `rollup-plugin-visualizer`.
- **golangci-lint new linters**: `gocyclo`, `gocritic` may produce many new warnings. Triage as non-blocking for initial pass.
- **ESLint `no-console`**: May flag intentional `console.log` debug output. Configure as `warn` only, not error.
- **app.go split**: No behavior change, purely organizational.

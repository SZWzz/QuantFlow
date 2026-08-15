# Contributing to QuantFlow Terminal

Thank you for your interest in QuantFlow Terminal -- a dual-mode quantitative finance terminal combining a Bloomberg-style panel terminal with visual workflow orchestration.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/shenyzw/quantflow.git
cd quantflow

# Install Go dependencies
cd app && go mod download && cd ..

# Install frontend dependencies
cd frontend && npm install && cd ..

# Install Python sidecar dependencies
cd python && pip install -e ".[dev,data]" && cd ..

# Run development mode
wails dev
```

## Pull Request Workflow

1. **Branch** -- Create a feature branch from `main`:
   ```bash
   git checkout main
   git pull origin main
   git checkout -b feat/my-feature
   ```

2. **Spec** -- For non-trivial changes, write a spec document first:
   ```
   docs/specs/YYYY-MM-DD-<slug>.md
   ```

3. **Plan** -- After the spec is approved, create an implementation plan:
   ```
   docs/superpowers/plans/YYYY-MM-DD-<feature-name>.md
   ```

4. **Implement** -- Write code following the project's coding standards (see below).

5. **Test** -- Run the full test suite:
   ```bash
   cd app && go vet ./... && go test ./... -v -count=1
   cd frontend && npx vue-tsc --noEmit && npx vitest run
   cd python && python -m pytest tests/ -x -q
   ```

6. **Review** -- Open a pull request against `main`. Ensure CI passes.

7. **Merge** -- Squash-merge into `main` with a descriptive commit message.

## Coding Standards

### Go (version 1.25+)
- Use `slog` for logging
- Explicit error returns; no panics in library code
- Favor `sync.Map` over `sync.RWMutex` for read-heavy caches
- Group imports: stdlib, third-party, internal

### Vue 3 (Composition API with `<script setup lang="ts">`)
- Use Pinia for state management
- Subscribe to data topics via `useDataStore().subscribe()`
- Never use `window.confirm()` or `window.alert()` -- use `confirmDialog()` / `alertDialog()` from `@/lib/wails`
- Style: scoped `<style>` blocks

### Python 3.12+
- Use `async def` for gRPC service methods
- Type hints required for all function signatures
- Format with `ruff`
- No direct SQLite access -- all storage goes through Go

## Commit Message Format

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:** `feat`, `fix`, `chore`, `refactor`, `docs`, `test`, `style`, `perf`, `ci`, `build`

**Scopes:** `Terminal`, `Workflow`, `Engine`, `Broker`, `MarketData`, `AI`, `Frontend`, `Storage`, `Python`, `Docs`

**Examples:**
```
feat(terminal): add dark mode toggle to panel header
fix(broker): handle Alpaca rate limit retry with exponential backoff
docs(workflow): add architecture diagram for DAG executor
```

## Detailed Rules

See [CLAUDE.md](./CLAUDE.md) for the full project guidelines, including:

- Changelog maintenance requirements
- Version date checks before commits
- Documentation requirements for financial-critical code
- Architecture overview and directory structure
- Market focus and key design decisions
- Technology stack and dependencies

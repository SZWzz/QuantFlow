# P2 Dev Infrastructure — CI/CD, Lint, Pre-commit, EditorConfig, TypeScript Strict

## Motivation

项目缺少现代 Go/TS 项目的标准开发基础设施，导致代码质量主要靠人工 review，没有自动化防守：

1. **无 CI/CD** — 无 PR check、无自动化测试运行、无 lint 检查
2. **无 golangci-lint** — `make lint` 仅跑 `go vet`，缺失 staticcheck、errcheck、bodyclose 等
3. **无 pre-commit hooks** — 提交前不自动验证
4. **无 `.editorconfig`** — 跨编辑器缩进/编码不统一
5. **`tsconfig.json` 未开启 `strict: true`** — 允许隐式 `any`，TypeScript 安全性打折
6. **无 `.env.example`** — 环境变量无文档

## Design

### 1. GitHub Actions CI

**.github/workflows/ci.yml**:

```yaml
name: CI
on: [push, pull_request]
jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go vet ./...
      - run: go test ./... -count=1

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: cd frontend && npm ci
      - run: cd frontend && npx vue-tsc --noEmit
      - run: cd frontend && npx vitest run

  python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: cd python && pip install -e ".[dev,data]"
      - run: cd python && python -m pytest tests/ -x -q
```

### 2. golangci-lint

**.golangci.yml**:

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
```

`make lint` 改为跑 `golangci-lint run` 而非仅 `go vet`。

### 3. Pre-commit Hooks

**.pre-commit-config.yaml**:

```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v2.0
    hooks:
      - id: golangci-lint
  - repo: https://github.com/pre-commit/mirrors-prettier
    rev: v4.0
    hooks:
      - id: prettier
        types_or: [javascript, typescript, vue, css]
  - repo: local
    hooks:
      - id: go-vet
        name: go vet
        entry: go vet ./...
        language: system
        files: \.go$
      - id: vue-tsc
        name: vue-tsc
        entry: bash -c 'cd frontend && npx vue-tsc --noEmit'
        language: system
        files: 'frontend/.*\.(ts|vue)$'
```

### 4. .editorconfig

```ini
root = true

[*]
indent_style = space
indent_size = 2
end_of_line = lf
charset = utf-8
trim_trailing_whitespace = true
insert_final_newline = true

[*.go]
indent_size = 4  # Go 标准 tab

[Makefile]
indent_style = tab
```

### 5. tsconfig.json 开启 strict

**关键改动**：
```json
{
  "compilerOptions": {
    "strict": true,
    // 移除独立设置：
    // "strictNullChecks": true,       // 被 strict 包含
    // "strictFunctionTypes": true,    // 被 strict 包含
    // "noImplicitReturns": true,      // 被 strict 包含
    // "noFallthroughCasesInSwitch": true // 被 strict 包含
  }
}
```

需修复 strict 开启后新增的类型错误（预估 20-50 处）。

### 6. .env.example

```env
# Broker API Keys (a-share brokers are optional; US/crypto require keys)
ALPACA_API_KEY=
ALPACA_SECRET_KEY=FUTU_OPENAPI_TOKEN=
BINANCE_API_KEY=
BINANCE_SECRET_KEY=

# Data API Keys
POLYGON_API_KEY=
FINNHUB_API_KEY=
FRED_API_KEY=

# AI / LLM
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
```

## Acceptance Criteria

- [ ] `.github/workflows/ci.yml` 存在且 GitHub 可识别
- [ ] `.golangci.yml` 存在，`make lint` 使用 golangci-lint
- [ ] `.pre-commit-config.yaml` 存在，`pre-commit run --all-files` 通过
- [ ] `.editorconfig` 存在
- [ ] `tsconfig.json` 开启 `strict: true`，`vue-tsc --noEmit` 通过
- [ ] `.env.example` 存在

## Risks / Trade-offs

- tsconfig strict 模式会暴露现有类型错误（~20-50 处）。可分阶段开启：先加 `noImplicitAny: true`，再逐步补到完整 strict。
- golangci-lint 首次运行可能报大量现有 lint 错误。建议先配置 `issues.exclude` 排除已知问题，逐步收紧。

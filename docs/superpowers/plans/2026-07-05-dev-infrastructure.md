# 实施计划：Dev Infrastructure

参考：`docs/specs/2026-07-05-dev-infrastructure.md`

## Task 1: 创建 .editorconfig

**新建文件 `/.editorconfig`**：

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
indent_size = 4
indent_style = tab

[{Makefile, makefile, GNUmakefile}]
indent_style = tab
indent_size = 4

[*.md]
trim_trailing_whitespace = false
```

---

## Task 2: 创建 .env.example

**新建文件 `/.env.example`**：

```env
# Trading Broker API Keys
ALPACA_API_KEY=
ALPACA_SECRET_KEY=
FUTU_OPENAPI_TOKEN=
BINANCE_API_KEY=
BINANCE_SECRET_KEY=

# Data Source API Keys
POLYGON_API_KEY=
FINNHUB_API_KEY=
FRED_API_KEY=
EASTMONEY_ACCOUNT=
EASTMONEY_PASSWORD=

# AI / LLM
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# Python Sidecar
PYTHON_SIDECAR_PORT=50051

# Application
CONFIG_DIR=~/.config/quantflow
LOG_LEVEL=info
```

---

## Task 3: 创建 GitHub Actions CI

**新建文件 `.github/workflows/ci.yml`**：

```yaml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...
      - run: go test ./... -count=1

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: cd frontend && npm ci
      - run: cd frontend && npx vue-tsc --noEmit
      - run: cd frontend && npx vitest run

  python:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - run: cd python && pip install -e ".[dev,data]"
      - run: cd python && python -m pytest tests/ -x -q
```

---

## Task 4: 创建 golangci-lint 配置

**新建文件 `.golangci.yml`**：

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

linters-settings:
  errcheck:
    exclude-functions:
      - io.ReadAll
      - io.Copy

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

**更新 `Makefile`** 中 lint 命令：

```makefile
.PHONY: lint
lint:  # 原: go vet ./...
	golangci-lint run ./...
```

---

## Task 5: 创建 pre-commit 配置

**新建文件 `.pre-commit-config.yaml`**：

```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.64.6
    hooks:
      - id: golangci-lint
        args: [--timeout=300s]
  - repo: https://github.com/pre-commit/mirrors-prettier
    rev: v4.0.0-alpha.8
    hooks:
      - id: prettier
        types_or: [javascript, typescript, vue, css, html]
  - repo: local
    hooks:
      - id: go-vet
        name: go vet
        entry: go vet ./...
        language: system
        files: \.go$
        pass_filenames: false
      - id: vue-tsc
        name: vue-tsc
        entry: bash -c 'cd frontend && npx vue-tsc --noEmit'
        language: system
        files: ^frontend/.*\.(ts|vue)$
        pass_filenames: false
      - id: vitest
        name: vitest
        entry: bash -c 'cd frontend && npx vitest run'
        language: system
        files: ^frontend/
        pass_filenames: false
```

---

## Task 6: tsconfig strict

**`frontend/tsconfig.json`**，将现有零散选项替换为 `"strict": true`：

```json
{
  "compilerOptions": {
    "strict": true,
    // 移除下列零散选项（被 strict 包含）：
    // "strictNullChecks": true,
    // "strictFunctionTypes": true,
    // "strictBindCallApply": true,
    // "strictPropertyInitialization": true,
    // "noImplicitAny": true,
    // "noImplicitThis": true,
    // "alwaysStrict": true
  }
}
```

更改后运行 `npx vue-tsc --noEmit` 查看新增错误数量，评估是否需要分阶段。

---

## Task 7: 验证

```bash
# EditorConfig
ls -la .editorconfig

# CI 配置文件语法（用 yamllint 或简单检查）
python -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"

# golangci-lint（如已安装）
golangci-lint run ./... --timeout=300s || echo "expected some existing lint errors"

# pre-commit（如已安装）
pre-commit run --all-files || echo "expected some existing errors"

# tsconfig
cd frontend && npx vue-tsc --noEmit || echo "expected some existing type errors"
```

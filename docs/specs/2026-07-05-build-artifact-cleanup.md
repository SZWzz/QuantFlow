# Build Artifact Cleanup — .gitignore and Repository Hygiene

## Motivation

审计发现 `build/` 和 `dist/` 目录被 Git 跟踪。`build/` 包含 33MB 的二进制文件和其他构建产物，不应版本管理。

影响：
- `git clone` 下载量增大
- 二进制文件无法 diff，占用仓库空间
- 构建产物冲突（不同机器产生不同二进制）

## Design

### 1. 更新 .gitignore

**`/.gitignore`** 追加：
```gitignore
# Build artifacts
build/
dist/
frontend/dist/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Environment
.env
*.local

# Python
__pycache__/
*.pyc
.pytest_cache/
*.egg-info/

# Go
vendor/
```

### 2. 从 Git 跟踪中移除已有文件

```bash
git rm -r --cached build/
git rm -r --cached dist/
git rm -r --cached frontend/dist/
```

### 3. 确认关键文件不被误伤

检查 `build/` 下有无必要的配置文件（如 Go 的 `app/build/` 有编译配置？不影响，重新检查目录结构）。如果 `resources/` 或 `build/` 下有 `.gitkeep` 文件，保留方式：在 `.gitignore` 中加 `!build/.gitkeep`。

## Acceptance Criteria

- [ ] `.gitignore` 阻止 `build/`、`dist/`、IDE 文件、OS 文件
- [ ] `git rm --cached` 执行后这些目录不再被跟踪
- [ ] 仓库大小减小 30MB+
- [ ] `git status` 干净（仅 .gitignore 有改动）

## Risks / Trade-offs

- 已经在仓库中的二进制文件仍存在于 Git 历史中（除非改写历史）。本 spec 只处理当前版本去跟踪，不影响历史。改写历史需要 `git filter-branch` 或 `git filter-repo`，风险大、需团队沟通。

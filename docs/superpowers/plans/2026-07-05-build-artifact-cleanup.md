# 实施计划：Build Artifact Cleanup

参考：`docs/specs/2026-07-05-build-artifact-cleanup.md`

## Task 1: 更新 .gitignore

**编辑 `/.gitignore`**，追加构建产物和 IDE 文件：

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

---

## Task 2: 从 Git 移除已跟踪的构建产物

```bash
git rm -r --cached build/
git rm -r --cached dist/
git rm -r --cached frontend/dist/
```

**检查**：`git status` 应显示 staged changes for these directories。

---

## Task 3: 检查是否误伤 .gitkeep 文件

```bash
find build/ -name '.gitkeep' 2>/dev/null
find dist/ -name '.gitkeep' 2>/dev/null
```

如果存在，在 `.gitignore` 中添加 `!build/**/.gitkeep` 等规则。

---

## Task 4: 验证

```bash
git status  # 确认 build/ dist/ frontend/dist/ 不再跟踪
git ls-files build/  # 应无输出
git ls-files dist/   # 应无输出
du -sh .git          # 记录当前大小，确认后续 push 不会继续增长
```

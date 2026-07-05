# Python Dependency Alignment — requirements.txt, pyproject.toml, requirements.lock

## Motivation

Python sidecar 的依赖管理有三处不一致：

1. **`pyproject.toml` 和 `requirements.txt` 的依赖列表不同** — 双方各有一些对方没有的包（missing 10+ entries on each side）
2. **`requirements.lock` 只锁了 10/34 个包** — 剩余 24 个包无版本锁定，部署时可能因版本差异出问题
3. **健康服务版本号硬编码为 `2026.6.26`**，与 `pyproject.toml` 的 `0.2.3` 不匹配

## Design

### 1. 对齐 pyproject.toml 和 requirements.txt

以 `pyproject.toml` 为 truth source，`requirements.txt` 由 `pip freeze` 生成。

当前两边的差异：

| 包 | pyproject.toml | requirements.txt | 结论 |
|---|:---:|:---:|------|
| grpcio | ✅ | ✅ | 保留 |
| protobuf | ✅ | ✅ | 保留 |
| pandas | ✅ | ✅ | 保留 |
| numpy | ✅ | ✅ | 保留 |
| pyarrow | ✅ | ✅ | 保留 |
| scikit-learn | ✅ | ✅ | 保留 |
| xgboost | ✅ | ✅ | 保留 |
| lightgbm | ✅ | ✅ | 保留 |
| pyyaml | ✅ | ✅ | 保留 |
| mootdx | dev/optional | ✅ | 加到 dev |
| httpx | ✅ | ❌ | 加到 requirements.txt |
| joblib | ✅ | ❌ | 加到 requirements.txt |
| torch | ✅ | ❌ | 加到 requirements.txt |
| gymnasium | ✅ | ❌ | 加到 requirements.txt |
| arch | ✅ | ❌ | 加到 requirements.txt |
| gplearn | ✅ | ❌ | 加到 requirements.txt |
| nltk | ❌ | ✅ | 加到 pyproject.toml optional |
| snownlp | ❌ | ✅ | 加到 pyproject.toml optional |
| ccxt | ❌ | ✅ | 加到 pyproject.toml optional |
| akshare | ❌ | ✅ | 加到 pyproject.toml optional |
| edgartools | ❌ | ✅ | 加到 pyproject.toml optional |
| scipy | ❌ | ✅ | 加到 pyproject.toml optional |
| statsmodels | ❌ | ✅ | 加到 pyproject.toml optional |
| textblob | ❌ | ✅ | 加到 pyproject.toml optional |
| jieba | ❌ | ✅ | 加到 pyproject.toml optional |
| yfinance | ❌ | ✅ | 加到 pyproject.toml optional |

### 2. 更新 requirements.lock

运行 `pip freeze > requirements.lock` 并在 CI 中验证 lock 文件与 pyproject.toml 一致。

### 3. 修复版本号

**`python/src/server.py:81`**：`version = "2026.6.26"` → 从 `pyproject.toml` 读取 `__version__`。

## Acceptance Criteria

- [ ] `pyproject.toml` 包含所有运行时依赖和 optional 分组
- [ ] `requirements.txt` 是 `pyproject.toml` 的直接导出（或同步维护的一致列表）
- [ ] `requirements.lock` 包含所有 34+ 包的版本锁定
- [ ] 健康服务返回的版本号与 `pyproject.toml` 一致
- [ ] `python -m pytest tests/ -x -q` 通过

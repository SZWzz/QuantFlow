# 实施计划：Python Deps Alignment

参考：`docs/specs/2026-07-05-python-deps-alignment.md`

## Task 1: 对齐 pyproject.toml

**`python/pyproject.toml`**，将 `requirements.txt` 中有但 pyproject 中没有的包加入 optional dependencies：

```toml
[project.optional-dependencies]
data = [
    "akshare>=1.14.0",
    "ccxt>=4.4.0",
    "yfinance>=0.2.38",
    "edgartools>=3.0.0",
]
nlp = [
    "nltk>=3.9",
    "snownlp>=0.12.3",
    "textblob>=0.18",
    "jieba>=0.42.1",
]
stats = [
    "scipy>=1.14.0",
    "statsmodels>=0.14.0",
]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.24",
    "mootdx>=0.2.0",
]
```

同时确保 `requirements.txt` 中的 `httpx`、`joblib`、`torch`、`gymnasium`、`arch`、`gplearn` 在 `[project.dependencies]` 中存在。

**检查差异**：运行 `python -c "import tomllib; d = tomllib.load(open('pyproject.toml')); print(d['project']['dependencies'])"` 对比。

---

## Task 2: 生成完整 requirements.lock

```bash
cd python
pip install -e ".[data,nlp,stats,dev]"
pip freeze > requirements.lock
```

或者在 CI 中运行以确保 lockfile 是最新的。

---

## Task 3: 修复版本号

**`python/src/server.py:81`**：

```python
# before
version = "2026.6.26"

# after
from importlib.metadata import version as _pkg_version
try:
    __version__ = _pkg_version("quantflow-python")
except Exception:
    __version__ = "0.0.0"
```

或者从 `pyproject.toml` 读取：
```python
import tomllib
with open(Path(__file__).parent.parent / "pyproject.toml", "rb") as f:
    __version__ = tomllib.load(f)["project"]["version"]
```

---

## Task 4: 验证

```bash
cd python
pip install -e ".[data,nlp,stats,dev]"
python -m pytest tests/ -x -q
python -c "from server import __version__; print(__version__)"
```
